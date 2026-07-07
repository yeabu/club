package app

import "strings"

type DashboardResponse struct {
	Source        string          `json:"source,omitempty"`
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
	ID              string     `json:"id"`
	ScanType        string     `json:"scanType"`
	Title           string     `json:"title"`
	ClassName       string     `json:"className"`
	TemplateID      string     `json:"templateId,omitempty"`
	TemplateVersion int        `json:"templateVersion,omitempty"`
	Pages           int        `json:"pages"`
	Notes           string     `json:"notes,omitempty"`
	Status          string     `json:"status"`
	Progress        int        `json:"progress"`
	FailureReason   string     `json:"failureReason,omitempty"`
	RetryCount      int        `json:"retryCount"`
	QueueStatus     string     `json:"queueStatus,omitempty"`
	QueueMessage    string     `json:"queueMessage,omitempty"`
	Files           []ScanFile `json:"files,omitempty"`
}

type ScanFile struct {
	Key           string `json:"key"`
	FileName      string `json:"fileName"`
	ContentType   string `json:"contentType"`
	Size          int64  `json:"size"`
	URL           string `json:"url"`
	Page          int    `json:"page,omitempty"`
	Status        string `json:"status,omitempty"`
	FailureReason string `json:"failureReason,omitempty"`
	StudentID     string `json:"studentId,omitempty"`
	StudentName   string `json:"studentName,omitempty"`
	MatchStatus   string `json:"matchStatus,omitempty"`
	MatchMethod   string `json:"matchMethod,omitempty"`
}

type ScanFileStatusRequest struct {
	FileKey       string `json:"fileKey"`
	Status        string `json:"status"`
	FailureReason string `json:"failureReason"`
	StudentID     string `json:"studentId"`
	StudentName   string `json:"studentName"`
	MatchStatus   string `json:"matchStatus"`
}

type ScanException struct {
	TaskID      string `json:"taskId"`
	Title       string `json:"title"`
	ClassName   string `json:"className"`
	FileKey     string `json:"fileKey,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	Page        int    `json:"page,omitempty"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	StudentID   string `json:"studentId,omitempty"`
	StudentName string `json:"studentName,omitempty"`
	MatchStatus string `json:"matchStatus,omitempty"`
	QueueStatus string `json:"queueStatus,omitempty"`
}

type ScanExceptionListResponse struct {
	Items []ScanException `json:"items"`
}

type ScanUploadResponse struct {
	Files []ScanFile `json:"files"`
}

type ScanTaskRequest struct {
	AssignmentID    string     `json:"assignmentId"`
	ExamID          string     `json:"examId"`
	ScanType        string     `json:"scanType"`
	Title           string     `json:"title"`
	ClassName       string     `json:"className"`
	TemplateID      string     `json:"templateId"`
	TemplateVersion int        `json:"templateVersion"`
	Pages           int        `json:"pages"`
	Notes           string     `json:"notes"`
	Files           []ScanFile `json:"files"`
}

type ScanTaskResponse struct {
	Status     string  `json:"status"`
	QueueID    string  `json:"queueId,omitempty"`
	QueueError string  `json:"queueError,omitempty"`
	Task       ScanJob `json:"task"`
}

type ScanTaskListResponse struct {
	Tasks []ScanJob `json:"tasks"`
}

type ScanTaskStatusRequest struct {
	Status        string `json:"status"`
	Progress      int    `json:"progress"`
	FailureReason string `json:"failureReason"`
	RetryCount    int    `json:"retryCount"`
}

type ScanWorkerResultRequest struct {
	Status        string         `json:"status"`
	Progress      int            `json:"progress"`
	FailureReason string         `json:"failureReason"`
	ModelVersion  string         `json:"modelVersion"`
	Result        map[string]any `json:"result"`
}

type ScanWorkerResultResponse struct {
	Status string                  `json:"status"`
	Task   ScanJob                 `json:"task"`
	Result *ScanWorkerResultRecord `json:"result,omitempty"`
}

type ScanWorkerResultRecord struct {
	TaskID        string         `json:"taskId"`
	Status        string         `json:"status"`
	Progress      int            `json:"progress"`
	FailureReason string         `json:"failureReason,omitempty"`
	ModelVersion  string         `json:"modelVersion,omitempty"`
	Result        map[string]any `json:"result"`
}

type ScanTaskRetryRequest struct {
	FileKey string `json:"fileKey"`
}

type ScanFileMatchRequest struct {
	FileKey     string `json:"fileKey"`
	StudentID   string `json:"studentId"`
	StudentName string `json:"studentName"`
	MatchMethod string `json:"matchMethod"`
}

