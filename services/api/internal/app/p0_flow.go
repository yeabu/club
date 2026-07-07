package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var errPendingReviewBlocksScores = errors.New("pending review items block score generation")

func defaultAutoGradingPolicy() AutoGradingPolicy {
	return AutoGradingPolicy{
		ObjectiveMinConfidence:  80,
		SubjectiveMinConfidence: 92,
		SubjectiveMaxScoreShare: 0.15,
		SamplingRate:            10,
		RequireReason:           true,
		AbnormalFallback:        true,
	}
}

func normalizePolicy(policy *AutoGradingPolicy) AutoGradingPolicy {
	if policy == nil {
		return defaultAutoGradingPolicy()
	}
	out := *policy
	if out.ObjectiveMinConfidence <= 0 {
		out.ObjectiveMinConfidence = 80
	}
	if out.SubjectiveMinConfidence <= 0 {
		out.SubjectiveMinConfidence = 92
	}
	if out.SubjectiveMaxScoreShare <= 0 {
		out.SubjectiveMaxScoreShare = 0.15
	}
	if out.SamplingRate < 0 {
		out.SamplingRate = 0
	}
	return out
}

func (s *Store) DefaultGradingPolicy(ctx context.Context) (AutoGradingPolicy, error) {
	var policy AutoGradingPolicy
	err := s.db.QueryRowContext(ctx, `
		SELECT objective_min_confidence, subjective_min_confidence, subjective_max_score_share, sampling_rate, require_reason, abnormal_fallback
		FROM grading_policies
		WHERE id='default'
		LIMIT 1`).Scan(
		&policy.ObjectiveMinConfidence, &policy.SubjectiveMinConfidence, &policy.SubjectiveMaxScoreShare,
		&policy.SamplingRate, &policy.RequireReason, &policy.AbnormalFallback,
	)
	if err == sql.ErrNoRows {
		return defaultAutoGradingPolicy(), nil
	}
	if err != nil {
		return AutoGradingPolicy{}, err
	}
	return normalizePolicy(&policy), nil
}

