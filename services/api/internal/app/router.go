package app

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type App struct {
	config Config
	store  *Store
}

func NewApp() *App {
	return NewAppWithConfig(LoadConfig())
}

func NewAppWithConfig(config Config) *App {
	initTimeout := time.Duration(envInt("DB_INIT_TIMEOUT_SECONDS", 120)) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()

	store, err := OpenStore(ctx, config)
	if err != nil {
		log.Fatalf("database unavailable; API requires a real database and fixture mode is disabled: %v", err)
	}
	return &App{config: config, store: store}
}

func dashboardReadTimeout() time.Duration {
	seconds := envInt("DB_DASHBOARD_TIMEOUT_SECONDS", 2)
	if seconds < 1 {
		seconds = 1
	}
	return time.Duration(seconds) * time.Second
}

func NewRouter() http.Handler {
	return NewRouterWithConfig(LoadConfig())
}

func NewRouterWithConfig(config Config) http.Handler {
	app := NewAppWithConfig(config)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", app.handleHealth)
	mux.HandleFunc("POST /api/auth/login", app.handleLogin)
	mux.HandleFunc("GET /api/dashboard", app.handleDashboard)
	mux.HandleFunc("GET /api/grading/subjective/reviews", app.handleSubjectiveReviewQueue)
	mux.HandleFunc("GET /api/grading/subjective/current", app.handleCurrentSubjective)
	mux.HandleFunc("GET /api/grading/subjective/reviews/{reviewID}", app.handleReviewSubjective)
	mux.HandleFunc("GET /api/grading/subjective/history", app.handleSubjectiveHistory)
	mux.HandleFunc("POST /api/grading/subjective/decision", app.handleSubjectiveDecision)
	mux.HandleFunc("PATCH /api/grading/objective/exceptions/{exceptionID}", app.handleObjectiveExceptionDecision)
	mux.HandleFunc("GET /api/grading/policies/default", app.handleDefaultGradingPolicy)
	mux.HandleFunc("PUT /api/grading/policies/default", app.handleSaveDefaultGradingPolicy)
	mux.HandleFunc("POST /api/scan/uploads", app.handleScanUpload)
	mux.HandleFunc("GET /api/scan/tasks", app.handleScanTasks)
	mux.HandleFunc("POST /api/scan/tasks", app.handleCreateScanTask)
	mux.HandleFunc("GET /api/scan/tasks/{taskID}", app.handleScanTask)
	mux.HandleFunc("PATCH /api/scan/tasks/{taskID}/status", app.handleUpdateScanTaskStatus)
	mux.HandleFunc("GET /api/scan/tasks/{taskID}/worker-result", app.handleScanWorkerResult)
	mux.HandleFunc("POST /api/scan/tasks/{taskID}/worker-result", app.handleSaveScanWorkerResult)
	mux.HandleFunc("POST /api/scan/tasks/{taskID}/retry", app.handleRetryScanTask)
	mux.HandleFunc("POST /api/scan/tasks/{taskID}/match", app.handleMatchScanFile)
	mux.HandleFunc("PATCH /api/scan/tasks/{taskID}/files/status", app.handleUpdateScanFileStatus)
	mux.HandleFunc("GET /api/scan/tasks/{taskID}/preview", app.handleScanTaskPreview)
	mux.HandleFunc("GET /api/scan/exceptions", app.handleScanExceptions)
	mux.HandleFunc("GET /api/papers", app.handlePapers)
	mux.HandleFunc("DELETE /api/papers/{paperID}", app.handleDeletePaper)
	mux.HandleFunc("GET /api/templates", app.handleTemplates)
	mux.HandleFunc("POST /api/templates", app.handleCreateTemplate)
	mux.HandleFunc("POST /api/template-ai-suggestions", app.handleTemplateAISuggestionsForSource)
	mux.HandleFunc("GET /api/templates/{templateID}", app.handleTemplate)
	mux.HandleFunc("PUT /api/templates/{templateID}", app.handleUpdateTemplate)
	mux.HandleFunc("DELETE /api/templates/{templateID}", app.handleDeleteTemplate)
	mux.HandleFunc("POST /api/templates/{templateID}/copy", app.handleCopyTemplate)
	mux.HandleFunc("PUT /api/templates/{templateID}/status", app.handleUpdateTemplateStatus)
	mux.HandleFunc("GET /api/templates/{templateID}/validation", app.handleTemplateValidation)
	mux.HandleFunc("GET /api/templates/{templateID}/diff", app.handleTemplateDiff)
	mux.HandleFunc("GET /api/templates/{templateID}/print", app.handleTemplatePrint)
	mux.HandleFunc("GET /api/templates/{templateID}/export", app.handleTemplateExport)
	mux.HandleFunc("POST /api/templates/{templateID}/ai-suggestions", app.handleTemplateAISuggestions)
	mux.HandleFunc("GET /api/templates/{templateID}/regions", app.handleTemplateRegions)
	mux.HandleFunc("PUT /api/templates/{templateID}/regions", app.handleSaveTemplateRegions)
	mux.HandleFunc("POST /api/templates/{templateID}/regions", app.handleCreateTemplateRegion)
	mux.HandleFunc("PUT /api/templates/{templateID}/regions/{regionID}", app.handleUpdateTemplateRegion)
	mux.HandleFunc("DELETE /api/templates/{templateID}/regions/{regionID}", app.handleDeleteTemplateRegion)
	mux.HandleFunc("POST /api/exams", app.handlePublishExam)
	mux.HandleFunc("GET /api/exams/{examID}", app.handleExam)
	mux.HandleFunc("PATCH /api/exams/{examID}/students/{studentID}", app.handleExamAttendance)
	mux.HandleFunc("GET /api/analytics/classroom", app.handleClassroomAnalytics)
	mux.HandleFunc("POST /api/analytics/generate-scores", app.handleGenerateScores)
	mux.HandleFunc("GET /api/analytics/export/scores.csv", app.handleExportScores)
	mux.HandleFunc("GET /api/mistakes", app.handleWrongQuestions)
	mux.HandleFunc("GET /api/mistakes/{mistakeID}", app.handleWrongQuestion)
	mux.HandleFunc("PATCH /api/mistakes/{mistakeID}/knowledge-points", app.handleWrongQuestionKnowledge)
	mux.HandleFunc("POST /api/mistakes/repractice", app.handleCreateRepracticeTask)
	mux.HandleFunc("POST /api/mistakes/repractice/{taskID}/submissions", app.handleSubmitRepracticeTask)
	mux.HandleFunc("GET /api/learning/profile", app.handleLearningProfile)
	mux.HandleFunc("GET /api/reports/guardian", app.handleGuardianReport)
	mux.HandleFunc("GET /api/knowledge-points", app.handleKnowledgePoints)
	mux.HandleFunc("POST /api/knowledge-points", app.handleCreateKnowledgePoint)
	mux.HandleFunc("GET /api/question-bank", app.handleQuestionBank)
	mux.HandleFunc("POST /api/question-bank", app.handleCreateQuestionBankItem)
	mux.HandleFunc("GET /api/paper-compositions", app.handlePaperCompositions)
	mux.HandleFunc("POST /api/paper-compositions", app.handleCreatePaperComposition)
	mux.HandleFunc("POST /api/paper-compositions/{compositionID}/ai-compose-request", app.handleAIComposeRequest)
	mux.HandleFunc("POST /api/paper-analysis/blank-paper-uploads", app.handleBlankPaperUpload)
	mux.HandleFunc("POST /api/answer-sheets/uploads", app.handleAnswerSheetUpload)
	mux.HandleFunc("POST /api/grading/tasks", app.handleCreateGradingTask)
	mux.HandleFunc("GET /api/organization/graph", app.handleOrganizationGraph)
	mux.HandleFunc("POST /api/organization/{kind}", app.handleOrganizationCreate)
	mux.HandleFunc("POST /api/guardian/invitations", app.handleGuardianInvitation)
	mux.HandleFunc("POST /api/guardian/certifications", app.handleGuardianCertification)
	mux.HandleFunc("GET /api/guardian/certifications", app.handleCertificationList)
	mux.HandleFunc("PATCH /api/guardian/certifications/{certificationID}", app.handleCertificationDecision)
	mux.HandleFunc("GET /api/guardian/certifications/{certificationID}/logs", app.handleCertificationLogs)
	mux.HandleFunc("GET /api/portal/student", app.handleStudentPortal)
	mux.HandleFunc("GET /api/portal/guardian", app.handleGuardianPortal)
	mux.HandleFunc("POST /api/ai/capabilities/{capability}/requests", app.handleAICapabilityRequest)
	mux.HandleFunc("GET /api/ai/tasks", app.handleAITasks)
	mux.HandleFunc("GET /api/ai/tasks/{taskID}", app.handleAITask)
	mux.HandleFunc("POST /api/ai/tasks/{taskID}/dispatch", app.handleDispatchAITask)
	mux.HandleFunc("POST /api/ai/tasks/{taskID}/callback", app.handleAITaskCallback)
	mux.HandleFunc("GET /api/ai/providers", app.handleAIProviders)
	mux.HandleFunc("PUT /api/ai/providers/current", app.handleSaveAIProvider)
	mux.HandleFunc("DELETE /api/ai/providers/{providerName}", app.handleDeleteAIProvider)
	mux.HandleFunc("GET /api/system/config", app.handleSystemConfig)
	mux.HandleFunc("PUT /api/system/config", app.handleSaveSystemConfig)
	mux.HandleFunc("GET /api/dev/connections", app.handleDevConnections)
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(localUploadRoot()))))

	return withCORS(mux)
}

