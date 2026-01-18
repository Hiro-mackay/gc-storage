package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// OAuthAccountRepository はOAuthアカウントリポジトリインターフェースを定義します
type OAuthAccountRepository interface {
	// Create はOAuthアカウントを作成します
	Create(ctx context.Context, account *entity.OAuthAccount) error

	// Update はOAuthアカウントを更新します
	Update(ctx context.Context, account *entity.OAuthAccount) error

	// FindByID はIDでOAuthアカウントを検索します
	FindByID(ctx context.Context, id uuid.UUID) (*entity.OAuthAccount, error)

	// FindByUserID はユーザーIDでOAuthアカウント一覧を取得します
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.OAuthAccount, error)

	// FindByProviderAndUserID はプロバイダーとプロバイダーユーザーIDでOAuthアカウントを検索します
	FindByProviderAndUserID(ctx context.Context, provider valueobject.OAuthProvider, providerUserID string) (*entity.OAuthAccount, error)

	// Delete はOAuthアカウントを削除します
	Delete(ctx context.Context, id uuid.UUID) error
}