func (s *Store) SaveDefaultGradingPolicy(ctx context.Context, policy AutoGradingPolicy) (AutoGradingPolicy, error) {
	policy = normalizePolicy(&policy)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO grading_policies
			(id, name, objective_min_confidence, subjective_min_confidence, subjective_max_score_share, sampling_rate, require_reason, abnormal_fallback)
		VALUES ('default', '默认自动阅卷策略', ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			objective_min_confidence=VALUES(objective_min_confidence),
			subjective_min_confidence=VALUES(subjective_min_confidence),
			subjective_max_score_share=VALUES(subjective_max_score_share),
			sampling_rate=VALUES(sampling_rate),
			require_reason=VALUES(require_reason),
			abnormal_fallback=VALUES(abnormal_fallback)`,
		policy.ObjectiveMinConfidence, policy.SubjectiveMinConfidence, policy.SubjectiveMaxScoreShare,
		policy.SamplingRate, policy.RequireReason, policy.AbnormalFallback,
	)
	return policy, err
}

func (s *Store) PublishExam(ctx context.Context, req ExamPublishRequest) (ExamRecord, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.SchoolID = strings.TrimSpace(req.SchoolID)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Grade = strings.TrimSpace(req.Grade)
	req.ClassID = strings.TrimSpace(req.ClassID)
	req.TeacherID = strings.TrimSpace(req.TeacherID)
	req.TemplateID = strings.TrimSpace(req.TemplateID)
	if req.Name == "" || req.SchoolID == "" || req.ClassID == "" || req.TeacherID == "" || req.TemplateID == "" {
		return ExamRecord{}, errors.New("name, schoolId, classId, teacherId and templateId are required")
	}
	template, err := s.Template(ctx, req.TemplateID)
	if err != nil {
		return ExamRecord{}, err
	}
	if template.Status != "published" {
		return ExamRecord{}, errTemplateNotPublished
	}
	if req.Subject == "" {
		req.Subject = template.Subject
	}
	if req.Grade == "" {
		req.Grade = template.Grade
	}
	if req.MaxScore <= 0 {
		req.MaxScore = float64(template.TotalScore)
	}
	policy := normalizePolicy(req.GradingPolicy)
	policyRaw, err := json.Marshal(policy)
	if err != nil {
		return ExamRecord{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ExamRecord{}, err
	}
	defer tx.Rollback()
	var className string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM classes WHERE id=? AND school_id=? LIMIT 1`, req.ClassID, req.SchoolID).Scan(&className); err != nil {
		return ExamRecord{}, err
	}
	var teacherOK int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM teacher_classes WHERE teacher_id=? AND class_id=?`, req.TeacherID, req.ClassID).Scan(&teacherOK); err != nil {
		return ExamRecord{}, err
	}
	if teacherOK == 0 {
		return ExamRecord{}, errors.New("teacher is not bound to class")
	}
	studentIDs := req.StudentIDs
	if len(studentIDs) == 0 {
		rows, err := tx.QueryContext(ctx, `SELECT id FROM students WHERE class_id=? ORDER BY student_no, name`, req.ClassID)
		if err != nil {
			return ExamRecord{}, err
		}
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				studentIDs = append(studentIDs, id)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return ExamRecord{}, err
		}
		rows.Close()
	}
	if len(studentIDs) == 0 {
		return ExamRecord{}, errors.New("studentIds or class roster is required")
	}
	examID := fmt.Sprintf("exam_%d", time.Now().UnixNano())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO exams
			(id, school_id, name, subject, grade, class_id, teacher_id, template_id, template_version, status, max_score, roster_locked_at, grading_policy_json, started_at, ended_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', ?, CURRENT_TIMESTAMP, ?, NULLIF(?, ''), NULLIF(?, ''))`,
		examID, req.SchoolID, req.Name, req.Subject, req.Grade, req.ClassID, req.TeacherID, req.TemplateID, template.Version, req.MaxScore, string(policyRaw), req.StartedAt, req.EndedAt,
	); err != nil {
		return ExamRecord{}, err
	}
	assignmentID := "assign_" + strings.TrimPrefix(examID, "exam_")
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO assignments (id, exam_id, title, kind, class_id, class_name, template_id, template_version, teacher_id, published_at, status)
		VALUES (?, ?, ?, 'exam', ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, 'open')`,
		assignmentID, examID, req.Name, req.ClassID, className, req.TemplateID, template.Version, req.TeacherID,
	); err != nil {
		return ExamRecord{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO assignment_classes (assignment_id, class_id, class_name, publish_status)
		VALUES (?, ?, ?, 'published')
		ON DUPLICATE KEY UPDATE class_name=VALUES(class_name), publish_status='published'`,
		assignmentID, req.ClassID, className,
	); err != nil {
		return ExamRecord{}, err
	}
	for _, studentID := range studentIDs {
		studentID = strings.TrimSpace(studentID)
		if studentID == "" {
			continue
		}
		var studentName string
		if err := tx.QueryRowContext(ctx, `SELECT name FROM students WHERE id=? AND class_id=? LIMIT 1`, studentID, req.ClassID).Scan(&studentName); err != nil {
			return ExamRecord{}, err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO exam_students (exam_id, student_id, student_name, class_id, status)
			VALUES (?, ?, ?, ?, 'registered')`, examID, studentID, studentName, req.ClassID,
		); err != nil {
			return ExamRecord{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return ExamRecord{}, err
	}
	return s.Exam(ctx, examID)
}

func (s *Store) Exam(ctx context.Context, examID string) (ExamRecord, error) {
	var item ExamRecord
	var policyRaw string
	var rosterLocked sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT e.id, e.name, e.school_id, e.subject, e.grade, COALESCE(e.class_id,''), COALESCE(c.name,''), COALESCE(e.teacher_id,''),
			e.template_id, e.template_version, e.status, e.max_score, e.roster_locked_at, COALESCE(e.grading_policy_json, JSON_OBJECT())
		FROM exams e
		LEFT JOIN classes c ON c.id=e.class_id
		WHERE e.id=?
		LIMIT 1`, examID).Scan(
		&item.ID, &item.Name, &item.SchoolID, &item.Subject, &item.Grade, &item.ClassID, &item.ClassName,
		&item.TeacherID, &item.TemplateID, &item.TemplateVersion, &item.Status, &item.MaxScore, &rosterLocked, &policyRaw,
	)
	if err != nil {
		return ExamRecord{}, err
	}
	item.RosterLocked = rosterLocked.Valid
	if err := json.Unmarshal([]byte(policyRaw), &item.GradingPolicy); err != nil {
		item.GradingPolicy = defaultAutoGradingPolicy()
	}
	item.GradingPolicy = normalizePolicy(&item.GradingPolicy)
	rows, err := s.db.QueryContext(ctx, `
		SELECT student_id, student_name, status
		FROM exam_students
		WHERE exam_id=?
		ORDER BY student_name`, examID)
	if err != nil {
		return ExamRecord{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var student ExamStudent
		if rows.Scan(&student.StudentID, &student.Name, &student.Status) == nil {
			item.Students = append(item.Students, student)
		}
	}
	return item, rows.Err()
}

func (s *Store) UpdateExamAttendance(ctx context.Context, examID string, studentID string, req ExamAttendanceRequest) (ExamRecord, error) {
	status := strings.TrimSpace(req.Status)
	switch status {
	case "registered", "absent", "makeup", "replaced":
	default:
		return ExamRecord{}, errors.New("status must be registered, absent, makeup or replaced")
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE exam_students
		SET status=?, note=?, updated_at=CURRENT_TIMESTAMP
		WHERE exam_id=? AND student_id=?`, status, strings.TrimSpace(req.Note), examID, studentID)
	if err != nil {
		return ExamRecord{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ExamRecord{}, err
	}
	if affected == 0 {
		return ExamRecord{}, sql.ErrNoRows
	}
	return s.Exam(ctx, examID)
}

func (s *Store) AssignmentIDForExam(ctx context.Context, examID string) (string, error) {
	var assignmentID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM assignments WHERE exam_id=? ORDER BY created_at DESC LIMIT 1`, examID).Scan(&assignmentID)
	return assignmentID, err
}

func (s *Store) StudentName(ctx context.Context, studentID string) (string, error) {
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM students WHERE id=? LIMIT 1`, studentID).Scan(&name)
	return name, err
}

func (s *Store) ResolveObjectiveException(ctx context.Context, exceptionID int64, req ObjectiveReviewDecisionRequest) (ObjectiveReviewException, error) {
	decision := strings.TrimSpace(req.Decision)
	if decision == "" {
		decision = "confirmed"
	}
	if decision != "confirmed" && decision != "rejected" {
		return ObjectiveReviewException{}, errors.New("decision must be confirmed or rejected")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ObjectiveReviewException{}, err
	}
	defer tx.Rollback()
	var ex ObjectiveReviewException
	var maxScore float64
	err = tx.QueryRowContext(ctx, `
		SELECT ex.id, ex.submission_id, COALESCE(sub.student_name, students.name, ''), ex.question_id, ex.question_no,
			ex.student_answer, ex.confidence, ex.reason, ex.status, ex.suggested_score, COALESCE(qt.score, og.max_score, 0)
		FROM objective_review_exceptions ex
		LEFT JOIN submissions sub ON sub.id=ex.submission_id
		LEFT JOIN students ON students.id=sub.student_id
		LEFT JOIN question_templates qt ON qt.id=ex.question_id
		LEFT JOIN objective_grades og ON og.submission_id=ex.submission_id AND og.question_id=ex.question_id
		WHERE ex.id=?
		LIMIT 1`, exceptionID).Scan(
		&ex.ID, &ex.SubmissionID, &ex.StudentName, &ex.QuestionID, &ex.QuestionNo, &ex.Answer,
		&ex.Confidence, &ex.Reason, &ex.Status, &ex.SuggestedScore, &maxScore,
	)
	if err != nil {
		return ObjectiveReviewException{}, err
	}
	score := req.Score
	if score < 0 || score > maxScore {
		return ObjectiveReviewException{}, errors.New("score is outside question range")
	}
	status := "resolved"
	if decision == "rejected" {
		status = "rejected"
	}
	if _, err := tx.ExecContext(ctx, `UPDATE objective_review_exceptions SET status=?, suggested_score=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, status, score, exceptionID); err != nil {
		return ObjectiveReviewException{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO question_scores (submission_id, question_id, question_no, score, max_score, source, status)
		VALUES (?, ?, ?, ?, ?, 'objective_review', 'final')
		ON DUPLICATE KEY UPDATE score=VALUES(score), max_score=VALUES(max_score), source=VALUES(source), status='final', updated_at=CURRENT_TIMESTAMP`,
		ex.SubmissionID, ex.QuestionID, ex.QuestionNo, score, maxScore,
	); err != nil {
		return ObjectiveReviewException{}, err
	}
	actorName := strings.TrimSpace(req.ActorName)
	if actorName == "" {
		actorName = "当前教师"
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO grading_history (submission_id, question_id, action, score, note, actor_name, review_stage)
		VALUES (?, ?, ?, ?, ?, ?, 'objective_review')`, ex.SubmissionID, ex.QuestionID, decision, score, req.TeacherNote, actorName,
	); err != nil {
		return ObjectiveReviewException{}, err
	}
	if err := tx.Commit(); err != nil {
		return ObjectiveReviewException{}, err
	}
	return s.ObjectiveException(ctx, exceptionID)
}

func (s *Store) ObjectiveException(ctx context.Context, exceptionID int64) (ObjectiveReviewException, error) {
	var item ObjectiveReviewException
	err := s.db.QueryRowContext(ctx, `
		SELECT ex.id, ex.submission_id, COALESCE(sub.student_name, students.name, ''), ex.question_id, ex.question_no,
			ex.student_answer, ex.confidence, ex.reason, ex.status, ex.suggested_score
		FROM objective_review_exceptions ex
		LEFT JOIN submissions sub ON sub.id=ex.submission_id
		LEFT JOIN students ON students.id=sub.student_id
		WHERE ex.id=?
		LIMIT 1`, exceptionID).Scan(
		&item.ID, &item.SubmissionID, &item.StudentName, &item.QuestionID, &item.QuestionNo,
		&item.Answer, &item.Confidence, &item.Reason, &item.Status, &item.SuggestedScore,
	)
	return item, err
}

func (s *Store) SubmitRepractice(ctx context.Context, taskID string, req RepracticeSubmissionRequest) (RepracticeSubmissionResponse, error) {
	taskID = strings.TrimSpace(taskID)
	req.StudentID = strings.TrimSpace(req.StudentID)
	if taskID == "" || req.StudentID == "" || len(req.Answers) == 0 {
		return RepracticeSubmissionResponse{}, errors.New("taskId, studentId and answers are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RepracticeSubmissionResponse{}, err
	}
	defer tx.Rollback()
	submitted := 0
	for _, answer := range req.Answers {
		if answer.WrongQuestionID <= 0 {
			continue
		}
		status := strings.TrimSpace(answer.Status)
		if status == "" {
			status = "submitted"
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO repractice_submissions (task_id, wrong_question_id, student_id, answer_text, status)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE answer_text=VALUES(answer_text), status=VALUES(status), submitted_at=CURRENT_TIMESTAMP`,
			taskID, answer.WrongQuestionID, req.StudentID, answer.Answer, status,
		); err != nil {
			return RepracticeSubmissionResponse{}, err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE wrong_questions
			SET correction_status='submitted', repractice_status='submitted', updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND student_id=?`, answer.WrongQuestionID, req.StudentID,
		); err != nil {
			return RepracticeSubmissionResponse{}, err
		}
		submitted++
	}
	if err := tx.Commit(); err != nil {
		return RepracticeSubmissionResponse{}, err
	}
	return RepracticeSubmissionResponse{Status: "submitted", TaskID: taskID, Submitted: submitted}, nil
}
