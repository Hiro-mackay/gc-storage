package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier はpgxpool.PoolとTxの共通インターフェース
// sqlcが生成するコードで使用するクエリ実行機能を提供
type Querier interface {
	// Exec はSQLを実行し、結果のコマンドタグを返す
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)

	// Query はSQLを実行し、結果の行セットを返す
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)

	// QueryRow はSQLを実行し、単一行を返す
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
