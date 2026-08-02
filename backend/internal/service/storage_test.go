package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

const testPublicURL = "https://cdn.example.com"

// fakeHeadObject is a configurable in-memory stand-in for the S3 client.
type fakeHeadObject struct {
	out *s3.HeadObjectOutput
	err error
}

func (f *fakeHeadObject) HeadObject(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return f.out, f.err
}

// fakePresign stands in for the presign client; PresignPutObject is a pure
// local computation (no network), so we just return a canned request.
type fakePresign struct{}

func (fakePresign) PresignPutObject(ctx context.Context, params *s3.PutObjectInput, _ ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if params == nil || params.Bucket == nil || params.Key == nil {
		return nil, errors.New("missing bucket/key")
	}
	return &v4.PresignedHTTPRequest{
		URL: "https://" + *params.Bucket + ".r2.cloudflarestorage.com/" + *params.Key,
	}, nil
}

func newTestStorage(client headObjectAPI) *StorageService {
	return &StorageService{
		client:    client,
		presign:   fakePresign{},
		bucket:    "test-bucket",
		publicURL: strings.TrimRight(testPublicURL, "/"),
		log:       slog.Default(),
	}
}

func notFoundError() error {
	return &smithy.OperationError{
		ServiceID:     "S3",
		OperationName: "HeadObject",
		Err: &smithyhttp.ResponseError{
			Response: &smithyhttp.Response{
				Response: &http.Response{StatusCode: 404, Status: "404 Not Found"},
			},
			Err: errors.New("Not Found"),
		},
	}
}

func TestObjectKeyFromPublicURL(t *testing.T) {
	tests := []struct {
		name    string
		urlStr  string
		wantKey string
		wantErr bool
	}{
		{"valid key", testPublicURL + "/event-covers/abc123.jpg", "event-covers/abc123.jpg", false},
		{"relative key without base", "event-covers/abc123.jpg", "event-covers/abc123.jpg", true},
		{"foreign host rejected", "https://evil.example.com/event-covers/x.jpg", "", true},
		{"non-http scheme", "ftp://cdn.example.com/x.jpg", "", true},
		{"empty path", testPublicURL + "/", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := objectKeyFromPublicURL(tt.urlStr, testPublicURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got key %q", key)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tt.wantKey {
				t.Fatalf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestObjectKeyFromPublicURL_NoBaseURL(t *testing.T) {
	// With an empty base URL the host check is skipped (storage is
	// unconfigured); the key is still extracted.
	key, err := objectKeyFromPublicURL("https://anything.example/x.jpg", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "x.jpg" {
		t.Fatalf("key = %q, want %q", key, "x.jpg")
	}
}

func TestIsAllowedImageType(t *testing.T) {
	for _, ct := range []string{"image/jpeg", "image/png", "image/webp"} {
		if !isAllowedImageType(ct) {
			t.Errorf("expected %q to be allowed", ct)
		}
	}
	for _, ct := range []string{"", "image/gif", "text/html", "application/octet-stream"} {
		if isAllowedImageType(ct) {
			t.Errorf("expected %q to be rejected", ct)
		}
	}
}

func TestVerifyImage_Success(t *testing.T) {
	ct := "image/jpeg"
	storage := newTestStorage(&fakeHeadObject{
		out: &s3.HeadObjectOutput{ContentType: &ct},
	})
	if err := storage.VerifyImage(context.Background(), testPublicURL+"/event-covers/abc.jpg"); err != nil {
		t.Fatalf("VerifyImage: %v", err)
	}
}

func TestVerifyImage_WrongContentType(t *testing.T) {
	ct := "text/html"
	storage := newTestStorage(&fakeHeadObject{
		out: &s3.HeadObjectOutput{ContentType: &ct},
	})
	err := storage.VerifyImage(context.Background(), testPublicURL+"/event-covers/abc.jpg")
	if err == nil {
		t.Fatal("expected error for text/html, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected content type") {
		t.Fatalf("error = %v, want content-type message", err)
	}
}

func TestVerifyImage_NotFound(t *testing.T) {
	storage := newTestStorage(&fakeHeadObject{err: notFoundError()})
	err := storage.VerifyImage(context.Background(), testPublicURL+"/event-covers/missing.jpg")
	if err == nil {
		t.Fatal("expected error for missing object, got nil")
	}
	if !strings.Contains(err.Error(), "image not found") {
		t.Fatalf("error = %v, want not-found message", err)
	}
}

func TestVerifyImage_ForeignDomain(t *testing.T) {
	// Domain mismatch must fail before any HeadObject call.
	storage := newTestStorage(&fakeHeadObject{out: &s3.HeadObjectOutput{}})
	if err := storage.VerifyImage(context.Background(), "https://evil.example.com/x.jpg"); err == nil {
		t.Fatal("expected error for foreign domain, got nil")
	}
}

func TestVerifyImage_Unconfigured(t *testing.T) {
	var storage *StorageService
	if err := storage.VerifyImage(context.Background(), testPublicURL+"/x.jpg"); err == nil {
		t.Fatal("expected error on nil storage, got nil")
	}
}

func TestGeneratePresignedURL_KeyFormat(t *testing.T) {
	storage := newTestStorage(&fakeHeadObject{})
	uploadURL, publicURL, err := storage.GeneratePresignedURL(context.Background(), "my photo.png", "image/png")
	if err != nil {
		t.Fatalf("GeneratePresignedURL: %v", err)
	}
	if !strings.HasPrefix(uploadURL, "https://") {
		t.Fatalf("upload URL = %q, want https", uploadURL)
	}
	key := strings.TrimPrefix(publicURL, testPublicURL+"/")
	if !strings.HasPrefix(key, "event-covers/") {
		t.Fatalf("key = %q, want event-covers/ prefix", key)
	}
	if !strings.HasSuffix(key, ".png") {
		t.Fatalf("key = %q, want .png suffix", key)
	}
	if strings.Contains(key, "my photo") {
		t.Fatalf("key = %q, must not leak original filename", key)
	}
}

func TestGeneratePresignedURL_DefaultExtension(t *testing.T) {
	storage := newTestStorage(&fakeHeadObject{})
	_, publicURL, err := storage.GeneratePresignedURL(context.Background(), "noext", "image/webp")
	if err != nil {
		t.Fatalf("GeneratePresignedURL: %v", err)
	}
	if !strings.HasSuffix(publicURL, ".jpg") {
		t.Fatalf("publicURL = %q, want .jpg fallback extension", publicURL)
	}
}

func TestGeneratePresignedURL_Unconfigured(t *testing.T) {
	var storage *StorageService
	if _, _, err := storage.GeneratePresignedURL(context.Background(), "a.jpg", "image/jpeg"); err == nil {
		t.Fatal("expected error on nil storage, got nil")
	}
}

func TestGeneratePresignedURL_KeyIsPathScoped(t *testing.T) {
	// The object key must live under event-covers/ so the bucket root
	// stays predictable and public URLs are namespaced.
	storage := newTestStorage(&fakeHeadObject{})
	_, publicURL, err := storage.GeneratePresignedURL(context.Background(), "x.png", "image/png")
	if err != nil {
		t.Fatalf("GeneratePresignedURL: %v", err)
	}
	key := strings.TrimPrefix(publicURL, testPublicURL+"/")
	if path.Dir(key) != "event-covers" {
		t.Fatalf("key dir = %q, want event-covers", path.Dir(key))
	}
}
