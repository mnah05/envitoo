package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// headObjectAPI is the subset of the AWS S3 client used by VerifyImage.
// Keeping it as an interface lets tests inject a fake without hitting R2.
type headObjectAPI interface {
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// presignPutObjectAPI is the subset of the AWS presign client used by
// GeneratePresignedURL. Presigning is a local computation (no network),
// so the real client works in tests with any endpoint.
type presignPutObjectAPI interface {
	PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// StorageService handles Cloudflare R2 (S3-compatible) object storage
// operations: generating presigned upload URLs and verifying uploaded objects.
type StorageService struct {
	client    headObjectAPI
	presign   presignPutObjectAPI
	bucket    string
	publicURL string
	log       *slog.Logger
}

// StorageConfig holds the configuration for the R2 storage service.
type StorageConfig struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string
}

// NewStorageService creates a new StorageService connected to Cloudflare R2.
// If any required config field is empty, returns nil (storage is optional).
func NewStorageService(cfg StorageConfig, log *slog.Logger) (*StorageService, error) {
	if cfg.AccountID == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" ||
		cfg.BucketName == "" || cfg.PublicURL == "" {
		log.Warn("R2 storage not configured, image uploads disabled")
		return nil, nil
	}

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID),
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
		config.WithEndpointResolverWithOptions(resolver),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	return &StorageService{
		client:    client,
		presign:   s3.NewPresignClient(client),
		bucket:    cfg.BucketName,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
		log:       log,
	}, nil
}

// GeneratePresignedURL creates a short-lived presigned PUT URL for direct
// upload to R2. The generated object key is a UUID with the original file
// extension appended, preventing collisions without exposing user filenames.
func (s *StorageService) GeneratePresignedURL(ctx context.Context, filename, contentType string) (uploadURL, publicURL string, err error) {
	if s == nil {
		return "", "", fmt.Errorf("storage service not configured")
	}

	ext := path.Ext(filename)
	if ext == "" {
		ext = ".jpg"
	}
	key := "event-covers/" + uuid.New().String() + ext

	req, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return "", "", fmt.Errorf("presign put object: %w", err)
	}

	return req.URL, s.publicURL + "/" + key, nil
}

// VerifyImage performs a HeadObject call to confirm the uploaded object exists
// and has the expected content type. Returns nil on success, or an error
// describing what went wrong.
func (s *StorageService) VerifyImage(ctx context.Context, publicURLStr string) error {
	if s == nil {
		return fmt.Errorf("storage service not configured")
	}

	key, err := objectKeyFromPublicURL(publicURLStr, s.publicURL)
	if err != nil {
		return err
	}

	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("image not found in storage: %w", err)
	}

	contentType := ""
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	if !isAllowedImageType(contentType) {
		return fmt.Errorf("unexpected content type: %s", contentType)
	}

	return nil
}

func objectKeyFromPublicURL(publicURLStr, baseURL string) (string, error) {
	u, err := url.Parse(publicURLStr)
	if err != nil {
		return "", fmt.Errorf("invalid image URL: %w", err)
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
	expectedHost := ""
	if baseURL != "" {
		base, _ := url.Parse(baseURL)
		if base != nil {
			expectedHost = base.Host
		}
	}
	if expectedHost != "" && u.Host != expectedHost {
		return "", fmt.Errorf("image URL does not match configured storage domain")
	}
	key := strings.TrimPrefix(u.Path, "/")
	if key == "" {
		return "", fmt.Errorf("empty object key in URL")
	}
	return key, nil
}

func isAllowedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}
