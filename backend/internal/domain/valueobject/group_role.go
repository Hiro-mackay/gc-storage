package valueobject

import "errors"

var (
	ErrInvalidGroupRole = errors.New("invalid group role")
)

// GroupRole はグループ内のメンバーシップロールを表す値オブジェクト
// Note: これはグループ内のロールであり、リソース権限のロールとは異なる
type GroupRole string

const (
	GroupRoleViewer      GroupRole = "viewer"
	GroupRoleContributor GroupRole = "contributor"
	GroupRoleOwner       GroupRole = "owner"
)

// NewGroupRole は文字列からGroupRoleを生成します
func NewGroupRole(role string) (GroupRole, error) {
	r := GroupRole(role)
	if !r.IsValid() {
		return "", ErrInvalidGroupRole
	}
	return r, nil
}

// IsValid はロールが有効かを判定します
func (r GroupRole) IsValid() bool {
	switch r {
	case GroupRoleViewer, GroupRoleContributor, GroupRoleOwner:
		return true
	default:
		return false
	}
}

// String は文字列を返します
func (r GroupRole) String() string {
	return string(r)
}

// IsOwner はオーナーかを判定します
func (r GroupRole) IsOwner() bool {
	return r == GroupRoleOwner
}

// CanInvite は招待可能かを判定します（Owner, Contributorのみ）
func (r GroupRole) CanInvite() bool {
	return r == GroupRoleOwner || r == GroupRoleContributor
}

// CanManageMembers はメンバー管理可能かを判定します（Ownerのみ）
func (r GroupRole) CanManageMembers() bool {
	return r == GroupRoleOwner
}

// Level はロールのレベルを返します（比較用）
func (r GroupRole) Level() int {
	switch r {
	case GroupRoleOwner:
		return 3
	case GroupRoleContributor:
		return 2
	case GroupRoleViewer:
		return 1
	default:
		return 0
	}
}

// CanAssign は指定されたロールを割り当て可能かを判定します
// 自分より低いロールのみ割り当て可能
func (r GroupRole) CanAssign(target GroupRole) bool {
	// Ownerロールは直接割り当て不可（譲渡を使用）
	if target == GroupRoleOwner {
		return false
	}
	return r.Level() > target.Level()
}

// InvitableRoles は招待時に指定可能なロールを返します
func InvitableRoles() []GroupRole {
	return []GroupRole{GroupRoleViewer, GroupRoleContributor}
}
