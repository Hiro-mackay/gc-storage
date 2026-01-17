# GC Storage データベース設計書

## 概要

本ドキュメントでは、GC StorageのPostgreSQLデータベースに関する**設計原則、接続方法、マイグレーション、非機能要件**について説明します。テーブル定義の詳細はスキーマファイルを参照してください。

---

## 1. 技術選定

| コンポーネント | 技術 | 理由 |
|--------------|------|------|
| RDBMS | PostgreSQL 15+ | JSONB対応、全文検索、信頼性 |
| ドライバ | pgx v5 | 高パフォーマンス、PostgreSQL専用最適化 |
| クエリビルダ | sqlc | 型安全なSQL、コード生成 |
| マイグレーション | golang-migrate | シンプル、CI/CD連携容易 |

---

## 2. スキーマ定義ルール

### 2.1 命名規則

| 対象 | 規則 | 例 |
|------|------|-----|
| テーブル名 | 複数形、スネークケース | `users`, `file_versions` |
| カラム名 | スネークケース | `created_at`, `folder_id` |
| 主キー | `id` | `id UUID PRIMARY KEY` |
| 外部キー | `{参照テーブル単数形}_id` | `user_id`, `folder_id` |
| インデックス | `idx_{テーブル}_{カラム}` | `idx_files_folder_id` |
| 制約 | `{テーブル}_{制約種類}_{内容}` | `users_email_unique` |

### 2.2 必須カラム

すべてのテーブルに以下のカラムを含める:

```sql
id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### 2.3 データ型方針

| ユースケース | データ型 | 備考 |
|------------|---------|------|
| 識別子 | UUID | 分散環境での一意性確保 |
| 日時 | TIMESTAMPTZ | タイムゾーン考慮 |
| 文字列（固定長） | VARCHAR(n) | 最大長を明示 |
| 文字列（可変長） | TEXT | 長さ制限なし |
| 金額 | NUMERIC(precision, scale) | 丸め誤差回避 |
| フラグ | BOOLEAN | true/false |
| JSON | JSONB | インデックス可能 |

### 2.4 Extension

```sql
-- UUID生成用
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 全文検索用（部分一致検索）
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
```

---

## 3. sqlcによるクエリ管理

### 3.1 ディレクトリ構成

```
backend/
├── db/
│   ├── sqlc.yaml           # sqlc設定
│   ├── schema/             # スキーマ定義
│   │   ├── 001_users.sql
│   │   ├── 002_files.sql
│   │   └── ...
│   └── query/              # SQLクエリ
│       ├── users.sql
│       ├── files.sql
│       └── ...
└── internal/
    └── infrastructure/
        └── persistence/
            └── postgres/
                └── sqlc/   # 生成コード出力先
```

### 3.2 sqlc.yaml設定

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/query/"
    schema: "db/schema/"
    gen:
      go:
        package: "sqlc"
        out: "internal/infrastructure/persistence/postgres/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
```

### 3.3 クエリ記述ルール

```sql
-- db/query/files.sql

-- name: GetFileByID :one
SELECT * FROM files
WHERE id = $1 AND status != 'deleted';

-- name: ListFilesByFolderID :many
SELECT * FROM files
WHERE folder_id = $1 AND status = 'active'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateFile :one
INSERT INTO files (name, folder_id, owner_id, size, mime_type, storage_key, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateFileStatus :exec
UPDATE files
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: CountFilesByFolderID :one
SELECT COUNT(*) FROM files
WHERE folder_id = $1 AND status = 'active';
```

**アノテーションルール:**

| アノテーション | 用途 | 戻り値 |
|--------------|------|--------|
| `:one` | 単一行取得 | 構造体 or error |
| `:many` | 複数行取得 | スライス or error |
| `:exec` | 更新/削除 | error |
| `:execrows` | 更新/削除（影響行数） | int64, error |

### 3.4 コード生成

```bash
# sqlcコード生成
sqlc generate

# 検証（生成せずチェック）
sqlc verify
```

---

## 4. Goアプリからの接続

### 4.1 接続設定

```go
// internal/infrastructure/database/postgres.go

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
    Host         string
    Port         string
    User         string
    Password     string
    DBName       string
    SSLMode      string
    MaxConns     int32
    MinConns     int32
    MaxConnLife  time.Duration
    MaxConnIdle  time.Duration
}

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
    dsn := fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=%s",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
    )

    poolConfig, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, err
    }

    // コネクションプール設定
    poolConfig.MaxConns = cfg.MaxConns
    poolConfig.MinConns = cfg.MinConns
    poolConfig.MaxConnLifetime = cfg.MaxConnLife
    poolConfig.MaxConnIdleTime = cfg.MaxConnIdle

    return pgxpool.NewWithConfig(ctx, poolConfig)
}
```

### 4.2 コネクションプール設計

