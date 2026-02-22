package query_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type getGroupTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
}

func newGetGroupTestDeps(t *testing.T) *getGroupTestDeps {
	t.Helper()
	return &getGroupTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
	}
}

func (d *getGroupTestDeps) newQuery() *query.GetGroupQuery {
	return query.NewGetGroupQuery(d.groupRepo, d.membershipRepo)
}

func TestGetGroupQuery_Execute_MemberRequests_ReturnsGroupWithMemberCount(t *testing.T) {
	ctx := context.Background()
	deps := newGetGroupTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	membership := newTestMembership(groupID, userID, valueobject.GroupRoleViewer)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, userID).Return(membership, nil)
	deps.membershipRepo.On("CountByGroupID", ctx, groupID).Return(3, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetGroupInput{
		GroupID: groupID,
		UserID:  userID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, group, output.Group)
	assert.Equal(t, membership, output.Membership)
	assert.Equal(t, 3, output.MemberCount)
}

func TestGetGroupQuery_Execute_NonMemberRequests_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newGetGroupTestDeps(t)

	nonMemberID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, nonMemberID).Return(nil, errors.New("not found"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetGroupInput{
		GroupID: groupID,
		UserID:  nonMemberID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestGetGroupQuery_Execute_GroupNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newGetGroupTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()

	deps.groupRepo.On("FindByID", ctx, groupID).Return(nil, errors.New("not found"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.GetGroupInput{
		GroupID: groupID,
		UserID:  userID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
}
