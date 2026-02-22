package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
)

// MockPermissionResolver is a mock of authz.PermissionResolver
type MockPermissionResolver struct {
	mock.Mock
}

func NewMockPermissionResolver(t *testing.T) *MockPermissionResolver {
	m := &MockPermissionResolver{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockPermissionResolver) HasPermission(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID, permission authz.Permission) (bool, error) {
	args := m.Called(ctx, userID, resourceType, resourceID, permission)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionResolver) CollectPermissions(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) (*authz.PermissionSet, error) {
	args := m.Called(ctx, userID, resourceType, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authz.PermissionSet), args.Error(1)
}

func (m *MockPermissionResolver) GetEffectiveRole(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) (authz.Role, error) {
	args := m.Called(ctx, userID, resourceType, resourceID)
	return args.Get(0).(authz.Role), args.Error(1)
}

func (m *MockPermissionResolver) IsOwner(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, resourceType, resourceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionResolver) CanGrantRole(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID, targetRole authz.Role) (bool, error) {
	args := m.Called(ctx, userID, resourceType, resourceID, targetRole)
	return args.Bool(0), args.Error(1)
}
