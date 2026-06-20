package app

import "testing"

func TestScanQueuePayloadIncludesTemplateVersionAndFiles(t *testing.T) {
	payload := scanQueuePayload(
		ScanJob{
			ID:         "scan_001",
			Title:      "期中卷",
			ClassName:  "六年级 3 班",
			TemplateID: "tpl_001",
			Pages:      2,
			Files: []ScanFile{
				{Key: "scan/20260619/a.pdf"},
				{Key: "scan/20260619/b.png"},
			},
		},
		PaperTemplate{ID: "tpl_001", Version: 3},
	)

	if payload.TaskID != "scan_001" {
		t.Fatalf("unexpected task id: %s", payload.TaskID)
	}
	if payload.TemplateVersion != 3 {
		t.Fatalf("expected template version 3, got %d", payload.TemplateVersion)
	}
	if len(payload.FileKeys) != 2 || payload.FileKeys[1] != "scan/20260619/b.png" {
		t.Fatalf("unexpected file keys: %#v", payload.FileKeys)
	}
}
