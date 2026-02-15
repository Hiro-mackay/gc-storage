package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ChangePasswordInput はパスワード変更の入力を定義します
type ChangePasswordInput struct {
	UserID           uuid.UUID
	CurrentPassword  string
	NewPassword      string
	CurrentSessionID string // 現在のセッションIDを保持するため
}

// ChangePasswordOutput はパスワード変更の出力を定義します
type ChangePasswordOutput struct {
	Message string
}

// ChangePasswordCommand はパスワード変更コマンドです
type ChangePasswordCommand struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

// NewChangePasswordCommand は新しいChangePasswordCommandを作成します
func NewChangePasswordCommand(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
) *ChangePasswordCommand {
	return &ChangePasswordCommand{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// Execute はパスワード変更を実行します
func (c *ChangePasswordCommand) Execute(ctx context.Context, input ChangePasswordInput) (*ChangePasswordOutput, error) {
	// 1. ユーザーを取得
	user, err := c.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}

	// 2. 現在のパスワードを検証
	currentPassword := valueobject.PasswordFromHash(user.PasswordHash)
	if !currentPassword.Verify(input.CurrentPassword) {
		return nil, apperror.NewUnauthorizedError("current password is incorrect")
	}

	// 3. 新しいパスワードのバリデーション
	newPassword, err := valueobject.NewPassword(input.NewPassword, user.Email.String())
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 4. パスワード更新
	user.PasswordHash = newPassword.Hash()
	user.UpdatedAt = time.Now()

	if err := c.userRepo.Update(ctx, user); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 5. 現在のセッション以外の全セッションを無効化（セキュリティ: アカウント侵害時の対策）
	if input.CurrentSessionID != "" {
		// 現在のセッションを保持
		currentSession, _ := c.sessionRepo.FindByID(ctx, input.CurrentSessionID)
		// 全セッション削除
		if err := c.sessionRepo.DeleteByUserID(ctx, input.UserID); err != nil {
			return nil, apperror.NewInternalError(err)
		}
		// 現在のセッションを再保存
		if currentSession != nil {
			_ = c.sessionRepo.Save(ctx, currentSession)
		}
	}

	return &ChangePasswordOutput{
		Message: "password changed successfully",
	}, nil
}
