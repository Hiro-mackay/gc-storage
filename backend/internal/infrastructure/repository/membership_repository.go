package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// MembershipRepository はメンバーシップリポジトリの実装です
type MembershipRepository struct {
	*database.BaseRepository
}

// NewMembershipRepository は新しいMembershipRepositoryを作成します
func NewMembershipRepository(txManager *database.TxManager) *MembershipRepository {
	return &MembershipRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はメンバーシップを作成します
func (r *MembershipRepository) Create(ctx context.Context, membership *entity.Membership) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateMembership(ctx, sqlcgen.CreateMembershipParams{
		ID:       membership.ID,
		GroupID:  membership.GroupID,
		UserID:   membership.UserID,
		Role:     membership.Role.String(),
		JoinedAt: membership.JoinedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでメンバーシップを検索します
func (r *MembershipRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Membership, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetMembershipByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("membership")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// Update はメンバーシップを更新します
func (r *MembershipRepository) Update(ctx context.Context, membership *entity.Membership) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	role := membership.Role.String()
	_, err := queries.UpdateMembership(ctx, sqlcgen.UpdateMembershipParams{
		ID:   membership.ID,
		Role: &role,
	})

	return r.HandleError(err)
}

// Delete はメンバーシップを削除します
func (r *MembershipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteMembership(ctx, id)
	return r.HandleError(err)
}

// FindByGroupID はグループIDでメンバーシップを検索します
func (r *MembershipRepository) FindByGroupID(ctx context.Context, groupID uuid.UUID) ([]*entity.Membership, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListMembershipsByGroupID(ctx, groupID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindByGroupIDWithUsers はグループIDでメンバーシップとユーザー情報を検索します
func (r *MembershipRepository) FindByGroupIDWithUsers(ctx context.Context, groupID uuid.UUID) ([]*entity.MembershipWithUser, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListMembershipsByGroupIDWithUsers(ctx, groupID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	membershipsWithUsers := make([]*entity.MembershipWithUser, 0, len(rows))
	for _, row := range rows {
		role, err := valueobject.NewGroupRole(row.Role)
		if err != nil {
			return nil, err
		}

		email, err := valueobject.NewEmail(row.Email)
		if err != nil {
			return nil, err
		}

		membership := entity.ReconstructMembership(
			row.ID,
			row.GroupID,
			row.UserID,
			role,
			row.JoinedAt,
		)

		user := &entity.User{
			ID:     row.UserID,
			Email:  email,
			Name:   row.DisplayName,
			Status: entity.UserStatus(row.UserStatus),
		}

		membershipsWithUsers = append(membershipsWithUsers, &entity.MembershipWithUser{
			Membership: membership,
			User:       user,
		})
	}

	return membershipsWithUsers, nil
}

// FindByUserID はユーザーIDでメンバーシップを検索します
func (r *MembershipRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Membership, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListMembershipsByUserID(ctx, userID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindByGroupAndUser はグループIDとユーザーIDでメンバーシップを検索します
func (r *MembershipRepository) FindByGroupAndUser(ctx context.Context, groupID, userID uuid.UUID) (*entity.Membership, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetMembershipByGroupAndUser(ctx, sqlcgen.GetMembershipByGroupAndUserParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("membership")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// Exists はメンバーシップが存在するかを確認します
func (r *MembershipRepository) Exists(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	exists, err := queries.MembershipExists(ctx, sqlcgen.MembershipExistsParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return false, r.HandleError(err)
	}

	return exists, nil
}

// CountByGroupID はグループIDでメンバー数をカウントします
func (r *MembershipRepository) CountByGroupID(ctx context.Context, groupID uuid.UUID) (int, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	count, err := queries.CountMembershipsByGroupID(ctx, groupID)
	if err != nil {
		return 0, r.HandleError(err)
	}

	return int(count), nil
}

// DeleteByGroupID はグループIDでメンバーシップを一括削除します
func (r *MembershipRepository) DeleteByGroupID(ctx context.Context, groupID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteMembershipsByGroupID(ctx, groupID)
	return r.HandleError(err)
}

// toEntity はsqlcgen.Membershipをentity.Membershipに変換します
func (r *MembershipRepository) toEntity(row sqlcgen.Membership) (*entity.Membership, error) {
	role, err := valueobject.NewGroupRole(row.Role)
	if err != nil {
		return nil, err
	}

	return entity.ReconstructMembership(
		row.ID,
		row.GroupID,
		row.UserID,
		role,
		row.JoinedAt,
	), nil
}

// toEntities は複数のsqlcgen.Membershipをentity.Membershipに変換します
func (r *MembershipRepository) toEntities(rows []sqlcgen.Membership) ([]*entity.Membership, error) {
	memberships := make([]*entity.Membership, 0, len(rows))
	for _, row := range rows {
		membership, err := r.toEntity(row)
		if err != nil {
			return nil, err
		}
		memberships = append(memberships, membership)
	}
	return memberships, nil
}

// インターフェースの実装を保証
var _ repository.MembershipRepository = (*MembershipRepository)(nil)
