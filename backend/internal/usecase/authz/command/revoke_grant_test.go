package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/authz/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type revokeGrantTestDeps struct {
	permissionGrantRepo *mocks.MockPermissionGrantRepository
	permissionResolver  *mocks.MockPermissionResolver
}

func newRevokeGrantTestDeps(t *testing.T) *revokeGrantTestDeps {
	t.Helper()
	return &revokeGrantTestDeps{
		permissionGrantRepo: mocks.NewMockPermissionGrantRepository(t),
		permissionResolver:  mocks.NewMockPermissionResolver(t),
	}
}

func (d *revokeGrantTestDeps) newCommand() *command.RevokeGrantCommand {
	return command.NewRevokeGrantCommand(d.permissionGrantRepo, d.permissionResolver)
}

func newGrant(resourceType authz.ResourceType, resourceID uuid.UUID, role authz.Role) *authz.PermissionGrant {
	return authz.ReconstructPermissionGrant(
		uuid.New(), resourceType, resourceID,
		authz.GranteeTypeUser, uuid.New(),
		role, uuid.New(), time.Now(),
	)
}

func TestRevokeGrantCommand_Execute_ValidGrant_GrantDeleted(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeGrantTestDeps(t)

	resourceID := uuid.New()
	revokedBy := uuid.New()
	grant := newGrant(authz.ResourceTypeFile, resourceID, authz.RoleContributor)

	input := command.RevokeGrantInput{
		GrantID:   grant.ID,
		RevokedBy: revokedBy,
	}

	deps.permissionGrantRepo.On("FindByID", ctx, grant.ID).Return(grant, nil)
	deps.permissionResolver.On("CanGrantRole", ctx, revokedBy, authz.ResourceTypeFile, resourceID, authz.RoleContributor).
		Return(true, nil)
	deps.permissionGrantRepo.On("Delete", ctx, grant.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, grant.ID, output.RevokedGrantID)
}

func TestRevokeGrantCommand_Execute_OwnerGrant_ReturnsBadRequest(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeGrantTestDeps(t)

	resourceID := uuid.New()
	ownerGrant := newGrant(authz.ResourceTypeFolder, resourceID, authz.RoleOwner)

	input := command.RevokeGrantInput{
		GrantID:   ownerGrant.ID,
		RevokedBy: uuid.New(),
	}

	deps.permissionGrantRepo.On("FindByID", ctx, ownerGrant.ID).Return(ownerGrant, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestRevokeGrantCommand_Execute_Unauthorized_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newRevokeGrantTestDeps(t)

	resourceID := uuid.New()
	revokedBy := uuid.New()
	grant := newGrant(authz.ResourceTypeFile, resourceID, authz.RoleViewer)

	input := command.RevokeGrantInput{
		GrantID:   grant.ID,
		RevokedBy: revokedBy,
	}

	deps.permissionGrantRepo.On("FindByID", ctx, grant.ID).Return(grant, nil)
	deps.permissionResolver.On("CanGrantRole", ctx, revokedBy, authz.ResourceTypeFile, resourceID, authz.RoleViewer).
		Return(false, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
