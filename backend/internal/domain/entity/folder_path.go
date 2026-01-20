package entity

import (
	"time"

	"github.com/google/uuid"
)

// FolderPath はフォルダ階層の閉包テーブルエントリ
type FolderPath struct {
	AncestorID   uuid.UUID
	DescendantID uuid.UUID
	PathLength   int
	CreatedAt    time.Time
}

// NewFolderPath は新しいFolderPathを作成します
func NewFolderPath(ancestorID, descendantID uuid.UUID, pathLength int) *FolderPath {
	return &FolderPath{
		AncestorID:   ancestorID,
		DescendantID: descendantID,
		PathLength:   pathLength,
		CreatedAt:    time.Now(),
	}
}

// NewSelfReference は自己参照エントリを作成します（path_length = 0）
func NewSelfReference(folderID uuid.UUID) *FolderPath {
	return NewFolderPath(folderID, folderID, 0)
}

// IsSelfReference は自己参照エントリかどうかを判定します
func (fp *FolderPath) IsSelfReference() bool {
	return fp.AncestorID == fp.DescendantID && fp.PathLength == 0
}

// BuildAncestorPaths は祖先パスエントリのリストを生成します
// 親フォルダの祖先パスから子フォルダの祖先パスを生成
func BuildAncestorPaths(folderID uuid.UUID, parentPaths []*FolderPath) []*FolderPath {
	paths := make([]*FolderPath, 0, len(parentPaths)+1)

	// 自己参照を追加
	paths = append(paths, NewSelfReference(folderID))

	// 親の祖先パスから新しいパスを生成
	for _, parentPath := range parentPaths {
		paths = append(paths, NewFolderPath(
			parentPath.AncestorID,
			folderID,
			parentPath.PathLength+1,
		))
	}

	return paths
}
