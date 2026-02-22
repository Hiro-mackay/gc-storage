package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// MockFolderRepository is a mock of repository.FolderRepository
type MockFolderRepository struct {
	mock.Mock
}

func NewMockFolderRepository(t *testing.T) *MockFolderRepository {
	m := &MockFolderRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockFolderRepository) Create(ctx context.Context, folder *entity.Folder) error {
	args := m.Called(ctx, folder)
	return args.Error(0)
}

func (m *MockFolderRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Folder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Folder), args.Error(1)
}

func (m *MockFolderRepository) Update(ctx context.Context, folder *entity.Folder) error {
	args := m.Called(ctx, folder)
	return args.Error(0)
}

func (m *MockFolderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFolderRepository) FindByParentID(ctx context.Context, parentID *uuid.UUID, ownerID uuid.UUID) ([]*entity.Folder, error) {
	args := m.Called(ctx, parentID, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Folder), args.Error(1)
}

func (m *MockFolderRepository) FindRootByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.Folder, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Folder), args.Error(1)
}

func (m *MockFolderRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.Folder, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Folder), args.Error(1)
}

func (m *MockFolderRepository) FindByCreatedBy(ctx context.Context, createdBy uuid.UUID) ([]*entity.Folder, error) {
	args := m.Called(ctx, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Folder), args.Error(1)
}

func (m *MockFolderRepository) ExistsByNameAndParent(ctx context.Context, name valueobject.FolderName, parentID *uuid.UUID, ownerID uuid.UUID) (bool, error) {
	args := m.Called(ctx, name, parentID, ownerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockFolderRepository) ExistsByNameAndOwnerRoot(ctx context.Context, name valueobject.FolderName, ownerID uuid.UUID) (bool, error) {
	args := m.Called(ctx, name, ownerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockFolderRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockFolderRepository) UpdateDepth(ctx context.Context, id uuid.UUID, depth int) error {
	args := m.Called(ctx, id, depth)
	return args.Error(0)
}

func (m *MockFolderRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockFolderRepository) BulkUpdateDepth(ctx context.Context, updates map[uuid.UUID]int) error {
	args := m.Called(ctx, updates)
	return args.Error(0)
}

// MockFolderClosureRepository is a mock of repository.FolderClosureRepository
type MockFolderClosureRepository struct {
	mock.Mock
}

func NewMockFolderClosureRepository(t *testing.T) *MockFolderClosureRepository {
	m := &MockFolderClosureRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockFolderClosureRepository) InsertSelfReference(ctx context.Context, folderID uuid.UUID) error {
	args := m.Called(ctx, folderID)
	return args.Error(0)
}

func (m *MockFolderClosureRepository) InsertAncestorPaths(ctx context.Context, paths []*entity.FolderPath) error {
	args := m.Called(ctx, paths)
	return args.Error(0)
}

func (m *MockFolderClosureRepository) DeleteByDescendant(ctx context.Context, descendantID uuid.UUID) error {
	args := m.Called(ctx, descendantID)
	return args.Error(0)
}

func (m *MockFolderClosureRepository) FindAncestorIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, folderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockFolderClosureRepository) FindDescendantIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, folderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockFolderClosureRepository) FindAncestorPaths(ctx context.Context, folderID uuid.UUID) ([]*entity.FolderPath, error) {
	args := m.Called(ctx, folderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.FolderPath), args.Error(1)
}

func (m *MockFolderClosureRepository) FindDescendantsWithDepth(ctx context.Context, folderID uuid.UUID) (map[uuid.UUID]int, error) {
	args := m.Called(ctx, folderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]int), args.Error(1)
}

func (m *MockFolderClosureRepository) DeleteSubtreePaths(ctx context.Context, folderID uuid.UUID) error {
	args := m.Called(ctx, folderID)
	return args.Error(0)
}

func (m *MockFolderClosureRepository) MoveSubtree(ctx context.Context, folderID uuid.UUID, newParentPaths []*entity.FolderPath) error {
	args := m.Called(ctx, folderID, newParentPaths)
	return args.Error(0)
}
