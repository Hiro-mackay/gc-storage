package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type createShareLinkTestDeps struct {
	shareLinkRepo      *mocks.MockShareLinkRepository
	permissionResolver *mocks.MockPermissionResolver
}

func newCreateShareLinkTestDeps(t *testing.T) *createShareLinkTestDeps {
	t.Helper()
	return &createShareLinkTestDeps{
		shareLinkRepo:      mocks.NewMockShareLinkRepository(t),
		permissionResolver: mocks.NewMockPermissionResolver(t),
	}
}

func (d *createShareLinkTestDeps) newCommand() *command.CreateShareLinkCommand {
	return command.NewCreateShareLinkCommand(d.shareLinkRepo, d.permissionResolver)
}

func TestCreateShareLinkCommand_Execute_ValidInput_ReturnsShareLink(t *testing.T) {
	ctx := context.Background()
	deps := newCreateShareLinkTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()

	input := command.CreateShareLinkInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		CreatedBy:    userID,
		Permission:   "read",
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFile, resourceID, authz.PermFileShare).Return(true, nil)
	deps.shareLinkRepo.On("Create", ctx, mockedShareLink()).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotNil(t, output.ShareLink)
	assert.Equal(t, resourceID, output.ShareLink.ResourceID)
}

func TestCreateShareLinkCommand_Execute_WithExpiry_SetsExpiresAt(t *testing.T) {
	ctx := context.Background()
	deps := newCreateShareLinkTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()
	expiry := time.Now().Add(24 * time.Hour)

	input := command.CreateShareLinkInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		CreatedBy:    userID,
		Permission:   "read",
		ExpiresAt:    &expiry,
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFile, resourceID, authz.PermFileShare).Return(true, nil)
	deps.shareLinkRepo.On("Create", ctx, mockedShareLink()).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output.ShareLink.ExpiresAt)
	assert.Equal(t, expiry, *output.ShareLink.ExpiresAt)
}

func TestCreateShareLinkCommand_Execute_WithPassword_HashesPassword(t *testing.T) {
	ctx := context.Background()
	deps := newCreateShareLinkTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()

	input := command.CreateShareLinkInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		CreatedBy:    userID,
		Permission:   "read",
		Password:     "secret123",
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFile, resourceID, authz.PermFileShare).Return(true, nil)
	deps.shareLinkRepo.On("Create", ctx, mockedShareLink()).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotEmpty(t, output.ShareLink.PasswordHash)
}

func TestCreateShareLinkCommand_Execute_InvalidResourceType_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newCreateShareLinkTestDeps(t)

	input := command.CreateShareLinkInput{
		ResourceType: "invalid",
		ResourceID:   uuid.New(),
		CreatedBy:    uuid.New(),
		Permission:   "read",
	}

	cmd := deps.newCommand()
	_, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestCreateShareLinkCommand_Execute_InvalidPermission_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newCreateShareLinkTestDeps(t)

	input := command.CreateShareLinkInput{
		ResourceType: "file",
		ResourceID:   uuid.New(),
		CreatedBy:    uuid.New(),
		Permission:   "admin",
	}

	cmd := deps.newCommand()
	_, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestCreateShareLinkCommand_Execute_NoSharePermission_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newCreateShareLinkTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()

	input := command.CreateShareLinkInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		CreatedBy:    userID,
		Permission:   "read",
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFile, resourceID, authz.PermFileShare).Return(false, nil)

	cmd := deps.newCommand()
	_, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestCreateShareLinkCommand_Execute_FolderResource_ChecksFolderSharePermission(t *testing.T) {
	ctx := context.Background()
	deps := newCreateShareLinkTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()

	input := command.CreateShareLinkInput{
		ResourceType: "folder",
		ResourceID:   resourceID,
		CreatedBy:    userID,
		Permission:   "write",
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFolder, resourceID, authz.PermFolderShare).Return(true, nil)
	deps.shareLinkRepo.On("Create", ctx, mockedShareLink()).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
}

// mockedShareLink returns a testify AnythingOfTypeArgument matcher for *entity.ShareLink
func mockedShareLink() interface{} {
	return mock.AnythingOfType("*entity.ShareLink")
}
