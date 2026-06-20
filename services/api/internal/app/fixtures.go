package app

func dashboardFixture() DashboardResponse {
	return DashboardResponse{
		Source: "fixtures",
		Metrics: []Metric{
			{Label: "待批试卷", Value: "128", Delta: "较昨日 +24", Tone: "primary"},
			{Label: "主观题待复核", Value: "36", Delta: "AI 已预评分", Tone: "warning"},
			{Label: "未提交作业", Value: "8", Delta: "3 人连续未交", Tone: "danger"},
			{Label: "班级平均分", Value: "81.6", Delta: "较上次 +3.2", Tone: "success"},
		},
		ScanQueue: []ScanJob{
			{ID: "scan_001", Title: "六年级数学期中卷", ClassName: "六年级 3 班", TemplateID: "tpl_001", TemplateVersion: 1, Pages: 96, Status: "OCR 识别中", Progress: 68, RetryCount: 0, QueueStatus: "queued", Files: []ScanFile{
				{Key: "mock/scan_001/zhangsan.png", FileName: "张三-第1页.png", ContentType: "image/png", Size: 204800, URL: "/mock/student-answer-q15.png", Page: 1, Status: "识别中", StudentID: "stu_001", StudentName: "张三", MatchStatus: "matched", MatchMethod: "name"},
			}},
			{ID: "scan_002", Title: "分数应用题专项", ClassName: "六年级 1 班", TemplateID: "tpl_001", TemplateVersion: 1, Pages: 42, Status: "等待 OMR", Progress: 32, RetryCount: 0, QueueStatus: "queued", Files: []ScanFile{
				{Key: "mock/scan_002/unmatched.png", FileName: "未匹配-第1页.png", ContentType: "image/png", Size: 178200, URL: "/mock/student-answer-q18.png", Page: 1, Status: "等待 OMR", MatchStatus: "pending"},
			}},
			{ID: "scan_003", Title: "几何面积小测", ClassName: "五年级 2 班", TemplateID: "tpl_001", TemplateVersion: 1, Pages: 48, Status: "待导入", Progress: 0, FailureReason: "OCR Worker 暂未消费", RetryCount: 1, QueueStatus: "failed", Files: []ScanFile{
				{Key: "mock/scan_003/error.png", FileName: "赵六-第1页.png", ContentType: "image/png", Size: 192000, URL: "/mock/student-answer-q18.png", Page: 1, Status: "失败", FailureReason: "识别超时", MatchStatus: "pending"},
			}},
		},
		ReviewQueue: []ReviewItem{
			{ID: "review_001", StudentName: "张三", PaperName: "六年级数学期中卷", QuestionNo: "15", AIAdvice: "8 / 10", Confidence: 86},
			{ID: "review_002", StudentName: "李四", PaperName: "六年级数学期中卷", QuestionNo: "18", AIAdvice: "6 / 8", Confidence: 78},
			{ID: "review_003", StudentName: "王五", PaperName: "分数应用题专项", QuestionNo: "7", AIAdvice: "4 / 6", Confidence: 74},
		},
		WeakPoints: []KnowledgeStat{
			{Name: "分数应用题", Accuracy: 42, WrongCount: 29},
			{Name: "几何面积", Accuracy: 51, WrongCount: 21},
			{Name: "比例换算", Accuracy: 64, WrongCount: 15},
		},
		HomeworkWatch: []HomeworkWatch{
			{StudentName: "李四", ClassName: "六年级 3 班", Missing: 3, Guardian: "李四家长"},
			{StudentName: "赵六", ClassName: "六年级 3 班", Missing: 2, Guardian: "赵六家长"},
		},
	}
}

func subjectiveFixture() SubjectiveGradingResponse {
	return SubjectiveGradingResponse{
		ReviewID:     "review_001",
		SubmissionID: "sub_001",
		QuestionID:   "q_015",
		PaperName:    "六年级数学期中卷",
		StudentName:  "张三",
		ClassName:    "六年级 3 班",
		QuestionNo:   "15",
		FullScore:    10,
		StandardAnswer: StandardAnswer{
			Content: "先设未知数 x，列出比例关系 3:5 = x:40，解得 x = 24。答：需要 24 千克。",
			ScoringRules: []string{
				"正确设未知数 2 分",
				"列出比例关系 4 分",
				"计算过程正确 2 分",
				"结果与答语完整 2 分",
			},
			Knowledge: []string{"比例", "应用题建模", "方程求解"},
		},
		StudentAnswer: StudentAnswer{
			OCRText:  "设需要 x 千克，3/5 = x/40，5x = 120，x = 24。答需要 24 千克。",
			ImageURL: "/mock/student-answer-q15.png",
		},
		AI: AIAdvice{
			Score:      8,
			Reason:     "建模和计算结果正确，但比例式书写不够规范，缺少单位换算说明。",
			Comments:   []string{"核心步骤完整", "建议扣除书写规范 1 分", "答语完整，可保留 1 分"},
			Confidence: 86,
		},
	}
}

