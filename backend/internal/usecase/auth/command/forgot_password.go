package command

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ForgotPasswordInput はパスワードリセットリクエストの入力を定義します
type ForgotPasswordInput struct {
	Email string
}

// ForgotPasswordOutput はパスワードリセットリクエストの出力を定義します
type ForgotPasswordOutput struct {
	Message string
}

// ForgotPasswordCommand はパスワードリセットリクエストコマンドです
type ForgotPasswordCommand struct {
	userRepo               repository.UserRepository
	passwordResetTokenRepo repository.PasswordResetTokenRepository
	emailSender            service.EmailSender
	appURL                 string
}

// NewForgotPasswordCommand は新しいForgotPasswordCommandを作成します
func NewForgotPasswordCommand(
	userRepo repository.UserRepository,
	passwordResetTokenRepo repository.PasswordResetTokenRepository,
	emailSender service.EmailSender,
	appURL string,
) *ForgotPasswordCommand {
	return &ForgotPasswordCommand{
		userRepo:               userRepo,
		passwordResetTokenRepo: passwordResetTokenRepo,
		emailSender:            emailSender,
		appURL:                 appURL,
	}
}

// Execute はパスワードリセットリクエストを実行します
func (c *ForgotPasswordCommand) Execute(ctx context.Context, input ForgotPasswordInput) (*ForgotPasswordOutput, error) {
	// セキュリティメッセージ（存在有無に関わらず同じメッセージを返す）
	securityMessage := "If your email address is registered, a password reset email has been sent."

	// 1. メールアドレスのバリデーション
	email, err := valueobject.NewEmail(input.Email)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. ユーザーを検索
	user, err := c.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// セキュリティ上、存在しないメールアドレスでも同じレスポンスを返す
		return &ForgotPasswordOutput{
			Message: securityMessage,
		}, nil
	}

	// 3. 既存のトークンを削除
	if err := c.passwordResetTokenRepo.DeleteByUserID(ctx, user.ID); err != nil {
		slog.Warn("failed to delete existing password reset tokens", "error", err, "user_id", user.ID)
	}

	// 4. 新しいトークンを作成（1時間有効）
	now := time.Now()
	token := &entity.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     generateSecureToken(),
		ExpiresAt: now.Add(1 * time.Hour),
		CreatedAt: now,
	}

	if err := c.passwordResetTokenRepo.Create(ctx, token); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 5. メール送信（emailSenderが設定されている場合のみ）
	if c.emailSender != nil {
		resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", c.appURL, token.Token)
		if err := c.emailSender.SendPasswordReset(ctx, user.Email.String(), user.Name, resetURL); err != nil {
			// メール送信失敗はログに記録するが、エラーは返さない
			slog.Error("failed to send password reset email", "error", err, "user_id", user.ID)
		}
	}

	return &ForgotPasswordOutput{
		Message: securityMessage,
	}, nil
}
