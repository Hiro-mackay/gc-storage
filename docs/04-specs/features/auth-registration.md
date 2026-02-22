# Registration + Email Verification

## Meta

| Item | Value |
|------|-------|
| Status | Ready |
| Priority | High |
| Tier | 1 (Auth) |
| Domain Refs | `03-domains/user.md` |
| Depends On | - |

---

## 1. User Stories

**Primary:**
> As a new user, I want to create an account with my email and password so that I can start using GC Storage.

**Secondary:**
> As a newly registered user, I want to verify my email address so that my account becomes active.

### Context
ユーザーはGC Storageを利用するためにアカウントを作成する必要がある。登録時にPersonal Folderが自動作成され、メール確認完了後にログイン可能になる。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-U001 | emailは全ユーザーで一意 | `03-domains/user.md` |
| R-U002 | password_hashはOAuth専用ユーザーのみNULL許容 | `03-domains/user.md` |
| R-U003 | nameは空文字不可、1-100文字 | `03-domains/user.md` |
| R-U006 | personal_folder_idはユーザー登録処理完了後に設定 | `03-domains/user.md` |
| R-U007 | UserとPersonal Folderは1対1の関係 | `03-domains/user.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-REG-001 | 登録時にUserはstatus=pending、email_verified=falseで作成 |
| FS-REG-002 | Personal Folderの初期名はユーザー名 |
| FS-REG-003 | 確認メール送信失敗でも登録自体は成功扱い |
| FS-REG-004 | 確認トークンの有効期限は24時間 |
| FS-REG-005 | メール確認完了でstatus=active、email_verified=trueに更新 |
| FS-REG-006 | 既に確認済みの場合は冪等にメッセージを返す |

### State Transitions
```
[Registration]       [Email Verification]
Guest --register--> User(pending) --verify--> User(active)
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register` | No | ユーザー登録 |
| POST | `/api/v1/auth/email/verify` | No | メール確認 |
| POST | `/api/v1/auth/email/resend` | No | 確認メール再送 |

### Request / Response Details

#### `POST /api/v1/auth/register` - ユーザー登録

**Request Body:**
```json
{
  "name": "Taro Yamada",
  "email": "taro@example.com",
  "password": "SecurePass1"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| name | string | Yes | 1-100文字 | 表示名 |
| email | string | Yes | RFC 5322, max 255 | メールアドレス |
| password | string | Yes | 8-256文字、大文字+数字含む | パスワード |

**Success Response (201):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Registration successful. Please check your email to verify your account."
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | バリデーションエラー | `VALIDATION_ERROR` |
| 409 | メールアドレス重複 | `CONFLICT` |

#### `POST /api/v1/auth/email/verify?token=xxx` - メール確認

**Query Parameter:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| token | string | Yes | 確認トークン |

**Success Response (200):**
```json
{
  "message": "Email verified successfully"
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | 無効または期限切れトークン | `VALIDATION_ERROR` |
| 404 | ユーザーが見つからない | `NOT_FOUND` |

#### `POST /api/v1/auth/email/resend` - 確認メール再送

**Request Body:**
```json
{
  "email": "taro@example.com"
}
```

**Success Response (200):**
```json
{
  "message": "If your email is registered, a verification link has been sent."
}
```

---

## 4. Frontend UI

### Layout / Wireframe
```
[Registration Page]
+-------------------------------------------+
|            GC Storage                     |
|  +-------------------------------------+  |
|  |      Create your account            |  |
|  |  [Name          ]                   |  |
|  |  [Email         ]                   |  |
|  |  [Password      ]                   |  |
|  |   * At least 8 characters           |  |
|  |   * At least one uppercase letter   |  |
|  |   * At least one number             |  |
|  |  [Confirm Password]                 |  |
|  |  [Create account]                   |  |
|  |  ---- or continue with ----         |  |
|  |  [Google] [GitHub]                  |  |
|  |  Already have an account? Log in    |  |
|  +-------------------------------------+  |
+-------------------------------------------+

[Registration Success]
+-------------------------------------------+
|  Check your email                         |
|  We've sent a verification link to        |
|  {email}. Please check your inbox.        |
|  [Back to login]                          |
+-------------------------------------------+

[Email Verification - Processing]
+-------------------------------------------+
|  (spinner) Verifying your email...        |
+-------------------------------------------+

[Email Verification - Success]
+-------------------------------------------+
|  Email verified!                          |
|  Your account has been verified.          |
|  [Go to login]                            |
+-------------------------------------------+

[Email Verification - Error]
+-------------------------------------------+
|  Verification failed                      |
|  Invalid or expired verification token.   |
|  [Resend verification email]              |
|  [Go to login]                            |
+-------------------------------------------+
```

### Components

| Component | Type | Description |
|-----------|------|-------------|
| RegisterPage | Page | 新規登録ページ (`/auth/register`) |
| VerifyEmailPage | Page | メール確認ページ (`/auth/verify-email`) |
| RegisterForm | Form | 登録フォーム (name, email, password, confirm) |
| PasswordStrengthIndicator | UI | パスワード強度リアルタイム表示 |
| OAuthButtons | UI | Google/GitHub OAuthボタン |

### State Management

| State | Store | Type | Description |
|-------|-------|------|-------------|
| registerMutation | TanStack Query | Server | 登録API呼び出し |
| verifyEmailMutation | TanStack Query | Server | メール確認API呼び出し |
| formState | useState | Local | フォーム入力値、バリデーション状態 |
| passwordStrength | useState | Local | パスワード強度チェック結果 |

### User Interactions
1. ユーザーがフォームに入力 -> パスワード強度がリアルタイム更新
2. 「Create account」クリック -> ボタン無効化、API呼び出し
3. 登録成功 -> 確認メッセージ画面表示
4. 確認メールのリンクをクリック -> VerifyEmailPage表示、自動確認処理
5. 確認成功 -> ログインページへ遷移可能

---

## 5. Integration Flow

### Registration Sequence
```
Client          Frontend        API             DB              Email
  |                |              |               |               |
  |-- submit ----->|              |               |               |
  |                |-- POST ----->|               |               |
  |                |  /register   |-- check dup ->|               |
  |                |              |<-- ok --------|               |
  |                |              |-- create folder->|            |
  |                |              |-- create user -->|            |
  |                |              |-- create token ->|            |
  |                |              |-- send email ----|---------->|
  |                |<-- 201 ------|               |               |
  |<-- success ----|              |               |               |
```

### Email Verification Sequence
```
Client          Frontend        API             DB
  |                |              |               |
  |-- click link ->|              |               |
  |                |-- POST ----->|               |
  |                |  /verify     |-- find token ->|
  |                |              |-- check expiry |
  |                |              |-- update user ->|
  |                |              |-- delete token ->|
  |                |<-- 200 ------|               |
  |<-- success ----|              |               |
```

### Error Handling Flow
- バリデーションエラー: APIが400返却 -> フロントエンドがフィールドごとにエラー表示
- メール重複: APIが409返却 -> 「An account with this email already exists」表示
- トークン期限切れ: APIが400返却 -> 再送リンク付きエラー画面表示

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: Given 有効なname/email/password, when 登録フォーム送信, then 201が返りPersonal Folderが自動作成される
- [ ] AC-02: Given 登録成功, when 確認メールのリンクをクリック, then ステータスがactiveに更新される
- [ ] AC-03: Given 確認完了, when ログインページへ遷移, then ログインが可能

### Validation Errors
- [ ] AC-10: Given パスワードが8文字未満, when 登録送信, then バリデーションエラーが表示される
- [ ] AC-11: Given 不正なメール形式, when 登録送信, then バリデーションエラーが表示される
- [ ] AC-12: Given 既存のメールアドレス, when 登録送信, then 重複エラーが表示される
- [ ] AC-13: Given パスワード不一致, when 登録送信, then クライアント側エラーが表示される

### Edge Cases
- [ ] AC-30: Given 確認済みトークン, when 再度確認リクエスト, then 冪等に成功メッセージが返る
- [ ] AC-31: Given 期限切れトークン, when 確認リクエスト, then エラーと再送リンクが表示される
- [ ] AC-32: Given 確認メール送信失敗, when 登録処理, then 登録自体は成功する

---

## 7. Test Plan

### Backend Unit Tests

| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| 正常な登録 | RegisterCommand | User(pending)作成、Personal Folder作成、トークン作成 |
| メール重複 | RegisterCommand | ConflictError返却 |
| パスワードバリデーション | Password VO | 8文字未満、大文字なし、数字なしで拒否 |
| 正常なメール確認 | VerifyEmailCommand | status=active、email_verified=true |
| 期限切れトークン | VerifyEmailCommand | ValidationError返却 |
| 確認済みユーザー | VerifyEmailCommand | 冪等にメッセージ返却 |

### Backend Integration Tests

| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| 登録成功 | POST /register | - | 201, user作成、folder作成 |
| 重複メール | POST /register | 既存user | 409 |
| メール確認成功 | POST /verify | pending user + token | 200, status=active |
| 無効トークン | POST /verify | - | 400 |

### Frontend Tests

| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| フォーム入力バリデーション | RegisterForm | Unit | 各フィールドのエラー表示 |
| パスワード強度表示 | PasswordStrengthIndicator | Unit | リアルタイム更新 |
| 登録成功フロー | RegisterPage | Integration | 成功画面遷移 |
| 確認成功フロー | VerifyEmailPage | Integration | 成功メッセージ表示 |

### E2E Tests (future)

| Test | Flow | Assertions |
|------|------|------------|
| 登録->メール確認->ログイン | フルフロー | アカウント作成から利用開始まで |

---

## 8. Implementation Notes

### Changed Files (Backend)

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/user.go` | User entity |
| Domain | `internal/domain/entity/email_verification_token.go` | Token entity |
| Domain | `internal/domain/valueobject/email.go` | Email VO |
| Domain | `internal/domain/valueobject/password.go` | Password VO |
| Domain | `internal/domain/repository/user_repository.go` | Repository IF |
| Domain | `internal/domain/repository/email_verification_token_repository.go` | Repository IF |
| UseCase | `internal/usecase/auth/command/register.go` | RegisterCommand |
| UseCase | `internal/usecase/auth/command/verify_email.go` | VerifyEmailCommand |
| UseCase | `internal/usecase/auth/command/resend_email_verification.go` | ResendCommand |
| Interface | `internal/interface/handler/auth_handler.go` | Register, VerifyEmail, Resend handlers |
| Interface | `internal/interface/dto/request/auth.go` | Request DTOs |
| Interface | `internal/interface/dto/response/auth.go` | Response DTOs |
| Infra | `internal/infrastructure/database/user_repository.go` | UserRepo impl |
| Infra | `internal/infrastructure/database/email_verification_token_repository.go` | TokenRepo impl |
| Infra | `internal/infrastructure/database/folder_repository.go` | FolderRepo impl |

### Changed Files (Frontend)

| Category | File | Change |
|----------|------|--------|
| Route | `src/app/routes/auth/register.tsx` | 登録ページ |
| Route | `src/app/routes/auth/verify-email.tsx` | メール確認ページ |
| Component | `src/components/auth/register-form.tsx` | 登録フォーム |
| Component | `src/components/auth/password-strength.tsx` | パスワード強度表示 |
| API | `src/lib/api/auth.ts` | register, verifyEmail API calls |

### Migration
```sql
-- users, email_verification_tokens, folders, folder_closures tables
-- See existing migrations
```

### Considerations
- **Security**: bcrypt cost 12, 確認トークン24時間有効、レート制限 10 req/min/IP
- **Performance**: メール送信はトランザクション外で非同期的に実行
- **Personal Folder**: 登録トランザクション内でFolder + FolderClosure を作成
