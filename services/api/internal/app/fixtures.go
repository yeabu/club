package app

func dashboardFixture() DashboardResponse {
	return DashboardResponse{
		Metrics: []Metric{
			{Label: "待批试卷", Value: "128", Delta: "较昨日 +24", Tone: "primary"},
			{Label: "主观题待复核", Value: "36", Delta: "AI 已预评分", Tone: "warning"},
			{Label: "未提交作业", Value: "8", Delta: "3 人连续未交", Tone: "danger"},
			{Label: "班级平均分", Value: "81.6", Delta: "较上次 +3.2", Tone: "success"},
		},
		ScanQueue: []ScanJob{
			{ID: "scan_001", Title: "六年级数学期中卷", ClassName: "六年级 3 班", Pages: 96, Status: "OCR 识别中", Progress: 68},
			{ID: "scan_002", Title: "分数应用题专项", ClassName: "六年级 1 班", Pages: 42, Status: "等待 OMR", Progress: 32},
			{ID: "scan_003", Title: "几何面积小测", ClassName: "五年级 2 班", Pages: 48, Status: "待导入", Progress: 0},
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
			Questions: []QuestionTemplate{
				{ID: "q_001", No: "1", Type: "single_choice", Score: 2, Knowledge: []string{"分数"}, Region: Region{Page: 1, X: 120, Y: 260, Width: 480, Height: 80}},
				{ID: "q_015", No: "15", Type: "subjective", Score: 10, Knowledge: []string{"比例", "应用题建模"}, Region: Region{Page: 2, X: 96, Y: 420, Width: 620, Height: 180}},
			},
		},
	}
}

func analyticsFixture() ClassroomAnalytics {
	return ClassroomAnalytics{
		ClassName:    "六年级 3 班",
		AverageScore: 81.6,
		HighestScore: 98,
		LowestScore:  54,
		QuestionStats: []QuestionStat{
			{No: "1", Accuracy: 96, Type: "单选题"},
			{No: "8", Accuracy: 82, Type: "填空题"},
			{No: "15", Accuracy: 42, Type: "应用题"},
			{No: "18", Accuracy: 38, Type: "应用题"},
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
	}
}
