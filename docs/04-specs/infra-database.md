# インフラ基盤: データベース (PostgreSQL)

## メタ情報

| 項目 | 値 |
|------|-----|
| ステータス | Draft |
| 優先度 | High |
| フェーズ | Phase 0 |
| 依存 | なし（最優先で実装） |

---

## 概要

PostgreSQLデータベースへの接続、トランザクション管理、リポジトリ基盤の実装仕様。
全ドメインの永続化層の基盤となる。

---

## 1. データベース接続

### 1.1 接続設定

**環境変数:**
```bash
DATABASE_URL=postgres://user:password@host:5432/gc_storage?sslmode=disable
```

**接続プール設定:**
```go
type DBConfig struct {
    MaxConns          int32         // 最大接続数: 25
    MinConns          int32         // 最小接続数: 5
    MaxConnLifetime   time.Duration // 接続の最大生存時間: 1時間
    MaxConnIdleTime   time.Duration // アイドル接続の最大時間: 30分
    HealthCheckPeriod time.Duration // ヘルスチェック間隔: 1分
}
```

### 1.2 接続クライアント

**ディレクトリ:** `backend/internal/infrastructure/database/`

```go
// backend/internal/infrastructure/database/postgres.go

package database

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type PostgresClient struct {
    pool *pgxpool.Pool
}

func NewPostgresClient(ctx context.Context, databaseURL string) (*PostgresClient, error) {
    config, err := pgxpool.ParseConfig(databaseURL)
    if err != nil {
        return nil, fmt.Errorf("failed to parse database URL: %w", err)
    }

    // プール設定
    config.MaxConns = 25
    config.MinConns = 5
    config.MaxConnLifetime = 1 * time.Hour
    config.MaxConnIdleTime = 30 * time.Minute
    config.HealthCheckPeriod = 1 * time.Minute

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("failed to create connection pool: %w", err)
    }

    // 接続確認
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &PostgresClient{pool: pool}, nil
}

func (c *PostgresClient) Pool() *pgxpool.Pool {
    return c.pool
}

func (c *PostgresClient) Close() {
    c.pool.Close()
}

func (c *PostgresClient) Health(ctx context.Context) error {
    return c.pool.Ping(ctx)
}
```

---

## 2. トランザクション管理

### 2.1 トランザクションマネージャーインターフェース

**ディレクトリ:** `backend/internal/domain/repository/`

```go
// backend/internal/domain/repository/transaction.go

package repository

import "context"

// TxManager はトランザクション管理のインターフェース
type TxManager interface {
    // WithTransaction はトランザクション内で関数を実行
    // 成功時はコミット、エラー時はロールバック
    WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

    // WithTransactionResult は戻り値ありのトランザクション
    WithTransactionResult[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error)
}
```

### 2.2 トランザクションマネージャー実装

```go
// backend/internal/infrastructure/database/transaction.go

package database

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// トランザクションをコンテキストに保持するためのキー
type txKey struct{}

type TxManager struct {
    pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
    return &TxManager{pool: pool}
}

func (m *TxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
    // 既存のトランザクションがあれば再利用（ネストトランザクション対応）
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

// ジェネリクス版（Go 1.21+）
func WithTransactionResult[T any](m *TxManager, ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
    var zero T

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
```

### 2.3 Querierインターフェース

```go
// backend/internal/infrastructure/database/querier.go

package database

import (
    "context"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
)

// Querier はpgxpool.PoolとTxの共通インターフェース
type Querier interface {
    Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
```

---

## 3. sqlc設定

### 3.1 設定ファイル

```yaml
# backend/sqlc.yaml

version: "2"
sql:
  - engine: "postgresql"
    queries: "./internal/infrastructure/database/queries/"
    schema: "../migrations/"
    gen:
      go:
        package: "sqlcgen"
        out: "./internal/infrastructure/database/sqlcgen"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
          - db_type: "jsonb"
            go_type: "json.RawMessage"
```

### 3.2 クエリファイル構成

```
backend/internal/infrastructure/database/queries/
├── users.sql
├── sessions.sql
├── oauth_accounts.sql
├── groups.sql
├── memberships.sql
├── invitations.sql
├── folders.sql
├── files.sql
├── file_versions.sql
├── permissions.sql
├── relationships.sql
└── share_links.sql
```

### 3.3 クエリ記述例

```sql
-- backend/internal/infrastructure/database/queries/users.sql

-- name: CreateUser :one
INSERT INTO users (
    id, email, name, password_hash, status, email_verified, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users SET
    name = COALESCE(sqlc.narg('name'), name),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    status = COALESCE(sqlc.narg('status'), status),
    email_verified = COALESCE(sqlc.narg('email_verified'), email_verified),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UserExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);
```

