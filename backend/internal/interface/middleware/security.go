package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeaders はセキュリティヘッダーを設定するミドルウェアを返します
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// XSS対策
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")

			// クリックジャッキング対策
			c.Response().Header().Set("X-Frame-Options", "DENY")

			// CSP
			c.Response().Header().Set("Content-Security-Policy", "default-src 'self'")

			// Referrer Policy
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions Policy
			c.Response().Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			return next(c)
		}
	}
}

// SecurityHeadersConfig はセキュリティヘッダー設定を定義します
type SecurityHeadersConfig struct {
	EnableHSTS    bool
	HSTSMaxAge    int
	CSPDirectives string
}

// DefaultSecurityHeadersConfig はデフォルトセキュリティヘッダー設定を返します
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		EnableHSTS:    false,
		HSTSMaxAge:    31536000, // 1年
		CSPDirectives: "default-src 'self'",
	}
}

// SecurityHeadersWithConfig は設定付きセキュリティヘッダーミドルウェアを返します
func SecurityHeadersWithConfig(cfg SecurityHeadersConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// XSS対策
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")

			// クリックジャッキング対策
			c.Response().Header().Set("X-Frame-Options", "DENY")

			// HTTPS強制（本番環境）
			if cfg.EnableHSTS {
				c.Response().Header().Set("Strict-Transport-Security",
					"max-age=31536000; includeSubDomains")
			}

			// CSP
			c.Response().Header().Set("Content-Security-Policy", cfg.CSPDirectives)

			// Referrer Policy
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions Policy
			c.Response().Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			return next(c)
		}
	}
}
