package di

import (
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/handler"
)

// Handlers はアプリケーションのハンドラーを保持します
type Handlers struct {
	Health     *handler.HealthHandler
	Auth       *handler.AuthHandler
	Profile    *handler.ProfileHandler
	Folder     *handler.FolderHandler
	File       *handler.FileHandler
	Upload     *handler.UploadHandler
	Trash      *handler.TrashHandler
	Group      *handler.GroupHandler
	Permission *handler.PermissionHandler
	ShareLink  *handler.ShareLinkHandler
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
		c.Auth.Logout,
		c.Auth.VerifyEmail,
		c.Auth.ResendEmailVerification,
		c.Auth.ForgotPassword,
		c.Auth.ResetPassword,
		c.Auth.ChangePassword,
		c.Auth.SetPassword,
		c.Auth.OAuthLogin,
	)

	// Profile Handler
	profileHandler := handler.NewProfileHandler(
		c.Profile.GetProfile,
		c.Profile.UpdateProfile,
	)

	// Folder Handler (if Storage is initialized)
	var folderHandler *handler.FolderHandler
	var fileHandler *handler.FileHandler
	var uploadHandler *handler.UploadHandler
	var trashHandler *handler.TrashHandler
	if c.Storage != nil {
		folderHandler = handler.NewFolderHandler(
			c.Storage.CreateFolder,
			c.Storage.RenameFolder,
			c.Storage.MoveFolder,
			c.Storage.DeleteFolder,
			c.Storage.GetFolder,
			c.Storage.ListFolderContents,
			c.Storage.GetAncestors,
		)
		fileHandler = handler.NewFileHandler(
			c.Storage.RenameFile,
			c.Storage.MoveFile,
			c.Storage.GetDownloadURL,
			c.Storage.ListFileVersions,
		)
		uploadHandler = handler.NewUploadHandler(
			c.Storage.InitiateUpload,
			c.Storage.CompleteUpload,
			c.Storage.AbortUpload,
			c.Storage.GetUploadStatus,
		)
		trashHandler = handler.NewTrashHandler(
			c.Storage.TrashFile,
			c.Storage.RestoreFile,
			c.Storage.PermanentlyDeleteFile,
			c.Storage.EmptyTrash,
			c.Storage.ListTrash,
		)
	}

	// Group Handler (if Collaboration is initialized)
	var groupHandler *handler.GroupHandler
	if c.Collaboration != nil {
		groupHandler = handler.NewGroupHandler(
			c.Collaboration.CreateGroup,
			c.Collaboration.UpdateGroup,
			c.Collaboration.DeleteGroup,
			c.Collaboration.InviteMember,
			c.Collaboration.AcceptInvitation,
			c.Collaboration.DeclineInvitation,
			c.Collaboration.CancelInvitation,
			c.Collaboration.RemoveMember,
			c.Collaboration.LeaveGroup,
			c.Collaboration.ChangeRole,
			c.Collaboration.TransferOwnership,
			c.Collaboration.GetGroup,
			c.Collaboration.ListMyGroups,
			c.Collaboration.ListMembers,
			c.Collaboration.ListInvitations,
			c.Collaboration.ListPendingInvitations,
		)
	}

	// Permission Handler (if Authz is initialized)
	var permissionHandler *handler.PermissionHandler
	if c.Authz != nil {
		permissionHandler = handler.NewPermissionHandler(
			c.Authz.GrantRole,
			c.Authz.RevokeGrant,
			c.Authz.ListGrants,
			c.Authz.CheckPermission,
		)
	}

	// ShareLink Handler (if Sharing is initialized)
	var shareLinkHandler *handler.ShareLinkHandler
	if c.Sharing != nil {
		shareLinkHandler = handler.NewShareLinkHandler(
			c.Sharing.CreateShareLink,
			c.Sharing.RevokeShareLink,
			c.Sharing.UpdateShareLink,
			c.Sharing.AccessShareLink,
			c.Sharing.ListShareLinks,
			c.Sharing.GetShareLinkHistory,
			c.Sharing.GetDownloadViaShare,
			c.config.App.URL,
		)
	}

	return &Handlers{
		Health:     healthHandler,
		Auth:       authHandler,
		Profile:    profileHandler,
		Folder:     folderHandler,
		File:       fileHandler,
		Upload:     uploadHandler,
		Trash:      trashHandler,
		Group:      groupHandler,
		Permission: permissionHandler,
		ShareLink:  shareLinkHandler,
	}
}

