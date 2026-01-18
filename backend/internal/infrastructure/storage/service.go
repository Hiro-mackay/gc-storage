package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
)

// ObjectInfo はオブジェクト情報を表します
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
	Metadata     map[string]string
}

// StorageService はストレージ操作を提供する統合サービスです
type StorageService struct {
	client     *minio.Client
	bucketName string
	presigned  *PresignedURLService
	multipart  *MultipartService
}

// NewStorageService は新しいStorageServiceを作成します
func NewStorageService(client *MinIOClient) *StorageService {
	return &StorageService{
		client:     client.Client(),
		bucketName: client.BucketName(),
		presigned:  NewPresignedURLService(client),
		multipart:  NewMultipartService(client),
	}
}

// GeneratePutURL はアップロード用Presigned URLを生成します
func (s *StorageService) GeneratePutURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	return s.presigned.GeneratePutURL(ctx, objectKey, expiry, nil)
}

// GenerateGetURL はダウンロード用Presigned URLを生成します
func (s *StorageService) GenerateGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	return s.presigned.GenerateGetURL(ctx, objectKey, expiry, nil)
}

// GenerateDownloadURL はダウンロード用URLを生成します
func (s *StorageService) GenerateDownloadURL(ctx context.Context, objectKey string, filename string) (string, error) {
	return s.presigned.GenerateDownloadURL(ctx, objectKey, filename)
}

// GeneratePreviewURL はプレビュー用URLを生成します
func (s *StorageService) GeneratePreviewURL(ctx context.Context, objectKey string, filename string) (string, error) {
	return s.presigned.GeneratePreviewURL(ctx, objectKey, filename)
}

// ObjectExists はオブジェクトが存在するか確認します
func (s *StorageService) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}
	return true, nil
}

// GetObjectInfo はオブジェクト情報を取得します
func (s *StorageService) GetObjectInfo(ctx context.Context, objectKey string) (*ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &ObjectInfo{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified,
		Metadata:     info.UserMetadata,
	}, nil
}

// DeleteObject はオブジェクトを削除します
func (s *StorageService) DeleteObject(ctx context.Context, objectKey string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// DeleteObjects は複数オブジェクトを一括削除します
func (s *StorageService) DeleteObjects(ctx context.Context, objectKeys []string) error {
	objectsCh := make(chan minio.ObjectInfo, len(objectKeys))

	go func() {
		defer close(objectsCh)
		for _, key := range objectKeys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	errorCh := s.client.RemoveObjects(ctx, s.bucketName, objectsCh, minio.RemoveObjectsOptions{})

	var errors []error
	for e := range errorCh {
		if e.Err != nil {
			errors = append(errors, fmt.Errorf("failed to delete %s: %w", e.ObjectName, e.Err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete some objects: %v", errors)
	}

	return nil
}

// CopyObject はオブジェクトをコピーします
func (s *StorageService) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucketName,
		Object: srcKey,
	}

	dst := minio.CopyDestOptions{
		Bucket: s.bucketName,
		Object: dstKey,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// GetObject はオブジェクトを直接取得します（内部使用のみ）
func (s *StorageService) GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return object, nil
}

// PutObject はオブジェクトを直接アップロードします（内部使用のみ、小さなファイル向け）
func (s *StorageService) PutObject(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucketName, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

// マルチパート操作の委譲
func (s *StorageService) CreateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	return s.multipart.CreateMultipartUpload(ctx, objectKey, nil)
}

func (s *StorageService) GeneratePartUploadURL(ctx context.Context, objectKey, uploadID string, partNumber int) (string, error) {
	return s.multipart.GeneratePartUploadURL(ctx, objectKey, uploadID, partNumber)
}

func (s *StorageService) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []CompletedPart) (string, error) {
	return s.multipart.CompleteMultipartUpload(ctx, objectKey, uploadID, parts)
}

func (s *StorageService) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	return s.multipart.AbortMultipartUpload(ctx, objectKey, uploadID)
}

func (s *StorageService) ListParts(ctx context.Context, objectKey, uploadID string) ([]PartInfo, error) {
	return s.multipart.ListParts(ctx, objectKey, uploadID)
}
