package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

const testAppURL = "http://localhost:3000"

type registerTestDeps struct {
	userRepo                   *mocks.MockUserRepository
	sessionRepo                *mocks.MockSessionRepository
	folderRepo                 *mocks.MockFolderRepository
	folderClosureRepo          *mocks.MockFolderClosureRepository
	relationshipRepo           *mocks.MockRelationshipRepository
	emailVerificationTokenRepo *mocks.MockEmailVerificationTokenRepository
	txManager                  *mocks.MockTransactionManager
}

func newRegisterTestDeps(t *testing.T) *registerTestDeps {
	t.Helper()
	return &registerTestDeps{
		userRepo:                   mocks.NewMockUserRepository(t),
		sessionRepo:                mocks.NewMockSessionRepository(t),
		folderRepo:                 mocks.NewMockFolderRepository(t),
		folderClosureRepo:          mocks.NewMockFolderClosureRepository(t),
		relationshipRepo:           mocks.NewMockRelationshipRepository(t),
		emailVerificationTokenRepo: mocks.NewMockEmailVerificationTokenRepository(t),
		txManager:                  mocks.NewMockTransactionManager(t),
	}
}

func (d *registerTestDeps) newCommand() *command.RegisterCommand {
	return command.NewRegisterCommand(
		d.userRepo,
		d.sessionRepo,
		d.folderRepo,
		d.folderClosureRepo,
		d.relationshipRepo,
		d.emailVerificationTokenRepo,
		d.txManager,
		nil,
		testAppURL,
	)
}

func newRegisterInput() command.RegisterInput {
	return command.RegisterInput{
		Email:     "newuser@example.com",
		Password:  testPassword,
		Name:      "New User",
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	}
}

func TestRegisterCommand_Execute_ValidInput_ReturnsUserAndSession(t *testing.T) {
	ctx := context.Background()
	deps := newRegisterTestDeps(t)
	input := newRegisterInput()

	deps.userRepo.On("Exists", ctx, mock.AnythingOfType("valueobject.Email")).Return(false, nil)
	deps.userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
	deps.folderRepo.On("Create", ctx, mock.AnythingOfType("*entity.Folder")).Return(nil)
	deps.folderClosureRepo.On("InsertSelfReference", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
	deps.relationshipRepo.On("Create", ctx, mock.AnythingOfType("*authz.Relationship")).Return(nil)
	deps.userRepo.On("SetPersonalFolderID", ctx, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil)
	deps.emailVerificationTokenRepo.On("Create", ctx, mock.AnythingOfType("*entity.EmailVerificationToken")).Return(nil)
	deps.sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.SessionID)
	assert.NotNil(t, output.User)
	assert.Equal(t, input.Name, output.User.Name)
}

func TestRegisterCommand_Execute_InvalidEmail_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newRegisterTestDeps(t)
	input := command.RegisterInput{
		Email:    "invalid-email",
		Password: testPassword,
		Name:     "New User",
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestRegisterCommand_Execute_WeakPassword_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newRegisterTestDeps(t)
	input := command.RegisterInput{
		Email:    "newuser@example.com",
		Password: "weak",
		Name:     "New User",
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestRegisterCommand_Execute_DuplicateEmail_ReturnsConflictError(t *testing.T) {
	ctx := context.Background()
	deps := newRegisterTestDeps(t)
	input := newRegisterInput()

	deps.userRepo.On("Exists", ctx, mock.AnythingOfType("valueobject.Email")).Return(true, nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestRegisterCommand_Execute_UserRepoCreateFails_ReturnsInternalError(t *testing.T) {
	ctx := context.Background()
	deps := newRegisterTestDeps(t)
	input := newRegisterInput()

	deps.userRepo.On("Exists", ctx, mock.AnythingOfType("valueobject.Email")).Return(false, nil)
	deps.userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(errors.New("db error"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeInternalError, appErr.Code)
}

func TestRegisterCommand_Execute_TokenCreateFails_ReturnsInternalError(t *testing.T) {
	ctx := context.Background()
	deps := newRegisterTestDeps(t)
	input := newRegisterInput()

	deps.userRepo.On("Exists", ctx, mock.AnythingOfType("valueobject.Email")).Return(false, nil)
	deps.userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
	deps.folderRepo.On("Create", ctx, mock.AnythingOfType("*entity.Folder")).Return(nil)
	deps.folderClosureRepo.On("InsertSelfReference", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
	deps.relationshipRepo.On("Create", ctx, mock.AnythingOfType("*authz.Relationship")).Return(nil)
	deps.userRepo.On("SetPersonalFolderID", ctx, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil)
	deps.emailVerificationTokenRepo.On("Create", ctx, mock.AnythingOfType("*entity.EmailVerificationToken")).Return(errors.New("db error"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeInternalError, appErr.Code)
}

func TestRegisterCommand_Execute_SessionSaveFails_ReturnsInternalError(t *testing.T) {
	ctx := context.Background()
	deps := newRegisterTestDeps(t)
	input := newRegisterInput()

	deps.userRepo.On("Exists", ctx, mock.AnythingOfType("valueobject.Email")).Return(false, nil)
	deps.userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
	deps.folderRepo.On("Create", ctx, mock.AnythingOfType("*entity.Folder")).Return(nil)
	deps.folderClosureRepo.On("InsertSelfReference", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
	deps.relationshipRepo.On("Create", ctx, mock.AnythingOfType("*authz.Relationship")).Return(nil)
	deps.userRepo.On("SetPersonalFolderID", ctx, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil)
	deps.emailVerificationTokenRepo.On("Create", ctx, mock.AnythingOfType("*entity.EmailVerificationToken")).Return(nil)
	deps.sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(errors.New("redis error"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeInternalError, appErr.Code)
}
