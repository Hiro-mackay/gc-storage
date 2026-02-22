package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// MockFileRepository is a mock of repository.FileRepository
type MockFileRepository struct {
	mock.Mock
}

func NewMockFileRepository(t *testing.T) *MockFileRepository {
	m := &MockFileRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockFileRepository) Create(ctx context.Context, file *entity.File) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockFileRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.File), args.Error(1)
}

func (m *MockFileRepository) Update(ctx context.Context, file *entity.File) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileRepository) FindByFolderID(ctx context.Context, folderID uuid.UUID) ([]*entity.File, error) {
	args := m.Called(ctx, folderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.File), args.Error(1)
}

func (m *MockFileRepository) FindByNameAndFolder(ctx context.Context, name valueobject.FileName, folderID uuid.UUID) (*entity.File, error) {
	args := m.Called(ctx, name, folderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.File), args.Error(1)
}

func (m *MockFileRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.File, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.File), args.Error(1)
}

func (m *MockFileRepository) FindByCreatedBy(ctx context.Context, createdBy uuid.UUID) ([]*entity.File, error) {
	args := m.Called(ctx, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.File), args.Error(1)
}

func (m *MockFileRepository) FindByStorageKey(ctx context.Context, storageKey valueobject.StorageKey) (*entity.File, error) {
	args := m.Called(ctx, storageKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.File), args.Error(1)
}

func (m *MockFileRepository) ExistsByNameAndFolder(ctx context.Context, name valueobject.FileName, folderID uuid.UUID) (bool, error) {
	args := m.Called(ctx, name, folderID)
	return args.Bool(0), args.Error(1)
}

func (m *MockFileRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.FileStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockFileRepository) FindByFolderIDs(ctx context.Context, folderIDs []uuid.UUID) ([]*entity.File, error) {
	args := m.Called(ctx, folderIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.File), args.Error(1)
}

func (m *MockFileRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockFileRepository) FindUploadFailed(ctx context.Context) ([]*entity.File, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.File), args.Error(1)
}
