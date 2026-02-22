package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
)

// MockPermissionGrantRepository is a mock of authz.PermissionGrantRepository
type MockPermissionGrantRepository struct {
	mock.Mock
}

func NewMockPermissionGrantRepository(t *testing.T) *MockPermissionGrantRepository {
	m := &MockPermissionGrantRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockPermissionGrantRepository) Create(ctx context.Context, grant *authz.PermissionGrant) error {
	args := m.Called(ctx, grant)
	return args.Error(0)
}

func (m *MockPermissionGrantRepository) FindByID(ctx context.Context, id uuid.UUID) (*authz.PermissionGrant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authz.PermissionGrant), args.Error(1)
}

func (m *MockPermissionGrantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPermissionGrantRepository) FindByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*authz.PermissionGrant, error) {
	args := m.Called(ctx, resourceType, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*authz.PermissionGrant), args.Error(1)
}

func (m *MockPermissionGrantRepository) FindByResourceAndGrantee(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID, granteeType authz.GranteeType, granteeID uuid.UUID) ([]*authz.PermissionGrant, error) {
	args := m.Called(ctx, resourceType, resourceID, granteeType, granteeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*authz.PermissionGrant), args.Error(1)
}

func (m *MockPermissionGrantRepository) FindByResourceGranteeAndRole(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID, granteeType authz.GranteeType, granteeID uuid.UUID, role authz.Role) (*authz.PermissionGrant, error) {
	args := m.Called(ctx, resourceType, resourceID, granteeType, granteeID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authz.PermissionGrant), args.Error(1)
}

func (m *MockPermissionGrantRepository) FindByGrantee(ctx context.Context, granteeType authz.GranteeType, granteeID uuid.UUID) ([]*authz.PermissionGrant, error) {
	args := m.Called(ctx, granteeType, granteeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*authz.PermissionGrant), args.Error(1)
}

func (m *MockPermissionGrantRepository) DeleteByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) error {
	args := m.Called(ctx, resourceType, resourceID)
	return args.Error(0)
}

func (m *MockPermissionGrantRepository) DeleteByGrantee(ctx context.Context, granteeType authz.GranteeType, granteeID uuid.UUID) error {
	args := m.Called(ctx, granteeType, granteeID)
	return args.Error(0)
}

func (m *MockPermissionGrantRepository) DeleteByResourceAndGrantee(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID, granteeType authz.GranteeType, granteeID uuid.UUID) error {
	args := m.Called(ctx, resourceType, resourceID, granteeType, granteeID)
	return args.Error(0)
}
