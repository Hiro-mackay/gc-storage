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

// FolderStatus はフォルダの状態を定義します
type FolderStatus string

const (
	FolderStatusActive FolderStatus = "active"
)

// IsValid はステータスが有効かを判定します
func (s FolderStatus) IsValid() bool {
	return s == FolderStatusActive
}

// フォルダ関連エラー
var (
	ErrFolderMaxDepthExceeded = errors.New("folder max depth exceeded")
	ErrFolderCircularMove     = errors.New("cannot move folder to itself or its descendants")
	ErrFolderNameConflict     = errors.New("folder name already exists in parent")
	ErrFolderNotActive        = errors.New("folder is not active")
)

// Folder はフォルダエンティティ（集約ルート）
// Note: フォルダにはゴミ箱がない。削除時、配下のファイルはArchivedFileへ移動し、
// フォルダ自体は直接削除される。
// Note: owner_typeは削除。フォルダは常にユーザーが所有者。グループはPermissionGrantでアクセス。
type Folder struct {
	ID        uuid.UUID
	Name      valueobject.FolderName
	ParentID  *uuid.UUID
	OwnerID   uuid.UUID // 現在の所有者ID（所有権譲渡で変更可能）
	CreatedBy uuid.UUID // 最初の作成者ID（不変、履歴追跡用）
	Depth     int
	Status    FolderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewFolder は新しいフォルダを作成します
// 新規作成時は owner_id = created_by = 作成者（createdBy引数）となります
func NewFolder(
	name valueobject.FolderName,
	parentID *uuid.UUID,
	createdBy uuid.UUID,
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
		OwnerID:   createdBy, // 新規作成時は owner_id = created_by
		CreatedBy: createdBy,
		Depth:     depth,
		Status:    FolderStatusActive,
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
	createdBy uuid.UUID,
	depth int,
	status FolderStatus,
	createdAt time.Time,
	updatedAt time.Time,
) *Folder {
	return &Folder{
		ID:        id,
		Name:      name,
		ParentID:  parentID,
		OwnerID:   ownerID,
		CreatedBy: createdBy,
		Depth:     depth,
		Status:    status,
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

func (f *Folder) EqualsID(id uuid.UUID) bool {
	return f.ID == id
}

// 名前が一致するかどうか判定します
func (f *Folder) EqualsName(name valueobject.FolderName) bool {
	return f.Name.Equals(name)
}

// IsOwnedBy は指定ユーザーが所有者かどうかを判定します
// Note: フォルダは常にユーザーが所有者（グループはPermissionGrantでアクセス）
func (f *Folder) IsOwnedBy(ownerID uuid.UUID) bool {
	return f.OwnerID == ownerID
}

// IsCreatedBy は指定ユーザーが作成者かどうかを判定します
func (f *Folder) IsCreatedBy(userID uuid.UUID) bool {
	return f.CreatedBy == userID
}

// IsActive はフォルダがアクティブかどうかを判定します
func (f *Folder) IsActive() bool {
	return f.Status == FolderStatusActive
}

// TransferOwnership は所有権を譲渡します
// Note: created_by は変更されません（不変）
func (f *Folder) TransferOwnership(newOwnerID uuid.UUID) {
	f.OwnerID = newOwnerID
	f.UpdatedAt = time.Now()
}
