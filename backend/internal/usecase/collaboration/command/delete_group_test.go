package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type deleteGroupTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
	invitationRepo *mocks.MockInvitationRepository
	txManager      *mocks.MockTransactionManager
}

func newDeleteGroupTestDeps(t *testing.T) *deleteGroupTestDeps {
	t.Helper()
	return &deleteGroupTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
		invitationRepo: mocks.NewMockInvitationRepository(t),
		txManager:      mocks.NewMockTransactionManager(t),
	}
}

func (d *deleteGroupTestDeps) newCommand() *command.DeleteGroupCommand {
	return command.NewDeleteGroupCommand(d.groupRepo, d.membershipRepo, d.invitationRepo, d.txManager)
}

func TestDeleteGroupCommand_Execute_OwnerDeletes_CascadesAndSucceeds(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteGroupTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.invitationRepo.On("DeleteByGroupID", ctx, groupID).Return(nil)
	deps.membershipRepo.On("DeleteByGroupID", ctx, groupID).Return(nil)
	deps.groupRepo.On("Delete", ctx, groupID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.DeleteGroupInput{
		GroupID:   groupID,
		DeletedBy: ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, groupID, output.DeletedGroupID)
}

func TestDeleteGroupCommand_Execute_NonOwnerDeletes_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteGroupTestDeps(t)

	ownerID := uuid.New()
	nonOwnerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.DeleteGroupInput{
		GroupID:   groupID,
		DeletedBy: nonOwnerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestDeleteGroupCommand_Execute_GroupNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteGroupTestDeps(t)

	groupID := uuid.New()
	deps.groupRepo.On("FindByID", ctx, groupID).Return(nil, errors.New("not found"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.DeleteGroupInput{
		GroupID:   groupID,
		DeletedBy: uuid.New(),
	})

	require.Error(t, err)
	assert.Nil(t, output)
}
