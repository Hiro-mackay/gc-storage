package di

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
)

// Middlewares はアプリケーションのミドルウェアを保持します
type Middlewares struct {
	JWTAuth    *middleware.JWTAuthMiddleware
	RateLimit  *middleware.RateLimitMiddleware
	Permission *middleware.PermissionMiddleware
}

// NewMiddlewares はContainerから全てのミドルウェアを初期化します
func NewMiddlewares(c *Container) *Middlewares {
	m := &Middlewares{
		JWTAuth:   middleware.NewJWTAuthMiddleware(c.JWTService, c.JWTBlacklist),
		RateLimit: middleware.NewRateLimitMiddleware(c.RateLimiter),
	}

	// Permission Middleware (if PermissionResolver is initialized)
	if c.PermissionResolver != nil {
		m.Permission = middleware.NewPermissionMiddleware(c.PermissionResolver)
	}

	return m
}
