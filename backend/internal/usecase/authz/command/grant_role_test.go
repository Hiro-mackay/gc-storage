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

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/authz/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type grantRoleTestDeps struct {
	permissionGrantRepo *mocks.MockPermissionGrantRepository
	permissionResolver  *mocks.MockPermissionResolver
}

func newGrantRoleTestDeps(t *testing.T) *grantRoleTestDeps {
	t.Helper()
	return &grantRoleTestDeps{
		permissionGrantRepo: mocks.NewMockPermissionGrantRepository(t),
		permissionResolver:  mocks.NewMockPermissionResolver(t),
	}
}

func (d *grantRoleTestDeps) newCommand() *command.GrantRoleCommand {
	return command.NewGrantRoleCommand(d.permissionGrantRepo, d.permissionResolver)
}

func newExistingGrant(resourceType authz.ResourceType, resourceID uuid.UUID, role authz.Role) *authz.PermissionGrant {
	return authz.ReconstructPermissionGrant(
		uuid.New(), resourceType, resourceID,
		authz.GranteeTypeUser, uuid.New(),
		role, uuid.New(), time.Now(),
	)
}

func TestGrantRoleCommand_Execute_ContributorRole_GrantCreated(t *testing.T) {
	ctx := context.Background()
	deps := newGrantRoleTestDeps(t)

	resourceID := uuid.New()
	grantedBy := uuid.New()
	granteeID := uuid.New()

	input := command.GrantRoleInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		GranteeType:  "user",
		GranteeID:    granteeID,
		Role:         "contributor",
		GrantedBy:    grantedBy,
	}

	deps.permissionResolver.On("CanGrantRole", ctx, grantedBy, authz.ResourceTypeFile, resourceID, authz.RoleContributor).
		Return(true, nil)
	deps.permissionGrantRepo.On("FindByResourceGranteeAndRole", ctx, authz.ResourceTypeFile, resourceID, authz.GranteeTypeUser, granteeID, authz.RoleContributor).
		Return(nil, errors.New("not found"))
	deps.permissionGrantRepo.On("Create", ctx, mock.AnythingOfType("*authz.PermissionGrant")).
		Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	require.NotNil(t, output.Grant)
	assert.Equal(t, authz.RoleContributor, output.Grant.Role)
	assert.Equal(t, authz.ResourceTypeFile, output.Grant.ResourceType)
	assert.Equal(t, resourceID, output.Grant.ResourceID)
}

func TestGrantRoleCommand_Execute_OwnerRole_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newGrantRoleTestDeps(t)

	input := command.GrantRoleInput{
		ResourceType: "file",
		ResourceID:   uuid.New(),
		GranteeType:  "user",
		GranteeID:    uuid.New(),
		Role:         "owner",
		GrantedBy:    uuid.New(),
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestGrantRoleCommand_Execute_HigherThanOwnRole_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newGrantRoleTestDeps(t)

	resourceID := uuid.New()
	grantedBy := uuid.New()

	input := command.GrantRoleInput{
		ResourceType: "folder",
		ResourceID:   resourceID,
		GranteeType:  "user",
		GranteeID:    uuid.New(),
		Role:         "contributor",
		GrantedBy:    grantedBy,
	}

	deps.permissionResolver.On("CanGrantRole", ctx, grantedBy, authz.ResourceTypeFolder, resourceID, authz.RoleContributor).
		Return(false, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestGrantRoleCommand_Execute_DuplicateRole_ReturnsConflict(t *testing.T) {
	ctx := context.Background()
	deps := newGrantRoleTestDeps(t)

	resourceID := uuid.New()
	grantedBy := uuid.New()
	granteeID := uuid.New()
	existing := newExistingGrant(authz.ResourceTypeFile, resourceID, authz.RoleViewer)

	input := command.GrantRoleInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		GranteeType:  "user",
		GranteeID:    granteeID,
		Role:         "viewer",
		GrantedBy:    grantedBy,
	}

	deps.permissionResolver.On("CanGrantRole", ctx, grantedBy, authz.ResourceTypeFile, resourceID, authz.RoleViewer).
		Return(true, nil)
	deps.permissionGrantRepo.On("FindByResourceGranteeAndRole", ctx, authz.ResourceTypeFile, resourceID, authz.GranteeTypeUser, granteeID, authz.RoleViewer).
		Return(existing, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}
