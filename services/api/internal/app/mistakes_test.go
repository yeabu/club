package app

import "testing"

func TestClassifyErrorType(t *testing.T) {
	tests := []struct {
		note  string
		score float64
		want  string
	}{
		{note: "计算过程有误", score: 2, want: "calculation"},
		{note: "没有理解题意，审题错误", score: 1, want: "reading"},
		{note: "单位和表达不完整", score: 6, want: "expression"},
		{note: "公式概念混淆", score: 0, want: "concept"},
		{note: "", score: 0, want: "concept"},
		{note: "步骤需要补充", score: 5, want: "other"},
	}
	for _, test := range tests {
		if got := classifyErrorType(test.note, test.score); got != test.want {
			t.Fatalf("classifyErrorType(%q, %.1f) = %q, want %q", test.note, test.score, got, test.want)
		}
	}
}

func TestTaskNineFixturesAreComplete(t *testing.T) {
	items := wrongQuestionsFixture()
	if len(items) < 3 {
		t.Fatalf("expected at least 3 wrong-question fixtures, got %d", len(items))
	}
	for _, item := range items {
		if item.StudentName == "" || item.QuestionNo == "" || item.KnowledgePoint == "" || item.ErrorType == "" {
			t.Fatalf("incomplete wrong-question fixture: %+v", item)
		}
		if item.MaxScore <= 0 || item.CorrectAnswer == "" || item.Explanation == "" {
			t.Fatalf("fixture is missing review detail: %+v", item)
		}
	}
	profile := learningProfileFixture()
	if len(profile.KnowledgeMastery) == 0 || len(profile.StudentRisks) == 0 {
		t.Fatalf("learning profile fixture is incomplete: %+v", profile)
	}
	report := guardianReportFixture("李四")
	if report.Summary == "" || len(report.Actions) == 0 {
		t.Fatalf("guardian report fixture is incomplete: %+v", report)
	}
}
