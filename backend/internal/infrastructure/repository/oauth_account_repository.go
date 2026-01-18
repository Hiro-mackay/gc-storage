package repository

import (
	"context"
	"errors"
	"time"

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

// OAuthAccountRepository はOAuthアカウントリポジトリの実装です
type OAuthAccountRepository struct {
	*database.BaseRepository
}

// NewOAuthAccountRepository は新しいOAuthAccountRepositoryを作成します
func NewOAuthAccountRepository(txManager *database.TxManager) *OAuthAccountRepository {
	return &OAuthAccountRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はOAuthアカウントを作成します
func (r *OAuthAccountRepository) Create(ctx context.Context, account *entity.OAuthAccount) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	var accessToken, refreshToken *string
	if account.AccessToken != "" {
		accessToken = &account.AccessToken
	}
	if account.RefreshToken != "" {
		refreshToken = &account.RefreshToken
	}

	var expiresAt pgtype.Timestamptz
	if !account.TokenExpiresAt.IsZero() {
		expiresAt = pgtype.Timestamptz{Time: account.TokenExpiresAt, Valid: true}
	}

	_, err := queries.CreateOAuthAccount(ctx, sqlcgen.CreateOAuthAccountParams{
		ID:             account.ID,
		UserID:         account.UserID,
		Provider:       string(account.Provider),
		ProviderUserID: account.ProviderUserID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		ExpiresAt:      expiresAt,
		CreatedAt:      account.CreatedAt,
		UpdatedAt:      account.UpdatedAt,
	})

	return r.HandleError(err)
}

// Update はOAuthアカウントを更新します
func (r *OAuthAccountRepository) Update(ctx context.Context, account *entity.OAuthAccount) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	var accessToken, refreshToken *string
	if account.AccessToken != "" {
		accessToken = &account.AccessToken
	}
	if account.RefreshToken != "" {
		refreshToken = &account.RefreshToken
	}

	var expiresAt pgtype.Timestamptz
	if !account.TokenExpiresAt.IsZero() {
		expiresAt = pgtype.Timestamptz{Time: account.TokenExpiresAt, Valid: true}
	}

	err := queries.UpdateOAuthTokens(ctx, sqlcgen.UpdateOAuthTokensParams{
		ID:           account.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでOAuthアカウントを検索します
func (r *OAuthAccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.OAuthAccount, error) {
	// Note: GetOAuthAccountByIDクエリを追加する必要がある
	// 現時点では未実装
	return nil, apperror.NewNotFoundError("oauth account")
}

// FindByUserID はユーザーIDでOAuthアカウント一覧を取得します
func (r *OAuthAccountRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.OAuthAccount, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	rows, err := queries.GetOAuthAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, r.HandleError(err)
	}

	accounts := make([]*entity.OAuthAccount, 0, len(rows))
	for _, row := range rows {
		account := r.toEntity(row)
		accounts = append(accounts, account)
	}

	return accounts, nil
}

// FindByProviderAndUserID はプロバイダーとプロバイダーユーザーIDでOAuthアカウントを検索します
func (r *OAuthAccountRepository) FindByProviderAndUserID(ctx context.Context, provider valueobject.OAuthProvider, providerUserID string) (*entity.OAuthAccount, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetOAuthAccountByProviderAndUserID(ctx, sqlcgen.GetOAuthAccountByProviderAndUserIDParams{
		Provider:       string(provider),
		ProviderUserID: providerUserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("oauth account")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row), nil
}

// Delete はOAuthアカウントを削除します
func (r *OAuthAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteOAuthAccount(ctx, id)
	return r.HandleError(err)
}

// toEntity はsqlcgen.OauthAccountをentity.OAuthAccountに変換します
func (r *OAuthAccountRepository) toEntity(row sqlcgen.OauthAccount) *entity.OAuthAccount {
	var accessToken, refreshToken string
	if row.AccessToken != nil {
		accessToken = *row.AccessToken
	}
	if row.RefreshToken != nil {
		refreshToken = *row.RefreshToken
	}

	var tokenExpiresAt time.Time
	if row.ExpiresAt.Valid {
		tokenExpiresAt = row.ExpiresAt.Time
	}

	return &entity.OAuthAccount{
		ID:             row.ID,
		UserID:         row.UserID,
		Provider:       valueobject.OAuthProvider(row.Provider),
		ProviderUserID: row.ProviderUserID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		TokenExpiresAt: tokenExpiresAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

// インターフェースの実装を保証
var _ repository.OAuthAccountRepository = (*OAuthAccountRepository)(nil)
