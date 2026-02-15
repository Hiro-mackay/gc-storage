package command

import (
	"context"
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ResetPasswordInput はパスワードリセットの入力を定義します
type ResetPasswordInput struct {
	Token    string
	Password string
}

// ResetPasswordOutput はパスワードリセットの出力を定義します
type ResetPasswordOutput struct {
	Message string
}

// ResetPasswordCommand はパスワードリセットコマンドです
type ResetPasswordCommand struct {
	userRepo               repository.UserRepository
	passwordResetTokenRepo repository.PasswordResetTokenRepository
	sessionRepo            repository.SessionRepository
	txManager              repository.TransactionManager
}

// NewResetPasswordCommand は新しいResetPasswordCommandを作成します
func NewResetPasswordCommand(
	userRepo repository.UserRepository,
	passwordResetTokenRepo repository.PasswordResetTokenRepository,
	sessionRepo repository.SessionRepository,
	txManager repository.TransactionManager,
) *ResetPasswordCommand {
	return &ResetPasswordCommand{
		userRepo:               userRepo,
		passwordResetTokenRepo: passwordResetTokenRepo,
		sessionRepo:            sessionRepo,
		txManager:              txManager,
	}
}

// Execute はパスワードリセットを実行します
func (c *ResetPasswordCommand) Execute(ctx context.Context, input ResetPasswordInput) (*ResetPasswordOutput, error) {
	// 1. トークンの検証
	if input.Token == "" {
		return nil, apperror.NewValidationError("token is required", nil)
	}

	// 2. トークンを検索
	token, err := c.passwordResetTokenRepo.FindByToken(ctx, input.Token)
	if err != nil {
		return nil, apperror.NewValidationError("invalid or expired token", nil)
	}

	// 3. トークンの有効性チェック
	if token.IsUsed() {
		return nil, apperror.NewValidationError("invalid or expired token", nil)
	}

	if token.IsExpired() {
		return nil, apperror.NewValidationError("token has expired", nil)
	}

	// 4. ユーザーを取得
	user, err := c.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}

	// 5. パスワードのバリデーション
	password, err := valueobject.NewPassword(input.Password, user.Email.String())
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 6. トランザクションでパスワード更新とトークン使用済みマーク
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// パスワード更新
		user.PasswordHash = password.Hash()
		user.UpdatedAt = time.Now()

		if err := c.userRepo.Update(ctx, user); err != nil {
			return err
		}

		// トークンを使用済みにマーク
		if err := c.passwordResetTokenRepo.MarkAsUsed(ctx, token.ID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 全セッションを無効化（セキュリティ: パスワードリセット後は再ログインが必要）
	_ = c.sessionRepo.DeleteByUserID(ctx, token.UserID)

	return &ResetPasswordOutput{
		Message: "password reset successfully",
	}, nil
}