type ScanTaskPreviewResponse struct {
	Task  ScanJob    `json:"task"`
	Files []ScanFile `json:"files"`
}

type ScanQueuePayload struct {
	TaskID          string     `json:"taskId"`
	ScanType        string     `json:"scanType"`
	Title           string     `json:"title"`
	ClassName       string     `json:"className"`
	TemplateID      string     `json:"templateId"`
	TemplateVersion int        `json:"templateVersion"`
	Pages           int        `json:"pages"`
	FileKeys        []string   `json:"fileKeys"`
	Files           []ScanFile `json:"files"`
	RetryCount      int        `json:"retryCount"`
	CreatedAt       string     `json:"createdAt"`
}

func normalizeScanType(value string) string {
	switch strings.TrimSpace(value) {
	case "paper":
		return "paper"
	case "answer_sheet":
		return "answer_sheet"
	default:
		return "answer_sheet"
	}
}

type TemplateAISuggestionRequest struct {
	SourceFileURL string `json:"sourceFileUrl"`
	PaperName     string `json:"paperName"`
	Subject       string `json:"subject,omitempty"`
	Grade         string `json:"grade,omitempty"`
	TargetScore   int    `json:"targetScore,omitempty"`
}

type TemplateAISuggestionResponse struct {
	PaperName          string             `json:"paperName"`
	QuestionCount      int                `json:"questionCount"`
	TotalScore         int                `json:"totalScore"`
	SuggestedQuestions []QuestionTemplate `json:"suggestedQuestions"`
	ReviewRequired     bool               `json:"reviewRequired"`
	Source             string             `json:"source"`
	Provider           string             `json:"provider,omitempty"`
	ProviderError      string             `json:"providerError,omitempty"`
}

type PaperManagementItem struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	ClassName       string         `json:"className"`
	Pages           int            `json:"pages"`
	Status          string         `json:"status"`
	Progress        int            `json:"progress"`
	SourceFileURL   string         `json:"sourceFileUrl"`
	SourceFileKey   string         `json:"sourceFileKey"`
	FileName        string         `json:"fileName"`
	ImportedAt      string         `json:"importedAt"`
	Template        *PaperTemplate `json:"template,omitempty"`
	TemplateID      string         `json:"templateId,omitempty"`
	TemplateVersion int            `json:"templateVersion,omitempty"`
}

type PaperManagementResponse struct {
	Items []PaperManagementItem `json:"items"`
}

type PaperDeleteResponse struct {
	Status      string   `json:"status"`
	DeletedID   string   `json:"deletedId"`
	DeletedKeys []string `json:"deletedKeys"`
}

type AIProviderSetting struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	DisplayName            string `json:"displayName"`
	BaseURL                string `json:"baseUrl"`
	Model                  string `json:"model"`
	TimeoutSeconds         int    `json:"timeoutSeconds"`
	APIKey                 string `json:"-"`
	APIKeyProvided         bool   `json:"apiKeyProvided"`
	CallbackSecret         string `json:"-"`
	CallbackSecretProvided bool   `json:"callbackSecretProvided"`
	Active                 bool   `json:"active"`
	Configured             bool   `json:"configured"`
	UpdatedAt              string `json:"updatedAt,omitempty"`
}

type AIProviderSettingRequest struct {
	Name            string `json:"name"`
	DisplayName     string `json:"displayName"`
	BaseURL         string `json:"baseUrl"`
	Model           string `json:"model"`
	APIKey          string `json:"apiKey"`
	TimeoutSeconds  int    `json:"timeoutSeconds"`
	CallbackSecret  string `json:"callbackSecret"`
	KeepExistingKey bool   `json:"keepExistingKey"`
}

type AIProviderSettingsResponse struct {
	Current  AIProviderSetting   `json:"current"`
	Channels []AIProviderSetting `json:"channels"`
}

type SystemConfig struct {
	Storage SystemStorageConfig `json:"storage"`
	Roles   []SystemRoleConfig  `json:"roles"`
	VIP     []SystemVIPConfig   `json:"vip"`
}

type SystemStorageConfig struct {
	Driver             string `json:"driver"`
	Endpoint           string `json:"endpoint"`
	Bucket             string `json:"bucket"`
	Region             string `json:"region"`
	PublicBaseURL      string `json:"publicBaseUrl"`
	MaxUploadMB        int    `json:"maxUploadMb"`
	AccessKeyProvided  bool   `json:"accessKeyProvided"`
	SecretKeyProvided  bool   `json:"secretKeyProvided"`
	AccessKey          string `json:"accessKey,omitempty"`
	SecretKey          string `json:"secretKey,omitempty"`
	KeepExistingSecret bool   `json:"keepExistingSecret,omitempty"`
}

