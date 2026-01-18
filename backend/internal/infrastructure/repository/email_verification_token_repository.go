package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
	"github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database/sqlcgen"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// EmailVerificationTokenRepository はメール確認トークンリポジトリの実装です
type EmailVerificationTokenRepository struct {
	*database.BaseRepository
}

// NewEmailVerificationTokenRepository は新しいEmailVerificationTokenRepositoryを作成します
func NewEmailVerificationTokenRepository(txManager *database.TxManager) *EmailVerificationTokenRepository {
	return &EmailVerificationTokenRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はトークンを作成します
func (r *EmailVerificationTokenRepository) Create(ctx context.Context, token *entity.EmailVerificationToken) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	_, err := queries.CreateEmailVerificationToken(ctx, sqlcgen.CreateEmailVerificationTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	})

	return r.HandleError(err)
}

// FindByToken はトークン文字列でトークンを検索します
func (r *EmailVerificationTokenRepository) FindByToken(ctx context.Context, token string) (*entity.EmailVerificationToken, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetEmailVerificationTokenByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("email verification token")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// FindByUserID はユーザーIDでトークンを検索します
func (r *EmailVerificationTokenRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.EmailVerificationToken, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetEmailVerificationTokenByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("email verification token")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// Delete はトークンを削除します
func (r *EmailVerificationTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteEmailVerificationToken(ctx, id)
	return r.HandleError(err)
}

// DeleteByUserID はユーザーの全トークンを削除します
func (r *EmailVerificationTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteEmailVerificationTokensByUserID(ctx, userID)
	return r.HandleError(err)
}

// DeleteExpired は期限切れトークンを削除します
func (r *EmailVerificationTokenRepository) DeleteExpired(ctx context.Context) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteExpiredEmailVerificationTokens(ctx)
	return r.HandleError(err)
}

// toEntity はsqlcgen.EmailVerificationTokenをentity.EmailVerificationTokenに変換します
func (r *EmailVerificationTokenRepository) toEntity(row sqlcgen.EmailVerificationToken) *entity.EmailVerificationToken {
	return &entity.EmailVerificationToken{
		ID:        row.ID,
		UserID:    row.UserID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}
}

// インターフェースの実装を保証
var _ repository.EmailVerificationTokenRepository = (*EmailVerificationTokenRepository)(nil)
