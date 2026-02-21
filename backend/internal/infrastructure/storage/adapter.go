package storage

import (
	"context"
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// StorageServiceAdapter はインフラ層のStorageServiceをドメイン層のインターフェースに適合させるアダプターです
type StorageServiceAdapter struct {
	svc *StorageService
}

// NewStorageServiceAdapter は新しいStorageServiceAdapterを作成します
func NewStorageServiceAdapter(svc *StorageService) *StorageServiceAdapter {
	return &StorageServiceAdapter{svc: svc}
}

// GeneratePutURL はアップロード用Presigned URLを生成します
func (a *StorageServiceAdapter) GeneratePutURL(ctx context.Context, objectKey string, expiry time.Duration) (*service.PresignedURL, error) {
	urlStr, err := a.svc.GeneratePutURL(ctx, objectKey, expiry)
	if err != nil {
		return nil, err
	}
	return &service.PresignedURL{
		URL:       urlStr,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// GenerateGetURL はダウンロード用Presigned URLを生成します
func (a *StorageServiceAdapter) GenerateGetURL(ctx context.Context, objectKey string, expiry time.Duration) (*service.PresignedURL, error) {
	urlStr, err := a.svc.GenerateGetURL(ctx, objectKey, expiry)
	if err != nil {
		return nil, err
	}
	return &service.PresignedURL{
		URL:       urlStr,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// CreateMultipartUpload はマルチパートアップロードを開始します
func (a *StorageServiceAdapter) CreateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	return a.svc.CreateMultipartUpload(ctx, objectKey)
}

// GeneratePartUploadURL はパートアップロード用URLを生成します
func (a *StorageServiceAdapter) GeneratePartUploadURL(ctx context.Context, objectKey, uploadID string, partNumber int) (*service.MultipartUploadURL, error) {
	urlStr, err := a.svc.GeneratePartUploadURL(ctx, objectKey, uploadID, partNumber)
	if err != nil {
		return nil, err
	}
	return &service.MultipartUploadURL{
		PartNumber: partNumber,
		URL:        urlStr,
		ExpiresAt:  time.Now().Add(PresignedMultipartExpiry),
	}, nil
}

// CompleteMultipartUpload はマルチパートアップロードを完了します
func (a *StorageServiceAdapter) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, etags []string) error {
	parts := make([]CompletedPart, len(etags))
	for i, etag := range etags {
		parts[i] = CompletedPart{
			PartNumber: i + 1,
			ETag:       etag,
		}
	}
	_, err := a.svc.CompleteMultipartUpload(ctx, objectKey, uploadID, parts)
	return err
}

// AbortMultipartUpload はマルチパートアップロードを中断します
func (a *StorageServiceAdapter) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	return a.svc.AbortMultipartUpload(ctx, objectKey, uploadID)
}

// DeleteObject はオブジェクトを削除します
func (a *StorageServiceAdapter) DeleteObject(ctx context.Context, objectKey string) error {
	return a.svc.DeleteObject(ctx, objectKey)
}

// DeleteObjects は複数オブジェクトを削除します
func (a *StorageServiceAdapter) DeleteObjects(ctx context.Context, objectKeys []string) error {
	return a.svc.DeleteObjects(ctx, objectKeys)
}
