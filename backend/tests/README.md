# Backend Tests

このディレクトリには、GC Storage バックエンドのテストコードが含まれています。

## ディレクトリ構成

```
tests/
├── README.md              # このファイル
├── integration/           # インテグレーションテスト
│   └── auth_test.go       # 認証APIテスト
└── testutil/              # テストユーティリティ
    ├── setup.go           # DB/Redis接続セットアップ
    ├── server.go          # テストサーバー構築
    └── http.go            # HTTPリクエスト/レスポンスヘルパー
```

---

## クイックスタート

### 前提条件

1. Docker がインストールされていること
2. Task がインストールされていること (`brew install go-task`)
3. Go 1.22+ がインストールされていること

### インテグレーションテスト実行

```bash
# プロジェクトルートで実行

# 方法1: インフラ起動 → テスト → インフラ停止（推奨）
task test:integration

# 方法2: インフラが既に起動している場合
task backend:test-integration

# 方法3: 全テスト実行（ユニット + インテグレーション）
task test
```

### ユニットテストのみ実行

```bash
task backend:test
```

---

## テストの種類

### 1. ユニットテスト (`internal/` 内)

- 個々の関数やメソッドをテスト
- モックを使用して外部依存を排除
- `go test ./internal/...` で実行

### 2. インテグレーションテスト (`tests/integration/`)

- 実際のデータベース・Redisを使用
- APIエンドポイントの E2E テスト
- `INTEGRATION_TEST=true` 環境変数で有効化
- `go test ./tests/integration/...` で実行

---

## インテグレーションテストの書き方

### 基本構造

```go
package integration

import (
    "net/http"
    "os"
    "testing"

    "github.com/stretchr/testify/suite"
    "github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// XxxTestSuite はテストスイートの定義
type XxxTestSuite struct {
    suite.Suite
    server *testutil.TestServer
}

// SetupSuite は全テスト前に1回だけ実行
func (s *XxxTestSuite) SetupSuite() {
    s.server = testutil.NewTestServer(s.T())
}

// TearDownSuite は全テスト後に1回だけ実行
func (s *XxxTestSuite) TearDownSuite() {
    testutil.CleanupTestEnvironment()
}

// SetupTest は各テスト前に実行（DBクリーンアップ）
func (s *XxxTestSuite) SetupTest() {
    s.server.Cleanup(s.T())
}

// TestXxxSuite はテストスイートのエントリーポイント
func TestXxxSuite(t *testing.T) {
    // インテグレーションテストのスキップ条件
    if os.Getenv("INTEGRATION_TEST") != "true" {
        t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
    }
    suite.Run(t, new(XxxTestSuite))
}
```

### HTTPリクエストの送信

```go
func (s *XxxTestSuite) TestEndpoint_Success() {
    // リクエスト送信
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/endpoint",
        Body: map[string]string{
            "field1": "value1",
            "field2": "value2",
        },
    })

    // アサーション（チェーン可能）
    resp.AssertStatus(http.StatusOK).
        AssertJSONPathExists("data.id").
        AssertJSONPath("data.field1", "value1")
}
```

### 認証が必要なエンドポイントのテスト

```go
func (s *XxxTestSuite) TestProtectedEndpoint() {
    // 1. ユーザー登録＆アクティベート
    s.registerAndActivateUser("test@example.com", "Password123", "Test User")

    // 2. ログインしてトークン取得
    loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/auth/login",
        Body: map[string]string{
            "email":    "test@example.com",
            "password": "Password123",
        },
    })
    loginResp.AssertStatus(http.StatusOK)

    data := loginResp.GetJSONData()
    accessToken := data["access_token"].(string)

    // 3. 認証付きリクエスト
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method:      http.MethodGet,
        Path:        "/api/v1/protected",
        AccessToken: accessToken,  // Authorization: Bearer {token}
    })

    resp.AssertStatus(http.StatusOK)
}

// ヘルパーメソッド
func (s *XxxTestSuite) registerAndActivateUser(email, password, name string) {
    // 登録
    testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/auth/register",
        Body: map[string]string{
            "email":    email,
            "password": password,
            "name":     name,
        },
    }).AssertStatus(http.StatusCreated)

    // DBで直接アクティベート
    _, err := s.server.Pool.Exec(
        context.Background(),
        "UPDATE users SET status = 'active' WHERE email = $1",
        email,
    )
    s.Require().NoError(err)
}
```

---

## アサーションメソッド

### ステータスコード

```go
resp.AssertStatus(http.StatusOK)           // 200
resp.AssertStatus(http.StatusCreated)      // 201
resp.AssertStatus(http.StatusBadRequest)   // 400
resp.AssertStatus(http.StatusUnauthorized) // 401
resp.AssertStatus(http.StatusConflict)     // 409
```

### JSON パス検証

