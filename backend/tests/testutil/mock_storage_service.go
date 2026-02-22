package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// MockStorageService はテスト用のStorageService実装です
type MockStorageService struct {
	// エラーを返すように設定できる
	PutURLError            error
	GetURLError            error
	CreateMultipartError   error
	GeneratePartURLError   error
	CompleteMultipartError error
	AbortMultipartError    error
	DeleteObjectError      error
	DeleteObjectsError     error
}

// NewMockStorageService は新しいMockStorageServiceを作成します
func NewMockStorageService() *MockStorageService {
	return &MockStorageService{}
}

// GeneratePutURL はアップロード用Presigned URLを生成します
func (m *MockStorageService) GeneratePutURL(ctx context.Context, objectKey string, expiry time.Duration) (*service.PresignedURL, error) {
	if m.PutURLError != nil {
		return nil, m.PutURLError
	}
	return &service.PresignedURL{
		URL:       fmt.Sprintf("http://mock-storage/upload/%s", objectKey),
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// GenerateGetURL はダウンロード用Presigned URLを生成します
func (m *MockStorageService) GenerateGetURL(ctx context.Context, objectKey string, expiry time.Duration) (*service.PresignedURL, error) {
	if m.GetURLError != nil {
		return nil, m.GetURLError
	}
	return &service.PresignedURL{
		URL:       fmt.Sprintf("http://mock-storage/download/%s", objectKey),
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// CreateMultipartUpload はマルチパートアップロードを開始します
func (m *MockStorageService) CreateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	if m.CreateMultipartError != nil {
		return "", m.CreateMultipartError
	}
	return fmt.Sprintf("mock-upload-id-%s", objectKey), nil
}

// GeneratePartUploadURL はパートアップロード用URLを生成します
func (m *MockStorageService) GeneratePartUploadURL(ctx context.Context, objectKey, uploadID string, partNumber int) (*service.MultipartUploadURL, error) {
	if m.GeneratePartURLError != nil {
		return nil, m.GeneratePartURLError
	}
	return &service.MultipartUploadURL{
		PartNumber: partNumber,
		URL:        fmt.Sprintf("http://mock-storage/upload/%s/part/%d", objectKey, partNumber),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}, nil
}

// CompleteMultipartUpload はマルチパートアップロードを完了します
func (m *MockStorageService) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, etags []string) error {
	if m.CompleteMultipartError != nil {
		return m.CompleteMultipartError
	}
	return nil
}

// AbortMultipartUpload はマルチパートアップロードを中断します
func (m *MockStorageService) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	if m.AbortMultipartError != nil {
		return m.AbortMultipartError
	}
	return nil
}

// DeleteObject はオブジェクトを削除します
func (m *MockStorageService) DeleteObject(ctx context.Context, objectKey string) error {
	if m.DeleteObjectError != nil {
		return m.DeleteObjectError
	}
	return nil
}

// DeleteObjects は複数オブジェクトを削除します
func (m *MockStorageService) DeleteObjects(ctx context.Context, objectKeys []string) error {
	if m.DeleteObjectsError != nil {
		return m.DeleteObjectsError
	}
	return nil
}

// SetPutURLError はGeneratePutURLでエラーを返すように設定します
func (m *MockStorageService) SetPutURLError(err error) {
	m.PutURLError = err
}

// SetGetURLError はGenerateGetURLでエラーを返すように設定します
func (m *MockStorageService) SetGetURLError(err error) {
	m.GetURLError = err
}

// Reset はモックの状態をリセットします
func (m *MockStorageService) Reset() {
	m.PutURLError = nil
	m.GetURLError = nil
	m.CreateMultipartError = nil
	m.GeneratePartURLError = nil
	m.CompleteMultipartError = nil
	m.AbortMultipartError = nil
	m.DeleteObjectError = nil
	m.DeleteObjectsError = nil
}
