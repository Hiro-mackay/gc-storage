# TDDワークフローガイド

## 概要

本ドキュメントでは、GC Storage プロジェクトにおけるTDD（テスト駆動開発）ワークフローを定義します。

### 目的

1. **テスト要件の先行定義**: 実装前に「あるべき挙動」を明確化
2. **設計品質の向上**: テストを先に書くことで、使いやすいAPIを設計
3. **リグレッション防止**: 安全なリファクタリングを可能に
4. **ドキュメントとしてのテスト**: テストコードが仕様書の役割を果たす

---

## TDDサイクル（Red-Green-Refactor）

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│    RED      │────▶│   GREEN     │────▶│  REFACTOR   │
│             │     │             │     │             │
│ テスト作成   │     │ 最小実装    │     │ コード改善   │
│ (FAIL)      │     │ (PASS)      │     │ (PASS維持)  │
└─────────────┘     └─────────────┘     └──────┬──────┘
       ▲                                       │
       └───────────────────────────────────────┘
```

---

## Phase 1: 要件定義（RED）

### 1.1 機能要件の整理

機能開発を始める前に、以下を明確にします。

| 項目 | 内容 |
|------|------|
| APIエンドポイント | HTTPメソッド、パス、認証要否 |
| 入力仕様 | リクエストボディ、パラメータ、バリデーションルール |
| 出力仕様 | レスポンス形式、HTTPステータスコード |
| エラーケース | バリデーションエラー、認証エラー、ビジネスエラー |

### 1.2 テストケース一覧作成

テストケースを網羅的に洗い出します。テストケース設計には `docs/04-specs/templates/TEST_SPEC_TEMPLATE.md` を使用してください。

**テストカテゴリ:**

| カテゴリ | 説明 | 例 |
|---------|------|-----|
| 正常系 | 期待通りの動作 | 有効なデータで登録成功 |
| 異常系（バリデーション） | 入力エラー | 空の値、不正なフォーマット |
| 異常系（認証・認可） | 権限エラー | 未認証、権限不足 |
| 境界値・エッジケース | 境界条件 | 最大値、最小値、空リスト |

### 1.3 テストコード作成

テストコードを作成し、全テストが **FAIL** することを確認します。

```go
// 例: tests/integration/auth_test.go

func (s *AuthTestSuite) TestRegister_Success() {
    // Given: 有効なユーザー情報

    // When: 登録APIを呼び出し
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/auth/register",
        Body: map[string]string{
            "email":    "test@example.com",
            "password": "Password123",
            "name":     "Test User",
        },
    })

    // Then: 201 Created が返る
    resp.AssertStatus(http.StatusCreated).
        AssertJSONPathExists("data.user_id")
}
```

**この時点でのチェックリスト:**

- [ ] 全テストケースが作成されている
- [ ] 全テストが FAIL する（実装がないため）
- [ ] テストは独立して実行可能
- [ ] テスト名が意図を明確に表現している

---

## Phase 2: 最小実装（GREEN）

### 2.1 インターフェース定義

まず、必要なインターフェースを定義します。

**順序:**

1. **Entity/ValueObject** - ドメインモデル
2. **Repository IF** - データアクセスインターフェース
3. **UseCase IF** - ユースケース入出力

```go
// 1. Entity
type User struct {
    ID        uuid.UUID
    Email     string
    Name      string
    // ...
}

// 2. Repository IF
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    FindByEmail(ctx context.Context, email string) (*User, error)
}

// 3. UseCase Input/Output
type RegisterInput struct {
    Email    string
    Password string
    Name     string
}

type RegisterOutput struct {
    UserID uuid.UUID
}
```

### 2.2 基本実装

テストをパスするための **最小限** の実装を行います。

**実装順序:**

1. **Handler** - HTTPリクエストの受付
2. **UseCase** - ビジネスロジック
3. **Repository** - データ永続化

```go
// UseCase 実装例
func (c *RegisterCommand) Execute(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
    // バリデーション
    if input.Email == "" {
        return nil, errors.New("email is required")
    }

    // ユーザー作成
    user, err := entity.NewUser(input.Email, input.Name)
    if err != nil {
        return nil, err
    }

    // 保存
    if err := c.userRepo.Create(ctx, user); err != nil {
        return nil, err
    }

    return &RegisterOutput{UserID: user.ID}, nil
}
```

### 2.3 テスト実行

全テストが **PASS** することを確認します。

```bash
# ユニットテスト
task backend:test

# インテグレーションテスト
task backend:test-integration

