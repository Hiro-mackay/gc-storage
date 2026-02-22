# Login (Email/OAuth) + Logout

## Meta

| Item | Value |
|------|-------|
| Status | Ready |
| Priority | High |
| Tier | 1 (Auth) |
| Domain Refs | `03-domains/user.md` |
| Depends On | `features/auth-registration.md` |

---

## 1. User Stories

**Primary:**
> As a registered user, I want to log in with my email and password so that I can access my files.

**Secondary:**
> As a user, I want to log in with Google or GitHub so that I don't need to remember a separate password.

> As a logged-in user, I want to log out so that my session is terminated securely.

### Context
認証方式はSession IDベース（HttpOnly Cookie `session_id`）を採用。JWTは使用しない。セッションはRedisに保存され、スライディングウィンドウ方式で有効期限を延長する。OAuth認証ではフロントエンドがリダイレクト・コールバックを処理し、認可コードをバックエンドに送信するフローを採用。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-U004 | status=suspended/deactivatedの場合ログイン不可。pendingはログイン可能 | `03-domains/user.md` |
| R-U005 | email_verified=falseの場合、重要操作に制限（基本操作は許可） | `03-domains/user.md` |
| R-S001 | expires_atを過ぎたセッションは無効 | `03-domains/user.md` |
| R-S002 | 同一ユーザーの有効セッションは最大10個 | `03-domains/user.md` |
| R-S003 | 新規セッション作成時、最古のセッションを自動失効 | `03-domains/user.md` |
| R-OA001 | 同一userに対して同一providerのアカウントは1つのみ | `03-domains/user.md` |
| R-OA002 | provider + provider_user_idの組み合わせは一意 | `03-domains/user.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-LOGIN-001 | status=pendingの場合もログインを許可する（メール確認は重要操作時にのみ要求） |
| FS-LOGIN-002 | OAuth専用ユーザーがパスワードログインを試みた場合「invalid credentials」汎用エラー（認証方式を漏洩しない） |
| FS-LOGIN-003 | ログイン成功時、Session IDをHttpOnly Cookieに設定 |
| FS-LOGIN-004 | セッション有効期限は7日、スライディングウィンドウで延長 |
| FS-LOGIN-005 | ログアウト時、Redisからセッション削除 + Cookie削除 |
| FS-LOGIN-006 | OAuth新規ユーザー作成時もPersonal Folderを自動作成 |
| FS-LOGIN-007 | OAuthでpendingユーザーにアカウント紐付けした場合、status=activeに変更 |

### State Transitions
```
[Login]
Guest --login(email/pwd)--> User(active|pending) + Session(Redis)
Guest --oauth--> User(active) + Session(Redis)

