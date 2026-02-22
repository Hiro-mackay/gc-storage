package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// MockFileVersionRepository is a mock of repository.FileVersionRepository
type MockFileVersionRepository struct {
	mock.Mock
}

func NewMockFileVersionRepository(t *testing.T) *MockFileVersionRepository {
	m := &MockFileVersionRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockFileVersionRepository) Create(ctx context.Context, version *entity.FileVersion) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockFileVersionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.FileVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.FileVersion), args.Error(1)
}

func (m *MockFileVersionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileVersionRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) ([]*entity.FileVersion, error) {
	args := m.Called(ctx, fileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.FileVersion), args.Error(1)
}

func (m *MockFileVersionRepository) FindByFileAndVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) (*entity.FileVersion, error) {
	args := m.Called(ctx, fileID, versionNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.FileVersion), args.Error(1)
}

func (m *MockFileVersionRepository) FindLatestByFileID(ctx context.Context, fileID uuid.UUID) (*entity.FileVersion, error) {
	args := m.Called(ctx, fileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.FileVersion), args.Error(1)
}

func (m *MockFileVersionRepository) BulkCreate(ctx context.Context, versions []*entity.FileVersion) error {
	args := m.Called(ctx, versions)
	return args.Error(0)
}

func (m *MockFileVersionRepository) DeleteByFileID(ctx context.Context, fileID uuid.UUID) error {
	args := m.Called(ctx, fileID)
	return args.Error(0)
}

func (m *MockFileVersionRepository) FindByFileIDs(ctx context.Context, fileIDs []uuid.UUID) ([]*entity.FileVersion, error) {
	args := m.Called(ctx, fileIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.FileVersion), args.Error(1)
}

func (m *MockFileVersionRepository) CountByFileID(ctx context.Context, fileID uuid.UUID) (int, error) {
	args := m.Called(ctx, fileID)
	return args.Int(0), args.Error(1)
}
