package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config はアプリケーション全体の設定を定義します
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	OAuth    OAuthConfig
	Security SecurityConfig
	App      AppConfig
}

// ServerConfig はサーバー設定を定義します
type ServerConfig struct {
	Port  int
	Debug bool
}

// DatabaseConfig はデータベース設定を定義します
type DatabaseConfig struct {
	URL string
}

// RedisConfig はRedis設定を定義します
type RedisConfig struct {
	URL string
}

// JWTConfig はJWT設定を定義します
type JWTConfig struct {
	SecretKey          string
	Issuer             string
	Audience           []string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// OAuthConfig はOAuth設定を定義します
type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
}

// SecurityConfig はセキュリティ設定を定義します
type SecurityConfig struct {
	CORSOrigins []string
	EnableHSTS  bool
}

// AppConfig はアプリケーション設定を定義します
type AppConfig struct {
	URL string
}

// Load は環境変数から設定を読み込みます
func Load() (*Config, error) {
	port := 8080
	if p := os.Getenv("SERVER_PORT"); p != "" {
		if _, err := fmt.Sscanf(p, "%d", &port); err != nil {
			return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
		}
	}

	appURL := getEnv("APP_URL", "http://localhost:3000")

	return &Config{
		Server: ServerConfig{
			Port:  port,
			Debug: os.Getenv("DEBUG") == "true",
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/gc_storage?sslmode=disable"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379/0"),
		},
		JWT: JWTConfig{
			SecretKey:          getEnv("JWT_SECRET_KEY", "your-secret-key-change-in-production"),
			Issuer:             "gc-storage",
			Audience:           []string{"gc-storage-api"},
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
		},
		OAuth: OAuthConfig{
			GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", appURL+"/auth/callback/google"),
			GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			GitHubRedirectURL:  getEnv("GITHUB_REDIRECT_URL", appURL+"/auth/callback/github"),
		},
		Security: SecurityConfig{
			CORSOrigins: parseCORSOrigins(getEnv("CORS_ORIGINS", appURL)),
			EnableHSTS:  os.Getenv("ENABLE_HSTS") == "true",
		},
		App: AppConfig{
			URL: appURL,
		},
	}, nil
}

// parseCORSOrigins はカンマ区切りのオリジン文字列をスライスに変換します
func parseCORSOrigins(origins string) []string {
	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// getEnv は環境変数を取得し、存在しない場合はデフォルト値を返します
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
