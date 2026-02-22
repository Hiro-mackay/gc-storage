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

type changeRoleTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
}

func newChangeRoleTestDeps(t *testing.T) *changeRoleTestDeps {
	t.Helper()
	return &changeRoleTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
	}
}

func (d *changeRoleTestDeps) newCommand() *command.ChangeRoleCommand {
	return command.NewChangeRoleCommand(d.groupRepo, d.membershipRepo)
}

func TestChangeRoleCommand_Execute_OwnerChangesViewerToContributor_Success(t *testing.T) {
	ctx := context.Background()
	deps := newChangeRoleTestDeps(t)

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
	deps.membershipRepo.On("Update", ctx, viewerMembership).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.ChangeRoleInput{
		GroupID:      groupID,
		TargetUserID: viewerID,
		NewRole:      "contributor",
		ChangedBy:    ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, valueobject.GroupRoleContributor, output.Membership.Role)
}

func TestChangeRoleCommand_Execute_CannotChangeToOwner_ValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newChangeRoleTestDeps(t)

	ownerID := uuid.New()
	viewerID := uuid.New()
	groupID := uuid.New()

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.ChangeRoleInput{
		GroupID:      groupID,
		TargetUserID: viewerID,
		NewRole:      "owner",
		ChangedBy:    ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestChangeRoleCommand_Execute_CannotChangeSelfRole_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newChangeRoleTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.ChangeRoleInput{
		GroupID:      groupID,
		TargetUserID: ownerID,
		NewRole:      "contributor",
		ChangedBy:    ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestChangeRoleCommand_Execute_NonOwnerChangesRole_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newChangeRoleTestDeps(t)

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
	output, err := cmd.Execute(ctx, command.ChangeRoleInput{
		GroupID:      groupID,
		TargetUserID: targetID,
		NewRole:      "viewer",
		ChangedBy:    viewerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
