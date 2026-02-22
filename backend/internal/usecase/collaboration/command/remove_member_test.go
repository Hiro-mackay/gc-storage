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

type removeMemberTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
}

func newRemoveMemberTestDeps(t *testing.T) *removeMemberTestDeps {
	t.Helper()
	return &removeMemberTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
	}
}

func (d *removeMemberTestDeps) newCommand() *command.RemoveMemberCommand {
	return command.NewRemoveMemberCommand(d.groupRepo, d.membershipRepo)
}

func TestRemoveMemberCommand_Execute_OwnerRemovesViewer_Success(t *testing.T) {
	ctx := context.Background()
	deps := newRemoveMemberTestDeps(t)

	ownerID := uuid.New()
	viewerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)
	viewerMembership := newTestMembership(groupID, viewerID, valueobject.GroupRoleViewer)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, viewerID).Return(viewerMembership, nil)
	deps.membershipRepo.On("Delete", ctx, viewerMembership.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.RemoveMemberInput{
		GroupID:      groupID,
		TargetUserID: viewerID,
		RemovedBy:    ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, viewerID, output.RemovedUserID)
}

func TestRemoveMemberCommand_Execute_CannotRemoveOwner_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newRemoveMemberTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)
	ownerTargetMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerTargetMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.RemoveMemberInput{
		GroupID:      groupID,
		TargetUserID: ownerID,
		RemovedBy:    ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestRemoveMemberCommand_Execute_ViewerCannotRemoveMembers_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newRemoveMemberTestDeps(t)

	ownerID := uuid.New()
	viewerID := uuid.New()
	targetID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	viewerMembership := newTestMembership(groupID, viewerID, valueobject.GroupRoleViewer)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, viewerID).Return(viewerMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.RemoveMemberInput{
		GroupID:      groupID,
		TargetUserID: targetID,
		RemovedBy:    viewerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
