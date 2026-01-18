# テストケース設計テンプレート

このテンプレートは、TDDワークフローの Phase 1（要件定義）で使用します。
新機能開発時にこのテンプレートをコピーして、テストケースを設計してください。

---

## 機能名: `[機能名を記入]`

### 1. 概要

| 項目 | 内容 |
|------|------|
| 目的 | [この機能が何を実現するか] |
| APIエンドポイント | `[HTTP_METHOD] /api/v1/[path]` |
| 認証要否 | 要 / 不要 |
| 関連ドメイン | [Identity / Storage / Collaboration / etc.] |

### 2. 入出力仕様

#### リクエスト

```json
{
  "field1": "string (必須, 説明)",
  "field2": 123,
  "field3": true
}
```

#### レスポンス（成功時）

```json
{
  "data": {
    "id": "uuid",
    "field": "value"
  },
  "meta": null
}
```

#### HTTPステータスコード

| ステータス | 説明 |
|-----------|------|
| 200 OK | 成功（GET, PUT, PATCH） |
| 201 Created | 成功（POST） |
| 204 No Content | 成功（DELETE） |
| 400 Bad Request | バリデーションエラー |
| 401 Unauthorized | 認証エラー |
| 403 Forbidden | 認可エラー |
| 404 Not Found | リソース未存在 |
| 409 Conflict | 競合（重複など） |
| 500 Internal Server Error | サーバーエラー |

---

### 3. テストケース一覧

#### 3.1 正常系

| # | テスト名 | 入力 | 期待結果 | HTTPステータス |
|---|---------|------|---------|---------------|
| 1 | [テスト名] | [入力データの概要] | [期待する結果] | [ステータスコード] |
| 2 | | | | |
| 3 | | | | |

#### 3.2 異常系（バリデーション）

| # | テスト名 | 入力 | 期待結果 | HTTPステータス |
|---|---------|------|---------|---------------|
| 1 | [フィールド名]_空の場合 | `{ "field": "" }` | VALIDATION_ERROR | 400 |
| 2 | [フィールド名]_不正フォーマット | `{ "field": "invalid" }` | VALIDATION_ERROR | 400 |
| 3 | [フィールド名]_最大長超過 | `{ "field": "a" * 256 }` | VALIDATION_ERROR | 400 |
| 4 | | | | |

#### 3.3 異常系（認証・認可）

| # | テスト名 | 条件 | 期待結果 | HTTPステータス |
|---|---------|------|---------|---------------|
| 1 | 未認証アクセス | Authorizationヘッダーなし | UNAUTHORIZED | 401 |
| 2 | 無効なトークン | 期限切れ/改ざんトークン | UNAUTHORIZED | 401 |
| 3 | 権限不足 | 他ユーザーのリソースへのアクセス | FORBIDDEN | 403 |
| 4 | | | | |

#### 3.4 異常系（ビジネスルール）

| # | テスト名 | 条件 | 期待結果 | HTTPステータス |
|---|---------|------|---------|---------------|
| 1 | [リソース]が存在しない | 存在しないIDを指定 | NOT_FOUND | 404 |
| 2 | [リソース]が重複 | 既存の値を指定 | CONFLICT | 409 |
| 3 | | | | |

#### 3.5 境界値・エッジケース

| # | テスト名 | 条件 | 期待結果 | HTTPステータス |
|---|---------|------|---------|---------------|
| 1 | [フィールド名]の最小値 | 許容される最小値 | 成功 | 200/201 |
| 2 | [フィールド名]の最大値 | 許容される最大値 | 成功 | 200/201 |
| 3 | 空のリスト | 対象データが0件 | 空配列を返す | 200 |
| 4 | | | | |

---

### 4. テストコード構造

