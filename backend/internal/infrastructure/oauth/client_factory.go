package oauth

import (
	"fmt"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// Config はOAuthクライアントの設定を保持します
type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
}

// ClientFactory はOAuthClientFactoryの実装です
type ClientFactory struct {
	config  Config
	clients map[valueobject.OAuthProvider]service.OAuthClient
}

// NewClientFactory は新しいClientFactoryを作成します
func NewClientFactory(config Config) *ClientFactory {
	factory := &ClientFactory{
		config:  config,
		clients: make(map[valueobject.OAuthProvider]service.OAuthClient),
	}

	// Googleクライアントの初期化
	if config.GoogleClientID != "" {
		factory.clients[valueobject.OAuthProviderGoogle] = NewGoogleClient(
			config.GoogleClientID,
			config.GoogleClientSecret,
			config.GoogleRedirectURL,
		)
	}

	// GitHubクライアントの初期化
	if config.GitHubClientID != "" {
		factory.clients[valueobject.OAuthProviderGitHub] = NewGitHubClient(
			config.GitHubClientID,
			config.GitHubClientSecret,
			config.GitHubRedirectURL,
		)
	}

	return factory
}

// GetClient は指定されたプロバイダーのOAuthClientを返します
func (f *ClientFactory) GetClient(provider valueobject.OAuthProvider) (service.OAuthClient, error) {
	client, ok := f.clients[provider]
	if !ok {
		return nil, fmt.Errorf("unsupported oauth provider: %s", provider)
	}
	return client, nil
}

// RegisterClient はカスタムクライアントを登録します（主にテスト用）
func (f *ClientFactory) RegisterClient(provider valueobject.OAuthProvider, client service.OAuthClient) {
	f.clients[provider] = client
}

// インターフェースの実装を保証
var _ service.OAuthClientFactory = (*ClientFactory)(nil)
