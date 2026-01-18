package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

// PresignedURLOptions はPresigned URL生成のオプションを定義します
type PresignedURLOptions struct {
	ContentType        string            // Content-Type (PUT時のみ)
	ContentDisposition string            // Content-Disposition (GET時のダウンロード名)
	Metadata           map[string]string // カスタムメタデータ
}

const (
	// Presigned URL有効期限
	PresignedUploadExpiry    = 15 * time.Minute
	PresignedMultipartExpiry = 1 * time.Hour
	PresignedDownloadExpiry  = 1 * time.Hour
	PresignedPreviewExpiry   = 15 * time.Minute

	// 最大ファイルサイズ
	MaxFileSize          int64 = 5 * 1024 * 1024 * 1024 // 5GB
	MultipartThreshold   int64 = 100 * 1024 * 1024      // 100MB
	MultipartPartSize    int64 = 64 * 1024 * 1024       // 64MB
	MaxMultipartParts    int   = 10000
	MaxConcurrentUploads int   = 5
)

// PresignedURLService はPresigned URL生成を提供します
type PresignedURLService struct {
	client     *minio.Client
	bucketName string
}

// NewPresignedURLService は新しいPresignedURLServiceを作成します
func NewPresignedURLService(client *MinIOClient) *PresignedURLService {
	return &PresignedURLService{
		client:     client.Client(),
		bucketName: client.BucketName(),
	}
}

// GeneratePutURL はアップロード用Presigned URLを生成します
func (s *PresignedURLService) GeneratePutURL(
	ctx context.Context,
	objectKey string,
	expiry time.Duration,
	opts *PresignedURLOptions,
) (string, error) {
	presignedURL, err := s.client.PresignedPutObject(
		ctx,
		s.bucketName,
		objectKey,
		expiry,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned put URL: %w", err)
	}

	return presignedURL.String(), nil
}

// GenerateGetURL はダウンロード用Presigned URLを生成します
func (s *PresignedURLService) GenerateGetURL(
	ctx context.Context,
	objectKey string,
	expiry time.Duration,
	opts *PresignedURLOptions,
) (string, error) {
	reqParams := make(url.Values)

	if opts != nil && opts.ContentDisposition != "" {
		reqParams.Set("response-content-disposition", opts.ContentDisposition)
	}

	presignedURL, err := s.client.PresignedGetObject(
		ctx,
		s.bucketName,
		objectKey,
		expiry,
		reqParams,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned get URL: %w", err)
	}

	return presignedURL.String(), nil
}

// GenerateDownloadURL はダウンロード用URLを生成します（ファイル名付き）
func (s *PresignedURLService) GenerateDownloadURL(
	ctx context.Context,
	objectKey string,
	filename string,
) (string, error) {
	opts := &PresignedURLOptions{
		ContentDisposition: fmt.Sprintf(`attachment; filename="%s"`, filename),
	}
	return s.GenerateGetURL(ctx, objectKey, PresignedDownloadExpiry, opts)
}

// GeneratePreviewURL はプレビュー用URLを生成します（インライン表示）
func (s *PresignedURLService) GeneratePreviewURL(
	ctx context.Context,
	objectKey string,
	filename string,
) (string, error) {
	opts := &PresignedURLOptions{
		ContentDisposition: fmt.Sprintf(`inline; filename="%s"`, filename),
	}
	return s.GenerateGetURL(ctx, objectKey, PresignedPreviewExpiry, opts)
}
