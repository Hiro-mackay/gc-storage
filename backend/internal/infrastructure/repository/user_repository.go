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

// UserRepository はユーザーリポジトリの実装です
type UserRepository struct {
	*database.BaseRepository
}

// NewUserRepository は新しいUserRepositoryを作成します
func NewUserRepository(txManager *database.TxManager) *UserRepository {
	return &UserRepository{
		BaseRepository: database.NewBaseRepository(txManager),
	}
}

// Create はユーザーを作成します
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	var emailVerifiedAt pgtype.Timestamptz
	if user.EmailVerified {
		emailVerifiedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}

	var passwordHash *string
	if user.PasswordHash != "" {
		passwordHash = &user.PasswordHash
	}

	_, err := queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		ID:              user.ID,
		Email:           user.Email.String(),
		PasswordHash:    passwordHash,
		DisplayName:     user.Name,
		Status:          string(user.Status),
		EmailVerifiedAt: emailVerifiedAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	})

	return r.HandleError(err)
}

// Update はユーザーを更新します
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	var emailVerifiedAt pgtype.Timestamptz
	if user.EmailVerified {
		emailVerifiedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}

	status := string(user.Status)
	_, err := queries.UpdateUser(ctx, sqlcgen.UpdateUserParams{
		ID:              user.ID,
		DisplayName:     &user.Name,
		PasswordHash:    &user.PasswordHash,
		Status:          &status,
		EmailVerifiedAt: emailVerifiedAt,
	})

	return r.HandleError(err)
}

// FindByID はIDでユーザーを検索します
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("user")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// FindByEmail はメールアドレスでユーザーを検索します
func (r *UserRepository) FindByEmail(ctx context.Context, email valueobject.Email) (*entity.User, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	row, err := queries.GetUserByEmail(ctx, email.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFoundError("user")
		}
		return nil, r.HandleError(err)
	}

	return r.toEntity(row)
}

// Exists はメールアドレスが存在するかを確認します
func (r *UserRepository) Exists(ctx context.Context, email valueobject.Email) (bool, error) {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	exists, err := queries.UserExistsByEmail(ctx, email.String())
	if err != nil {
		return false, r.HandleError(err)
	}

	return exists, nil
}

// Delete はユーザーを削除します（論理削除）
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	querier := r.Querier(ctx)
	queries := sqlcgen.New(querier)

	err := queries.DeleteUser(ctx, id)
	return r.HandleError(err)
}

// toEntity はsqlcgen.Userをentity.Userに変換します
func (r *UserRepository) toEntity(row sqlcgen.User) (*entity.User, error) {
	email, err := valueobject.NewEmail(row.Email)
	if err != nil {
		return nil, err
	}

	var passwordHash string
	if row.PasswordHash != nil {
		passwordHash = *row.PasswordHash
	}

	return &entity.User{
		ID:            row.ID,
		Email:         email,
		Name:          row.DisplayName,
		PasswordHash:  passwordHash,
		Status:        entity.UserStatus(row.Status),
		EmailVerified: row.EmailVerifiedAt.Valid,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

// インターフェースの実装を保証
var _ repository.UserRepository = (*UserRepository)(nil)
