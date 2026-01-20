package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
)

// FolderClosureRepository はフォルダ閉包テーブルリポジトリの実装です
type FolderClosureRepository struct {
	*database.BaseRepository
}

// NewFolderClosureRepository は新しいFolderClosureRepositoryを作成します
func NewFolderClosureRepository(txManager *database.TxManager) *FolderClosureRepository {
	return &FolderClosureRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// InsertSelfReference は自己参照パスエントリを挿入します
func (r *FolderClosureRepository) InsertSelfReference(ctx context.Context, folderID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.InsertFolderPath(ctx, sqlcgen.InsertFolderPathParams{
		AncestorID:   folderID,
		DescendantID: folderID,
		PathLength:   0,
	})

	return r.HandleError(err)
}

// InsertAncestorPaths は祖先パスエントリを一括挿入します
func (r *FolderClosureRepository) InsertAncestorPaths(ctx context.Context, paths []*entity.FolderPath) error {
	if len(paths) == 0 {
		return nil
	}

	querier := r.Querier(ctx)

	// CopyFrom を使用してバルクインサート
	params := make([]sqlcgen.InsertFolderPathsBulkParams, len(paths))
	for i, path := range paths {
		params[i] = sqlcgen.InsertFolderPathsBulkParams{
			AncestorID:   path.AncestorID,
			DescendantID: path.DescendantID,
			PathLength:   int32(path.PathLength),
			CreatedAt:    path.CreatedAt,
		}
	}

	queries := sqlcgen.New(querier)
	_, err := queries.InsertFolderPathsBulk(ctx, params)
	return r.HandleError(err)
}

// DeleteByDescendant は子孫IDで全パスエントリを削除します
func (r *FolderClosureRepository) DeleteByDescendant(ctx context.Context, descendantID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteFolderPathsByDescendant(ctx, descendantID)
	return r.HandleError(err)
}

// FindAncestorIDs は祖先フォルダIDを取得します（自己参照を除く）
func (r *FolderClosureRepository) FindAncestorIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	ids, err := queries.GetAncestorIDs(ctx, folderID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return ids, nil
}

// FindDescendantIDs は子孫フォルダIDを取得します（自己参照を除く）
func (r *FolderClosureRepository) FindDescendantIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	ids, err := queries.GetDescendantIDs(ctx, folderID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return ids, nil
}

// FindAncestorPaths は祖先パスエントリを取得します
func (r *FolderClosureRepository) FindAncestorPaths(ctx context.Context, folderID uuid.UUID) ([]*entity.FolderPath, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.GetAncestorPaths(ctx, folderID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindDescendantsWithDepth は子孫フォルダとその相対深さを取得します
func (r *FolderClosureRepository) FindDescendantsWithDepth(ctx context.Context, folderID uuid.UUID) (map[uuid.UUID]int, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.GetDescendantsWithDepth(ctx, folderID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	result := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		result[row.DescendantID] = int(row.PathLength)
	}

	return result, nil
}

// DeleteSubtreePaths はサブツリー全体のパスエントリを削除します
func (r *FolderClosureRepository) DeleteSubtreePaths(ctx context.Context, folderID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteSubtreePaths(ctx, folderID)
	return r.HandleError(err)
}

// MoveSubtree はサブツリーを新しい親に移動します
func (r *FolderClosureRepository) MoveSubtree(ctx context.Context, folderID uuid.UUID, newParentPaths []*entity.FolderPath) error {
	querier := r.Querier(ctx)

	// 1. サブツリー内の全子孫を取得（自己参照を含む）
	queries := sqlcgen.New(querier)
	descendantIDs, err := queries.GetSelfAndDescendantIDs(ctx, folderID)
	if err != nil {
		return r.HandleError(err)
	}

	// 2. サブツリー内の各ノードについて、古いパスを削除して新しいパスを挿入
	for _, descendantID := range descendantIDs {
		// 古いパスを削除
		if err := queries.DeleteFolderPathsByDescendant(ctx, descendantID); err != nil {
			return r.HandleError(err)
		}

		// 自己参照を挿入
		if err := queries.InsertFolderPath(ctx, sqlcgen.InsertFolderPathParams{
			AncestorID:   descendantID,
			DescendantID: descendantID,
			PathLength:   0,
		}); err != nil {
			return r.HandleError(err)
		}
	}

	// 3. サブツリー内の階層関係を再構築
	// まず元のフォルダ(folderID)に新しい親からのパスを設定
	for _, parentPath := range newParentPaths {
		if err := queries.InsertFolderPath(ctx, sqlcgen.InsertFolderPathParams{
			AncestorID:   parentPath.AncestorID,
			DescendantID: folderID,
			PathLength:   int32(parentPath.PathLength + 1),
		}); err != nil {
			return r.HandleError(err)
		}
	}

	// 4. サブツリー内の他のノードにパスを再設定（再帰的に）
	// 新しい親パスを取得して子孫に伝播
	folderPaths, err := queries.GetAncestorPaths(ctx, folderID)
	if err != nil {
		return r.HandleError(err)
	}

	for _, descendantID := range descendantIDs {
		if descendantID == folderID {
			continue // 元フォルダはすでに処理済み
		}

		// 子孫の相対深さを計算（サブツリー内での位置）
		relativeDepth := 0
		for _, d := range descendantIDs {
			if d == descendantID {
				break
			}
			relativeDepth++
		}

		// 祖先パスを追加
		for _, ancestorPath := range folderPaths {
			if err := queries.InsertFolderPath(ctx, sqlcgen.InsertFolderPathParams{
				AncestorID:   ancestorPath.AncestorID,
				DescendantID: descendantID,
				PathLength:   ancestorPath.PathLength + int32(relativeDepth),
			}); err != nil {
				return r.HandleError(err)
			}
		}
	}

	return nil
}

// toEntity はsqlcgen.FolderPathをentity.FolderPathに変換します
func (r *FolderClosureRepository) toEntity(row sqlcgen.FolderPath) *entity.FolderPath {
	return &entity.FolderPath{
		AncestorID:   row.AncestorID,
		DescendantID: row.DescendantID,
		PathLength:   int(row.PathLength),
		CreatedAt:    row.CreatedAt,
	}
}

// toEntities はsqlcgen.FolderPath配列をentity.FolderPath配列に変換します
func (r *FolderClosureRepository) toEntities(rows []sqlcgen.FolderPath) []*entity.FolderPath {
	entities := make([]*entity.FolderPath, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// インターフェースの実装を保証
var _ repository.FolderClosureRepository = (*FolderClosureRepository)(nil)
