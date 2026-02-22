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

type listTrashTestDeps struct {
	archivedFileRepo *mocks.MockArchivedFileRepository
}

func newListTrashTestDeps(t *testing.T) *listTrashTestDeps {
	t.Helper()
	return &listTrashTestDeps{
		archivedFileRepo: mocks.NewMockArchivedFileRepository(t),
	}
}

func (d *listTrashTestDeps) newQuery() *query.ListTrashQuery {
	return query.NewListTrashQuery(d.archivedFileRepo)
}

func newArchivedFileItem(ownerID uuid.UUID, name string) *entity.ArchivedFile {
	fileID := uuid.New()
	fileName, _ := valueobject.NewFileName(name)
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructArchivedFile(
		uuid.New(),
		fileID,
		uuid.New(),
		"/docs/"+name,
		fileName,
		mimeType,
		1024,
		ownerID,
		ownerID,
		storageKey,
		time.Now().Add(-time.Hour),
		ownerID,
		time.Now().Add(29*24*time.Hour),
	)
}

func TestListTrashQuery_Execute_ReturnsItems(t *testing.T) {
	ctx := context.Background()
	deps := newListTrashTestDeps(t)

	ownerID := uuid.New()
	file1 := newArchivedFileItem(ownerID, "report.txt")
	file2 := newArchivedFileItem(ownerID, "notes.txt")
	archivedFiles := []*entity.ArchivedFile{file1, file2}

	deps.archivedFileRepo.On("FindByOwnerWithPagination", ctx, ownerID, query.DefaultTrashLimit+1, (*uuid.UUID)(nil)).
		Return(archivedFiles, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListTrashInput{
		OwnerID: ownerID,
		UserID:  ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Items, 2)
	assert.False(t, output.HasMore)
	assert.Nil(t, output.NextCursor)
	assert.Equal(t, "report.txt", output.Items[0].Name)
	assert.Equal(t, "notes.txt", output.Items[1].Name)
}

func TestListTrashQuery_Execute_WithPagination_SetsNextCursor(t *testing.T) {
	ctx := context.Background()
	deps := newListTrashTestDeps(t)

	ownerID := uuid.New()
	limit := 2

	// create limit+1 items to trigger HasMore
	files := make([]*entity.ArchivedFile, limit+1)
	for i := range files {
		files[i] = newArchivedFileItem(ownerID, "file.txt")
	}

	deps.archivedFileRepo.On("FindByOwnerWithPagination", ctx, ownerID, limit+1, (*uuid.UUID)(nil)).
		Return(files, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListTrashInput{
		OwnerID: ownerID,
		UserID:  ownerID,
		Limit:   limit,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Items, limit)
	assert.True(t, output.HasMore)
	require.NotNil(t, output.NextCursor)
	assert.Equal(t, output.Items[limit-1].ID, *output.NextCursor)
}

func TestListTrashQuery_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newListTrashTestDeps(t)

	ownerID := uuid.New()
	otherUserID := uuid.New()

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListTrashInput{
		OwnerID: ownerID,
		UserID:  otherUserID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestListTrashQuery_Execute_LimitExceedsMax_ClampsToMax(t *testing.T) {
	ctx := context.Background()
	deps := newListTrashTestDeps(t)

	ownerID := uuid.New()

	deps.archivedFileRepo.On("FindByOwnerWithPagination", ctx, ownerID, query.MaxTrashLimit+1, (*uuid.UUID)(nil)).
		Return([]*entity.ArchivedFile{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListTrashInput{
		OwnerID: ownerID,
		UserID:  ownerID,
		Limit:   9999,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output.Items)
}

func TestListTrashQuery_Execute_ZeroLimit_UsesDefault(t *testing.T) {
	ctx := context.Background()
	deps := newListTrashTestDeps(t)

	ownerID := uuid.New()

	deps.archivedFileRepo.On("FindByOwnerWithPagination", ctx, ownerID, query.DefaultTrashLimit+1, (*uuid.UUID)(nil)).
		Return([]*entity.ArchivedFile{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListTrashInput{
		OwnerID: ownerID,
		UserID:  ownerID,
		Limit:   0,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output.Items)
}
