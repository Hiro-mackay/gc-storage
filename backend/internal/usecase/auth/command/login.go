package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// LoginInput はログインの入力を定義します
type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

// LoginOutput はログインの出力を定義します
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
	User         *entity.User
}

// LoginCommand はログインコマンドです
type LoginCommand struct {
	userRepo     repository.UserRepository
	sessionStore *cache.SessionStore
	jwtService   *jwt.JWTService
}

// NewLoginCommand は新しいLoginCommandを作成します
func NewLoginCommand(
	userRepo repository.UserRepository,
	sessionStore *cache.SessionStore,
	jwtService *jwt.JWTService,
) *LoginCommand {
	return &LoginCommand{
		userRepo:     userRepo,
		sessionStore: sessionStore,
		jwtService:   jwtService,
	}
}

// Execute はログインを実行します
func (c *LoginCommand) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// 1. メールアドレスでユーザーを検索
	email, err := valueobject.NewEmail(input.Email)
	if err != nil {
		return nil, apperror.NewUnauthorizedError("invalid credentials")
	}

	user, err := c.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, apperror.NewUnauthorizedError("invalid credentials")
	}

	// 2. パスワード検証
	if user.PasswordHash == "" {
		// OAuth専用ユーザー
		return nil, apperror.NewUnauthorizedError("please use OAuth to login")
	}

	password := valueobject.PasswordFromHash(user.PasswordHash)
	if !password.Verify(input.Password) {
		return nil, apperror.NewUnauthorizedError("invalid credentials")
	}

	// 3. ユーザー状態チェック
	if user.Status != entity.UserStatusActive {
		switch user.Status {
		case entity.UserStatusPending:
			return nil, apperror.NewUnauthorizedError("please verify your email first")
		case entity.UserStatusSuspended:
			return nil, apperror.NewUnauthorizedError("account suspended")
		case entity.UserStatusDeactivated:
			return nil, apperror.NewUnauthorizedError("account deactivated")
		default:
			return nil, apperror.NewUnauthorizedError("account is not active")
		}
	}

	// 4. セッション作成
	sessionID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(c.jwtService.GetRefreshTokenExpiry())

	accessToken, refreshToken, err := c.jwtService.GenerateTokenPair(user.ID, user.Email.String(), sessionID)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	session := &cache.Session{
		ID:           sessionID,
		UserID:       user.ID,
		RefreshToken: refreshToken,
		UserAgent:    input.UserAgent,
		IPAddress:    input.IPAddress,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		LastUsedAt:   now,
	}

	if err := c.sessionStore.Save(ctx, session); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(c.jwtService.GetAccessTokenExpiry().Seconds()),
		User:         user,
	}, nil
}
