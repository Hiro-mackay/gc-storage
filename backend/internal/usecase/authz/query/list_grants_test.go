package query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/authz/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type listGrantsTestDeps struct {
	permissionGrantRepo *mocks.MockPermissionGrantRepository
	permissionResolver  *mocks.MockPermissionResolver
}

func newListGrantsTestDeps(t *testing.T) *listGrantsTestDeps {
	t.Helper()
	return &listGrantsTestDeps{
		permissionGrantRepo: mocks.NewMockPermissionGrantRepository(t),
		permissionResolver:  mocks.NewMockPermissionResolver(t),
	}
}

func (d *listGrantsTestDeps) newQuery() *query.ListGrantsQuery {
	return query.NewListGrantsQuery(d.permissionGrantRepo, d.permissionResolver)
}

func buildGrants(resourceType authz.ResourceType, resourceID uuid.UUID, n int) []*authz.PermissionGrant {
	grants := make([]*authz.PermissionGrant, n)
	for i := range grants {
		grants[i] = authz.ReconstructPermissionGrant(
			uuid.New(), resourceType, resourceID,
			authz.GranteeTypeUser, uuid.New(),
			authz.RoleViewer, uuid.New(), time.Now(),
		)
	}
	return grants
}

func TestListGrantsQuery_Execute_WithPermission_AllGrantsReturned(t *testing.T) {
	ctx := context.Background()
	deps := newListGrantsTestDeps(t)

	userID := uuid.New()
	resourceID := uuid.New()
	grants := buildGrants(authz.ResourceTypeFolder, resourceID, 3)

	input := query.ListGrantsInput{
		ResourceType: "folder",
		ResourceID:   resourceID,
		UserID:       userID,
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFolder, resourceID, authz.PermPermissionRead).
		Return(true, nil)
	deps.permissionGrantRepo.On("FindByResource", ctx, authz.ResourceTypeFolder, resourceID).
		Return(grants, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Grants, 3)
}

func TestListGrantsQuery_Execute_WithoutPermission_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newListGrantsTestDeps(t)

	userID := uuid.New()
	resourceID := uuid.New()

	input := query.ListGrantsInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		UserID:       userID,
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFile, resourceID, authz.PermPermissionRead).
		Return(false, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestListGrantsQuery_Execute_InvalidResourceType_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newListGrantsTestDeps(t)

	input := query.ListGrantsInput{
		ResourceType: "invalid_type",
		ResourceID:   uuid.New(),
		UserID:       uuid.New(),
	}

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}