```go
// tests/integration/[feature]_test.go

package integration

import (
    "net/http"
    "testing"

    "github.com/stretchr/testify/suite"
    "github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// [Feature]TestSuite はテストスイートの定義
type [Feature]TestSuite struct {
    suite.Suite
    server *testutil.TestServer
}

// SetupSuite は全テスト前に1回だけ実行
func (s *[Feature]TestSuite) SetupSuite() {
    s.server = testutil.NewTestServer(s.T())
}

// TearDownSuite は全テスト後に1回だけ実行
func (s *[Feature]TestSuite) TearDownSuite() {
    testutil.CleanupTestEnvironment()
}

// SetupTest は各テスト前に実行（DBクリーンアップ）
func (s *[Feature]TestSuite) SetupTest() {
    s.server.Cleanup(s.T())
}

// Test[Feature]Suite はテストスイートのエントリーポイント
func Test[Feature]Suite(t *testing.T) {
    if os.Getenv("INTEGRATION_TEST") != "true" {
        t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
    }
    suite.Run(t, new([Feature]TestSuite))
}

// =============================================================================
// 正常系テスト
// =============================================================================

func (s *[Feature]TestSuite) Test[Action]_Success() {
    // Given: 前提条件のセットアップ
    // - 必要なテストデータの作成
    // - 認証ユーザーの準備

    // When: APIリクエストの実行
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/[endpoint]",
        Body: map[string]interface{}{
            "field1": "value1",
            "field2": 123,
        },
        AccessToken: accessToken, // 認証が必要な場合
    })

    // Then: レスポンスの検証
    resp.AssertStatus(http.StatusCreated).
        AssertJSONPathExists("data.id").
        AssertJSONPath("data.field1", "value1")
}

// =============================================================================
// 異常系テスト（バリデーション）
// =============================================================================

func (s *[Feature]TestSuite) Test[Action]_InvalidInput_[Field]Empty() {
    // Given: 空のフィールド

    // When: APIリクエスト
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/[endpoint]",
        Body: map[string]interface{}{
            "field1": "",
        },
    })

    // Then: バリデーションエラー
    resp.AssertStatus(http.StatusBadRequest).
        AssertJSONError("VALIDATION_ERROR", "")
}

// =============================================================================
// 異常系テスト（認証・認可）
// =============================================================================

func (s *[Feature]TestSuite) Test[Action]_Unauthorized() {
    // Given: 認証なし

    // When: APIリクエスト（トークンなし）
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/[endpoint]",
        Body:   map[string]interface{}{},
        // AccessToken なし
    })

    // Then: 401 Unauthorized
    resp.AssertStatus(http.StatusUnauthorized).
        AssertJSONError("UNAUTHORIZED", "")
}

func (s *[Feature]TestSuite) Test[Action]_Forbidden() {
    // Given: 他ユーザーのリソースへのアクセス

    // When: APIリクエスト
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method:      http.MethodGet,
        Path:        "/api/v1/[endpoint]/[other-user-resource-id]",
        AccessToken: accessToken,
    })

    // Then: 403 Forbidden
    resp.AssertStatus(http.StatusForbidden).
        AssertJSONError("FORBIDDEN", "")
}

// =============================================================================
// ヘルパーメソッド
// =============================================================================

// create[Resource] はテストデータ作成のヘルパー
func (s *[Feature]TestSuite) create[Resource]() *[Resource] {
    // テストデータを作成して返す
}
```

---

### 5. チェックリスト

#### Phase 1: 要件定義（RED）

- [ ] 機能要件が整理されている
- [ ] 入出力仕様が明確である
- [ ] 正常系テストケースが網羅されている
- [ ] 異常系（バリデーション）テストケースが網羅されている
- [ ] 異常系（認証・認可）テストケースが網羅されている
- [ ] 境界値・エッジケースが考慮されている
- [ ] テストコードが作成され、全て FAIL することを確認

#### Phase 2: 最小実装（GREEN）

- [ ] Entity/ValueObject が実装されている
- [ ] Repository IF が定義されている
- [ ] UseCase が実装されている
- [ ] Handler が実装されている
- [ ] 全テストが PASS することを確認

#### Phase 3: リファクタリング（REFACTOR）

- [ ] 重複コードが排除されている
- [ ] 命名が適切である
- [ ] エラーハンドリングが適切である
- [ ] 全テストが PASS することを確認

---

### 6. 参考: よく使うテストパターン

#### パスワードバリデーション

```go
func (s *AuthTestSuite) TestRegister_WeakPassword() {
    testCases := []struct {
        name     string
        password string
    }{
        {"短すぎる", "Pass1"},
        {"大文字なし", "password123"},
        {"小文字なし", "PASSWORD123"},
        {"数字なし", "PasswordABC"},
    }

    for _, tc := range testCases {
        s.Run(tc.name, func() {
            resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
                Method: http.MethodPost,
                Path:   "/api/v1/auth/register",
                Body: map[string]string{
                    "email":    "test@example.com",
                    "password": tc.password,
                    "name":     "Test User",
                },
            })
            resp.AssertStatus(http.StatusBadRequest).
                AssertJSONError("VALIDATION_ERROR", "")
        })
    }
}
```

#### 認証付きリクエスト

```go
func (s *[Feature]TestSuite) Test[Action]_WithAuth() {
    // ユーザー登録＆ログイン
    s.registerAndActivateUser("test@example.com", "Password123", "Test User")
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

    // 認証付きリクエスト
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method:      http.MethodGet,
        Path:        "/api/v1/[protected-endpoint]",
        AccessToken: accessToken,
    })

    resp.AssertStatus(http.StatusOK)
}
```

#### DBデータの直接検証

```go
func (s *[Feature]TestSuite) Test[Action]_DatabaseSideEffect() {
    // APIリクエスト後、DBの状態を直接検証
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{...})
    resp.AssertStatus(http.StatusCreated)

    // DBの状態を検証
    var count int
    err := s.server.Pool.QueryRow(
        context.Background(),
        "SELECT COUNT(*) FROM [table] WHERE [condition]",
    ).Scan(&count)
    s.Require().NoError(err)
    s.Equal(1, count)
}
```

---

## 関連ドキュメント

- [TDD_WORKFLOW.md](../../01-policies/TDD_WORKFLOW.md) - TDDワークフローガイド
- [TESTING.md](../../01-policies/TESTING.md) - テスト戦略全般

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-18 | 1.0.0 | 初版作成 |
