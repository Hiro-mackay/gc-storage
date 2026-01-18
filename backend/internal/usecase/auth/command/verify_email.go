package command

import (
	"context"
	"time"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// VerifyEmailInput はメール確認の入力を定義します
type VerifyEmailInput struct {
	Token string
}

// VerifyEmailOutput はメール確認の出力を定義します
type VerifyEmailOutput struct {
	Message string
}

// VerifyEmailCommand はメール確認コマンドです
type VerifyEmailCommand struct {
	userRepo                   repository.UserRepository
	emailVerificationTokenRepo repository.EmailVerificationTokenRepository
	txManager                  repository.TransactionManager
}

// NewVerifyEmailCommand は新しいVerifyEmailCommandを作成します
func NewVerifyEmailCommand(
	userRepo repository.UserRepository,
	emailVerificationTokenRepo repository.EmailVerificationTokenRepository,
	txManager repository.TransactionManager,
) *VerifyEmailCommand {
	return &VerifyEmailCommand{
		userRepo:                   userRepo,
		emailVerificationTokenRepo: emailVerificationTokenRepo,
		txManager:                  txManager,
	}
}

// Execute はメール確認を実行します
func (c *VerifyEmailCommand) Execute(ctx context.Context, input VerifyEmailInput) (*VerifyEmailOutput, error) {
	// 1. トークンの検証
	if input.Token == "" {
		return nil, apperror.NewValidationError("token is required", nil)
	}

	// 2. トークンを検索
	token, err := c.emailVerificationTokenRepo.FindByToken(ctx, input.Token)
	if err != nil {
		return nil, apperror.NewValidationError("invalid or expired token", nil)
	}

	// 3. トークンの有効期限チェック
	if token.IsExpired() {
		// 期限切れトークンを削除
		_ = c.emailVerificationTokenRepo.Delete(ctx, token.ID)
		return nil, apperror.NewValidationError("token has expired", nil)
	}

	// 4. ユーザーを取得
	user, err := c.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, apperror.NewNotFoundError("user")
	}

	// 5. 既に確認済みの場合
	if user.EmailVerified {
		// トークンを削除
		_ = c.emailVerificationTokenRepo.Delete(ctx, token.ID)
		return &VerifyEmailOutput{
			Message: "email already verified",
		}, nil
	}

	// 6. トランザクションでユーザー更新とトークン削除
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// ユーザーのステータスを active に、メール確認済みに更新
		user.Status = entity.UserStatusActive
		user.EmailVerified = true
		user.UpdatedAt = time.Now()

		if err := c.userRepo.Update(ctx, user); err != nil {
			return err
		}

		// トークンを削除
		if err := c.emailVerificationTokenRepo.Delete(ctx, token.ID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	return &VerifyEmailOutput{
		Message: "email verified successfully",
	}, nil
}
