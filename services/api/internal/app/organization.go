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
	Schools []OrganizationNode `json:"schools"`
	Counts  map[string]int     `json:"counts"`
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
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	kind := r.PathValue("kind")
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
	if decodeJSON(r, &req) != nil || req.Token == "" || req.GuardianID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token and guardianId are required"})
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
	id := fmt.Sprintf("%s_%d", strings.TrimSuffix(kind, "s"), time.Now().UnixNano())
	var query string
	var args []any
	switch kind {
	case "schools":
		query, args = `INSERT INTO schools (id, name) VALUES (?, ?)`, []any{id, req.Name}
	case "grades":
		if req.SchoolID == "" {
			return "", errors.New("schoolId is required")
		}
		query, args = `INSERT INTO grades (id, school_id, name, stage, sort_order) VALUES (?, ?, ?, ?, 0)`, []any{id, req.SchoolID, req.Name, req.Stage}
	case "subjects":
		if req.SchoolID == "" {
			return "", errors.New("schoolId is required")
		}
		query, args = `INSERT INTO subjects (id, school_id, name, code) VALUES (?, ?, ?, ?)`, []any{id, req.SchoolID, req.Name, req.Code}
	case "classes":
		if req.SchoolID == "" || req.GradeID == "" {
			return "", errors.New("schoolId and gradeId are required")
		}
		query, args = `INSERT INTO classes (id, school_id, grade_id, name, grade) SELECT ?, ?, ?, ?, name FROM grades WHERE id = ?`, []any{id, req.SchoolID, req.GradeID, req.Name, req.GradeID}
	case "teachers":
		if req.SchoolID == "" || req.GradeID == "" || req.SubjectID == "" {
			return "", errors.New("schoolId, gradeId and subjectId are required")
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
		return id, tx.Commit()
	case "students":
		if req.ClassID == "" {
			return "", errors.New("classId is required")
		}
		query, args = `INSERT INTO students (id, class_id, student_no, name, guardian_name) VALUES (?, ?, ?, ?, '')`, []any{id, req.ClassID, req.StudentNo, req.Name}
	default:
		return "", errors.New("unsupported organization entity")
	}
	_, err := s.db.ExecContext(ctx, query, args...)
	return id, err
}

func (s *Store) OrganizationGraph(ctx context.Context) (OrganizationGraphResponse, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.name, g.id, g.name, c.id, c.name FROM schools s LEFT JOIN grades g ON g.school_id=s.id LEFT JOIN classes c ON c.grade_id=g.id ORDER BY s.name, g.sort_order, c.name`)
	if err != nil {
		return OrganizationGraphResponse{}, err
	}
	defer rows.Close()
	schools := map[string]*OrganizationNode{}
	grades := map[string]*OrganizationNode{}
	counts := map[string]int{"schools": 0, "grades": 0, "classes": 0, "teachers": 0, "students": 0, "subjects": 0, "pendingCertifications": 0}
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
	for key, table := range map[string]string{"teachers": "teachers", "students": "students", "subjects": "subjects"} {
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
	return result, rows.Err()
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
	id := fmt.Sprintf("cert_%d", time.Now().UnixNano())
	relationship := req.Relationship
	if relationship == "" {
		relationship = "parent"
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO guardian_certifications (id, invitation_id, student_id, guardian_id, relationship, evidence_object_key) VALUES (?, ?, ?, ?, ?, ?)`, id, invitationID, studentID, req.GuardianID, relationship, req.EvidenceObjectKey)
	return id, err
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
	return OrganizationGraphResponse{Counts: map[string]int{"schools": 1, "grades": 1, "classes": 1, "teachers": 1, "students": 3, "subjects": 3, "pendingCertifications": 0}, Schools: []OrganizationNode{{ID: "school_001", Name: "示范学校", Type: "school", Children: []OrganizationNode{{ID: "grade_6", Name: "六年级", Type: "grade", Children: []OrganizationNode{{ID: "class_603", Name: "六年级 3 班", Type: "class"}}}}}}}
}