[Logout]
User(active|pending) --logout--> Guest (Session deleted)
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/login` | No | メール/パスワードログイン |
| POST | `/api/v1/auth/oauth/:provider` | No | OAuthログイン |
| POST | `/api/v1/auth/logout` | Required | ログアウト |
| GET | `/api/v1/me` | Required | 現在のユーザー情報取得 |

### Request / Response Details

#### `POST /api/v1/auth/login` - メール/パスワードログイン

**Request Body:**
```json
{
  "email": "taro@example.com",
  "password": "SecurePass1"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| email | string | Yes | 有効なメール形式 | メールアドレス |
| password | string | Yes | 必須 | パスワード |

**Success Response (200):**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "taro@example.com",
    "name": "Taro Yamada",
    "status": "active",
    "email_verified": true,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

**Response Headers:**
```
Set-Cookie: session_id=xxx; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=604800
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 401 | 認証情報不正（ユーザー未存在、パスワード不一致、OAuth専用ユーザー） | `UNAUTHORIZED` |
| 401 | アカウント停止/無効化 | `UNAUTHORIZED` |

> Note: セキュリティ上、認証失敗時はすべて汎用的な「invalid credentials」エラーを返す。OAuth専用ユーザーかどうかも漏洩しない。pendingユーザーはログイン可能。

#### `POST /api/v1/auth/oauth/:provider` - OAuthログイン

**Path Parameter:**

| Field | Type | Values | Description |
|-------|------|--------|-------------|
| provider | string | `google`, `github` | OAuthプロバイダー |

**Request Body:**
```json
{
  "code": "authorization_code_from_provider"
}
```

**Success Response (200):**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "taro@example.com",
    "name": "Taro Yamada",
    "status": "active",
    "email_verified": true,
    "created_at": "2024-01-01T00:00:00Z"
  },
  "is_new_user": false
}
```

**Response Headers:**
```
Set-Cookie: session_id=xxx; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=604800
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | 無効なプロバイダー | `VALIDATION_ERROR` |
| 400 | 無効な認可コード | `VALIDATION_ERROR` |
| 401 | アカウント停止/無効化 | `UNAUTHORIZED` |

#### `POST /api/v1/auth/logout` - ログアウト

**Request:** Cookie `session_id` が必須

**Success Response (200):**
```json
{
  "message": "logged out successfully"
}
```

**Response Headers:**
```
Set-Cookie: session_id=; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=-1
```

#### `GET /api/v1/me` - 現在のユーザー情報取得

**Request:** Cookie `session_id` が必須

**Success Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "taro@example.com",
  "name": "Taro Yamada",
  "status": "active",
  "email_verified": true,
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 401 | セッションなし/無効/期限切れ | `UNAUTHORIZED` |

---

## 4. Frontend UI

### Layout / Wireframe
```
[Login Page]
+-------------------------------------------+
|            GC Storage                     |
|  +-------------------------------------+  |
|  |          Log in                     |  |
|  |  [Email         ]                   |  |
|  |  [Password      ]                   |  |
|  |  [Log in]                           |  |
|  |  Forgot password?                   |  |
|  |  ---- or continue with ----         |  |
|  |  [Google] [GitHub]                  |  |
|  |  Don't have an account? Sign up     |  |
|  +-------------------------------------+  |
+-------------------------------------------+

[OAuth Callback - Processing]
+-------------------------------------------+
|  (spinner) Completing authentication...   |
+-------------------------------------------+

[OAuth Callback - Error]
+-------------------------------------------+
|  Authentication failed                    |
|  Authorization was denied.                |
|  [Back to login]                          |
+-------------------------------------------+
```

### Components

| Component | Type | Description |
|-----------|------|-------------|
| LoginPage | Page | ログインページ (`/auth/login`) |
| OAuthCallbackPage | Page | OAuthコールバック (`/auth/callback/:provider`) |
| LoginForm | Form | メール/パスワードログインフォーム |
| OAuthButtons | UI | Google/GitHub OAuthボタン |

### State Management

| State | Store | Type | Description |
|-------|-------|------|-------------|
| loginMutation | TanStack Query | Server | ログインAPI呼び出し |
| oauthLoginMutation | TanStack Query | Server | OAuthログインAPI呼び出し |
| meQuery | TanStack Query | Server | GET /me、認証状態確認 |
| authState | Zustand | Global | ログイン状態、現在のユーザー情報 |

### User Interactions
1. メールログイン: フォーム入力 -> 送信 -> Cookie設定 -> `/files` 遷移
2. OAuthログイン: ボタンクリック -> プロバイダーへリダイレクト -> コールバック -> API送信 -> `/files` 遷移
3. ログアウト: ボタンクリック -> セッション削除 -> `/auth/login` 遷移

---

## 5. Integration Flow

### Password Login Sequence
```
Client          Frontend        API             DB              Redis
  |                |              |               |               |
  |-- submit ----->|              |               |               |
  |                |-- POST ----->|               |               |
  |                |  /login      |-- find user ->|               |
  |                |              |-- verify pwd  |               |
  |                |              |-- check status|               |
  |                |              |-- check limit |------------->|
  |                |              |-- save session|------------->|
  |                |<-- 200 ------|               |               |
  |                |  Set-Cookie  |               |               |
  |<-- redirect -->|              |               |               |
  |   to /files    |              |               |               |
```

### OAuth Login Sequence
```
Client       Frontend     Provider     API          DB          Redis
  |             |             |          |            |            |
  |-- click --->|             |          |            |            |
  |             |-- redirect->|          |            |            |
  |             |<-- code ----|          |            |            |
  |             |-- POST /oauth/:prov ->|            |            |
  |             |             |          |-- exchange |            |
  |             |             |<---------|            |            |
  |             |             |-- token->|            |            |
  |             |             |          |-- get info |            |
  |             |             |<---------|            |            |
  |             |             |          |-- upsert ->|            |
  |             |             |          |-- session  |----------->|
  |             |<-- 200 + Set-Cookie --|            |            |
  |<-- /files --|             |          |            |            |
```

### Session Validation (Middleware)
```
Client          Frontend        API             Redis           DB
  |                |              |               |               |
  |-- request ---->|              |               |               |
  |   Cookie: session_id         |               |               |
  |                |-- GET /api ->|               |               |
  |                |              |-- find sess ->|               |
  |                |              |-- check exp   |               |
  |                |              |-- find user   |------------->|
  |                |              |-- refresh sess|->|            |
  |                |<-- 200 ------|               |               |
```

### Error Handling Flow
- 認証失敗: 401 -> 「Invalid email or password」表示（汎用エラー、OAuth専用も同じ）
- アカウント停止: 401 -> 「account suspended」表示
- OAuth拒否: コールバックでエラーパラメータ検出 -> エラー画面表示
- セッション期限切れ: 401 -> ログインページへリダイレクト

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: Given 正しいメール/パスワード, when ログイン送信, then session_id Cookieが設定されユーザー情報が返る
- [ ] AC-02: Given ログイン成功, when /files に遷移, then ファイル一覧が表示される
- [ ] AC-03: Given Google OAuth, when 認可完了, then セッションが作成されログインされる
- [ ] AC-04: Given GitHub OAuth, when 認可完了, then セッションが作成されログインされる
- [ ] AC-05: Given ログイン状態, when ログアウト, then セッションが削除されCookieがクリアされる
- [ ] AC-06: Given ログイン状態, when GET /me, then 現在のユーザー情報が返る

### Validation Errors
- [ ] AC-10: Given 不正なメール/パスワード, when ログイン送信, then 401「invalid credentials」が返る
- [ ] AC-11: Given 未確認メール(pending), when ログイン送信, then ログイン成功しアプリ利用可能
- [ ] AC-12: Given OAuth専用ユーザー, when パスワードログイン, then 401「invalid credentials」が返る（認証方式を漏洩しない）

### Authorization
- [ ] AC-20: Given セッションなし, when 認証必須API呼び出し, then 401が返る
- [ ] AC-21: Given 期限切れセッション, when API呼び出し, then 401が返る

### Edge Cases
- [ ] AC-30: Given 10セッション存在, when 新規ログイン, then 最古セッションが削除され新規作成される
- [ ] AC-31: Given 停止中アカウント, when ログイン, then 401が返る
- [ ] AC-32: Given OAuthで新規ユーザー, when 初回ログイン, then User + Personal Folder + Profileが作成される
- [ ] AC-33: Given 既存メールのpendingユーザー, when OAuth紐付け, then status=activeに変更される

---

## 7. Test Plan

### Backend Unit Tests

| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| 正常ログイン(active) | LoginCommand | セッション作成、SessionID返却 |
| 正常ログイン(pending) | LoginCommand | セッション作成、SessionID返却（pending許可） |
| パスワード不正 | LoginCommand | UnauthorizedError「invalid credentials」 |
| OAuth専用ユーザー | LoginCommand | UnauthorizedError「invalid credentials」（汎用エラー） |
| セッション上限 | LoginCommand | 最古セッション削除後に新規作成 |
| OAuth新規ユーザー | OAuthLoginCommand | User + Folder + Profile作成 |
| OAuth既存ユーザー紐付け | OAuthLoginCommand | OAuthAccount作成 |
| ログアウト | LogoutCommand | セッション削除 |

### Backend Integration Tests

| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| パスワードログイン | POST /login | active user | 200, Set-Cookie |
| OAuthログイン | POST /oauth/google | - | 200, Set-Cookie |
| ログアウト | POST /logout | active session | 200, Cookie cleared |
| ユーザー情報取得 | GET /me | active session | 200, user info |
| 未認証アクセス | GET /me | no session | 401 |

### Frontend Tests

| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| ログインフォーム | LoginForm | Unit | バリデーション、送信処理 |
| OAuthコールバック | OAuthCallbackPage | Integration | 認可コード処理、リダイレクト |
| 認証ガード | AuthGuard | Unit | 未認証時リダイレクト |
| ログアウト | LogoutButton | Unit | セッション削除、リダイレクト |

### E2E Tests (future)

| Test | Flow | Assertions |
|------|------|------------|
| メールログイン->ファイル一覧 | login -> /files | ファイル一覧表示 |
| OAuthログイン->ファイル一覧 | oauth -> /files | ファイル一覧表示 |
| ログアウト->ログインページ | logout -> /auth/login | ログインページ表示 |

---

## 8. Implementation Notes

### Changed Files (Backend)

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/session.go` | Session entity, TTL, Refresh |
| Domain | `internal/domain/entity/oauth_account.go` | OAuthAccount entity |
| Domain | `internal/domain/service/oauth_client.go` | OAuthClient IF |
| Domain | `internal/domain/repository/session_repository.go` | SessionRepo IF |
| Domain | `internal/domain/repository/oauth_account_repository.go` | OAuthAccountRepo IF |
| UseCase | `internal/usecase/auth/command/login.go` | LoginCommand |
| UseCase | `internal/usecase/auth/command/oauth_login.go` | OAuthLoginCommand |
| UseCase | `internal/usecase/auth/command/logout.go` | LogoutCommand |
| Interface | `internal/interface/handler/auth_handler.go` | Login, OAuthLogin, Logout, Me |
| Interface | `internal/interface/middleware/auth.go` | Session validation middleware |
| Infra | `internal/infrastructure/redis/session_repository.go` | Redis SessionRepo impl |
| Infra | `internal/infrastructure/oauth/google.go` | Google OAuth client |
| Infra | `internal/infrastructure/oauth/github.go` | GitHub OAuth client |

### Changed Files (Frontend)

| Category | File | Change |
|----------|------|--------|
| Page | `src/features/auth/pages/login-page.tsx` | ログインページ |
| Page | `src/features/auth/pages/oauth-callback-page.tsx` | OAuthコールバック |
| Component | `src/features/auth/components/oauth-buttons.tsx` | OAuthボタン |
| Hook | `src/features/auth/hooks/use-oauth-callback.ts` | OAuthコールバック処理 |
| Query | `src/features/auth/api/queries.ts` | meQuery |
| Mutation | `src/features/auth/api/mutations.ts` | login, oauthLogin, logout |
| Store | `src/stores/auth-store.ts` | 認証状態管理 (Zustand) |

### Considerations
- **Security**: Cookie設定 (HttpOnly, Secure, SameSite=Lax)、レート制限 10 req/min/IP
- **Session**: Redis TTL 7日、スライディングウィンドウ延長、最大10セッション/ユーザー
- **OAuth**: SameSite=Lax でリダイレクト時のCookie送信を許可
