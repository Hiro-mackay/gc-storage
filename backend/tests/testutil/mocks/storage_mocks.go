package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
)

// MockStorageService is a mock of service.StorageService
type MockStorageService struct {
	mock.Mock
}

func NewMockStorageService(t *testing.T) *MockStorageService {
	m := &MockStorageService{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockStorageService) GeneratePutURL(ctx context.Context, objectKey string, expiry time.Duration) (*service.PresignedURL, error) {
	args := m.Called(ctx, objectKey, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PresignedURL), args.Error(1)
}

func (m *MockStorageService) GenerateGetURL(ctx context.Context, objectKey string, expiry time.Duration) (*service.PresignedURL, error) {
	args := m.Called(ctx, objectKey, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PresignedURL), args.Error(1)
}

func (m *MockStorageService) CreateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	args := m.Called(ctx, objectKey)
	return args.String(0), args.Error(1)
}

func (m *MockStorageService) GeneratePartUploadURL(ctx context.Context, objectKey, uploadID string, partNumber int) (*service.MultipartUploadURL, error) {
	args := m.Called(ctx, objectKey, uploadID, partNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.MultipartUploadURL), args.Error(1)
}

func (m *MockStorageService) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, etags []string) error {
	args := m.Called(ctx, objectKey, uploadID, etags)
	return args.Error(0)
}

func (m *MockStorageService) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	args := m.Called(ctx, objectKey, uploadID)
	return args.Error(0)
}

func (m *MockStorageService) DeleteObject(ctx context.Context, objectKey string) error {
	args := m.Called(ctx, objectKey)
	return args.Error(0)
}

func (m *MockStorageService) DeleteObjects(ctx context.Context, objectKeys []string) error {
	args := m.Called(ctx, objectKeys)
	return args.Error(0)
}