func templatesFixture() []PaperTemplate {
	return []PaperTemplate{
		{
			ID:            "tpl_001",
			Name:          "六年级数学期中卷",
			Subject:       "数学",
			Grade:         "六年级",
			QuestionCount: 25,
			TotalScore:    100,
			SourceFileURL: "/mock/templates/tpl_001-blank-paper.pdf",
			Status:        "published",
			Version:       1,
			Questions: []QuestionTemplate{
				{ID: "q_001", No: "1", Type: "single_choice", Score: 2, StandardAnswer: "A", ScoringRules: []string{"选对 A 得 2 分"}, Knowledge: []string{"分数"}, Region: Region{Page: 1, X: 120, Y: 260, Width: 480, Height: 80}},
				{ID: "q_015", No: "15", Type: "subjective", Score: 10, StandardAnswer: "先设未知数 x，列出比例关系 3:5 = x:40，解得 x = 24。答：需要 24 千克。", ScoringRules: []string{"正确设未知数 2 分", "列出比例关系 4 分", "计算过程正确 2 分", "结果与答语完整 2 分"}, Knowledge: []string{"比例", "应用题建模"}, Region: Region{Page: 2, X: 96, Y: 420, Width: 620, Height: 180}},
			},
		},
		{
			ID:            "tpl_draft_001",
			Name:          "六年级数学期中卷 v2",
			Subject:       "数学",
			Grade:         "六年级",
			QuestionCount: 2,
			TotalScore:    12,
			SourceFileURL: "/mock/templates/tpl_001-blank-paper.pdf",
			Status:        "draft",
			Version:       2,
			ParentID:      "tpl_001",
			Questions: []QuestionTemplate{
				{ID: "q_draft_001", No: "1", Type: "single_choice", Score: 2, StandardAnswer: "A", ScoringRules: []string{"选对得分"}, Knowledge: []string{"分数"}, Region: Region{Page: 1, X: 130, Y: 260, Width: 460, Height: 80}},
				{ID: "q_draft_015", No: "15", Type: "subjective", Score: 10, StandardAnswer: "按比例关系列式求解。", ScoringRules: []string{"列式 4 分", "计算 4 分", "答语 2 分"}, Knowledge: []string{"比例"}, Region: Region{Page: 2, X: 96, Y: 420, Width: 620, Height: 180}},
			},
		},
		{
			ID:            "tpl_disabled_001",
			Name:          "旧版几何面积小测",
			Subject:       "数学",
			Grade:         "五年级",
			QuestionCount: 1,
			TotalScore:    8,
			SourceFileURL: "/mock/templates/old-geometry-quiz.pdf",
			Status:        "disabled",
			Version:       1,
			Questions: []QuestionTemplate{
				{ID: "q_disabled_018", No: "18", Type: "subjective", Score: 8, StandardAnswer: "拆分图形后计算面积。", ScoringRules: []string{"拆分图形 2 分", "公式正确 2 分", "计算正确 4 分"}, Knowledge: []string{"几何面积"}, Region: Region{Page: 1, X: 110, Y: 640, Width: 600, Height: 160}},
			},
		},
	}
}

