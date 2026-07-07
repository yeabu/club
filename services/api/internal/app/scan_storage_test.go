package app

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestHandleScanUploadStoresAllowedFile(t *testing.T) {
	t.Setenv("UPLOAD_ROOT", t.TempDir())
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "paper.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d})
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/scan/uploads", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	(&App{}).handleScanUpload(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var result ScanUploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected one file, got %d", len(result.Files))
	}
	if result.Files[0].Key == "" || filepath.Ext(result.Files[0].Key) != ".png" {
		t.Fatalf("unexpected object key: %#v", result.Files[0])
	}
}

func TestHandleScanUploadRejectsUnsupportedFile(t *testing.T) {
	t.Setenv("UPLOAD_ROOT", t.TempDir())
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "paper.exe")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("not allowed"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/scan/uploads", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	(&App{}).handleScanUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestNormalizeOBSEndpointStripsBucketFromBareHost(t *testing.T) {
	got := normalizeOBSEndpoint("yess.obs.cn-south-1.myhuaweicloud.com", "yess")
	want := "https://obs.cn-south-1.myhuaweicloud.com"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestOBSObjectURLUsesBucketHost(t *testing.T) {
	got := obsObjectURL(OBSConfig{Endpoint: "yess.obs.cn-south-1.myhuaweicloud.com", Bucket: "yess"}, "scan/demo.pdf")
	want := "https://yess.obs.cn-south-1.myhuaweicloud.com/scan/demo.pdf"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestSafeFileNamePreservesExtensionForChinesePDFName(t *testing.T) {
	got := safeFileName("数学二年级下册期末知识梳理.pdf")
	if got != "scan-file.pdf" {
		t.Fatalf("expected scan-file.pdf, got %s", got)
	}
}

func TestSafeFileNameKeepsASCIIStemAndExtension(t *testing.T) {
	got := safeFileName("paper final.PDF")
	if got != "paper-final.pdf" {
		t.Fatalf("expected paper-final.pdf, got %s", got)
	}
}

func TestScanUploadFallsBackToLocalWhenOBSUnavailable(t *testing.T) {
	t.Setenv("UPLOAD_ROOT", t.TempDir())
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "paper.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d})
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/scan/uploads", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	app := &App{config: Config{StorageDriver: "obs"}}

	app.handleScanUpload(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 fallback, got %d: %s", rec.Code, rec.Body.String())
	}
	var result ScanUploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("expected one file, got %d", len(result.Files))
	}
	if result.Files[0].URL == "" || result.Files[0].FailureReason == "" {
		t.Fatalf("expected local fallback metadata, got %#v", result.Files[0])
	}
}
