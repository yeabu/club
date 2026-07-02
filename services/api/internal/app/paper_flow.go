package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PaperComposition struct {
	ID            string                     `json:"id"`
	Title         string                     `json:"title"`
	SchoolID      string                     `json:"schoolId,omitempty"`
	GradeID       string                     `json:"gradeId,omitempty"`
	GradeName     string                     `json:"gradeName,omitempty"`
	SubjectID     string                     `json:"subjectId,omitempty"`
	Subject       string                     `json:"subject"`
	Mode          string                     `json:"mode"`
	Status        string                     `json:"status"`
	QuestionCount int                        `json:"questionCount"`
	TotalScore    float64                    `json:"totalScore"`
	CreatedBy     string                     `json:"createdBy,omitempty"`
	Questions     []PaperCompositionQuestion `json:"questions,omitempty"`
	CreatedAt     string                     `json:"createdAt,omitempty"`
}

type PaperCompositionQuestion struct {
	QuestionBankItem
	SortOrder int     `json:"sortOrder"`
	Score     float64 `json:"score"`
}

type PaperCompositionRequest struct {
	Title       string             `json:"title"`
	SchoolID    string             `json:"schoolId"`
	GradeID     string             `json:"gradeId"`
	GradeName   string             `json:"gradeName"`
	SubjectID   string             `json:"subjectId"`
	Subject     string             `json:"subject"`
	CreatedBy   string             `json:"createdBy"`
	QuestionIDs []string           `json:"questionIds"`
	Scores      map[string]float64 `json:"scores"`
}

type AITaskRecord struct {
	ID              string         `json:"id"`
	TaskType        string         `json:"taskType"`
	Status          string         `json:"status"`
	Provider        string         `json:"provider"`
	Request         map[string]any `json:"request"`
	Result          map[string]any `json:"result,omitempty"`
	SourceObjectKey string         `json:"sourceObjectKey,omitempty"`
	SourceURL       string         `json:"sourceUrl,omitempty"`
	OwnerType       string         `json:"ownerType,omitempty"`
	OwnerID         string         `json:"ownerId,omitempty"`
	CreatedBy       string         `json:"createdBy,omitempty"`
	Message         string         `json:"message,omitempty"`
	ErrorMessage    string         `json:"errorMessage,omitempty"`
	CreatedAt       string         `json:"createdAt,omitempty"`
}

type AnswerSheetUpload struct {
	ID            string `json:"id"`
	CompositionID string `json:"compositionId,omitempty"`
	StudentID     string `json:"studentId,omitempty"`
	StudentName   string `json:"studentName,omitempty"`
	ObjectKey     string `json:"objectKey"`
	URL           string `json:"url"`
	Status        string `json:"status"`
	GradingTaskID string `json:"gradingTaskId,omitempty"`
}

func (app *App) handlePaperCompositions(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": paperCompositionFixture(), "counts": map[string]int{"compositions": len(paperCompositionFixture())}})
		return
	}
	items, err := app.store.PaperCompositions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "counts": map[string]int{"compositions": len(items)}})
}

