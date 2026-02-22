package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// MockUploadSessionRepository is a mock of repository.UploadSessionRepository
type MockUploadSessionRepository struct {
	mock.Mock
}

func NewMockUploadSessionRepository(t *testing.T) *MockUploadSessionRepository {
	m := &MockUploadSessionRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockUploadSessionRepository) Create(ctx context.Context, session *entity.UploadSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockUploadSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.UploadSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UploadSession), args.Error(1)
}

func (m *MockUploadSessionRepository) Update(ctx context.Context, session *entity.UploadSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockUploadSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUploadSessionRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) (*entity.UploadSession, error) {
	args := m.Called(ctx, fileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UploadSession), args.Error(1)
}

func (m *MockUploadSessionRepository) FindByStorageKey(ctx context.Context, storageKey valueobject.StorageKey) (*entity.UploadSession, error) {
	args := m.Called(ctx, storageKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UploadSession), args.Error(1)
}

func (m *MockUploadSessionRepository) FindExpired(ctx context.Context) ([]*entity.UploadSession, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.UploadSession), args.Error(1)
}

func (m *MockUploadSessionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.UploadSessionStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
