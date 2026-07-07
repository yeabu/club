package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
)

var demoInitialPasswords = map[string]string{
	"user_admin_001":    "Admin@123456",
	"user_teacher_001":  "Teacher@123456",
	"user_stu_001":      "Student@123456",
	"user_guardian_001": "Guardian@123456",
}

var demoLoginAliases = map[string]string{
	"user_admin_001":      "user_admin_001",
	"13800000000":         "user_admin_001",
	"admin@example.local": "user_admin_001",

	"user_teacher_001":      "user_teacher_001",
	"13800000001":           "user_teacher_001",
	"teacher@example.local": "user_teacher_001",

	"user_stu_001":          "user_stu_001",
	"13800000021":           "user_stu_001",
	"student@example.local": "user_stu_001",

	"user_guardian_001":      "user_guardian_001",
	"13800000011":            "user_guardian_001",
	"guardian@example.local": "user_guardian_001",
}

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req AuthLoginRequest
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.Account) == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account and password are required"})
		return
	}
	if app.store == nil {
		auth, ok := fallbackLogin(strings.TrimSpace(req.Account), req.Password)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid account or password"})
			return
		}
		writeJSON(w, http.StatusOK, AuthLoginResponse{Status: "ok", User: auth, AuthHeaders: authHeaders(auth)})
		return
	}
	auth, err := app.store.AuthenticateUser(r.Context(), strings.TrimSpace(req.Account), req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid account or password"})
		return
	}
	writeJSON(w, http.StatusOK, AuthLoginResponse{Status: "ok", User: auth, AuthHeaders: authHeaders(auth)})
}

func (app *App) requestAuth(ctx context.Context, r *http.Request) (AuthContext, error) {
	auth := AuthContext{
		UserID:     strings.TrimSpace(r.Header.Get("X-User-ID")),
		StudentID:  strings.TrimSpace(r.Header.Get("X-Student-ID")),
		GuardianID: strings.TrimSpace(r.Header.Get("X-Guardian-ID")),
		TeacherID:  strings.TrimSpace(r.Header.Get("X-Teacher-ID")),
		SchoolID:   strings.TrimSpace(r.Header.Get("X-School-ID")),
	}
	if role := strings.TrimSpace(r.Header.Get("X-Role")); role != "" {
		auth.RoleNames = splitCSV(decodeHeaderValue(role))
	}
	if app.store != nil && auth.UserID == "" && app.config.AppEnv != "production" && r.Method == http.MethodGet {
		devAuth, err := app.store.AuthContext(ctx, "user_teacher_001")
		if err == nil {
			return devAuth, nil
		}
	}
	if app.store == nil || auth.UserID == "" {
		return auth, nil
	}
	return app.store.AuthContext(ctx, auth.UserID)
}

func requireAuth(w http.ResponseWriter, auth AuthContext) bool {
	if auth.UserID == "" && auth.StudentID == "" && auth.GuardianID == "" && auth.TeacherID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login is required"})
		return false
	}
	return true
}

