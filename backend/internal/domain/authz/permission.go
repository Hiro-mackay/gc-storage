package authz

import "errors"

var (
	ErrInvalidPermission = errors.New("invalid permission")
)

// Permission は個別の権限を表す型
type Permission string

// File permissions
const (
	PermFileRead     Permission = "file:read"
	PermFileWrite    Permission = "file:write"
	PermFileDelete   Permission = "file:delete"
	PermFileShare    Permission = "file:share"
	PermFileMove     Permission = "file:move"
	PermFileDownload Permission = "file:download"
)

// Folder permissions
const (
	PermFolderRead    Permission = "folder:read"
	PermFolderWrite   Permission = "folder:write"
	PermFolderCreate  Permission = "folder:create"
	PermFolderDelete  Permission = "folder:delete"
	PermFolderShare   Permission = "folder:share"
	PermFolderMoveIn  Permission = "folder:move_in"
	PermFolderMoveOut Permission = "folder:move_out"
)

// Admin permissions
const (
	PermManageAccess Permission = "manage:access"
)

// allPermissions は全ての有効な権限
var allPermissions = map[Permission]bool{
	PermFileRead:      true,
	PermFileWrite:     true,
	PermFileDelete:    true,
	PermFileShare:     true,
	PermFileMove:      true,
	PermFileDownload:  true,
	PermFolderRead:    true,
	PermFolderWrite:   true,
	PermFolderCreate:  true,
	PermFolderDelete:  true,
	PermFolderShare:   true,
	PermFolderMoveIn:  true,
	PermFolderMoveOut: true,
	PermManageAccess:  true,
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
	case PermFileRead, PermFileWrite, PermFileDelete, PermFileShare, PermFileMove, PermFileDownload:
		return true
	default:
		return false
	}
}

// IsFolderPermission はフォルダ権限かを判定します
func (p Permission) IsFolderPermission() bool {
	switch p {
	case PermFolderRead, PermFolderWrite, PermFolderCreate, PermFolderDelete, PermFolderShare, PermFolderMoveIn, PermFolderMoveOut:
		return true
	default:
		return false
	}
}
