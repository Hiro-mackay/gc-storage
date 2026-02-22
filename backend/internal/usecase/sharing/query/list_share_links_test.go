package query_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type listShareLinksTestDeps struct {
	shareLinkRepo      *mocks.MockShareLinkRepository
	permissionResolver *mocks.MockPermissionResolver
}

func newListShareLinksTestDeps(t *testing.T) *listShareLinksTestDeps {
	t.Helper()
	return &listShareLinksTestDeps{
		shareLinkRepo:      mocks.NewMockShareLinkRepository(t),
		permissionResolver: mocks.NewMockPermissionResolver(t),
	}
}

func (d *listShareLinksTestDeps) newQuery() *query.ListShareLinksQuery {
	return query.NewListShareLinksQuery(d.shareLinkRepo, d.permissionResolver)
}

func TestListShareLinksQuery_Execute_ValidInput_ReturnsShareLinks(t *testing.T) {
	ctx := context.Background()
	deps := newListShareLinksTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()

	shareLinks := []*entity.ShareLink{
		buildTestShareLink(userID, authz.ResourceTypeFile),
		buildTestShareLink(userID, authz.ResourceTypeFile),
	}
	shareLinks[0].ResourceID = resourceID
	shareLinks[1].ResourceID = resourceID

	input := query.ListShareLinksInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		UserID:       userID,
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFile, resourceID, authz.PermFileShare).Return(true, nil)
	deps.shareLinkRepo.On("FindByResource", ctx, authz.ResourceTypeFile, resourceID).Return(shareLinks, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.ShareLinks, 2)
}

func TestListShareLinksQuery_Execute_NoPermission_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newListShareLinksTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()

	input := query.ListShareLinksInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		UserID:       userID,
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFile, resourceID, authz.PermFileShare).Return(false, nil)

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestListShareLinksQuery_Execute_InvalidResourceType_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newListShareLinksTestDeps(t)

	input := query.ListShareLinksInput{
		ResourceType: "invalid",
		ResourceID:   uuid.New(),
		UserID:       uuid.New(),
	}

	q := deps.newQuery()
	_, err := q.Execute(ctx, input)

	require.Error(t, err)
	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok)
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestListShareLinksQuery_Execute_FolderResource_ChecksFolderSharePermission(t *testing.T) {
	ctx := context.Background()
	deps := newListShareLinksTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()

	input := query.ListShareLinksInput{
		ResourceType: "folder",
		ResourceID:   resourceID,
		UserID:       userID,
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFolder, resourceID, authz.PermFolderShare).Return(true, nil)
	deps.shareLinkRepo.On("FindByResource", ctx, authz.ResourceTypeFolder, resourceID).Return([]*entity.ShareLink{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Empty(t, output.ShareLinks)
}

func TestListShareLinksQuery_Execute_EmptyResult_ReturnsEmptySlice(t *testing.T) {
	ctx := context.Background()
	deps := newListShareLinksTestDeps(t)
	userID := uuid.New()
	resourceID := uuid.New()

	input := query.ListShareLinksInput{
		ResourceType: "file",
		ResourceID:   resourceID,
		UserID:       userID,
	}

	deps.permissionResolver.On("HasPermission", ctx, userID, authz.ResourceTypeFile, resourceID, authz.PermFileShare).Return(true, nil)
	deps.shareLinkRepo.On("FindByResource", ctx, authz.ResourceTypeFile, resourceID).Return([]*entity.ShareLink{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	assert.Empty(t, output.ShareLinks)
}
