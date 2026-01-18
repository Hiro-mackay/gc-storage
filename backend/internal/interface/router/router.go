package router

import (
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/di"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
)

// Router はルート定義を管理します
type Router struct {
	echo        *echo.Echo
	handlers    *di.Handlers
	middlewares *di.Middlewares
}

// NewRouter は新しいRouterを作成します
func NewRouter(e *echo.Echo, handlers *di.Handlers, middlewares *di.Middlewares) *Router {
	return &Router{
		echo:        e,
		handlers:    handlers,
		middlewares: middlewares,
	}
}

// Setup は全てのルートを設定します
func (r *Router) Setup() {
	r.setupHealthRoutes()
	r.setupAPIRoutes()
}

// setupHealthRoutes はヘルスチェックルートを設定します
func (r *Router) setupHealthRoutes() {
	if r.handlers.Health == nil {
		return
	}
	r.echo.GET("/health", r.handlers.Health.Check)
	r.echo.GET("/ready", r.handlers.Health.Ready)
}

// setupAPIRoutes はAPIルートを設定します
func (r *Router) setupAPIRoutes() {
	api := r.echo.Group("/api/v1")

	// Debug route
	api.GET("/", func(c echo.Context) error {
		return presenter.OK(c, map[string]string{
			"message": "GC Storage API v1",
		})
	})

	r.setupAuthRoutes(api)
	r.setupUserRoutes(api)
}

// setupAuthRoutes は認証関連ルートを設定します
func (r *Router) setupAuthRoutes(api *echo.Group) {
	authGroup := api.Group("/auth")

	// Public auth routes
	authGroup.POST("/register", r.handlers.Auth.Register,
		r.middlewares.RateLimit.ByIP(middleware.RateLimitAuthSignup))
	authGroup.POST("/login", r.handlers.Auth.Login,
		r.middlewares.RateLimit.ByIP(middleware.RateLimitAuthLogin))
	authGroup.POST("/refresh", r.handlers.Auth.Refresh)

	// OAuth routes (public)
	authGroup.POST("/oauth/:provider", r.handlers.Auth.OAuthLogin,
		r.middlewares.RateLimit.ByIP(middleware.RateLimitAuthLogin))

	// Email verification routes (public)
	emailGroup := authGroup.Group("/email")
	emailGroup.POST("/verify", r.handlers.Auth.VerifyEmail)
	emailGroup.POST("/resend", r.handlers.Auth.ResendEmailVerification,
		r.middlewares.RateLimit.ByIP(middleware.RateLimitAuthSignup))

	// Password reset routes (public)
	passwordGroup := authGroup.Group("/password")
	passwordGroup.POST("/forgot", r.handlers.Auth.ForgotPassword,
		r.middlewares.RateLimit.ByIP(middleware.RateLimitAuthLogin))
	passwordGroup.POST("/reset", r.handlers.Auth.ResetPassword,
		r.middlewares.RateLimit.ByIP(middleware.RateLimitAuthLogin))

	// Password change route (authenticated)
	passwordGroup.POST("/change", r.handlers.Auth.ChangePassword, r.middlewares.JWTAuth.Authenticate())

	// Auth routes (authenticated)
	authGroup.POST("/logout", r.handlers.Auth.Logout, r.middlewares.JWTAuth.Authenticate())
}

// setupUserRoutes はユーザー関連ルートを設定します
func (r *Router) setupUserRoutes(api *echo.Group) {
	api.GET("/me", r.handlers.Auth.Me, r.middlewares.JWTAuth.Authenticate())
}
