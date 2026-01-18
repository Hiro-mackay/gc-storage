package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// PasswordResetTokenRepository はパスワードリセットトークンリポジトリの実装です
type PasswordResetTokenRepository struct {
	*database.BaseRepository
}

// NewPasswordResetTokenRepository は新しいPasswordResetTokenRepositoryを作成します
func NewPasswordResetTokenRepository(txManager *database.TxManager) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はトークンを作成します
func (r *PasswordResetTokenRepository) Create(ctx context.Context, token *entity.PasswordResetToken) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreatePasswordResetToken(ctx, sqlcgen.CreatePasswordResetTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	})

	return r.HandleError(err)
}

// FindByToken はトークン文字列でトークンを検索します
func (r *PasswordResetTokenRepository) FindByToken(ctx context.Context, token string) (*entity.PasswordResetToken, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetPasswordResetTokenByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("password reset token")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// MarkAsUsed はトークンを使用済みにします
func (r *PasswordResetTokenRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.MarkPasswordResetTokenAsUsed(ctx, id)
	return r.HandleError(err)
}

// Delete はトークンを削除します
func (r *PasswordResetTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeletePasswordResetToken(ctx, id)
	return r.HandleError(err)
}

// DeleteByUserID はユーザーの全トークンを削除します
func (r *PasswordResetTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeletePasswordResetTokensByUserID(ctx, userID)
	return r.HandleError(err)
}

// DeleteExpired は期限切れトークンを削除します
func (r *PasswordResetTokenRepository) DeleteExpired(ctx context.Context) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteExpiredPasswordResetTokens(ctx)
	return r.HandleError(err)
}

// toEntity はsqlcgen.PasswordResetTokenをentity.PasswordResetTokenに変換します
func (r *PasswordResetTokenRepository) toEntity(row sqlcgen.PasswordResetToken) *entity.PasswordResetToken {
	var usedAt *time.Time
	if row.UsedAt.Valid {
		usedAt = &row.UsedAt.Time
	}

	return &entity.PasswordResetToken{
		ID:        row.ID,
		UserID:    row.UserID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		UsedAt:    usedAt,
		CreatedAt: row.CreatedAt,
	}
}

// インターフェースの実装を保証
var _ repository.PasswordResetTokenRepository = (*PasswordResetTokenRepository)(nil)
