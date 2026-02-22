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
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type getFolderTestDeps struct {
	folderRepo         *mocks.MockFolderRepository
	permissionResolver *mocks.MockPermissionResolver
}

func newGetFolderTestDeps(t *testing.T) *getFolderTestDeps {
	t.Helper()
	return &getFolderTestDeps{
		folderRepo:         mocks.NewMockFolderRepository(t),
		permissionResolver: mocks.NewMockPermissionResolver(t),
	}
}

func (d *getFolderTestDeps) newQuery() *query.GetFolderQuery {
	return query.NewGetFolderQuery(d.folderRepo, d.permissionResolver)
}

func newQueryFolderEntity(ownerID uuid.UUID) *entity.Folder {
	folderName, _ := valueobject.NewFolderName("my-folder")
	return entity.ReconstructFolder(
		uuid.New(), folderName, nil, ownerID, ownerID, 0,
		entity.FolderStatusActive, time.Now(), time.Now(),
	)
}

func TestGetFolderQuery_Execute_WithPermission_ReturnsFolder(t *testing.T) {
	ctx := context.Background()
	deps := newGetFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newQueryFolderEntity(ownerID)

	input := query.GetFolderInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.permissionResolver.On("HasPermission", ctx, ownerID, authz.ResourceTypeFolder, folder.ID, authz.PermFolderRead).Return(true, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, folder.ID, output.Folder.ID)
}

func TestGetFolderQuery_Execute_FolderNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newGetFolderTestDeps(t)

	folderID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("folder")

	input := query.GetFolderInput{
		FolderID: folderID,
		UserID:   uuid.New(),
	}

	deps.folderRepo.On("FindByID", ctx, folderID).Return(nil, notFoundErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestGetFolderQuery_Execute_NoPermission_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newGetFolderTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folder := newQueryFolderEntity(ownerID)

	input := query.GetFolderInput{
		FolderID: folder.ID,
		UserID:   differentUserID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.permissionResolver.On("HasPermission", ctx, differentUserID, authz.ResourceTypeFolder, folder.ID, authz.PermFolderRead).Return(false, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestGetFolderQuery_Execute_PermissionResolverError_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newGetFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newQueryFolderEntity(ownerID)
	resolverErr := errors.New("resolver error")

	input := query.GetFolderInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.permissionResolver.On("HasPermission", ctx, ownerID, authz.ResourceTypeFolder, folder.ID, authz.PermFolderRead).Return(false, resolverErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, resolverErr, err)
}
