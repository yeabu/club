package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

type Store struct {
	db *sql.DB
}

func OpenStore(ctx context.Context, config Config) (*Store, error) {
	rootDSN := mysqlConfig(config, "").FormatDSN()
	rootDB, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return nil, err
	}
	defer rootDB.Close()

	if _, err := rootDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+config.MySQL.Database+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return nil, err
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
	if envBool("AUTO_MIGRATE", true) {
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
	if err := s.ensureSeedConstraints(ctx); err != nil {
		return err
	}
	return nil
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
		SELECT id, title, class_name, pages, status, progress
		FROM scan_jobs
		ORDER BY created_at DESC
		LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []ScanJob
	for rows.Next() {
		var job ScanJob
		if err := rows.Scan(&job.ID, &job.Title, &job.ClassName, &job.Pages, &job.Status, &job.Progress); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) ReviewQueue(ctx context.Context) ([]ReviewItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, student_name, paper_name, question_no, ai_score, full_score, confidence
		FROM subjective_reviews
		WHERE status = 'pending'
		ORDER BY updated_at DESC
		LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []ReviewItem
	for rows.Next() {
		var item ReviewItem
		var aiScore, fullScore float64
		if err := rows.Scan(&item.ID, &item.StudentName, &item.PaperName, &item.QuestionNo, &aiScore, &fullScore, &item.Confidence); err != nil {
			return nil, err
		}
		item.AIAdvice = fmt.Sprintf("%.0f / %.0f", aiScore, fullScore)
		reviews = append(reviews, item)
	}
	return reviews, rows.Err()
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

	var stats []KnowledgeStat
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

	var items []HomeworkWatch
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
		WHERE status = 'pending'
		ORDER BY updated_at DESC
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
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO grading_decisions (submission_id, question_id, final_score, decision, teacher_note)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE final_score = VALUES(final_score), decision = VALUES(decision), teacher_note = VALUES(teacher_note), updated_at = CURRENT_TIMESTAMP`,
		req.SubmissionID, req.QuestionID, req.FinalScore, req.Decision, req.TeacherNote,
	)
	if err != nil {
		return GradingDecisionResponse{}, err
	}
	_, _ = s.db.ExecContext(ctx, "UPDATE subjective_reviews SET status = 'reviewed', updated_at = CURRENT_TIMESTAMP WHERE submission_id = ? AND question_id = ?", req.SubmissionID, req.QuestionID)

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
		SELECT id, name, subject, grade, question_count, total_score
		FROM paper_templates
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []PaperTemplate
	for rows.Next() {
		var item PaperTemplate
		if err := rows.Scan(&item.ID, &item.Name, &item.Subject, &item.Grade, &item.QuestionCount, &item.TotalScore); err != nil {
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

func (s *Store) TemplateQuestions(ctx context.Context, templateID string) ([]QuestionTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, question_no, question_type, score, knowledge_json, page_no, x, y, width, height
		FROM question_templates
		WHERE template_id = ?
		ORDER BY page_no ASC, question_no + 0 ASC`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []QuestionTemplate
	for rows.Next() {
		var item QuestionTemplate
		var knowledgeJSON string
		if err := rows.Scan(&item.ID, &item.No, &item.Type, &item.Score, &knowledgeJSON, &item.Region.Page, &item.Region.X, &item.Region.Y, &item.Region.Width, &item.Region.Height); err != nil {
			return nil, err
		}
		decodeStringSlice(knowledgeJSON, &item.Knowledge)
		questions = append(questions, item)
	}
	return questions, rows.Err()
}

func (s *Store) ClassroomAnalytics(ctx context.Context) (ClassroomAnalytics, error) {
	var analytics ClassroomAnalytics
	analytics.ClassName = "六年级 3 班"
	if err := s.db.QueryRowContext(ctx, "SELECT AVG(score), MAX(score), MIN(score) FROM exam_scores WHERE class_name = ?", analytics.ClassName).Scan(&analytics.AverageScore, &analytics.HighestScore, &analytics.LowestScore); err != nil {
		return ClassroomAnalytics{}, err
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
