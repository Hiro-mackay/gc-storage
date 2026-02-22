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

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type deleteFolderTestDeps struct {
	folderRepo              *mocks.MockFolderRepository
	folderClosureRepo       *mocks.MockFolderClosureRepository
	fileRepo                *mocks.MockFileRepository
	fileVersionRepo         *mocks.MockFileVersionRepository
	archivedFileRepo        *mocks.MockArchivedFileRepository
	archivedFileVersionRepo *mocks.MockArchivedFileVersionRepository
	txManager               *mocks.MockTransactionManager
	userRepo                *mocks.MockUserRepository
}

func newDeleteFolderTestDeps(t *testing.T) *deleteFolderTestDeps {
	t.Helper()
	return &deleteFolderTestDeps{
		folderRepo:              mocks.NewMockFolderRepository(t),
		folderClosureRepo:       mocks.NewMockFolderClosureRepository(t),
		fileRepo:                mocks.NewMockFileRepository(t),
		fileVersionRepo:         mocks.NewMockFileVersionRepository(t),
		archivedFileRepo:        mocks.NewMockArchivedFileRepository(t),
		archivedFileVersionRepo: mocks.NewMockArchivedFileVersionRepository(t),
		txManager:               mocks.NewMockTransactionManager(t),
		userRepo:                mocks.NewMockUserRepository(t),
	}
}

func (d *deleteFolderTestDeps) newCommand() *command.DeleteFolderCommand {
	return command.NewDeleteFolderCommand(
		d.folderRepo,
		d.folderClosureRepo,
		d.fileRepo,
		d.fileVersionRepo,
		d.archivedFileRepo,
		d.archivedFileVersionRepo,
		d.txManager,
		d.userRepo,
	)
}

func newDeleteUserWithoutPersonalFolder(userID uuid.UUID) *entity.User {
	email, _ := valueobject.NewEmail("delete@example.com")
	user := entity.NewUser(email, "Delete User", "hashedpw")
	user.ID = userID
	return user
}

func newActiveFileEntityForDelete(ownerID, folderID uuid.UUID) *entity.File {
	name, _ := valueobject.NewFileName("file.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(uuid.New())
	return entity.ReconstructFile(
		uuid.New(), folderID, ownerID, ownerID,
		name, mimeType, 1024, storageKey, 1,
		entity.FileStatusActive, time.Now(), time.Now(),
	)
}

func TestDeleteFolderCommand_Execute_EmptyFolder_DeletesFolder(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newDeleteUserWithoutPersonalFolder(ownerID)

	input := command.DeleteFolderInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)
	deps.fileRepo.On("FindByFolderIDs", ctx, []uuid.UUID{folder.ID}).Return([]*entity.File{}, nil)
	deps.folderClosureRepo.On("DeleteSubtreePaths", ctx, folder.ID).Return(nil)
	deps.folderRepo.On("Delete", ctx, folder.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 1, output.DeletedFolderCount)
	assert.Equal(t, 0, output.ArchivedFileCount)
}

func TestDeleteFolderCommand_Execute_FolderWithFiles_ArchivesFilesAndDeletesFolder(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newDeleteUserWithoutPersonalFolder(ownerID)
	file := newActiveFileEntityForDelete(ownerID, folder.ID)

	fileVersion := &entity.FileVersion{
		ID:             uuid.New(),
		FileID:         file.ID,
		VersionNumber:  1,
		MinioVersionID: "minio-v1",
		Size:           1024,
		Checksum:       "abc123",
		UploadedBy:     ownerID,
		CreatedAt:      time.Now(),
	}

	input := command.DeleteFolderInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)
	deps.fileRepo.On("FindByFolderIDs", ctx, []uuid.UUID{folder.ID}).Return([]*entity.File{file}, nil)
	deps.fileVersionRepo.On("FindByFileID", ctx, file.ID).Return([]*entity.FileVersion{fileVersion}, nil)
	deps.folderClosureRepo.On("FindAncestorIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)
	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.archivedFileRepo.On("Create", ctx, mock.AnythingOfType("*entity.ArchivedFile")).Return(nil)
	deps.archivedFileVersionRepo.On("BulkCreate", ctx, mock.AnythingOfType("[]*entity.ArchivedFileVersion")).Return(nil)
	deps.fileVersionRepo.On("DeleteByFileID", ctx, file.ID).Return(nil)
	deps.fileRepo.On("Delete", ctx, file.ID).Return(nil)
	deps.folderClosureRepo.On("DeleteSubtreePaths", ctx, folder.ID).Return(nil)
	deps.folderRepo.On("Delete", ctx, folder.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 1, output.DeletedFolderCount)
	assert.Equal(t, 1, output.ArchivedFileCount)
}