func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"database": app.store != nil,
		"config":   app.config.Public(),
	})
}

func (app *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	timeout := dashboardReadTimeout()
	dbCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	type dashboardResult struct {
		data DashboardResponse
		err  error
	}
	resultCh := make(chan dashboardResult, 1)
	go func() {
		data, err := app.store.Dashboard(dbCtx)
		resultCh <- dashboardResult{data: data, err: err}
	}()
	select {
	case result := <-resultCh:
		if result.err == nil {
			writeJSON(w, http.StatusOK, result.data)
			return
		}
		log.Printf("dashboard db query failed: %v", result.err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dashboard query failed"})
	case <-time.After(timeout):
		cancel()
		log.Printf("dashboard db query timed out after %s", timeout)
		writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "dashboard query timed out"})
	}
}

func (app *App) handleScanUpload(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one scan file is required"})
		return
	}
	files := make([]ScanFile, 0, len(headers))
	for _, header := range headers {
		file, err := saveScanUpload(app.config, header)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		files = append(files, file)
	}
	if app.store != nil {
		if err := app.store.SaveObjectFiles(r.Context(), files, "scan_upload", "scan_upload", ""); err != nil {
			log.Printf("scan upload object metadata save failed: %v", err)
		}
	}
	writeJSON(w, http.StatusCreated, ScanUploadResponse{Files: files})
}

