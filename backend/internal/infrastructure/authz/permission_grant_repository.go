package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// PermissionGrantRepository は権限付与リポジトリの実装です
type PermissionGrantRepository struct {
	*database.BaseRepository
}

// NewPermissionGrantRepository は新しいPermissionGrantRepositoryを作成します
func NewPermissionGrantRepository(txManager *database.TxManager) *PermissionGrantRepository {
	return &PermissionGrantRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create は権限付与を作成します
func (r *PermissionGrantRepository) Create(ctx context.Context, grant *authz.PermissionGrant) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreatePermissionGrant(ctx, sqlcgen.CreatePermissionGrantParams{
		ID:           grant.ID,
		ResourceType: grant.ResourceType.String(),
		ResourceID:   grant.ResourceID,
		GranteeType:  grant.GranteeType.String(),
		GranteeID:    grant.GranteeID,
		Role:         grant.Role.String(),
		GrantedBy:    grant.GrantedBy,
		GrantedAt:    grant.GrantedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDで権限付与を検索します
func (r *PermissionGrantRepository) FindByID(ctx context.Context, id uuid.UUID) (*authz.PermissionGrant, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetPermissionGrantByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("permission_grant")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// Delete は権限付与を削除します
func (r *PermissionGrantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeletePermissionGrant(ctx, id)
	return r.HandleError(err)
}

// FindByResource はリソースで権限付与を検索します
func (r *PermissionGrantRepository) FindByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*authz.PermissionGrant, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListPermissionGrantsByResource(ctx, sqlcgen.ListPermissionGrantsByResourceParams{
		ResourceType: resourceType.String(),
		ResourceID:   resourceID,
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindByResourceAndGrantee はリソースと付与対象で権限付与を検索します
func (r *PermissionGrantRepository) FindByResourceAndGrantee(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID, granteeType authz.GranteeType, granteeID uuid.UUID) ([]*authz.PermissionGrant, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListPermissionGrantsByResourceAndGrantee(ctx, sqlcgen.ListPermissionGrantsByResourceAndGranteeParams{
		ResourceType: resourceType.String(),
		ResourceID:   resourceID,
		GranteeType:  granteeType.String(),
		GranteeID:    granteeID,
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindByResourceGranteeAndRole はリソース、付与対象、ロールで権限付与を検索します
func (r *PermissionGrantRepository) FindByResourceGranteeAndRole(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID, granteeType authz.GranteeType, granteeID uuid.UUID, role authz.Role) (*authz.PermissionGrant, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetPermissionGrantByResourceGranteeAndRole(ctx, sqlcgen.GetPermissionGrantByResourceGranteeAndRoleParams{
		ResourceType: resourceType.String(),
		ResourceID:   resourceID,
		GranteeType:  granteeType.String(),
		GranteeID:    granteeID,
		Role:         role.String(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("permission_grant")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// FindByGrantee は付与対象で権限付与を検索します
func (r *PermissionGrantRepository) FindByGrantee(ctx context.Context, granteeType authz.GranteeType, granteeID uuid.UUID) ([]*authz.PermissionGrant, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListPermissionGrantsByGrantee(ctx, sqlcgen.ListPermissionGrantsByGranteeParams{
		GranteeType: granteeType.String(),
		GranteeID:   granteeID,
	})
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// DeleteByResource はリソースで権限付与を一括削除します
func (r *PermissionGrantRepository) DeleteByResource(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeletePermissionGrantsByResource(ctx, sqlcgen.DeletePermissionGrantsByResourceParams{
		ResourceType: resourceType.String(),
		ResourceID:   resourceID,
	})
	return r.HandleError(err)
}

// DeleteByGrantee は付与対象で権限付与を一括削除します
func (r *PermissionGrantRepository) DeleteByGrantee(ctx context.Context, granteeType authz.GranteeType, granteeID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeletePermissionGrantsByGrantee(ctx, sqlcgen.DeletePermissionGrantsByGranteeParams{
		GranteeType: granteeType.String(),
		GranteeID:   granteeID,
	})
	return r.HandleError(err)
}

// DeleteByResourceAndGrantee はリソースと付与対象で権限付与を削除します
func (r *PermissionGrantRepository) DeleteByResourceAndGrantee(ctx context.Context, resourceType authz.ResourceType, resourceID uuid.UUID, granteeType authz.GranteeType, granteeID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeletePermissionGrantsByResourceAndGrantee(ctx, sqlcgen.DeletePermissionGrantsByResourceAndGranteeParams{
		ResourceType: resourceType.String(),
		ResourceID:   resourceID,
		GranteeType:  granteeType.String(),
		GranteeID:    granteeID,
	})
	return r.HandleError(err)
}

// toEntity はsqlcgen.PermissionGrantをauthz.PermissionGrantに変換します
func (r *PermissionGrantRepository) toEntity(row sqlcgen.PermissionGrant) (*authz.PermissionGrant, error) {
	resourceType, err := authz.NewResourceType(row.ResourceType)
	if err != nil {
		return nil, err
	}

	granteeType, err := authz.NewGranteeType(row.GranteeType)
	if err != nil {
		return nil, err
	}

	role, err := authz.NewRole(row.Role)
	if err != nil {
		return nil, err
	}

	return authz.ReconstructPermissionGrant(
		row.ID,
		resourceType,
		row.ResourceID,
		granteeType,
		row.GranteeID,
		role,
		row.GrantedBy,
		row.GrantedAt,
	), nil
}

// toEntities は複数のsqlcgen.PermissionGrantをauthz.PermissionGrantに変換します
func (r *PermissionGrantRepository) toEntities(rows []sqlcgen.PermissionGrant) ([]*authz.PermissionGrant, error) {
	grants := make([]*authz.PermissionGrant, 0, len(rows))
	for _, row := range rows {
		grant, err := r.toEntity(row)
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	return grants, nil
}

// インターフェースの実装を保証
var _ authz.PermissionGrantRepository = (*PermissionGrantRepository)(nil)