func (app *App) handleCreatePaperComposition(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req PaperCompositionRequest
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.Title) == "" || len(req.QuestionIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and questionIds are required"})
		return
	}
	item, err := app.store.CreatePaperComposition(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (app *App) handleAIComposeRequest(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req map[string]any
	if decodeJSON(r, &req) != nil {
		req = map[string]any{}
	}
	task, err := app.store.CreateAITask(r.Context(), "ai_paper_composition", req, "", "", "paper_composition", r.PathValue("compositionID"), stringValue(req["createdBy"]))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func (app *App) handleBlankPaperUpload(w http.ResponseWriter, r *http.Request) {
	app.handlePaperFileUpload(w, r, "blank_paper", "paper_template_analysis")
}

func (app *App) handleAnswerSheetUpload(w http.ResponseWriter, r *http.Request) {
	app.handlePaperFileUpload(w, r, "student_answer", "answer_sheet_uploaded")
}

func (app *App) handlePaperFileUpload(w http.ResponseWriter, r *http.Request, purpose, taskType string) {
	r.Body = http.MaxBytesReader(w, r.Body, maxScanUploadBytes*20)
	if err := r.ParseMultipartForm(maxScanUploadBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart upload"})
		return
	}
	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		headers = r.MultipartForm.File["file"]
	}
	if len(headers) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one file is required"})
		return
	}
	compositionID := r.FormValue("compositionId")
	studentID := r.FormValue("studentId")
	studentName := r.FormValue("studentName")
	createdBy := r.FormValue("createdBy")
	files := make([]ScanFile, 0, len(headers))
	uploads := []AnswerSheetUpload{}
	tasks := []AITaskRecord{}
	for _, header := range headers {
		file, err := saveScanUpload(app.config, header)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		files = append(files, file)
		if app.store != nil {
			ownerType := "paper"
			ownerID := compositionID
			if purpose == "student_answer" {
				ownerType = "student"
				ownerID = studentID
			}
			if err := app.store.SaveObjectFiles(r.Context(), []ScanFile{file}, purpose, ownerType, ownerID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			taskRequest := map[string]any{"compositionId": compositionID, "studentId": studentID, "studentName": studentName, "fileName": file.FileName, "phase": "reserved"}
			task, err := app.store.CreateAITask(r.Context(), taskType, taskRequest, file.Key, file.URL, ownerType, ownerID, createdBy)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			tasks = append(tasks, task)
			if purpose == "student_answer" {
				upload, err := app.store.CreateAnswerSheetUpload(r.Context(), compositionID, studentID, studentName, file, task.ID)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				uploads = append(uploads, upload)
			}
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"files": files, "uploads": uploads, "tasks": tasks})
}

func (app *App) handleCreateGradingTask(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database is required"})
		return
	}
	var req map[string]any
	if decodeJSON(r, &req) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	mode := stringValue(req["mode"])
	if mode == "" {
		mode = "standard_answer"
	}
	taskType := "standard_answer_grading"
	if mode == "ai" {
		taskType = "ai_grading"
	}
	task, err := app.store.CreateAITask(r.Context(), taskType, req, "", "", "paper_composition", stringValue(req["compositionId"]), stringValue(req["createdBy"]))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func (s *Store) PaperCompositions(ctx context.Context) ([]PaperComposition, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,COALESCE(school_id,''),COALESCE(grade_id,''),COALESCE(grade_name,''),COALESCE(subject_id,''),subject,mode,status,question_count,total_score,COALESCE(created_by,''),created_at FROM paper_compositions ORDER BY updated_at DESC, created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PaperComposition{}
	for rows.Next() {
		var item PaperComposition
		var created time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.SchoolID, &item.GradeID, &item.GradeName, &item.SubjectID, &item.Subject, &item.Mode, &item.Status, &item.QuestionCount, &item.TotalScore, &item.CreatedBy, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = created.Format(time.RFC3339)
		item.Questions, _ = s.CompositionQuestions(ctx, item.ID)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreatePaperComposition(ctx context.Context, req PaperCompositionRequest) (PaperComposition, error) {
	if req.Subject == "" {
		req.Subject = "数学"
	}
	id := fmt.Sprintf("paper_%d", time.Now().UnixNano())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PaperComposition{}, err
	}
	defer tx.Rollback()
	total := 0.0
	for index, questionID := range req.QuestionIDs {
		score := req.Scores[questionID]
		if score <= 0 {
			score = 5
		}
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM question_bank WHERE id=?`, questionID).Scan(&exists); err != nil {
			return PaperComposition{}, err
		}
		if exists == 0 {
			return PaperComposition{}, errors.New("questionId is invalid: " + questionID)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO paper_composition_questions (composition_id,question_id,sort_order,score) VALUES (?,?,?,?)`, id, questionID, index+1, score); err != nil {
			return PaperComposition{}, err
		}
		total += score
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO paper_compositions (id,title,school_id,grade_id,grade_name,subject_id,subject,mode,status,question_count,total_score,created_by) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, id, req.Title, req.SchoolID, req.GradeID, req.GradeName, req.SubjectID, req.Subject, "manual", "draft", len(req.QuestionIDs), total, req.CreatedBy); err != nil {
		return PaperComposition{}, err
	}
	if err := tx.Commit(); err != nil {
		return PaperComposition{}, err
	}
	questions, _ := s.CompositionQuestions(ctx, id)
	return PaperComposition{ID: id, Title: req.Title, SchoolID: req.SchoolID, GradeID: req.GradeID, GradeName: req.GradeName, SubjectID: req.SubjectID, Subject: req.Subject, Mode: "manual", Status: "draft", QuestionCount: len(req.QuestionIDs), TotalScore: total, CreatedBy: req.CreatedBy, Questions: questions}, nil
}

func (s *Store) CompositionQuestions(ctx context.Context, compositionID string) ([]PaperCompositionQuestion, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT qb.id,COALESCE(qb.school_id,''),COALESCE(qb.grade_id,''),COALESCE(qb.subject_id,''),qb.subject,qb.question_type,qb.difficulty,qb.content,COALESCE(qb.answer,''),COALESCE(qb.analysis,''),qb.source,qb.status,pcq.sort_order,pcq.score FROM paper_composition_questions pcq JOIN question_bank qb ON qb.id=pcq.question_id WHERE pcq.composition_id=? ORDER BY pcq.sort_order`, compositionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PaperCompositionQuestion{}
	for rows.Next() {
		var item PaperCompositionQuestion
		if err := rows.Scan(&item.ID, &item.SchoolID, &item.GradeID, &item.SubjectID, &item.Subject, &item.QuestionType, &item.Difficulty, &item.Content, &item.Answer, &item.Analysis, &item.Source, &item.Status, &item.SortOrder, &item.Score); err != nil {
			return nil, err
		}
		item.Knowledge, _ = s.questionKnowledge(ctx, item.ID)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateAITask(ctx context.Context, taskType string, request map[string]any, sourceKey, sourceURL, ownerType, ownerID, createdBy string) (AITaskRecord, error) {
	if request == nil {
		request = map[string]any{}
	}
	id := fmt.Sprintf("aitask_%d", time.Now().UnixNano())
	payload, err := json.Marshal(request)
	if err != nil {
		return AITaskRecord{}, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO ai_tasks (id,task_type,status,provider,request_json,source_object_key,source_url,owner_type,owner_id,created_by) VALUES (?,?,?,?,?,?,?,?,?,?)`, id, taskType, "pending", "third_party_reserved", string(payload), sourceKey, sourceURL, ownerType, ownerID, createdBy); err != nil {
		return AITaskRecord{}, err
	}
	return AITaskRecord{ID: id, TaskType: taskType, Status: "pending", Provider: "third_party_reserved", Request: request, SourceObjectKey: sourceKey, SourceURL: sourceURL, OwnerType: ownerType, OwnerID: ownerID, CreatedBy: createdBy, Message: "第三方 AI 任务已创建，等待人工或调度器派发"}, nil
}