func (app *App) handleScanTasks(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	tasks, err := app.store.ScanJobs(r.Context())
	if err != nil {
		log.Printf("scan tasks query failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan tasks query failed"})
		return
	}
	writeJSON(w, http.StatusOK, ScanTaskListResponse{Tasks: tasks})
}

func (app *App) handleScanTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "taskID is required"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	task, err := app.store.ScanTask(r.Context(), taskID)
	if err == nil {
		writeJSON(w, http.StatusOK, ScanTaskResponse{Status: "ok", Task: task})
		return
	}
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan task not found"})
		return
	}
	log.Printf("scan task query failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan task query failed"})
}

func (app *App) handlePapers(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	items, err := app.store.Papers(r.Context())
	if err != nil {
		log.Printf("papers query failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "papers query failed"})
		return
	}
	writeJSON(w, http.StatusOK, PaperManagementResponse{Items: items})
}

func (app *App) handleDeletePaper(w http.ResponseWriter, r *http.Request) {
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if !hasRole(auth, "教务管理员", "任课教师") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	paperID := strings.TrimSpace(r.PathValue("paperID"))
	if paperID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "paper id is required"})
		return
	}
	_, files, err := app.store.Paper(r.Context(), paperID)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "paper not found"})
		return
	}
	if err != nil {
		log.Printf("paper query failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "paper query failed"})
		return
	}
	deletedKeys := []string{}
	for _, file := range files {
		if strings.TrimSpace(file.Key) == "" {
			continue
		}
		if err := deleteStoredScanFile(app.config, file); err != nil {
			log.Printf("paper object delete failed: paper=%s key=%s err=%v", paperID, file.Key, err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "paper object delete failed", "detail": err.Error()})
			return
		}
		deletedKeys = append(deletedKeys, file.Key)
	}
	if err := app.store.DeletePaper(r.Context(), paperID, files); err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "paper not found"})
			return
		}
		log.Printf("paper delete failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "paper delete failed"})
		return
	}
	writeJSON(w, http.StatusOK, PaperDeleteResponse{Status: "deleted", DeletedID: paperID, DeletedKeys: deletedKeys})
}

func (app *App) handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role is required"})
		return
	}
	config, err := app.store.SystemConfig(r.Context())
	if err != nil {
		log.Printf("system config query failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "system config query failed"})
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (app *App) handleSaveSystemConfig(w http.ResponseWriter, r *http.Request) {
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role is required"})
		return
	}
	var req SystemConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	saved, err := app.store.SaveSystemConfig(r.Context(), req)
	if err != nil {
		log.Printf("system config save failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "system config save failed"})
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (app *App) handleCreateScanTask(w http.ResponseWriter, r *http.Request) {
	var req ScanTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.ClassName = strings.TrimSpace(req.ClassName)
	req.ScanType = normalizeScanType(req.ScanType)
	req.AssignmentID = strings.TrimSpace(req.AssignmentID)
	req.ExamID = strings.TrimSpace(req.ExamID)
	req.TemplateID = strings.TrimSpace(req.TemplateID)
	req.Notes = strings.TrimSpace(req.Notes)
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	if req.ExamID != "" {
		exam, err := app.store.Exam(r.Context(), req.ExamID)
		if err != nil {
			if err == sql.ErrNoRows {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "exam not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exam query failed"})
			return
		}
		if exam.Status != "published" && exam.Status != "ended" && exam.Status != "scanning" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "scan task must bind a published or ended exam"})
			return
		}
		req.Title = exam.Name
		req.ClassName = exam.ClassName
		req.TemplateID = exam.TemplateID
		req.TemplateVersion = exam.TemplateVersion
		if assignmentID, err := app.store.AssignmentIDForExam(r.Context(), exam.ID); err == nil {
			req.AssignmentID = assignmentID
		}
	}
	if req.Title == "" || req.ClassName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and className are required"})
		return
	}
	if req.ScanType == "answer_sheet" && req.TemplateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "answer sheet scan must bind a template"})
		return
	}
	if req.Pages <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pages must be greater than 0"})
		return
	}
	if len(req.Files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "files are required"})
		return
	}
	var template PaperTemplate
	if req.TemplateID != "" {
		var err error
		template, err = app.store.Template(r.Context(), req.TemplateID)
		if err != nil {
			if err == sql.ErrNoRows {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
				return
			}
			log.Printf("scan task template query failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template query failed"})
			return
		}
		if template.Status != "published" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "scan task must bind a published template"})
			return
		}
		req.TemplateVersion = template.Version
	}
	task, err := app.store.CreateScanTask(r.Context(), req)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		if err == errTemplateNotPublished {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "scan task must bind a published template"})
			return
		}
		log.Printf("scan task create failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan task create failed"})
		return
	}
	queueID, queueErr := enqueueScanTask(r.Context(), app.config.Redis, scanQueuePayload(task, template))
	if queueErr != nil {
		message := queueErr.Error()
		if err := app.store.UpdateScanQueueStatus(r.Context(), task.ID, "failed", message); err != nil {
			log.Printf("scan queue status update failed: %v", err)
		}
		task.QueueStatus = "failed"
		task.QueueMessage = message
		task.Status = "队列投递失败"
		task.FailureReason = message
		writeJSON(w, http.StatusCreated, ScanTaskResponse{Status: "created", QueueError: message, Task: task})
		return
	}
	if err := app.store.UpdateScanQueueStatus(r.Context(), task.ID, "queued", queueID); err != nil {
		log.Printf("scan queue status update failed: %v", err)
	}
	task.QueueStatus = "queued"
	task.QueueMessage = queueID
	writeJSON(w, http.StatusCreated, ScanTaskResponse{Status: "created", QueueID: queueID, Task: task})
}

