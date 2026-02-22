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

type listFileVersionsTestDeps struct {
	fileRepo        *mocks.MockFileRepository
	fileVersionRepo *mocks.MockFileVersionRepository
}

func newListFileVersionsTestDeps(t *testing.T) *listFileVersionsTestDeps {
	t.Helper()
	return &listFileVersionsTestDeps{
		fileRepo:        mocks.NewMockFileRepository(t),
		fileVersionRepo: mocks.NewMockFileVersionRepository(t),
	}
}

func (d *listFileVersionsTestDeps) newQuery() *query.ListFileVersionsQuery {
	return query.NewListFileVersionsQuery(d.fileRepo, d.fileVersionRepo)
}

func newActiveFileForVersionList(ownerID uuid.UUID) *entity.File {
	name, _ := valueobject.NewFileName("doc.pdf")
	mimeType, _ := valueobject.NewMimeType("application/pdf")
	fileID := uuid.New()
	storageKey := valueobject.NewStorageKey(fileID)
	return entity.ReconstructFile(
		fileID, uuid.New(), ownerID, ownerID,
		name, mimeType, 2048, storageKey, 2,
		entity.FileStatusActive, time.Now(), time.Now(),
	)
}

func TestListFileVersionsQuery_Execute_TwoVersions_ReturnsVersionList(t *testing.T) {
	ctx := context.Background()
	deps := newListFileVersionsTestDeps(t)

	ownerID := uuid.New()
	file := newActiveFileForVersionList(ownerID)

	uploaderID := uuid.New()
	versions := []*entity.FileVersion{
		entity.ReconstructFileVersion(uuid.New(), file.ID, 1, "mv1", 1024, "sha256:v1", uploaderID, time.Now().Add(-1*time.Hour)),
		entity.ReconstructFileVersion(uuid.New(), file.ID, 2, "mv2", 2048, "sha256:v2", uploaderID, time.Now()),
	}

	input := query.ListFileVersionsInput{
		FileID: file.ID,
		UserID: ownerID,
	}

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.fileVersionRepo.On("FindByFileID", ctx, file.ID).Return(versions, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, file.ID, output.FileID)
	assert.Equal(t, "doc.pdf", output.FileName)
	require.Len(t, output.Versions, 2)
	assert.Equal(t, 1, output.Versions[0].VersionNumber)
	assert.False(t, output.Versions[0].IsLatest)
	assert.Equal(t, 2, output.Versions[1].VersionNumber)
	assert.True(t, output.Versions[1].IsLatest)
}

func TestListFileVersionsQuery_Execute_EmptyVersions_ReturnsEmptyList(t *testing.T) {
	ctx := context.Background()
	deps := newListFileVersionsTestDeps(t)

	ownerID := uuid.New()
	file := newActiveFileForVersionList(ownerID)

	input := query.ListFileVersionsInput{
		FileID: file.ID,
		UserID: ownerID,
	}

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)
	deps.fileVersionRepo.On("FindByFileID", ctx, file.ID).Return([]*entity.FileVersion{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, file.ID, output.FileID)
	assert.Empty(t, output.Versions)
}

func TestListFileVersionsQuery_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newListFileVersionsTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	file := newActiveFileForVersionList(ownerID)

	input := query.ListFileVersionsInput{
		FileID: file.ID,
		UserID: differentUserID,
	}

	deps.fileRepo.On("FindByID", ctx, file.ID).Return(file, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestListFileVersionsQuery_Execute_FileNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newListFileVersionsTestDeps(t)

	fileID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("file")

	input := query.ListFileVersionsInput{
		FileID: fileID,
		UserID: uuid.New(),
	}

	deps.fileRepo.On("FindByID", ctx, fileID).Return(nil, notFoundErr)

	q := deps.newQuery()
	output, err := q.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}
