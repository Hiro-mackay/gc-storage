package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// FolderRepository はフォルダリポジトリのインターフェース
// Note: owner_typeは削除されたため、フォルダは常にユーザー所有として扱う
type FolderRepository interface {
	// 基本CRUD
	Create(ctx context.Context, folder *entity.Folder) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Folder, error)
	Update(ctx context.Context, folder *entity.Folder) error
	Delete(ctx context.Context, id uuid.UUID) error

	// 検索
	FindByParentID(ctx context.Context, parentID *uuid.UUID, ownerID uuid.UUID) ([]*entity.Folder, error)
	FindRootByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.Folder, error)
	FindByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entity.Folder, error)
	FindByCreatedBy(ctx context.Context, createdBy uuid.UUID) ([]*entity.Folder, error)

	// 存在チェック
	ExistsByNameAndParent(ctx context.Context, name valueobject.FolderName, parentID *uuid.UUID, ownerID uuid.UUID) (bool, error)
	ExistsByNameAndOwnerRoot(ctx context.Context, name valueobject.FolderName, ownerID uuid.UUID) (bool, error)
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)

	// 深さ更新
	UpdateDepth(ctx context.Context, id uuid.UUID, depth int) error

	// 一括操作
	BulkDelete(ctx context.Context, ids []uuid.UUID) error
	BulkUpdateDepth(ctx context.Context, folderDepths map[uuid.UUID]int) error
}

// FolderClosureRepository はフォルダ閉包テーブルリポジトリのインターフェース
// Note: DBテーブル名は folder_paths だが、概念的にはクロージャテーブル
type FolderClosureRepository interface {
	// パスエントリ操作
	InsertSelfReference(ctx context.Context, folderID uuid.UUID) error
	InsertAncestorPaths(ctx context.Context, paths []*entity.FolderPath) error
	DeleteByDescendant(ctx context.Context, descendantID uuid.UUID) error

	// 階層クエリ
	FindAncestorIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error)
	FindDescendantIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error)
	FindAncestorPaths(ctx context.Context, folderID uuid.UUID) ([]*entity.FolderPath, error)
	FindDescendantsWithDepth(ctx context.Context, folderID uuid.UUID) (map[uuid.UUID]int, error)

	// 移動操作
	DeleteSubtreePaths(ctx context.Context, folderID uuid.UUID) error
	MoveSubtree(ctx context.Context, folderID uuid.UUID, newParentPaths []*entity.FolderPath) error
}

// FolderWithHierarchy は階層操作を含むフォルダリポジトリ
// ドメインサービスから使用される複合インターフェース
type FolderWithHierarchy interface {
	// CreateWithHierarchy はフォルダと閉包テーブルエントリを一括作成します
	CreateWithHierarchy(ctx context.Context, folder *entity.Folder, paths []*entity.FolderPath) error

	// MoveWithHierarchy はフォルダを移動し、閉包テーブルを更新します
	MoveWithHierarchy(ctx context.Context, folder *entity.Folder, descendants []*entity.Folder) error

	// DeleteWithSubtree はフォルダとサブツリーを削除します
	// 戻り値: 削除されたフォルダID一覧
	DeleteWithSubtree(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error)

	// GetAncestors は祖先フォルダを取得します（パンくずリスト用）
	GetAncestors(ctx context.Context, folderID uuid.UUID) ([]*entity.Folder, error)
}
