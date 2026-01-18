package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// VerificationTokenRepository はメール確認トークンリポジトリインターフェースを定義します
type VerificationTokenRepository interface {
	// Create はトークンを作成します
	Create(ctx context.Context, token *entity.VerificationToken) error

	// FindByToken はトークン文字列でトークンを検索します
	FindByToken(ctx context.Context, token string) (*entity.VerificationToken, error)

	// FindByUserID はユーザーIDでトークンを検索します
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.VerificationToken, error)

	// Delete はトークンを削除します
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID はユーザーの全トークンを削除します
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// DeleteExpired は期限切れトークンを削除します
	DeleteExpired(ctx context.Context) error
}

// PasswordResetTokenRepository はパスワードリセットトークンリポジトリインターフェースを定義します
type PasswordResetTokenRepository interface {
	// Create はトークンを作成します
	Create(ctx context.Context, token *entity.PasswordResetToken) error

	// FindByToken はトークン文字列でトークンを検索します
	FindByToken(ctx context.Context, token string) (*entity.PasswordResetToken, error)

	// MarkAsUsed はトークンを使用済みにします
	MarkAsUsed(ctx context.Context, id uuid.UUID) error

	// Delete はトークンを削除します
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID はユーザーの全トークンを削除します
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// DeleteExpired は期限切れトークンを削除します
	DeleteExpired(ctx context.Context) error
}
