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

type getAncestorsTestDeps struct {
	folderRepo        *mocks.MockFolderRepository
	folderClosureRepo *mocks.MockFolderClosureRepository
}

func newGetAncestorsTestDeps(t *testing.T) *getAncestorsTestDeps {
	t.Helper()
	return &getAncestorsTestDeps{
		folderRepo:        mocks.NewMockFolderRepository(t),
		folderClosureRepo: mocks.NewMockFolderClosureRepository(t),
	}
}

func (d *getAncestorsTestDeps) newQuery() *query.GetAncestorsQuery {
	return query.NewGetAncestorsQuery(d.folderRepo, d.folderClosureRepo)
}

func newAncestorFolderEntity(ownerID uuid.UUID, name string) *entity.Folder {
	folderName, _ := valueobject.NewFolderName(name)
	return entity.ReconstructFolder(
		uuid.New(), folderName, nil, ownerID, ownerID, 0,
		entity.FolderStatusActive, time.Now(), time.Now(),
	)
}

func TestGetAncestorsQuery_Execute_RootFolder_ReturnsEmptyAncestors(t *testing.T) {
	ctx := context.Background()
	deps := newGetAncestorsTestDeps(t)

	ownerID := uuid.New()
	folder := newAncestorFolderEntity(ownerID, "root-folder")

	input := query.GetAncestorsInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.folderClosureRepo.On("FindAncestorIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output.Ancestors)
}

func TestGetAncestorsQuery_Execute_ChildFolder_ReturnsAncestorsInRootFirstOrder(t *testing.T) {
	ctx := context.Background()
	deps := newGetAncestorsTestDeps(t)

	ownerID := uuid.New()
	rootFolder := newAncestorFolderEntity(ownerID, "root")
	parentFolder := newAncestorFolderEntity(ownerID, "parent")
	childFolder := newAncestorFolderEntity(ownerID, "child")

	// FindAncestorIDs returns in path_length ascending order (deepest ancestor last)
	// So for child -> parent -> root, it returns [root, parent] in path_length order
	// But the closure table returns them in ascending path_length: [parent(1), root(2)]
	// Then we reverse them to get root-first order: [root, parent]
	ancestorIDs := []uuid.UUID{parentFolder.ID, rootFolder.ID}

	input := query.GetAncestorsInput{
		FolderID: childFolder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, childFolder.ID).Return(childFolder, nil)
	deps.folderClosureRepo.On("FindAncestorIDs", ctx, childFolder.ID).Return(ancestorIDs, nil)
	deps.folderRepo.On("FindByID", ctx, parentFolder.ID).Return(parentFolder, nil)
	deps.folderRepo.On("FindByID", ctx, rootFolder.ID).Return(rootFolder, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Ancestors, 2)
	// After reversal: [rootFolder, parentFolder]
	assert.Equal(t, rootFolder.ID, output.Ancestors[0].ID)
	assert.Equal(t, parentFolder.ID, output.Ancestors[1].ID)
}

func TestGetAncestorsQuery_Execute_FolderNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newGetAncestorsTestDeps(t)

	folderID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("folder")

	input := query.GetAncestorsInput{
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

func TestGetAncestorsQuery_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newGetAncestorsTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folder := newAncestorFolderEntity(ownerID, "folder")

	input := query.GetAncestorsInput{
		FolderID: folder.ID,
		UserID:   differentUserID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestGetAncestorsQuery_Execute_ClosureRepoError_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newGetAncestorsTestDeps(t)

	ownerID := uuid.New()
	folder := newAncestorFolderEntity(ownerID, "folder")
	repoErr := errors.New("database error")

	input := query.GetAncestorsInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.folderClosureRepo.On("FindAncestorIDs", ctx, folder.ID).Return(nil, repoErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, repoErr, err)
}
