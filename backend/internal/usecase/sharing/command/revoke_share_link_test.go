package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type revokeShareLinkTestDeps struct {
	shareLinkRepo      *mocks.MockShareLinkRepository
	permissionResolver *mocks.MockPermissionResolver
}

func newRevokeShareLinkTestDeps(t *testing.T) *revokeShareLinkTestDeps {
	t.Helper()
	return &revokeShareLinkTestDeps{
		shareLinkRepo:      mocks.NewMockShareLinkRepository(t),
		permissionResolver: mocks.NewMockPermissionResolver(t),
	}
}

func (d *revokeShareLinkTestDeps) newCommand() *command.RevokeShareLinkCommand {
	return command.NewRevokeShareLinkCommand(d.shareLinkRepo, d.permissionResolver)
}

func buildActiveShareLinkWithType(createdBy uuid.UUID, resourceType authz.ResourceType) *entity.ShareLink {
	token, _ := valueobject.NewShareToken()
	perm, _ := valueobject.NewSharePermission("read")
	now := time.Now()
	return &entity.ShareLink{
		ID:           uuid.New(),
		Token:        token,
		ResourceType: resourceType,
		ResourceID:   uuid.New(),
		CreatedBy:    createdBy,
		Permission:   perm,
		Status:       valueobject.ShareLinkStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestRevokeShareLinkCommand_Execute_CreatorRevokes_Succeeds(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeShareLinkTestDeps(t)
	userID := uuid.New()
	shareLink := buildActiveFileShareLink(userID)

	input := command.RevokeShareLinkInput{
		ShareLinkID: shareLink.ID,
		RevokedBy:   userID,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, shareLink.ID, output.RevokedShareLinkID)
}

func TestRevokeShareLinkCommand_Execute_NonCreatorWithPermission_Succeeds(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeShareLinkTestDeps(t)
	creatorID := uuid.New()
	otherUserID := uuid.New()
	shareLink := buildActiveFileShareLink(creatorID)

	input := command.RevokeShareLinkInput{
		ShareLinkID: shareLink.ID,
		RevokedBy:   otherUserID,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.permissionResolver.On("HasPermission", ctx, otherUserID, authz.ResourceTypeFile, shareLink.ResourceID, authz.PermFileShare).Return(true, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, shareLink.ID, output.RevokedShareLinkID)
}

func TestRevokeShareLinkCommand_Execute_NonCreatorWithoutPermission_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeShareLinkTestDeps(t)
	creatorID := uuid.New()
	otherUserID := uuid.New()
	shareLink := buildActiveFileShareLink(creatorID)

	input := command.RevokeShareLinkInput{
		ShareLinkID: shareLink.ID,
		RevokedBy:   otherUserID,
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

func TestRevokeShareLinkCommand_Execute_AlreadyRevoked_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeShareLinkTestDeps(t)
	userID := uuid.New()
	shareLink := buildActiveFileShareLink(userID)
	shareLink.Revoke()

	input := command.RevokeShareLinkInput{
		ShareLinkID: shareLink.ID,
		RevokedBy:   userID,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)

	cmd := deps.newCommand()
	_, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestRevokeShareLinkCommand_Execute_ShareLinkNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeShareLinkTestDeps(t)
	nonExistentID := uuid.New()

	input := command.RevokeShareLinkInput{
		ShareLinkID: nonExistentID,
		RevokedBy:   uuid.New(),
	}

	deps.shareLinkRepo.On("FindByID", ctx, nonExistentID).Return(nil, apperror.NewNotFoundError("share link"))

	cmd := deps.newCommand()
	_, err := cmd.Execute(ctx, input)

	require.Error(t, err)
}

func TestRevokeShareLinkCommand_Execute_FolderShareLink_ChecksFolderPermission(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeShareLinkTestDeps(t)
	creatorID := uuid.New()
	otherUserID := uuid.New()
	shareLink := buildActiveShareLinkWithType(creatorID, authz.ResourceTypeFolder)

	input := command.RevokeShareLinkInput{
		ShareLinkID: shareLink.ID,
		RevokedBy:   otherUserID,
	}

	deps.shareLinkRepo.On("FindByID", ctx, shareLink.ID).Return(shareLink, nil)
	deps.permissionResolver.On("HasPermission", ctx, otherUserID, authz.ResourceTypeFolder, shareLink.ResourceID, authz.PermFolderShare).Return(true, nil)
	deps.shareLinkRepo.On("Update", ctx, shareLink).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, shareLink.ID, output.RevokedShareLinkID)
}