func TestDeleteFolderCommand_Execute_FolderWithDescendants_DeletesAllFolders(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	childID := uuid.New()
	user := newDeleteUserWithoutPersonalFolder(ownerID)

	input := command.DeleteFolderInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folder.ID).Return([]uuid.UUID{childID}, nil)
	deps.fileRepo.On("FindByFolderIDs", ctx, []uuid.UUID{folder.ID, childID}).Return([]*entity.File{}, nil)
	deps.folderClosureRepo.On("DeleteSubtreePaths", ctx, folder.ID).Return(nil)
	// folderIDs = [folder.ID, childID], reversed deletion order: childID first, then folder.ID
	deps.folderRepo.On("Delete", ctx, childID).Return(nil)
	deps.folderRepo.On("Delete", ctx, folder.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 2, output.DeletedFolderCount)
	assert.Equal(t, 0, output.ArchivedFileCount)
}

func TestDeleteFolderCommand_Execute_FolderNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteFolderTestDeps(t)

	folderID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("folder")

	input := command.DeleteFolderInput{
		FolderID: folderID,
		UserID:   uuid.New(),
	}

	deps.folderRepo.On("FindByID", ctx, folderID).Return(nil, notFoundErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestDeleteFolderCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteFolderTestDeps(t)

	ownerID := uuid.New()
	differentUserID := uuid.New()
	folder := newRootFolderEntity(ownerID)

	input := command.DeleteFolderInput{
		FolderID: folder.ID,
		UserID:   differentUserID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestDeleteFolderCommand_Execute_PersonalFolder_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newUserWithPersonalFolder(folder.ID)
	user.ID = ownerID

	input := command.DeleteFolderInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestDeleteFolderCommand_Execute_SkipsInactiveFiles(t *testing.T) {
	ctx := context.Background()
	deps := newDeleteFolderTestDeps(t)

	ownerID := uuid.New()
	folder := newRootFolderEntity(ownerID)
	user := newDeleteUserWithoutPersonalFolder(ownerID)

	// Create an uploading (inactive) file
	name, _ := valueobject.NewFileName("uploading.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(uuid.New())
	uploadingFile := entity.ReconstructFile(
		uuid.New(), folder.ID, ownerID, ownerID,
		name, mimeType, 1024, storageKey, 1,
		entity.FileStatusUploading, time.Now(), time.Now(),
	)

	input := command.DeleteFolderInput{
		FolderID: folder.ID,
		UserID:   ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, folder.ID).Return(folder, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.folderClosureRepo.On("FindDescendantIDs", ctx, folder.ID).Return([]uuid.UUID{}, nil)
	deps.fileRepo.On("FindByFolderIDs", ctx, []uuid.UUID{folder.ID}).Return([]*entity.File{uploadingFile}, nil)
	deps.folderClosureRepo.On("DeleteSubtreePaths", ctx, folder.ID).Return(nil)
	deps.folderRepo.On("Delete", ctx, folder.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 1, output.DeletedFolderCount)
	assert.Equal(t, 0, output.ArchivedFileCount)
}
