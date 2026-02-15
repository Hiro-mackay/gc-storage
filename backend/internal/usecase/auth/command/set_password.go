package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// SetPasswordInput はパスワード設定の入力を定義します
type SetPasswordInput struct {
	UserID           uuid.UUID
	Password         string
	CurrentSessionID string // 現在のセッションIDを保持するため
}

// SetPasswordOutput はパスワード設定の出力を定義します
type SetPasswordOutput struct {
	Message string
}

// SetPasswordCommand はOAuthユーザーのパスワード設定コマンドです
type SetPasswordCommand struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

// NewSetPasswordCommand は新しいSetPasswordCommandを作成します
func NewSetPasswordCommand(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
) *SetPasswordCommand {
	return &SetPasswordCommand{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// Execute はパスワード設定を実行します
func (c *SetPasswordCommand) Execute(ctx context.Context, input SetPasswordInput) (*SetPasswordOutput, error) {
	// 1. ユーザーを取得
	user, err := c.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}

	// 2. すでにパスワードが設定されているかチェック
	if user.PasswordHash != "" {
		return nil, apperror.NewValidationError("password already set, use change password instead", nil)
	}

	// 3. 新しいパスワードのバリデーション
	password, err := valueobject.NewPassword(input.Password, user.Email.String())
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 4. パスワード設定
	user.PasswordHash = password.Hash()
	user.UpdatedAt = time.Now()

	if err := c.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 5. 現在のセッション以外の全セッションを無効化
	if input.CurrentSessionID != "" {
		currentSession, _ := c.sessionRepo.FindByID(ctx, input.CurrentSessionID)
		if err := c.sessionRepo.DeleteByUserID(ctx, input.UserID); err != nil {
			return nil, apperror.NewInternalError(err)
		}
		if currentSession != nil {
			_ = c.sessionRepo.Save(ctx, currentSession)
		}
	}

	return &SetPasswordOutput{
		Message: "password set successfully",
	}, nil
}
