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
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type oauthTestDeps struct {
	userRepo          *mocks.MockUserRepository
	profileRepo       *mocks.MockUserProfileRepository
	oauthAccountRepo  *mocks.MockOAuthAccountRepository
	folderRepo        *mocks.MockFolderRepository
	folderClosureRepo *mocks.MockFolderClosureRepository
	relationshipRepo  *mocks.MockRelationshipRepository
	oauthFactory      *mocks.MockOAuthClientFactory
	txManager         *mocks.MockTransactionManager
	sessionRepo       *mocks.MockSessionRepository
	oauthClient       *mocks.MockOAuthClient
}

func newOAuthTestDeps(t *testing.T) *oauthTestDeps {
	return &oauthTestDeps{
		userRepo:          mocks.NewMockUserRepository(t),
		profileRepo:       mocks.NewMockUserProfileRepository(t),
		oauthAccountRepo:  mocks.NewMockOAuthAccountRepository(t),
		folderRepo:        mocks.NewMockFolderRepository(t),
		folderClosureRepo: mocks.NewMockFolderClosureRepository(t),
		relationshipRepo:  mocks.NewMockRelationshipRepository(t),
		oauthFactory:      mocks.NewMockOAuthClientFactory(t),
		txManager:         mocks.NewMockTransactionManager(t),
		sessionRepo:       mocks.NewMockSessionRepository(t),
		oauthClient:       mocks.NewMockOAuthClient(t),
	}
}

func (d *oauthTestDeps) newCommand() *command.OAuthLoginCommand {
	return command.NewOAuthLoginCommand(
		d.userRepo,
		d.profileRepo,
		d.oauthAccountRepo,
		d.folderRepo,
		d.folderClosureRepo,
		d.relationshipRepo,
		d.oauthFactory,
		d.txManager,
		d.sessionRepo,
	)
}

func newOAuthInput() command.OAuthLoginInput {
	return command.OAuthLoginInput{
		Provider:  "google",
		Code:      "valid-auth-code",
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	}
}

func newOAuthTokens() *service.OAuthTokens {
	return &service.OAuthTokens{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
	}
}

func newOAuthUserInfo() *service.OAuthUserInfo {
	return &service.OAuthUserInfo{
		ProviderUserID: "google-user-123",
		Email:          "oauth@example.com",
		Name:           "OAuth User",
		AvatarURL:      "https://example.com/avatar.jpg",
	}
}

