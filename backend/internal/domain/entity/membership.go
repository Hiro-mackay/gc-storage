package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// Membership はグループメンバーシップエンティティ
type Membership struct {
	ID       uuid.UUID
	GroupID  uuid.UUID
	UserID   uuid.UUID
	Role     valueobject.GroupRole
	JoinedAt time.Time
}

// NewMembership は新しいメンバーシップを作成します
func NewMembership(
	groupID uuid.UUID,
	userID uuid.UUID,
	role valueobject.GroupRole,
) *Membership {
	return &Membership{
		ID:       uuid.New(),
		GroupID:  groupID,
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	}
}

// NewOwnerMembership はオーナー用のメンバーシップを作成します
func NewOwnerMembership(groupID uuid.UUID, userID uuid.UUID) *Membership {
	return NewMembership(groupID, userID, valueobject.GroupRoleOwner)
}

// ReconstructMembership はDBからメンバーシップを復元します
func ReconstructMembership(
	id uuid.UUID,
	groupID uuid.UUID,
	userID uuid.UUID,
	role valueobject.GroupRole,
	joinedAt time.Time,
) *Membership {
	return &Membership{
		ID:       id,
		GroupID:  groupID,
		UserID:   userID,
		Role:     role,
		JoinedAt: joinedAt,
	}
}

// IsOwner はオーナーかを判定します
func (m *Membership) IsOwner() bool {
	return m.Role.IsOwner()
}

// CanInvite は招待可能かを判定します
func (m *Membership) CanInvite() bool {
	return m.Role.CanInvite()
}

// CanManageMembers はメンバー管理可能かを判定します
func (m *Membership) CanManageMembers() bool {
	return m.Role.CanManageMembers()
}

// CanChangeRoleTo は指定されたロールへの変更が可能かを判定します
func (m *Membership) CanChangeRoleTo(targetRole valueobject.GroupRole) bool {
	return m.Role.CanAssign(targetRole)
}

// ChangeRole はロールを変更します
func (m *Membership) ChangeRole(newRole valueobject.GroupRole) {
	m.Role = newRole
}

// CanLeave は脱退可能かを判定します
// 以下の条件を満たす場合に脱退可能:
// - 指定されたグループのメンバーシップである
// - 指定されたユーザーのメンバーシップである
// - オーナーではない（オーナーは所有権譲渡が必要）
func (m *Membership) CanLeave(groupID, userID uuid.UUID) bool {
	if !m.BelongsToGroup(groupID) {
		return false
	}
	if !m.IsMember(userID) {
		return false
	}
	return !m.IsOwner()
}

// PromoteToOwner はオーナーに昇格します
func (m *Membership) PromoteToOwner() {
	m.Role = valueobject.GroupRoleOwner
}

// DemoteToContributor はContributorに降格します
func (m *Membership) DemoteToContributor() {
	m.Role = valueobject.GroupRoleContributor
}

// IsMember は指定ユーザーのメンバーシップかを判定します
func (m *Membership) IsMember(userID uuid.UUID) bool {
	return m.UserID == userID
}

// BelongsToGroup は指定グループのメンバーシップかを判定します
func (m *Membership) BelongsToGroup(groupID uuid.UUID) bool {
	return m.GroupID == groupID
}

// MembershipWithUser はメンバーシップとユーザー情報を結合した構造体
type MembershipWithUser struct {
	Membership *Membership
	User       *User
}