```go
// 値の検証
resp.AssertJSONPath("data.user.email", "test@example.com")
resp.AssertJSONPath("data.message", "success")

// 存在確認（値は問わない）
resp.AssertJSONPathExists("data.id")
resp.AssertJSONPathExists("data.access_token")
```

### エラーレスポンス検証

```go
// エラーコードとメッセージ両方
resp.AssertJSONError("VALIDATION_ERROR", "email is required")

// エラーコードのみ（メッセージは任意）
resp.AssertJSONError("UNAUTHORIZED", "")
```

### Cookie 検証

```go
cookie := resp.GetCookie("refresh_token")
s.NotNil(cookie)
s.True(cookie.HttpOnly)
s.Equal("/api/v1/auth", cookie.Path)
```

### レスポンスデータ取得

```go
// data フィールドを取得
data := resp.GetJSONData()
userId := data["user_id"].(string)
accessToken := data["access_token"].(string)

// 全JSONを取得
json := resp.GetJSON()
```

---

## テストデータの管理

### テスト間の分離

各テスト前に `SetupTest()` が呼ばれ、以下がクリーンアップされます：

- PostgreSQL テーブル（TRUNCATE CASCADE）
- Redis キー（FLUSHDB）

```go
func (s *XxxTestSuite) SetupTest() {
    s.server.Cleanup(s.T())
}
```

### 追加テーブルのクリーンアップ

新しいテーブルを追加した場合は、`server.go` の `Cleanup` メソッドを更新：

```go
func (ts *TestServer) Cleanup(t *testing.T) {
    t.Helper()
    // テーブル名をここに追加（外部キー制約の順序に注意）
    TruncateTables(t, ts.Pool, "sessions", "oauth_accounts", "users", "new_table")
    FlushRedis(t, ts.Redis)
}
```

---

## テストサーバーの拡張

### 新しいハンドラの追加

`testutil/server.go` を編集：

```go
func NewTestServer(t *testing.T) *TestServer {
    // ... 既存のコード ...

    // 新しいユースケース
    newUC := usecase.NewXxxUseCase(xxxRepo, txManager)

    // 新しいハンドラ
    newHandler := handler.NewXxxHandler(newUC)

    // ルート追加
    setupTestRoutes(e, authHandler, newHandler, jwtAuthMiddleware, rateLimitMiddleware)

    return &TestServer{
        // ... 既存のフィールド ...
        XxxHandler: newHandler,
    }
}

func setupTestRoutes(
    e *echo.Echo,
    authHandler *handler.AuthHandler,
    xxxHandler *handler.XxxHandler,  // 追加
    jwtAuthMiddleware *middleware.JWTAuthMiddleware,
    rateLimitMiddleware *middleware.RateLimitMiddleware,
) {
    // ... 既存のルート ...

    // 新しいルート
    api.GET("/xxx", xxxHandler.List, jwtAuthMiddleware.Authenticate())
    api.POST("/xxx", xxxHandler.Create, jwtAuthMiddleware.Authenticate())
}
```

---

## トラブルシューティング

### テストがスキップされる

```
=== RUN   TestAuthSuite
    auth_test.go:40: Skipping integration tests. Set INTEGRATION_TEST=true to run.
```

**解決策**: `task backend:test-integration` を使用するか、環境変数を設定：

```bash
INTEGRATION_TEST=true go test -v ./tests/integration/...
```

### データベース接続エラー

```
Failed to connect to test database: connection refused
```

**解決策**: インフラが起動しているか確認：

```bash
task infra:up
docker compose ps  # すべて running であること
```

### テスト間でデータが残る

**解決策**: `SetupTest()` で `Cleanup` が呼ばれているか確認。新しいテーブルは明示的に追加が必要。

### Redis 接続エラー

```
Failed to ping Redis: connection refused
```

**解決策**: Redis が起動しているか確認。デフォルトは `redis://localhost:6379/1`（DB 1 を使用）。

---

## 環境変数

| 変数名 | 説明 | デフォルト |
|--------|------|----------|
| `INTEGRATION_TEST` | `true` でインテグレーションテスト有効 | (未設定でスキップ) |
| `TEST_DATABASE_URL` | テスト用DB接続URL | `postgres://postgres:postgres@localhost:5432/gc_storage?sslmode=disable` |
| `TEST_REDIS_URL` | テスト用Redis接続URL | `redis://localhost:6379/1` |

**注意**: Taskfile.yml では `DATABASE_URL` と `REDIS_URL` が自動的に `TEST_*` 変数にマッピングされます。

---

## 参考

- [testify/suite ドキュメント](https://pkg.go.dev/github.com/stretchr/testify/suite)
- [Echo Testing](https://echo.labstack.com/docs/testing)
- [docs/01-policies/TESTING.md](../../docs/01-policies/TESTING.md) - テスト方針全体
