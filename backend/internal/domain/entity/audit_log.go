package entity

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction は監査ログのアクション種別を定義します
type AuditAction string

const (
	AuditActionLogin          AuditAction = "auth.login"
	AuditActionLogout         AuditAction = "auth.logout"
	AuditActionRegister       AuditAction = "auth.register"
	AuditActionPasswordChange AuditAction = "auth.password_change"

	AuditActionFileUpload   AuditAction = "file.upload"
	AuditActionFileDownload AuditAction = "file.download"
	AuditActionFileRename   AuditAction = "file.rename"
	AuditActionFileMove     AuditAction = "file.move"
	AuditActionFileTrash    AuditAction = "file.trash"
	AuditActionFileRestore  AuditAction = "file.restore"

	AuditActionFolderCreate AuditAction = "folder.create"
	AuditActionFolderRename AuditAction = "folder.rename"
	AuditActionFolderMove   AuditAction = "folder.move"
	AuditActionFolderDelete AuditAction = "folder.delete"

	AuditActionGroupCreate       AuditAction = "group.create"
	AuditActionGroupDelete       AuditAction = "group.delete"
	AuditActionGroupMemberInvite AuditAction = "group.member_invite"
	AuditActionGroupMemberRemove AuditAction = "group.member_remove"
	AuditActionGroupMemberLeave  AuditAction = "group.member_leave"

	AuditActionShareLinkCreate AuditAction = "share.link_create"
	AuditActionShareLinkRevoke AuditAction = "share.link_revoke"
	AuditActionShareLinkAccess AuditAction = "share.link_access"

	AuditActionPermissionGrant  AuditAction = "permission.grant"
	AuditActionPermissionRevoke AuditAction = "permission.revoke"
)

// AuditResourceType はリソースの種類を定義します
type AuditResourceType string

const (
	AuditResourceUser       AuditResourceType = "user"
	AuditResourceFile       AuditResourceType = "file"
	AuditResourceFolder     AuditResourceType = "folder"
	AuditResourceGroup      AuditResourceType = "group"
	AuditResourceShareLink  AuditResourceType = "share_link"
	AuditResourcePermission AuditResourceType = "permission"
)

// AuditLog は監査ログエントリを表します
type AuditLog struct {
	ID           uuid.UUID
	UserID       *uuid.UUID
	Action       AuditAction
	ResourceType AuditResourceType
	ResourceID   *uuid.UUID
	Details      map[string]interface{}
	IPAddress    string
	UserAgent    string
	RequestID    string
	CreatedAt    time.Time
}