func (app *App) handleUpdateScanTaskStatus(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "taskID is required"})
		return
	}
	var req ScanTaskStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	req.FailureReason = strings.TrimSpace(req.FailureReason)
	if req.Status == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status is required"})
		return
	}
	task, err := app.store.UpdateScanTaskStatus(r.Context(), taskID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan task not found"})
			return
		}
		log.Printf("scan task status update failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan task status update failed"})
		return
	}
	writeJSON(w, http.StatusOK, ScanTaskResponse{Status: "updated", Task: task})
}

func (app *App) handleSaveScanWorkerResult(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "taskID is required"})
		return
	}
	var req ScanWorkerResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	task, err := app.store.SaveScanWorkerResult(r.Context(), taskID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan task not found"})
			return
		}
		log.Printf("scan worker result save failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan worker result save failed"})
		return
	}
	record, err := app.store.ScanWorkerResult(r.Context(), taskID)
	if err != nil {
		log.Printf("scan worker result query failed: %v", err)
		writeJSON(w, http.StatusOK, ScanWorkerResultResponse{Status: "saved", Task: task})
		return
	}
	writeJSON(w, http.StatusOK, ScanWorkerResultResponse{Status: "saved", Task: task, Result: &record})
}

func (app *App) handleScanWorkerResult(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "taskID is required"})
		return
	}
	task, err := app.store.ScanTask(r.Context(), taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan task not found"})
			return
		}
		log.Printf("scan task query failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan task query failed"})
		return
	}
	record, err := app.store.ScanWorkerResult(r.Context(), taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan worker result not found"})
			return
		}
		log.Printf("scan worker result query failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan worker result query failed"})
		return
	}
	writeJSON(w, http.StatusOK, ScanWorkerResultResponse{Status: "ok", Task: task, Result: &record})
}

func (app *App) handleRetryScanTask(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "taskID is required"})
		return
	}
	var req ScanTaskRetryRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	req.FileKey = strings.TrimSpace(req.FileKey)
	task, err := app.store.RetryScanTask(r.Context(), taskID, req.FileKey)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan task not found"})
			return
		}
		log.Printf("scan task retry failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan task retry failed"})
		return
	}
	var template PaperTemplate
	if task.TemplateID != "" {
		var err error
		template, err = app.store.Template(r.Context(), task.TemplateID)
		if err != nil {
			log.Printf("scan task retry template query failed: %v", err)
		}
	}
	queueTask := task
	if req.FileKey != "" {
		queueTask.Files = filterScanFiles(task.Files, req.FileKey)
	}
	queueID, queueErr := enqueueScanTask(r.Context(), app.config.Redis, scanQueuePayload(queueTask, template))
	if queueErr != nil {
		message := queueErr.Error()
		if err := app.store.UpdateScanQueueStatus(r.Context(), task.ID, "failed", message); err != nil {
			log.Printf("scan retry queue status update failed: %v", err)
		}
		task.QueueStatus = "failed"
		task.QueueMessage = message
		task.Status = "队列投递失败"
		task.FailureReason = message
		writeJSON(w, http.StatusOK, ScanTaskResponse{Status: "retried", QueueError: message, Task: task})
		return
	}
	if err := app.store.UpdateScanQueueStatus(r.Context(), task.ID, "queued", queueID); err != nil {
		log.Printf("scan retry queue status update failed: %v", err)
	}
	task, _ = app.store.ScanTask(r.Context(), task.ID)
	writeJSON(w, http.StatusOK, ScanTaskResponse{Status: "retried", QueueID: queueID, Task: task})
}

func (app *App) handleMatchScanFile(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "taskID is required"})
		return
	}
	var req ScanFileMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if strings.TrimSpace(req.FileKey) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fileKey is required"})
		return
	}
	task, err := app.store.MatchScanFile(r.Context(), taskID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan file not found"})
			return
		}
		log.Printf("scan file match failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan file match failed"})
		return
	}
	writeJSON(w, http.StatusOK, ScanTaskResponse{Status: "matched", Task: task})
}

