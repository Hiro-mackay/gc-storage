package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// フォルダ関連の定数
const (
	MaxFolderDepth = 20
)

// フォルダ関連エラー
var (
	ErrFolderMaxDepthExceeded = errors.New("folder max depth exceeded")
	ErrFolderCircularMove     = errors.New("cannot move folder to itself or its descendants")
	ErrFolderNameConflict     = errors.New("folder name already exists in parent")
)

// Folder はフォルダエンティティ（集約ルート）
// Note: フォルダにはゴミ箱がない。削除時、配下のファイルはArchivedFileへ移動し、
// フォルダ自体は直接削除される。
type Folder struct {
	ID        uuid.UUID
	Name      valueobject.FolderName
	ParentID  *uuid.UUID
	OwnerID   uuid.UUID
	OwnerType valueobject.OwnerType
	Depth     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewFolder は新しいフォルダを作成します
func NewFolder(
	name valueobject.FolderName,
	parentID *uuid.UUID,
	ownerID uuid.UUID,
	ownerType valueobject.OwnerType,
	depth int,
) (*Folder, error) {
	if depth > MaxFolderDepth {
		return nil, ErrFolderMaxDepthExceeded
	}

	now := time.Now()
	return &Folder{
		ID:        uuid.New(),
		Name:      name,
		ParentID:  parentID,
		OwnerID:   ownerID,
		OwnerType: ownerType,
		Depth:     depth,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ReconstructFolder はDBからフォルダを復元します
func ReconstructFolder(
	id uuid.UUID,
	name valueobject.FolderName,
	parentID *uuid.UUID,
	ownerID uuid.UUID,
	ownerType valueobject.OwnerType,
	depth int,
	createdAt time.Time,
	updatedAt time.Time,
) *Folder {
	return &Folder{
		ID:        id,
		Name:      name,
		ParentID:  parentID,
		OwnerID:   ownerID,
		OwnerType: ownerType,
		Depth:     depth,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// CanMoveTo は指定した親フォルダへ移動可能かどうかを判定します
func (f *Folder) CanMoveTo(newParentID *uuid.UUID, newDepth int, descendantIDs []uuid.UUID) error {
	// 自身への移動は不可
	if newParentID != nil && *newParentID == f.ID {
		return ErrFolderCircularMove
	}

	// 子孫フォルダへの移動は不可（循環参照防止）
	if newParentID != nil {
		for _, descendantID := range descendantIDs {
			if *newParentID == descendantID {
				return ErrFolderCircularMove
			}
		}
	}

	// 深さ制限チェック
	if newDepth > MaxFolderDepth {
		return ErrFolderMaxDepthExceeded
	}

	return nil
}

// MoveTo はフォルダを移動します
func (f *Folder) MoveTo(newParentID *uuid.UUID, newDepth int) {
	f.ParentID = newParentID
	f.Depth = newDepth
	f.UpdatedAt = time.Now()
}

// Rename はフォルダ名を変更します
func (f *Folder) Rename(newName valueobject.FolderName) {
	f.Name = newName
	f.UpdatedAt = time.Now()
}

// UpdateDepth は深さを更新します
func (f *Folder) UpdateDepth(newDepth int) {
	f.Depth = newDepth
	f.UpdatedAt = time.Now()
}

// IsRoot はルートフォルダかどうかを判定します
func (f *Folder) IsRoot() bool {
	return f.ParentID == nil
}

// IsOwnedBy は指定ユーザー/グループが所有者かどうかを判定します
func (f *Folder) IsOwnedBy(ownerID uuid.UUID, ownerType valueobject.OwnerType) bool {
	return f.OwnerID == ownerID && f.OwnerType == ownerType
}
