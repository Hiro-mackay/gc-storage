package authz

import (
	"time"

	"github.com/google/uuid"
)

// PermissionGrant は権限付与を表すエンティティ
type PermissionGrant struct {
	ID           uuid.UUID
	ResourceType ResourceType
	ResourceID   uuid.UUID
	GranteeType  GranteeType
	GranteeID    uuid.UUID
	Role         Role
	GrantedBy    uuid.UUID
	GrantedAt    time.Time
}

// NewPermissionGrant は新しいPermissionGrantを生成します
func NewPermissionGrant(
	resourceType ResourceType,
	resourceID uuid.UUID,
	granteeType GranteeType,
	granteeID uuid.UUID,
	role Role,
	grantedBy uuid.UUID,
) *PermissionGrant {
	return &PermissionGrant{
		ID:           uuid.New(),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		GranteeType:  granteeType,
		GranteeID:    granteeID,
		Role:         role,
		GrantedBy:    grantedBy,
		GrantedAt:    time.Now(),
	}
}

// ReconstructPermissionGrant はDBから復元するためのコンストラクタ
func ReconstructPermissionGrant(
	id uuid.UUID,
	resourceType ResourceType,
	resourceID uuid.UUID,
	granteeType GranteeType,
	granteeID uuid.UUID,
	role Role,
	grantedBy uuid.UUID,
	grantedAt time.Time,
) *PermissionGrant {
	return &PermissionGrant{
		ID:           id,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		GranteeType:  granteeType,
		GranteeID:    granteeID,
		Role:         role,
		GrantedBy:    grantedBy,
		GrantedAt:    grantedAt,
	}
}

// IsForUser はユーザー向けの付与かを判定します
func (pg *PermissionGrant) IsForUser() bool {
	return pg.GranteeType.IsUser()
}

// IsForGroup はグループ向けの付与かを判定します
func (pg *PermissionGrant) IsForGroup() bool {
	return pg.GranteeType.IsGroup()
}

// IsForFile はファイルへの付与かを判定します
func (pg *PermissionGrant) IsForFile() bool {
	return pg.ResourceType == ResourceTypeFile
}

// IsForFolder はフォルダへの付与かを判定します
func (pg *PermissionGrant) IsForFolder() bool {
	return pg.ResourceType == ResourceTypeFolder
}

// HasPermission は指定された権限を持つかを判定します
func (pg *PermissionGrant) HasPermission(perm Permission) bool {
	return pg.Role.HasPermission(perm)
}

// Permissions は付与されている権限の一覧を返します
func (pg *PermissionGrant) Permissions() []Permission {
	return pg.Role.Permissions()
}

// PermissionSet は付与されている権限のセットを返します
func (pg *PermissionGrant) PermissionSet() *PermissionSet {
	return FromRole(pg.Role)
}
