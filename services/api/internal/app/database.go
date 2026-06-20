package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

type Store struct {
	db *sql.DB
}

var errTemplateLocked = errors.New("template is not draft")
var errTemplateNotPublished = errors.New("template is not published")

func OpenStore(ctx context.Context, config Config) (*Store, error) {
	autoMigrate := envBool("AUTO_MIGRATE", true)
	if autoMigrate {
		rootDSN := mysqlConfig(config, "").FormatDSN()
		rootDB, err := sql.Open("mysql", rootDSN)
		if err != nil {
			return nil, err
		}
		defer rootDB.Close()

		if _, err := rootDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+config.MySQL.Database+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
			return nil, err
		}
	}

	dsn := mysqlConfig(config, config.MySQL.Database).FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(12)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	store := &Store{db: db}
	if autoMigrate {
		if err := store.Migrate(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
		if err := store.Seed(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return store, nil
}

func mysqlConfig(config Config, database string) *mysql.Config {
	return &mysql.Config{
		User:                     config.MySQL.User,
		Passwd:                   config.MySQL.Password,
		Net:                      "tcp",
		Addr:                     config.MySQL.Host + ":" + config.MySQL.Port,
		DBName:                   database,
		ParseTime:                true,
		Loc:                      time.Local,
		Timeout:                  10 * time.Second,
		ReadTimeout:              10 * time.Second,
		WriteTimeout:             10 * time.Second,
		AllowNativePasswords:     true,
		AllowFallbackToPlaintext: true,
		TLSConfig:                "preferred",
		Params: map[string]string{
			"charset": "utf8mb4",
		},
	}
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	for _, stmt := range schemaStatements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := s.ensureSchemaPatchColumns(ctx); err != nil {
		return err
	}
	for _, stmt := range schemaPatchStatements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := s.ensureSeedConstraints(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureSchemaPatchColumns(ctx context.Context) error {
	columns := []struct {
		table      string
		column     string
		definition string
	}{
		{table: "paper_templates", column: "version", definition: "INT NOT NULL DEFAULT 1"},
		{table: "paper_templates", column: "parent_id", definition: "VARCHAR(40) DEFAULT ''"},
		{table: "paper_templates", column: "source_file_url", definition: "VARCHAR(255) DEFAULT ''"},
		{table: "question_templates", column: "scoring_rules_json", definition: "JSON NULL AFTER standard_answer"},
		{table: "classes", column: "campus_id", definition: "VARCHAR(40) DEFAULT '' AFTER school_id"},
		{table: "classes", column: "grade_id", definition: "VARCHAR(40) DEFAULT '' AFTER campus_id"},
		{table: "students", column: "user_id", definition: "VARCHAR(40) DEFAULT '' AFTER id"},
		{table: "students", column: "student_no", definition: "VARCHAR(40) DEFAULT '' AFTER class_id"},
		{table: "teachers", column: "user_id", definition: "VARCHAR(40) DEFAULT '' AFTER id"},
		{table: "teachers", column: "role", definition: "VARCHAR(40) DEFAULT 'subject_teacher' AFTER subject"},
		{table: "assignments", column: "exam_id", definition: "VARCHAR(40) DEFAULT '' AFTER id"},
		{table: "assignments", column: "kind", definition: "VARCHAR(40) NOT NULL DEFAULT 'exam' AFTER title"},
		{table: "assignments", column: "class_id", definition: "VARCHAR(40) DEFAULT '' AFTER kind"},
		{table: "assignments", column: "template_version", definition: "INT NOT NULL DEFAULT 1 AFTER template_id"},
		{table: "assignments", column: "teacher_id", definition: "VARCHAR(40) DEFAULT '' AFTER template_version"},
		{table: "assignments", column: "published_at", definition: "DATETIME NULL AFTER teacher_id"},
		{table: "assignments", column: "completed_at", definition: "DATETIME NULL AFTER due_at"},
		{table: "scan_jobs", column: "assignment_id", definition: "VARCHAR(40) DEFAULT '' AFTER id"},
		{table: "scan_jobs", column: "template_id", definition: "VARCHAR(40) DEFAULT '' AFTER class_name"},
		{table: "scan_jobs", column: "template_version", definition: "INT NOT NULL DEFAULT 1 AFTER template_id"},
		{table: "scan_jobs", column: "created_by", definition: "VARCHAR(40) DEFAULT '' AFTER template_version"},
		{table: "scan_jobs", column: "notes", definition: "TEXT AFTER pages"},
		{table: "scan_jobs", column: "files_json", definition: "JSON NULL AFTER notes"},
		{table: "scan_jobs", column: "failure_reason", definition: "TEXT AFTER progress"},
		{table: "scan_jobs", column: "retry_count", definition: "INT NOT NULL DEFAULT 0 AFTER failure_reason"},
		{table: "scan_jobs", column: "queue_status", definition: "VARCHAR(30) NOT NULL DEFAULT 'pending' AFTER retry_count"},
		{table: "scan_jobs", column: "queue_message", definition: "VARCHAR(255) DEFAULT '' AFTER queue_status"},
		{table: "submissions", column: "scan_task_id", definition: "VARCHAR(40) DEFAULT '' AFTER student_id"},
		{table: "submissions", column: "student_name", definition: "VARCHAR(80) DEFAULT '' AFTER scan_task_id"},
		{table: "submissions", column: "class_name", definition: "VARCHAR(100) DEFAULT '' AFTER student_name"},
		{table: "submissions", column: "page_count", definition: "INT NOT NULL DEFAULT 0 AFTER file_url"},
		{table: "submissions", column: "matched_status", definition: "VARCHAR(30) NOT NULL DEFAULT 'matched' AFTER page_count"},
		{table: "submissions", column: "graded_at", definition: "DATETIME NULL AFTER submitted_at"},
		{table: "subjective_reviews", column: "review_stage", definition: "VARCHAR(40) NOT NULL DEFAULT 'first_review' AFTER status"},
		{table: "subjective_reviews", column: "assignee_id", definition: "VARCHAR(40) DEFAULT '' AFTER review_stage"},
		{table: "subjective_reviews", column: "priority", definition: "INT NOT NULL DEFAULT 0 AFTER assignee_id"},
		{table: "grading_history", column: "actor_name", definition: "VARCHAR(80) DEFAULT '' AFTER actor_id"},
		{table: "grading_history", column: "review_stage", definition: "VARCHAR(40) DEFAULT 'first_review' AFTER actor_name"},
		{table: "grading_history", column: "model_version", definition: "VARCHAR(80) DEFAULT '' AFTER review_stage"},
		{table: "wrong_questions", column: "submission_id", definition: "VARCHAR(40) DEFAULT '' AFTER question_id"},
		{table: "wrong_questions", column: "question_no", definition: "VARCHAR(20) DEFAULT '' AFTER submission_id"},
		{table: "wrong_questions", column: "score", definition: "DECIMAL(6,2) NOT NULL DEFAULT 0 AFTER source_paper"},
		{table: "wrong_questions", column: "max_score", definition: "DECIMAL(6,2) NOT NULL DEFAULT 0 AFTER score"},
		{table: "wrong_questions", column: "correct_answer", definition: "TEXT AFTER max_score"},
		{table: "wrong_questions", column: "student_answer", definition: "TEXT AFTER correct_answer"},
		{table: "wrong_questions", column: "explanation", definition: "TEXT AFTER student_answer"},
		{table: "wrong_questions", column: "correction_status", definition: "VARCHAR(30) NOT NULL DEFAULT 'pending' AFTER explanation"},
		{table: "wrong_questions", column: "repractice_status", definition: "VARCHAR(30) NOT NULL DEFAULT 'not_assigned' AFTER correction_status"},
		{table: "wrong_questions", column: "corrected_at", definition: "DATETIME NULL AFTER repractice_status"},
		{table: "wrong_questions", column: "updated_at", definition: "TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at"},
	}
	for _, item := range columns {
		exists, err := s.columnExists(ctx, item.table, item.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", item.table, item.column, item.definition)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) columnExists(ctx context.Context, tableName string, columnName string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
			AND COLUMN_NAME = ?`, tableName, columnName).Scan(&count)
	return count > 0, err
}

func (s *Store) Seed(ctx context.Context) error {
	for _, stmt := range seedStatements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Dashboard(ctx context.Context) (DashboardResponse, error) {
	scanJobs, err := s.ScanJobs(ctx)
	if err != nil {
		return DashboardResponse{}, err
	}
	reviews, err := s.ReviewQueue(ctx)
	if err != nil {
		return DashboardResponse{}, err
	}
	weakPoints, err := s.WeakPoints(ctx)
	if err != nil {
		return DashboardResponse{}, err
	}
	homework, err := s.HomeworkWatch(ctx)
	if err != nil {
		return DashboardResponse{}, err
	}
	average, err := s.ClassAverageScore(ctx)
	if err != nil {
		return DashboardResponse{}, err
	}

	return DashboardResponse{
		Source: "database",
		Metrics: []Metric{
			{Label: "待批试卷", Value: fmt.Sprintf("%d", pendingPages(scanJobs)), Delta: "来自扫描队列", Tone: "primary"},
			{Label: "主观题待复核", Value: fmt.Sprintf("%d", len(reviews)), Delta: "AI 已预评分", Tone: "warning"},
			{Label: "未提交作业", Value: fmt.Sprintf("%d", len(homework)), Delta: "需提醒家长", Tone: "danger"},
			{Label: "班级平均分", Value: fmt.Sprintf("%.1f", average), Delta: "最近一次考试", Tone: "success"},
		},
		ScanQueue:     scanJobs,
		ReviewQueue:   reviews,
		WeakPoints:    weakPoints,
		HomeworkWatch: homework,
	}, nil
}

func (s *Store) ScanJobs(ctx context.Context) ([]ScanJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, class_name, template_id, template_version, pages, COALESCE(notes, ''), COALESCE(files_json, JSON_ARRAY()),
			status, progress, COALESCE(failure_reason, ''), retry_count, queue_status, COALESCE(queue_message, '')
		FROM scan_jobs
		ORDER BY created_at DESC
		LIMIT 30`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []ScanJob{}
	for rows.Next() {
		var job ScanJob
		var filesJSON string
		if err := rows.Scan(
			&job.ID, &job.Title, &job.ClassName, &job.TemplateID, &job.TemplateVersion, &job.Pages, &job.Notes, &filesJSON,
			&job.Status, &job.Progress, &job.FailureReason, &job.RetryCount, &job.QueueStatus, &job.QueueMessage,
		); err != nil {
			return nil, err
		}
		decodeScanFiles(filesJSON, &job.Files)
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) ScanTask(ctx context.Context, taskID string) (ScanJob, error) {
	var job ScanJob
	var filesJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, class_name, template_id, template_version, pages, COALESCE(notes, ''), COALESCE(files_json, JSON_ARRAY()),
			status, progress, COALESCE(failure_reason, ''), retry_count, queue_status, COALESCE(queue_message, '')
		FROM scan_jobs
		WHERE id = ?
		LIMIT 1`, taskID).Scan(
		&job.ID, &job.Title, &job.ClassName, &job.TemplateID, &job.TemplateVersion, &job.Pages, &job.Notes, &filesJSON,
		&job.Status, &job.Progress, &job.FailureReason, &job.RetryCount, &job.QueueStatus, &job.QueueMessage,
	)
	if err != nil {
		return ScanJob{}, err
	}
	decodeScanFiles(filesJSON, &job.Files)
	return job, nil
}

func (s *Store) CreateScanTask(ctx context.Context, req ScanTaskRequest) (ScanJob, error) {
	templateVersion := req.TemplateVersion
	if req.TemplateID != "" {
		template, err := s.Template(ctx, req.TemplateID)
		if err != nil {
			return ScanJob{}, err
		}
		if template.Status != "published" {
			return ScanJob{}, errTemplateNotPublished
		}
		templateVersion = template.Version
	}
	req.Files = s.matchScanFiles(ctx, req.ClassName, req.Files)
	filesJSON, err := json.Marshal(req.Files)
	if err != nil {
		return ScanJob{}, err
	}
	job := ScanJob{
		ID:              fmt.Sprintf("scan_%d", time.Now().UnixNano()),
		Title:           req.Title,
		ClassName:       req.ClassName,
		TemplateID:      req.TemplateID,
		TemplateVersion: templateVersion,
		Pages:           req.Pages,
		Notes:           req.Notes,
		Status:          "排队中",
		Progress:        0,
		QueueStatus:     "pending",
		Files:           req.Files,
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO scan_jobs
			(id, title, class_name, template_id, template_version, pages, notes, files_json, status, progress, failure_reason, retry_count, queue_status, queue_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', 0, ?, '')`,
		job.ID, job.Title, job.ClassName, job.TemplateID, job.TemplateVersion, job.Pages, job.Notes, string(filesJSON), job.Status, job.Progress, job.QueueStatus,
	)
	if err != nil {
		return ScanJob{}, err
	}
	return job, nil
}

func (s *Store) SaveObjectFiles(ctx context.Context, files []ScanFile, purpose string, ownerType string, ownerID string) error {
	if len(files) == 0 {
		return nil
	}
	for _, file := range files {
		metadata, err := json.Marshal(map[string]any{
			"fileName":      file.FileName,
			"page":          file.Page,
			"status":        file.Status,
			"matchStatus":   file.MatchStatus,
			"matchMethod":   file.MatchMethod,
			"studentId":     file.StudentID,
			"studentName":   file.StudentName,
			"failureReason": file.FailureReason,
		})
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO object_files
				(object_key, bucket, storage_driver, url, content_type, size_bytes, purpose, owner_type, owner_id, metadata_json)
			VALUES (?, 'local-dev', 'local', ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				url = VALUES(url),
				content_type = VALUES(content_type),
				size_bytes = VALUES(size_bytes),
				purpose = VALUES(purpose),
				owner_type = VALUES(owner_type),
				owner_id = VALUES(owner_id),
				metadata_json = VALUES(metadata_json)`,
			file.Key, file.URL, file.ContentType, file.Size, purpose, ownerType, ownerID, string(metadata),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateScanQueueStatus(ctx context.Context, taskID string, queueStatus string, queueMessage string) error {
	status := "排队中"
	failureReason := ""
	if queueStatus == "failed" {
		status = "队列投递失败"
		failureReason = queueMessage
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE scan_jobs
		SET queue_status = ?, queue_message = ?, status = ?, failure_reason = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		queueStatus, queueMessage, status, failureReason, taskID,
	)
	return err
}

func (s *Store) UpdateScanTaskStatus(ctx context.Context, taskID string, req ScanTaskStatusRequest) (ScanJob, error) {
	progress := req.Progress
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	retryCount := req.RetryCount
	if retryCount < 0 {
		retryCount = 0
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE scan_jobs
		SET status = ?, progress = ?, failure_reason = ?, retry_count = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		req.Status, progress, req.FailureReason, retryCount, taskID,
	)
	if err != nil {
		return ScanJob{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ScanJob{}, err
	}
	if affected == 0 {
		return ScanJob{}, sql.ErrNoRows
	}
	return s.ScanTask(ctx, taskID)
}

func (s *Store) SaveScanWorkerResult(ctx context.Context, taskID string, req ScanWorkerResultRequest) (ScanJob, error) {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "识别完成"
	}
	progress := req.Progress
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	if req.Result == nil {
		req.Result = map[string]any{}
	}
	raw, err := json.Marshal(req.Result)
	if err != nil {
		return ScanJob{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ScanJob{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO worker_task_results (task_id, status, progress, failure_reason, model_version, result_json)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status),
			progress = VALUES(progress),
			failure_reason = VALUES(failure_reason),
			model_version = VALUES(model_version),
			result_json = VALUES(result_json),
			updated_at = CURRENT_TIMESTAMP`,
		taskID, status, progress, req.FailureReason, req.ModelVersion, string(raw),
	); err != nil {
		return ScanJob{}, err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE scan_jobs
		SET status = ?, progress = ?, failure_reason = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		status, progress, req.FailureReason, taskID,
	)
	if err != nil {
		return ScanJob{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ScanJob{}, err
	}
	if affected == 0 {
		return ScanJob{}, sql.ErrNoRows
	}
	if err := tx.Commit(); err != nil {
		return ScanJob{}, err
	}
	return s.ScanTask(ctx, taskID)
}

func (s *Store) ScanWorkerResult(ctx context.Context, taskID string) (ScanWorkerResultRecord, error) {
	var record ScanWorkerResultRecord
	var raw string
	err := s.db.QueryRowContext(ctx, `
		SELECT task_id, status, progress, COALESCE(failure_reason, ''), COALESCE(model_version, ''), result_json
		FROM worker_task_results
		WHERE task_id = ?
		LIMIT 1`, taskID).Scan(&record.TaskID, &record.Status, &record.Progress, &record.FailureReason, &record.ModelVersion, &raw)
	if err != nil {
		return ScanWorkerResultRecord{}, err
	}
	if err := json.Unmarshal([]byte(raw), &record.Result); err != nil {
		return ScanWorkerResultRecord{}, err
	}
	return record, nil
}

func (s *Store) RetryScanTask(ctx context.Context, taskID string, fileKey string) (ScanJob, error) {
	task, err := s.ScanTask(ctx, taskID)
	if err != nil {
		return ScanJob{}, err
	}
	files := task.Files
	for index := range files {
		if fileKey == "" || files[index].Key == fileKey {
			files[index].Status = "待重试"
			files[index].FailureReason = ""
		}
	}
	filesJSON, err := json.Marshal(files)
	if err != nil {
		return ScanJob{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE scan_jobs
		SET status = '排队中',
			progress = 0,
			failure_reason = '',
			retry_count = retry_count + 1,
			queue_status = 'pending',
			queue_message = '',
			files_json = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		string(filesJSON), taskID,
	)
	if err != nil {
		return ScanJob{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ScanJob{}, err
	}
	if affected == 0 {
		return ScanJob{}, sql.ErrNoRows
	}
	message := "整批任务重试"
	if fileKey != "" {
		message = "单文件重试"
	}
	if err := s.insertScanTaskLog(ctx, taskID, fileKey, "retry", message); err != nil {
		log.Printf("scan retry log insert failed: %v", err)
	}
	return s.ScanTask(ctx, taskID)
}

func (s *Store) MatchScanFile(ctx context.Context, taskID string, req ScanFileMatchRequest) (ScanJob, error) {
	req.FileKey = strings.TrimSpace(req.FileKey)
	req.StudentID = strings.TrimSpace(req.StudentID)
	req.StudentName = strings.TrimSpace(req.StudentName)
	req.MatchMethod = strings.TrimSpace(req.MatchMethod)
	if req.MatchMethod == "" {
		req.MatchMethod = "manual"
	}
	task, err := s.ScanTask(ctx, taskID)
	if err != nil {
		return ScanJob{}, err
	}
	found := false
	for index := range task.Files {
		if task.Files[index].Key == req.FileKey {
			task.Files[index].StudentID = req.StudentID
			task.Files[index].StudentName = req.StudentName
			task.Files[index].MatchMethod = req.MatchMethod
			if req.StudentID == "" && req.StudentName == "" {
				task.Files[index].MatchStatus = "pending"
			} else {
				task.Files[index].MatchStatus = "matched"
			}
			found = true
			break
		}
	}
	if !found {
		return ScanJob{}, sql.ErrNoRows
	}
	filesJSON, err := json.Marshal(task.Files)
	if err != nil {
		return ScanJob{}, err
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE scan_jobs SET files_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", string(filesJSON), taskID); err != nil {
		return ScanJob{}, err
	}
	if err := s.insertScanTaskLog(ctx, taskID, req.FileKey, "match", req.StudentName); err != nil {
		log.Printf("scan match log insert failed: %v", err)
	}
	return s.ScanTask(ctx, taskID)
}

func (s *Store) insertScanTaskLog(ctx context.Context, taskID string, fileKey string, action string, message string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO scan_task_logs (task_id, file_key, action, message) VALUES (?, ?, ?, ?)", taskID, fileKey, action, message)
	return err
}

func (s *Store) matchScanFiles(ctx context.Context, className string, files []ScanFile) []ScanFile {
	type student struct {
		id   string
		name string
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT students.id, students.name
		FROM students
		JOIN classes ON classes.id = students.class_id
		WHERE classes.name = ?`, className)
	if err != nil {
		log.Printf("scan student match query failed: %v", err)
		return markScanFilesPending(files)
	}
	defer rows.Close()
	students := []student{}
	for rows.Next() {
		var item student
		if err := rows.Scan(&item.id, &item.name); err != nil {
			log.Printf("scan student match scan failed: %v", err)
			continue
		}
		students = append(students, item)
	}
	for index := range files {
		files[index].Page = index + 1
		if files[index].Status == "" {
			files[index].Status = "uploaded"
		}
		files[index].MatchStatus = "pending"
		name := strings.ToLower(files[index].FileName)
		for _, student := range students {
			if strings.Contains(name, strings.ToLower(student.name)) {
				files[index].StudentID = student.id
				files[index].StudentName = student.name
				files[index].MatchStatus = "matched"
				files[index].MatchMethod = "name"
				break
			}
		}
	}
	return files
}

func markScanFilesPending(files []ScanFile) []ScanFile {
	for index := range files {
		files[index].Page = index + 1
		if files[index].Status == "" {
			files[index].Status = "uploaded"
		}
		if files[index].MatchStatus == "" {
			files[index].MatchStatus = "pending"
		}
	}
	return files
}

func (s *Store) ReviewQueue(ctx context.Context) ([]ReviewItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, student_name, class_name, paper_name, question_no, ai_score, full_score, confidence, status, review_stage
		FROM subjective_reviews
		WHERE status IN ('pending', 'second_review', 'arbitration')
		ORDER BY priority DESC, updated_at DESC
		LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []ReviewItem{}
	for rows.Next() {
		var item ReviewItem
		var aiScore, fullScore float64
		if err := rows.Scan(&item.ID, &item.StudentName, &item.ClassName, &item.PaperName, &item.QuestionNo, &aiScore, &fullScore, &item.Confidence, &item.Status, &item.ReviewStage); err != nil {
			return nil, err
		}
		item.AIAdvice = fmt.Sprintf("%.0f / %.0f", aiScore, fullScore)
		reviews = append(reviews, item)
	}
	return reviews, rows.Err()
}

func (s *Store) GradingHistory(ctx context.Context, submissionID string, questionID string) ([]GradingHistoryItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, submission_id, question_id, action, score, COALESCE(note, ''), COALESCE(actor_name, ''), COALESCE(review_stage, ''), COALESCE(model_version, ''), created_at
		FROM grading_history
		WHERE submission_id = ? AND question_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 20`, submissionID, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []GradingHistoryItem{}
	for rows.Next() {
		var item GradingHistoryItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.SubmissionID, &item.QuestionID, &item.Action, &item.Score, &item.Note, &item.ActorName, &item.ReviewStage, &item.ModelVersion, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) WeakPoints(ctx context.Context) ([]KnowledgeStat, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, accuracy, wrong_count
		FROM knowledge_stats
		ORDER BY accuracy ASC, wrong_count DESC
		LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := []KnowledgeStat{}
	for rows.Next() {
		var item KnowledgeStat
		if err := rows.Scan(&item.Name, &item.Accuracy, &item.WrongCount); err != nil {
			return nil, err
		}
		stats = append(stats, item)
	}
	return stats, rows.Err()
}

func (s *Store) HomeworkWatch(ctx context.Context) ([]HomeworkWatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT student_name, class_name, missing, guardian
		FROM homework_watch
		WHERE missing > 0
		ORDER BY missing DESC
		LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []HomeworkWatch{}
	for rows.Next() {
		var item HomeworkWatch
		if err := rows.Scan(&item.StudentName, &item.ClassName, &item.Missing, &item.Guardian); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ClassAverageScore(ctx context.Context) (float64, error) {
	var average sql.NullFloat64
	err := s.db.QueryRowContext(ctx, "SELECT AVG(score) FROM exam_scores WHERE class_name = '六年级 3 班'").Scan(&average)
	if err != nil {
		return 0, err
	}
	if !average.Valid {
		return 0, nil
	}
	return average.Float64, nil
}

func (s *Store) CurrentSubjective(ctx context.Context) (SubjectiveGradingResponse, error) {
	return s.subjectiveByQuery(ctx, `
		SELECT id, submission_id, question_id, paper_name, student_name, class_name, question_no,
			full_score, standard_answer, scoring_rules_json, knowledge_json, student_ocr_text,
			student_image_url, ai_score, ai_reason, ai_comments_json, confidence
		FROM subjective_reviews
		WHERE status IN ('pending', 'second_review', 'arbitration')
		ORDER BY priority DESC, updated_at DESC
		LIMIT 1`)
}

func (s *Store) SubjectiveByReviewID(ctx context.Context, reviewID string) (SubjectiveGradingResponse, error) {
	return s.subjectiveByQuery(ctx, `
		SELECT id, submission_id, question_id, paper_name, student_name, class_name, question_no,
			full_score, standard_answer, scoring_rules_json, knowledge_json, student_ocr_text,
			student_image_url, ai_score, ai_reason, ai_comments_json, confidence
		FROM subjective_reviews
		WHERE id = ?
		LIMIT 1`, reviewID)
}

func (s *Store) subjectiveByQuery(ctx context.Context, query string, args ...any) (SubjectiveGradingResponse, error) {
	var row struct {
		ReviewID      string
		SubmissionID  string
		QuestionID    string
		PaperName     string
		StudentName   string
		ClassName     string
		QuestionNo    string
		FullScore     float64
		Standard      string
		RulesJSON     string
		KnowledgeJSON string
		OCRText       string
		ImageURL      string
		AIScore       float64
		AIReason      string
		AIComments    string
		Confidence    int
	}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&row.ReviewID, &row.SubmissionID, &row.QuestionID, &row.PaperName, &row.StudentName, &row.ClassName, &row.QuestionNo,
		&row.FullScore, &row.Standard, &row.RulesJSON, &row.KnowledgeJSON, &row.OCRText,
		&row.ImageURL, &row.AIScore, &row.AIReason, &row.AIComments, &row.Confidence,
	)
	if err != nil {
		return SubjectiveGradingResponse{}, err
	}

	var rules, knowledge, comments []string
	decodeStringSlice(row.RulesJSON, &rules)
	decodeStringSlice(row.KnowledgeJSON, &knowledge)
	decodeStringSlice(row.AIComments, &comments)

	return SubjectiveGradingResponse{
		ReviewID:     row.ReviewID,
		SubmissionID: row.SubmissionID,
		QuestionID:   row.QuestionID,
		PaperName:    row.PaperName,
		StudentName:  row.StudentName,
		ClassName:    row.ClassName,
		QuestionNo:   row.QuestionNo,
		FullScore:    row.FullScore,
		StandardAnswer: StandardAnswer{
			Content:      row.Standard,
			ScoringRules: rules,
			Knowledge:    knowledge,
		},
		StudentAnswer: StudentAnswer{
			OCRText:  row.OCRText,
			ImageURL: row.ImageURL,
		},
		AI: AIAdvice{
			Score:      row.AIScore,
			Reason:     row.AIReason,
			Comments:   comments,
			Confidence: row.Confidence,
		},
	}, nil
}

func (s *Store) SaveSubjectiveDecision(ctx context.Context, req GradingDecisionRequest) (GradingDecisionResponse, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return GradingDecisionResponse{}, err
	}
	defer tx.Rollback()

	var review struct {
		ID             string
		PaperName      string
		StudentName    string
		ClassName      string
		StudentID      string
		QuestionNo     string
		FullScore      float64
		StandardAnswer string
		KnowledgeJSON  string
		OCRText        string
		ImageURL       string
	}
	err = tx.QueryRowContext(ctx, `
		SELECT sr.id, sr.paper_name, sr.student_name, sr.class_name, COALESCE(sub.student_id, ''),
			sr.question_no, sr.full_score, sr.standard_answer, sr.knowledge_json, sr.student_ocr_text, sr.student_image_url
		FROM subjective_reviews sr
		LEFT JOIN submissions sub ON sub.id = sr.submission_id
		WHERE sr.submission_id = ? AND sr.question_id = ?
		LIMIT 1`, req.SubmissionID, req.QuestionID).Scan(
		&review.ID, &review.PaperName, &review.StudentName, &review.ClassName, &review.StudentID,
		&review.QuestionNo, &review.FullScore, &review.StandardAnswer, &review.KnowledgeJSON, &review.OCRText, &review.ImageURL,
	)
	if err != nil {
		return GradingDecisionResponse{}, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO grading_decisions (submission_id, question_id, final_score, decision, teacher_note)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE final_score = VALUES(final_score), decision = VALUES(decision), teacher_note = VALUES(teacher_note), updated_at = CURRENT_TIMESTAMP`,
		req.SubmissionID, req.QuestionID, req.FinalScore, req.Decision, req.TeacherNote,
	)
	if err != nil {
		return GradingDecisionResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO question_scores (submission_id, question_id, question_no, score, max_score, source, status)
		VALUES (?, ?, ?, ?, ?, 'teacher', 'final')
		ON DUPLICATE KEY UPDATE question_no = VALUES(question_no), score = VALUES(score), max_score = VALUES(max_score),
			source = VALUES(source), status = VALUES(status), updated_at = CURRENT_TIMESTAMP`,
		req.SubmissionID, req.QuestionID, review.QuestionNo, req.FinalScore, review.FullScore,
	); err != nil {
		return GradingDecisionResponse{}, err
	}
	actorName := strings.TrimSpace(req.ActorName)
	if actorName == "" {
		actorName = "当前教师"
	}
	reviewStage := strings.TrimSpace(req.ReviewStage)
	if reviewStage == "" {
		reviewStage = "first_review"
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO grading_history (submission_id, question_id, action, score, note, actor_id, actor_name, review_stage, model_version)
		VALUES (?, ?, ?, ?, ?, '', ?, ?, ?)`,
		req.SubmissionID, req.QuestionID, req.Decision, req.FinalScore, req.TeacherNote, actorName, reviewStage, req.ModelVersion,
	); err != nil {
		return GradingDecisionResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO student_answers (submission_id, question_id, question_no, answer_text, image_url)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE question_no = VALUES(question_no), answer_text = VALUES(answer_text), image_url = VALUES(image_url), updated_at = CURRENT_TIMESTAMP`,
		req.SubmissionID, req.QuestionID, review.QuestionNo, review.OCRText, review.ImageURL,
	); err != nil {
		return GradingDecisionResponse{}, err
	}
	if req.FinalScore < review.FullScore {
		knowledgePoint := firstKnowledgePoint(review.KnowledgeJSON)
		result, err := tx.ExecContext(ctx, `
			UPDATE wrong_questions
			SET student_id = ?,
				question_no = ?,
				knowledge_point = ?,
				wrong_reason = ?,
				source_paper = ?,
				score = ?,
				max_score = ?,
				correct_answer = ?,
				student_answer = ?,
				explanation = ?,
				correction_status = 'pending',
				repractice_status = 'not_assigned',
				updated_at = CURRENT_TIMESTAMP
			WHERE submission_id = ? AND question_id = ?`,
			review.StudentID, review.QuestionNo, knowledgePoint, wrongReason(req, review.FullScore), review.PaperName,
			req.FinalScore, review.FullScore, review.StandardAnswer, review.OCRText, req.TeacherNote,
			req.SubmissionID, req.QuestionID,
		)
		if err != nil {
			return GradingDecisionResponse{}, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return GradingDecisionResponse{}, err
		}
		if affected == 0 {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO wrong_questions
					(student_id, question_id, submission_id, question_no, knowledge_point, wrong_reason, source_paper, score, max_score, correct_answer, student_answer, explanation, correction_status, repractice_status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', 'not_assigned')`,
				review.StudentID, req.QuestionID, req.SubmissionID, review.QuestionNo, knowledgePoint, wrongReason(req, review.FullScore), review.PaperName,
				req.FinalScore, review.FullScore, review.StandardAnswer, review.OCRText, req.TeacherNote,
			); err != nil {
				return GradingDecisionResponse{}, err
			}
		}
	}
	nextStatus := "reviewed"
	if req.Decision == "second_review" {
		nextStatus = "second_review"
		reviewStage = "second_review"
	}
	if req.Decision == "arbitration" {
		nextStatus = "arbitration"
		reviewStage = "arbitration"
	}
	if _, err := tx.ExecContext(ctx, "UPDATE subjective_reviews SET status = ?, review_stage = ?, updated_at = CURRENT_TIMESTAMP WHERE submission_id = ? AND question_id = ?", nextStatus, reviewStage, req.SubmissionID, req.QuestionID); err != nil {
		return GradingDecisionResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE submissions SET status = 'graded', graded_at = CURRENT_TIMESTAMP WHERE id = ?", req.SubmissionID); err != nil {
		return GradingDecisionResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return GradingDecisionResponse{}, err
	}

	nextQuestion := ""
	nextReview, err := s.CurrentSubjective(ctx)
	if err == nil {
		nextQuestion = nextReview.QuestionID
	} else if err != sql.ErrNoRows {
		return GradingDecisionResponse{}, err
	}

	response := GradingDecisionResponse{
		Status:       "saved",
		FinalScore:   req.FinalScore,
		NextQuestion: nextQuestion,
	}
	if err == nil {
		response.NextReview = &nextReview
	}
	return response, nil
}

func firstKnowledgePoint(raw string) string {
	var knowledge []string
	decodeStringSlice(raw, &knowledge)
	if len(knowledge) == 0 || strings.TrimSpace(knowledge[0]) == "" {
		return "未归类"
	}
	return strings.TrimSpace(knowledge[0])
}

func wrongReason(req GradingDecisionRequest, fullScore float64) string {
	if strings.TrimSpace(req.TeacherNote) != "" {
		return strings.TrimSpace(req.TeacherNote)
	}
	if req.FinalScore <= 0 {
		return "未得分，需订正"
	}
	return fmt.Sprintf("得分 %.1f/%.1f，需订正", req.FinalScore, fullScore)
}

func (s *Store) ResetDemo(ctx context.Context) (DashboardResponse, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return DashboardResponse{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM grading_decisions WHERE submission_id IN ('sub_001', 'sub_002')"); err != nil {
		return DashboardResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE subjective_reviews SET status = 'pending', updated_at = CURRENT_TIMESTAMP WHERE id IN ('review_001', 'review_002')"); err != nil {
		return DashboardResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return DashboardResponse{}, err
	}
	return s.Dashboard(ctx)
}

func (s *Store) Templates(ctx context.Context) ([]PaperTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, subject, grade, question_count, total_score, source_file_url, status, version, parent_id
		FROM paper_templates
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := []PaperTemplate{}
	for rows.Next() {
		var item PaperTemplate
		if err := rows.Scan(&item.ID, &item.Name, &item.Subject, &item.Grade, &item.QuestionCount, &item.TotalScore, &item.SourceFileURL, &item.Status, &item.Version, &item.ParentID); err != nil {
			return nil, err
		}
		questions, err := s.TemplateQuestions(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		item.Questions = questions
		templates = append(templates, item)
	}
	return templates, rows.Err()
}

func (s *Store) Template(ctx context.Context, templateID string) (PaperTemplate, error) {
	var item PaperTemplate
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, subject, grade, question_count, total_score, source_file_url, status, version, parent_id
		FROM paper_templates
		WHERE id = ?
		LIMIT 1`, templateID).Scan(&item.ID, &item.Name, &item.Subject, &item.Grade, &item.QuestionCount, &item.TotalScore, &item.SourceFileURL, &item.Status, &item.Version, &item.ParentID)
	if err != nil {
		return PaperTemplate{}, err
	}
	questions, err := s.TemplateQuestions(ctx, item.ID)
	if err != nil {
		return PaperTemplate{}, err
	}
	item.Questions = questions
	return item, nil
}

func (s *Store) CreateTemplate(ctx context.Context, template PaperTemplate) (PaperTemplate, error) {
	template.ID = templateRecordID("tpl", template.ID)
	if template.Status == "" {
		template.Status = "draft"
	}
	if template.Version == 0 {
		template.Version = 1
	}
	return s.saveTemplate(ctx, template, false)
}

func (s *Store) UpdateTemplate(ctx context.Context, templateID string, template PaperTemplate) (PaperTemplate, error) {
	template.ID = templateID
	return s.saveTemplate(ctx, template, true)
}

func (s *Store) DeleteTemplate(ctx context.Context, templateID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM question_templates WHERE template_id = ?", templateID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM paper_templates WHERE id = ?", templateID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (s *Store) CopyTemplate(ctx context.Context, templateID string) (PaperTemplate, error) {
	source, err := s.Template(ctx, templateID)
	if err != nil {
		return PaperTemplate{}, err
	}
	parentID := source.ParentID
	if parentID == "" {
		parentID = source.ID
	}
	source.ID = templateRecordID("tpl", "")
	source.Name = fmt.Sprintf("%s v%d", source.Name, source.Version+1)
	source.Status = "draft"
	source.Version++
	source.ParentID = parentID
	for index := range source.Questions {
		source.Questions[index].ID = templateRecordID("q", "")
	}
	return s.saveTemplate(ctx, source, false)
}

func (s *Store) UpdateTemplateStatus(ctx context.Context, templateID string, status string) (PaperTemplate, error) {
	if !validTemplateStatus(status) {
		return PaperTemplate{}, fmt.Errorf("invalid template status: %s", status)
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE paper_templates
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, status, templateID)
	if err != nil {
		return PaperTemplate{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PaperTemplate{}, err
	}
	if affected == 0 {
		return PaperTemplate{}, sql.ErrNoRows
	}
	return s.Template(ctx, templateID)
}

func (s *Store) saveTemplate(ctx context.Context, template PaperTemplate, requireExisting bool) (PaperTemplate, error) {
	if template.Subject == "" {
		template.Subject = "数学"
	}
	if template.Grade == "" {
		template.Grade = "六年级"
	}
	if template.Status == "" {
		template.Status = "draft"
	}
	if template.Version == 0 {
		template.Version = 1
	}
	template.QuestionCount = len(template.Questions)
	totalScore := 0.0
	for _, question := range template.Questions {
		totalScore += question.Score
	}
	template.TotalScore = int(totalScore + 0.5)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PaperTemplate{}, err
	}
	defer tx.Rollback()

	if requireExisting {
		if err := ensureTemplateDraftTx(ctx, tx, template.ID); err != nil {
			return PaperTemplate{}, err
		}
		result, err := tx.ExecContext(ctx, `
				UPDATE paper_templates
				SET name = ?, subject = ?, grade = ?, question_count = ?, total_score = ?, source_file_url = ?, status = ?, version = ?, parent_id = ?, updated_at = CURRENT_TIMESTAMP
				WHERE id = ?`,
			template.Name, template.Subject, template.Grade, template.QuestionCount, template.TotalScore, template.SourceFileURL, template.Status, template.Version, template.ParentID, template.ID,
		)
		if err != nil {
			return PaperTemplate{}, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return PaperTemplate{}, err
		}
		if affected == 0 {
			return PaperTemplate{}, sql.ErrNoRows
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
				INSERT INTO paper_templates (id, name, subject, grade, question_count, total_score, source_file_url, status, version, parent_id)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			template.ID, template.Name, template.Subject, template.Grade, template.QuestionCount, template.TotalScore, template.SourceFileURL, template.Status, template.Version, template.ParentID,
		); err != nil {
			return PaperTemplate{}, err
		}
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM question_templates WHERE template_id = ?", template.ID); err != nil {
		return PaperTemplate{}, err
	}
	for index, question := range template.Questions {
		question.ID = templateRecordID("q", question.ID)
		if question.No == "" {
			question.No = fmt.Sprintf("%d", index+1)
		}
		knowledgeJSON, err := json.Marshal(question.Knowledge)
		if err != nil {
			return PaperTemplate{}, err
		}
		if _, err := tx.ExecContext(ctx, `
				INSERT INTO question_templates
					(id, template_id, question_no, question_type, score, standard_answer, scoring_rules_json, knowledge_json, page_no, x, y, width, height)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			question.ID, template.ID, question.No, question.Type, question.Score, question.StandardAnswer, scoringRulesJSON(question.ScoringRules), string(knowledgeJSON),
			question.Region.Page, question.Region.X, question.Region.Y, question.Region.Width, question.Region.Height,
		); err != nil {
			return PaperTemplate{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return PaperTemplate{}, err
	}
	return s.Template(ctx, template.ID)
}

func (s *Store) TemplateQuestions(ctx context.Context, templateID string) ([]QuestionTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, question_no, question_type, score, COALESCE(standard_answer, ''), COALESCE(scoring_rules_json, JSON_ARRAY()), knowledge_json, page_no, x, y, width, height
		FROM question_templates
		WHERE template_id = ?
		ORDER BY page_no ASC, question_no + 0 ASC`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	questions := []QuestionTemplate{}
	for rows.Next() {
		var item QuestionTemplate
		var knowledgeJSON string
		var scoringRulesJSON string
		if err := rows.Scan(&item.ID, &item.No, &item.Type, &item.Score, &item.StandardAnswer, &scoringRulesJSON, &knowledgeJSON, &item.Region.Page, &item.Region.X, &item.Region.Y, &item.Region.Width, &item.Region.Height); err != nil {
			return nil, err
		}
		decodeStringSlice(scoringRulesJSON, &item.ScoringRules)
		decodeStringSlice(knowledgeJSON, &item.Knowledge)
		questions = append(questions, item)
	}
	return questions, rows.Err()
}

func (s *Store) TemplateQuestion(ctx context.Context, templateID string, questionID string) (QuestionTemplate, error) {
	var item QuestionTemplate
	var knowledgeJSON string
	var scoringRulesJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, question_no, question_type, score, COALESCE(standard_answer, ''), COALESCE(scoring_rules_json, JSON_ARRAY()), knowledge_json, page_no, x, y, width, height
		FROM question_templates
		WHERE template_id = ? AND id = ?
	LIMIT 1`, templateID, questionID).Scan(
		&item.ID, &item.No, &item.Type, &item.Score, &item.StandardAnswer, &scoringRulesJSON, &knowledgeJSON,
		&item.Region.Page, &item.Region.X, &item.Region.Y, &item.Region.Width, &item.Region.Height,
	)
	if err != nil {
		return QuestionTemplate{}, err
	}
	decodeStringSlice(scoringRulesJSON, &item.ScoringRules)
	decodeStringSlice(knowledgeJSON, &item.Knowledge)
	return item, nil
}

func (s *Store) CreateTemplateQuestion(ctx context.Context, templateID string, question QuestionTemplate) (QuestionTemplate, PaperTemplate, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	defer tx.Rollback()

	if err := templateExistsTx(ctx, tx, templateID); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	if err := ensureTemplateDraftTx(ctx, tx, templateID); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	question.ID = templateRecordID("q", question.ID)
	if question.No == "" {
		question.No = nextTemplateQuestionNo(ctx, tx, templateID)
	}
	if question.Region.Page == 0 {
		question.Region.Page = 1
	}
	if err := insertTemplateQuestionTx(ctx, tx, templateID, question); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	if err := refreshTemplateStatsTx(ctx, tx, templateID); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	if err := tx.Commit(); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	savedQuestion, err := s.TemplateQuestion(ctx, templateID, question.ID)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	template, err := s.Template(ctx, templateID)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	return savedQuestion, template, nil
}

func (s *Store) UpdateTemplateQuestion(ctx context.Context, templateID string, questionID string, question QuestionTemplate) (QuestionTemplate, PaperTemplate, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	defer tx.Rollback()

	if err := templateExistsTx(ctx, tx, templateID); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	if err := ensureTemplateDraftTx(ctx, tx, templateID); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	current, err := s.TemplateQuestion(ctx, templateID, questionID)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	question.ID = questionID
	if question.No == "" {
		question.No = current.No
	}
	if question.Type == "" {
		question.Type = current.Type
	}
	if question.Score == 0 {
		question.Score = current.Score
	}
	if question.StandardAnswer == "" {
		question.StandardAnswer = current.StandardAnswer
	}
	if len(question.ScoringRules) == 0 {
		question.ScoringRules = current.ScoringRules
	}
	if len(question.Knowledge) == 0 {
		question.Knowledge = current.Knowledge
	}
	if question.Region.Page == 0 {
		question.Region = current.Region
	}
	knowledgeJSON, err := json.Marshal(question.Knowledge)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE question_templates
		SET question_no = ?, question_type = ?, score = ?, standard_answer = ?, scoring_rules_json = ?, knowledge_json = ?, page_no = ?, x = ?, y = ?, width = ?, height = ?, updated_at = CURRENT_TIMESTAMP
		WHERE template_id = ? AND id = ?`,
		question.No, question.Type, question.Score, question.StandardAnswer, scoringRulesJSON(question.ScoringRules), string(knowledgeJSON),
		question.Region.Page, question.Region.X, question.Region.Y, question.Region.Width, question.Region.Height,
		templateID, questionID,
	)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	if affected == 0 {
		return QuestionTemplate{}, PaperTemplate{}, sql.ErrNoRows
	}
	if err := refreshTemplateStatsTx(ctx, tx, templateID); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	if err := tx.Commit(); err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	savedQuestion, err := s.TemplateQuestion(ctx, templateID, questionID)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	template, err := s.Template(ctx, templateID)
	if err != nil {
		return QuestionTemplate{}, PaperTemplate{}, err
	}
	return savedQuestion, template, nil
}

func (s *Store) DeleteTemplateQuestion(ctx context.Context, templateID string, questionID string) (PaperTemplate, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PaperTemplate{}, err
	}
	defer tx.Rollback()

	if err := templateExistsTx(ctx, tx, templateID); err != nil {
		return PaperTemplate{}, err
	}
	if err := ensureTemplateDraftTx(ctx, tx, templateID); err != nil {
		return PaperTemplate{}, err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM question_templates WHERE template_id = ? AND id = ?", templateID, questionID)
	if err != nil {
		return PaperTemplate{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PaperTemplate{}, err
	}
	if affected == 0 {
		return PaperTemplate{}, sql.ErrNoRows
	}
	if err := refreshTemplateStatsTx(ctx, tx, templateID); err != nil {
		return PaperTemplate{}, err
	}
	if err := tx.Commit(); err != nil {
		return PaperTemplate{}, err
	}
	return s.Template(ctx, templateID)
}

func (s *Store) SaveTemplateQuestions(ctx context.Context, templateID string, questions []QuestionTemplate) (PaperTemplate, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PaperTemplate{}, err
	}
	defer tx.Rollback()

	if err := templateExistsTx(ctx, tx, templateID); err != nil {
		return PaperTemplate{}, err
	}
	if err := ensureTemplateDraftTx(ctx, tx, templateID); err != nil {
		return PaperTemplate{}, err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM question_templates WHERE template_id = ?", templateID); err != nil {
		return PaperTemplate{}, err
	}
	for index, question := range questions {
		question.ID = templateRecordID("q", question.ID)
		if question.No == "" {
			question.No = fmt.Sprintf("%d", index+1)
		}
		if question.Region.Page == 0 {
			question.Region.Page = 1
		}
		if err := insertTemplateQuestionTx(ctx, tx, templateID, question); err != nil {
			return PaperTemplate{}, err
		}
	}
	if err := refreshTemplateStatsTx(ctx, tx, templateID); err != nil {
		return PaperTemplate{}, err
	}
	if err := tx.Commit(); err != nil {
		return PaperTemplate{}, err
	}
	return s.Template(ctx, templateID)
}

func (s *Store) ClassroomAnalytics(ctx context.Context) (ClassroomAnalytics, error) {
	analytics := ClassroomAnalytics{
		ClassName:      "六年级 3 班",
		QuestionStats:  []QuestionStat{},
		KnowledgeStats: []KnowledgeStat{},
		StudentRisks:   []StudentRisk{},
	}
	var averageScore, highestScore, lowestScore sql.NullFloat64
	if err := s.db.QueryRowContext(ctx, "SELECT AVG(score), MAX(score), MIN(score) FROM exam_scores WHERE class_name = ?", analytics.ClassName).Scan(&averageScore, &highestScore, &lowestScore); err != nil {
		return ClassroomAnalytics{}, err
	}
	if averageScore.Valid {
		analytics.AverageScore = averageScore.Float64
	}
	if highestScore.Valid {
		analytics.HighestScore = highestScore.Float64
	}
	if lowestScore.Valid {
		analytics.LowestScore = lowestScore.Float64
	}

	questionRows, err := s.db.QueryContext(ctx, "SELECT question_no, accuracy, question_type FROM question_stats ORDER BY accuracy ASC")
	if err != nil {
		return ClassroomAnalytics{}, err
	}
	defer questionRows.Close()
	for questionRows.Next() {
		var item QuestionStat
		if err := questionRows.Scan(&item.No, &item.Accuracy, &item.Type); err != nil {
			return ClassroomAnalytics{}, err
		}
		analytics.QuestionStats = append(analytics.QuestionStats, item)
	}

	knowledge, err := s.WeakPoints(ctx)
	if err != nil {
		return ClassroomAnalytics{}, err
	}
	analytics.KnowledgeStats = knowledge

	riskRows, err := s.db.QueryContext(ctx, "SELECT student_name, risk, weakness_json FROM student_risks ORDER BY updated_at DESC")
	if err != nil {
		return ClassroomAnalytics{}, err
	}
	defer riskRows.Close()
	for riskRows.Next() {
		var item StudentRisk
		var weaknessJSON string
		if err := riskRows.Scan(&item.StudentName, &item.Risk, &weaknessJSON); err != nil {
			return ClassroomAnalytics{}, err
		}
		decodeStringSlice(weaknessJSON, &item.Weakness)
		analytics.StudentRisks = append(analytics.StudentRisks, item)
	}

	return analytics, nil
}

func insertTemplateQuestionTx(ctx context.Context, tx *sql.Tx, templateID string, question QuestionTemplate) error {
	knowledgeJSON, err := json.Marshal(question.Knowledge)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO question_templates
			(id, template_id, question_no, question_type, score, standard_answer, scoring_rules_json, knowledge_json, page_no, x, y, width, height)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		question.ID, templateID, question.No, question.Type, question.Score, question.StandardAnswer, scoringRulesJSON(question.ScoringRules), string(knowledgeJSON),
		question.Region.Page, question.Region.X, question.Region.Y, question.Region.Width, question.Region.Height,
	)
	return err
}

func scoringRulesJSON(rules []string) string {
	raw, err := json.Marshal(rules)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func validTemplateStatus(status string) bool {
	return status == "draft" || status == "published" || status == "disabled"
}

func templateExistsTx(ctx context.Context, tx *sql.Tx, templateID string) error {
	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT 1 FROM paper_templates WHERE id = ? LIMIT 1", templateID).Scan(&exists); err != nil {
		return err
	}
	return nil
}

func ensureTemplateDraftTx(ctx context.Context, tx *sql.Tx, templateID string) error {
	var status string
	if err := tx.QueryRowContext(ctx, "SELECT status FROM paper_templates WHERE id = ? LIMIT 1", templateID).Scan(&status); err != nil {
		return err
	}
	if status != "draft" {
		return errTemplateLocked
	}
	return nil
}

func nextTemplateQuestionNo(ctx context.Context, tx *sql.Tx, templateID string) string {
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM question_templates WHERE template_id = ?", templateID).Scan(&count); err != nil {
		return "1"
	}
	return fmt.Sprintf("%d", count+1)
}

func refreshTemplateStatsTx(ctx context.Context, tx *sql.Tx, templateID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE paper_templates
		SET question_count = (
				SELECT COUNT(*)
				FROM question_templates
				WHERE template_id = ?
			),
			total_score = (
				SELECT COALESCE(ROUND(SUM(score)), 0)
				FROM question_templates
				WHERE template_id = ?
			),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		templateID, templateID, templateID,
	)
	return err
}

func pendingPages(jobs []ScanJob) int {
	total := 0
	for _, job := range jobs {
		if job.Progress < 100 {
			total += job.Pages
		}
	}
	return total
}

func decodeStringSlice(raw string, target *[]string) {
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		log.Printf("failed to decode json slice: %v", err)
	}
}

func decodeScanFiles(raw string, target *[]ScanFile) {
	if raw == "" {
		*target = []ScanFile{}
		return
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		log.Printf("failed to decode scan files: %v", err)
		*target = []ScanFile{}
	}
}

func templateRecordID(prefix string, current string) string {
	if current != "" && len(current) <= 40 {
		return current
	}
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func (s *Store) ensureSeedConstraints(ctx context.Context) error {
	cleanupStatements := []string{
		`DELETE later FROM exam_scores later
			JOIN exam_scores earlier
				ON later.student_name = earlier.student_name
				AND later.class_name = earlier.class_name
				AND later.paper_name = earlier.paper_name
				AND later.exam_at = earlier.exam_at
				AND later.id > earlier.id`,
		`DELETE later FROM student_risks later
			JOIN student_risks earlier
				ON later.student_name = earlier.student_name
				AND later.risk = earlier.risk
				AND later.id > earlier.id`,
	}
	for _, stmt := range cleanupStatements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	indexes := []struct {
		table string
		name  string
		stmt  string
	}{
		{
			table: "exam_scores",
			name:  "uk_exam_scores_seed",
			stmt:  "ALTER TABLE exam_scores ADD UNIQUE KEY uk_exam_scores_seed (student_name, class_name, paper_name, exam_at)",
		},
		{
			table: "student_risks",
			name:  "uk_student_risks_student_risk",
			stmt:  "ALTER TABLE student_risks ADD UNIQUE KEY uk_student_risks_student_risk (student_name, risk)",
		},
	}
	for _, index := range indexes {
		exists, err := s.indexExists(ctx, index.table, index.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := s.db.ExecContext(ctx, index.stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) indexExists(ctx context.Context, tableName string, indexName string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
			AND table_name = ?
			AND index_name = ?`, tableName, indexName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
