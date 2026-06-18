package app

type DashboardResponse struct {
	Metrics       []Metric        `json:"metrics"`
	ScanQueue     []ScanJob       `json:"scanQueue"`
	ReviewQueue   []ReviewItem    `json:"reviewQueue"`
	WeakPoints    []KnowledgeStat `json:"weakPoints"`
	HomeworkWatch []HomeworkWatch `json:"homeworkWatch"`
}

type Metric struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Delta string `json:"delta"`
	Tone  string `json:"tone"`
}

type ScanJob struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	ClassName string `json:"className"`
	Pages     int    `json:"pages"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
}

type ReviewItem struct {
	ID          string `json:"id"`
	StudentName string `json:"studentName"`
	PaperName   string `json:"paperName"`
	QuestionNo  string `json:"questionNo"`
	AIAdvice    string `json:"aiAdvice"`
	Confidence  int    `json:"confidence"`
}

type KnowledgeStat struct {
	Name       string `json:"name"`
	Accuracy   int    `json:"accuracy"`
	WrongCount int    `json:"wrongCount"`
}

type HomeworkWatch struct {
	StudentName string `json:"studentName"`
	ClassName   string `json:"className"`
	Missing     int    `json:"missing"`
	Guardian    string `json:"guardian"`
}

type SubjectiveGradingResponse struct {
	ReviewID       string         `json:"reviewId"`
	SubmissionID   string         `json:"submissionId"`
	QuestionID     string         `json:"questionId"`
	PaperName      string         `json:"paperName"`
	StudentName    string         `json:"studentName"`
	ClassName      string         `json:"className"`
	QuestionNo     string         `json:"questionNo"`
	FullScore      float64        `json:"fullScore"`
	StandardAnswer StandardAnswer `json:"standardAnswer"`
	StudentAnswer  StudentAnswer  `json:"studentAnswer"`
	AI             AIAdvice       `json:"ai"`
}

type StandardAnswer struct {
	Content      string   `json:"content"`
	ScoringRules []string `json:"scoringRules"`
	Knowledge    []string `json:"knowledge"`
}

type StudentAnswer struct {
	OCRText  string `json:"ocrText"`
	ImageURL string `json:"imageUrl"`
}

type AIAdvice struct {
	Score      float64  `json:"score"`
	Reason     string   `json:"reason"`
	Comments   []string `json:"comments"`
	Confidence int      `json:"confidence"`
}

type GradingDecisionRequest struct {
	SubmissionID string  `json:"submissionId"`
	QuestionID   string  `json:"questionId"`
	FinalScore   float64 `json:"finalScore"`
	Decision     string  `json:"decision"`
	TeacherNote  string  `json:"teacherNote"`
}

type GradingDecisionResponse struct {
	Status       string                     `json:"status"`
	FinalScore   float64                    `json:"finalScore"`
	NextQuestion string                     `json:"nextQuestion"`
	NextReview   *SubjectiveGradingResponse `json:"nextReview,omitempty"`
}

type PaperTemplate struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Subject       string             `json:"subject"`
	Grade         string             `json:"grade"`
	QuestionCount int                `json:"questionCount"`
	TotalScore    int                `json:"totalScore"`
	Questions     []QuestionTemplate `json:"questions"`
}

type QuestionTemplate struct {
	ID        string   `json:"id"`
	No        string   `json:"no"`
	Type      string   `json:"type"`
	Score     float64  `json:"score"`
	Knowledge []string `json:"knowledge"`
	Region    Region   `json:"region"`
}

type Region struct {
	Page   int `json:"page"`
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ClassroomAnalytics struct {
	ClassName      string          `json:"className"`
	AverageScore   float64         `json:"averageScore"`
	HighestScore   float64         `json:"highestScore"`
	LowestScore    float64         `json:"lowestScore"`
	QuestionStats  []QuestionStat  `json:"questionStats"`
	KnowledgeStats []KnowledgeStat `json:"knowledgeStats"`
	StudentRisks   []StudentRisk   `json:"studentRisks"`
}

type QuestionStat struct {
	No       string `json:"no"`
	Accuracy int    `json:"accuracy"`
	Type     string `json:"type"`
}

type StudentRisk struct {
	StudentName string   `json:"studentName"`
	Risk        string   `json:"risk"`
	Weakness    []string `json:"weakness"`
}
