package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type cancelInvitationTestDeps struct {
	invitationRepo *mocks.MockInvitationRepository
	membershipRepo *mocks.MockMembershipRepository
	groupRepo      *mocks.MockGroupRepository
}

func newCancelInvitationTestDeps(t *testing.T) *cancelInvitationTestDeps {
	t.Helper()
	return &cancelInvitationTestDeps{
		invitationRepo: mocks.NewMockInvitationRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
		groupRepo:      mocks.NewMockGroupRepository(t),
	}
}

func (d *cancelInvitationTestDeps) newCommand() *command.CancelInvitationCommand {
	return command.NewCancelInvitationCommand(
		d.invitationRepo,
		d.membershipRepo,
		d.groupRepo,
	)
}

func buildPendingInvitation(groupID uuid.UUID, email string, inviterID uuid.UUID) *entity.Invitation {
	emailVO, _ := valueobject.NewEmail(email)
	return entity.ReconstructInvitation(
		uuid.New(),
		groupID,
		emailVO,
		"cancel-token",
		valueobject.GroupRoleViewer,
		inviterID,
		time.Now().Add(24*time.Hour),
		valueobject.InvitationStatusPending,
		time.Now(),
	)
}

func TestCancelInvitationCommand_Execute_OwnerCancels_Success(t *testing.T) {
	ctx := context.Background()
	deps := newCancelInvitationTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID

	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)
	invitation := buildPendingInvitation(groupID, "invited@example.com", ownerID)
	invitationID := invitation.ID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.invitationRepo.On("FindByID", ctx, invitationID).Return(invitation, nil)
	deps.invitationRepo.On("Update", ctx, mock.AnythingOfType("*entity.Invitation")).Return(nil).Run(func(args mock.Arguments) {
		inv := args.Get(1).(*entity.Invitation)
		assert.True(t, inv.Status.IsCancelled())
	})

	cmd := deps.newCommand()
	err := cmd.Execute(ctx, command.CancelInvitationInput{
		InvitationID: invitationID,
		GroupID:      groupID,
		CancelledBy:  ownerID,
	})

	require.NoError(t, err)
}

func TestCancelInvitationCommand_Execute_NonOwner_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newCancelInvitationTestDeps(t)

	contributorID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	contributorMembership := newTestMembership(groupID, contributorID, valueobject.GroupRoleContributor)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, contributorID).Return(contributorMembership, nil)

	cmd := deps.newCommand()
	err := cmd.Execute(ctx, command.CancelInvitationInput{
		InvitationID: uuid.New(),
		GroupID:      groupID,
		CancelledBy:  contributorID,
	})

	require.Error(t, err)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
