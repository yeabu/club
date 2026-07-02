package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OrganizationNode struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Type     string             `json:"type"`
	Meta     string             `json:"meta,omitempty"`
	Children []OrganizationNode `json:"children,omitempty"`
}

type OrganizationGraphResponse struct {
	Schools        []OrganizationNode     `json:"schools"`
	Counts         map[string]int         `json:"counts"`
	Grades         []OrganizationListItem `json:"grades,omitempty"`
	Classes        []OrganizationListItem `json:"classes,omitempty"`
	Subjects       []OrganizationListItem `json:"subjects,omitempty"`
	ClassSubjects  []OrganizationListItem `json:"classSubjects,omitempty"`
	Teachers       []OrganizationListItem `json:"teachers,omitempty"`
	Students       []OrganizationListItem `json:"students,omitempty"`
	Certifications []CertificationRecord  `json:"certifications,omitempty"`
}

type OrganizationListItem struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	SchoolID     string   `json:"schoolId,omitempty"`
	GradeID      string   `json:"gradeId,omitempty"`
	ClassID      string   `json:"classId,omitempty"`
	SubjectID    string   `json:"subjectId,omitempty"`
	TeacherID    string   `json:"teacherId,omitempty"`
	StudentNo    string   `json:"studentNo,omitempty"`
	Mobile       string   `json:"mobile,omitempty"`
	Relationship string   `json:"relationship,omitempty"`
	GradeIDs     []string `json:"gradeIds,omitempty"`
	SubjectIDs   []string `json:"subjectIds,omitempty"`
	Meta         string   `json:"meta,omitempty"`
}

