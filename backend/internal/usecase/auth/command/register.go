package command

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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

// RegisterInput は登録の入力を定義します
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// RegisterOutput は登録の出力を定義します
type RegisterOutput struct {
	UserID uuid.UUID
}

// RegisterCommand はユーザー登録コマンドです
type RegisterCommand struct {
	userRepo                   repository.UserRepository
	emailVerificationTokenRepo repository.EmailVerificationTokenRepository
	txManager                  repository.TransactionManager
	emailSender                service.EmailSender
	appURL                     string
}

// NewRegisterCommand は新しいRegisterCommandを作成します
func NewRegisterCommand(
	userRepo repository.UserRepository,
	emailVerificationTokenRepo repository.EmailVerificationTokenRepository,
	txManager repository.TransactionManager,
	emailSender service.EmailSender,
	appURL string,
) *RegisterCommand {
	return &RegisterCommand{
		userRepo:                   userRepo,
		emailVerificationTokenRepo: emailVerificationTokenRepo,
		txManager:                  txManager,
		emailSender:                emailSender,
		appURL:                     appURL,
	}
}

// Execute はユーザー登録を実行します
func (c *RegisterCommand) Execute(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	// 1. メールアドレスのバリデーション
	email, err := valueobject.NewEmail(input.Email)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. パスワードのバリデーション
	password, err := valueobject.NewPassword(input.Password, input.Email)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 3. メールアドレスの重複チェック
	exists, err := c.userRepo.Exists(ctx, email)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}
	if exists {
		return nil, apperror.NewConflictError("email already exists")
	}

	var user *entity.User
	var verificationToken *entity.EmailVerificationToken

	// 4. トランザクションでユーザー作成
	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		now := time.Now()

		// ユーザー作成
		user = &entity.User{
			ID:            uuid.New(),
			Email:         email,
			Name:          input.Name,
			PasswordHash:  password.Hash(),
			Status:        entity.UserStatusPending,
			EmailVerified: false,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := c.userRepo.Create(ctx, user); err != nil {
			return err
		}

		// 確認トークン作成（emailVerificationTokenRepoが設定されている場合のみ）
		if c.emailVerificationTokenRepo != nil {
			verificationToken = &entity.EmailVerificationToken{
				ID:        uuid.New(),
				UserID:    user.ID,
				Token:     generateSecureToken(),
				ExpiresAt: now.Add(24 * time.Hour),
				CreatedAt: now,
			}

			if err := c.emailVerificationTokenRepo.Create(ctx, verificationToken); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 5. 確認メール送信（トランザクション外で実行、失敗しても登録は成功扱い）
	if c.emailSender != nil && verificationToken != nil {
		verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", c.appURL, verificationToken.Token)
		if err := c.emailSender.SendEmailVerification(ctx, user.Email.String(), user.Name, verifyURL); err != nil {
			// メール送信失敗はログに記録するが、エラーは返さない
			slog.Error("failed to send verification email", "error", err, "user_id", user.ID)
		}
	}

	return &RegisterOutput{UserID: user.ID}, nil
}

// generateSecureToken はセキュアなトークンを生成します
func generateSecureToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
