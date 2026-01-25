package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// Group はグループエンティティ（集約ルート）
// Note: Groupは論理削除をサポートしません。削除は物理削除のみです。
type Group struct {
	ID          uuid.UUID
	Name        valueobject.GroupName
	Description string
	OwnerID     uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewGroup は新しいグループを作成します
func NewGroup(
	name valueobject.GroupName,
	description string,
	ownerID uuid.UUID,
) *Group {
	now := time.Now()
	return &Group{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ReconstructGroup はDBからグループを復元します
func ReconstructGroup(
	id uuid.UUID,
	name valueobject.GroupName,
	description string,
	ownerID uuid.UUID,
	createdAt time.Time,
	updatedAt time.Time,
) *Group {
	return &Group{
		ID:          id,
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// IsOwnedBy は指定ユーザーがオーナーかを判定します
func (g *Group) IsOwnedBy(userID uuid.UUID) bool {
	return g.OwnerID == userID
}

// Rename はグループ名を変更します
func (g *Group) Rename(newName valueobject.GroupName) {
	g.Name = newName
	g.UpdatedAt = time.Now()
}

// UpdateDescription は説明を更新します
func (g *Group) UpdateDescription(description string) {
	g.Description = description
	g.UpdatedAt = time.Now()
}

// TransferOwnership はオーナーを変更します
func (g *Group) TransferOwnership(newOwnerID uuid.UUID) {
	g.OwnerID = newOwnerID
	g.UpdatedAt = time.Now()
}
