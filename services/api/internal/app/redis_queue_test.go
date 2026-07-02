package app

import "testing"

func TestScanQueuePayloadIncludesTemplateVersionAndFiles(t *testing.T) {
	payload := scanQueuePayload(
		ScanJob{
			ID:         "scan_001",
			ScanType:   "answer_sheet",
			Title:      "期中卷",
			ClassName:  "六年级 3 班",
			TemplateID: "tpl_001",
			Pages:      2,
			Files: []ScanFile{
				{Key: "scan/20260619/a.pdf", FileName: "a.pdf", URL: "https://example.test/a.pdf"},
				{Key: "scan/20260619/b.png", FileName: "b.png", URL: "https://example.test/b.png"},
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
	if payload.ScanType != "answer_sheet" {
		t.Fatalf("expected answer_sheet scan type, got %s", payload.ScanType)
	}
	if len(payload.FileKeys) != 2 || payload.FileKeys[1] != "scan/20260619/b.png" {
		t.Fatalf("unexpected file keys: %#v", payload.FileKeys)
	}
	if len(payload.Files) != 2 || payload.Files[1].URL != "https://example.test/b.png" {
		t.Fatalf("unexpected files: %#v", payload.Files)
	}
}