func TestOAuthLoginCommand_Execute_InvalidProvider_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newOAuthTestDeps(t)
	input := command.OAuthLoginInput{
		Provider: "invalid-provider",
		Code:     "some-code",
	}

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestOAuthLoginCommand_Execute_InvalidCode_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newOAuthTestDeps(t)
	input := newOAuthInput()

	deps.oauthFactory.On("GetClient", valueobject.OAuthProvider("google")).
		Return(deps.oauthClient, nil)
	deps.oauthClient.On("ExchangeCode", ctx, "valid-auth-code").
		Return(nil, errors.New("invalid code"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestOAuthLoginCommand_Execute_ExistingOAuthAccount_ReturnsUser(t *testing.T) {
	ctx := context.Background()
	deps := newOAuthTestDeps(t)
	input := newOAuthInput()
	tokens := newOAuthTokens()
	userInfo := newOAuthUserInfo()
	userID := uuid.New()

	existingUser := &entity.User{
		ID:            userID,
		Name:          "OAuth User",
		Status:        entity.UserStatusActive,
		EmailVerified: true,
	}

	existingOAuth := &entity.OAuthAccount{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       valueobject.OAuthProvider("google"),
		ProviderUserID: "google-user-123",
		AccessToken:    "old-token",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	deps.oauthFactory.On("GetClient", valueobject.OAuthProvider("google")).Return(deps.oauthClient, nil)
	deps.oauthClient.On("ExchangeCode", ctx, "valid-auth-code").Return(tokens, nil)
	deps.oauthClient.On("GetUserInfo", ctx, "access-token").Return(userInfo, nil)
	deps.oauthAccountRepo.On("FindByProviderAndUserID", ctx, valueobject.OAuthProvider("google"), "google-user-123").Return(existingOAuth, nil)
	deps.userRepo.On("FindByID", ctx, userID).Return(existingUser, nil)
	deps.oauthAccountRepo.On("Update", ctx, mock.AnythingOfType("*entity.OAuthAccount")).Return(nil)
	deps.sessionRepo.On("CountByUserID", ctx, userID).Return(int64(0), nil)
	deps.sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, userID, output.User.ID)
	assert.False(t, output.IsNewUser)
	assert.NotEmpty(t, output.SessionID)
}

func TestOAuthLoginCommand_Execute_NewUser_CreatesUserAndFolder(t *testing.T) {
	ctx := context.Background()
	deps := newOAuthTestDeps(t)
	input := newOAuthInput()
	tokens := newOAuthTokens()
	userInfo := newOAuthUserInfo()

	email, _ := valueobject.NewEmail("oauth@example.com")

	deps.oauthFactory.On("GetClient", valueobject.OAuthProvider("google")).Return(deps.oauthClient, nil)
	deps.oauthClient.On("ExchangeCode", ctx, "valid-auth-code").Return(tokens, nil)
	deps.oauthClient.On("GetUserInfo", ctx, "access-token").Return(userInfo, nil)

	// No existing OAuth account
	deps.oauthAccountRepo.On("FindByProviderAndUserID", ctx, valueobject.OAuthProvider("google"), "google-user-123").
		Return(nil, errors.New("not found"))
	// No existing user by email
	deps.userRepo.On("FindByEmail", ctx, email).Return(nil, errors.New("not found"))
	// Create user
	deps.userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil)
	// Create folder
	deps.folderRepo.On("Create", ctx, mock.AnythingOfType("*entity.Folder")).Return(nil)
	// Create closure
	deps.folderClosureRepo.On("InsertSelfReference", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
	// Create relationship
	deps.relationshipRepo.On("Create", ctx, mock.AnythingOfType("*authz.Relationship")).Return(nil)
	// Set personal folder
	deps.userRepo.On("SetPersonalFolderID", ctx, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil)
	// Create profile
	deps.profileRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserProfile")).Return(nil)
	// Create OAuth account
	deps.oauthAccountRepo.On("Create", ctx, mock.AnythingOfType("*entity.OAuthAccount")).Return(nil)
	// Session management
	deps.sessionRepo.On("CountByUserID", ctx, mock.AnythingOfType("uuid.UUID")).Return(int64(0), nil)
	deps.sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.True(t, output.IsNewUser)
	assert.Equal(t, entity.UserStatusActive, output.User.Status)
	assert.True(t, output.User.EmailVerified)
}

func TestOAuthLoginCommand_Execute_PendingUserSameEmail_ActivatesUser(t *testing.T) {
	ctx := context.Background()
	deps := newOAuthTestDeps(t)
	input := newOAuthInput()
	tokens := newOAuthTokens()
	userInfo := newOAuthUserInfo()

	email, _ := valueobject.NewEmail("oauth@example.com")
	userID := uuid.New()

	pendingUser := &entity.User{
		ID:            userID,
		Email:         email,
		Name:          "Pending User",
		Status:        entity.UserStatusPending,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	deps.oauthFactory.On("GetClient", valueobject.OAuthProvider("google")).Return(deps.oauthClient, nil)
	deps.oauthClient.On("ExchangeCode", ctx, "valid-auth-code").Return(tokens, nil)
	deps.oauthClient.On("GetUserInfo", ctx, "access-token").Return(userInfo, nil)

	// No existing OAuth account
	deps.oauthAccountRepo.On("FindByProviderAndUserID", ctx, valueobject.OAuthProvider("google"), "google-user-123").
		Return(nil, errors.New("not found"))
	// Found existing pending user by email
	deps.userRepo.On("FindByEmail", ctx, email).Return(pendingUser, nil)
	// Create OAuth account link
	deps.oauthAccountRepo.On("Create", ctx, mock.AnythingOfType("*entity.OAuthAccount")).Return(nil)
	// Update user to active
	deps.userRepo.On("Update", ctx, mock.MatchedBy(func(u *entity.User) bool {
		return u.Status == entity.UserStatusActive && u.EmailVerified
	})).Return(nil)
	// Session management
	deps.sessionRepo.On("CountByUserID", ctx, userID).Return(int64(0), nil)
	deps.sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.False(t, output.IsNewUser)
	assert.Equal(t, entity.UserStatusActive, output.User.Status)
	assert.True(t, output.User.EmailVerified)
}

func TestOAuthLoginCommand_Execute_SuspendedUser_ReturnsUnauthorized(t *testing.T) {
	ctx := context.Background()
	deps := newOAuthTestDeps(t)
	input := newOAuthInput()
	tokens := newOAuthTokens()
	userInfo := newOAuthUserInfo()
	userID := uuid.New()

	suspendedUser := &entity.User{
		ID:            userID,
		Status:        entity.UserStatusSuspended,
		EmailVerified: true,
	}

	existingOAuth := &entity.OAuthAccount{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       valueobject.OAuthProvider("google"),
		ProviderUserID: "google-user-123",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	deps.oauthFactory.On("GetClient", valueobject.OAuthProvider("google")).Return(deps.oauthClient, nil)
	deps.oauthClient.On("ExchangeCode", ctx, "valid-auth-code").Return(tokens, nil)
	deps.oauthClient.On("GetUserInfo", ctx, "access-token").Return(userInfo, nil)
	deps.oauthAccountRepo.On("FindByProviderAndUserID", ctx, valueobject.OAuthProvider("google"), "google-user-123").Return(existingOAuth, nil)
	deps.userRepo.On("FindByID", ctx, userID).Return(suspendedUser, nil)
	// Token update is skipped for non-active users

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeUnauthorized, appErr.Code)
	assert.Contains(t, appErr.Message, "account suspended")
	deps.oauthAccountRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestOAuthLoginCommand_Execute_SessionLimit_DeletesOldest(t *testing.T) {
	ctx := context.Background()
	deps := newOAuthTestDeps(t)
	input := newOAuthInput()
	tokens := newOAuthTokens()
	userInfo := newOAuthUserInfo()
	userID := uuid.New()

	activeUser := &entity.User{
		ID:            userID,
		Status:        entity.UserStatusActive,
		EmailVerified: true,
	}

	existingOAuth := &entity.OAuthAccount{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       valueobject.OAuthProvider("google"),
		ProviderUserID: "google-user-123",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	deps.oauthFactory.On("GetClient", valueobject.OAuthProvider("google")).Return(deps.oauthClient, nil)
	deps.oauthClient.On("ExchangeCode", ctx, "valid-auth-code").Return(tokens, nil)
	deps.oauthClient.On("GetUserInfo", ctx, "access-token").Return(userInfo, nil)
	deps.oauthAccountRepo.On("FindByProviderAndUserID", ctx, valueobject.OAuthProvider("google"), "google-user-123").Return(existingOAuth, nil)
	deps.userRepo.On("FindByID", ctx, userID).Return(activeUser, nil)
	deps.oauthAccountRepo.On("Update", ctx, mock.AnythingOfType("*entity.OAuthAccount")).Return(nil)
	deps.sessionRepo.On("CountByUserID", ctx, userID).Return(int64(entity.MaxActiveSessionsPerUser), nil)
	deps.sessionRepo.On("DeleteOldestByUserID", ctx, userID).Return(nil)
	deps.sessionRepo.On("Save", ctx, mock.AnythingOfType("*entity.Session")).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, output)
	deps.sessionRepo.AssertCalled(t, "DeleteOldestByUserID", ctx, userID)
}
