package query_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type listMembersTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
}

func newListMembersTestDeps(t *testing.T) *listMembersTestDeps {
	t.Helper()
	return &listMembersTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
	}
}

func (d *listMembersTestDeps) newQuery() *query.ListMembersQuery {
	return query.NewListMembersQuery(d.groupRepo, d.membershipRepo)
}

func TestListMembersQuery_Execute_MemberRequests_ReturnsMembers(t *testing.T) {
	ctx := context.Background()
	deps := newListMembersTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	members := []*entity.MembershipWithUser{
		{Membership: newTestMembership(groupID, userID, valueobject.GroupRoleOwner), User: &entity.User{}},
		{Membership: newTestMembership(groupID, uuid.New(), valueobject.GroupRoleViewer), User: &entity.User{}},
	}

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("Exists", ctx, groupID, userID).Return(true, nil)
	deps.membershipRepo.On("FindByGroupIDWithUsers", ctx, groupID).Return(members, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListMembersInput{
		GroupID: groupID,
		UserID:  userID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Members, 2)
}

func TestListMembersQuery_Execute_NonMemberRequests_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newListMembersTestDeps(t)

	nonMemberID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("Exists", ctx, groupID, nonMemberID).Return(false, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListMembersInput{
		GroupID: groupID,
		UserID:  nonMemberID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestListMembersQuery_Execute_GroupNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newListMembersTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()

	deps.groupRepo.On("FindByID", ctx, groupID).Return(nil, errors.New("not found"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListMembersInput{
		GroupID: groupID,
		UserID:  userID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
}

func TestListMembersQuery_Execute_ExistsCheckError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newListMembersTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("Exists", ctx, groupID, userID).Return(false, errors.New("db error"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListMembersInput{
		GroupID: groupID,
		UserID:  userID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
}
