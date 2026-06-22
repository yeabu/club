package app

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

const maxScanUploadBytes int64 = 25 * 1024 * 1024

var allowedScanExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".pdf":  true,
	".png":  true,
	".webp": true,
	".zip":  true,
}

var allowedScanContentTypes = map[string]bool{
	"application/pdf":              true,
	"application/zip":              true,
	"application/x-zip-compressed": true,
	"image/jpeg":                   true,
	"image/png":                    true,
	"image/webp":                   true,
}

func saveScanUpload(config Config, header *multipart.FileHeader) (ScanFile, error) {
	if header == nil {
		return ScanFile{}, errors.New("file is required")
	}
	if header.Size <= 0 {
		return ScanFile{}, errors.New("file is empty")
	}
	if header.Size > maxScanUploadBytes {
		return ScanFile{}, fmt.Errorf("%s exceeds %d bytes", header.Filename, maxScanUploadBytes)
	}

	extension := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedScanExtensions[extension] {
		return ScanFile{}, fmt.Errorf("%s file type is not supported", header.Filename)
	}

	file, err := header.Open()
	if err != nil {
		return ScanFile{}, err
	}
	defer file.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	contentType := http.DetectContentType(head[:n])
	if extension == ".zip" && strings.Contains(contentType, "octet-stream") {
		contentType = "application/zip"
	}
	if !allowedScanContentTypes[contentType] {
		return ScanFile{}, fmt.Errorf("%s content type is not supported", header.Filename)
	}
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return ScanFile{}, err
		}
	} else {
		return ScanFile{}, errors.New("uploaded file cannot be rewound")
	}

	datePath := time.Now().Format("20060102")
	key := filepath.ToSlash(filepath.Join("scan", datePath, fmt.Sprintf("%d_%s", time.Now().UnixNano(), safeFileName(header.Filename))))
	if config.StorageDriver == "obs" {
		return saveOBSUpload(config.OBS, file, header, key, contentType)
	}
	targetPath := filepath.Join(localUploadRoot(), filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return ScanFile{}, err
	}
	target, err := os.Create(targetPath)
	if err != nil {
		return ScanFile{}, err
	}
	defer target.Close()

	written, err := io.Copy(target, io.LimitReader(file, maxScanUploadBytes+1))
	if err != nil {
		return ScanFile{}, err
	}
	if written > maxScanUploadBytes {
		return ScanFile{}, fmt.Errorf("%s exceeds %d bytes", header.Filename, maxScanUploadBytes)
	}

	return ScanFile{
		Key:         key,
		FileName:    header.Filename,
		ContentType: contentType,
		Size:        written,
		URL:         "/uploads/" + key,
	}, nil
}

func saveOBSUpload(config OBSConfig, body io.Reader, header *multipart.FileHeader, key, contentType string) (ScanFile, error) {
	if config.Endpoint == "" || config.Bucket == "" || config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return ScanFile{}, errors.New("OBS configuration is incomplete")
	}
	client, err := obs.New(config.AccessKeyID, config.SecretAccessKey, normalizeOBSEndpoint(config.Endpoint, config.Bucket))
	if err != nil {
		return ScanFile{}, fmt.Errorf("create OBS client: %w", err)
	}
	defer client.Close()
	output, err := client.PutObject(&obs.PutObjectInput{
		PutObjectBasicInput: obs.PutObjectBasicInput{
			ObjectOperationInput: obs.ObjectOperationInput{Bucket: config.Bucket, Key: key},
			HttpHeader:           obs.HttpHeader{ContentType: contentType},
			ContentLength:        header.Size,
		},
		Body: body,
	})
	if err != nil {
		return ScanFile{}, fmt.Errorf("upload to OBS: %w", err)
	}
	objectURL := output.ObjectUrl
	if objectURL == "" {
		objectURL = strings.TrimRight(config.Endpoint, "/") + "/" + key
	}
	return ScanFile{Key: key, FileName: header.Filename, ContentType: contentType, Size: header.Size, URL: objectURL}, nil
}

func normalizeOBSEndpoint(raw, bucket string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}
	parsed.Host = strings.TrimPrefix(parsed.Host, bucket+".")
	return strings.TrimRight(parsed.String(), "/")
}

func localUploadRoot() string {
	if root := env("UPLOAD_ROOT", ""); root != "" {
		return root
	}
	if path, ok := findUp(filepath.Join("config", "config.yaml")); ok {
		serviceRoot := filepath.Dir(filepath.Dir(path))
		return filepath.Join(serviceRoot, "data", "uploads")
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "data", "uploads")
	}
	return filepath.Join(os.TempDir(), "club-uploads")
}

func safeFileName(name string) string {
	base := filepath.Base(name)
	base = strings.TrimSpace(base)
	if base == "" || base == "." {
		return "scan-file"
	}
	replacer := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	return strings.Trim(replacer.ReplaceAllString(base, "-"), ".-")
}
