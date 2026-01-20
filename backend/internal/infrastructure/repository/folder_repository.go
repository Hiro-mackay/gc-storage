package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// FolderRepository はフォルダリポジトリの実装です
type FolderRepository struct {
	*database.BaseRepository
}

// NewFolderRepository は新しいFolderRepositoryを作成します
func NewFolderRepository(txManager *database.TxManager) *FolderRepository {
	return &FolderRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はフォルダを作成します
func (r *FolderRepository) Create(ctx context.Context, folder *entity.Folder) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateFolder(ctx, sqlcgen.CreateFolderParams{
		ID:        folder.ID,
		Name:      folder.Name.String(),
		ParentID:  uuidToPgtype(folder.ParentID),
		OwnerID:   folder.OwnerID,
		OwnerType: sqlcgen.OwnerType(folder.OwnerType),
		Depth:     int32(folder.Depth),
		CreatedAt: folder.CreatedAt,
		UpdatedAt: folder.UpdatedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでフォルダを検索します
func (r *FolderRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Folder, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetFolderByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("folder")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// Update はフォルダを更新します
func (r *FolderRepository) Update(ctx context.Context, folder *entity.Folder) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	name := folder.Name.String()
	depth := int32(folder.Depth)
	_, err := queries.UpdateFolder(ctx, sqlcgen.UpdateFolderParams{
		ID:       folder.ID,
		Name:     &name,
		ParentID: uuidToPgtype(folder.ParentID),
		Depth:    &depth,
	})

	return r.HandleError(err)
}

// Delete はフォルダを削除します
func (r *FolderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteFolder(ctx, id)
	return r.HandleError(err)
}

// FindByParentID は親IDでフォルダを検索します
func (r *FolderRepository) FindByParentID(ctx context.Context, parentID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.Folder, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFoldersByParentID(ctx, sqlcgen.ListFoldersByParentIDParams{
		ParentID:  uuidToPgtype(parentID),
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindRootByOwner はオーナーのルートフォルダを検索します
func (r *FolderRepository) FindRootByOwner(ctx context.Context, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.Folder, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListRootFoldersByOwner(ctx, sqlcgen.ListRootFoldersByOwnerParams{
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// FindByOwner はオーナーの全フォルダを検索します
func (r *FolderRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.Folder, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListFoldersByOwner(ctx, sqlcgen.ListFoldersByOwnerParams{
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows), nil
}

// ExistsByNameAndParent は同名フォルダの存在チェックをします
func (r *FolderRepository) ExistsByNameAndParent(ctx context.Context, name valueobject.FolderName, parentID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	exists, err := queries.FolderExistsByNameAndParent(ctx, sqlcgen.FolderExistsByNameAndParentParams{
		ParentID:  uuidToPgtype(parentID),
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
		Name:      name.String(),
	})

	return exists, r.HandleError(err)
}

// ExistsByNameAndOwnerRoot はルートレベルでの同名フォルダの存在チェックをします
func (r *FolderRepository) ExistsByNameAndOwnerRoot(ctx context.Context, name valueobject.FolderName, ownerID uuid.UUID, ownerType valueobject.OwnerType) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	exists, err := queries.FolderExistsByNameAtRoot(ctx, sqlcgen.FolderExistsByNameAtRootParams{
		OwnerID:   ownerID,
		OwnerType: sqlcgen.OwnerType(ownerType),
		Name:      name.String(),
	})

	return exists, r.HandleError(err)
}

// ExistsByID はIDでフォルダの存在チェックをします
func (r *FolderRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	exists, err := queries.FolderExistsByID(ctx, id)
	return exists, r.HandleError(err)
}

// UpdateDepth は深さを更新します
func (r *FolderRepository) UpdateDepth(ctx context.Context, id uuid.UUID, depth int) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.UpdateFolderDepth(ctx, sqlcgen.UpdateFolderDepthParams{
		ID:    id,
		Depth: int32(depth),
	})

	return r.HandleError(err)
}

// BulkDelete はフォルダを一括削除します
func (r *FolderRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteFoldersBulk(ctx, ids)
	return r.HandleError(err)
}

// BulkUpdateDepth はフォルダの深さを一括更新します
func (r *FolderRepository) BulkUpdateDepth(ctx context.Context, folderDepths map[uuid.UUID]int) error {
	if len(folderDepths) == 0 {
		return nil
	}

	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	ids := make([]uuid.UUID, 0, len(folderDepths))
	depths := make([]int32, 0, len(folderDepths))
	for id, depth := range folderDepths {
		ids = append(ids, id)
		depths = append(depths, int32(depth))
	}

	err := queries.BulkUpdateFolderDepth(ctx, sqlcgen.BulkUpdateFolderDepthParams{
		Column1: ids,
		Column2: depths,
	})

	return r.HandleError(err)
}

// toEntity はsqlcgen.Folderをentity.Folderに変換します
func (r *FolderRepository) toEntity(row sqlcgen.Folder) *entity.Folder {
	name, _ := valueobject.NewFolderName(row.Name)
	ownerType := valueobject.OwnerType(row.OwnerType)

	return entity.ReconstructFolder(
		row.ID,
		name,
		pgtypeToUUID(row.ParentID),
		row.OwnerID,
		ownerType,
		int(row.Depth),
		row.CreatedAt,
		row.UpdatedAt,
	)
}

// toEntities はsqlcgen.Folder配列をentity.Folder配列に変換します
func (r *FolderRepository) toEntities(rows []sqlcgen.Folder) []*entity.Folder {
	entities := make([]*entity.Folder, len(rows))
	for i, row := range rows {
		entities[i] = r.toEntity(row)
	}
	return entities
}

// uuidToPgtype はuuid.UUIDをpgtype.UUIDに変換します
func uuidToPgtype(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// pgtypeToUUID はpgtype.UUIDを*uuid.UUIDに変換します
func pgtypeToUUID(pg pgtype.UUID) *uuid.UUID {
	if !pg.Valid {
		return nil
	}
	id := uuid.UUID(pg.Bytes)
	return &id
}

// インターフェースの実装を保証
var _ repository.FolderRepository = (*FolderRepository)(nil)
