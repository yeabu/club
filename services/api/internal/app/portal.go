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
	Highest   float64 `json:"highest,omitempty"`
	Lowest    float64 `json:"lowest,omitempty"`
	Average   float64 `json:"average,omitempty"`
	Personal  float64 `json:"personal"`
}

type PortalScoreStats struct {
	Scope   string  `json:"scope"`
	Name    string  `json:"name"`
	Highest float64 `json:"highest"`
	Lowest  float64 `json:"lowest"`
	Average float64 `json:"average"`
	Rank    int     `json:"rank,omitempty"`
	Total   int     `json:"total,omitempty"`
}

type PortalHomework struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	DueAt   string `json:"dueAt"`
}

type PortalHomeworkSummary struct {
	Total          int `json:"total"`
	Completed      int `json:"completed"`
	Pending        int `json:"pending"`
	Overdue        int `json:"overdue"`
	Completion     int `json:"completion"`
	NeedsAttention int `json:"needsAttention"`
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
	CTA         string `json:"cta"`
	PriceLabel  string `json:"priceLabel,omitempty"`
	ValuePoint  string `json:"valuePoint,omitempty"`
}

type PortalOffer struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CTA         string `json:"cta"`
	PriceLabel  string `json:"priceLabel"`
}

type StudentPortalResponse struct {
	StudentID       string                  `json:"studentId"`
	StudentName     string                  `json:"studentName"`
	GradeName       string                  `json:"gradeName"`
	ClassName       string                  `json:"className"`
	ScoreSummary    PortalScoreSummary      `json:"scoreSummary"`
	GradeSummary    *PortalScoreStats       `json:"gradeSummary,omitempty"`
	ClassSummary    *PortalScoreStats       `json:"classSummary,omitempty"`
	HomeworkSummary PortalHomeworkSummary   `json:"homeworkSummary"`
	Homework        []PortalHomework        `json:"homework"`
	ScoreTrend      []PortalScorePoint      `json:"scoreTrend"`
	Mistakes        []PortalSubjectMistakes `json:"mistakes"`
	WeakPoints      []string                `json:"weakPoints"`
	AI              []AICapability          `json:"ai"`
	Offers          []PortalOffer           `json:"offers,omitempty"`
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
		children := []GuardianChild{
			{StudentID: "stu_001", StudentName: "张三", GradeName: "六年级", ClassName: "六年级 3 班"},
			{StudentID: "stu_002", StudentName: "李四", GradeName: "六年级", ClassName: "六年级 3 班"},
		}
		selectedID := studentID
		if selectedID == "" {
			selectedID = children[0].StudentID
		}
		selectedName := "张三"
		if selectedID == "stu_002" {
			selectedName = "李四"
		}
		writeJSON(w, http.StatusOK, GuardianPortalResponse{GuardianID: guardianID, Children: children, Selected: studentPortalFixture(selectedID, selectedName)})
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
	var task *AITaskRecord
	if app.store != nil {
		_, err := app.store.db.ExecContext(r.Context(), `INSERT INTO ai_capability_requests (capability,user_id,student_id,channel) VALUES (?,?,?,?)`, capability, req.UserID, req.StudentID, req.Channel)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		taskType := "student_learning_analysis"
		if capability == "ladder" {
			taskType = "personalized_ladder_plan"
		}
		created, err := app.store.CreateAITask(r.Context(), taskType, map[string]any{"capability": capability, "studentId": req.StudentID, "channel": req.Channel, "userId": req.UserID}, "", "", "student", req.StudentID, req.UserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		task = &created
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "pending", "capability": capability, "available": app.config.AIProvider.BaseURL != "" && app.config.AIProvider.APIKey != "", "message": "能力接口已接入 AI 任务队列，配置三方 Provider 后可派发", "task": task})
}

func (s *Store) StudentPortal(ctx context.Context, studentID string) (StudentPortalResponse, error) {
	var data StudentPortalResponse
	var gradeID string
	err := s.db.QueryRowContext(ctx, `SELECT st.id,st.name,g.id,COALESCE(g.name,''),c.name FROM students st JOIN classes c ON c.id=st.class_id LEFT JOIN grades g ON g.id=c.grade_id WHERE st.id=?`, studentID).Scan(&data.StudentID, &data.StudentName, &gradeID, &data.GradeName, &data.ClassName)
	if err != nil {
		return data, err
	}
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
				if v.Subject == "" {
					v.Subject = "综合"
				}
				data.Homework = append(data.Homework, v)
			}
		}
	}
	data.HomeworkSummary = summarizePortalHomework(data.Homework)
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
		data.WeakPoints = portalWeakPoints(wrong)
	} else {
		data.Mistakes = []PortalSubjectMistakes{}
		data.WeakPoints = []string{}
	}
	data.AI = portalAICapabilities()
	data.Offers = portalOffers()
	return data, nil
}