// NewHandlersForTest はテスト用にハンドラーを初期化します（HealthHandlerなし）
func NewHandlersForTest(c *Container) *Handlers {
	authHandler := handler.NewAuthHandler(
		c.Auth.Register,
		c.Auth.Login,
		c.Auth.Logout,
		c.Auth.VerifyEmail,
		c.Auth.ResendEmailVerification,
		c.Auth.ForgotPassword,
		c.Auth.ResetPassword,
		c.Auth.ChangePassword,
		c.Auth.SetPassword,
		c.Auth.OAuthLogin,
	)

	profileHandler := handler.NewProfileHandler(
		c.Profile.GetProfile,
		c.Profile.UpdateProfile,
	)

	// Storage Handlers (if Storage is initialized)
	var folderHandler *handler.FolderHandler
	var fileHandler *handler.FileHandler
	var uploadHandler *handler.UploadHandler
	var trashHandler *handler.TrashHandler
	if c.Storage != nil {
		folderHandler = handler.NewFolderHandler(
			c.Storage.CreateFolder,
			c.Storage.RenameFolder,
			c.Storage.MoveFolder,
			c.Storage.DeleteFolder,
			c.Storage.GetFolder,
			c.Storage.ListFolderContents,
			c.Storage.GetAncestors,
		)
		fileHandler = handler.NewFileHandler(
			c.Storage.RenameFile,
			c.Storage.MoveFile,
			c.Storage.GetDownloadURL,
			c.Storage.ListFileVersions,
		)
		uploadHandler = handler.NewUploadHandler(
			c.Storage.InitiateUpload,
			c.Storage.CompleteUpload,
			c.Storage.AbortUpload,
			c.Storage.GetUploadStatus,
		)
		trashHandler = handler.NewTrashHandler(
			c.Storage.TrashFile,
			c.Storage.RestoreFile,
			c.Storage.PermanentlyDeleteFile,
			c.Storage.EmptyTrash,
			c.Storage.ListTrash,
		)
	}

	// Group Handler (if Collaboration is initialized)
	var groupHandler *handler.GroupHandler
	if c.Collaboration != nil {
		groupHandler = handler.NewGroupHandler(
			c.Collaboration.CreateGroup,
			c.Collaboration.UpdateGroup,
			c.Collaboration.DeleteGroup,
			c.Collaboration.InviteMember,
			c.Collaboration.AcceptInvitation,
			c.Collaboration.DeclineInvitation,
			c.Collaboration.CancelInvitation,
			c.Collaboration.RemoveMember,
			c.Collaboration.LeaveGroup,
			c.Collaboration.ChangeRole,
			c.Collaboration.TransferOwnership,
			c.Collaboration.GetGroup,
			c.Collaboration.ListMyGroups,
			c.Collaboration.ListMembers,
			c.Collaboration.ListInvitations,
			c.Collaboration.ListPendingInvitations,
		)
	}

	// Permission Handler (if Authz is initialized)
	var permissionHandler *handler.PermissionHandler
	if c.Authz != nil {
		permissionHandler = handler.NewPermissionHandler(
			c.Authz.GrantRole,
			c.Authz.RevokeGrant,
			c.Authz.ListGrants,
			c.Authz.CheckPermission,
		)
	}

	// ShareLink Handler (if Sharing is initialized)
	var shareLinkHandler *handler.ShareLinkHandler
	if c.Sharing != nil {
		shareLinkHandler = handler.NewShareLinkHandler(
			c.Sharing.CreateShareLink,
			c.Sharing.RevokeShareLink,
			c.Sharing.UpdateShareLink,
			c.Sharing.AccessShareLink,
			c.Sharing.ListShareLinks,
			c.Sharing.GetShareLinkHistory,
			c.Sharing.GetDownloadViaShare,
			c.config.App.URL,
		)
	}

	return &Handlers{
		Health:     nil, // テストではHealthHandlerは不要
		Auth:       authHandler,
		Profile:    profileHandler,
		Folder:     folderHandler,
		File:       fileHandler,
		Upload:     uploadHandler,
		Trash:      trashHandler,
		Group:      groupHandler,
		Permission: permissionHandler,
		ShareLink:  shareLinkHandler,
	}
}
