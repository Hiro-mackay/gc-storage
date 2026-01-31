package di

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
)

// Middlewares はアプリケーションのミドルウェアを保持します
type Middlewares struct {
	SessionAuth *middleware.SessionAuthMiddleware
	RateLimit   *middleware.RateLimitMiddleware
	Permission  *middleware.PermissionMiddleware
}

// NewMiddlewares はContainerから全てのミドルウェアを初期化します
func NewMiddlewares(c *Container) *Middlewares {
	m := &Middlewares{
		SessionAuth: middleware.NewSessionAuthMiddleware(c.SessionRepo, c.UserRepo),
		RateLimit:   middleware.NewRateLimitMiddleware(c.RateLimiter),
	}

	// Permission Middleware (if PermissionResolver is initialized)
	if c.PermissionResolver != nil {
		m.Permission = middleware.NewPermissionMiddleware(c.PermissionResolver)
	}

	return m
}