func (s *Store) portalScoreStats(ctx context.Context, scope, name, studentName, condition string, arg any) PortalScoreStats {
	stats := PortalScoreStats{Scope: scope, Name: name}
	query := `
		SELECT COALESCE(MAX(es.score),0), COALESCE(MIN(es.score),0), COALESCE(AVG(es.score),0), COUNT(*)
		FROM exam_scores es
		LEFT JOIN students st ON st.name=es.student_name
		LEFT JOIN classes c ON c.id=st.class_id
		WHERE ` + condition
	_ = s.db.QueryRowContext(ctx, query, arg).Scan(&stats.Highest, &stats.Lowest, &stats.Average, &stats.Total)
	if stats.Total > 0 {
		rankQuery := `
			SELECT COUNT(*) + 1
			FROM (
				SELECT es.student_name, MAX(es.score) AS score
				FROM exam_scores es
				LEFT JOIN students st ON st.name=es.student_name
				LEFT JOIN classes c ON c.id=st.class_id
				WHERE ` + condition + `
				GROUP BY es.student_name
			) ranked
			WHERE ranked.score > COALESCE((SELECT score FROM exam_scores WHERE student_name=? ORDER BY exam_at DESC LIMIT 1),0)`
		_ = s.db.QueryRowContext(ctx, rankQuery, arg, studentName).Scan(&stats.Rank)
	}
	return stats
}

func summarizePortalHomework(items []PortalHomework) PortalHomeworkSummary {
	summary := PortalHomeworkSummary{Total: len(items)}
	for _, item := range items {
		switch item.Status {
		case "graded", "completed", "uploaded", "reviewed":
			summary.Completed++
		case "overdue", "missing":
			summary.Overdue++
			summary.Pending++
		default:
			summary.Pending++
		}
	}
	if summary.Total > 0 {
		summary.Completion = int(float64(summary.Completed) / float64(summary.Total) * 100)
	}
	summary.NeedsAttention = summary.Pending + summary.Overdue
	return summary
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
		subject := portalSubjectFromWrongQuestion(item)
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

func portalSubjectFromWrongQuestion(item WrongQuestion) string {
	text := item.SourcePaper + item.KnowledgePoint
	switch {
	case strings.Contains(text, "语文"):
		return "语文"
	case strings.Contains(text, "英语"):
		return "英语"
	case strings.Contains(text, "数学"), strings.Contains(text, "分数"), strings.Contains(text, "比例"), strings.Contains(text, "几何"):
		return "数学"
	default:
		return "综合"
	}
}

func portalWeakPoints(items []WrongQuestion) []string {
	counts := map[string]int{}
	for _, item := range items {
		if item.KnowledgePoint != "" {
			counts[item.KnowledgePoint]++
		}
	}
	keys := []string{}
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] == counts[keys[j]] {
			return keys[i] < keys[j]
		}
		return counts[keys[i]] > counts[keys[j]]
	})
	if len(keys) > 5 {
		keys = keys[:5]
	}
	return keys
}

func portalAICapabilities() []AICapability {
	return []AICapability{
		{Key: "analysis", Name: "AI 学情分析", Status: "planned", Description: "多维分析学科与知识点短板，输出补漏地图", CTA: "登记分析意向", PriceLabel: "后续付费", ValuePoint: "定位问题"},
		{Key: "ladder", Name: "天梯攻略", Status: "planned", Description: "根据补漏地图生成阶段练习册并周期复核", CTA: "登记提升计划", PriceLabel: "后续付费", ValuePoint: "持续复核"},
	}
}

func portalOffers() []PortalOffer {
	return []PortalOffer{
		{Key: "analysis", Name: "AI 学情深度分析", Description: "把成绩、作业和错题整理成知识点短板与掌握程度，生成可解释的补漏地图。", CTA: "预约开通", PriceLabel: "即将开放"},
		{Key: "ladder", Name: "天梯提升攻略", Description: "围绕薄弱知识点生成阶段练习册，练完复核，再规划下一轮提升路径。", CTA: "登记购买意向", PriceLabel: "即将开放"},
	}
}

func studentPortalFixture(id, name string) StudentPortalResponse {
	homework := []PortalHomework{{ID: "assign_001", Title: "六年级数学期中卷", Subject: "数学", Status: "graded", DueAt: "2026-06-23 18:00"}, {ID: "assign_002", Title: "分数应用题专项", Subject: "数学", Status: "pending", DueAt: "2026-06-25 18:00"}}
	scoreTrend := []PortalScorePoint{{Label: "单元测验一", Score: 76}, {Label: "月考", Score: 81}, {Label: "期中考试", Score: 85}}
	personalScore := 85.0
	if id == "stu_002" || name == "李四" {
		homework = []PortalHomework{
			{ID: "assign_003", Title: "语文阅读订正", Subject: "语文", Status: "pending", DueAt: "2026-07-03 18:00"},
			{ID: "assign_004", Title: "英语短文表达", Subject: "英语", Status: "overdue", DueAt: "2026-07-01 18:00"},
			{ID: "assign_005", Title: "数学比例专项", Subject: "数学", Status: "graded", DueAt: "2026-06-30 18:00"},
		}
		scoreTrend = []PortalScorePoint{{Label: "单元测验一", Score: 82}, {Label: "月考", Score: 79}, {Label: "期中考试", Score: 78}}
		personalScore = 78
	}
	wrong := []WrongQuestion{}
	for _, item := range wrongQuestionsFixture() {
		if item.StudentName == name {
			wrong = append(wrong, item)
		}
	}
	return StudentPortalResponse{
		StudentID: id, StudentName: name, GradeName: "六年级", ClassName: "六年级 3 班",
		ScoreSummary:    PortalScoreSummary{GradeName: "六年级", ClassName: "六年级 3 班", Personal: personalScore},
		HomeworkSummary: summarizePortalHomework(homework),
		Homework:        homework,
		ScoreTrend:      scoreTrend,
		Mistakes:        groupPortalMistakes(wrong),
		WeakPoints:      portalWeakPoints(wrong),
		AI:              portalAICapabilities(),
		Offers:          portalOffers(),
	}
}
