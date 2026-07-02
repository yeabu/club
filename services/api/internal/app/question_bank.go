package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type KnowledgePoint struct {
	ID        string `json:"id"`
	SchoolID  string `json:"schoolId,omitempty"`
	GradeID   string `json:"gradeId,omitempty"`
	SubjectID string `json:"subjectId,omitempty"`
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	ParentID  string `json:"parentId,omitempty"`
	Code      string `json:"code,omitempty"`
}

type QuestionBankItem struct {
	ID             string           `json:"id"`
	SchoolID       string           `json:"schoolId,omitempty"`
	GradeID        string           `json:"gradeId,omitempty"`
	SubjectID      string           `json:"subjectId,omitempty"`
	Subject        string           `json:"subject"`
	QuestionType   string           `json:"questionType"`
	Difficulty     string           `json:"difficulty"`
	Content        string           `json:"content"`
	Answer         string           `json:"answer"`
	Analysis       string           `json:"analysis"`
	Source         string           `json:"source"`
	Status         string           `json:"status"`
	Knowledge      []KnowledgePoint `json:"knowledge"`
	LinkedMistakes int              `json:"linkedMistakes"`
	CreatedAt      string           `json:"createdAt,omitempty"`
}

type QuestionBankRequest struct {
	SchoolID          string   `json:"schoolId"`
	GradeID           string   `json:"gradeId"`
	SubjectID         string   `json:"subjectId"`
	Subject           string   `json:"subject"`
	QuestionType      string   `json:"questionType"`
	Difficulty        string   `json:"difficulty"`
	Content           string   `json:"content"`
	Answer            string   `json:"answer"`
	Analysis          string   `json:"analysis"`
	Source            string   `json:"source"`
	CreatedBy         string   `json:"createdBy"`
	KnowledgePointIDs []string `json:"knowledgePointIds"`
	KnowledgePoints   []string `json:"knowledgePoints"`
}

type KnowledgePointRequest struct {
	SchoolID  string `json:"schoolId"`
	GradeID   string `json:"gradeId"`
	SubjectID string `json:"subjectId"`
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	ParentID  string `json:"parentId"`
	Code      string `json:"code"`
}

func (app *App) handleKnowledgePoints(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": knowledgePointFixture(), "counts": map[string]int{"knowledgePoints": len(knowledgePointFixture())}})
		return
	}
	items, err := app.store.KnowledgePoints(r.Context(), r.URL.Query())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "counts": map[string]int{"knowledgePoints": len(items)}})
}

func (app *App) handleCreateKnowledgePoint(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req KnowledgePointRequest
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	item, err := app.store.CreateKnowledgePoint(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (app *App) handleQuestionBank(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": questionBankFixture(), "counts": map[string]int{"questions": len(questionBankFixture()), "knowledgePoints": len(knowledgePointFixture()), "linkedMistakes": 0}})
		return
	}
	items, counts, err := app.store.QuestionBank(r.Context(), r.URL.Query())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "counts": counts})
}

func (app *App) handleCreateQuestionBankItem(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req QuestionBankRequest
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}
	item, err := app.store.CreateQuestionBankItem(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (app *App) handleWrongQuestionKnowledge(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	id, err := strconv.ParseInt(r.PathValue("mistakeID"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mistakeID is invalid"})
		return
	}
	var req struct {
		KnowledgePointIDs []string `json:"knowledgePointIds"`
	}
	if decodeJSON(r, &req) != nil || len(req.KnowledgePointIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "knowledgePointIds is required"})
		return
	}
	if err := app.store.LinkWrongQuestionKnowledge(r.Context(), id, req.KnowledgePointIDs); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "linked", "wrongQuestionId": id, "knowledgePointIds": req.KnowledgePointIDs})
}

