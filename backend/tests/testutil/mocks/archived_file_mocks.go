package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// MockArchivedFileRepository is a mock of repository.ArchivedFileRepository
type MockArchivedFileRepository struct {
	mock.Mock
}

func NewMockArchivedFileRepository(t *testing.T) *MockArchivedFileRepository {
	m := &MockArchivedFileRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockArchivedFileRepository) Create(ctx context.Context, archivedFile *entity.ArchivedFile) error {
	args := m.Called(ctx, archivedFile)
	return args.Error(0)
}

func (m *MockArchivedFileRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ArchivedFile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ArchivedFile), args.Error(1)
}

func (m *MockArchivedFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockArchivedFileRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.ArchivedFile, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ArchivedFile), args.Error(1)
}

func (m *MockArchivedFileRepository) FindByOwnerWithPagination(ctx context.Context, ownerID uuid.UUID, limit int, cursor *uuid.UUID) ([]*entity.ArchivedFile, error) {
	args := m.Called(ctx, ownerID, limit, cursor)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ArchivedFile), args.Error(1)
}

func (m *MockArchivedFileRepository) FindExpired(ctx context.Context) ([]*entity.ArchivedFile, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ArchivedFile), args.Error(1)
}

func (m *MockArchivedFileRepository) FindByOriginalFileID(ctx context.Context, originalFileID uuid.UUID) (*entity.ArchivedFile, error) {
	args := m.Called(ctx, originalFileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ArchivedFile), args.Error(1)
}

// MockArchivedFileVersionRepository is a mock of repository.ArchivedFileVersionRepository
type MockArchivedFileVersionRepository struct {
	mock.Mock
}

func NewMockArchivedFileVersionRepository(t *testing.T) *MockArchivedFileVersionRepository {
	m := &MockArchivedFileVersionRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockArchivedFileVersionRepository) BulkCreate(ctx context.Context, versions []*entity.ArchivedFileVersion) error {
	args := m.Called(ctx, versions)
	return args.Error(0)
}

func (m *MockArchivedFileVersionRepository) FindByArchivedFileID(ctx context.Context, archivedFileID uuid.UUID) ([]*entity.ArchivedFileVersion, error) {
	args := m.Called(ctx, archivedFileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ArchivedFileVersion), args.Error(1)
}

func (m *MockArchivedFileVersionRepository) DeleteByArchivedFileID(ctx context.Context, archivedFileID uuid.UUID) error {
	args := m.Called(ctx, archivedFileID)
	return args.Error(0)
}
