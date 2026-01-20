package service

import (
	"context"
	"time"
)

// PresignedURL はPresigned URL情報を表します
type PresignedURL struct {
	URL       string
	ExpiresAt time.Time
}

// MultipartUploadURL はマルチパートアップロードURL情報を表します
type MultipartUploadURL struct {
	PartNumber int
	URL        string
	ExpiresAt  time.Time
}

// StorageService はストレージ操作のドメインサービスインターフェースです
type StorageService interface {
	// シングルパートアップロード用URL生成
	GeneratePutURL(ctx context.Context, objectKey string, expiry time.Duration) (*PresignedURL, error)

	// ダウンロード用URL生成
	GenerateGetURL(ctx context.Context, objectKey string, expiry time.Duration) (*PresignedURL, error)

	// マルチパートアップロード開始
	CreateMultipartUpload(ctx context.Context, objectKey string) (uploadID string, err error)

	// マルチパートアップロード用パートURL生成
	GeneratePartUploadURL(ctx context.Context, objectKey, uploadID string, partNumber int) (*MultipartUploadURL, error)

	// マルチパートアップロード完了
	CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, etags []string) error

	// マルチパートアップロード中断
	AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error

	// オブジェクト削除
	DeleteObject(ctx context.Context, objectKey string) error

	// 複数オブジェクト削除
	DeleteObjects(ctx context.Context, objectKeys []string) error
}
