package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// MockGroupRepository is a mock of repository.GroupRepository
type MockGroupRepository struct {
	mock.Mock
}

func NewMockGroupRepository(t *testing.T) *MockGroupRepository {
	m := &MockGroupRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockGroupRepository) Create(ctx context.Context, group *entity.Group) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockGroupRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Group, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Group), args.Error(1)
}

func (m *MockGroupRepository) Update(ctx context.Context, group *entity.Group) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *MockGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGroupRepository) FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*entity.Group, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Group), args.Error(1)
}

func (m *MockGroupRepository) FindByMemberID(ctx context.Context, userID uuid.UUID) ([]*entity.Group, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Group), args.Error(1)
}

func (m *MockGroupRepository) FindActiveByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*entity.Group, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Group), args.Error(1)
}

func (m *MockGroupRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// MockMembershipRepository is a mock of repository.MembershipRepository
type MockMembershipRepository struct {
	mock.Mock
}

func NewMockMembershipRepository(t *testing.T) *MockMembershipRepository {
	m := &MockMembershipRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockMembershipRepository) Create(ctx context.Context, membership *entity.Membership) error {
	args := m.Called(ctx, membership)
	return args.Error(0)
}

func (m *MockMembershipRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Membership, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Membership), args.Error(1)
}

func (m *MockMembershipRepository) Update(ctx context.Context, membership *entity.Membership) error {
	args := m.Called(ctx, membership)
	return args.Error(0)
}

func (m *MockMembershipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMembershipRepository) FindByGroupID(ctx context.Context, groupID uuid.UUID) ([]*entity.Membership, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Membership), args.Error(1)
}

func (m *MockMembershipRepository) FindByGroupIDWithUsers(ctx context.Context, groupID uuid.UUID) ([]*entity.MembershipWithUser, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.MembershipWithUser), args.Error(1)
}

func (m *MockMembershipRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Membership, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Membership), args.Error(1)
}

func (m *MockMembershipRepository) FindByGroupAndUser(ctx context.Context, groupID, userID uuid.UUID) (*entity.Membership, error) {
	args := m.Called(ctx, groupID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Membership), args.Error(1)
}

func (m *MockMembershipRepository) Exists(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, groupID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMembershipRepository) CountByGroupID(ctx context.Context, groupID uuid.UUID) (int, error) {
	args := m.Called(ctx, groupID)
	return args.Int(0), args.Error(1)
}

func (m *MockMembershipRepository) DeleteByGroupID(ctx context.Context, groupID uuid.UUID) error {
	args := m.Called(ctx, groupID)
	return args.Error(0)
}

// MockInvitationRepository is a mock of repository.InvitationRepository
type MockInvitationRepository struct {
	mock.Mock
}

func NewMockInvitationRepository(t *testing.T) *MockInvitationRepository {
	m := &MockInvitationRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockInvitationRepository) Create(ctx context.Context, invitation *entity.Invitation) error {
	args := m.Called(ctx, invitation)
	return args.Error(0)
}

func (m *MockInvitationRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Invitation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Invitation), args.Error(1)
}

func (m *MockInvitationRepository) Update(ctx context.Context, invitation *entity.Invitation) error {
	args := m.Called(ctx, invitation)
	return args.Error(0)
}

func (m *MockInvitationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInvitationRepository) FindByToken(ctx context.Context, token string) (*entity.Invitation, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Invitation), args.Error(1)
}

func (m *MockInvitationRepository) FindPendingByGroupID(ctx context.Context, groupID uuid.UUID) ([]*entity.Invitation, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Invitation), args.Error(1)
}

func (m *MockInvitationRepository) FindPendingByEmail(ctx context.Context, email valueobject.Email) ([]*entity.Invitation, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Invitation), args.Error(1)
}

func (m *MockInvitationRepository) FindPendingByGroupAndEmail(ctx context.Context, groupID uuid.UUID, email valueobject.Email) (*entity.Invitation, error) {
	args := m.Called(ctx, groupID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Invitation), args.Error(1)
}

func (m *MockInvitationRepository) DeleteByGroupID(ctx context.Context, groupID uuid.UUID) error {
	args := m.Called(ctx, groupID)
	return args.Error(0)
}

func (m *MockInvitationRepository) ExpireOld(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
