package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strings"
)

type PortalScoreSummary struct {
	GradeName string  `json:"gradeName"`
	ClassName string  `json:"className"`
	Highest   float64 `json:"highest"`
	Lowest    float64 `json:"lowest"`
	Average   float64 `json:"average"`
	Personal  float64 `json:"personal"`
}

type PortalHomework struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	DueAt   string `json:"dueAt"`
}

type PortalScorePoint struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

type PortalSubjectMistakes struct {
	Subject       string          `json:"subject"`
	PaperCount    int             `json:"paperCount"`
	HomeworkCount int             `json:"homeworkCount"`
	Items         []WrongQuestion `json:"items"`
}

type AICapability struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type StudentPortalResponse struct {
	StudentID    string                  `json:"studentId"`
	StudentName  string                  `json:"studentName"`
	GradeName    string                  `json:"gradeName"`
	ClassName    string                  `json:"className"`
	ScoreSummary PortalScoreSummary      `json:"scoreSummary"`
	Homework     []PortalHomework        `json:"homework"`
	ScoreTrend   []PortalScorePoint      `json:"scoreTrend"`
	Mistakes     []PortalSubjectMistakes `json:"mistakes"`
	AI           []AICapability          `json:"ai"`
}

type GuardianChild struct {
	StudentID   string `json:"studentId"`
	StudentName string `json:"studentName"`
	GradeName   string `json:"gradeName"`
	ClassName   string `json:"className"`
}

type GuardianPortalResponse struct {
	GuardianID string                `json:"guardianId"`
	Children   []GuardianChild       `json:"children"`
	Selected   StudentPortalResponse `json:"selected"`
}

func (app *App) handleStudentPortal(w http.ResponseWriter, r *http.Request) {
	studentID := strings.TrimSpace(r.URL.Query().Get("studentId"))
	if studentID == "" {
		studentID = "stu_001"
	}
	if app.store == nil {
		writeJSON(w, http.StatusOK, studentPortalFixture(studentID, "张三"))
		return
	}
	data, err := app.store.StudentPortal(r.Context(), studentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "student portal not found"})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (app *App) handleGuardianPortal(w http.ResponseWriter, r *http.Request) {
	guardianID := strings.TrimSpace(r.URL.Query().Get("guardianId"))
	if guardianID == "" {
		guardianID = "guardian_001"
	}
	studentID := strings.TrimSpace(r.URL.Query().Get("studentId"))
	if app.store == nil {
		child := studentPortalFixture("stu_001", "张三")
		writeJSON(w, http.StatusOK, GuardianPortalResponse{GuardianID: guardianID, Children: []GuardianChild{{StudentID: child.StudentID, StudentName: child.StudentName, GradeName: child.GradeName, ClassName: child.ClassName}}, Selected: child})
		return
	}
	data, err := app.store.GuardianPortal(r.Context(), guardianID, studentID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "approved guardian relationship is required"})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (app *App) handleAICapabilityRequest(w http.ResponseWriter, r *http.Request) {
	capability := r.PathValue("capability")
	if capability != "analysis" && capability != "ladder" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown capability"})
		return
	}
	var req struct {
		UserID    string `json:"userId"`
		StudentID string `json:"studentId"`
		Channel   string `json:"channel"`
	}
	if decodeJSON(r, &req) != nil || req.StudentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "studentId is required"})
		return
	}
	if req.Channel == "" {
		req.Channel = "portal"
	}
	if app.store != nil {
		_, err := app.store.db.ExecContext(r.Context(), `INSERT INTO ai_capability_requests (capability,user_id,student_id,channel) VALUES (?,?,?,?)`, capability, req.UserID, req.StudentID, req.Channel)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "waitlisted", "capability": capability, "available": false, "message": "能力接口已预留，模型厂商接入后开放"})
}