func (s *Store) KnowledgePoints(ctx context.Context, values mapValues) ([]KnowledgePoint, error) {
	query := `SELECT id,COALESCE(school_id,''),COALESCE(grade_id,''),COALESCE(subject_id,''),name,subject,COALESCE(parent_id,''),COALESCE(code,'') FROM knowledge_points WHERE 1=1`
	args := []any{}
	if subject := strings.TrimSpace(values.Get("subject")); subject != "" {
		query += ` AND subject=?`
		args = append(args, subject)
	}
	if subjectID := strings.TrimSpace(values.Get("subjectId")); subjectID != "" {
		query += ` AND subject_id=?`
		args = append(args, subjectID)
	}
	if gradeID := strings.TrimSpace(values.Get("gradeId")); gradeID != "" {
		query += ` AND grade_id=?`
		args = append(args, gradeID)
	}
	if search := strings.TrimSpace(values.Get("search")); search != "" {
		query += ` AND name LIKE ?`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY subject,name`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []KnowledgePoint{}
	for rows.Next() {
		var item KnowledgePoint
		if err := rows.Scan(&item.ID, &item.SchoolID, &item.GradeID, &item.SubjectID, &item.Name, &item.Subject, &item.ParentID, &item.Code); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateKnowledgePoint(ctx context.Context, req KnowledgePointRequest) (KnowledgePoint, error) {
	if req.Subject == "" {
		req.Subject = "数学"
	}
	if req.SchoolID == "" {
		req.SchoolID = "school_001"
	}
	id := fmt.Sprintf("kp_%d", time.Now().UnixNano())
	_, err := s.db.ExecContext(ctx, `INSERT INTO knowledge_points (id,school_id,grade_id,subject_id,name,subject,parent_id,code) VALUES (?,?,?,?,?,?,?,?)`, id, req.SchoolID, req.GradeID, req.SubjectID, req.Name, req.Subject, req.ParentID, req.Code)
	if err != nil {
		return KnowledgePoint{}, err
	}
	return KnowledgePoint{ID: id, SchoolID: req.SchoolID, GradeID: req.GradeID, SubjectID: req.SubjectID, Name: req.Name, Subject: req.Subject, ParentID: req.ParentID, Code: req.Code}, nil
}

func (s *Store) QuestionBank(ctx context.Context, values mapValues) ([]QuestionBankItem, map[string]int, error) {
	query := `SELECT qb.id,COALESCE(qb.school_id,''),COALESCE(qb.grade_id,''),COALESCE(qb.subject_id,''),qb.subject,qb.question_type,qb.difficulty,qb.content,COALESCE(qb.answer,''),COALESCE(qb.analysis,''),qb.source,qb.status,qb.created_at FROM question_bank qb WHERE 1=1`
	args := []any{}
	if subject := strings.TrimSpace(values.Get("subject")); subject != "" {
		query += ` AND qb.subject=?`
		args = append(args, subject)
	}
	if gradeID := strings.TrimSpace(values.Get("gradeId")); gradeID != "" {
		query += ` AND qb.grade_id=?`
		args = append(args, gradeID)
	}
	if questionType := strings.TrimSpace(values.Get("type")); questionType != "" {
		query += ` AND qb.question_type=?`
		args = append(args, questionType)
	}
	if difficulty := strings.TrimSpace(values.Get("difficulty")); difficulty != "" {
		query += ` AND qb.difficulty=?`
		args = append(args, difficulty)
	}
	if knowledge := strings.TrimSpace(values.Get("knowledge")); knowledge != "" {
		query += ` AND EXISTS (SELECT 1 FROM question_bank_knowledge_points qkp WHERE qkp.question_id=qb.id AND (qkp.knowledge_point_id=? OR qkp.knowledge_point_name=?))`
		args = append(args, knowledge, knowledge)
	}
	if search := strings.TrimSpace(values.Get("search")); search != "" {
		query += ` AND (qb.content LIKE ? OR qb.answer LIKE ? OR qb.analysis LIKE ?)`
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern, pattern)
	}
	query += ` ORDER BY qb.updated_at DESC, qb.created_at DESC LIMIT 200`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []QuestionBankItem{}
	for rows.Next() {
		var item QuestionBankItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.SchoolID, &item.GradeID, &item.SubjectID, &item.Subject, &item.QuestionType, &item.Difficulty, &item.Content, &item.Answer, &item.Analysis, &item.Source, &item.Status, &createdAt); err != nil {
			return nil, nil, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		item.Knowledge, _ = s.questionKnowledge(ctx, item.ID)
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wrong_question_knowledge_points wkp JOIN question_bank_knowledge_points qkp ON qkp.knowledge_point_id=wkp.knowledge_point_id WHERE qkp.question_id=?`, item.ID).Scan(&item.LinkedMistakes)
		items = append(items, item)
	}
	var knowledgeCount, linkedMistakes int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM knowledge_points`).Scan(&knowledgeCount)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wrong_question_knowledge_points`).Scan(&linkedMistakes)
	return items, map[string]int{"questions": len(items), "knowledgePoints": knowledgeCount, "linkedMistakes": linkedMistakes}, rows.Err()
}

