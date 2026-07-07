package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math"
	"net/http"
	"sort"
	"strings"
)

var errTemplateValidationFailed = errors.New("template validation failed")

func (app *App) handleTemplateValidation(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("templateID")
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	result, err := app.store.ValidateTemplateByID(r.Context(), templateID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "template validation failed"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (app *App) handleTemplateDiff(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("templateID")
	baseID := strings.TrimSpace(r.URL.Query().Get("baseTemplateId"))
	if templateID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "templateID is required"})
		return
	}
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	result, err := app.store.TemplateDiff(r.Context(), templateID, baseID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (app *App) handleTemplatePrint(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("templateID")
	var template PaperTemplate
	var err error
	if app.store != nil {
		template, err = app.store.Template(r.Context(), templateID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
	} else {
		for _, item := range templatesFixture() {
			if item.ID == templateID {
				template = item
			}
		}
		if template.ID == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s-print.html"`, template.ID))
	_, _ = w.Write([]byte(TemplatePrintableHTML(template)))
}

func (app *App) handleTemplateExport(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("templateID")
	var template PaperTemplate
	var err error
	if app.store != nil {
		template, err = app.store.Template(r.Context(), templateID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
	} else {
		for _, item := range templatesFixture() {
			if item.ID == templateID {
				template = item
			}
		}
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-template.json"`, template.ID))
	writeJSON(w, http.StatusOK, template)
}

func (app *App) handleScanExceptions(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusOK, ScanExceptionListResponse{Items: ScanExceptionsFromJobs(dashboardFixture().ScanQueue)})
		return
	}
	items, err := app.store.ScanExceptions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan exceptions query failed"})
		return
	}
	writeJSON(w, http.StatusOK, ScanExceptionListResponse{Items: items})
}

func (app *App) handleUpdateScanFileStatus(w http.ResponseWriter, r *http.Request) {
	if app.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database unavailable"})
		return
	}
	taskID := r.PathValue("taskID")
	var req ScanFileStatusRequest
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.FileKey) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "fileKey is required"})
		return
	}
	task, err := app.store.UpdateScanFileStatus(r.Context(), taskID, req)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scan file not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan file status update failed"})
		return
	}
	writeJSON(w, http.StatusOK, ScanTaskResponse{Status: "updated", Task: task})
}

func ValidateTemplate(template PaperTemplate) TemplateValidationResponse {
	result := TemplateValidationResponse{TemplateID: template.ID, Valid: true, Issues: []TemplateValidationIssue{}}
	add := func(code, severity, message string, q QuestionTemplate) {
		result.Issues = append(result.Issues, TemplateValidationIssue{Code: code, Severity: severity, Message: message, QuestionID: q.ID, QuestionNo: q.No, Page: q.Region.Page})
		if severity == "error" {
			result.Valid = false
		}
	}
	if strings.TrimSpace(template.Name) == "" {
		add("template_name_required", "error", "答题卡名称不能为空", QuestionTemplate{})
	}
	if len(template.Questions) == 0 {
		add("question_required", "error", "至少需要一道题目", QuestionTemplate{})
		return result
	}
	questionNos := map[string]QuestionTemplate{}
	for _, q := range template.Questions {
		if strings.TrimSpace(q.No) == "" {
			add("question_no_required", "error", "题号不能为空", q)
		}
		if existing, ok := questionNos[q.No]; ok {
			add("duplicate_question_no", "error", "题号重复："+q.No, q)
			add("duplicate_question_no", "error", "题号重复："+existing.No, existing)
		}
		questionNos[q.No] = q
		if q.Score <= 0 {
			add("score_required", "error", "题目分值必须大于 0", q)
		}
		if q.Region.Page <= 0 || q.Region.Width <= 0 || q.Region.Height <= 0 {
			add("region_required", "error", "题区页码、宽度和高度必须有效", q)
		}
		if isObjectiveQuestion(q.Type) && strings.TrimSpace(q.StandardAnswer) == "" {
			add("objective_answer_required", "error", "客观题发布前必须填写标准答案", q)
		}
		if !isObjectiveQuestion(q.Type) && len(q.ScoringRules) == 0 {
			add("subjective_rules_required", "warning", "主观题建议补充评分规则", q)
		}
	}
	for i := 0; i < len(template.Questions); i++ {
		for j := i + 1; j < len(template.Questions); j++ {
			left := template.Questions[i]
			right := template.Questions[j]
			if regionsOverlap(left.Region, right.Region) {
				add("region_overlap", "error", fmt.Sprintf("题区与第 %s 题重叠", right.No), left)
				add("region_overlap", "error", fmt.Sprintf("题区与第 %s 题重叠", left.No), right)
			}
		}
	}
	return result
}

