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

// GroupRepository はグループリポジトリの実装です
type GroupRepository struct {
	*database.BaseRepository
}

// NewGroupRepository は新しいGroupRepositoryを作成します
func NewGroupRepository(txManager *database.TxManager) *GroupRepository {
	return &GroupRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はグループを作成します
func (r *GroupRepository) Create(ctx context.Context, group *entity.Group) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateGroup(ctx, sqlcgen.CreateGroupParams{
		ID:          group.ID,
		Name:        group.Name.String(),
		Description: stringToPtr(group.Description),
		OwnerID:     group.OwnerID,
		Status:      "active", // Groupは論理削除をサポートしないため常にactive
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでグループを検索します
func (r *GroupRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Group, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("group")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// Update はグループを更新します
func (r *GroupRepository) Update(ctx context.Context, group *entity.Group) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	name := group.Name.String()
	_, err := queries.UpdateGroup(ctx, sqlcgen.UpdateGroupParams{
		ID:          group.ID,
		Name:        &name,
		Description: &group.Description,
		OwnerID:     pgtype.UUID{Bytes: group.OwnerID, Valid: true},
		Status:      nil, // statusは更新しない（常にactive）
	})

	return r.HandleError(err)
}

// Delete はグループを物理削除します
func (r *GroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteGroup(ctx, id)
	return r.HandleError(err)
}

// FindByOwnerID はオーナーIDでグループを検索します
func (r *GroupRepository) FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*entity.Group, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListGroupsByOwnerID(ctx, ownerID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindActiveByOwnerID はオーナーIDでアクティブなグループを検索します
func (r *GroupRepository) FindActiveByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*entity.Group, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListActiveGroupsByOwnerID(ctx, ownerID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindByMemberID はメンバーIDでグループを検索します
func (r *GroupRepository) FindByMemberID(ctx context.Context, userID uuid.UUID) ([]*entity.Group, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListGroupsByMemberID(ctx, userID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// ExistsByID はIDでグループが存在するかを確認します
func (r *GroupRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	exists, err := queries.GroupExistsByID(ctx, id)
	if err != nil {
		return false, r.HandleError(err)
	}

	return exists, nil
}

// toEntity はsqlcgen.Groupをentity.Groupに変換します
func (r *GroupRepository) toEntity(row sqlcgen.Group) (*entity.Group, error) {
	name, err := valueobject.NewGroupName(row.Name)
	if err != nil {
		return nil, err
	}

	description := ""
	if row.Description != nil {
		description = *row.Description
	}

	return entity.ReconstructGroup(
		row.ID,
		name,
		description,
		row.OwnerID,
		row.CreatedAt,
		row.UpdatedAt,
	), nil
}

// toEntities は複数のsqlcgen.Groupをentity.Groupに変換します
func (r *GroupRepository) toEntities(rows []sqlcgen.Group) ([]*entity.Group, error) {
	groups := make([]*entity.Group, 0, len(rows))
	for _, row := range rows {
		group, err := r.toEntity(row)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

// stringToPtr は文字列をポインタに変換します
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// インターフェースの実装を保証
var _ repository.GroupRepository = (*GroupRepository)(nil)
