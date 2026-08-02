package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/mnah05/envitoo/internal/dto"
	"github.com/mnah05/envitoo/internal/service"
)

func newTestUploadServer(t *testing.T, storage *service.StorageService) *Server {
	t.Helper()
	return &Server{
		validate: dto.NewValidator(),
		storage:  storage,
		log:      slog.Default(),
	}
}

func newFakeStorage(t *testing.T) *service.StorageService {
	t.Helper()
	st, err := service.NewStorageService(service.StorageConfig{
		AccountID:       "test-account",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		BucketName:      "test-bucket",
		PublicURL:       "https://cdn.example.com",
	}, slog.Default())
	if err != nil {
		t.Fatalf("NewStorageService: %v", err)
	}
	if st == nil {
		t.Fatal("storage unexpectedly nil")
	}
	return st
}

func doPresignRequest(s *Server, body string) *httptest.ResponseRecorder {
	e := echo.New()
	e.POST("/api/upload/presigned-url", s.GeneratePresignedURL)
	req := httptest.NewRequest(http.MethodPost, "/api/upload/presigned-url", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestGeneratePresignedURL_ValidRequest(t *testing.T) {
	s := newTestUploadServer(t, newFakeStorage(t))
	rec := doPresignRequest(s, `{"filename":"cover.png","content_type":"image/png","size":1024}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp dto.PresignedURLResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.HasPrefix(resp.UploadURL, "https://") {
		t.Fatalf("upload_url = %q, want https URL", resp.UploadURL)
	}
	if !strings.HasPrefix(resp.PublicURL, "https://cdn.example.com/event-covers/") {
		t.Fatalf("public_url = %q, want cdn event-covers URL", resp.PublicURL)
	}
}

func TestGeneratePresignedURL_RejectsBadContentType(t *testing.T) {
	s := newTestUploadServer(t, newFakeStorage(t))
	rec := doPresignRequest(s, `{"filename":"cover.exe","content_type":"application/octet-stream","size":1024}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestGeneratePresignedURL_RejectsOversizeFile(t *testing.T) {
	s := newTestUploadServer(t, newFakeStorage(t))
	rec := doPresignRequest(s, `{"filename":"cover.jpg","content_type":"image/jpeg","size":5242881}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestGeneratePresignedURL_RejectsMalformedBody(t *testing.T) {
	s := newTestUploadServer(t, newFakeStorage(t))
	rec := doPresignRequest(s, `{"filename":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestGeneratePresignedURL_UnconfiguredStorage(t *testing.T) {
	s := newTestUploadServer(t, nil)
	rec := doPresignRequest(s, `{"filename":"cover.jpg","content_type":"image/jpeg","size":1024}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body: %s", rec.Code, rec.Body.String())
	}
}