func templateAISuggestionFixture(template PaperTemplate, req TemplateAISuggestionRequest) TemplateAISuggestionResponse {
	paperName := req.PaperName
	if paperName == "" {
		paperName = template.Name
	}
	questions := []QuestionTemplate{
		{
			ID:             templateRecordID("ai_q", ""),
			No:             "1",
			Type:           "single_choice",
			Score:          2,
			StandardAnswer: "A",
			ScoringRules:   []string{"选对 A 得 2 分"},
			Knowledge:      []string{"分数"},
			Region:         Region{Page: 1, X: 120, Y: 260, Width: 480, Height: 80},
		},
		{
			ID:             templateRecordID("ai_q", ""),
			No:             "15",
			Type:           "subjective",
			Score:          10,
			StandardAnswer: "先设未知数并列比例关系，计算后写完整答语。",
			ScoringRules:   []string{"建模 2 分", "列式 4 分", "计算 2 分", "答语 2 分"},
			Knowledge:      []string{"比例", "应用题建模"},
			Region:         Region{Page: 2, X: 96, Y: 420, Width: 620, Height: 180},
		},
		{
			ID:             templateRecordID("ai_q", ""),
			No:             "18",
			Type:           "subjective",
			Score:          8,
			StandardAnswer: "拆分图形并根据面积公式计算。",
			ScoringRules:   []string{"图形拆分 2 分", "公式 2 分", "计算 3 分", "单位 1 分"},
			Knowledge:      []string{"几何面积"},
			Region:         Region{Page: 2, X: 110, Y: 640, Width: 600, Height: 160},
		},
	}
	total := 0
	for _, question := range questions {
		total += int(question.Score + 0.5)
	}
	return TemplateAISuggestionResponse{
		PaperName:          paperName,
		QuestionCount:      len(questions),
		TotalScore:         total,
		SuggestedQuestions: questions,
		ReviewRequired:     true,
		Source:             "worker-mock",
	}
}

func analyticsFixture() ClassroomAnalytics {
	return ClassroomAnalytics{
		ClassName:      "六年级 3 班",
		AverageScore:   81.6,
		HighestScore:   98,
		LowestScore:    54,
		StudentCount:   42,
		GradedCount:    40,
		CompletionRate: 95,
		PassRate:       88,
		ExcellentRate:  22,
		QuestionStats: []QuestionStat{
			{No: "1", Accuracy: 96, Type: "单选题"},
			{No: "8", Accuracy: 82, Type: "填空题"},
			{No: "15", Accuracy: 42, Type: "应用题"},
			{No: "18", Accuracy: 38, Type: "应用题"},
		},
		QuestionDetails: []QuestionDetailStat{
			{No: "1", Type: "单选题", Accuracy: 96, ScoreRate: 96, Difficulty: "容易", Discrimination: 52, TypicalError: "整体掌握较好，关注个别粗心"},
			{No: "8", Type: "填空题", Accuracy: 82, ScoreRate: 82, Difficulty: "容易", Discrimination: 56, TypicalError: "单位换算漏写"},
			{No: "15", Type: "应用题", Accuracy: 42, ScoreRate: 48, Difficulty: "偏难", Discrimination: 78, TypicalError: "比例关系建模不稳定"},
			{No: "18", Type: "应用题", Accuracy: 38, ScoreRate: 44, Difficulty: "偏难", Discrimination: 81, TypicalError: "图形拆分和公式迁移错误"},
		},
		KnowledgeStats: []KnowledgeStat{
			{Name: "分数应用题", Accuracy: 42, WrongCount: 29},
			{Name: "几何面积", Accuracy: 51, WrongCount: 21},
			{Name: "比例换算", Accuracy: 64, WrongCount: 15},
			{Name: "计算基础", Accuracy: 88, WrongCount: 6},
		},
		StudentRisks: []StudentRisk{
			{StudentName: "李四", Risk: "连续 3 次未提交作业", Weakness: []string{"分数应用题", "比例"}},
			{StudentName: "赵六", Risk: "本次低于班均 18 分", Weakness: []string{"几何面积"}},
		},
		StudentScores: []StudentScoreSummary{
			{StudentName: "赵六", ClassName: "六年级 3 班", Score: 88, Rank: 1, Weakness: []string{"几何面积"}},
			{StudentName: "张三", ClassName: "六年级 3 班", Score: 85, Rank: 2, Weakness: []string{"表达规范"}},
			{StudentName: "王五", ClassName: "六年级 3 班", Score: 82, Rank: 3, Weakness: []string{"计算基础"}},
			{StudentName: "李四", ClassName: "六年级 3 班", Score: 72, Rank: 4, Weakness: []string{"分数应用题", "比例"}},
		},
		ScoreBands: []ScoreBand{
			{Label: "0-59", Min: 0, Max: 59, Count: 1},
			{Label: "60-69", Min: 60, Max: 69, Count: 3},
			{Label: "70-79", Min: 70, Max: 79, Count: 10},
			{Label: "80-89", Min: 80, Max: 89, Count: 18},
			{Label: "90-100", Min: 90, Max: 100, Count: 8},
		},
		ObjectiveExceptions: []ObjectiveReviewException{
			{ID: 1, SubmissionID: "sub_002", StudentName: "李四", QuestionID: "q_001", QuestionNo: "1", Answer: "B", Confidence: 68, Reason: "低置信度且答案与标准答案不一致", Status: "pending", SuggestedScore: 0},
		},
	}
}