type SystemRoleConfig struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Enabled     bool     `json:"enabled"`
}

type SystemVIPConfig struct {
	Level          string `json:"level"`
	Name           string `json:"name"`
	TokenQuota     int    `json:"tokenQuota"`
	StorageQuotaGB int    `json:"storageQuotaGb"`
	Enabled        bool   `json:"enabled"`
}

type ReviewItem struct {
	ID          string `json:"id"`
	StudentName string `json:"studentName"`
	ClassName   string `json:"className,omitempty"`
	PaperName   string `json:"paperName"`
	QuestionNo  string `json:"questionNo"`
	AIAdvice    string `json:"aiAdvice"`
	Confidence  int    `json:"confidence"`
	Status      string `json:"status,omitempty"`
	ReviewStage string `json:"reviewStage,omitempty"`
}

type ReviewQueueResponse struct {
	Items []ReviewItem `json:"items"`
}

type KnowledgeStat struct {
	Name       string `json:"name"`
	Accuracy   int    `json:"accuracy"`
	WrongCount int    `json:"wrongCount"`
}

type WrongQuestion struct {
	ID               int64    `json:"id"`
	StudentID        string   `json:"studentId"`
	StudentName      string   `json:"studentName"`
	ClassName        string   `json:"className"`
	SubmissionID     string   `json:"submissionId"`
	QuestionID       string   `json:"questionId"`
	QuestionNo       string   `json:"questionNo"`
	QuestionType     string   `json:"questionType"`
	KnowledgePoint   string   `json:"knowledgePoint"`
	ErrorType        string   `json:"errorType"`
	WrongReason      string   `json:"wrongReason"`
	SourcePaper      string   `json:"sourcePaper"`
	OriginalQuestion string   `json:"originalQuestion"`
	Score            float64  `json:"score"`
	MaxScore         float64  `json:"maxScore"`
	CorrectAnswer    string   `json:"correctAnswer"`
	StudentAnswer    string   `json:"studentAnswer"`
	AnswerImageURL   string   `json:"answerImageUrl"`
	Explanation      string   `json:"explanation"`
	CorrectionStatus string   `json:"correctionStatus"`
	RepracticeStatus string   `json:"repracticeStatus"`
	CreatedAt        string   `json:"createdAt"`
	Knowledge        []string `json:"knowledge"`
}

type WrongQuestionListResponse struct {
	Items []WrongQuestion `json:"items"`
}

type WrongQuestionFilters struct {
	Paper       string
	ClassName   string
	StudentName string
	Knowledge   string
	ErrorType   string
	Search      string
}

type RepracticeTaskRequest struct {
	WrongQuestionIDs []int64 `json:"wrongQuestionIds"`
	Title            string  `json:"title"`
	DueAt            string  `json:"dueAt"`
}

type RepracticeTaskResponse struct {
	Status      string   `json:"status"`
	TaskID      string   `json:"taskId"`
	LinkedCount int      `json:"linkedCount"`
	Knowledge   []string `json:"knowledge"`
}

type RepracticeSubmissionRequest struct {
	StudentID string                       `json:"studentId"`
	Answers   []RepracticeSubmissionAnswer `json:"answers"`
}

type RepracticeSubmissionAnswer struct {
	WrongQuestionID int64  `json:"wrongQuestionId"`
	Answer          string `json:"answer"`
	Status          string `json:"status"`
}

type RepracticeSubmissionResponse struct {
	Status    string `json:"status"`
	TaskID    string `json:"taskId"`
	Submitted int    `json:"submitted"`
}

type KnowledgeMastery struct {
	Name            string `json:"name"`
	Mastery         int    `json:"mastery"`
	PreviousMastery int    `json:"previousMastery"`
	Trend           int    `json:"trend"`
	WrongCount      int    `json:"wrongCount"`
	StudentCount    int    `json:"studentCount"`
}

type LearningProfileResponse struct {
	ClassName        string             `json:"className"`
	KnowledgeMastery []KnowledgeMastery `json:"knowledgeMastery"`
	StudentRisks     []StudentRisk      `json:"studentRisks"`
	HomeworkWatch    []HomeworkWatch    `json:"homeworkWatch"`
}

