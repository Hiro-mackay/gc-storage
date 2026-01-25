package authz

import "errors"

var (
	ErrInvalidRole = errors.New("invalid role")
)

// Role はリソースに対するロールを表す型
// ロール階層: Owner > Content Manager > Contributor > Viewer
type Role string

const (
	RoleViewer         Role = "viewer"
	RoleContributor    Role = "contributor"
	RoleContentManager Role = "content_manager"
	RoleOwner          Role = "owner"
)

// NewRole は文字列からRoleを生成します
func NewRole(r string) (Role, error) {
	role := Role(r)
	if !role.IsValid() {
		return "", ErrInvalidRole
	}
	return role, nil
}

// IsValid はロールが有効かを判定します
func (r Role) IsValid() bool {
	switch r {
	case RoleViewer, RoleContributor, RoleContentManager, RoleOwner:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (r Role) String() string {
	return string(r)
}

// Level はロールのレベルを返します（比較用）
func (r Role) Level() int {
	switch r {
	case RoleOwner:
		return 4
	case RoleContentManager:
		return 3
	case RoleContributor:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// Includes は指定されたロールを含むかを判定します（階層的に）
func (r Role) Includes(other Role) bool {
	return r.Level() >= other.Level()
}

// CanGrant は指定されたロールを付与可能かを判定します
// Ownerは直接付与不可（所有権譲渡を使用）
func (r Role) CanGrant(targetRole Role) bool {
	if targetRole == RoleOwner {
		return false
	}
	return r.Level() > targetRole.Level()
}

// GrantableRoles は付与可能なロールの一覧を返します
func (r Role) GrantableRoles() []Role {
	var roles []Role
	for _, role := range []Role{RoleViewer, RoleContributor, RoleContentManager} {
		if r.CanGrant(role) {
			roles = append(roles, role)
		}
	}
	return roles
}

// Permissions はロールに含まれる権限一覧を返します
func (r Role) Permissions() []Permission {
	switch r {
	case RoleOwner:
		return []Permission{
			PermFileRead, PermFileWrite, PermFileDelete, PermFileShare, PermFileMove, PermFileDownload,
			PermFolderRead, PermFolderWrite, PermFolderCreate, PermFolderDelete, PermFolderShare, PermFolderMoveIn, PermFolderMoveOut,
			PermManageAccess,
		}
	case RoleContentManager:
		return []Permission{
			PermFileRead, PermFileWrite, PermFileDelete, PermFileShare, PermFileMove, PermFileDownload,
			PermFolderRead, PermFolderWrite, PermFolderCreate, PermFolderDelete, PermFolderShare, PermFolderMoveIn, PermFolderMoveOut,
		}
	case RoleContributor:
		return []Permission{
			PermFileRead, PermFileWrite, PermFileDownload,
			PermFolderRead, PermFolderWrite, PermFolderCreate, PermFolderMoveIn,
		}
	case RoleViewer:
		return []Permission{
			PermFileRead, PermFileDownload,
			PermFolderRead,
		}
	default:
		return nil
	}
}

// HasPermission はロールが指定された権限を持つかを判定します
func (r Role) HasPermission(perm Permission) bool {
	for _, p := range r.Permissions() {
		if p == perm {
			return true
		}
	}
	return false
}

// AllGrantableRoles は付与可能な全てのロールを返します（Ownerを除く）
func AllGrantableRoles() []Role {
	return []Role{RoleViewer, RoleContributor, RoleContentManager}
}