func wrongQuestionsFixture() []WrongQuestion {
	return []WrongQuestion{
		{ID: 1, StudentID: "stu_002", StudentName: "李四", ClassName: "六年级 3 班", SubmissionID: "sub_002", QuestionID: "q_001", QuestionNo: "1", QuestionType: "single_choice", KnowledgePoint: "分数", ErrorType: "concept", WrongReason: "标准答案为 A，学生选择 B", SourcePaper: "六年级数学期中卷", OriginalQuestion: "比较两个分数的大小，选择正确答案。", Score: 0, MaxScore: 2, CorrectAnswer: "A", StudentAnswer: "B", AnswerImageURL: "/mock/student-answer-q18.png", Explanation: "回顾分数大小比较方法。", CorrectionStatus: "pending", RepracticeStatus: "not_assigned", CreatedAt: "2026-06-20T09:00:00+08:00", Knowledge: []string{"分数"}},
		{ID: 2, StudentID: "stu_001", StudentName: "张三", ClassName: "六年级 3 班", SubmissionID: "sub_001", QuestionID: "q_015", QuestionNo: "15", QuestionType: "subjective", KnowledgePoint: "比例", ErrorType: "expression", WrongReason: "比例关系书写不规范", SourcePaper: "六年级数学期中卷", OriginalQuestion: "根据比例关系解决实际问题。", Score: 8, MaxScore: 10, CorrectAnswer: "设未知数并列比例求解，结果为 24 千克。", StudentAnswer: "3/5 = x/40，x = 24。", AnswerImageURL: "/mock/student-answer-q15.png", Explanation: "建模正确，补充规范比例式和单位说明。", CorrectionStatus: "pending", RepracticeStatus: "not_assigned", CreatedAt: "2026-06-20T09:10:00+08:00", Knowledge: []string{"比例"}},
		{ID: 3, StudentID: "stu_002", StudentName: "李四", ClassName: "六年级 3 班", SubmissionID: "sub_002", QuestionID: "q_018", QuestionNo: "18", QuestionType: "subjective", KnowledgePoint: "几何面积", ErrorType: "calculation", WrongReason: "图形拆分后面积计算不完整", SourcePaper: "六年级数学期中卷", OriginalQuestion: "将组合图形拆分后计算总面积。", Score: 6, MaxScore: 8, CorrectAnswer: "长方形与三角形面积相加。", StudentAnswer: "长方形面积 36，三角形面积 12。", AnswerImageURL: "/mock/student-answer-q18.png", Explanation: "标出拆分依据并写完整单位。", CorrectionStatus: "pending", RepracticeStatus: "not_assigned", CreatedAt: "2026-06-20T09:20:00+08:00", Knowledge: []string{"几何面积"}},
	}
}

func learningProfileFixture() LearningProfileResponse {
	return LearningProfileResponse{
		ClassName: "六年级 3 班",
		KnowledgeMastery: []KnowledgeMastery{
			{Name: "分数应用题", Mastery: 42, PreviousMastery: 38, Trend: 4, WrongCount: 29, StudentCount: 16},
			{Name: "几何面积", Mastery: 51, PreviousMastery: 57, Trend: -6, WrongCount: 21, StudentCount: 13},
			{Name: "比例换算", Mastery: 64, PreviousMastery: 59, Trend: 5, WrongCount: 15, StudentCount: 9},
		},
		StudentRisks: []StudentRisk{
			{StudentName: "李四", Risk: "连续 3 次未提交作业", Weakness: []string{"分数应用题", "比例"}},
			{StudentName: "赵六", Risk: "本次低于班均 18 分", Weakness: []string{"几何面积"}},
		},
		HomeworkWatch: []HomeworkWatch{
			{StudentName: "李四", ClassName: "六年级 3 班", Missing: 3, Guardian: "李四家长"},
			{StudentName: "赵六", ClassName: "六年级 3 班", Missing: 2, Guardian: "赵六家长"},
		},
	}
}

func guardianReportFixture(studentName string) GuardianReportResponse {
	if studentName == "" {
		studentName = "李四"
	}
	return GuardianReportResponse{StudentName: studentName, ClassName: "六年级 3 班", Summary: "本次成绩 72 分，共有 2 道题需要继续巩固。", Score: 72, WrongCount: 2, Weakness: []string{"分数", "几何面积"}, Actions: []string{"每天安排 15 分钟订正", "优先复习薄弱知识点", "完成再练后和孩子一起检查步骤"}}
}