func decodeHeaderValue(value string) string {
	if !strings.Contains(value, "%") {
		return value
	}
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

func hasRole(auth AuthContext, roles ...string) bool {
	for _, current := range auth.RoleNames {
		for _, role := range roles {
			if current == role {
				return true
			}
		}
	}
	return false
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (s *Store) AuthContext(ctx context.Context, userID string) (AuthContext, error) {
	auth := AuthContext{UserID: userID}
	err := s.db.QueryRowContext(ctx, `SELECT school_id FROM users WHERE id=? AND status='active' LIMIT 1`, userID).Scan(&auth.SchoolID)
	if err != nil {
		return auth, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT roles.name
		FROM user_roles
		JOIN roles ON roles.id = user_roles.role_id
		WHERE user_roles.user_id = ?
		ORDER BY roles.name`, userID)
	if err != nil {
		return auth, err
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		if rows.Scan(&role) == nil {
			auth.RoleNames = append(auth.RoleNames, role)
		}
	}
	if err := rows.Err(); err != nil {
		return auth, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM students WHERE user_id=? LIMIT 1`, userID).Scan(&auth.StudentID); err != nil && err != sql.ErrNoRows {
		return auth, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM guardians WHERE user_id=? LIMIT 1`, userID).Scan(&auth.GuardianID); err != nil && err != sql.ErrNoRows {
		return auth, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM teachers WHERE user_id=? LIMIT 1`, userID).Scan(&auth.TeacherID); err != nil && err != sql.ErrNoRows {
		return auth, err
	}
	return auth, nil
}

func (s *Store) AuthenticateUser(ctx context.Context, account string, password string) (AuthContext, error) {
	var userID string
	err := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE status='active'
			AND (id=? OR mobile=? OR email=?)
			AND password_hash=?
		LIMIT 1`, account, account, account, hashDemoPassword(password)).Scan(&userID)
	if err != nil {
		return s.authenticateSeedUser(ctx, account, password, err)
	}
	return s.AuthContext(ctx, userID)
}

func (s *Store) authenticateSeedUser(ctx context.Context, account string, password string, originalErr error) (AuthContext, error) {
	if strings.EqualFold(strings.TrimSpace(s.config.AppEnv), "production") {
		return AuthContext{}, originalErr
	}
	userID, ok := seedUserIDForLogin(account, password)
	if !ok {
		return AuthContext{}, originalErr
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash=?
		WHERE id=? AND status='active'`,
		hashDemoPassword(password), userID)
	if err != nil {
		return AuthContext{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AuthContext{}, err
	}
	if affected == 0 {
		return AuthContext{}, originalErr
	}
	return s.AuthContext(ctx, userID)
}

func (s *Store) CanAccessStudent(ctx context.Context, auth AuthContext, studentID string) (bool, error) {
	studentID = strings.TrimSpace(studentID)
	if studentID == "" {
		return false, nil
	}
	if auth.StudentID == studentID {
		return true, nil
	}
	if auth.GuardianID != "" {
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM student_guardians WHERE guardian_id=? AND student_id=?`, auth.GuardianID, studentID).Scan(&count); err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	if auth.TeacherID != "" {
		var count int
		err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM students
			JOIN teacher_classes ON teacher_classes.class_id = students.class_id
			WHERE teacher_classes.teacher_id=? AND students.id=?`, auth.TeacherID, studentID).Scan(&count)
		if err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return hasRole(auth, "教务管理员"), nil
}

func fallbackLogin(account string, password string) (AuthContext, bool) {
	userID, ok := seedUserIDForLogin(account, password)
	if !ok {
		return AuthContext{}, false
	}
	switch userID {
	case "user_admin_001":
		return AuthContext{UserID: userID, SchoolID: "school_001", RoleNames: []string{"教务管理员"}}, true
	case "user_teacher_001":
		return AuthContext{UserID: userID, SchoolID: "school_001", RoleNames: []string{"任课教师"}, TeacherID: "teacher_001"}, true
	case "user_stu_001":
		return AuthContext{UserID: userID, SchoolID: "school_001", RoleNames: []string{}, StudentID: "stu_001"}, true
	case "user_guardian_001":
		return AuthContext{UserID: userID, SchoolID: "school_001", RoleNames: []string{"家长"}, GuardianID: "guardian_001"}, true
	}
	return AuthContext{}, false
}

func seedUserIDForLogin(account string, password string) (string, bool) {
	userID := demoLoginAliases[strings.TrimSpace(account)]
	if userID == "" {
		return "", false
	}
	return userID, demoInitialPasswords[userID] == password
}

func authHeaders(auth AuthContext) map[string]string {
	headers := map[string]string{"X-User-ID": auth.UserID}
	if auth.SchoolID != "" {
		headers["X-School-ID"] = auth.SchoolID
	}
	if auth.TeacherID != "" {
		headers["X-Teacher-ID"] = auth.TeacherID
	}
	if auth.StudentID != "" {
		headers["X-Student-ID"] = auth.StudentID
	}
	if auth.GuardianID != "" {
		headers["X-Guardian-ID"] = auth.GuardianID
	}
	if len(auth.RoleNames) > 0 {
		headers["X-Role"] = strings.Join(auth.RoleNames, ",")
	}
	return headers
}

func hashDemoPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}