func (app *App) handleScanTaskPreview(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "taskID is required"})
		return
	}
	if app.store != nil {
		task, err := app.store.ScanTask(r.Context(), taskID)
		if err == nil {
			writeJSON(w, http.StatusOK, ScanTaskPreviewResponse{Task: task, Files: task.Files})
			return
		}
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan task not found"})
			return
		}
		log.Printf("scan preview query failed: %v", err)
	}
	for _, task := range dashboardFixture().ScanQueue {
		if task.ID == taskID {
			writeJSON(w, http.StatusOK, ScanTaskPreviewResponse{Task: task, Files: task.Files})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan task not found"})
}

func filterScanFiles(files []ScanFile, fileKey string) []ScanFile {
	if fileKey == "" {
		return files
	}
	filtered := []ScanFile{}
	for _, file := range files {
		if file.Key == fileKey {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func (app *App) handleSubjectiveReviewQueue(w http.ResponseWriter, r *http.Request) {
	if app.store != nil {
		items, err := app.store.ReviewQueue(r.Context())
		if err == nil {
			writeJSON(w, http.StatusOK, ReviewQueueResponse{Items: items})
			return
		}
		log.Printf("subjective review queue db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, ReviewQueueResponse{Items: dashboardFixture().ReviewQueue})
}

func (app *App) handleCurrentSubjective(w http.ResponseWriter, r *http.Request) {
	if app.store != nil {
		data, err := app.store.CurrentSubjective(r.Context())
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no pending subjective reviews"})
			return
		}
		log.Printf("current subjective db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, subjectiveFixture())
}

func (app *App) handleSubjectiveHistory(w http.ResponseWriter, r *http.Request) {
	submissionID := strings.TrimSpace(r.URL.Query().Get("submissionId"))
	questionID := strings.TrimSpace(r.URL.Query().Get("questionId"))
	if submissionID == "" || questionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "submissionId and questionId are required"})
		return
	}
	if app.store != nil {
		items, err := app.store.GradingHistory(r.Context(), submissionID, questionID)
		if err == nil {
			writeJSON(w, http.StatusOK, GradingHistoryResponse{Items: items})
			return
		}
		log.Printf("subjective history db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, GradingHistoryResponse{Items: []GradingHistoryItem{}})
}

func (app *App) handleReviewSubjective(w http.ResponseWriter, r *http.Request) {
	reviewID := r.PathValue("reviewID")
	if reviewID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reviewID is required"})
		return
	}
	if app.store != nil {
		data, err := app.store.SubjectiveByReviewID(r.Context(), reviewID)
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "subjective review not found"})
			return
		}
		log.Printf("subjective review db query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, subjectiveFixture())
}

func (app *App) handleSubjectiveDecision(w http.ResponseWriter, r *http.Request) {
	var req GradingDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.SubmissionID == "" || req.QuestionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "submissionId and questionId are required"})
		return
	}
	if app.store != nil {
		auth, err := app.requestAuth(r.Context(), r)
		if err != nil || !requireAuth(w, auth) {
			return
		}
		if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
			return
		}
		if req.ActorName == "" {
			req.ActorName = auth.UserID
		}
		data, err := app.store.SaveSubjectiveDecision(r.Context(), req)
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		if strings.Contains(err.Error(), "final score") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		log.Printf("subjective decision db write failed: %v", err)
	}
	writeJSON(w, http.StatusOK, GradingDecisionResponse{
		Status:       "saved",
		FinalScore:   req.FinalScore,
		NextQuestion: "q_018",
	})
}

func (app *App) handleObjectiveExceptionDecision(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("exceptionID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid exception id"})
		return
	}
	var req ObjectiveReviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusOK, ObjectiveReviewDecisionResponse{Status: "resolved", Exception: ObjectiveReviewException{ID: id, Status: "resolved", SuggestedScore: req.Score}})
		return
	}
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	item, err := app.store.ResolveObjectiveException(r.Context(), id, req)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "objective exception not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ObjectiveReviewDecisionResponse{Status: item.Status, Exception: item})
}

func (app *App) handleDefaultGradingPolicy(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusOK, defaultAutoGradingPolicy())
		return
	}
	policy, err := app.store.DefaultGradingPolicy(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "grading policy query failed"})
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (app *App) handleSaveDefaultGradingPolicy(w http.ResponseWriter, r *http.Request) {
	var req AutoGradingPolicy
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusOK, normalizePolicy(&req))
		return
	}
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	policy, err := app.store.SaveDefaultGradingPolicy(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "grading policy save failed"})
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (app *App) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	data, err := app.store.Templates(r.Context())
	if err != nil {
		log.Printf("templates db query failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "templates query failed"})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (app *App) handleTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	data, err := app.store.Template(r.Context(), templateID)
	if err == nil {
		writeJSON(w, http.StatusOK, data)
		return
	}
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
		return
	}
	log.Printf("template db query failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template query failed"})
}

func (app *App) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	template, ok := decodeTemplateBody(w, r)
	if !ok {
		return
	}
	data, err := app.store.CreateTemplate(r.Context(), template)
	if err != nil {
		log.Printf("template create failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template create failed"})
		return
	}
	writeJSON(w, http.StatusCreated, TemplateMutationResponse{Status: "created", Template: data})
}

func (app *App) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	template, ok := decodeTemplateBody(w, r)
	if !ok {
		return
	}
	data, err := app.store.UpdateTemplate(r.Context(), templateID, template)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		if err == errTemplateLocked {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "published or disabled template must be copied before editing"})
			return
		}
		log.Printf("template update failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template update failed"})
		return
	}
	writeJSON(w, http.StatusOK, TemplateMutationResponse{Status: "updated", Template: data})
}

func (app *App) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	if err := app.store.DeleteTemplate(r.Context(), templateID); err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		log.Printf("template delete failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template delete failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (app *App) handleCopyTemplate(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	data, err := app.store.CopyTemplate(r.Context(), templateID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		log.Printf("template copy failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template copy failed"})
		return
	}
	writeJSON(w, http.StatusCreated, TemplateMutationResponse{Status: "copied", Template: data})
}

func (app *App) handleUpdateTemplateStatus(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	var req TemplateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if !validTemplateStatus(req.Status) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid template status"})
		return
	}
	data, err := app.store.UpdateTemplateStatus(r.Context(), templateID, req.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		if err == errTemplateValidationFailed {
			validation, _ := app.store.ValidateTemplateByID(r.Context(), templateID)
			writeJSON(w, http.StatusConflict, validation)
			return
		}
		log.Printf("template status update failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template status update failed"})
		return
	}
	writeJSON(w, http.StatusOK, TemplateMutationResponse{Status: "status_updated", Template: data})
}

