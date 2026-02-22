package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type leaveGroupTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
}

func newLeaveGroupTestDeps(t *testing.T) *leaveGroupTestDeps {
	t.Helper()
	return &leaveGroupTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
	}
}

func (d *leaveGroupTestDeps) newCommand() *command.LeaveGroupCommand {
	return command.NewLeaveGroupCommand(d.groupRepo, d.membershipRepo)
}

func TestLeaveGroupCommand_Execute_ContributorLeaves_Success(t *testing.T) {
	ctx := context.Background()
	deps := newLeaveGroupTestDeps(t)

	ownerID := uuid.New()
	contributorID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	contributorMembership := newTestMembership(groupID, contributorID, valueobject.GroupRoleContributor)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, contributorID).Return(contributorMembership, nil)
	deps.membershipRepo.On("Delete", ctx, contributorMembership.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.LeaveGroupInput{
		GroupID: groupID,
		UserID:  contributorID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, groupID, output.LeftGroupID)
}

func TestLeaveGroupCommand_Execute_ViewerLeaves_Success(t *testing.T) {
	ctx := context.Background()
	deps := newLeaveGroupTestDeps(t)

	ownerID := uuid.New()
	viewerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	viewerMembership := newTestMembership(groupID, viewerID, valueobject.GroupRoleViewer)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, viewerID).Return(viewerMembership, nil)
	deps.membershipRepo.On("Delete", ctx, viewerMembership.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.LeaveGroupInput{
		GroupID: groupID,
		UserID:  viewerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, groupID, output.LeftGroupID)
}

func TestLeaveGroupCommand_Execute_OwnerCannotLeave_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newLeaveGroupTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.LeaveGroupInput{
		GroupID: groupID,
		UserID:  ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestLeaveGroupCommand_Execute_GroupNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newLeaveGroupTestDeps(t)

	groupID := uuid.New()
	deps.groupRepo.On("FindByID", ctx, groupID).Return(nil, errors.New("not found"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.LeaveGroupInput{
		GroupID: groupID,
		UserID:  uuid.New(),
	})

	require.Error(t, err)
	assert.Nil(t, output)
}