type GuardianReportResponse struct {
	StudentName string   `json:"studentName"`
	ClassName   string   `json:"className"`
	Summary     string   `json:"summary"`
	Score       float64  `json:"score"`
	WrongCount  int      `json:"wrongCount"`
	Weakness    []string `json:"weakness"`
	Actions     []string `json:"actions"`
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
	ActorName    string  `json:"actorName"`
	ReviewStage  string  `json:"reviewStage"`
	ModelVersion string  `json:"modelVersion"`
}

type GradingDecisionResponse struct {
	Status       string                     `json:"status"`
	FinalScore   float64                    `json:"finalScore"`
	NextQuestion string                     `json:"nextQuestion"`
	NextReview   *SubjectiveGradingResponse `json:"nextReview,omitempty"`
}

type GradingHistoryResponse struct {
	Items []GradingHistoryItem `json:"items"`
}

type GradingHistoryItem struct {
	ID           int64   `json:"id"`
	SubmissionID string  `json:"submissionId"`
	QuestionID   string  `json:"questionId"`
	Action       string  `json:"action"`
	Score        float64 `json:"score"`
	Note         string  `json:"note"`
	ActorName    string  `json:"actorName"`
	ReviewStage  string  `json:"reviewStage"`
	ModelVersion string  `json:"modelVersion"`
	CreatedAt    string  `json:"createdAt"`
}

type TemplateMutationResponse struct {
	Status   string        `json:"status"`
	Template PaperTemplate `json:"template"`
}

type TemplateRegionMutationResponse struct {
	Status   string           `json:"status"`
	Question QuestionTemplate `json:"question"`
	Template PaperTemplate    `json:"template"`
}

type TemplateRegionsRequest struct {
	Questions []QuestionTemplate `json:"questions"`
}

type TemplateStatusRequest struct {
	Status string `json:"status"`
}

type TemplateValidationIssue struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	QuestionID string `json:"questionId,omitempty"`
	QuestionNo string `json:"questionNo,omitempty"`
	Page       int    `json:"page,omitempty"`
}

type TemplateValidationResponse struct {
	TemplateID string                    `json:"templateId"`
	Valid      bool                      `json:"valid"`
	Issues     []TemplateValidationIssue `json:"issues"`
}

type TemplateDiffResponse struct {
	BaseTemplateID   string               `json:"baseTemplateId"`
	TargetTemplateID string               `json:"targetTemplateId"`
	AddedQuestions   []QuestionTemplate   `json:"addedQuestions"`
	RemovedQuestions []QuestionTemplate   `json:"removedQuestions"`
	ChangedQuestions []TemplateDiffChange `json:"changedQuestions"`
	Summary          []string             `json:"summary"`
}

type TemplateDiffChange struct {
	QuestionNo string   `json:"questionNo"`
	Fields     []string `json:"fields"`
}

type PaperTemplate struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Subject       string             `json:"subject"`
	Grade         string             `json:"grade"`
	QuestionCount int                `json:"questionCount"`
	TotalScore    int                `json:"totalScore"`
	SourceFileURL string             `json:"sourceFileUrl"`
	Status        string             `json:"status"`
	Version       int                `json:"version"`
	ParentID      string             `json:"parentId"`
	Questions     []QuestionTemplate `json:"questions"`
}

type QuestionTemplate struct {
	ID             string   `json:"id"`
	No             string   `json:"no"`
	Type           string   `json:"type"`
	Score          float64  `json:"score"`
	StandardAnswer string   `json:"standardAnswer"`
	ScoringRules   []string `json:"scoringRules"`
	Knowledge      []string `json:"knowledge"`
	Region         Region   `json:"region"`
}