| パラメータ | 推奨値 | 説明 |
|-----------|--------|------|
| MaxConns | 25 | 最大接続数（CPU数 × 2 + ディスク数） |
| MinConns | 5 | 最小接続数（アイドル時も維持） |
| MaxConnLifetime | 1h | 接続の最大生存時間 |
| MaxConnIdleTime | 30m | アイドル接続の最大維持時間 |

**注意事項:**
- コンテナ環境では`MaxConns`を控えめに設定（スケールアウト考慮）
- PostgreSQLの`max_connections`との整合性を確認
- 計算式: `MaxConns × レプリカ数 < max_connections`

### 4.3 ヘルスチェック

```go
func (p *Pool) HealthCheck(ctx context.Context) error {
    return p.pool.Ping(ctx)
}
```

---

## 5. マイグレーション

### 5.1 golang-migrate設定

```bash
# インストール
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 5.2 マイグレーションファイル

```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_files_table.up.sql
├── 000002_create_files_table.down.sql
├── 000003_add_files_index.up.sql
├── 000003_add_files_index.down.sql
└── ...
```

**命名規則:** `{番号}_{説明}.{up|down}.sql`

### 5.3 マイグレーションコマンド

```bash
# 新規マイグレーション作成
migrate create -ext sql -dir migrations -seq create_users_table

# マイグレーション適用（全て）
migrate -path migrations -database "${DATABASE_URL}" up

# マイグレーション適用（指定数）
migrate -path migrations -database "${DATABASE_URL}" up 2

# ロールバック（1つ）
migrate -path migrations -database "${DATABASE_URL}" down 1

# 特定バージョンへ移動
migrate -path migrations -database "${DATABASE_URL}" goto 5

# 現在のバージョン確認
migrate -path migrations -database "${DATABASE_URL}" version

# 強制バージョン設定（dirty状態解消）
migrate -path migrations -database "${DATABASE_URL}" force 3
```

### 5.4 マイグレーション作成ルール

**up.sqlのルール:**
- 1ファイル1操作を原則とする
- `IF NOT EXISTS`を使用して冪等性を確保
- トランザクション内で実行される

**down.sqlのルール:**
- up.sqlの逆操作を記述
- データ削除を伴う場合は慎重に検討
- 完全なロールバックが難しい場合はコメントで明記

**例:**

```sql
-- 000001_create_users_table.up.sql
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
```

```sql
-- 000001_create_users_table.down.sql
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
```

### 5.5 CI/CD連携

```yaml
# GitHub Actions例
- name: Run migrations
  run: |
    migrate -path migrations -database "${{ secrets.DATABASE_URL }}" up
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
```

**デプロイフロー:**
1. PRマージ前にマイグレーションのdry-run実行
2. デプロイ時にマイグレーション自動適用
3. 失敗時はロールバックして通知

---

## 6. 非機能要件

### 6.1 パフォーマンス目標

| 指標 | 目標値 | 備考 |
|------|--------|------|
| クエリ応答時間（P95） | < 100ms | 単純なSELECT |
| クエリ応答時間（P99） | < 500ms | JOIN含む複雑なクエリ |
| 同時接続数 | 1000+ | コネクションプーラー使用時 |
| 書き込みスループット | 1000 ops/sec | ファイルメタデータ更新 |

### 6.2 インデックス戦略

| インデックス種類 | 用途 | 使用場面 |
|----------------|------|---------|
| B-tree | 等価・範囲検索 | 主キー、外部キー、日時 |
| GIN (pg_trgm) | 部分一致検索 | ファイル名検索 |
| GIN (JSONB) | JSON内検索 | メタデータ検索 |
| Partial Index | 条件付き高速化 | `status = 'active'`のみ |

### 6.3 パーティショニング

大量データが想定されるテーブルにはパーティショニングを適用:

| テーブル | パーティション方式 | キー |
|---------|------------------|------|
| audit_logs | RANGE | created_at（月次） |
| file_versions | RANGE | created_at（月次） |

### 6.4 レプリケーション戦略

| 構成 | 用途 |
|------|------|
| Streaming Replication | 高可用性（フェイルオーバー用） |
| Read Replica | 読み取り負荷分散 |

**読み書き分離:**
- 書き込み: プライマリへ
- 読み取り: レプリカへ（許容遅延内）

### 6.5 バックアップ方針

| 方式 | 頻度 | 保持期間 |
|------|------|---------|
| WAL Archiving | 継続的 | 7日 |
| フルバックアップ | 日次 | 30日 |
| Point-in-Time Recovery | 随時 | 7日以内 |

### 6.6 監視項目

| 項目 | 閾値 | アクション |
|------|------|----------|
| コネクション使用率 | > 80% | プール拡張検討 |
| クエリ実行時間 | > 1s | スロークエリ調査 |
| ディスク使用率 | > 80% | ストレージ拡張 |
| レプリケーション遅延 | > 10s | ネットワーク/負荷調査 |

---

## 関連ドキュメント

- [バックエンド設計](./BACKEND.md)
- [API設計](./API.md)
- [フロントエンド設計](./FRONTEND.md)
