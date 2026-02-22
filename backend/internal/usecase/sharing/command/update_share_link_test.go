package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type updateShareLinkTestDeps struct {
	shareLinkRepo      *mocks.MockShareLinkRepository
	permissionResolver *mocks.MockPermissionResolver
}

func newUpdateShareLinkTestDeps(t *testing.T) *updateShareLinkTestDeps {
	t.Helper()
	return &updateShareLinkTestDeps{
		shareLinkRepo:      mocks.NewMockShareLinkRepository(t),
		permissionResolver: mocks.NewMockPermissionResolver(t),
	}
}

func (d *updateShareLinkTestDeps) newCommand() *command.UpdateShareLinkCommand {
	return command.NewUpdateShareLinkCommand(d.shareLinkRepo, d.permissionResolver)
}

func buildActiveFileShareLink(createdBy uuid.UUID) *entity.ShareLink {
	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	now := time.Now()
	return &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: authz.ResourceTypeFile,
		ResourceID:   uuid.New(),
		CreatedBy:    createdBy,
		Permission:   perm,
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestUpdateShareLinkCommand_Execute_ValidInput_UpdatesExpiryAndMaxAccess(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateShareLinkTestDeps(t)
	userID := uuid.New()
	shareLink := buildActiveFileShareLink(userID)

	newExpiry := time.Now().Add(24 * time.Hour)
	newMax := 10
	input := command.UpdateShareLinkInput{
		ShareLinkID:    shareLink.ID,
		UpdatedBy:      userID,
		ExpiresAt:      &newExpiry,
		MaxAccessCount: &newMax,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, shareLink.ID, output.ShareLink.ID)
	assert.Equal(t, &newExpiry, output.ShareLink.ExpiresAt)
	assert.Equal(t, &newMax, output.ShareLink.MaxAccessCount)
}

func TestUpdateShareLinkCommand_Execute_WithPassword_HashesPassword(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateShareLinkTestDeps(t)
	userID := uuid.New()
	shareLink := buildActiveFileShareLink(userID)

	newPassword := "newpassword123"
	input := command.UpdateShareLinkInput{
		ShareLinkID: shareLink.ID,
		UpdatedBy:   userID,
		Password:    &newPassword,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotEmpty(t, output.ShareLink.PasswordHash)
	err = bcrypt.CompareHashAndPassword([]byte(output.ShareLink.PasswordHash), []byte(newPassword))
	assert.NoError(t, err)
}

func TestUpdateShareLinkCommand_Execute_NonCreatorWithPermission_Succeeds(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateShareLinkTestDeps(t)
	creatorID := uuid.New()
	otherUserID := uuid.New()
	shareLink := buildActiveFileShareLink(creatorID)

	newMax := 5
	input := command.UpdateShareLinkInput{
		ShareLinkID:    shareLink.ID,
		UpdatedBy:      otherUserID,
		MaxAccessCount: &newMax,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.permissionResolver.On("HasPermission", ctx, otherUserID, authz.ResourceTypeFile, shareLink.ResourceID, authz.PermFileShare).Return(true, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
}

func TestUpdateShareLinkCommand_Execute_NonCreatorWithoutPermission_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateShareLinkTestDeps(t)
	creatorID := uuid.New()
	otherUserID := uuid.New()
	shareLink := buildActiveFileShareLink(creatorID)

	newMax := 5
	input := command.UpdateShareLinkInput{
		ShareLinkID:    shareLink.ID,
		UpdatedBy:      otherUserID,
		MaxAccessCount: &newMax,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.permissionResolver.On("HasPermission", ctx, otherUserID, authz.ResourceTypeFile, shareLink.ResourceID, authz.PermFileShare).Return(false, nil)

	cmd := deps.newCommand()
	_, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestUpdateShareLinkCommand_Execute_ShareLinkNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateShareLinkTestDeps(t)
	nonExistentID := uuid.New()

	input := command.UpdateShareLinkInput{
		ShareLinkID: nonExistentID,
		UpdatedBy:   uuid.New(),
	}

	deps.shareLinkRepo.On("FindByID", ctx, nonExistentID).Return(nil, apperror.NewNotFoundError("share link"))

	cmd := deps.newCommand()
	_, err := cmd.Execute(ctx, input)

	require.Error(t, err)
}

func TestUpdateShareLinkCommand_Execute_InactiveShareLink_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newUpdateShareLinkTestDeps(t)
	userID := uuid.New()
	shareLink := buildActiveFileShareLink(userID)
	shareLink.Revoke()

	newMax := 5
	input := command.UpdateShareLinkInput{
		ShareLinkID:    shareLink.ID,
		UpdatedBy:      userID,
		MaxAccessCount: &newMax,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)

	cmd := deps.newCommand()
	_, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
