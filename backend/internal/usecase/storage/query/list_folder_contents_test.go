package query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type listFolderContentsTestDeps struct {
	folderRepo *mocks.MockFolderRepository
	fileRepo   *mocks.MockFileRepository
}

func newListFolderContentsTestDeps(t *testing.T) *listFolderContentsTestDeps {
	t.Helper()
	return &listFolderContentsTestDeps{
		folderRepo: mocks.NewMockFolderRepository(t),
		fileRepo:   mocks.NewMockFileRepository(t),
	}
}

func (d *listFolderContentsTestDeps) newQuery() *query.ListFolderContentsQuery {
	return query.NewListFolderContentsQuery(d.folderRepo, d.fileRepo)
}

func newListContentFolderEntity(ownerID uuid.UUID) *entity.Folder {
	folderName, _ := valueobject.NewFolderName("parent-folder")
	return entity.ReconstructFolder(
		uuid.New(), folderName, nil, ownerID, ownerID, 0,
		entity.FolderStatusActive, time.Now(), time.Now(),
	)
}

func newListContentFileEntity(ownerID, folderID uuid.UUID, status entity.FileStatus) *entity.File {
	name, _ := valueobject.NewFileName("file.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(uuid.New())
	return entity.ReconstructFile(
		uuid.New(), folderID, ownerID, ownerID,
		name, mimeType, 512, storageKey, 1,
		status, time.Now(), time.Now(),
	)
}

func TestListFolderContentsQuery_Execute_RootLevel_ReturnsFoldersOnly(t *testing.T) {
	ctx := context.Background()
	deps := newListFolderContentsTestDeps(t)

	ownerID := uuid.New()
	subFolder := newListContentFolderEntity(ownerID)

	input := query.ListFolderContentsInput{
		FolderID: nil,
		OwnerID:  ownerID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindRootByOwner", ctx, ownerID).Return([]*entity.Folder{subFolder}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Nil(t, output.Folder)
	assert.Len(t, output.Folders, 1)
	assert.Empty(t, output.Files)
}

func TestListFolderContentsQuery_Execute_SpecificFolder_ReturnsFoldersAndFiles(t *testing.T) {
	ctx := context.Background()
	deps := newListFolderContentsTestDeps(t)

	ownerID := uuid.New()
	folder := newListContentFolderEntity(ownerID)
	subFolder := newListContentFolderEntity(ownerID)
	activeFile := newListContentFileEntity(ownerID, folder.ID, entity.FileStatusActive)

	folderID := folder.ID
	input := query.ListFolderContentsInput{
		FolderID: &folderID,
		OwnerID:  ownerID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folderID).Return(folder, nil)
	deps.folderRepo.On("FindByParentID", ctx, &folderID, ownerID).Return([]*entity.Folder{subFolder}, nil)
	deps.fileRepo.On("FindByFolderID", ctx, folderID).Return([]*entity.File{activeFile}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, folder.ID, output.Folder.ID)
	assert.Len(t, output.Folders, 1)
	assert.Len(t, output.Files, 1)
}

func TestListFolderContentsQuery_Execute_FiltersInactiveFiles(t *testing.T) {
	ctx := context.Background()
	deps := newListFolderContentsTestDeps(t)

	ownerID := uuid.New()
	folder := newListContentFolderEntity(ownerID)
	activeFile := newListContentFileEntity(ownerID, folder.ID, entity.FileStatusActive)
	uploadingFile := newListContentFileEntity(ownerID, folder.ID, entity.FileStatusUploading)

	folderID := folder.ID
	input := query.ListFolderContentsInput{
		FolderID: &folderID,
		OwnerID:  ownerID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folderID).Return(folder, nil)
	deps.folderRepo.On("FindByParentID", ctx, &folderID, ownerID).Return([]*entity.Folder{}, nil)
	deps.fileRepo.On("FindByFolderID", ctx, folderID).Return([]*entity.File{activeFile, uploadingFile}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Files, 1)
	assert.Equal(t, activeFile.ID, output.Files[0].ID)
}

func TestListFolderContentsQuery_Execute_FolderNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newListFolderContentsTestDeps(t)

	folderID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("folder")

	input := query.ListFolderContentsInput{
		FolderID: &folderID,
		OwnerID:  uuid.New(),
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

func TestListFolderContentsQuery_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newListFolderContentsTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folder := newListContentFolderEntity(ownerID)

	folderID := folder.ID
	input := query.ListFolderContentsInput{
		FolderID: &folderID,
		OwnerID:  ownerID,
		UserID:   differentUserID,
	}

	deps.folderRepo.On("FindByID", ctx, folderID).Return(folder, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestListFolderContentsQuery_Execute_RootLevelRepoError_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newListFolderContentsTestDeps(t)

	ownerID := uuid.New()
	repoErr := errors.New("database error")

	input := query.ListFolderContentsInput{
		FolderID: nil,
		OwnerID:  ownerID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindRootByOwner", ctx, ownerID).Return(nil, repoErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, repoErr, err)
}