func (app *App) handleTemplateAISuggestions(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	var req TemplateAISuggestionRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	var template PaperTemplate
	var err error
	template, err = app.store.Template(r.Context(), templateID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		log.Printf("template ai suggestion query failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template query failed"})
		return
	}
	if req.PaperName == "" {
		req.PaperName = template.Name
	}
	if req.SourceFileURL == "" {
		req.SourceFileURL = template.SourceFileURL
	}
	suggestion, err := app.templateAISuggestions(r.Context(), template, req)
	if errors.Is(err, errAIProviderConfigRequired) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "ai provider is not configured"})
		return
	}
	if err != nil {
		log.Printf("template ai suggestion provider failed: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "ai provider request failed", "detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, suggestion)
}

func (app *App) handleTemplateAISuggestionsForSource(w http.ResponseWriter, r *http.Request) {
	var req TemplateAISuggestionRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	template := PaperTemplate{
		ID:            "tpl_ai_source",
		Name:          strings.TrimSpace(req.PaperName),
		Subject:       strings.TrimSpace(req.Subject),
		Grade:         strings.TrimSpace(req.Grade),
		TotalScore:    req.TargetScore,
		SourceFileURL: strings.TrimSpace(req.SourceFileURL),
		Status:        "draft",
		Version:       1,
	}
	if template.Name == "" {
		template.Name = "未命名试卷"
	}
	suggestion, err := app.templateAISuggestions(r.Context(), template, req)
	if errors.Is(err, errAIProviderConfigRequired) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "ai provider is not configured"})
		return
	}
	if err != nil {
		log.Printf("template ai suggestion provider failed: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "ai provider request failed", "detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, suggestion)
}

func (app *App) templateAISuggestions(ctx context.Context, template PaperTemplate, req TemplateAISuggestionRequest) (TemplateAISuggestionResponse, error) {
	if req.PaperName == "" {
		req.PaperName = template.Name
	}
	if req.SourceFileURL == "" {
		req.SourceFileURL = template.SourceFileURL
	}
	if req.Subject == "" {
		req.Subject = template.Subject
	}
	if req.Grade == "" {
		req.Grade = template.Grade
	}
	if req.TargetScore == 0 {
		req.TargetScore = template.TotalScore
	}
	return app.generateTemplateAISuggestionWithProvider(ctx, template, req)
}

func (app *App) handleTemplateRegions(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	data, err := app.store.Template(r.Context(), templateID)
	if err == nil {
		writeJSON(w, http.StatusOK, data.Questions)
		return
	}
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
		return
	}
	log.Printf("template regions db query failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template regions query failed"})
}

func (app *App) handleSaveTemplateRegions(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	var req TemplateRegionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	data, err := app.store.SaveTemplateQuestions(r.Context(), templateID, req.Questions)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		if err == errTemplateLocked {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "published or disabled template must be copied before editing"})
			return
		}
		log.Printf("template regions save failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template regions save failed"})
		return
	}
	writeJSON(w, http.StatusOK, TemplateMutationResponse{Status: "regions_saved", Template: data})
}

func (app *App) handleCreateTemplateRegion(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	question, ok := decodeQuestionTemplateBody(w, r)
	if !ok {
		return
	}
	savedQuestion, template, err := app.store.CreateTemplateQuestion(r.Context(), templateID, question)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		if err == errTemplateLocked {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "published or disabled template must be copied before editing"})
			return
		}
		log.Printf("template region create failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template region create failed"})
		return
	}
	writeJSON(w, http.StatusCreated, TemplateRegionMutationResponse{Status: "created", Question: savedQuestion, Template: template})
}

func (app *App) handleUpdateTemplateRegion(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	regionID := r.PathValue("regionID")
	if templateID == "" || regionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID and regionID are required"})
		return
	}
	question, ok := decodeQuestionTemplateBody(w, r)
	if !ok {
		return
	}
	savedQuestion, template, err := app.store.UpdateTemplateQuestion(r.Context(), templateID, regionID, question)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template region not found"})
			return
		}
		if err == errTemplateLocked {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "published or disabled template must be copied before editing"})
			return
		}
		log.Printf("template region update failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template region update failed"})
		return
	}
	writeJSON(w, http.StatusOK, TemplateRegionMutationResponse{Status: "updated", Question: savedQuestion, Template: template})
}

func (app *App) handleDeleteTemplateRegion(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	templateID := r.PathValue("templateID")
	regionID := r.PathValue("regionID")
	if templateID == "" || regionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID and regionID are required"})
		return
	}
	template, err := app.store.DeleteTemplateQuestion(r.Context(), templateID, regionID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template region not found"})
			return
		}
		if err == errTemplateLocked {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "published or disabled template must be copied before editing"})
			return
		}
		log.Printf("template region delete failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template region delete failed"})
		return
	}
	writeJSON(w, http.StatusOK, TemplateMutationResponse{Status: "deleted", Template: template})
}

func (app *App) handlePublishExam(w http.ResponseWriter, r *http.Request) {
	var req ExamPublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusCreated, ExamRecord{ID: fmt.Sprintf("exam_%d", time.Now().UnixMilli()), Name: req.Name, SchoolID: req.SchoolID, Subject: req.Subject, Grade: req.Grade, ClassID: req.ClassID, TeacherID: req.TeacherID, TemplateID: req.TemplateID, Status: "published", RosterLocked: true, GradingPolicy: normalizePolicy(req.GradingPolicy)})
		return
	}
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	if req.SchoolID == "" {
		req.SchoolID = auth.SchoolID
	}
	if req.TeacherID == "" {
		req.TeacherID = auth.TeacherID
	}
	exam, err := app.store.PublishExam(r.Context(), req)
	if err != nil {
		if err == errTemplateNotPublished {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "exam must bind a published template"})
			return
		}
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "referenced template, class or student not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, exam)
}

