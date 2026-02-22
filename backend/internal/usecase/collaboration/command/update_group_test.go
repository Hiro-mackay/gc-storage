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

type updateGroupTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
}

func newUpdateGroupTestDeps(t *testing.T) *updateGroupTestDeps {
	t.Helper()
	return &updateGroupTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
	}
}

func (d *updateGroupTestDeps) newCommand() *command.UpdateGroupCommand {
	return command.NewUpdateGroupCommand(d.groupRepo, d.membershipRepo)
}

func TestUpdateGroupCommand_Execute_OwnerUpdatesName_Success(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateGroupTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)
	newName := "New Name"

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.groupRepo.On("Update", ctx, group).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateGroupInput{
		GroupID:   groupID,
		Name:      &newName,
		UpdatedBy: ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, "New Name", output.Group.Name.String())
}

func TestUpdateGroupCommand_Execute_NonOwnerUpdates_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateGroupTestDeps(t)

	contributorID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID
	contributorMembership := newTestMembership(groupID, contributorID, valueobject.GroupRoleContributor)
	newName := "New Name"

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, contributorID).Return(contributorMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateGroupInput{
		GroupID:   groupID,
		Name:      &newName,
		UpdatedBy: contributorID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestUpdateGroupCommand_Execute_ViewerUpdates_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateGroupTestDeps(t)

	viewerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID
	viewerMembership := newTestMembership(groupID, viewerID, valueobject.GroupRoleViewer)
	newName := "New Name"

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, viewerID).Return(viewerMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateGroupInput{
		GroupID:   groupID,
		Name:      &newName,
		UpdatedBy: viewerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestUpdateGroupCommand_Execute_GroupNotFound_NotFoundError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateGroupTestDeps(t)

	groupID := uuid.New()
	newName := "New Name"

	deps.groupRepo.On("FindByID", ctx, groupID).Return(nil, errors.New("not found"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.UpdateGroupInput{
		GroupID:   groupID,
		Name:      &newName,
		UpdatedBy: uuid.New(),
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}
