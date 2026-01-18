package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// データベースエラー
var (
	ErrNotFound = errors.New("record not found")
	ErrConflict = errors.New("record already exists")
)

// BaseRepository はリポジトリの基底構造体
type BaseRepository struct {
	txManager *TxManager
}

// NewBaseRepository は新しいBaseRepositoryを作成する
func NewBaseRepository(txManager *TxManager) *BaseRepository {
	return &BaseRepository{txManager: txManager}
}

// Querier はクエリ実行用のインターフェースを返す
// トランザクション中であればTx、そうでなければPoolを返す
func (r *BaseRepository) Querier(ctx context.Context) Querier {
	return r.txManager.GetQuerier(ctx)
}

// TxManager はトランザクションマネージャーを返す
func (r *BaseRepository) TxManager() *TxManager {
	return r.txManager
}

// HandleError はpgxのエラーを適切なドメインエラーに変換する
func (r *BaseRepository) HandleError(err error) error {
	if err == nil {
		return nil
	}

	// レコードが見つからない場合
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	// PostgreSQLエラーコードの処理
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return ErrConflict
		case "23503": // foreign_key_violation
			return errors.New("foreign key violation: " + pgErr.Detail)
		case "23514": // check_violation
			return errors.New("check constraint violation: " + pgErr.Detail)
		}
	}

	return err
}

// IsNotFoundError はエラーがNotFoundエラーかどうかを判定する
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsConflictError はエラーがConflictエラーかどうかを判定する
func IsConflictError(err error) bool {
	return errors.Is(err, ErrConflict)
}
