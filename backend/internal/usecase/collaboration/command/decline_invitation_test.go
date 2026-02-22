package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/command"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type declineInvitationTestDeps struct {
	invitationRepo *mocks.MockInvitationRepository
	userRepo       *mocks.MockUserRepository
}

func newDeclineInvitationTestDeps(t *testing.T) *declineInvitationTestDeps {
	t.Helper()
	return &declineInvitationTestDeps{
		invitationRepo: mocks.NewMockInvitationRepository(t),
		userRepo:       mocks.NewMockUserRepository(t),
	}
}

func (d *declineInvitationTestDeps) newCommand() *command.DeclineInvitationCommand {
	return command.NewDeclineInvitationCommand(
		d.invitationRepo,
		d.userRepo,
	)
}

func TestDeclineInvitationCommand_Execute_Success_StatusDeclined(t *testing.T) {
	ctx := context.Background()
	deps := newDeclineInvitationTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()
	token := "decline-token"
	email := "user@example.com"

	emailVO, _ := valueobject.NewEmail(email)
	invitation := entity.ReconstructInvitation(
		uuid.New(),
		groupID,
		emailVO,
		token,
		valueobject.GroupRoleViewer,
		uuid.New(),
		time.Now().Add(24*time.Hour),
		valueobject.InvitationStatusPending,
		time.Now(),
	)

	user := newTestUser(email)
	user.ID = userID

	deps.invitationRepo.On("FindByToken", ctx, token).Return(invitation, nil)
	deps.userRepo.On("FindByID", ctx, userID).Return(user, nil)
	deps.invitationRepo.On("Update", ctx, mock.AnythingOfType("*entity.Invitation")).Return(nil).Run(func(args mock.Arguments) {
		inv := args.Get(1).(*entity.Invitation)
		assert.True(t, inv.Status.IsDeclined())
	})

	cmd := deps.newCommand()
	err := cmd.Execute(ctx, command.DeclineInvitationInput{
		Token:  token,
		UserID: userID,
	})

	require.NoError(t, err)
}
