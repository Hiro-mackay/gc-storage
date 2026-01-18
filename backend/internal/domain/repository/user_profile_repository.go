package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// UserProfileRepository はユーザープロファイルリポジトリインターフェースを定義します
type UserProfileRepository interface {
	// Create はユーザープロファイルを作成します
	Create(ctx context.Context, profile *entity.UserProfile) error

	// Update はユーザープロファイルを更新します
	Update(ctx context.Context, profile *entity.UserProfile) error

	// FindByUserID はユーザーIDでプロファイルを検索します
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.UserProfile, error)

	// Upsert はプロファイルを作成または更新します
	Upsert(ctx context.Context, profile *entity.UserProfile) error
}
