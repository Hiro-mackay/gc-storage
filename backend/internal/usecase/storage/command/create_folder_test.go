package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type createFolderTestDeps struct {
	folderRepo         *mocks.MockFolderRepository
	folderClosureRepo  *mocks.MockFolderClosureRepository
	relationshipRepo   *mocks.MockRelationshipRepository
	permissionResolver *mocks.MockPermissionResolver
	txManager          *mocks.MockTransactionManager
}

func newCreateFolderTestDeps(t *testing.T) *createFolderTestDeps {
	t.Helper()
	return &createFolderTestDeps{
		folderRepo:         mocks.NewMockFolderRepository(t),
		folderClosureRepo:  mocks.NewMockFolderClosureRepository(t),
		relationshipRepo:   mocks.NewMockRelationshipRepository(t),
		permissionResolver: mocks.NewMockPermissionResolver(t),
		txManager:          mocks.NewMockTransactionManager(t),
	}
}

func (d *createFolderTestDeps) newCommand() *command.CreateFolderCommand {
	return command.NewCreateFolderCommand(
		d.folderRepo,
		d.folderClosureRepo,
		d.relationshipRepo,
		d.permissionResolver,
		d.txManager,
	)
}

func TestCreateFolderCommand_Execute_RootFolder_ReturnsFolder(t *testing.T) {
	ctx := context.Background()
	deps := newCreateFolderTestDeps(t)

	ownerID := uuid.New()
	input := command.CreateFolderInput{
		Name:     "my-folder",
		ParentID: nil,
		OwnerID:  ownerID,
	}

	deps.folderRepo.On("ExistsByNameAndOwnerRoot", ctx, mock.AnythingOfType("valueobject.FolderName"), ownerID).Return(false, nil)
	deps.folderRepo.On("Create", ctx, mock.AnythingOfType("*entity.Folder")).Return(nil)
	deps.folderClosureRepo.On("InsertSelfReference", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
	deps.relationshipRepo.On("Create", ctx, mock.AnythingOfType("*authz.Relationship")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, "my-folder", output.Folder.Name.String())
	assert.Nil(t, output.Folder.ParentID)
	assert.Equal(t, ownerID, output.Folder.OwnerID)
	assert.Equal(t, 0, output.Folder.Depth)
}

func TestCreateFolderCommand_Execute_ChildFolder_ReturnsFolder(t *testing.T) {
	ctx := context.Background()
	deps := newCreateFolderTestDeps(t)

	ownerID := uuid.New()
	parentID := uuid.New()
	parent := newRootFolderEntity(ownerID)
	parent.ID = parentID

	input := command.CreateFolderInput{
		Name:     "child-folder",
		ParentID: &parentID,
		OwnerID:  ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, parentID).Return(parent, nil)
	deps.permissionResolver.On("HasPermission", ctx, ownerID, authz.ResourceTypeFolder, parentID, authz.PermFolderCreate).Return(true, nil)
	deps.folderRepo.On("ExistsByNameAndParent", ctx, mock.AnythingOfType("valueobject.FolderName"), &parentID, ownerID).Return(false, nil)
	deps.folderRepo.On("Create", ctx, mock.AnythingOfType("*entity.Folder")).Return(nil)
	deps.folderClosureRepo.On("InsertSelfReference", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
	deps.folderClosureRepo.On("FindAncestorPaths", ctx, parentID).Return([]*entity.FolderPath{}, nil)
	deps.folderClosureRepo.On("InsertAncestorPaths", ctx, mock.AnythingOfType("[]*entity.FolderPath")).Return(nil)
	deps.relationshipRepo.On("Create", ctx, mock.AnythingOfType("*authz.Relationship")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, "child-folder", output.Folder.Name.String())
	assert.Equal(t, &parentID, output.Folder.ParentID)
	assert.Equal(t, 1, output.Folder.Depth)
}

func TestCreateFolderCommand_Execute_InvalidName_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newCreateFolderTestDeps(t)

	input := command.CreateFolderInput{
		Name:     "",
		ParentID: nil,
		OwnerID:  uuid.New(),
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestCreateFolderCommand_Execute_ParentNotFound_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newCreateFolderTestDeps(t)

	parentID := uuid.New()
	notFoundErr := apperror.NewNotFoundError("folder")

	input := command.CreateFolderInput{
		Name:     "child-folder",
		ParentID: &parentID,
		OwnerID:  uuid.New(),
	}

	deps.folderRepo.On("FindByID", ctx, parentID).Return(nil, notFoundErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCreateFolderCommand_Execute_NoPermission_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()
	deps := newCreateFolderTestDeps(t)

	ownerID := uuid.New()
	parentID := uuid.New()
	parent := newRootFolderEntity(ownerID)
	parent.ID = parentID

	input := command.CreateFolderInput{
		Name:     "child-folder",
		ParentID: &parentID,
		OwnerID:  ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, parentID).Return(parent, nil)
	deps.permissionResolver.On("HasPermission", ctx, ownerID, authz.ResourceTypeFolder, parentID, authz.PermFolderCreate).Return(false, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestCreateFolderCommand_Execute_DuplicateName_ReturnsConflict(t *testing.T) {
	ctx := context.Background()
	deps := newCreateFolderTestDeps(t)

	ownerID := uuid.New()
	input := command.CreateFolderInput{
		Name:     "duplicate-folder",
		ParentID: nil,
		OwnerID:  ownerID,
	}

	deps.folderRepo.On("ExistsByNameAndOwnerRoot", ctx, mock.AnythingOfType("valueobject.FolderName"), ownerID).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestCreateFolderCommand_Execute_MaxDepthExceeded_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newCreateFolderTestDeps(t)

	ownerID := uuid.New()
	parentID := uuid.New()
	// parent at max depth so child would exceed limit
	parent := newRootFolderEntity(ownerID)
	parent.ID = parentID
	parent.Depth = entity.MaxFolderDepth

	input := command.CreateFolderInput{
		Name:     "too-deep-folder",
		ParentID: &parentID,
		OwnerID:  ownerID,
	}

	deps.folderRepo.On("FindByID", ctx, parentID).Return(parent, nil)
	deps.permissionResolver.On("HasPermission", ctx, ownerID, authz.ResourceTypeFolder, parentID, authz.PermFolderCreate).Return(true, nil)
	deps.folderRepo.On("ExistsByNameAndParent", ctx, mock.AnythingOfType("valueobject.FolderName"), &parentID, ownerID).Return(false, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestCreateFolderCommand_Execute_RepoCreateError_PropagatesError(t *testing.T) {
	ctx := context.Background()
	deps := newCreateFolderTestDeps(t)

	ownerID := uuid.New()
	repoErr := errors.New("database error")

	input := command.CreateFolderInput{
		Name:     "my-folder",
		ParentID: nil,
		OwnerID:  ownerID,
	}

	deps.folderRepo.On("ExistsByNameAndOwnerRoot", ctx, mock.AnythingOfType("valueobject.FolderName"), ownerID).Return(false, nil)
	deps.folderRepo.On("Create", ctx, mock.AnythingOfType("*entity.Folder")).Return(repoErr)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, repoErr, err)
}