---

## 4. リポジトリ基盤

### 4.1 ベースリポジトリ

```go
// backend/internal/infrastructure/database/base_repository.go

package database

import (
    "context"
    "errors"

    "github.com/jackc/pgx/v5"
)

var (
    ErrNotFound = errors.New("record not found")
    ErrConflict = errors.New("record already exists")
)

type BaseRepository struct {
    txManager *TxManager
}

func NewBaseRepository(txManager *TxManager) *BaseRepository {
    return &BaseRepository{txManager: txManager}
}

func (r *BaseRepository) Querier(ctx context.Context) Querier {
    return r.txManager.GetQuerier(ctx)
}

// HandleError はpgxのエラーを適切なドメインエラーに変換
func (r *BaseRepository) HandleError(err error) error {
    if err == nil {
        return nil
    }

    if errors.Is(err, pgx.ErrNoRows) {
        return ErrNotFound
    }

    // PostgreSQLエラーコードの処理
    // 23505: unique_violation
    // 23503: foreign_key_violation
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505":
            return ErrConflict
        }
    }

    return err
}
```

### 4.2 リポジトリ実装例

```go
// backend/internal/infrastructure/repository/user_repository.go

package repository

import (
    "context"
    "time"

    "github.com/google/uuid"
    "gc-storage/internal/domain/entity"
    domainRepo "gc-storage/internal/domain/repository"
    "gc-storage/internal/infrastructure/database"
    "gc-storage/internal/infrastructure/database/sqlcgen"
)

type userRepository struct {
    *database.BaseRepository
    queries *sqlcgen.Queries
}

func NewUserRepository(base *database.BaseRepository, queries *sqlcgen.Queries) domainRepo.UserRepository {
    return &userRepository{
        BaseRepository: base,
        queries:        queries,
    }
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
    _, err := r.queries.CreateUser(ctx, r.Querier(ctx), sqlcgen.CreateUserParams{
        ID:            user.ID,
        Email:         user.Email.String(),
        Name:          user.Name,
        PasswordHash:  user.PasswordHash,
        Status:        string(user.Status),
        EmailVerified: user.EmailVerified,
        CreatedAt:     user.CreatedAt,
        UpdatedAt:     user.UpdatedAt,
    })
    return r.HandleError(err)
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
    row, err := r.queries.GetUserByID(ctx, r.Querier(ctx), id)
    if err != nil {
        return nil, r.HandleError(err)
    }
    return r.toEntity(row), nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email entity.Email) (*entity.User, error) {
    row, err := r.queries.GetUserByEmail(ctx, r.Querier(ctx), email.String())
    if err != nil {
        return nil, r.HandleError(err)
    }
    return r.toEntity(row), nil
}

func (r *userRepository) toEntity(row sqlcgen.User) *entity.User {
    email, _ := entity.NewEmail(row.Email)
    return &entity.User{
        ID:            row.ID,
        Email:         email,
        Name:          row.Name,
        PasswordHash:  row.PasswordHash,
        Status:        entity.UserStatus(row.Status),
        EmailVerified: row.EmailVerified,
        CreatedAt:     row.CreatedAt,
        UpdatedAt:     row.UpdatedAt,
    }
}
```

---

## 5. マイグレーション管理

### 5.1 マイグレーションツール

**使用ツール:** golang-migrate

**コマンド:**
```bash
# マイグレーション適用
task migrate:up

# ロールバック
task migrate:down

# 新規マイグレーション作成
task migrate:create NAME=create_example_table
```

### 5.2 マイグレーションファイル規約

**命名:** `{sequence}_{description}.up.sql` / `{sequence}_{description}.down.sql`

**例:**
```
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_groups.up.sql
├── 000002_create_groups.down.sql
...
```

### 5.3 共通カラム規約

```sql
-- 全テーブル共通
id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

-- updated_atトリガー
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_{table}_updated_at
    BEFORE UPDATE ON {table}
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

---

## 6. エラーハンドリング

### 6.1 データベースエラーの分類

```go
// backend/pkg/apperror/database.go

package apperror

type DatabaseError struct {
    Code    string
    Message string
    Cause   error
}

func (e *DatabaseError) Error() string {
    return e.Message
}

func (e *DatabaseError) Unwrap() error {
    return e.Cause
}

// エラーコード
const (
    DBErrNotFound     = "DB_NOT_FOUND"
    DBErrConflict     = "DB_CONFLICT"
    DBErrConnection   = "DB_CONNECTION"
    DBErrTransaction  = "DB_TRANSACTION"
    DBErrConstraint   = "DB_CONSTRAINT"
)