type OrgCreateRequest struct {
	SchoolID     string `json:"schoolId"`
	GradeID      string `json:"gradeId"`
	ClassID      string `json:"classId"`
	UserID       string `json:"userId"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Stage        string `json:"stage"`
	SubjectID    string `json:"subjectId"`
	TeacherID    string `json:"teacherId"`
	StudentNo    string `json:"studentNo"`
	Mobile       string `json:"mobile"`
	Relationship string `json:"relationship"`
}

type GuardianInvitationRequest struct {
	StudentID  string `json:"studentId"`
	TeacherID  string `json:"teacherId"`
	MobileHint string `json:"mobileHint"`
}

type GuardianCertificationRequest struct {
	Token             string `json:"token"`
	GuardianID        string `json:"guardianId"`
	GuardianName      string `json:"guardianName"`
	Mobile            string `json:"mobile"`
	Relationship      string `json:"relationship"`
	EvidenceObjectKey string `json:"evidenceObjectKey"`
}

type CertificationDecisionRequest struct {
	Status     string `json:"status"`
	ReviewerID string `json:"reviewerId"`
	ReviewNote string `json:"reviewNote"`
}

type CertificationRecord struct {
	ID           string `json:"id"`
	StudentID    string `json:"studentId"`
	StudentName  string `json:"studentName"`
	GuardianID   string `json:"guardianId"`
	GuardianName string `json:"guardianName"`
	Relationship string `json:"relationship"`
	Status       string `json:"status"`
	SubmittedAt  string `json:"submittedAt"`
}

func decodeJSON(r *http.Request, target any) error {
	return json.NewDecoder(r.Body).Decode(target)
}

func (app *App) handleOrganizationGraph(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusOK, organizationFixture())
		return
	}
	result, err := app.store.OrganizationGraph(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (app *App) handleOrganizationCreate(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req OrgCreateRequest
	kind := r.PathValue("kind")
	if err := decodeJSON(r, &req); err != nil || (strings.TrimSpace(req.Name) == "" && kind != "class-subjects") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	id, err := app.store.CreateOrganizationEntity(r.Context(), kind, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "id": id, "kind": kind})
}

func (app *App) handleGuardianInvitation(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req GuardianInvitationRequest
	if decodeJSON(r, &req) != nil || req.StudentID == "" || req.TeacherID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "studentId and teacherId are required"})
		return
	}
	id, token, expiresAt, err := app.store.CreateGuardianInvitation(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "token": token, "invitePath": "/guardian/certify?token=" + token, "expiresAt": expiresAt})
}

func (app *App) handleGuardianCertification(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req GuardianCertificationRequest
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token is required"})
		return
	}
	if strings.TrimSpace(req.GuardianID) == "" && strings.TrimSpace(req.GuardianName) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "guardianId or guardianName is required"})
		return
	}
	id, err := app.store.SubmitGuardianCertification(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "pending"})
}

func (app *App) handleCertificationList(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []CertificationRecord{}})
		return
	}
	items, err := app.store.Certifications(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (app *App) handleCertificationDecision(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req CertificationDecisionRequest
	if decodeJSON(r, &req) != nil || (req.Status != "approved" && req.Status != "rejected") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be approved or rejected"})
		return
	}
	if err := app.store.DecideGuardianCertification(r.Context(), r.PathValue("certificationID"), req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (s *Store) CreateOrganizationEntity(ctx context.Context, kind string, req OrgCreateRequest) (string, error) {
	idPrefix := strings.ReplaceAll(strings.TrimSuffix(kind, "s"), "-", "_")
	id := fmt.Sprintf("%s_%d", idPrefix, time.Now().UnixNano())
	var query string
	var args []any
	switch kind {
	case "schools":
		query, args = `INSERT INTO schools (id, name) VALUES (?, ?)`, []any{id, req.Name}
	case "grades":
		if req.SchoolID == "" {
			return "", errors.New("schoolId is required")
		}
		if err := s.requireReference(ctx, `SELECT COUNT(*) FROM schools WHERE id=?`, []any{req.SchoolID}, "schoolId is invalid"); err != nil {
			return "", err
		}
		if req.Stage == "" {
			req.Stage = "primary"
		}
		query, args = `INSERT INTO grades (id, school_id, name, stage, sort_order) VALUES (?, ?, ?, ?, 0)`, []any{id, req.SchoolID, req.Name, req.Stage}
	case "subjects":
		if req.SchoolID == "" {
			return "", errors.New("schoolId is required")
		}
		if err := s.requireReference(ctx, `SELECT COUNT(*) FROM schools WHERE id=?`, []any{req.SchoolID}, "schoolId is invalid"); err != nil {
			return "", err
		}
		query, args = `INSERT INTO subjects (id, school_id, name, code) VALUES (?, ?, ?, ?)`, []any{id, req.SchoolID, req.Name, req.Code}
	case "classes":
		if req.SchoolID == "" || req.GradeID == "" {
			return "", errors.New("schoolId and gradeId are required")
		}
		if err := s.requireReference(ctx, `SELECT COUNT(*) FROM grades WHERE id=? AND school_id=?`, []any{req.GradeID, req.SchoolID}, "gradeId is invalid for this school"); err != nil {
			return "", err
		}
		query, args = `INSERT INTO classes (id, school_id, grade_id, name, grade) SELECT ?, ?, ?, ?, name FROM grades WHERE id = ?`, []any{id, req.SchoolID, req.GradeID, req.Name, req.GradeID}
	case "class-subjects":
		if req.SchoolID == "" || req.GradeID == "" || req.ClassID == "" || req.SubjectID == "" {
			return "", errors.New("schoolId, gradeId, classId and subjectId are required")
		}
		if err := s.requireReference(ctx, `SELECT COUNT(*) FROM classes WHERE id=? AND school_id=? AND grade_id=?`, []any{req.ClassID, req.SchoolID, req.GradeID}, "classId is invalid for this school and grade"); err != nil {
			return "", err
		}
		if err := s.requireReference(ctx, `SELECT COUNT(*) FROM subjects WHERE id=? AND school_id=?`, []any{req.SubjectID, req.SchoolID}, "subjectId is invalid for this school"); err != nil {
			return "", err
		}
		if req.TeacherID != "" {
			if err := s.requireReference(ctx, `SELECT COUNT(*) FROM teachers WHERE id=? AND school_id=?`, []any{req.TeacherID, req.SchoolID}, "teacherId is invalid for this school"); err != nil {
				return "", err
			}
		}
		query, args = `INSERT INTO class_subjects (id, school_id, grade_id, class_id, subject_id, teacher_id) VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE teacher_id=VALUES(teacher_id), status='active'`, []any{id, req.SchoolID, req.GradeID, req.ClassID, req.SubjectID, req.TeacherID}
	case "teachers":
		if req.SchoolID == "" || req.GradeID == "" || req.SubjectID == "" {
			return "", errors.New("schoolId, gradeId and subjectId are required")
		}
		if err := s.requireReference(ctx, `SELECT COUNT(*) FROM grades WHERE id=? AND school_id=?`, []any{req.GradeID, req.SchoolID}, "gradeId is invalid for this school"); err != nil {
			return "", err
		}
		if err := s.requireReference(ctx, `SELECT COUNT(*) FROM subjects WHERE id=? AND school_id=?`, []any{req.SubjectID, req.SchoolID}, "subjectId is invalid for this school"); err != nil {
			return "", err
		}
		if req.ClassID != "" {
			if err := s.requireReference(ctx, `SELECT COUNT(*) FROM classes WHERE id=? AND grade_id=?`, []any{req.ClassID, req.GradeID}, "classId is invalid for this grade"); err != nil {
				return "", err
			}
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return "", err
		}
		defer tx.Rollback()
		userID := "user_" + id
		if _, err = tx.ExecContext(ctx, `INSERT INTO users (id, school_id, display_name, mobile) VALUES (?, ?, ?, ?)`, userID, req.SchoolID, req.Name, req.Mobile); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO teachers (id, user_id, school_id, name, subject) SELECT ?, ?, ?, ?, name FROM subjects WHERE id = ?`, id, userID, req.SchoolID, req.Name, req.SubjectID); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO teacher_grades (teacher_id, grade_id) VALUES (?, ?)`, id, req.GradeID); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES (?, ?)`, id, req.SubjectID); err != nil {
			return "", err
		}
		if req.ClassID != "" {
			if _, err = tx.ExecContext(ctx, `INSERT INTO teacher_classes (teacher_id, class_id, subject) SELECT ?, ?, name FROM subjects WHERE id=?`, id, req.ClassID, req.SubjectID); err != nil {
				return "", err
			}
		}
		_, _ = tx.ExecContext(ctx, `INSERT IGNORE INTO user_roles (user_id, role_id, org_type, org_id) VALUES (?, 'role_teacher', 'school', ?)`, userID, req.SchoolID)
		return id, tx.Commit()
	case "students":
		if req.ClassID == "" {
			return "", errors.New("classId is required")
		}
		userID := "user_" + id
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return "", err
		}
		defer tx.Rollback()
		var schoolID string
		if err = tx.QueryRowContext(ctx, `SELECT school_id FROM classes WHERE id=?`, req.ClassID).Scan(&schoolID); err != nil {
			return "", errors.New("classId is invalid")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO users (id, school_id, display_name, mobile) VALUES (?, ?, ?, ?)`, userID, schoolID, req.Name, req.Mobile); err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO students (id, user_id, class_id, student_no, name, guardian_name) VALUES (?, ?, ?, ?, ?, '')`, id, userID, req.ClassID, req.StudentNo, req.Name); err != nil {
			return "", err
		}
		return id, tx.Commit()
	default:
		return "", errors.New("unsupported organization entity")
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return "", err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return "", errors.New("referenced entity is invalid")
	}
	return id, nil
}

