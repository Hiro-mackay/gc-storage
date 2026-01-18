package di

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/handler"
)

// Handlers はアプリケーションのハンドラーを保持します
type Handlers struct {
	Health  *handler.HealthHandler
	Auth    *handler.AuthHandler
	Profile *handler.ProfileHandler
	// 今後追加されるハンドラー:
	// Storage *handler.StorageHandler
	// Share   *handler.ShareHandler
	// Group   *handler.GroupHandler
}

// NewHandlers はContainerから全てのハンドラーを初期化します
func NewHandlers(c *Container) *Handlers {
	// Health Handler
	healthHandler := handler.NewHealthHandler()
	if c.PgClient != nil {
		healthHandler.RegisterChecker("postgres", c.PgClient)
	}
	if c.RedisClient != nil {
		healthHandler.RegisterChecker("redis", c.RedisClient)
	}

	// Auth Handler
	authHandler := handler.NewAuthHandler(
		c.Auth.Register,
		c.Auth.Login,
		c.Auth.RefreshToken,
		c.Auth.Logout,
		c.Auth.VerifyEmail,
		c.Auth.ResendEmailVerification,
		c.Auth.ForgotPassword,
		c.Auth.ResetPassword,
		c.Auth.ChangePassword,
		c.Auth.SetPassword,
		c.Auth.OAuthLogin,
		c.Auth.GetUser,
	)

	// Profile Handler
	profileHandler := handler.NewProfileHandler(
		c.Profile.GetProfile,
		c.Profile.UpdateProfile,
	)

	return &Handlers{
		Health:  healthHandler,
		Auth:    authHandler,
		Profile: profileHandler,
	}
}

// NewHandlersForTest はテスト用にハンドラーを初期化します（HealthHandlerなし）
func NewHandlersForTest(c *Container) *Handlers {
	authHandler := handler.NewAuthHandler(
		c.Auth.Register,
		c.Auth.Login,
		c.Auth.RefreshToken,
		c.Auth.Logout,
		c.Auth.VerifyEmail,
		c.Auth.ResendEmailVerification,
		c.Auth.ForgotPassword,
		c.Auth.ResetPassword,
		c.Auth.ChangePassword,
		c.Auth.SetPassword,
		c.Auth.OAuthLogin,
		c.Auth.GetUser,
	)

	profileHandler := handler.NewProfileHandler(
		c.Profile.GetProfile,
		c.Profile.UpdateProfile,
	)

	return &Handlers{
		Health:  nil, // テストではHealthHandlerは不要
		Auth:    authHandler,
		Profile: profileHandler,
	}
}
