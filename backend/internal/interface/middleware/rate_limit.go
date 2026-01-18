package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RateLimitType はレート制限の種類を定義します
type RateLimitType string

const (
	// 認証関連
	RateLimitAuthLogin  RateLimitType = "auth_login"
	RateLimitAuthSignup RateLimitType = "auth_signup"

	// API関連
	RateLimitAPIDefault RateLimitType = "api_default"
	RateLimitAPIUpload  RateLimitType = "api_upload"
	RateLimitAPISearch  RateLimitType = "api_search"

	// 共有リンク
	RateLimitShareAccess RateLimitType = "share_access"
)

// レート制限設定
var rateLimitConfigs = map[RateLimitType]cache.RateLimitConfig{
	RateLimitAuthLogin: {
		Type:     "auth:login",
		Requests: 5,
		Window:   time.Minute,
	},
	RateLimitAuthSignup: {
		Type:     "auth:signup",
		Requests: 3,
		Window:   time.Minute,
	},
	RateLimitAPIDefault: {
		Type:     "api:default",
		Requests: 100,
		Window:   time.Minute,
	},
	RateLimitAPIUpload: {
		Type:     "api:upload",
		Requests: 10,
		Window:   time.Minute,
	},
	RateLimitAPISearch: {
		Type:     "api:search",
		Requests: 30,
		Window:   time.Minute,
	},
	RateLimitShareAccess: {
		Type:     "share:access",
		Requests: 20,
		Window:   time.Minute,
	},
}

// RateLimitMiddleware はレート制限ミドルウェアを提供します
type RateLimitMiddleware struct {
	limiter *cache.RateLimiter
}

// NewRateLimitMiddleware は新しいRateLimitMiddlewareを作成します
func NewRateLimitMiddleware(limiter *cache.RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// ByIP はIPアドレスでレート制限するミドルウェアを返します
func (m *RateLimitMiddleware) ByIP(limitType RateLimitType) echo.MiddlewareFunc {
	config := rateLimitConfigs[limitType]
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			identifier := c.RealIP()
			result, err := m.limiter.Allow(c.Request().Context(), identifier, config)
			if err != nil {
				// レート制限チェックに失敗した場合はリクエストを許可
				return next(c)
			}

			// レスポンスヘッダーを設定
			setRateLimitHeaders(c, result)

			if !result.Allowed {
				return apperror.NewTooManyRequestsError("rate limit exceeded")
			}

			return next(c)
		}
	}
}

// ByUser はユーザーIDでレート制限するミドルウェアを返します
func (m *RateLimitMiddleware) ByUser(limitType RateLimitType) echo.MiddlewareFunc {
	config := rateLimitConfigs[limitType]
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := GetUserID(c)
			if userID == "" {
				// ユーザーIDがない場合はIPでフォールバック
				userID = c.RealIP()
			}

			result, err := m.limiter.Allow(c.Request().Context(), userID, config)
			if err != nil {
				// レート制限チェックに失敗した場合はリクエストを許可
				return next(c)
			}

			// レスポンスヘッダーを設定
			setRateLimitHeaders(c, result)

			if !result.Allowed {
				return apperror.NewTooManyRequestsError("rate limit exceeded")
			}

			return next(c)
		}
	}
}

// setRateLimitHeaders はレート制限ヘッダーを設定します
func setRateLimitHeaders(c echo.Context, result *cache.RateLimitResult) {
	c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	c.Response().Header().Set("X-RateLimit-Reset", result.ResetAt.Format("2006-01-02T15:04:05Z"))
}
