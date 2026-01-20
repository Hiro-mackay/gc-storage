package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// FolderHierarchyService はフォルダ階層に関するドメインサービス
type FolderHierarchyService interface {
	// CalculateNewDepth は移動後の深さを計算します
	CalculateNewDepth(newParent *entity.Folder) int

	// ValidateMove はフォルダ移動の妥当性を検証します
	ValidateMove(ctx context.Context, folder *entity.Folder, newParentID *uuid.UUID, descendantIDs []uuid.UUID) error

	// ValidateDepthAfterMove は移動後の深さ制約を検証します
	ValidateDepthAfterMove(folder *entity.Folder, newDepth int, maxDescendantDepth int) error

	// BuildAncestorPaths はクロージャテーブル用のパスエントリを構築します
	BuildAncestorPaths(folderID uuid.UUID, parentPaths []*entity.FolderPath) []*entity.FolderPath
}

// folderHierarchyServiceImpl はFolderHierarchyServiceの実装
type folderHierarchyServiceImpl struct{}

// NewFolderHierarchyService は新しいFolderHierarchyServiceを作成します
func NewFolderHierarchyService() FolderHierarchyService {
	return &folderHierarchyServiceImpl{}
}

// CalculateNewDepth は移動後の深さを計算します
func (s *folderHierarchyServiceImpl) CalculateNewDepth(newParent *entity.Folder) int {
	if newParent == nil {
		return 0
	}
	return newParent.Depth + 1
}

// ValidateMove はフォルダ移動の妥当性を検証します
func (s *folderHierarchyServiceImpl) ValidateMove(
	ctx context.Context,
	folder *entity.Folder,
	newParentID *uuid.UUID,
	descendantIDs []uuid.UUID,
) error {
	// 自身への移動は不可
	if newParentID != nil && *newParentID == folder.ID {
		return entity.ErrFolderCircularMove
	}

	// 子孫フォルダへの移動は不可（循環参照防止）
	if newParentID != nil {
		for _, descendantID := range descendantIDs {
			if *newParentID == descendantID {
				return entity.ErrFolderCircularMove
			}
		}
	}

	return nil
}

// ValidateDepthAfterMove は移動後の深さ制約を検証します
func (s *folderHierarchyServiceImpl) ValidateDepthAfterMove(
	folder *entity.Folder,
	newDepth int,
	maxDescendantDepth int,
) error {
	// 移動後の最深部が制限を超えないかチェック
	if newDepth+maxDescendantDepth > entity.MaxFolderDepth {
		return entity.ErrFolderMaxDepthExceeded
	}
	return nil
}

// BuildAncestorPaths はクロージャテーブル用のパスエントリを構築します
func (s *folderHierarchyServiceImpl) BuildAncestorPaths(
	folderID uuid.UUID,
	parentPaths []*entity.FolderPath,
) []*entity.FolderPath {
	return entity.BuildAncestorPaths(folderID, parentPaths)
}