# 全テスト
task backend:test-all
```

**この時点でのチェックリスト:**

- [ ] 全テストが PASS している
- [ ] 実装は最小限（過剰な機能追加をしていない）
- [ ] エラーハンドリングが実装されている

---

## Phase 3: リファクタリング（REFACTOR）

### 3.1 コード改善

テストを維持しながら、コードの品質を向上させます。

**リファクタリング対象:**

| 観点 | アクション |
|------|----------|
| 重複排除 | 共通処理の抽出 |
| 命名改善 | より明確な変数名・関数名 |
| 構造最適化 | 責務の分離、レイヤー整理 |
| パフォーマンス | N+1問題の解消、インデックス追加 |

### 3.2 テスト実行

リファクタリング後も全テストが **PASS** することを確認します。

```bash
task backend:test-all
```

**この時点でのチェックリスト:**

- [ ] 全テストが PASS している
- [ ] コードの可読性が向上している
- [ ] 重複コードが排除されている
- [ ] 新しいバグを導入していない

---

## テスト種別ごとの実装ガイド

### インテグレーションテスト（推奨）

API単位でエンドツーエンドの動作を検証します。

```go
// tests/integration/xxx_test.go
func (s *XXXTestSuite) TestFeature_Scenario() {
    // Given: テストデータのセットアップ

    // When: APIリクエスト
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/xxx",
        Body:   map[string]interface{}{...},
    })

    // Then: レスポンス検証
    resp.AssertStatus(http.StatusOK).
        AssertJSONPath("data.field", expectedValue)
}
```

### ユニットテスト

個別の関数・メソッドの動作を検証します。

```go
// internal/domain/entity/xxx_test.go
func TestNewXXX_ValidInput_ReturnsInstance(t *testing.T) {
    // Given
    input := XXXInput{...}

    // When
    result, err := NewXXX(input)

    // Then
    require.NoError(t, err)
    assert.Equal(t, expected, result.Field)
}
```

---

## テストファイル構成

```
backend/
├── internal/
│   ├── domain/entity/
│   │   ├── user.go
│   │   └── user_test.go          # Entity ユニットテスト
│   ├── usecase/auth/command/
│   │   ├── register.go
│   │   └── register_test.go      # UseCase ユニットテスト（モック使用）
│   └── interface/handler/
│       ├── auth_handler.go
│       └── auth_handler_test.go  # Handler ユニットテスト
└── tests/
    ├── integration/
    │   └── auth_test.go          # インテグレーションテスト
    └── testutil/
        ├── setup.go              # テスト環境セットアップ
        ├── server.go             # テストサーバー
        └── http.go               # HTTPヘルパー
```

---

## テストコード品質基準

### 命名規則

```go
// 関数名: Test{対象}_{条件}_{期待結果}
func TestRegister_ValidInput_ReturnsUserID(t *testing.T) {}
func TestRegister_EmptyEmail_ReturnsValidationError(t *testing.T) {}
func TestRegister_DuplicateEmail_ReturnsConflictError(t *testing.T) {}
```

### テスト構造（Given-When-Then / AAA）

```go
func TestExample(t *testing.T) {
    // Given (Arrange): テストの前提条件
    input := "test@example.com"

    // When (Act): テスト対象の実行
    result, err := ValidateEmail(input)

    // Then (Assert): 結果の検証
    require.NoError(t, err)
    assert.True(t, result)
}
```

### テストの独立性

- 各テストは他のテストに依存しない
- テスト実行順序に依存しない
- 外部状態（DB、ファイル）は各テストでクリーンアップ

---

## ワークフロー実践例

### 例: ユーザー登録機能のTDD

#### Step 1: テストケース設計

`docs/04-specs/templates/TEST_SPEC_TEMPLATE.md` を使用してテストケースを設計。

#### Step 2: テストコード作成（RED）

```go
func (s *AuthTestSuite) TestRegister_Success() {
    resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
        Method: http.MethodPost,
        Path:   "/api/v1/auth/register",
        Body: map[string]string{
            "email":    "test@example.com",
            "password": "Password123",
            "name":     "Test User",
        },
    })

    resp.AssertStatus(http.StatusCreated).
        AssertJSONPathExists("data.user_id").
        AssertJSONPath("data.message", "Registration successful...")
}

func (s *AuthTestSuite) TestRegister_InvalidEmail() {
    // ... (バリデーションエラーのテスト)
}

func (s *AuthTestSuite) TestRegister_DuplicateEmail() {
    // ... (重複エラーのテスト)
}
```

テスト実行 → 全 FAIL を確認

#### Step 3: 最小実装（GREEN）

Handler → UseCase → Repository の順に実装し、テストが PASS するまで繰り返し。

#### Step 4: リファクタリング（REFACTOR）

- エラーメッセージの改善
- バリデーションロジックの共通化
- 不要なコードの削除

テスト実行 → 全 PASS を確認

---

## 関連ドキュメント

- [TESTING.md](./TESTING.md) - テスト戦略全般
- [CODING_STANDARDS.md](./CODING_STANDARDS.md) - コーディング規約
- [TEST_SPEC_TEMPLATE.md](../04-specs/templates/TEST_SPEC_TEMPLATE.md) - テストケース設計テンプレート

---

## 更新履歴

| 日付 | バージョン | 内容 |
|------|-----------|------|
| 2026-01-18 | 1.0.0 | 初版作成 |