func NewNotFoundError(resource string) *DatabaseError {
    return &DatabaseError{
        Code:    DBErrNotFound,
        Message: resource + " not found",
    }
}

func NewConflictError(message string) *DatabaseError {
    return &DatabaseError{
        Code:    DBErrConflict,
        Message: message,
    }
}
```

---

## 7. DI設定

### 7.1 Wire設定

```go
// backend/internal/infrastructure/wire.go

//go:build wireinject

package infrastructure

import (
    "context"

    "github.com/google/wire"
    "gc-storage/internal/infrastructure/database"
)

var DatabaseSet = wire.NewSet(
    database.NewPostgresClient,
    database.NewTxManager,
    database.NewBaseRepository,
    ProvideQueries,
)

func ProvideQueries(client *database.PostgresClient) *sqlcgen.Queries {
    return sqlcgen.New(client.Pool())
}
```

---

## 8. テスト

### 8.1 テスト用ヘルパー

```go
// backend/internal/infrastructure/database/testhelper/db.go

package testhelper

import (
    "context"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
)

func SetupTestDB(t *testing.T) *pgxpool.Pool {
    t.Helper()

    ctx := context.Background()
    pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/gc_storage_test?sslmode=disable")
    if err != nil {
        t.Fatalf("failed to connect to test database: %v", err)
    }

    t.Cleanup(func() {
        pool.Close()
    })

    return pool
}

func TruncateTables(t *testing.T, pool *pgxpool.Pool, tables ...string) {
    t.Helper()

    ctx := context.Background()
    for _, table := range tables {
        _, err := pool.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE")
        if err != nil {
            t.Fatalf("failed to truncate table %s: %v", table, err)
        }
    }
}
```

### 8.2 リポジトリテスト例

```go
// backend/internal/infrastructure/repository/user_repository_test.go

func TestUserRepository_Create(t *testing.T) {
    pool := testhelper.SetupTestDB(t)
    testhelper.TruncateTables(t, pool, "users")

    txManager := database.NewTxManager(pool)
    base := database.NewBaseRepository(txManager)
    queries := sqlcgen.New(pool)
    repo := NewUserRepository(base, queries)

    ctx := context.Background()

    t.Run("正常系_ユーザー作成成功", func(t *testing.T) {
        email, _ := entity.NewEmail("test@example.com")
        user := &entity.User{
            ID:            uuid.New(),
            Email:         email,
            Name:          "Test User",
            PasswordHash:  ptrString("hashed"),
            Status:        entity.UserStatusActive,
            EmailVerified: false,
            CreatedAt:     time.Now(),
            UpdatedAt:     time.Now(),
        }

        err := repo.Create(ctx, user)
        assert.NoError(t, err)

        found, err := repo.FindByID(ctx, user.ID)
        assert.NoError(t, err)
        assert.Equal(t, user.Email, found.Email)
    })

    t.Run("異常系_メール重複", func(t *testing.T) {
        // 重複テスト...
    })
}
```

---

## 9. ディレクトリ構成

```
backend/internal/infrastructure/database/
├── postgres.go           # PostgresClient
├── transaction.go        # TxManager
├── querier.go           # Querierインターフェース
├── base_repository.go   # BaseRepository
├── queries/             # SQLクエリファイル（sqlc用）
│   ├── users.sql
│   ├── sessions.sql
│   └── ...
├── sqlcgen/             # sqlc生成コード（自動生成）
│   ├── db.go
│   ├── models.go
│   └── queries.sql.go
└── testhelper/          # テスト用ヘルパー
    └── db.go
```

---

## 10. 受け入れ基準

### 機能要件

- [ ] PostgreSQLへの接続・切断が正常に動作する
- [ ] コネクションプールが設定通りに動作する
- [ ] トランザクションのコミット・ロールバックが正常に動作する
- [ ] ネストされたトランザクションが正しく処理される
- [ ] sqlcによるコード生成が正常に動作する
- [ ] マイグレーションの適用・ロールバックが動作する

### 非機能要件

- [ ] ヘルスチェックが1秒以内に応答する
- [ ] 接続エラー時に適切なエラーメッセージが返される
- [ ] トランザクション失敗時にロールバックされる
- [ ] パニック時にもロールバックされる

---

## 関連ドキュメント

- [DATABASE.md](../../02-architecture/DATABASE.md) - スキーマ設計
- [BACKEND.md](../../02-architecture/BACKEND.md) - Clean Architecture
- [CODING_STANDARDS.md](../../01-policies/CODING_STANDARDS.md) - 命名規則
