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

type transferOwnershipTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
	txManager      *mocks.MockTransactionManager
}

func newTransferOwnershipTestDeps(t *testing.T) *transferOwnershipTestDeps {
	t.Helper()
	return &transferOwnershipTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
		txManager:      mocks.NewMockTransactionManager(t),
	}
}

func (d *transferOwnershipTestDeps) newCommand() *command.TransferOwnershipCommand {
	return command.NewTransferOwnershipCommand(d.groupRepo, d.membershipRepo, d.txManager)
}

func TestTransferOwnershipCommand_Execute_OwnerTransfersToContributor_RolesSwapped(t *testing.T) {
	ctx := context.Background()
	deps := newTransferOwnershipTestDeps(t)

	ownerID := uuid.New()
	newOwnerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)
	newOwnerMembership := newTestMembership(groupID, newOwnerID, valueobject.GroupRoleContributor)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, newOwnerID).Return(newOwnerMembership, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.groupRepo.On("Update", ctx, group).Return(nil)
	deps.membershipRepo.On("Update", ctx, newOwnerMembership).Return(nil)
	deps.membershipRepo.On("Update", ctx, ownerMembership).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.TransferOwnershipInput{
		GroupID:        groupID,
		NewOwnerID:     newOwnerID,
		CurrentOwnerID: ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, valueobject.GroupRoleOwner, output.NewOwnerMembership.Role)
	assert.Equal(t, valueobject.GroupRoleContributor, output.OldOwnerMembership.Role)
	assert.Equal(t, newOwnerID, output.Group.OwnerID)
}

func TestTransferOwnershipCommand_Execute_NonOwnerTransfers_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newTransferOwnershipTestDeps(t)

	ownerID := uuid.New()
	nonOwnerID := uuid.New()
	newOwnerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.TransferOwnershipInput{
		GroupID:        groupID,
		NewOwnerID:     newOwnerID,
		CurrentOwnerID: nonOwnerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestTransferOwnershipCommand_Execute_TransferToSelf_ValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newTransferOwnershipTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.TransferOwnershipInput{
		GroupID:        groupID,
		NewOwnerID:     ownerID,
		CurrentOwnerID: ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestTransferOwnershipCommand_Execute_NewOwnerNotMember_ValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newTransferOwnershipTestDeps(t)

	ownerID := uuid.New()
	nonMemberID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, nonMemberID).Return(nil, errors.New("not found"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.TransferOwnershipInput{
		GroupID:        groupID,
		NewOwnerID:     nonMemberID,
		CurrentOwnerID: ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
