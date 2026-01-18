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

// ResendEmailVerificationInput は確認メール再送の入力を定義します
type ResendEmailVerificationInput struct {
	Email string
}

// ResendEmailVerificationOutput は確認メール再送の出力を定義します
type ResendEmailVerificationOutput struct {
	Message string
}

// ResendEmailVerificationCommand は確認メール再送コマンドです
type ResendEmailVerificationCommand struct {
	userRepo                   repository.UserRepository
	emailVerificationTokenRepo repository.EmailVerificationTokenRepository
	emailSender                service.EmailSender
	appURL                     string
}

// NewResendEmailVerificationCommand は新しいResendEmailVerificationCommandを作成します
func NewResendEmailVerificationCommand(
	userRepo repository.UserRepository,
	emailVerificationTokenRepo repository.EmailVerificationTokenRepository,
	emailSender service.EmailSender,
	appURL string,
) *ResendEmailVerificationCommand {
	return &ResendEmailVerificationCommand{
		userRepo:                   userRepo,
		emailVerificationTokenRepo: emailVerificationTokenRepo,
		emailSender:                emailSender,
		appURL:                     appURL,
	}
}

// Execute は確認メール再送を実行します
func (c *ResendEmailVerificationCommand) Execute(ctx context.Context, input ResendEmailVerificationInput) (*ResendEmailVerificationOutput, error) {
	// 1. メールアドレスのバリデーション
	email, err := valueobject.NewEmail(input.Email)
	if err != nil {
		// セキュリティ上、存在しないメールアドレスでも同じレスポンスを返す
		return &ResendEmailVerificationOutput{
			Message: "If your email address is registered, a verification email has been sent.",
		}, nil
	}

	// 2. ユーザーを検索
	user, err := c.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// セキュリティ上、存在しないメールアドレスでも同じレスポンスを返す
		return &ResendEmailVerificationOutput{
			Message: "If your email address is registered, a verification email has been sent.",
		}, nil
	}

	// 3. 既に確認済みの場合
	if user.EmailVerified {
		return &ResendEmailVerificationOutput{
			Message: "If your email address is registered, a verification email has been sent.",
		}, nil
	}

	// 4. 既存のトークンを削除
	_ = c.emailVerificationTokenRepo.DeleteByUserID(ctx, user.ID)

	// 5. 新しいトークンを作成
	now := time.Now()
	token := &entity.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     generateSecureToken(),
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	if err := c.emailVerificationTokenRepo.Create(ctx, token); err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 6. メール送信（emailSenderが設定されている場合のみ）
	if c.emailSender != nil {
		verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", c.appURL, token.Token)
		if err := c.emailSender.SendEmailVerification(ctx, user.Email.String(), user.Name, verifyURL); err != nil {
			// メール送信失敗はログに記録するが、エラーは返さない
			slog.Error("failed to send verification email", "error", err, "user_id", user.ID)
		}
	}

	return &ResendEmailVerificationOutput{
		Message: "If your email address is registered, a verification email has been sent.",
	}, nil
}
