package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// MockShareLinkRepository is a mock of repository.ShareLinkRepository
type MockShareLinkRepository struct {
	mock.Mock
}

func NewMockShareLinkRepository(t *testing.T) *MockShareLinkRepository {
	m := &MockShareLinkRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockShareLinkRepository) Create(ctx context.Context, link *entity.ShareLink) error {
	args := m.Called(ctx, link)
	return args.Error(0)
}

func (m *MockShareLinkRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ShareLink, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ShareLink), args.Error(1)
}

func (m *MockShareLinkRepository) Update(ctx context.Context, link *entity.ShareLink) error {
	args := m.Called(ctx, link)
	return args.Error(0)
}

func (m *MockShareLinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockShareLinkRepository) FindByToken(ctx context.Context, token valueobject.ShareToken) (*entity.ShareLink, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ShareLink), args.Error(1)
}

func (m *MockShareLinkRepository) FindByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*entity.ShareLink, error) {
	args := m.Called(ctx, resourceType, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ShareLink), args.Error(1)
}

func (m *MockShareLinkRepository) FindByCreator(ctx context.Context, createdBy uuid.UUID) ([]*entity.ShareLink, error) {
	args := m.Called(ctx, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ShareLink), args.Error(1)
}

func (m *MockShareLinkRepository) FindActiveByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*entity.ShareLink, error) {
	args := m.Called(ctx, resourceType, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ShareLink), args.Error(1)
}

func (m *MockShareLinkRepository) FindExpired(ctx context.Context) ([]*entity.ShareLink, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ShareLink), args.Error(1)
}

func (m *MockShareLinkRepository) UpdateStatusBatch(ctx context.Context, ids []uuid.UUID, status valueobject.ShareLinkStatus) (int64, error) {
	args := m.Called(ctx, ids, status)
	return args.Get(0).(int64), args.Error(1)
}

// MockShareLinkAccessRepository is a mock of repository.ShareLinkAccessRepository
type MockShareLinkAccessRepository struct {
	mock.Mock
}

func NewMockShareLinkAccessRepository(t *testing.T) *MockShareLinkAccessRepository {
	m := &MockShareLinkAccessRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockShareLinkAccessRepository) Create(ctx context.Context, access *entity.ShareLinkAccess) error {
	args := m.Called(ctx, access)
	return args.Error(0)
}

func (m *MockShareLinkAccessRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ShareLinkAccess, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ShareLinkAccess), args.Error(1)
}

func (m *MockShareLinkAccessRepository) FindByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) ([]*entity.ShareLinkAccess, error) {
	args := m.Called(ctx, shareLinkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ShareLinkAccess), args.Error(1)
}

func (m *MockShareLinkAccessRepository) FindByShareLinkIDWithPagination(ctx context.Context, shareLinkID uuid.UUID, limit, offset int) ([]*entity.ShareLinkAccess, error) {
	args := m.Called(ctx, shareLinkID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ShareLinkAccess), args.Error(1)
}

func (m *MockShareLinkAccessRepository) CountByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) (int, error) {
	args := m.Called(ctx, shareLinkID)
	return args.Int(0), args.Error(1)
}

func (m *MockShareLinkAccessRepository) DeleteByShareLinkID(ctx context.Context, shareLinkID uuid.UUID) error {
	args := m.Called(ctx, shareLinkID)
	return args.Error(0)
}

func (m *MockShareLinkAccessRepository) AnonymizeOldAccesses(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
