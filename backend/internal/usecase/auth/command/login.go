package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
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
	SessionID string
	User      *entity.User
}

// LoginCommand はログインコマンドです
type LoginCommand struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

// NewLoginCommand は新しいLoginCommandを作成します
func NewLoginCommand(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
) *LoginCommand {
	return &LoginCommand{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
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

	// 2. パスワード検証 (FS-LOGIN-002: OAuth専用ユーザーも同じエラーを返す)
	if user.PasswordHash == "" {
		return nil, apperror.NewUnauthorizedError("invalid credentials")
	}

	password := valueobject.PasswordFromHash(user.PasswordHash)
	if !password.Verify(input.Password) {
		return nil, apperror.NewUnauthorizedError("invalid credentials")
	}

	// 3. ユーザー状態チェック（pending は許可、メール認証は重要操作時にのみ要求）
	switch user.Status {
	case entity.UserStatusActive, entity.UserStatusPending:
		// OK: ログイン許可
	case entity.UserStatusSuspended:
		return nil, apperror.NewUnauthorizedError("account suspended")
	case entity.UserStatusDeactivated:
		return nil, apperror.NewUnauthorizedError("account deactivated")
	default:
		return nil, apperror.NewUnauthorizedError("account is not active")
	}

	// 4. セッション制限チェック (R-SS002)
	sessionCount, err := c.sessionRepo.CountByUserID(ctx, user.ID)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 最大セッション数に達している場合は最古のセッションを削除
	if sessionCount >= int64(entity.MaxActiveSessionsPerUser) {
		if err := c.sessionRepo.DeleteOldestByUserID(ctx, user.ID); err != nil {
			return nil, apperror.NewInternalError(err)
		}
	}

	// 5. セッション作成
	sessionID := uuid.New().String()
	now := time.Now()

	session := &entity.Session{
		ID:         sessionID,
		UserID:     user.ID,
		UserAgent:  input.UserAgent,
		IPAddress:  input.IPAddress,
		ExpiresAt:  now.Add(entity.SessionTTL),
		CreatedAt:  now,
		LastUsedAt: now,
	}

	if err := c.sessionRepo.Save(ctx, session); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	return &LoginOutput{
		SessionID: sessionID,
		User:      user,
	}, nil
}
