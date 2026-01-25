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

// InvitationRepository は招待リポジトリの実装です
type InvitationRepository struct {
	*database.BaseRepository
}

// NewInvitationRepository は新しいInvitationRepositoryを作成します
func NewInvitationRepository(txManager *database.TxManager) *InvitationRepository {
	return &InvitationRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create は招待を作成します
func (r *InvitationRepository) Create(ctx context.Context, invitation *entity.Invitation) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateInvitation(ctx, sqlcgen.CreateInvitationParams{
		ID:        invitation.ID,
		GroupID:   invitation.GroupID,
		Email:     invitation.Email.String(),
		Token:     invitation.Token,
		Role:      invitation.Role.String(),
		InvitedBy: invitation.InvitedBy,
		ExpiresAt: invitation.ExpiresAt,
		Status:    invitation.Status.String(),
		CreatedAt: invitation.CreatedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDで招待を検索します
func (r *InvitationRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Invitation, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetInvitationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("invitation")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// Update は招待を更新します
func (r *InvitationRepository) Update(ctx context.Context, invitation *entity.Invitation) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	status := invitation.Status.String()
	_, err := queries.UpdateInvitation(ctx, sqlcgen.UpdateInvitationParams{
		ID:     invitation.ID,
		Status: &status,
	})

	return r.HandleError(err)
}

// Delete は招待を削除します
func (r *InvitationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteInvitation(ctx, id)
	return r.HandleError(err)
}

// FindByToken はトークンで招待を検索します
func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*entity.Invitation, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetInvitationByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("invitation")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// FindPendingByGroupID はグループIDで保留中の招待を検索します
func (r *InvitationRepository) FindPendingByGroupID(ctx context.Context, groupID uuid.UUID) ([]*entity.Invitation, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListPendingInvitationsByGroupID(ctx, groupID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindPendingByEmail はメールアドレスで保留中の招待を検索します
func (r *InvitationRepository) FindPendingByEmail(ctx context.Context, email valueobject.Email) ([]*entity.Invitation, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.ListPendingInvitationsByEmail(ctx, email.String())
	if err != nil {
		return nil, r.HandleError(err)
	}

	return r.toEntities(rows)
}

// FindPendingByGroupAndEmail はグループIDとメールアドレスで保留中の招待を検索します
func (r *InvitationRepository) FindPendingByGroupAndEmail(ctx context.Context, groupID uuid.UUID, email valueobject.Email) (*entity.Invitation, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetPendingInvitationByGroupAndEmail(ctx, sqlcgen.GetPendingInvitationByGroupAndEmailParams{
		GroupID: groupID,
		Email:   email.String(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("invitation")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// DeleteByGroupID はグループIDで招待を一括削除します
func (r *InvitationRepository) DeleteByGroupID(ctx context.Context, groupID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteInvitationsByGroupID(ctx, groupID)
	return r.HandleError(err)
}

// ExpireOld は古い招待を期限切れにします
func (r *InvitationRepository) ExpireOld(ctx context.Context) (int64, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	count, err := queries.ExpireOldInvitations(ctx)
	if err != nil {
		return 0, r.HandleError(err)
	}

	return count, nil
}

// toEntity はsqlcgen.Invitationをentity.Invitationに変換します
func (r *InvitationRepository) toEntity(row sqlcgen.Invitation) (*entity.Invitation, error) {
	email, err := valueobject.NewEmail(row.Email)
	if err != nil {
		return nil, err
	}

	role, err := valueobject.NewGroupRole(row.Role)
	if err != nil {
		return nil, err
	}

	status, err := valueobject.NewInvitationStatus(row.Status)
	if err != nil {
		return nil, err
	}

	return entity.ReconstructInvitation(
		row.ID,
		row.GroupID,
		email,
		row.Token,
		role,
		row.InvitedBy,
		row.ExpiresAt,
		status,
		row.CreatedAt,
	), nil
}

// toEntities は複数のsqlcgen.Invitationをentity.Invitationに変換します
func (r *InvitationRepository) toEntities(rows []sqlcgen.Invitation) ([]*entity.Invitation, error) {
	invitations := make([]*entity.Invitation, 0, len(rows))
	for _, row := range rows {
		invitation, err := r.toEntity(row)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, invitation)
	}
	return invitations, nil
}

// インターフェースの実装を保証
var _ repository.InvitationRepository = (*InvitationRepository)(nil)