func (app *App) handleExam(w http.ResponseWriter, r *http.Request) {
	examID := r.PathValue("examID")
	if examID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "examID is required"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusOK, ExamRecord{ID: examID, Name: "六年级数学期中卷", Status: "published", TemplateID: "tpl_001", TemplateVersion: 1, RosterLocked: true, GradingPolicy: defaultAutoGradingPolicy()})
		return
	}
	exam, err := app.store.Exam(r.Context(), examID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "exam not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "exam query failed"})
		return
	}
	writeJSON(w, http.StatusOK, exam)
}

func (app *App) handleExamAttendance(w http.ResponseWriter, r *http.Request) {
	examID := r.PathValue("examID")
	studentID := r.PathValue("studentID")
	if examID == "" || studentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "examID and studentID are required"})
		return
	}
	var req ExamAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusOK, ExamRecord{ID: examID, Status: "published", RosterLocked: true, Students: []ExamStudent{{StudentID: studentID, Status: req.Status}}, GradingPolicy: defaultAutoGradingPolicy()})
		return
	}
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	exam, err := app.store.UpdateExamAttendance(r.Context(), examID, studentID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "exam student not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, exam)
}

func (app *App) handleClassroomAnalytics(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	timeout := dashboardReadTimeout()
	dbCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	type analyticsResult struct {
		data ClassroomAnalytics
		err  error
	}
	resultCh := make(chan analyticsResult, 1)
	go func() {
		data, err := app.store.ClassroomAnalytics(dbCtx)
		resultCh <- analyticsResult{data: data, err: err}
	}()
	select {
	case result := <-resultCh:
		if result.err == nil {
			writeJSON(w, http.StatusOK, result.data)
			return
		}
		log.Printf("classroom analytics db query failed: %v", result.err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "classroom analytics query failed"})
	case <-time.After(timeout):
		cancel()
		log.Printf("classroom analytics db query timed out after %s", timeout)
		writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "classroom analytics query timed out"})
	}
}

func (app *App) handleGenerateScores(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
		return
	}
	className := strings.TrimSpace(r.URL.Query().Get("className"))
	result, err := app.store.GenerateExamScores(r.Context(), className)
	if err != nil {
		if err == errPendingReviewBlocksScores {
			result.Blocked = true
			result.Reason = "pending objective or subjective reviews must be resolved before score generation"
			writeJSON(w, http.StatusConflict, result)
			return
		}
		log.Printf("generate scores failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "generate scores failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (app *App) handleExportScores(w http.ResponseWriter, r *http.Request) {
	data := analyticsFixture()
	if app.store != nil {
		if dbData, err := app.store.ClassroomAnalytics(r.Context()); err == nil {
			data = dbData
		} else {
			log.Printf("score export analytics query failed: %v", err)
		}
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="scores.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"学生", "班级", "总分", "排名", "薄弱知识点"})
	for _, item := range data.StudentScores {
		_ = writer.Write([]string{
			item.StudentName,
			item.ClassName,
			fmt.Sprintf("%.1f", item.Score),
			fmt.Sprintf("%d", item.Rank),
			strings.Join(item.Weakness, "、"),
		})
	}
	writer.Flush()
}

func (app *App) handleWrongQuestions(w http.ResponseWriter, r *http.Request) {
	filters := WrongQuestionFilters{
		Paper:       strings.TrimSpace(r.URL.Query().Get("paper")),
		ClassName:   strings.TrimSpace(r.URL.Query().Get("className")),
		StudentName: strings.TrimSpace(r.URL.Query().Get("studentName")),
		Knowledge:   strings.TrimSpace(r.URL.Query().Get("knowledge")),
		ErrorType:   strings.TrimSpace(r.URL.Query().Get("errorType")),
		Search:      strings.TrimSpace(r.URL.Query().Get("search")),
	}
	if app.store != nil {
		auth, err := app.requestAuth(r.Context(), r)
		if err != nil || !requireAuth(w, auth) {
			return
		}
		if auth.StudentID != "" || auth.GuardianID != "" {
			studentID := auth.StudentID
			if auth.GuardianID != "" {
				studentID = strings.TrimSpace(r.URL.Query().Get("studentId"))
			}
			if studentID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "studentId is required"})
				return
			}
			allowed, err := app.store.CanAccessStudent(r.Context(), auth, studentID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "student scope check failed"})
				return
			}
			if !allowed {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "student is outside current user scope"})
				return
			}
			studentName, err := app.store.StudentName(r.Context(), studentID)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "student not found"})
				return
			}
			filters.StudentName = studentName
		} else if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
			return
		}
		items, err := app.store.WrongQuestions(r.Context(), filters)
		if err == nil {
			writeJSON(w, http.StatusOK, WrongQuestionListResponse{Items: items})
			return
		}
		log.Printf("wrong questions query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, WrongQuestionListResponse{Items: wrongQuestionsFixture()})
}

