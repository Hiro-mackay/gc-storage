package di

import (
	infraRepo "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/repository"
	authcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	authqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/query"
)

// AuthUseCases はAuth関連のUseCaseを保持します
type AuthUseCases struct {
	// Commands
	Register               *authcmd.RegisterCommand
	Login                  *authcmd.LoginCommand
	RefreshToken           *authcmd.RefreshTokenCommand
	Logout                 *authcmd.LogoutCommand
	VerifyEmail            *authcmd.VerifyEmailCommand
	ResendEmailVerification *authcmd.ResendEmailVerificationCommand
	ForgotPassword         *authcmd.ForgotPasswordCommand
	ResetPassword          *authcmd.ResetPasswordCommand
	ChangePassword         *authcmd.ChangePasswordCommand
	SetPassword            *authcmd.SetPasswordCommand
	OAuthLogin             *authcmd.OAuthLoginCommand

	// Queries
	GetUser *authqry.GetUserQuery
}

// NewAuthUseCases は新しいAuthUseCasesを作成します
func NewAuthUseCases(c *Container, appURL string) *AuthUseCases {
	// RegisterCommandに必要なフォルダリポジトリを作成
	folderRepo := infraRepo.NewFolderRepository(c.TxManager)
	folderClosureRepo := infraRepo.NewFolderClosureRepository(c.TxManager)

	return &AuthUseCases{
		// Commands
		Register: authcmd.NewRegisterCommand(
			c.UserRepo,
			folderRepo,
			folderClosureRepo,
			c.EmailVerificationTokenRepo,
			c.TxManager,
			c.EmailService,
			appURL,
		),
		Login: authcmd.NewLoginCommand(
			c.UserRepo,
			c.SessionRepo,
			c.JWTService,
		),
		RefreshToken: authcmd.NewRefreshTokenCommand(
			c.UserRepo,
			c.SessionRepo,
			c.JWTService,
			c.JWTBlacklist,
		),
		Logout: authcmd.NewLogoutCommand(
			c.SessionRepo,
			c.JWTBlacklist,
		),
		VerifyEmail: authcmd.NewVerifyEmailCommand(
			c.UserRepo,
			c.EmailVerificationTokenRepo,
			c.TxManager,
		),
		ResendEmailVerification: authcmd.NewResendEmailVerificationCommand(
			c.UserRepo,
			c.EmailVerificationTokenRepo,
			c.EmailService,
			appURL,
		),
		ForgotPassword: authcmd.NewForgotPasswordCommand(
			c.UserRepo,
			c.PasswordResetTokenRepo,
			c.EmailService,
			appURL,
		),
		ResetPassword: authcmd.NewResetPasswordCommand(
			c.UserRepo,
			c.PasswordResetTokenRepo,
			c.TxManager,
		),
		ChangePassword: authcmd.NewChangePasswordCommand(
			c.UserRepo,
		),
		SetPassword: authcmd.NewSetPasswordCommand(
			c.UserRepo,
		),
		OAuthLogin: authcmd.NewOAuthLoginCommand(
			c.UserRepo,
			c.UserProfileRepo,
			c.OAuthAccountRepo,
			c.OAuthFactory,
			c.TxManager,
			c.SessionRepo,
			c.JWTService,
		),

		// Queries
		GetUser: authqry.NewGetUserQuery(c.UserRepo),
	}
}
