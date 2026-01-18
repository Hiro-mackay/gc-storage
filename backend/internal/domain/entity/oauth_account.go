package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// OAuthAccount はOAuth連携アカウントエンティティを定義します
type OAuthAccount struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Provider       valueobject.OAuthProvider
	ProviderUserID string
	Email          string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsTokenExpired はアクセストークンが期限切れかを判定します
func (o *OAuthAccount) IsTokenExpired() bool {
	return time.Now().After(o.TokenExpiresAt)
}

// NeedsRefresh はトークンリフレッシュが必要かを判定します
func (o *OAuthAccount) NeedsRefresh() bool {
	// 期限の5分前からリフレッシュ推奨
	return time.Now().Add(5 * time.Minute).After(o.TokenExpiresAt)
}