func (s *Store) ValidateTemplateByID(ctx context.Context, templateID string) (TemplateValidationResponse, error) {
	template, err := s.Template(ctx, templateID)
	if err != nil {
		return TemplateValidationResponse{}, err
	}
	return ValidateTemplate(template), nil
}

func (s *Store) TemplateDiff(ctx context.Context, targetID string, baseID string) (TemplateDiffResponse, error) {
	target, err := s.Template(ctx, targetID)
	if err != nil {
		return TemplateDiffResponse{}, err
	}
	if strings.TrimSpace(baseID) == "" {
		baseID = target.ParentID
	}
	if strings.TrimSpace(baseID) == "" {
		return TemplateDiffResponse{}, errors.New("baseTemplateId is required when target has no parent")
	}
	base, err := s.Template(ctx, baseID)
	if err != nil {
		return TemplateDiffResponse{}, err
	}
	result := DiffTemplates(base, target)
	return result, nil
}

func DiffTemplates(base PaperTemplate, target PaperTemplate) TemplateDiffResponse {
	result := TemplateDiffResponse{BaseTemplateID: base.ID, TargetTemplateID: target.ID}
	baseByNo := map[string]QuestionTemplate{}
	targetByNo := map[string]QuestionTemplate{}
	for _, q := range base.Questions {
		baseByNo[q.No] = q
	}
	for _, q := range target.Questions {
		targetByNo[q.No] = q
		if _, ok := baseByNo[q.No]; !ok {
			result.AddedQuestions = append(result.AddedQuestions, q)
		}
	}
	for _, q := range base.Questions {
		targetQuestion, ok := targetByNo[q.No]
		if !ok {
			result.RemovedQuestions = append(result.RemovedQuestions, q)
			continue
		}
		fields := changedTemplateQuestionFields(q, targetQuestion)
		if len(fields) > 0 {
			result.ChangedQuestions = append(result.ChangedQuestions, TemplateDiffChange{QuestionNo: q.No, Fields: fields})
		}
	}
	result.Summary = append(result.Summary, fmt.Sprintf("新增 %d 题", len(result.AddedQuestions)))
	result.Summary = append(result.Summary, fmt.Sprintf("删除 %d 题", len(result.RemovedQuestions)))
	result.Summary = append(result.Summary, fmt.Sprintf("修改 %d 题", len(result.ChangedQuestions)))
	return result
}

func TemplatePrintableHTML(template PaperTemplate) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><title>`)
	b.WriteString(html.EscapeString(template.Name))
	b.WriteString(`</title><style>body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;margin:32px;color:#111827}.meta{color:#667085}.question{border:1px solid #d0d5dd;border-radius:6px;margin:14px 0;padding:12px;break-inside:avoid}.box{height:64px;border:1px dashed #98a2b3;margin-top:10px}.choice{display:flex;gap:18px;margin-top:10px}@media print{button{display:none}body{margin:12mm}}</style></head><body><button onclick="window.print()">打印</button>`)
	b.WriteString(`<h1>`)
	b.WriteString(html.EscapeString(template.Name))
	b.WriteString(`</h1><p class="meta">`)
	b.WriteString(html.EscapeString(fmt.Sprintf("%s · %s · 版本 %d · 总分 %d", template.Subject, template.Grade, template.Version, template.TotalScore)))
	b.WriteString(`</p><section>`)
	for _, q := range template.Questions {
		b.WriteString(`<article class="question"><strong>第 `)
		b.WriteString(html.EscapeString(q.No))
		b.WriteString(` 题</strong> <span class="meta">`)
		b.WriteString(html.EscapeString(fmt.Sprintf("%s · %.1f 分", q.Type, q.Score)))
		b.WriteString(`</span>`)
		if isObjectiveQuestion(q.Type) {
			b.WriteString(`<div class="choice"><span>○ A</span><span>○ B</span><span>○ C</span><span>○ D</span></div>`)
		} else {
			b.WriteString(`<div class="box"></div>`)
		}
		b.WriteString(`</article>`)
	}
	b.WriteString(`</section></body></html>`)
	return b.String()
}

