package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

type Store struct {
	db     *sql.DB
	config Config
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

	store := &Store{db: db, config: config}
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
		{table: "users", column: "school_id", definition: "VARCHAR(40) NOT NULL DEFAULT '' AFTER id"},
		{table: "users", column: "email", definition: "VARCHAR(120) DEFAULT '' AFTER mobile"},
		{table: "users", column: "status", definition: "VARCHAR(30) NOT NULL DEFAULT 'active' AFTER email"},
		{table: "knowledge_points", column: "school_id", definition: "VARCHAR(40) DEFAULT '' AFTER id"},
		{table: "knowledge_points", column: "grade_id", definition: "VARCHAR(40) DEFAULT '' AFTER school_id"},
		{table: "knowledge_points", column: "subject_id", definition: "VARCHAR(40) DEFAULT '' AFTER grade_id"},
		{table: "knowledge_points", column: "code", definition: "VARCHAR(80) DEFAULT '' AFTER parent_id"},
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
		{table: "scan_jobs", column: "scan_type", definition: "VARCHAR(30) NOT NULL DEFAULT 'answer_sheet' AFTER assignment_id"},
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
		{table: "objective_review_exceptions", column: "suggested_score", definition: "DECIMAL(6,2) NOT NULL DEFAULT 0 AFTER status"},
		{table: "subjective_reviews", column: "review_stage", definition: "VARCHAR(40) NOT NULL DEFAULT 'first_review' AFTER status"},
		{table: "subjective_reviews", column: "assignee_id", definition: "VARCHAR(40) DEFAULT '' AFTER review_stage"},
		{table: "subjective_reviews", column: "priority", definition: "INT NOT NULL DEFAULT 0 AFTER assignee_id"},
		{table: "grading_history", column: "actor_name", definition: "VARCHAR(80) DEFAULT '' AFTER actor_id"},
		{table: "grading_history", column: "review_stage", definition: "VARCHAR(40) DEFAULT 'first_review' AFTER actor_name"},
		{table: "grading_history", column: "model_version", definition: "VARCHAR(80) DEFAULT '' AFTER review_stage"},
		{table: "wrong_questions", column: "submission_id", definition: "VARCHAR(40) DEFAULT '' AFTER question_id"},
		{table: "wrong_questions", column: "question_no", definition: "VARCHAR(20) DEFAULT '' AFTER submission_id"},
		{table: "wrong_questions", column: "knowledge_point_id", definition: "VARCHAR(40) DEFAULT '' AFTER question_no"},
		{table: "wrong_questions", column: "error_type", definition: "VARCHAR(40) NOT NULL DEFAULT 'other' AFTER knowledge_point"},
		{table: "wrong_questions", column: "original_question", definition: "TEXT AFTER source_paper"},
		{table: "wrong_questions", column: "score", definition: "DECIMAL(6,2) NOT NULL DEFAULT 0 AFTER source_paper"},
		{table: "wrong_questions", column: "max_score", definition: "DECIMAL(6,2) NOT NULL DEFAULT 0 AFTER score"},
		{table: "wrong_questions", column: "correct_answer", definition: "TEXT AFTER max_score"},
		{table: "wrong_questions", column: "student_answer", definition: "TEXT AFTER correct_answer"},
		{table: "wrong_questions", column: "answer_image_url", definition: "VARCHAR(255) DEFAULT '' AFTER student_answer"},
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
		SELECT id, scan_type, title, class_name, template_id, template_version, pages, COALESCE(notes, ''), COALESCE(files_json, JSON_ARRAY()),
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
			&job.ID, &job.ScanType, &job.Title, &job.ClassName, &job.TemplateID, &job.TemplateVersion, &job.Pages, &job.Notes, &filesJSON,
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
		SELECT id, scan_type, title, class_name, template_id, template_version, pages, COALESCE(notes, ''), COALESCE(files_json, JSON_ARRAY()),
			status, progress, COALESCE(failure_reason, ''), retry_count, queue_status, COALESCE(queue_message, '')
		FROM scan_jobs
		WHERE id = ?
	LIMIT 1`, taskID).Scan(
		&job.ID, &job.ScanType, &job.Title, &job.ClassName, &job.TemplateID, &job.TemplateVersion, &job.Pages, &job.Notes, &filesJSON,
		&job.Status, &job.Progress, &job.FailureReason, &job.RetryCount, &job.QueueStatus, &job.QueueMessage,
	)
	if err != nil {
		return ScanJob{}, err
	}
	decodeScanFiles(filesJSON, &job.Files)
	return job, nil
}

func (s *Store) CreateScanTask(ctx context.Context, req ScanTaskRequest) (ScanJob, error) {
	req.ScanType = normalizeScanType(req.ScanType)
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
		ScanType:        req.ScanType,
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
			(id, scan_type, title, class_name, template_id, template_version, pages, notes, files_json, status, progress, failure_reason, retry_count, queue_status, queue_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', 0, ?, '')`,
		job.ID, job.ScanType, job.Title, job.ClassName, job.TemplateID, job.TemplateVersion, job.Pages, job.Notes, string(filesJSON), job.Status, job.Progress, job.QueueStatus,
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
		driver := s.config.StorageDriver
		bucket := "local-dev"
		if driver == "obs" {
			bucket = s.config.OBS.Bucket
		} else if driver == "minio" {
			bucket = s.config.MinIO.Bucket
		} else {
			driver = "local"
		}
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
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				url = VALUES(url),
				content_type = VALUES(content_type),
				size_bytes = VALUES(size_bytes),
				purpose = VALUES(purpose),
				owner_type = VALUES(owner_type),
				owner_id = VALUES(owner_id),
				metadata_json = VALUES(metadata_json)`,
			file.Key, bucket, driver, file.URL, file.ContentType, file.Size, purpose, ownerType, ownerID, string(metadata),
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
	if err := s.persistObjectiveResultsFromWorker(ctx, tx, taskID, req.Result); err != nil {
		return ScanJob{}, err
	}
	if err := tx.Commit(); err != nil {
		return ScanJob{}, err
	}
	return s.ScanTask(ctx, taskID)
}

func (s *Store) persistObjectiveResultsFromWorker(ctx context.Context, tx *sql.Tx, taskID string, result map[string]any) error {
	answers := workerOMRAnswers(result)
	if len(answers) == 0 {
		return nil
	}
	var templateID string
	if err := tx.QueryRowContext(ctx, "SELECT template_id FROM scan_jobs WHERE id = ? LIMIT 1", taskID).Scan(&templateID); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT sub.id, COALESCE(sub.student_name, students.name)
		FROM submissions sub
		LEFT JOIN students ON students.id = sub.student_id
		WHERE sub.scan_task_id = ?
		ORDER BY sub.submitted_at ASC`, taskID)
	if err != nil {
		return err
	}
	defer rows.Close()
	submissions := []struct {
		id          string
		studentName string
	}{}
	for rows.Next() {
		var item struct {
			id          string
			studentName string
		}
		if err := rows.Scan(&item.id, &item.studentName); err != nil {
			return err
		}
		submissions = append(submissions, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(submissions) == 0 {
		return nil
	}
	questionRows, err := tx.QueryContext(ctx, `
		SELECT id, question_no, question_type, score, COALESCE(standard_answer, '')
		FROM question_templates
		WHERE template_id = ? AND question_type IN ('single_choice', 'choice', 'judge', 'fill_blank', 'objective')
		ORDER BY CAST(question_no AS UNSIGNED), question_no`, templateID)
	if err != nil {
		return err
	}
	defer questionRows.Close()
	questions := map[string]struct {
		id     string
		no     string
		qtype  string
		score  float64
		answer string
	}{}
	for questionRows.Next() {
		var item struct {
			id     string
			no     string
			qtype  string
			score  float64
			answer string
		}
		if err := questionRows.Scan(&item.id, &item.no, &item.qtype, &item.score, &item.answer); err != nil {
			return err
		}
		questions[item.no] = item
	}
	if err := questionRows.Err(); err != nil {
		return err
	}
	for _, submission := range submissions {
		for _, answer := range answers {
			question, ok := questions[answer.QuestionNo]
			if !ok {
				continue
			}
			studentAnswer := strings.TrimSpace(answer.Selected)
			correctAnswer := strings.TrimSpace(question.answer)
			isCorrect := correctAnswer != "" && strings.EqualFold(studentAnswer, correctAnswer)
			score := 0.0
			if isCorrect {
				score = question.score
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO objective_grades (submission_id, question_id, student_answer, correct_answer, score, max_score, is_correct, confidence)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE
					student_answer = VALUES(student_answer),
					correct_answer = VALUES(correct_answer),
					score = VALUES(score),
					max_score = VALUES(max_score),
					is_correct = VALUES(is_correct),
					confidence = VALUES(confidence)`,
				submission.id, question.id, studentAnswer, correctAnswer, score, question.score, isCorrect, answer.Confidence,
			); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO question_scores (submission_id, question_id, question_no, score, max_score, source, status)
				VALUES (?, ?, ?, ?, ?, 'omr', 'final')
				ON DUPLICATE KEY UPDATE score = VALUES(score), max_score = VALUES(max_score), source = VALUES(source), status = VALUES(status), updated_at = CURRENT_TIMESTAMP`,
				submission.id, question.id, question.no, score, question.score,
			); err != nil {
				return err
			}
			if answer.Confidence < 80 || studentAnswer == "" {
				reason := "低置信度客观题需人工确认"
				if studentAnswer == "" {
					reason = "未识别到客观题答案"
				}
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO objective_review_exceptions (submission_id, question_id, question_no, student_answer, confidence, reason, status, suggested_score)
					VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)
					ON DUPLICATE KEY UPDATE student_answer = VALUES(student_answer), confidence = VALUES(confidence), reason = VALUES(reason), suggested_score = VALUES(suggested_score), updated_at = CURRENT_TIMESTAMP`,
					submission.id, question.id, question.no, studentAnswer, answer.Confidence, reason, score,
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type workerOMRAnswer struct {
	QuestionNo string
	Selected   string
	Confidence int
}

func workerOMRAnswers(result map[string]any) []workerOMRAnswer {
	rawResults, ok := result["omrResults"].([]any)
	if !ok {
		return nil
	}
	answers := []workerOMRAnswer{}
	for _, rawResult := range rawResults {
		resultMap, ok := rawResult.(map[string]any)
		if !ok {
			continue
		}
		rawAnswers, ok := resultMap["answers"].([]any)
		if !ok {
			continue
		}
		for _, rawAnswer := range rawAnswers {
			answerMap, ok := rawAnswer.(map[string]any)
			if !ok {
				continue
			}
			answers = append(answers, workerOMRAnswer{
				QuestionNo: fmt.Sprint(answerMap["questionNo"]),
				Selected:   fmt.Sprint(answerMap["selected"]),
				Confidence: numberToInt(answerMap["confidence"]),
			})
		}
	}
	return answers
}

func numberToInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(typed)
		return parsed
	default:
		return 0
	}
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

func (s *Store) WrongQuestions(ctx context.Context, filters WrongQuestionFilters) ([]WrongQuestion, error) {
	query := `
		SELECT wq.id, wq.student_id, COALESCE(students.name, sub.student_name, ''), COALESCE(sub.class_name, classes.name, ''),
			wq.submission_id, wq.question_id, wq.question_no, COALESCE(qt.question_type, ''), wq.knowledge_point,
			COALESCE(wq.error_type, 'other'), wq.wrong_reason, wq.source_paper, COALESCE(wq.original_question, ''),
			wq.score, wq.max_score, COALESCE(wq.correct_answer, ''), COALESCE(wq.student_answer, ''),
			COALESCE(wq.answer_image_url, ''), COALESCE(wq.explanation, ''), wq.correction_status, wq.repractice_status, wq.created_at
		FROM wrong_questions wq
		LEFT JOIN submissions sub ON sub.id = wq.submission_id
		LEFT JOIN students ON students.id = wq.student_id
		LEFT JOIN classes ON classes.id = students.class_id
		LEFT JOIN question_templates qt ON qt.id = wq.question_id
		WHERE 1 = 1`
	args := []any{}
	if filters.Paper != "" {
		query += " AND wq.source_paper = ?"
		args = append(args, filters.Paper)
	}
	if filters.ClassName != "" {
		query += " AND COALESCE(sub.class_name, classes.name, '') = ?"
		args = append(args, filters.ClassName)
	}
	if filters.StudentName != "" {
		query += " AND COALESCE(students.name, sub.student_name, '') = ?"
		args = append(args, filters.StudentName)
	}
	if filters.Knowledge != "" {
		query += " AND wq.knowledge_point = ?"
		args = append(args, filters.Knowledge)
	}
	if filters.ErrorType != "" {
		query += " AND wq.error_type = ?"
		args = append(args, filters.ErrorType)
	}
	if filters.Search != "" {
		query += " AND (wq.question_no LIKE ? OR wq.wrong_reason LIKE ? OR wq.source_paper LIKE ?)"
		pattern := "%" + filters.Search + "%"
		args = append(args, pattern, pattern, pattern)
	}
	query += " ORDER BY wq.updated_at DESC, wq.id DESC LIMIT 200"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []WrongQuestion{}
	for rows.Next() {
		item, err := scanWrongQuestion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) WrongQuestion(ctx context.Context, id int64) (WrongQuestion, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT wq.id, wq.student_id, COALESCE(students.name, sub.student_name, ''), COALESCE(sub.class_name, classes.name, ''),
			wq.submission_id, wq.question_id, wq.question_no, COALESCE(qt.question_type, ''), wq.knowledge_point,
			COALESCE(wq.error_type, 'other'), wq.wrong_reason, wq.source_paper, COALESCE(wq.original_question, ''),
			wq.score, wq.max_score, COALESCE(wq.correct_answer, ''), COALESCE(wq.student_answer, ''),
			COALESCE(wq.answer_image_url, ''), COALESCE(wq.explanation, ''), wq.correction_status, wq.repractice_status, wq.created_at
		FROM wrong_questions wq
		LEFT JOIN submissions sub ON sub.id = wq.submission_id
		LEFT JOIN students ON students.id = wq.student_id
		LEFT JOIN classes ON classes.id = students.class_id
		LEFT JOIN question_templates qt ON qt.id = wq.question_id
		WHERE wq.id = ?`, id)
	return scanWrongQuestion(row)
}

func scanWrongQuestion(scanner interface{ Scan(...any) error }) (WrongQuestion, error) {
	var item WrongQuestion
	var createdAt time.Time
	err := scanner.Scan(
		&item.ID, &item.StudentID, &item.StudentName, &item.ClassName, &item.SubmissionID, &item.QuestionID,
		&item.QuestionNo, &item.QuestionType, &item.KnowledgePoint, &item.ErrorType, &item.WrongReason,
		&item.SourcePaper, &item.OriginalQuestion, &item.Score, &item.MaxScore, &item.CorrectAnswer,
		&item.StudentAnswer, &item.AnswerImageURL, &item.Explanation, &item.CorrectionStatus,
		&item.RepracticeStatus, &createdAt,
	)
	if err != nil {
		return WrongQuestion{}, err
	}
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.Knowledge = []string{item.KnowledgePoint}
	return item, nil
}

func (s *Store) CreateRepracticeTask(ctx context.Context, req RepracticeTaskRequest) (RepracticeTaskResponse, error) {
	if len(req.WrongQuestionIDs) == 0 {
		return RepracticeTaskResponse{}, errors.New("wrongQuestionIds is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RepracticeTaskResponse{}, err
	}
	defer tx.Rollback()
	knowledgeSet := map[string]bool{}
	for _, id := range req.WrongQuestionIDs {
		var knowledge string
		if err := tx.QueryRowContext(ctx, "SELECT knowledge_point FROM wrong_questions WHERE id = ?", id).Scan(&knowledge); err != nil {
			return RepracticeTaskResponse{}, err
		}
		knowledgeSet[knowledge] = true
		if _, err := tx.ExecContext(ctx, "UPDATE wrong_questions SET repractice_status = 'assigned', updated_at = CURRENT_TIMESTAMP WHERE id = ?", id); err != nil {
			return RepracticeTaskResponse{}, err
		}
	}
	knowledge := make([]string, 0, len(knowledgeSet))
	for item := range knowledgeSet {
		knowledge = append(knowledge, item)
	}
	idsJSON, _ := json.Marshal(req.WrongQuestionIDs)
	knowledgeJSON, _ := json.Marshal(knowledge)
	taskID := fmt.Sprintf("repractice_%d", time.Now().UnixMilli())
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "错题订正与再练"
	}
	var dueAt any
	if strings.TrimSpace(req.DueAt) != "" {
		dueAt = req.DueAt
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO repractice_tasks (id, title, class_name, wrong_question_ids_json, knowledge_json, status, due_at)
		VALUES (?, ?, '六年级 3 班', ?, ?, 'assigned', ?)`, taskID, title, string(idsJSON), string(knowledgeJSON), dueAt); err != nil {
		return RepracticeTaskResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return RepracticeTaskResponse{}, err
	}
	return RepracticeTaskResponse{Status: "created", TaskID: taskID, LinkedCount: len(req.WrongQuestionIDs), Knowledge: knowledge}, nil
}

func (s *Store) LearningProfile(ctx context.Context, className string) (LearningProfileResponse, error) {
	if strings.TrimSpace(className) == "" {
		className = "六年级 3 班"
	}
	response := LearningProfileResponse{ClassName: className, KnowledgeMastery: []KnowledgeMastery{}, StudentRisks: []StudentRisk{}, HomeworkWatch: []HomeworkWatch{}}
	rows, err := s.db.QueryContext(ctx, `
		SELECT knowledge_point, mastery, wrong_count, student_count
		FROM knowledge_mastery_history
		WHERE class_name = ? AND student_id = ''
		ORDER BY knowledge_point, recorded_at DESC`, className)
	if err != nil {
		return LearningProfileResponse{}, err
	}
	defer rows.Close()
	byName := map[string]int{}
	for rows.Next() {
		var name string
		var mastery, wrongCount, studentCount int
		if err := rows.Scan(&name, &mastery, &wrongCount, &studentCount); err != nil {
			return LearningProfileResponse{}, err
		}
		index, exists := byName[name]
		if !exists {
			response.KnowledgeMastery = append(response.KnowledgeMastery, KnowledgeMastery{Name: name, Mastery: mastery, PreviousMastery: mastery, WrongCount: wrongCount, StudentCount: studentCount})
			byName[name] = len(response.KnowledgeMastery) - 1
			continue
		}
		if response.KnowledgeMastery[index].PreviousMastery == response.KnowledgeMastery[index].Mastery {
			response.KnowledgeMastery[index].PreviousMastery = mastery
			response.KnowledgeMastery[index].Trend = response.KnowledgeMastery[index].Mastery - mastery
		}
	}
	if response.StudentRisks, err = s.StudentRisks(ctx); err != nil {
		return LearningProfileResponse{}, err
	}
	if response.HomeworkWatch, err = s.HomeworkWatch(ctx); err != nil {
		return LearningProfileResponse{}, err
	}
	return response, nil
}

func (s *Store) GuardianReport(ctx context.Context, studentName string) (GuardianReportResponse, error) {
	if strings.TrimSpace(studentName) == "" {
		studentName = "李四"
	}
	response := GuardianReportResponse{StudentName: studentName, Weakness: []string{}, Actions: []string{}}
	_ = s.db.QueryRowContext(ctx, `SELECT class_name, score FROM exam_scores WHERE student_name = ? ORDER BY exam_at DESC LIMIT 1`, studentName).Scan(&response.ClassName, &response.Score)
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT wq.knowledge_point
		FROM wrong_questions wq
		LEFT JOIN students ON students.id = wq.student_id
		LEFT JOIN submissions sub ON sub.id = wq.submission_id
		WHERE COALESCE(students.name, sub.student_name, '') = ?
		ORDER BY wq.knowledge_point`, studentName)
	if err != nil {
		return GuardianReportResponse{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return GuardianReportResponse{}, err
		}
		response.Weakness = append(response.Weakness, item)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM wrong_questions wq
		LEFT JOIN students ON students.id = wq.student_id
		LEFT JOIN submissions sub ON sub.id = wq.submission_id
		WHERE COALESCE(students.name, sub.student_name, '') = ?`, studentName).Scan(&response.WrongCount); err != nil {
		return GuardianReportResponse{}, err
	}
	response.Summary = fmt.Sprintf("本次成绩 %.0f 分，共有 %d 道题需要继续巩固。", response.Score, response.WrongCount)
	response.Actions = []string{"每天安排 15 分钟订正", "优先复习薄弱知识点", "完成再练后和孩子一起检查步骤"}
	return response, nil
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

func (s *Store) StudentRisks(ctx context.Context) ([]StudentRisk, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT student_name, risk, weakness_json FROM student_risks ORDER BY updated_at DESC LIMIT 20")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []StudentRisk{}
	for rows.Next() {
		var item StudentRisk
		var weaknessJSON string
		if err := rows.Scan(&item.StudentName, &item.Risk, &weaknessJSON); err != nil {
			return nil, err
		}
		decodeStringSlice(weaknessJSON, &item.Weakness)
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
		errorType := classifyErrorType(req.TeacherNote, req.FinalScore)
		result, err := tx.ExecContext(ctx, `
			UPDATE wrong_questions
			SET student_id = ?,
				question_no = ?,
				knowledge_point = ?,
				error_type = ?,
				wrong_reason = ?,
				source_paper = ?,
				score = ?,
				max_score = ?,
				correct_answer = ?,
				student_answer = ?,
				answer_image_url = ?,
				explanation = ?,
				correction_status = 'pending',
				repractice_status = 'not_assigned',
				updated_at = CURRENT_TIMESTAMP
			WHERE submission_id = ? AND question_id = ?`,
			review.StudentID, review.QuestionNo, knowledgePoint, errorType, wrongReason(req, review.FullScore), review.PaperName,
			req.FinalScore, review.FullScore, review.StandardAnswer, review.OCRText, review.ImageURL, req.TeacherNote,
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
					(student_id, question_id, submission_id, question_no, knowledge_point, error_type, wrong_reason, source_paper, score, max_score, correct_answer, student_answer, answer_image_url, explanation, correction_status, repractice_status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', 'not_assigned')`,
				review.StudentID, req.QuestionID, req.SubmissionID, review.QuestionNo, knowledgePoint, errorType, wrongReason(req, review.FullScore), review.PaperName,
				req.FinalScore, review.FullScore, review.StandardAnswer, review.OCRText, review.ImageURL, req.TeacherNote,
			); err != nil {
				return GradingDecisionResponse{}, err
			}
		}
	} else if _, err := tx.ExecContext(ctx, "DELETE FROM wrong_questions WHERE submission_id = ? AND question_id = ?", req.SubmissionID, req.QuestionID); err != nil {
		return GradingDecisionResponse{}, err
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

func classifyErrorType(note string, score float64) string {
	normalized := strings.TrimSpace(note)
	switch {
	case strings.Contains(normalized, "计算") || strings.Contains(normalized, "运算"):
		return "calculation"
	case strings.Contains(normalized, "审题") || strings.Contains(normalized, "题意"):
		return "reading"
	case strings.Contains(normalized, "表达") || strings.Contains(normalized, "单位") || strings.Contains(normalized, "书写"):
		return "expression"
	case strings.Contains(normalized, "概念") || strings.Contains(normalized, "公式"):
		return "concept"
	case score <= 0:
		return "concept"
	default:
		return "other"
	}
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
		ClassName:       "六年级 3 班",
		QuestionStats:   []QuestionStat{},
		QuestionDetails: []QuestionDetailStat{},
		KnowledgeStats:  []KnowledgeStat{},
		StudentRisks:    []StudentRisk{},
		StudentScores:   []StudentScoreSummary{},
		ScoreBands: []ScoreBand{
			{Label: "0-59", Min: 0, Max: 59},
			{Label: "60-69", Min: 60, Max: 69},
			{Label: "70-79", Min: 70, Max: 79},
			{Label: "80-89", Min: 80, Max: 89},
			{Label: "90-100", Min: 90, Max: 100},
		},
		ObjectiveExceptions: []ObjectiveReviewException{},
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
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM students
		JOIN classes ON classes.id = students.class_id
		WHERE classes.name = ?`, analytics.ClassName).Scan(&analytics.StudentCount); err != nil {
		return ClassroomAnalytics{}, err
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exam_scores WHERE class_name = ?", analytics.ClassName).Scan(&analytics.GradedCount); err != nil {
		return ClassroomAnalytics{}, err
	}
	if analytics.StudentCount > 0 {
		analytics.CompletionRate = int(float64(analytics.GradedCount) / float64(analytics.StudentCount) * 100)
	}
	var passCount, excellentCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exam_scores WHERE class_name = ? AND score >= 60", analytics.ClassName).Scan(&passCount); err != nil {
		return ClassroomAnalytics{}, err
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exam_scores WHERE class_name = ? AND score >= 90", analytics.ClassName).Scan(&excellentCount); err != nil {
		return ClassroomAnalytics{}, err
	}
	if analytics.GradedCount > 0 {
		analytics.PassRate = int(float64(passCount) / float64(analytics.GradedCount) * 100)
		analytics.ExcellentRate = int(float64(excellentCount) / float64(analytics.GradedCount) * 100)
	}
	for index := range analytics.ScoreBands {
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exam_scores WHERE class_name = ? AND score BETWEEN ? AND ?", analytics.ClassName, analytics.ScoreBands[index].Min, analytics.ScoreBands[index].Max).Scan(&analytics.ScoreBands[index].Count); err != nil {
			return ClassroomAnalytics{}, err
		}
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
		analytics.QuestionDetails = append(analytics.QuestionDetails, QuestionDetailStat{
			No:             item.No,
			Type:           item.Type,
			Accuracy:       item.Accuracy,
			ScoreRate:      item.Accuracy,
			Difficulty:     difficultyLabel(item.Accuracy),
			Discrimination: discriminationScore(item.Accuracy),
			TypicalError:   typicalQuestionError(item.Type, item.Accuracy),
		})
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
	scoreRows, err := s.db.QueryContext(ctx, `
		SELECT student_name, class_name, score
		FROM exam_scores
		WHERE class_name = ?
		ORDER BY score DESC, student_name ASC`, analytics.ClassName)
	if err != nil {
		return ClassroomAnalytics{}, err
	}
	defer scoreRows.Close()
	rank := 0
	for scoreRows.Next() {
		rank++
		var item StudentScoreSummary
		if err := scoreRows.Scan(&item.StudentName, &item.ClassName, &item.Score); err != nil {
			return ClassroomAnalytics{}, err
		}
		item.Rank = rank
		item.Weakness = studentWeaknessFromAnalytics(item.StudentName, analytics.StudentRisks, analytics.KnowledgeStats)
		analytics.StudentScores = append(analytics.StudentScores, item)
	}
	exceptionRows, err := s.db.QueryContext(ctx, `
		SELECT ex.id, ex.submission_id, COALESCE(sub.student_name, students.name, ''), ex.question_id, ex.question_no,
			ex.student_answer, ex.confidence, ex.reason, ex.status, ex.suggested_score
		FROM objective_review_exceptions ex
		LEFT JOIN submissions sub ON sub.id = ex.submission_id
		LEFT JOIN students ON students.id = sub.student_id
		ORDER BY ex.updated_at DESC
		LIMIT 20`)
	if err != nil {
		return ClassroomAnalytics{}, err
	}
	defer exceptionRows.Close()
	for exceptionRows.Next() {
		var item ObjectiveReviewException
		if err := exceptionRows.Scan(&item.ID, &item.SubmissionID, &item.StudentName, &item.QuestionID, &item.QuestionNo, &item.Answer, &item.Confidence, &item.Reason, &item.Status, &item.SuggestedScore); err != nil {
			return ClassroomAnalytics{}, err
		}
		analytics.ObjectiveExceptions = append(analytics.ObjectiveExceptions, item)
	}

	return analytics, nil
}

func (s *Store) GenerateExamScores(ctx context.Context, className string) (ScoreGenerationResponse, error) {
	if strings.TrimSpace(className) == "" {
		className = "六年级 3 班"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ScoreGenerationResponse{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO question_scores (submission_id, question_id, question_no, score, max_score, source, status)
		SELECT og.submission_id, og.question_id, qt.question_no, og.score, og.max_score, 'omr', 'final'
		FROM objective_grades og
		JOIN question_templates qt ON qt.id = og.question_id
		WHERE NOT EXISTS (
			SELECT 1 FROM objective_review_exceptions ex
			WHERE ex.submission_id = og.submission_id AND ex.question_id = og.question_id AND ex.status = 'pending'
		)
		ON DUPLICATE KEY UPDATE
			score = VALUES(score),
			max_score = VALUES(max_score),
			source = VALUES(source),
			status = VALUES(status),
			updated_at = CURRENT_TIMESTAMP`); err != nil {
		return ScoreGenerationResponse{}, err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT sub.id, COALESCE(sub.student_name, students.name), COALESCE(sub.class_name, classes.name), COALESCE(assignments.title, '未命名考试'), COALESCE(SUM(qs.score), 0)
		FROM submissions sub
		JOIN assignments ON assignments.id = sub.assignment_id
		JOIN students ON students.id = sub.student_id
		JOIN classes ON classes.id = students.class_id
		LEFT JOIN question_scores qs ON qs.submission_id = sub.id
		WHERE COALESCE(sub.class_name, classes.name) = ?
		GROUP BY sub.id, sub.student_name, students.name, sub.class_name, classes.name, assignments.title`, className)
	if err != nil {
		return ScoreGenerationResponse{}, err
	}
	type generatedScoreRow struct {
		SubmissionID string
		StudentName  string
		ClassName    string
		PaperName    string
		Score        float64
	}
	scoreData := []generatedScoreRow{}
	for rows.Next() {
		var item generatedScoreRow
		if err := rows.Scan(&item.SubmissionID, &item.StudentName, &item.ClassName, &item.PaperName, &item.Score); err != nil {
			rows.Close()
			return ScoreGenerationResponse{}, err
		}
		scoreData = append(scoreData, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return ScoreGenerationResponse{}, err
	}
	rows.Close()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO wrong_questions
			(student_id, question_id, submission_id, question_no, knowledge_point, error_type, wrong_reason, source_paper,
			score, max_score, correct_answer, student_answer, answer_image_url, explanation, correction_status, repractice_status)
		SELECT sub.student_id, og.question_id, og.submission_id, qt.question_no,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(qt.knowledge_json, '$[0]')), '未归类'),
			'concept', '客观题答案与标准答案不一致', assignments.title,
			og.score, og.max_score, og.correct_answer, og.student_answer, COALESCE(sub.file_url, ''),
			'核对标准答案并完成同知识点订正。', 'pending', 'not_assigned'
		FROM objective_grades og
		JOIN question_templates qt ON qt.id = og.question_id
		JOIN submissions sub ON sub.id = og.submission_id
		JOIN assignments ON assignments.id = sub.assignment_id
		WHERE og.is_correct = FALSE
			AND NOT EXISTS (
				SELECT 1 FROM objective_review_exceptions ex
				WHERE ex.submission_id = og.submission_id AND ex.question_id = og.question_id AND ex.status = 'pending'
			)
		ON DUPLICATE KEY UPDATE
			student_id = VALUES(student_id), question_no = VALUES(question_no), knowledge_point = VALUES(knowledge_point),
			error_type = VALUES(error_type), wrong_reason = VALUES(wrong_reason), source_paper = VALUES(source_paper),
			score = VALUES(score), max_score = VALUES(max_score), correct_answer = VALUES(correct_answer),
			student_answer = VALUES(student_answer), answer_image_url = VALUES(answer_image_url), explanation = VALUES(explanation),
			correction_status = 'pending', updated_at = CURRENT_TIMESTAMP`); err != nil {
		return ScoreGenerationResponse{}, err
	}
	generated := 0
	for _, item := range scoreData {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO exam_scores (student_name, class_name, paper_name, score, exam_at)
			VALUES (?, ?, ?, ?, CURRENT_DATE())
			ON DUPLICATE KEY UPDATE score = VALUES(score)`,
			item.StudentName, item.ClassName, item.PaperName, item.Score,
		); err != nil {
			return ScoreGenerationResponse{}, err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE submissions SET status = 'graded', graded_at = CURRENT_TIMESTAMP WHERE id = ?", item.SubmissionID); err != nil {
			return ScoreGenerationResponse{}, err
		}
		generated++
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO knowledge_mastery_history (class_name, student_id, knowledge_point, mastery, wrong_count, student_count, recorded_at)
		SELECT COALESCE(sub.class_name, classes.name), '',
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(qt.knowledge_json, '$[0]')), '未归类'),
			ROUND(LEAST(100, GREATEST(0,
				AVG(CASE WHEN qs.max_score > 0 THEN qs.score / qs.max_score ELSE 0 END) * 80
				+ (1 - COUNT(wq.id) / GREATEST(COUNT(qs.id), 1)) * 20
			))),
			COUNT(wq.id), COUNT(DISTINCT sub.student_id), CURRENT_DATE()
		FROM question_scores qs
		JOIN submissions sub ON sub.id = qs.submission_id
		JOIN students ON students.id = sub.student_id
		JOIN classes ON classes.id = students.class_id
		JOIN question_templates qt ON qt.id = qs.question_id
		LEFT JOIN wrong_questions wq ON wq.submission_id = qs.submission_id AND wq.question_id = qs.question_id
		WHERE COALESCE(sub.class_name, classes.name) = ?
		GROUP BY COALESCE(sub.class_name, classes.name), COALESCE(JSON_UNQUOTE(JSON_EXTRACT(qt.knowledge_json, '$[0]')), '未归类')
		ON DUPLICATE KEY UPDATE mastery = VALUES(mastery), wrong_count = VALUES(wrong_count), student_count = VALUES(student_count)`, className); err != nil {
		return ScoreGenerationResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return ScoreGenerationResponse{}, err
	}
	return ScoreGenerationResponse{Status: "generated", ClassName: className, Generated: generated}, nil
}

func difficultyLabel(accuracy int) string {
	if accuracy < 50 {
		return "偏难"
	}
	if accuracy < 75 {
		return "中等"
	}
	return "容易"
}

func discriminationScore(accuracy int) int {
	if accuracy < 45 {
		return 62
	}
	if accuracy < 75 {
		return 78
	}
	return 52
}

func typicalQuestionError(questionType string, accuracy int) string {
	if accuracy >= 80 {
		return "整体掌握较好，关注个别粗心"
	}
	if strings.Contains(questionType, "选择") || strings.Contains(questionType, "single") {
		return "选项干扰识别不足"
	}
	return "关键步骤或知识点迁移不稳定"
}

func studentWeaknessFromAnalytics(studentName string, risks []StudentRisk, knowledge []KnowledgeStat) []string {
	for _, risk := range risks {
		if risk.StudentName == studentName && len(risk.Weakness) > 0 {
			return risk.Weakness
		}
	}
	result := []string{}
	for index, item := range knowledge {
		if index >= 2 {
			break
		}
		result = append(result, item.Name)
	}
	return result
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