func (app *App) handleWrongQuestion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("mistakeID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mistake id"})
		return
	}
	if app.store != nil {
		item, err := app.store.WrongQuestion(r.Context(), id)
		if err == nil {
			writeJSON(w, http.StatusOK, item)
			return
		}
		if err != sql.ErrNoRows {
			log.Printf("wrong question query failed: %v", err)
		}
	}
	for _, item := range wrongQuestionsFixture() {
		if item.ID == id {
			writeJSON(w, http.StatusOK, item)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "mistake not found"})
}

func (app *App) handleCreateRepracticeTask(w http.ResponseWriter, r *http.Request) {
	var req RepracticeTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.WrongQuestionIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "wrongQuestionIds is required"})
		return
	}
	if app.store != nil {
		result, err := app.store.CreateRepracticeTask(r.Context(), req)
		if err == nil {
			writeJSON(w, http.StatusCreated, result)
			return
		}
		log.Printf("repractice task create failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "repractice task create failed"})
		return
	}
	writeJSON(w, http.StatusCreated, RepracticeTaskResponse{Status: "created", TaskID: fmt.Sprintf("repractice_%d", time.Now().UnixMilli()), LinkedCount: len(req.WrongQuestionIDs), Knowledge: []string{}})
}

func (app *App) handleSubmitRepracticeTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskID")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "taskID is required"})
		return
	}
	var req RepracticeSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusCreated, RepracticeSubmissionResponse{Status: "submitted", TaskID: taskID, Submitted: len(req.Answers)})
		return
	}
	auth, err := app.requestAuth(r.Context(), r)
	if err != nil || !requireAuth(w, auth) {
		return
	}
	if req.StudentID == "" {
		req.StudentID = auth.StudentID
	}
	allowed, err := app.store.CanAccessStudent(r.Context(), auth, req.StudentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "student scope check failed"})
		return
	}
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "student is outside current user scope"})
		return
	}
	result, err := app.store.SubmitRepractice(r.Context(), taskID, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (app *App) handleLearningProfile(w http.ResponseWriter, r *http.Request) {
	className := strings.TrimSpace(r.URL.Query().Get("className"))
	if app.store != nil {
		auth, err := app.requestAuth(r.Context(), r)
		if err != nil || !requireAuth(w, auth) {
			return
		}
		if auth.TeacherID == "" && !hasRole(auth, "教务管理员") {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "teacher role is required"})
			return
		}
		data, err := app.store.LearningProfile(r.Context(), className)
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("learning profile query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, learningProfileFixture())
}

func (app *App) handleGuardianReport(w http.ResponseWriter, r *http.Request) {
	studentName := strings.TrimSpace(r.URL.Query().Get("studentName"))
	if app.store != nil {
		auth, err := app.requestAuth(r.Context(), r)
		if err != nil || !requireAuth(w, auth) {
			return
		}
		studentID := strings.TrimSpace(r.URL.Query().Get("studentId"))
		if studentID == "" {
			studentID = auth.StudentID
		}
		if studentID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "studentId is required"})
			return
		}
		allowed, err := app.store.CanAccessStudent(r.Context(), auth, studentID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "student scope check failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "student is outside current user scope"})
			return
		}
		studentName, err = app.store.StudentName(r.Context(), studentID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "student not found"})
			return
		}
		data, err := app.store.GuardianReport(r.Context(), studentName)
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("guardian report query failed: %v", err)
	}
	writeJSON(w, http.StatusOK, guardianReportFixture(studentName))
}

func (app *App) handleDevConnections(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, CheckDevConnections(app.config))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-ID, X-School-ID, X-Role, X-Student-ID, X-Guardian-ID, X-Teacher-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decodeTemplateBody(w http.ResponseWriter, r *http.Request) (PaperTemplate, bool) {
	var template PaperTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return PaperTemplate{}, false
	}
	if template.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "template name is required"})
		return PaperTemplate{}, false
	}
	return template, true
}

func decodeQuestionTemplateBody(w http.ResponseWriter, r *http.Request) (QuestionTemplate, bool) {
	var question QuestionTemplate
	if err := json.NewDecoder(r.Body).Decode(&question); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return QuestionTemplate{}, false
	}
	if question.Type == "" {
		question.Type = "subjective"
	}
	if question.Score == 0 {
		question.Score = 10
	}
	if question.Region.Width <= 0 || question.Region.Height <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "region width and height are required"})
		return QuestionTemplate{}, false
	}
	return question, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	data = normalizeAPIError(status, data)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type APIErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func normalizeAPIError(status int, data any) any {
	if status < http.StatusBadRequest {
		return data
	}
	message := ""
	field := ""
	switch value := data.(type) {
	case map[string]string:
		message = value["error"]
		field = value["field"]
	case map[string]any:
		if raw, ok := value["error"].(string); ok {
			message = raw
		}
		if raw, ok := value["field"].(string); ok {
			field = raw
		}
	}
	if message == "" {
		return data
	}
	return APIErrorResponse{
		Error: APIError{
			Code:    apiErrorCode(status, message),
			Message: message,
			Field:   field,
		},
	}
}

func apiErrorCode(status int, message string) string {
	normalized := strings.ToLower(message)
	switch {
	case status == http.StatusBadRequest && strings.Contains(normalized, "required"):
		return "VALIDATION_REQUIRED"
	case status == http.StatusBadRequest:
		return "BAD_REQUEST"
	case status == http.StatusForbidden:
		return "FORBIDDEN"
	case status == http.StatusNotFound:
		return "NOT_FOUND"
	case status == http.StatusConflict:
		return "CONFLICT"
	case status == http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case status >= http.StatusInternalServerError:
		return "INTERNAL_ERROR"
	default:
		return "REQUEST_ERROR"
	}
}