func (s *Store) requireReference(ctx context.Context, query string, args []any, message string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return errors.New(message)
	}
	return nil
}

func (s *Store) OrganizationGraph(ctx context.Context) (OrganizationGraphResponse, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.name, g.id, g.name, c.id, c.name FROM schools s LEFT JOIN grades g ON g.school_id=s.id LEFT JOIN classes c ON c.grade_id=g.id ORDER BY s.name, g.sort_order, c.name`)
	if err != nil {
		return OrganizationGraphResponse{}, err
	}
	defer rows.Close()
	schools := map[string]*OrganizationNode{}
	grades := map[string]*OrganizationNode{}
	counts := map[string]int{"schools": 0, "grades": 0, "classes": 0, "teachers": 0, "students": 0, "subjects": 0, "classSubjects": 0, "pendingCertifications": 0}
	for rows.Next() {
		var sid, sn string
		var gid, gn, cid, cn sql.NullString
		if err := rows.Scan(&sid, &sn, &gid, &gn, &cid, &cn); err != nil {
			return OrganizationGraphResponse{}, err
		}
		if schools[sid] == nil {
			schools[sid] = &OrganizationNode{ID: sid, Name: sn, Type: "school"}
			counts["schools"]++
		}
		if gid.Valid && grades[gid.String] == nil {
			node := OrganizationNode{ID: gid.String, Name: gn.String, Type: "grade"}
			grades[gid.String] = &node
			schools[sid].Children = append(schools[sid].Children, node)
			counts["grades"]++
		}
		if cid.Valid {
			for i := range schools[sid].Children {
				if schools[sid].Children[i].ID == gid.String {
					schools[sid].Children[i].Children = append(schools[sid].Children[i].Children, OrganizationNode{ID: cid.String, Name: cn.String, Type: "class"})
					counts["classes"]++
					break
				}
			}
		}
	}
	for key, table := range map[string]string{"teachers": "teachers", "students": "students", "subjects": "subjects", "classSubjects": "class_subjects"} {
		var count int
		_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count)
		counts[key] = count
	}
	var pending int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM guardian_certifications WHERE status='pending'`).Scan(&pending)
	counts["pendingCertifications"] = pending
	result := OrganizationGraphResponse{Counts: counts, Schools: []OrganizationNode{}}
	for _, school := range schools {
		result.Schools = append(result.Schools, *school)
	}
	if rows.Err() != nil {
		return OrganizationGraphResponse{}, rows.Err()
	}
	result.Grades, _ = s.organizationList(ctx, `SELECT id,name,school_id,'' AS grade_id,'' AS class_id,'' AS subject_id,'' AS student_no,'' AS mobile,'' AS relationship FROM grades ORDER BY sort_order,name`)
	result.Classes, _ = s.organizationList(ctx, `SELECT id,name,school_id,grade_id,'' AS class_id,'' AS subject_id,'' AS student_no,'' AS mobile,'' AS relationship FROM classes ORDER BY name`)
	result.Subjects, _ = s.organizationList(ctx, `SELECT id,name,school_id,'' AS grade_id,'' AS class_id,'' AS subject_id,'' AS student_no,'' AS mobile,'' AS relationship FROM subjects ORDER BY name`)
	result.ClassSubjects, _ = s.classSubjectList(ctx)
	result.Teachers, _ = s.teacherList(ctx)
	result.Students, _ = s.organizationList(ctx, `SELECT st.id,st.name,c.school_id,c.grade_id,st.class_id,'' AS subject_id,st.student_no,'' AS mobile,'' AS relationship FROM students st JOIN classes c ON c.id=st.class_id ORDER BY c.name, st.student_no, st.name`)
	result.Certifications, _ = s.Certifications(ctx, "pending")
	return result, nil
}