func (s *Store) CreateAnswerSheetUpload(ctx context.Context, compositionID, studentID, studentName string, file ScanFile, taskID string) (AnswerSheetUpload, error) {
	id := fmt.Sprintf("answer_%d", time.Now().UnixNano())
	if _, err := s.db.ExecContext(ctx, `INSERT INTO answer_sheet_uploads (id,composition_id,student_id,student_name,object_key,url,status,grading_task_id) VALUES (?,?,?,?,?,?,?,?)`, id, compositionID, studentID, studentName, file.Key, file.URL, "uploaded", taskID); err != nil {
		return AnswerSheetUpload{}, err
	}
	return AnswerSheetUpload{ID: id, CompositionID: compositionID, StudentID: studentID, StudentName: studentName, ObjectKey: file.Key, URL: file.URL, Status: "uploaded", GradingTaskID: taskID}, nil
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func paperCompositionFixture() []PaperComposition {
	return []PaperComposition{{ID: "paper_demo", Title: "分数与比例专项练习", Subject: "数学", GradeName: "六年级", Mode: "manual", Status: "draft", QuestionCount: 2, TotalScore: 10, Questions: []PaperCompositionQuestion{{QuestionBankItem: questionBankFixture()[0], SortOrder: 1, Score: 5}, {QuestionBankItem: questionBankFixture()[1], SortOrder: 2, Score: 5}}}}
}
