package jwt

import "time"

// Config はJWT設定を定義します
type Config struct {
	SecretKey          string        // HMAC署名用シークレットキー
	Issuer             string        // 発行者
	Audience           []string      // 対象者
	AccessTokenExpiry  time.Duration // アクセストークン有効期限
	RefreshTokenExpiry time.Duration // リフレッシュトークン有効期限
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
	return Config{
		Issuer:             "gc-storage",
		Audience:           []string{"gc-storage-api"},
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
}

// Validate は設定を検証します
func (c Config) Validate() error {
	if c.SecretKey == "" {
		return ErrSecretKeyRequired
	}
	if len(c.SecretKey) < 32 {
		return ErrSecretKeyTooShort
	}
	return nil
}