func (s *Store) classSubjectList(ctx context.Context) ([]OrganizationListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT cs.id, CONCAT(c.name, ' · ', sub.name), cs.school_id, cs.grade_id, cs.class_id, cs.subject_id, COALESCE(cs.teacher_id, ''), COALESCE(t.name, '')
		FROM class_subjects cs
		JOIN classes c ON c.id=cs.class_id
		JOIN subjects sub ON sub.id=cs.subject_id
		LEFT JOIN teachers t ON t.id=cs.teacher_id
		WHERE cs.status='active'
		ORDER BY c.name, sub.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []OrganizationListItem{}
	for rows.Next() {
		var item OrganizationListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.SchoolID, &item.GradeID, &item.ClassID, &item.SubjectID, &item.TeacherID, &item.Meta); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) organizationList(ctx context.Context, query string) ([]OrganizationListItem, error) {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []OrganizationListItem{}
	for rows.Next() {
		var item OrganizationListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.SchoolID, &item.GradeID, &item.ClassID, &item.SubjectID, &item.StudentNo, &item.Mobile, &item.Relationship); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) teacherList(ctx context.Context) ([]OrganizationListItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,school_id,subject FROM teachers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []OrganizationListItem{}
	for rows.Next() {
		var item OrganizationListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.SchoolID, &item.Meta); err != nil {
			return nil, err
		}
		item.GradeIDs, _ = s.teacherRelationIDs(ctx, `SELECT grade_id FROM teacher_grades WHERE teacher_id=? ORDER BY grade_id`, item.ID)
		item.SubjectIDs, _ = s.teacherRelationIDs(ctx, `SELECT subject_id FROM teacher_subjects WHERE teacher_id=? ORDER BY subject_id`, item.ID)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) teacherRelationIDs(ctx context.Context, query, teacherID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, query, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Store) CreateGuardianInvitation(ctx context.Context, req GuardianInvitationRequest) (string, string, string, error) {
	var allowed int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM students st JOIN classes c ON c.id=st.class_id JOIN teacher_grades tg ON tg.grade_id=c.grade_id WHERE st.id=? AND tg.teacher_id=?`, req.StudentID, req.TeacherID).Scan(&allowed); err != nil || allowed == 0 {
		return "", "", "", errors.New("teacher is not assigned to this student's grade")
	}
	id := fmt.Sprintf("invite_%d", time.Now().UnixNano())
	token := fmt.Sprintf("%x", sha256.Sum256([]byte(id+time.Now().String())))[:32]
	hash := sha256.Sum256([]byte(token))
	expires := time.Now().Add(7 * 24 * time.Hour)
	_, err := s.db.ExecContext(ctx, `INSERT INTO guardian_invitations (id, student_id, teacher_id, token_hash, mobile_hint, expires_at) VALUES (?, ?, ?, ?, ?, ?)`, id, req.StudentID, req.TeacherID, hex.EncodeToString(hash[:]), req.MobileHint, expires)
	return id, token, expires.Format(time.RFC3339), err
}

func (s *Store) SubmitGuardianCertification(ctx context.Context, req GuardianCertificationRequest) (string, error) {
	hash := sha256.Sum256([]byte(req.Token))
	var invitationID, studentID string
	err := s.db.QueryRowContext(ctx, `SELECT id, student_id FROM guardian_invitations WHERE token_hash=? AND status='pending' AND expires_at>NOW()`, hex.EncodeToString(hash[:])).Scan(&invitationID, &studentID)
	if err != nil {
		return "", errors.New("invitation is invalid or expired")
	}
	guardianID, err := s.ensureGuardian(ctx, req, studentID)
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("cert_%d", time.Now().UnixNano())
	relationship := req.Relationship
	if relationship == "" {
		relationship = "parent"
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO guardian_certifications (id, invitation_id, student_id, guardian_id, relationship, evidence_object_key) VALUES (?, ?, ?, ?, ?, ?)`, id, invitationID, studentID, guardianID, relationship, req.EvidenceObjectKey)
	return id, err
}

