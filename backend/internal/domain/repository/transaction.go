package repository

import (
	"context"
)

// TransactionManager はトランザクション管理インターフェースを定義します
type TransactionManager interface {
	// WithTransaction はトランザクション内で処理を実行します
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
