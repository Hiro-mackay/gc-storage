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

type inviteMemberTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
	invitationRepo *mocks.MockInvitationRepository
	userRepo       *mocks.MockUserRepository
	emailSender    *mocks.MockEmailSender
}

func newInviteMemberTestDeps(t *testing.T) *inviteMemberTestDeps {
	t.Helper()
	return &inviteMemberTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
		invitationRepo: mocks.NewMockInvitationRepository(t),
		userRepo:       mocks.NewMockUserRepository(t),
		emailSender:    mocks.NewMockEmailSender(t),
	}
}

func (d *inviteMemberTestDeps) newCommand() *command.InviteMemberCommand {
	return command.NewInviteMemberCommand(
		d.groupRepo,
		d.membershipRepo,
		d.invitationRepo,
		d.userRepo,
		d.emailSender,
		"http://localhost:3000",
	)
}

func newTestGroup(ownerID uuid.UUID) *entity.Group {
	name, _ := valueobject.NewGroupName("Test Group")
	return entity.ReconstructGroup(uuid.New(), name, "", ownerID, time.Now(), time.Now())
}

func newTestMembership(groupID, userID uuid.UUID, role valueobject.GroupRole) *entity.Membership {
	return entity.ReconstructMembership(uuid.New(), groupID, userID, role, time.Now())
}

func newTestUser(email string) *entity.User {
	vo, _ := valueobject.NewEmail(email)
	return &entity.User{
		ID:    uuid.New(),
		Email: vo,
		Name:  "Test User",
	}
}

func TestInviteMemberCommand_Execute_OwnerInvitesViewer_Success(t *testing.T) {
	ctx := context.Background()
	deps := newInviteMemberTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)
	invitedEmail := "viewer@example.com"

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(nil, errors.New("not found"))
	deps.invitationRepo.On("FindPendingByGroupAndEmail", ctx, groupID, mock.AnythingOfType("valueobject.Email")).Return(nil, errors.New("not found"))
	deps.invitationRepo.On("Create", ctx, mock.AnythingOfType("*entity.Invitation")).Return(nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(newTestUser("owner@example.com"), nil)
	deps.emailSender.On("SendGroupInvitation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.InviteMemberInput{
		GroupID:   groupID,
		Email:     invitedEmail,
		Role:      "viewer",
		InvitedBy: ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.NotNil(t, output.Invitation)
}

func TestInviteMemberCommand_Execute_OwnerRoleSpecified_ValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newInviteMemberTestDeps(t)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.InviteMemberInput{
		GroupID:   uuid.New(),
		Email:     "someone@example.com",
		Role:      "owner",
		InvitedBy: uuid.New(),
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestInviteMemberCommand_Execute_AlreadyMember_ConflictError(t *testing.T) {
	ctx := context.Background()
	deps := newInviteMemberTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)
	existingUser := newTestUser("existing@example.com")

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(existingUser, nil)
	deps.membershipRepo.On("Exists", ctx, groupID, existingUser.ID).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.InviteMemberInput{
		GroupID:   groupID,
		Email:     "existing@example.com",
		Role:      "viewer",
		InvitedBy: ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestInviteMemberCommand_Execute_DuplicateEmail_ConflictError(t *testing.T) {
	ctx := context.Background()
	deps := newInviteMemberTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID
	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)
	pendingEmail, _ := valueobject.NewEmail("pending@example.com")
	existingInvitation := entity.ReconstructInvitation(
		uuid.New(), groupID, pendingEmail, "token", valueobject.GroupRoleViewer,
		ownerID, time.Now().Add(24*time.Hour), valueobject.InvitationStatusPending, time.Now(),
	)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(nil, errors.New("not found"))
	deps.invitationRepo.On("FindPendingByGroupAndEmail", ctx, groupID, mock.AnythingOfType("valueobject.Email")).Return(existingInvitation, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.InviteMemberInput{
		GroupID:   groupID,
		Email:     "pending@example.com",
		Role:      "viewer",
		InvitedBy: ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestInviteMemberCommand_Execute_ContributorInvitesViewer_Success(t *testing.T) {
	ctx := context.Background()
	deps := newInviteMemberTestDeps(t)

	contributorID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID
	contributorMembership := newTestMembership(groupID, contributorID, valueobject.GroupRoleContributor)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, contributorID).Return(contributorMembership, nil)
	deps.userRepo.On("FindByEmail", ctx, mock.AnythingOfType("valueobject.Email")).Return(nil, errors.New("not found"))
	deps.invitationRepo.On("FindPendingByGroupAndEmail", ctx, groupID, mock.AnythingOfType("valueobject.Email")).Return(nil, errors.New("not found"))
	deps.invitationRepo.On("Create", ctx, mock.AnythingOfType("*entity.Invitation")).Return(nil)
	deps.userRepo.On("FindByID", ctx, contributorID).Return(newTestUser("contributor@example.com"), nil)
	deps.emailSender.On("SendGroupInvitation", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.InviteMemberInput{
		GroupID:   groupID,
		Email:     "newviewer@example.com",
		Role:      "viewer",
		InvitedBy: contributorID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.NotNil(t, output.Invitation)
}

func TestInviteMemberCommand_Execute_ContributorInvitesContributor_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newInviteMemberTestDeps(t)

	contributorID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID
	contributorMembership := newTestMembership(groupID, contributorID, valueobject.GroupRoleContributor)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, contributorID).Return(contributorMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.InviteMemberInput{
		GroupID:   groupID,
		Email:     "another@example.com",
		Role:      "contributor",
		InvitedBy: contributorID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestInviteMemberCommand_Execute_ViewerInvites_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newInviteMemberTestDeps(t)

	viewerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID
	viewerMembership := newTestMembership(groupID, viewerID, valueobject.GroupRoleViewer)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, viewerID).Return(viewerMembership, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.InviteMemberInput{
		GroupID:   groupID,
		Email:     "someone@example.com",
		Role:      "viewer",
		InvitedBy: viewerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
