package authz

import "errors"

var (
	ErrInvalidPermission = errors.New("invalid permission")
)

// Permission は個別の権限を表す型
type Permission string

// File permissions
const (
	PermFileRead            Permission = "file:read"
	PermFileWrite           Permission = "file:write"
	PermFileRename          Permission = "file:rename"
	PermFileDelete          Permission = "file:delete"
	PermFileRestore         Permission = "file:restore"
	PermFileMoveIn          Permission = "file:move_in"
	PermFileMoveOut         Permission = "file:move_out"
	PermFileShare           Permission = "file:share"
	PermFileMove            Permission = "file:move"
	PermFileDownload        Permission = "file:download"
	PermFilePermanentDelete Permission = "file:permanent_delete"
)

// Folder permissions
const (
	PermFolderRead    Permission = "folder:read"
	PermFolderWrite   Permission = "folder:write"
	PermFolderCreate  Permission = "folder:create"
	PermFolderRename  Permission = "folder:rename"
	PermFolderDelete  Permission = "folder:delete"
	PermFolderShare   Permission = "folder:share"
	PermFolderMoveIn  Permission = "folder:move_in"
	PermFolderMoveOut Permission = "folder:move_out"
)

// Permission permissions
const (
	PermPermissionRead   Permission = "permission:read"
	PermPermissionGrant  Permission = "permission:grant"
	PermPermissionRevoke Permission = "permission:revoke"
)

// Root permissions
const (
	PermRootDelete Permission = "root:delete"
)

// Admin permissions
const (
	PermManageAccess Permission = "manage:access"
)

// allPermissions は全ての有効な権限
var allPermissions = map[Permission]bool{
	PermFileRead:            true,
	PermFileWrite:           true,
	PermFileRename:          true,
	PermFileDelete:          true,
	PermFileRestore:         true,
	PermFileMoveIn:          true,
	PermFileMoveOut:         true,
	PermFileShare:           true,
	PermFileMove:            true,
	PermFileDownload:        true,
	PermFilePermanentDelete: true,
	PermFolderRead:          true,
	PermFolderWrite:         true,
	PermFolderCreate:        true,
	PermFolderRename:        true,
	PermFolderDelete:        true,
	PermFolderShare:         true,
	PermFolderMoveIn:        true,
	PermFolderMoveOut:       true,
	PermPermissionRead:      true,
	PermPermissionGrant:     true,
	PermPermissionRevoke:    true,
	PermRootDelete:          true,
	PermManageAccess:        true,
}

// NewPermission は文字列からPermissionを生成します
func NewPermission(p string) (Permission, error) {
	perm := Permission(p)
	if !perm.IsValid() {
		return "", ErrInvalidPermission
	}
	return perm, nil
}

// IsValid は権限が有効かを判定します
func (p Permission) IsValid() bool {
	return allPermissions[p]
}

// String は文字列を返します
func (p Permission) String() string {
	return string(p)
}

// IsFilePermission はファイル権限かを判定します
func (p Permission) IsFilePermission() bool {
	switch p {
	case PermFileRead, PermFileWrite, PermFileRename, PermFileDelete, PermFileRestore,
		PermFileMoveIn, PermFileMoveOut, PermFileShare, PermFileMove, PermFileDownload,
		PermFilePermanentDelete:
		return true
	default:
		return false
	}
}

// IsFolderPermission はフォルダ権限かを判定します
func (p Permission) IsFolderPermission() bool {
	switch p {
	case PermFolderRead, PermFolderWrite, PermFolderCreate, PermFolderRename, PermFolderDelete,
		PermFolderShare, PermFolderMoveIn, PermFolderMoveOut:
		return true
	default:
		return false
	}
}

// IsPermissionPermission はパーミッション管理権限かを判定します
func (p Permission) IsPermissionPermission() bool {
	switch p {
	case PermPermissionRead, PermPermissionGrant, PermPermissionRevoke:
		return true
	default:
		return false
	}
}