func (s *Store) CreateQuestionBankItem(ctx context.Context, req QuestionBankRequest) (QuestionBankItem, error) {
	if req.Subject == "" {
		req.Subject = "数学"
	}
	if req.QuestionType == "" {
		req.QuestionType = "subjective"
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}
	if req.Source == "" {
		req.Source = "manual"
	}
	id := fmt.Sprintf("qb_%d", time.Now().UnixNano())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return QuestionBankItem{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO question_bank (id,school_id,grade_id,subject_id,subject,question_type,difficulty,content,answer,analysis,source,created_by) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, id, req.SchoolID, req.GradeID, req.SubjectID, req.Subject, req.QuestionType, req.Difficulty, req.Content, req.Answer, req.Analysis, req.Source, req.CreatedBy); err != nil {
		return QuestionBankItem{}, err
	}
	knowledge, err := s.resolveKnowledgeForQuestion(ctx, tx, req)
	if err != nil {
		return QuestionBankItem{}, err
	}
	for _, item := range knowledge {
		if _, err = tx.ExecContext(ctx, `INSERT INTO question_bank_knowledge_points (question_id,knowledge_point_id,knowledge_point_name) VALUES (?,?,?)`, id, item.ID, item.Name); err != nil {
			return QuestionBankItem{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return QuestionBankItem{}, err
	}
	return QuestionBankItem{ID: id, SchoolID: req.SchoolID, GradeID: req.GradeID, SubjectID: req.SubjectID, Subject: req.Subject, QuestionType: req.QuestionType, Difficulty: req.Difficulty, Content: req.Content, Answer: req.Answer, Analysis: req.Analysis, Source: req.Source, Status: "active", Knowledge: knowledge}, nil
}

func (s *Store) resolveKnowledgeForQuestion(ctx context.Context, tx *sql.Tx, req QuestionBankRequest) ([]KnowledgePoint, error) {
	items := []KnowledgePoint{}
	for _, id := range req.KnowledgePointIDs {
		var item KnowledgePoint
		if err := tx.QueryRowContext(ctx, `SELECT id,COALESCE(school_id,''),COALESCE(grade_id,''),COALESCE(subject_id,''),name,subject,COALESCE(parent_id,''),COALESCE(code,'') FROM knowledge_points WHERE id=?`, id).Scan(&item.ID, &item.SchoolID, &item.GradeID, &item.SubjectID, &item.Name, &item.Subject, &item.ParentID, &item.Code); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	for _, name := range req.KnowledgePoints {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		var item KnowledgePoint
		err := tx.QueryRowContext(ctx, `SELECT id,COALESCE(school_id,''),COALESCE(grade_id,''),COALESCE(subject_id,''),name,subject,COALESCE(parent_id,''),COALESCE(code,'') FROM knowledge_points WHERE name=? AND subject=?`, name, req.Subject).Scan(&item.ID, &item.SchoolID, &item.GradeID, &item.SubjectID, &item.Name, &item.Subject, &item.ParentID, &item.Code)
		if errors.Is(err, sql.ErrNoRows) {
			item = KnowledgePoint{ID: fmt.Sprintf("kp_%d", time.Now().UnixNano()), SchoolID: req.SchoolID, GradeID: req.GradeID, SubjectID: req.SubjectID, Name: name, Subject: req.Subject}
			if _, err = tx.ExecContext(ctx, `INSERT INTO knowledge_points (id,school_id,grade_id,subject_id,name,subject) VALUES (?,?,?,?,?,?)`, item.ID, item.SchoolID, item.GradeID, item.SubjectID, item.Name, item.Subject); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, errors.New("at least one knowledge point is required")
	}
	return items, nil
}

func (s *Store) questionKnowledge(ctx context.Context, questionID string) ([]KnowledgePoint, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT kp.id,COALESCE(kp.school_id,''),COALESCE(kp.grade_id,''),COALESCE(kp.subject_id,''),kp.name,kp.subject,COALESCE(kp.parent_id,''),COALESCE(kp.code,'') FROM question_bank_knowledge_points qkp JOIN knowledge_points kp ON kp.id=qkp.knowledge_point_id WHERE qkp.question_id=? ORDER BY kp.subject,kp.name`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []KnowledgePoint{}
	for rows.Next() {
		var item KnowledgePoint
		if err := rows.Scan(&item.ID, &item.SchoolID, &item.GradeID, &item.SubjectID, &item.Name, &item.Subject, &item.ParentID, &item.Code); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) LinkWrongQuestionKnowledge(ctx context.Context, wrongQuestionID int64, knowledgePointIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM wrong_question_knowledge_points WHERE wrong_question_id=?`, wrongQuestionID); err != nil {
		return err
	}
	firstName, firstID := "", ""
	for _, id := range knowledgePointIDs {
		var name string
		if err = tx.QueryRowContext(ctx, `SELECT name FROM knowledge_points WHERE id=?`, id).Scan(&name); err != nil {
			return err
		}
		if firstID == "" {
			firstID, firstName = id, name
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO wrong_question_knowledge_points (wrong_question_id,knowledge_point_id,knowledge_point_name) VALUES (?,?,?)`, wrongQuestionID, id, name); err != nil {
			return err
		}
	}
	if firstID == "" {
		return errors.New("knowledgePointIds is required")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wrong_questions SET knowledge_point_id=?, knowledge_point=? WHERE id=?`, firstID, firstName, wrongQuestionID); err != nil {
		return err
	}
	return tx.Commit()
}

type mapValues interface {
	Get(string) string
}

func knowledgePointFixture() []KnowledgePoint {
	return []KnowledgePoint{
		{ID: "kp_001", Name: "分数应用题", Subject: "数学", GradeID: "grade_6", SubjectID: "subject_math"},
		{ID: "kp_002", Name: "几何面积", Subject: "数学", GradeID: "grade_6", SubjectID: "subject_math"},
		{ID: "kp_003", Name: "比例换算", Subject: "数学", GradeID: "grade_6", SubjectID: "subject_math"},
	}
}

func questionBankFixture() []QuestionBankItem {
	return []QuestionBankItem{
		{ID: "qb_001", Subject: "数学", QuestionType: "single_choice", Difficulty: "easy", Content: "比较 3/5 和 4/7 的大小，选择正确答案。", Answer: "A", Analysis: "通分后比较。", Source: "template_split", Status: "active", Knowledge: []KnowledgePoint{{ID: "kp_005", Name: "分数", Subject: "数学"}}},
		{ID: "qb_002", Subject: "数学", QuestionType: "subjective", Difficulty: "medium", Content: "一桶油用去 3/5 后还剩 40 千克，这桶油原来有多少千克？", Answer: "100 千克", Analysis: "剩余为 2/5。", Source: "manual", Status: "active", Knowledge: []KnowledgePoint{{ID: "kp_001", Name: "分数应用题", Subject: "数学"}}},
	}
}
