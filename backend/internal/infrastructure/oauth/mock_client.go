package oauth

import (
	"context"
	"fmt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// MockOAuthClient はテスト用のモックOAuthクライアントです
type MockOAuthClient struct {
	provider     valueobject.OAuthProvider
	tokens       *service.OAuthTokens
	userInfo     *service.OAuthUserInfo
	exchangeErr  error
	userInfoErr  error
}

// MockOAuthClientOption はMockOAuthClientの設定オプションです
type MockOAuthClientOption func(*MockOAuthClient)

// WithExchangeError は認可コード交換時のエラーを設定します
func WithExchangeError(err error) MockOAuthClientOption {
	return func(c *MockOAuthClient) {
		c.exchangeErr = err
	}
}

// WithUserInfoError はユーザー情報取得時のエラーを設定します
func WithUserInfoError(err error) MockOAuthClientOption {
	return func(c *MockOAuthClient) {
		c.userInfoErr = err
	}
}

// WithTokens はトークンを設定します
func WithTokens(tokens *service.OAuthTokens) MockOAuthClientOption {
	return func(c *MockOAuthClient) {
		c.tokens = tokens
	}
}

// WithUserInfo はユーザー情報を設定します
func WithUserInfo(userInfo *service.OAuthUserInfo) MockOAuthClientOption {
	return func(c *MockOAuthClient) {
		c.userInfo = userInfo
	}
}

// NewMockOAuthClient は新しいMockOAuthClientを作成します
func NewMockOAuthClient(provider valueobject.OAuthProvider, opts ...MockOAuthClientOption) *MockOAuthClient {
	c := &MockOAuthClient{
		provider: provider,
		tokens: &service.OAuthTokens{
			AccessToken:  "mock-access-token",
			RefreshToken: "mock-refresh-token",
			ExpiresIn:    3600,
		},
		userInfo: &service.OAuthUserInfo{
			ProviderUserID: "mock-provider-user-id",
			Email:          "mock@example.com",
			Name:           "Mock User",
			AvatarURL:      "https://example.com/avatar.png",
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ExchangeCode は認可コードをトークンに交換します（モック実装）
func (c *MockOAuthClient) ExchangeCode(ctx context.Context, code string) (*service.OAuthTokens, error) {
	if c.exchangeErr != nil {
		return nil, c.exchangeErr
	}

	// "invalid-code"の場合はエラーを返す
	if code == "invalid-code" {
		return nil, fmt.Errorf("invalid authorization code")
	}

	return c.tokens, nil
}

// GetUserInfo はアクセストークンを使用してユーザー情報を取得します（モック実装）
func (c *MockOAuthClient) GetUserInfo(ctx context.Context, accessToken string) (*service.OAuthUserInfo, error) {
	if c.userInfoErr != nil {
		return nil, c.userInfoErr
	}

	return c.userInfo, nil
}

// Provider はプロバイダー種別を返します
func (c *MockOAuthClient) Provider() valueobject.OAuthProvider {
	return c.provider
}

// SetUserInfo はユーザー情報を設定します（テスト用）
func (c *MockOAuthClient) SetUserInfo(userInfo *service.OAuthUserInfo) {
	c.userInfo = userInfo
}

// SetTokens はトークンを設定します（テスト用）
func (c *MockOAuthClient) SetTokens(tokens *service.OAuthTokens) {
	c.tokens = tokens
}

// SetExchangeError は認可コード交換時のエラーを設定します（テスト用）
func (c *MockOAuthClient) SetExchangeError(err error) {
	c.exchangeErr = err
}

// SetUserInfoError はユーザー情報取得時のエラーを設定します（テスト用）
func (c *MockOAuthClient) SetUserInfoError(err error) {
	c.userInfoErr = err
}

// インターフェースの実装を保証
var _ service.OAuthClient = (*MockOAuthClient)(nil)