func (s *Store) UpdateScanFileStatus(ctx context.Context, taskID string, req ScanFileStatusRequest) (ScanJob, error) {
	task, err := s.ScanTask(ctx, taskID)
	if err != nil {
		return ScanJob{}, err
	}
	found := false
	for index := range task.Files {
		if task.Files[index].Key != req.FileKey {
			continue
		}
		if strings.TrimSpace(req.Status) != "" {
			task.Files[index].Status = strings.TrimSpace(req.Status)
		}
		task.Files[index].FailureReason = strings.TrimSpace(req.FailureReason)
		if strings.TrimSpace(req.StudentID) != "" {
			task.Files[index].StudentID = strings.TrimSpace(req.StudentID)
		}
		if strings.TrimSpace(req.StudentName) != "" {
			task.Files[index].StudentName = strings.TrimSpace(req.StudentName)
		}
		if strings.TrimSpace(req.MatchStatus) != "" {
			task.Files[index].MatchStatus = strings.TrimSpace(req.MatchStatus)
		}
		found = true
		break
	}
	if !found {
		return ScanJob{}, sql.ErrNoRows
	}
	raw, err := marshalScanFiles(task.Files)
	if err != nil {
		return ScanJob{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE scan_jobs SET files_json=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, raw, taskID); err != nil {
		return ScanJob{}, err
	}
	_ = s.insertScanTaskLog(ctx, taskID, req.FileKey, "file_status", req.Status)
	return s.ScanTask(ctx, taskID)
}

func (s *Store) ScanExceptions(ctx context.Context) ([]ScanException, error) {
	jobs, err := s.ScanJobs(ctx)
	if err != nil {
		return nil, err
	}
	return ScanExceptionsFromJobs(jobs), nil
}

func ScanExceptionsFromJobs(jobs []ScanJob) []ScanException {
	items := []ScanException{}
	for _, job := range jobs {
		if job.QueueStatus == "failed" {
			items = append(items, ScanException{TaskID: job.ID, Title: job.Title, ClassName: job.ClassName, Status: job.Status, Reason: job.QueueMessage, QueueStatus: job.QueueStatus})
		}
		pageSeen := map[int]int{}
		for _, file := range job.Files {
			if file.Page > 0 {
				pageSeen[file.Page]++
			}
			reason := scanFileExceptionReason(file)
			if reason == "" {
				continue
			}
			items = append(items, ScanException{
				TaskID: job.ID, Title: job.Title, ClassName: job.ClassName, FileKey: file.Key, FileName: file.FileName,
				Page: file.Page, Status: file.Status, Reason: reason, StudentID: file.StudentID, StudentName: file.StudentName,
				MatchStatus: file.MatchStatus, QueueStatus: job.QueueStatus,
			})
		}
		if job.Pages > 0 && len(job.Files) < job.Pages {
			items = append(items, ScanException{TaskID: job.ID, Title: job.Title, ClassName: job.ClassName, Status: "missing_pages", Reason: fmt.Sprintf("扫描文件数量少于登记页数，缺少 %d 页/文件", job.Pages-len(job.Files))})
		}
		for page := 1; page <= job.Pages; page++ {
			if pageSeen[page] > 1 {
				items = append(items, ScanException{TaskID: job.ID, Title: job.Title, ClassName: job.ClassName, Page: page, Status: "duplicate_page", Reason: "页码重复"})
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TaskID == items[j].TaskID {
			return items[i].Page < items[j].Page
		}
		return items[i].TaskID > items[j].TaskID
	})
	return items
}

func scanFileExceptionReason(file ScanFile) string {
	if strings.TrimSpace(file.FailureReason) != "" {
		return file.FailureReason
	}
	switch strings.TrimSpace(file.MatchStatus) {
	case "pending", "failed", "unmatched":
		return "学生匹配待处理"
	}
	switch strings.TrimSpace(file.Status) {
	case "失败", "failed", "error":
		return "文件处理失败"
	case "missing_page":
		return "缺页"
	}
	return ""
}

func marshalScanFiles(files []ScanFile) (string, error) {
	raw, err := json.Marshal(files)
	return string(raw), err
}

func changedTemplateQuestionFields(left QuestionTemplate, right QuestionTemplate) []string {
	fields := []string{}
	if left.Type != right.Type {
		fields = append(fields, "type")
	}
	if math.Abs(left.Score-right.Score) > 0.001 {
		fields = append(fields, "score")
	}
	if left.StandardAnswer != right.StandardAnswer {
		fields = append(fields, "standardAnswer")
	}
	if strings.Join(left.ScoringRules, "\n") != strings.Join(right.ScoringRules, "\n") {
		fields = append(fields, "scoringRules")
	}
	if strings.Join(left.Knowledge, "\n") != strings.Join(right.Knowledge, "\n") {
		fields = append(fields, "knowledge")
	}
	if left.Region != right.Region {
		fields = append(fields, "region")
	}
	return fields
}

func regionsOverlap(left Region, right Region) bool {
	if left.Page <= 0 || right.Page <= 0 || left.Page != right.Page {
		return false
	}
	if left.Width <= 0 || left.Height <= 0 || right.Width <= 0 || right.Height <= 0 {
		return false
	}
	return left.X < right.X+right.Width &&
		left.X+left.Width > right.X &&
		left.Y < right.Y+right.Height &&
		left.Y+left.Height > right.Y
}

func isObjectiveQuestion(questionType string) bool {
	switch strings.TrimSpace(questionType) {
	case "single_choice", "choice", "judge", "fill_blank", "objective":
		return true
	default:
		return false
	}
}
