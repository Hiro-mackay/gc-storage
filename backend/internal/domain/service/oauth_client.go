package service

import (
	"context"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// OAuthUserInfo はOAuthプロバイダーから取得したユーザー情報を表します
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	Name           string
	AvatarURL      string
}

// OAuthTokens はOAuthプロバイダーから取得したトークンを表します
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
}

// OAuthClient はOAuthプロバイダーとの通信を行うインターフェースです
type OAuthClient interface {
	// ExchangeCode は認可コードをトークンに交換します
	ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error)

	// GetUserInfo はアクセストークンを使用してユーザー情報を取得します
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)

	// Provider はプロバイダー種別を返します
	Provider() valueobject.OAuthProvider
}

// OAuthClientFactory はプロバイダーに応じたOAuthClientを生成するファクトリーです
type OAuthClientFactory interface {
	// GetClient は指定されたプロバイダーのOAuthClientを返します
	GetClient(provider valueobject.OAuthProvider) (OAuthClient, error)
}
