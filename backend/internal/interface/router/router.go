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
	r.setupStorageRoutes(api)
	r.setupGroupRoutes(api)
	r.setupPermissionRoutes(api)
	r.setupShareLinkRoutes(api)
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

	// Password set route (authenticated, for OAuth-only users)
	passwordGroup.POST("/set", r.handlers.Auth.SetPassword, r.middlewares.JWTAuth.Authenticate())

	// Auth routes (authenticated)
	authGroup.POST("/logout", r.handlers.Auth.Logout, r.middlewares.JWTAuth.Authenticate())
}

// setupUserRoutes はユーザー関連ルートを設定します
func (r *Router) setupUserRoutes(api *echo.Group) {
	api.GET("/me", r.handlers.Auth.Me, r.middlewares.JWTAuth.Authenticate())

	// Profile routes (authenticated)
	meGroup := api.Group("/me", r.middlewares.JWTAuth.Authenticate())
	meGroup.GET("/profile", r.handlers.Profile.GetProfile)
	meGroup.PUT("/profile", r.handlers.Profile.UpdateProfile)
}

// setupStorageRoutes はストレージ関連ルートを設定します
func (r *Router) setupStorageRoutes(api *echo.Group) {
	// Folder routes (authenticated)
	if r.handlers.Folder != nil {
		foldersGroup := api.Group("/folders", r.middlewares.JWTAuth.Authenticate())
		foldersGroup.POST("", r.handlers.Folder.CreateFolder)
		foldersGroup.GET("/root/contents", r.handlers.Folder.ListFolderContents)
		foldersGroup.GET("/:id", r.handlers.Folder.GetFolder)
		foldersGroup.GET("/:id/contents", r.handlers.Folder.ListFolderContents)
		foldersGroup.GET("/:id/ancestors", r.handlers.Folder.GetAncestors)
		foldersGroup.PATCH("/:id/rename", r.handlers.Folder.RenameFolder)
		foldersGroup.PATCH("/:id/move", r.handlers.Folder.MoveFolder)
		foldersGroup.DELETE("/:id", r.handlers.Folder.DeleteFolder)
	}

	// File routes (authenticated)
	if r.handlers.File != nil {
		filesGroup := api.Group("/files", r.middlewares.JWTAuth.Authenticate())
		filesGroup.POST("/upload", r.handlers.File.InitiateUpload)
		filesGroup.GET("/upload/:sessionId", r.handlers.File.GetUploadStatus)
		filesGroup.GET("/:id/download", r.handlers.File.GetDownloadURL)
		filesGroup.GET("/:id/versions", r.handlers.File.ListFileVersions)
		filesGroup.PATCH("/:id/rename", r.handlers.File.RenameFile)
		filesGroup.PATCH("/:id/move", r.handlers.File.MoveFile)
		filesGroup.POST("/:id/trash", r.handlers.File.TrashFile)

		// Upload completion webhook (unauthenticated for MinIO webhook)
		api.POST("/files/upload/complete", r.handlers.File.CompleteUpload)

		// Trash routes (authenticated)
		trashGroup := api.Group("/trash", r.middlewares.JWTAuth.Authenticate())
		trashGroup.GET("", r.handlers.File.ListTrash)
		trashGroup.POST("/:id/restore", r.handlers.File.RestoreFile)
	}
}

// setupGroupRoutes はグループ関連ルートを設定します
func (r *Router) setupGroupRoutes(api *echo.Group) {
	if r.handlers.Group == nil {
		return
	}

	// Group routes (authenticated)
	groupsGroup := api.Group("/groups", r.middlewares.JWTAuth.Authenticate())
	groupsGroup.POST("", r.handlers.Group.CreateGroup)
	groupsGroup.GET("", r.handlers.Group.ListMyGroups)
	groupsGroup.GET("/:id", r.handlers.Group.GetGroup)
	groupsGroup.PATCH("/:id", r.handlers.Group.UpdateGroup)
	groupsGroup.DELETE("/:id", r.handlers.Group.DeleteGroup)

	// Group member routes
	groupsGroup.GET("/:id/members", r.handlers.Group.ListMembers)
	groupsGroup.DELETE("/:id/members/:userId", r.handlers.Group.RemoveMember)
	groupsGroup.PATCH("/:id/members/:userId/role", r.handlers.Group.ChangeRole)

	// Group invitation routes
	groupsGroup.POST("/:id/invitations", r.handlers.Group.InviteMember)
	groupsGroup.GET("/:id/invitations", r.handlers.Group.ListInvitations)
	groupsGroup.DELETE("/:id/invitations/:invitationId", r.handlers.Group.CancelInvitation)

	// Group actions
	groupsGroup.POST("/:id/leave", r.handlers.Group.LeaveGroup)
	groupsGroup.POST("/:id/transfer", r.handlers.Group.TransferOwnership)

	// Invitation routes (authenticated)
	invitationsGroup := api.Group("/invitations", r.middlewares.JWTAuth.Authenticate())
	invitationsGroup.GET("/pending", r.handlers.Group.ListPendingInvitations)
	invitationsGroup.POST("/:token/accept", r.handlers.Group.AcceptInvitation)
	invitationsGroup.POST("/:token/decline", r.handlers.Group.DeclineInvitation)
}

// setupPermissionRoutes は権限関連ルートを設定します
func (r *Router) setupPermissionRoutes(api *echo.Group) {
	if r.handlers.Permission == nil {
		return
	}

	// File permission routes (authenticated)
	filesGroup := api.Group("/files", r.middlewares.JWTAuth.Authenticate())
	filesGroup.GET("/:id/permissions", r.handlers.Permission.ListFileGrants)
	filesGroup.POST("/:id/permissions", r.handlers.Permission.GrantFileRole)

	// Folder permission routes (authenticated)
	foldersGroup := api.Group("/folders", r.middlewares.JWTAuth.Authenticate())
	foldersGroup.GET("/:id/permissions", r.handlers.Permission.ListFolderGrants)
	foldersGroup.POST("/:id/permissions", r.handlers.Permission.GrantFolderRole)

	// Permission grant routes (authenticated)
	permissionsGroup := api.Group("/permissions", r.middlewares.JWTAuth.Authenticate())
	permissionsGroup.DELETE("/:id", r.handlers.Permission.RevokeGrant)
}

// setupShareLinkRoutes は共有リンク関連ルートを設定します
func (r *Router) setupShareLinkRoutes(api *echo.Group) {
	if r.handlers.ShareLink == nil {
		return
	}

	// Share link creation routes (authenticated)
	filesGroup := api.Group("/files", r.middlewares.JWTAuth.Authenticate())
	filesGroup.POST("/:id/share", r.handlers.ShareLink.CreateFileShareLink)
	filesGroup.GET("/:id/share", r.handlers.ShareLink.ListFileShareLinks)

	foldersGroup := api.Group("/folders", r.middlewares.JWTAuth.Authenticate())
	foldersGroup.POST("/:id/share", r.handlers.ShareLink.CreateFolderShareLink)
	foldersGroup.GET("/:id/share", r.handlers.ShareLink.ListFolderShareLinks)

	// Share link management routes (authenticated)
	shareLinksGroup := api.Group("/share-links", r.middlewares.JWTAuth.Authenticate())
	shareLinksGroup.DELETE("/:id", r.handlers.ShareLink.RevokeShareLink)

	// Public share link access routes (no authentication required)
	shareGroup := api.Group("/share")
	shareGroup.GET("/:token", r.handlers.ShareLink.GetShareLinkInfo)
	shareGroup.POST("/:token/access", r.handlers.ShareLink.AccessShareLink)
}
