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
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type listMyGroupsTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
}

func newListMyGroupsTestDeps(t *testing.T) *listMyGroupsTestDeps {
	t.Helper()
	return &listMyGroupsTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
	}
}

func (d *listMyGroupsTestDeps) newQuery() *query.ListMyGroupsQuery {
	return query.NewListMyGroupsQuery(d.groupRepo, d.membershipRepo)
}

func TestListMyGroupsQuery_Execute_UserWithGroups_ReturnsGroupsWithMemberships(t *testing.T) {
	ctx := context.Background()
	deps := newListMyGroupsTestDeps(t)

	userID := uuid.New()

	group1 := newTestGroup(userID)
	group1.ID = uuid.New()
	group2 := newTestGroup(uuid.New())
	group2.ID = uuid.New()
	groups := []*entity.Group{group1, group2}

	membership1 := newTestMembership(group1.ID, userID, valueobject.GroupRoleOwner)
	membership2 := newTestMembership(group2.ID, userID, valueobject.GroupRoleViewer)

	deps.groupRepo.On("FindByMemberID", ctx, userID).Return(groups, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, group1.ID, userID).Return(membership1, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, group2.ID, userID).Return(membership2, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListMyGroupsInput{UserID: userID})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Groups, 2)
	assert.Equal(t, group1, output.Groups[0].Group)
	assert.Equal(t, membership1, output.Groups[0].Membership)
	assert.Equal(t, group2, output.Groups[1].Group)
	assert.Equal(t, membership2, output.Groups[1].Membership)
}

func TestListMyGroupsQuery_Execute_UserWithNoGroups_ReturnsEmptyList(t *testing.T) {
	ctx := context.Background()
	deps := newListMyGroupsTestDeps(t)

	userID := uuid.New()

	deps.groupRepo.On("FindByMemberID", ctx, userID).Return([]*entity.Group{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListMyGroupsInput{UserID: userID})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output.Groups)
}

func TestListMyGroupsQuery_Execute_MembershipNotFound_SkipsGroup(t *testing.T) {
	ctx := context.Background()
	deps := newListMyGroupsTestDeps(t)

	userID := uuid.New()

	group1 := newTestGroup(userID)
	group1.ID = uuid.New()
	group2 := newTestGroup(uuid.New())
	group2.ID = uuid.New()
	groups := []*entity.Group{group1, group2}

	membership1 := newTestMembership(group1.ID, userID, valueobject.GroupRoleOwner)

	deps.groupRepo.On("FindByMemberID", ctx, userID).Return(groups, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, group1.ID, userID).Return(membership1, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, group2.ID, userID).Return(nil, errors.New("not found"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListMyGroupsInput{UserID: userID})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Groups, 1)
	assert.Equal(t, group1, output.Groups[0].Group)
}

func TestListMyGroupsQuery_Execute_RepositoryError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newListMyGroupsTestDeps(t)

	userID := uuid.New()

	deps.groupRepo.On("FindByMemberID", ctx, userID).Return(nil, errors.New("db error"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListMyGroupsInput{UserID: userID})

	require.Error(t, err)
	assert.Nil(t, output)
}
