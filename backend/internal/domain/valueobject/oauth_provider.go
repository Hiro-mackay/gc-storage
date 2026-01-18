package valueobject

// OAuthProvider はOAuthプロバイダーを表す値オブジェクトです
type OAuthProvider string

const (
	OAuthProviderGoogle OAuthProvider = "google"
	OAuthProviderGitHub OAuthProvider = "github"
)

// String はプロバイダー名を返します
func (p OAuthProvider) String() string {
	return string(p)
}

// IsValid はプロバイダーが有効かを判定します
func (p OAuthProvider) IsValid() bool {
	switch p {
	case OAuthProviderGoogle, OAuthProviderGitHub:
		return true
	default:
		return false
	}
}

// AllOAuthProviders は全プロバイダーを返します
func AllOAuthProviders() []OAuthProvider {
	return []OAuthProvider{
		OAuthProviderGoogle,
		OAuthProviderGitHub,
	}
}