func (s *Store) StudentPortal(ctx context.Context, studentID string) (StudentPortalResponse, error) {
	var data StudentPortalResponse
	err := s.db.QueryRowContext(ctx, `SELECT st.id,st.name,g.name,c.name FROM students st JOIN classes c ON c.id=st.class_id LEFT JOIN grades g ON g.id=c.grade_id WHERE st.id=?`, studentID).Scan(&data.StudentID, &data.StudentName, &data.GradeName, &data.ClassName)
	if err != nil {
		return data, err
	}
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(score),0),COALESCE(MIN(score),0),COALESCE(AVG(score),0) FROM exam_scores WHERE class_name=?`, data.ClassName).Scan(&data.ScoreSummary.Highest, &data.ScoreSummary.Lowest, &data.ScoreSummary.Average)
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(score,0) FROM exam_scores WHERE student_name=? ORDER BY exam_at DESC LIMIT 1`, data.StudentName).Scan(&data.ScoreSummary.Personal)
	data.ScoreSummary.GradeName = data.GradeName
	data.ScoreSummary.ClassName = data.ClassName
	data.Homework = []PortalHomework{}
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.title,COALESCE(e.subject,''),CASE WHEN sub.id IS NULL THEN 'pending' ELSE sub.status END,COALESCE(DATE_FORMAT(a.due_at,'%Y-%m-%d %H:%i'),'') FROM assignments a LEFT JOIN exams e ON e.id=a.exam_id LEFT JOIN submissions sub ON sub.assignment_id=a.id AND sub.student_id=? WHERE a.class_id=(SELECT class_id FROM students WHERE id=?) ORDER BY a.created_at DESC LIMIT 8`, studentID, studentID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v PortalHomework
			if rows.Scan(&v.ID, &v.Title, &v.Subject, &v.Status, &v.DueAt) == nil {
				data.Homework = append(data.Homework, v)
			}
		}
	}
	data.ScoreTrend = []PortalScorePoint{}
	trendRows, err := s.db.QueryContext(ctx, `SELECT paper_name,score FROM exam_scores WHERE student_name=? ORDER BY exam_at`, data.StudentName)
	if err == nil {
		defer trendRows.Close()
		for trendRows.Next() {
			var v PortalScorePoint
			if trendRows.Scan(&v.Label, &v.Score) == nil {
				data.ScoreTrend = append(data.ScoreTrend, v)
			}
		}
	}
	wrong, err := s.WrongQuestions(ctx, WrongQuestionFilters{StudentName: data.StudentName})
	if err == nil {
		data.Mistakes = groupPortalMistakes(wrong)
	} else {
		data.Mistakes = []PortalSubjectMistakes{}
	}
	data.AI = portalAICapabilities()
	return data, nil
}

func (s *Store) GuardianPortal(ctx context.Context, guardianID, studentID string) (GuardianPortalResponse, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT st.id,st.name,g.name,c.name FROM student_guardians sg JOIN students st ON st.id=sg.student_id JOIN classes c ON c.id=st.class_id LEFT JOIN grades g ON g.id=c.grade_id WHERE sg.guardian_id=? ORDER BY st.name`, guardianID)
	if err != nil {
		return GuardianPortalResponse{}, err
	}
	defer rows.Close()
	children := []GuardianChild{}
	for rows.Next() {
		var c GuardianChild
		if rows.Scan(&c.StudentID, &c.StudentName, &c.GradeName, &c.ClassName) != nil {
			continue
		}
		children = append(children, c)
	}
	if len(children) == 0 {
		return GuardianPortalResponse{}, sql.ErrNoRows
	}
	if studentID == "" {
		studentID = children[0].StudentID
	}
	allowed := false
	for _, c := range children {
		if c.StudentID == studentID {
			allowed = true
		}
	}
	if !allowed {
		return GuardianPortalResponse{}, errors.New("guardian is not approved for this student")
	}
	selected, err := s.StudentPortal(ctx, studentID)
	return GuardianPortalResponse{GuardianID: guardianID, Children: children, Selected: selected}, err
}

func groupPortalMistakes(items []WrongQuestion) []PortalSubjectMistakes {
	groups := map[string]*PortalSubjectMistakes{}
	for _, item := range items {
		subject := "数学"
		g := groups[subject]
		if g == nil {
			g = &PortalSubjectMistakes{Subject: subject, Items: []WrongQuestion{}}
			groups[subject] = g
		}
		g.Items = append(g.Items, item)
		g.PaperCount++
	}
	keys := []string{}
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := []PortalSubjectMistakes{}
	for _, k := range keys {
		result = append(result, *groups[k])
	}
	return result
}

func portalAICapabilities() []AICapability {
	return []AICapability{{Key: "analysis", Name: "AI 学情分析", Status: "planned", Description: "多维分析学科与知识点短板，输出补漏地图"}, {Key: "ladder", Name: "天梯攻略", Status: "planned", Description: "根据补漏地图生成阶段练习册并周期复核"}}
}

func studentPortalFixture(id, name string) StudentPortalResponse {
	return StudentPortalResponse{StudentID: id, StudentName: name, GradeName: "六年级", ClassName: "六年级 3 班", ScoreSummary: PortalScoreSummary{GradeName: "六年级", ClassName: "六年级 3 班", Highest: 96, Lowest: 62, Average: 81.6, Personal: 85}, Homework: []PortalHomework{{ID: "assign_001", Title: "六年级数学期中卷", Subject: "数学", Status: "graded", DueAt: "2026-06-23 18:00"}, {ID: "assign_002", Title: "分数应用题专项", Subject: "数学", Status: "pending", DueAt: "2026-06-25 18:00"}}, ScoreTrend: []PortalScorePoint{{Label: "单元测验一", Score: 76}, {Label: "月考", Score: 81}, {Label: "期中考试", Score: 85}}, Mistakes: groupPortalMistakes(wrongQuestionsFixture()), AI: portalAICapabilities()}
}