type Region struct {
	Page   int `json:"page"`
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ClassroomAnalytics struct {
	ClassName           string                     `json:"className"`
	AverageScore        float64                    `json:"averageScore"`
	HighestScore        float64                    `json:"highestScore"`
	LowestScore         float64                    `json:"lowestScore"`
	StudentCount        int                        `json:"studentCount"`
	GradedCount         int                        `json:"gradedCount"`
	CompletionRate      int                        `json:"completionRate"`
	PassRate            int                        `json:"passRate"`
	ExcellentRate       int                        `json:"excellentRate"`
	QuestionStats       []QuestionStat             `json:"questionStats"`
	QuestionDetails     []QuestionDetailStat       `json:"questionDetails"`
	KnowledgeStats      []KnowledgeStat            `json:"knowledgeStats"`
	StudentRisks        []StudentRisk              `json:"studentRisks"`
	StudentScores       []StudentScoreSummary      `json:"studentScores"`
	ScoreBands          []ScoreBand                `json:"scoreBands"`
	ObjectiveExceptions []ObjectiveReviewException `json:"objectiveExceptions"`
}

type ScoreBand struct {
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Count int    `json:"count"`
}

type QuestionDetailStat struct {
	No             string `json:"no"`
	Type           string `json:"type"`
	Accuracy       int    `json:"accuracy"`
	ScoreRate      int    `json:"scoreRate"`
	Difficulty     string `json:"difficulty"`
	Discrimination int    `json:"discrimination"`
	TypicalError   string `json:"typicalError"`
}

type StudentScoreSummary struct {
	StudentName string   `json:"studentName"`
	ClassName   string   `json:"className"`
	Score       float64  `json:"score"`
	Rank        int      `json:"rank"`
	Weakness    []string `json:"weakness"`
}

type ObjectiveReviewException struct {
	ID             int64   `json:"id"`
	SubmissionID   string  `json:"submissionId"`
	StudentName    string  `json:"studentName"`
	QuestionID     string  `json:"questionId"`
	QuestionNo     string  `json:"questionNo"`
	Answer         string  `json:"answer"`
	Confidence     int     `json:"confidence"`
	Reason         string  `json:"reason"`
	Status         string  `json:"status"`
	SuggestedScore float64 `json:"suggestedScore"`
}

type ObjectiveReviewDecisionRequest struct {
	Score       float64 `json:"score"`
	Decision    string  `json:"decision"`
	TeacherNote string  `json:"teacherNote"`
	ActorName   string  `json:"actorName"`
}

type ObjectiveReviewDecisionResponse struct {
	Status    string                   `json:"status"`
	Exception ObjectiveReviewException `json:"exception"`
}

type ScoreGenerationResponse struct {
	Status    string `json:"status"`
	ExamID    string `json:"examId,omitempty"`
	ClassName string `json:"className"`
	Generated int    `json:"generated"`
	Blocked   bool   `json:"blocked,omitempty"`
	Reason    string `json:"reason,omitempty"`
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

type AuthContext struct {
	UserID     string   `json:"userId"`
	SchoolID   string   `json:"schoolId,omitempty"`
	RoleNames  []string `json:"roleNames"`
	StudentID  string   `json:"studentId,omitempty"`
	GuardianID string   `json:"guardianId,omitempty"`
	TeacherID  string   `json:"teacherId,omitempty"`
}

type AuthLoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type AuthLoginResponse struct {
	Status      string            `json:"status"`
	User        AuthContext       `json:"user"`
	AuthHeaders map[string]string `json:"authHeaders"`
}

type ExamPublishRequest struct {
	Name          string             `json:"name"`
	SchoolID      string             `json:"schoolId"`
	Subject       string             `json:"subject"`
	Grade         string             `json:"grade"`
	ClassID       string             `json:"classId"`
	TeacherID     string             `json:"teacherId"`
	TemplateID    string             `json:"templateId"`
	MaxScore      float64            `json:"maxScore"`
	StartedAt     string             `json:"startedAt"`
	EndedAt       string             `json:"endedAt"`
	StudentIDs    []string           `json:"studentIds"`
	GradingPolicy *AutoGradingPolicy `json:"gradingPolicy,omitempty"`
}

type ExamRecord struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	SchoolID        string            `json:"schoolId"`
	Subject         string            `json:"subject"`
	Grade           string            `json:"grade"`
	ClassID         string            `json:"classId"`
	ClassName       string            `json:"className"`
	TeacherID       string            `json:"teacherId"`
	TemplateID      string            `json:"templateId"`
	TemplateVersion int               `json:"templateVersion"`
	Status          string            `json:"status"`
	MaxScore        float64           `json:"maxScore"`
	RosterLocked    bool              `json:"rosterLocked"`
	Students        []ExamStudent     `json:"students,omitempty"`
	GradingPolicy   AutoGradingPolicy `json:"gradingPolicy"`
}

type ExamStudent struct {
	StudentID string `json:"studentId"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

type ExamAttendanceRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type AutoGradingPolicy struct {
	ObjectiveMinConfidence  int     `json:"objectiveMinConfidence"`
	SubjectiveMinConfidence int     `json:"subjectiveMinConfidence"`
	SubjectiveMaxScoreShare float64 `json:"subjectiveMaxScoreShare"`
	SamplingRate            int     `json:"samplingRate"`
	RequireReason           bool    `json:"requireReason"`
	AbnormalFallback        bool    `json:"abnormalFallback"`
}
