package app

import "testing"

func TestValidateTemplateDetectsRegionOverlap(t *testing.T) {
	template := PaperTemplate{
		ID:      "tpl_test",
		Name:    "测试答题卡",
		Subject: "数学",
		Grade:   "六年级",
		Questions: []QuestionTemplate{
			{ID: "q1", No: "1", Type: "single_choice", Score: 2, StandardAnswer: "A", Region: Region{Page: 1, X: 10, Y: 10, Width: 100, Height: 80}},
			{ID: "q2", No: "2", Type: "subjective", Score: 8, ScoringRules: []string{"步骤完整"}, Region: Region{Page: 1, X: 80, Y: 50, Width: 100, Height: 80}},
		},
	}
	result := ValidateTemplate(template)
	if result.Valid {
		t.Fatal("overlapping template should be invalid")
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Code == "region_overlap" && issue.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected region_overlap issue, got %+v", result.Issues)
	}
}

func TestDiffTemplatesReportsChangedFields(t *testing.T) {
	base := templatesFixture()[0]
	target := base
	target.ID = "tpl_target"
	target.ParentID = base.ID
	target.Questions = append([]QuestionTemplate{}, base.Questions...)
	target.Questions[0].Score = 3
	target.Questions = append(target.Questions, QuestionTemplate{ID: "q_new", No: "99", Type: "subjective", Score: 5})

	diff := DiffTemplates(base, target)
	if len(diff.AddedQuestions) != 1 {
		t.Fatalf("expected one added question, got %+v", diff.AddedQuestions)
	}
	if len(diff.ChangedQuestions) != 1 || diff.ChangedQuestions[0].Fields[0] != "score" {
		t.Fatalf("expected score change, got %+v", diff.ChangedQuestions)
	}
}

func TestScanExceptionsFromJobsDetectsFileAndPageIssues(t *testing.T) {
	items := ScanExceptionsFromJobs([]ScanJob{{
		ID: "scan_test", Title: "测试扫描", ClassName: "六年级 3 班", Pages: 3, QueueStatus: "queued",
		Files: []ScanFile{{Key: "a", FileName: "a.png", Page: 1, Status: "failed", FailureReason: "识别失败"}, {Key: "b", FileName: "b.png", Page: 1, MatchStatus: "pending"}},
	}})
	if len(items) < 3 {
		t.Fatalf("expected file failure, pending match and missing page exceptions, got %+v", items)
	}
}

func TestNormalizeMobile(t *testing.T) {
	if got := normalizeMobile("+86 138-0000-0011"); got != "8613800000011" {
		t.Fatalf("unexpected normalized mobile: %s", got)
	}
}

func TestStandardAIResultShape(t *testing.T) {
	task := AITaskRecord{ID: "aitask_1", TaskType: "paper_template_analysis", Provider: "generic-http", OwnerType: "template", OwnerID: "tpl_1"}
	result := standardAIResult(task, "succeeded", "generic-http", map[string]any{"modelVersion": "model-v1", "outputs": map[string]any{"ok": true}, "metrics": map[string]any{"latencyMs": 10}}, "")
	if result["schemaVersion"] != "ai-result.v1" || result["status"] != "succeeded" || result["modelVersion"] != "model-v1" {
		t.Fatalf("unexpected standard result: %+v", result)
	}
	if _, ok := result["outputs"].(map[string]any); !ok {
		t.Fatalf("outputs should be a map: %+v", result["outputs"])
	}
}
