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

type restoreFileTestDeps struct {
	fileRepo                *mocks.MockFileRepository
	fileVersionRepo         *mocks.MockFileVersionRepository
	folderRepo              *mocks.MockFolderRepository
	archivedFileRepo        *mocks.MockArchivedFileRepository
	archivedFileVersionRepo *mocks.MockArchivedFileVersionRepository
	userRepo                *mocks.MockUserRepository
	txManager               *mocks.MockTransactionManager
}

func newRestoreFileTestDeps(t *testing.T) *restoreFileTestDeps {
	t.Helper()
	return &restoreFileTestDeps{
		fileRepo:                mocks.NewMockFileRepository(t),
		fileVersionRepo:         mocks.NewMockFileVersionRepository(t),
		folderRepo:              mocks.NewMockFolderRepository(t),
		archivedFileRepo:        mocks.NewMockArchivedFileRepository(t),
		archivedFileVersionRepo: mocks.NewMockArchivedFileVersionRepository(t),
		userRepo:                mocks.NewMockUserRepository(t),
		txManager:               mocks.NewMockTransactionManager(t),
	}
}

func (d *restoreFileTestDeps) newCommand() *command.RestoreFileCommand {
	return command.NewRestoreFileCommand(
		d.fileRepo,
		d.fileVersionRepo,
		d.folderRepo,
		d.archivedFileRepo,
		d.archivedFileVersionRepo,
		d.userRepo,
		d.txManager,
	)
}

func newArchivedFile(ownerID, originalFolderID uuid.UUID) *entity.ArchivedFile {
	name, _ := valueobject.NewFileName("report.pdf")
	mimeType, _ := valueobject.NewMimeType("application/pdf")
	storageKey := valueobject.NewStorageKey(uuid.New())
	return entity.ReconstructArchivedFile(
		uuid.New(),
		uuid.New(),
		originalFolderID,
		"/documents/report.pdf",
		name,
		mimeType,
		2048,
		ownerID,
		ownerID,
		storageKey,
		time.Now().Add(-time.Hour),
		ownerID,
		time.Now().Add(29*24*time.Hour), // not expired
	)
}

func newExpiredArchivedFile(ownerID, originalFolderID uuid.UUID) *entity.ArchivedFile {
	name, _ := valueobject.NewFileName("old.txt")
	mimeType, _ := valueobject.NewMimeType("text/plain")
	storageKey := valueobject.NewStorageKey(uuid.New())
	return entity.ReconstructArchivedFile(
		uuid.New(),
		uuid.New(),
		originalFolderID,
		"/documents/old.txt",
		name,
		mimeType,
		512,
		ownerID,
		ownerID,
		storageKey,
		time.Now().Add(-31*24*time.Hour),
		ownerID,
		time.Now().Add(-time.Hour), // expired
	)
}

func newRestoreUserWithPersonalFolder(personalFolderID uuid.UUID) *entity.User {
	email, _ := valueobject.NewEmail("user@example.com")
	user := entity.NewUser(email, "Test User", "hash")
	user.SetPersonalFolder(personalFolderID)
	return user
}

func TestRestoreFileCommand_Execute_Success_RestoresToOriginalFolder(t *testing.T) {
	ctx := context.Background()
	deps := newRestoreFileTestDeps(t)

	ownerID := uuid.New()
	originalFolderID := uuid.New()
	archivedFile := newArchivedFile(ownerID, originalFolderID)

	deps.archivedFileRepo.On("FindByID", ctx, archivedFile.ID).Return(archivedFile, nil)
	deps.folderRepo.On("ExistsByID", ctx, originalFolderID).Return(true, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, archivedFile.Name, originalFolderID).Return(false, nil)
	deps.archivedFileVersionRepo.On("FindByArchivedFileID", ctx, archivedFile.ID).Return([]*entity.ArchivedFileVersion{}, nil)
	deps.fileRepo.On("Create", ctx, mock.AnythingOfType("*entity.File")).Return(nil)
	deps.archivedFileVersionRepo.On("DeleteByArchivedFileID", ctx, archivedFile.ID).Return(nil)
	deps.archivedFileRepo.On("Delete", ctx, archivedFile.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.RestoreFileInput{
		ArchivedFileID:  archivedFile.ID,
		RestoreFolderID: nil,
		UserID:          ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, originalFolderID, output.FolderID)
	assert.NotEmpty(t, output.Name)
}

func TestRestoreFileCommand_Execute_OriginalFolderMissing_FallsBackToPersonalFolder(t *testing.T) {
	ctx := context.Background()
	deps := newRestoreFileTestDeps(t)

	ownerID := uuid.New()
	originalFolderID := uuid.New()
	personalFolderID := uuid.New()
	archivedFile := newArchivedFile(ownerID, originalFolderID)
	user := newRestoreUserWithPersonalFolder(personalFolderID)

	deps.archivedFileRepo.On("FindByID", ctx, archivedFile.ID).Return(archivedFile, nil)
	deps.folderRepo.On("ExistsByID", ctx, originalFolderID).Return(false, nil)
	deps.userRepo.On("FindByID", ctx, ownerID).Return(user, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, archivedFile.Name, personalFolderID).Return(false, nil)
	deps.archivedFileVersionRepo.On("FindByArchivedFileID", ctx, archivedFile.ID).Return([]*entity.ArchivedFileVersion{}, nil)
	deps.fileRepo.On("Create", ctx, mock.AnythingOfType("*entity.File")).Return(nil)
	deps.archivedFileVersionRepo.On("DeleteByArchivedFileID", ctx, archivedFile.ID).Return(nil)
	deps.archivedFileRepo.On("Delete", ctx, archivedFile.ID).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.RestoreFileInput{
		ArchivedFileID:  archivedFile.ID,
		RestoreFolderID: nil,
		UserID:          ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, personalFolderID, output.FolderID)
}

func TestRestoreFileCommand_Execute_NotOwner_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newRestoreFileTestDeps(t)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	originalFolderID := uuid.New()
	archivedFile := newArchivedFile(ownerID, originalFolderID)

	deps.archivedFileRepo.On("FindByID", ctx, archivedFile.ID).Return(archivedFile, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.RestoreFileInput{
		ArchivedFileID:  archivedFile.ID,
		RestoreFolderID: nil,
		UserID:          otherUserID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestRestoreFileCommand_Execute_Expired_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newRestoreFileTestDeps(t)

	ownerID := uuid.New()
	originalFolderID := uuid.New()
	archivedFile := newExpiredArchivedFile(ownerID, originalFolderID)

	deps.archivedFileRepo.On("FindByID", ctx, archivedFile.ID).Return(archivedFile, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.RestoreFileInput{
		ArchivedFileID:  archivedFile.ID,
		RestoreFolderID: nil,
		UserID:          ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestRestoreFileCommand_Execute_DuplicateName_ReturnsConflict(t *testing.T) {
	ctx := context.Background()
	deps := newRestoreFileTestDeps(t)

	ownerID := uuid.New()
	originalFolderID := uuid.New()
	archivedFile := newArchivedFile(ownerID, originalFolderID)

	deps.archivedFileRepo.On("FindByID", ctx, archivedFile.ID).Return(archivedFile, nil)
	deps.folderRepo.On("ExistsByID", ctx, originalFolderID).Return(true, nil)
	deps.fileRepo.On("ExistsByNameAndFolder", ctx, archivedFile.Name, originalFolderID).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.RestoreFileInput{
		ArchivedFileID:  archivedFile.ID,
		RestoreFolderID: nil,
		UserID:          ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}
