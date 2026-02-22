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

type acceptInvitationTestDeps struct {
	invitationRepo *mocks.MockInvitationRepository
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
	userRepo       *mocks.MockUserRepository
	txManager      *mocks.MockTransactionManager
}

func newAcceptInvitationTestDeps(t *testing.T) *acceptInvitationTestDeps {
	t.Helper()
	return &acceptInvitationTestDeps{
		invitationRepo: mocks.NewMockInvitationRepository(t),
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
		userRepo:       mocks.NewMockUserRepository(t),
		txManager:      mocks.NewMockTransactionManager(t),
	}
}

func (d *acceptInvitationTestDeps) newCommand() *command.AcceptInvitationCommand {
	return command.NewAcceptInvitationCommand(
		d.invitationRepo,
		d.groupRepo,
		d.membershipRepo,
		d.userRepo,
		d.txManager,
	)
}

func newPendingInvitation(groupID uuid.UUID, email string, inviterID uuid.UUID) *entity.Invitation {
	vo, _ := valueobject.NewEmail(email)
	return entity.ReconstructInvitation(
		uuid.New(),
		groupID,
		vo,
		"valid-token",
		valueobject.GroupRoleViewer,
		inviterID,
		time.Now().Add(24*time.Hour),
		valueobject.InvitationStatusPending,
		time.Now(),
	)
}

func TestAcceptInvitationCommand_Execute_Success_MembershipCreated(t *testing.T) {
	ctx := context.Background()
	deps := newAcceptInvitationTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()
	token := "valid-token"
	email := "user@example.com"

	invitation := newPendingInvitation(groupID, email, uuid.New())
	invitation.Token = token

	user := newTestUser(email)
	user.ID = userID

	group := newTestGroup(uuid.New())
	group.ID = groupID

	deps.invitationRepo.On("FindByToken", ctx, token).Return(invitation, nil)
	deps.userRepo.On("FindByID", ctx, userID).Return(user, nil)
	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("Exists", ctx, groupID, userID).Return(false, nil)
	deps.invitationRepo.On("Update", ctx, mock.AnythingOfType("*entity.Invitation")).Return(nil)
	deps.membershipRepo.On("Create", ctx, mock.AnythingOfType("*entity.Membership")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.AcceptInvitationInput{
		Token:  token,
		UserID: userID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.NotNil(t, output.Membership)
	assert.Equal(t, groupID, output.Membership.GroupID)
	assert.Equal(t, userID, output.Membership.UserID)
}

func TestAcceptInvitationCommand_Execute_ExpiredInvitation_ValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newAcceptInvitationTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()
	token := "expired-token"
	email := "user@example.com"

	emailVO, _ := valueobject.NewEmail(email)
	expiredInvitation := entity.ReconstructInvitation(
		uuid.New(),
		groupID,
		emailVO,
		token,
		valueobject.GroupRoleViewer,
		uuid.New(),
		time.Now().Add(-1*time.Hour), // expired
		valueobject.InvitationStatusPending,
		time.Now().Add(-48*time.Hour),
	)

	deps.invitationRepo.On("FindByToken", ctx, token).Return(expiredInvitation, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.AcceptInvitationInput{
		Token:  token,
		UserID: userID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestAcceptInvitationCommand_Execute_WrongEmail_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newAcceptInvitationTestDeps(t)

	userID := uuid.New()
	groupID := uuid.New()
	token := "valid-token"

	invitation := newPendingInvitation(groupID, "invited@example.com", uuid.New())
	invitation.Token = token

	wrongUser := newTestUser("other@example.com")
	wrongUser.ID = userID

	deps.invitationRepo.On("FindByToken", ctx, token).Return(invitation, nil)
	deps.userRepo.On("FindByID", ctx, userID).Return(wrongUser, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.AcceptInvitationInput{
		Token:  token,
		UserID: userID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
