package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Client struct {
	client *minio.Client
	bucket string
}

func NewS3Client(cfg config.Config) (*S3Client, error) {
	if cfg.S3Endpoint == "" || cfg.S3Bucket == "" {
		return nil, fmt.Errorf("s3 not configured")
	}

	endpoint := cfg.S3Endpoint
	secure := strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: secure,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	if cfg.S3ForcePathStyle {
		_ = client
	}

	return &S3Client{client: client, bucket: cfg.S3Bucket}, nil
}

func (s *S3Client) PutObject(ctx context.Context, key, contentType string, data []byte) (string, error) {
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("s3://%s/%s", s.bucket, key), nil
}

func (s *S3Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func TenantImportKey(orgID, importRunID, filename string) string {
	return fmt.Sprintf("%s/imports/%s/raw/%s", orgID, importRunID, filename)
}
