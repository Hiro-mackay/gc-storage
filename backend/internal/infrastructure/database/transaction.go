package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// txKey はトランザクションをコンテキストに保持するためのキー
type txKey struct{}

// TxManager はトランザクションを管理する
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager は新しいTxManagerを作成する
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithTransaction はトランザクション内で関数を実行する
// 成功時はコミット、エラー時はロールバック
// 既存のトランザクションがある場合は再利用（ネストトランザクション対応）
func (m *TxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// 既存のトランザクションがあれば再利用
	if tx := m.getTxFromContext(ctx); tx != nil {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// トランザクションをコンテキストに設定
	txCtx := context.WithValue(ctx, txKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v, original error: %w", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getTxFromContext はコンテキストからトランザクションを取得する
func (m *TxManager) getTxFromContext(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}

// GetQuerier はトランザクション中であればTx、そうでなければPoolを返す
func (m *TxManager) GetQuerier(ctx context.Context) Querier {
	if tx := m.getTxFromContext(ctx); tx != nil {
		return tx
	}
	return m.pool
}

// WithTransactionResult は戻り値ありのトランザクションを実行する
func WithTransactionResult[T any](m *TxManager, ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	// 既存のトランザクションがあれば再利用
	if tx := m.getTxFromContext(ctx); tx != nil {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	result, err := fn(txCtx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return zero, fmt.Errorf("rollback failed: %v, original error: %w", rbErr, err)
		}
		return zero, err
	}

	if err := tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}