func (s *Store) ensureGuardian(ctx context.Context, req GuardianCertificationRequest, studentID string) (string, error) {
	if strings.TrimSpace(req.GuardianID) != "" {
		var exists int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM guardians WHERE id=?`, req.GuardianID).Scan(&exists); err != nil {
			return "", err
		}
		if exists == 0 {
			return "", errors.New("guardianId is invalid")
		}
		return req.GuardianID, nil
	}
	guardianName := strings.TrimSpace(req.GuardianName)
	if guardianName == "" {
		return "", errors.New("guardianName is required")
	}
	schoolID := "school_001"
	if err := s.db.QueryRowContext(ctx, `SELECT c.school_id FROM students st JOIN classes c ON c.id=st.class_id WHERE st.id=?`, studentID).Scan(&schoolID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	guardianID := fmt.Sprintf("guardian_%d", time.Now().UnixNano())
	userID := "user_" + guardianID
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO users (id, school_id, display_name, mobile) VALUES (?, ?, ?, ?)`, userID, schoolID, guardianName, req.Mobile); err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO guardians (id, user_id, name, mobile, relationship) VALUES (?, ?, ?, ?, ?)`, guardianID, userID, guardianName, req.Mobile, req.Relationship); err != nil {
		return "", err
	}
	_, _ = tx.ExecContext(ctx, `INSERT IGNORE INTO user_roles (user_id, role_id, org_type, org_id) VALUES (?, 'role_guardian', 'student', '')`, userID)
	return guardianID, tx.Commit()
}

func (s *Store) Certifications(ctx context.Context, status string) ([]CertificationRecord, error) {
	if status == "" {
		status = "pending"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT gc.id,gc.student_id,st.name,gc.guardian_id,g.name,gc.relationship,gc.status,gc.submitted_at FROM guardian_certifications gc JOIN students st ON st.id=gc.student_id JOIN guardians g ON g.id=gc.guardian_id WHERE gc.status=? ORDER BY gc.submitted_at`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []CertificationRecord{}
	for rows.Next() {
		var v CertificationRecord
		var submitted time.Time
		if err := rows.Scan(&v.ID, &v.StudentID, &v.StudentName, &v.GuardianID, &v.GuardianName, &v.Relationship, &v.Status, &submitted); err != nil {
			return nil, err
		}
		v.SubmittedAt = submitted.Format(time.RFC3339)
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) DecideGuardianCertification(ctx context.Context, id string, req CertificationDecisionRequest) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var studentID, guardianID, relationship, invitationID, status string
	if err = tx.QueryRowContext(ctx, `SELECT student_id,guardian_id,relationship,invitation_id,status FROM guardian_certifications WHERE id=? FOR UPDATE`, id).Scan(&studentID, &guardianID, &relationship, &invitationID, &status); err != nil {
		return err
	}
	if status != "pending" {
		return errors.New("certification has already been reviewed")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE guardian_certifications SET status=?,reviewer_id=?,review_note=?,reviewed_at=NOW() WHERE id=?`, req.Status, req.ReviewerID, req.ReviewNote, id); err != nil {
		return err
	}
	if req.Status == "approved" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO student_guardians (student_id,guardian_id,relationship,is_primary) VALUES (?,?,?,FALSE) ON DUPLICATE KEY UPDATE relationship=VALUES(relationship)`, studentID, guardianID, relationship); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE guardian_invitations SET status='used' WHERE id=?`, invitationID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func organizationFixture() OrganizationGraphResponse {
	return OrganizationGraphResponse{
		Counts:        map[string]int{"schools": 1, "grades": 1, "classes": 1, "teachers": 1, "students": 3, "subjects": 3, "classSubjects": 1, "pendingCertifications": 0},
		Schools:       []OrganizationNode{{ID: "school_001", Name: "示范学校", Type: "school", Children: []OrganizationNode{{ID: "grade_6", Name: "六年级", Type: "grade", Children: []OrganizationNode{{ID: "class_603", Name: "六年级 3 班", Type: "class"}}}}}},
		Grades:        []OrganizationListItem{{ID: "grade_6", Name: "六年级", SchoolID: "school_001"}},
		Classes:       []OrganizationListItem{{ID: "class_603", Name: "六年级 3 班", SchoolID: "school_001", GradeID: "grade_6"}},
		Subjects:      []OrganizationListItem{{ID: "subject_math", Name: "数学", SchoolID: "school_001"}},
		ClassSubjects: []OrganizationListItem{{ID: "course_603_math", Name: "六年级 3 班 · 数学", SchoolID: "school_001", GradeID: "grade_6", ClassID: "class_603", SubjectID: "subject_math", TeacherID: "teacher_001", Meta: "陈老师"}},
		Teachers:      []OrganizationListItem{{ID: "teacher_001", Name: "陈老师", SchoolID: "school_001", GradeIDs: []string{"grade_6"}, SubjectIDs: []string{"subject_math"}, Meta: "数学"}},
		Students:      []OrganizationListItem{{ID: "stu_001", Name: "张三", SchoolID: "school_001", GradeID: "grade_6", ClassID: "class_603", StudentNo: "60301"}},
	}
}
