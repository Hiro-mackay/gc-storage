# Password Reset + Change

## Meta

| Item | Value |
|------|-------|
| Status | Ready |
| Priority | High |
| Tier | 1 (Auth) |
| Domain Refs | `03-domains/user.md` |
| Depends On | `features/auth-login.md` |

---

## 1. User Stories

**Primary:**
> As a user who forgot my password, I want to receive a password reset email so that I can regain access to my account.

**Secondary:**
> As a logged-in user, I want to change my password so that I can keep my account secure.

### Context
パスワードリセットはメールベースのトークン方式を採用。ユーザー列挙攻撃を防ぐため、リセット要求は常に成功メッセージを返す。パスワード変更は認証済みユーザーのみが実行可能で、現在のパスワード確認が必要。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-U001 | emailは全ユーザーで一意 | `03-domains/user.md` |
| R-U004 | status=suspended/deactivatedの場合ログイン不可 | `03-domains/user.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-PW-001 | リセットトークンの有効期限は1時間 |
| FS-PW-002 | リセット要求は常に成功メッセージを返す（列挙攻撃対策） |
| FS-PW-003 | pending状態のユーザーにはリセットメールを送らない |
| FS-PW-004 | OAuth専用ユーザーにはリセットメールを送らない |
| FS-PW-005 | リセットトークンは使用済みマーク方式で再利用防止 |
| FS-PW-006 | パスワード変更には現在のパスワード確認が必要 |
| FS-PW-007 | パスワードはbcrypt cost 12でハッシュ化 |
| FS-PW-008 | パスワードにメールアドレスを含めない |

### State Transitions
```
[Password Reset]
User --forgot--> ResetToken(active) --reset--> ResetToken(used) + Password updated

[Password Change]
User(active) --change--> Password updated
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/password/forgot` | No | パスワードリセット要求 |
| POST | `/api/v1/auth/password/reset` | No | パスワードリセット実行 |
| POST | `/api/v1/auth/password/change` | Required | パスワード変更 |

### Request / Response Details

#### `POST /api/v1/auth/password/forgot` - パスワードリセット要求

**Request Body:**
```json
{
  "email": "taro@example.com"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| email | string | Yes | 有効なメール形式 | メールアドレス |

**Success Response (200):**
```json
{
  "message": "If your email is registered, you will receive a password reset link."
}
```

**Note:** ユーザーが存在しない場合も同じレスポンスを返す（列挙攻撃対策）。

#### `POST /api/v1/auth/password/reset` - パスワードリセット実行

**Request Body:**
```json
{
  "token": "reset_token_string",
  "password": "NewSecurePass1"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| token | string | Yes | 必須 | リセットトークン |
| password | string | Yes | 8-256文字、大文字+数字含む | 新しいパスワード |

**Success Response (200):**
```json
{
  "message": "Password reset successfully"
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | 無効/期限切れ/使用済みトークン | `VALIDATION_ERROR` |
| 400 | パスワードバリデーションエラー | `VALIDATION_ERROR` |
| 404 | ユーザーが見つからない | `NOT_FOUND` |

#### `POST /api/v1/auth/password/change` - パスワード変更

**Request Body:**
```json
{
  "current_password": "OldSecurePass1",
  "new_password": "NewSecurePass2"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| current_password | string | Yes | 必須 | 現在のパスワード |
| new_password | string | Yes | 8-256文字、大文字+数字含む | 新しいパスワード |

**Success Response (200):**
```json
{
  "message": "Password changed successfully"
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | パスワードバリデーションエラー | `VALIDATION_ERROR` |
| 401 | 未認証 | `UNAUTHORIZED` |
| 401 | 現在のパスワード不正 | `UNAUTHORIZED` |

---

## 4. Frontend UI

### Layout / Wireframe
```
[Forgot Password Page]
+-------------------------------------------+
|       Forgot your password?               |
|  Enter your email address and we'll       |
|  send you a link to reset your password.  |
|  [Email         ]                         |
|  [Send reset link]                        |
|  <- Back to login                         |
+-------------------------------------------+

[Forgot Password - Success]
+-------------------------------------------+
|  Check your email                         |
|  If an account exists for {email},        |
|  we've sent a password reset link.        |
|  <- Back to login                         |
+-------------------------------------------+

[Reset Password Page]
+-------------------------------------------+
|       Reset your password                 |
|  Enter your new password below.           |
|  [New Password      ]                     |
|   * At least 8 characters                 |
|   * At least one uppercase letter         |
|   * At least one number                   |
|  [Confirm New Password]                   |
|  [Reset password]                         |
+-------------------------------------------+

[Reset Password - Success]
+-------------------------------------------+
|  Password reset successful                |
|  Your password has been reset.            |
|  You can now log in with your new         |
|  password.                                |
|  [Go to login]                            |
+-------------------------------------------+

[Change Password - in Settings]
+-------------------------------------------+
|  Change Password                          |
|  [Current Password  ]                     |
|  [New Password      ]                     |
|   * At least 8 characters                 |
|   * At least one uppercase letter         |
|   * At least one number                   |
|  [Confirm New Password]                   |
|  [Update password]                        |
+-------------------------------------------+
```

### Components

| Component | Type | Description |
|-----------|------|-------------|
| ForgotPasswordPage | Page | パスワードリセット要求ページ (`/auth/forgot-password`) |
| ResetPasswordPage | Page | パスワードリセット実行ページ (`/auth/reset-password`) |
| ChangePasswordForm | Form | パスワード変更フォーム (設定画面内) |
| PasswordStrengthIndicator | UI | パスワード強度リアルタイム表示 |

### State Management

| State | Store | Type | Description |
|-------|-------|------|-------------|
| forgotPasswordMutation | TanStack Query | Server | リセット要求API呼び出し |
| resetPasswordMutation | TanStack Query | Server | リセット実行API呼び出し |
| changePasswordMutation | TanStack Query | Server | パスワード変更API呼び出し |
| formState | useState | Local | フォーム入力値、バリデーション状態 |

### User Interactions
1. Forgot: メール入力 -> 送信 -> 確認メッセージ表示
2. Reset: メール内リンクをクリック -> 新パスワード入力 -> 送信 -> ログイン画面へ遷移
3. Change: 現在のパスワード + 新パスワード入力 -> 送信 -> 成功メッセージ

---

## 5. Integration Flow

### Password Reset Request Sequence
```
Client          Frontend        API             DB              Email
  |                |              |               |               |
  |-- submit ----->|              |               |               |
  |                |-- POST ----->|               |               |
  |                |  /forgot     |-- find user ->|               |
  |                |              |-- create token>|              |
  |                |              |-- send email ----|---------->|
  |                |<-- 200 ------|               |               |
  |<-- success ----|              |               |               |
```

### Password Reset Execution Sequence
```
Client          Frontend        API             DB
  |                |              |               |
  |-- submit ----->|              |               |
  |                |-- POST ----->|               |
  |                |  /reset      |-- find token ->|
  |                |              |-- check valid  |
  |                |              |-- find user -->|
  |                |              |-- validate pwd |
  |                |              |-- update pwd ->|
  |                |              |-- mark used -->|
  |                |<-- 200 ------|               |
  |<-- success ----|              |               |
```

### Password Change Sequence
```
Client          Frontend        API             DB
  |                |              |               |
  |-- submit ----->|              |               |
  |                |-- POST ----->|               |
  |                |  /change     |-- find user ->|
  |                |  (auth req)  |-- verify old  |
  |                |              |-- validate new|
  |                |              |-- update pwd ->|
  |                |<-- 200 ------|               |
  |<-- success ----|              |               |
```

### Error Handling Flow
- リセット要求: 常に200を返す（ユーザー列挙防止）
- 無効トークン: 400 -> 「無効なトークン。新しいリセットを要求してください」表示
- 期限切れトークン: 400 -> 「トークン期限切れ」表示
- 使用済みトークン: 400 -> 「トークン使用済み」表示
- 現在のパスワード不正: 401 -> エラー表示

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: Given 登録済みメール, when リセット要求送信, then 成功メッセージが返りメールが送信される
- [ ] AC-02: Given 有効なリセットトークン, when 新パスワード送信, then パスワードが更新される
- [ ] AC-03: Given リセット成功, when ログイン画面で新パスワード入力, then ログインできる
- [ ] AC-04: Given ログイン状態, when 正しい現在パスワードと新パスワード送信, then パスワードが変更される

### Validation Errors
- [ ] AC-10: Given 弱いパスワード, when リセット送信, then バリデーションエラーが返る
- [ ] AC-11: Given パスワード不一致, when リセット送信, then クライアント側エラーが表示される
- [ ] AC-12: Given 現在のパスワード不正, when パスワード変更送信, then 401エラーが返る

### Authorization
- [ ] AC-20: Given 未認証, when パスワード変更API呼び出し, then 401が返る

### Edge Cases
- [ ] AC-30: Given 存在しないメール, when リセット要求, then 同じ成功メッセージが返る（列挙防止）
- [ ] AC-31: Given 期限切れトークン, when リセット送信, then 「期限切れ」エラーが返る
- [ ] AC-32: Given 使用済みトークン, when リセット送信, then 「使用済み」エラーが返る
- [ ] AC-33: Given OAuth専用ユーザー, when リセット要求, then メールは送信されない（成功メッセージは返る）
- [ ] AC-34: Given pendingユーザー, when リセット要求, then メールは送信されない（成功メッセージは返る）

---

## 7. Test Plan

### Backend Unit Tests

| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| リセット要求（存在するユーザー） | ForgotPasswordCommand | トークン作成、メール送信 |
| リセット要求（存在しないメール） | ForgotPasswordCommand | エラーなく成功メッセージ返却 |
| リセット要求（OAuth専用） | ForgotPasswordCommand | メール送信なし、成功メッセージ返却 |
| リセット要求（pending） | ForgotPasswordCommand | メール送信なし、成功メッセージ返却 |
| リセット実行（正常） | ResetPasswordCommand | パスワード更新、トークン使用済み |
| リセット実行（期限切れ） | ResetPasswordCommand | ValidationError |
| リセット実行（使用済み） | ResetPasswordCommand | ValidationError |
| パスワード変更（正常） | ChangePasswordCommand | パスワード更新 |
| パスワード変更（現在パスワード不正） | ChangePasswordCommand | UnauthorizedError |

### Backend Integration Tests

| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| リセット要求 | POST /password/forgot | active user | 200 |
| リセット実行 | POST /password/reset | active token | 200 |
| 無効トークン | POST /password/reset | expired token | 400 |
| パスワード変更 | POST /password/change | auth session | 200 |
| 未認証パスワード変更 | POST /password/change | no session | 401 |

### Frontend Tests

| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| リセット要求フォーム | ForgotPasswordPage | Unit | メール入力、送信、成功表示 |
| リセット実行フォーム | ResetPasswordPage | Unit | パスワード入力、強度表示、送信 |
| パスワード変更フォーム | ChangePasswordForm | Unit | 3フィールド入力、バリデーション |
| 無効トークンエラー | ResetPasswordPage | Integration | エラーメッセージ表示 |

### E2E Tests (future)

| Test | Flow | Assertions |
|------|------|------------|
| リセットフロー | forgot -> email -> reset -> login | 新パスワードでログイン |
| パスワード変更フロー | settings -> change -> re-login | 新パスワードでログイン |

---

## 8. Implementation Notes

### Changed Files (Backend)

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/password_reset_token.go` | PasswordResetToken entity |
| Domain | `internal/domain/repository/password_reset_token_repository.go` | Repository IF |
| Domain | `internal/domain/service/password_reset_service.go` | PasswordResetService IF |
| UseCase | `internal/usecase/auth/command/forgot_password.go` | ForgotPasswordCommand |
| UseCase | `internal/usecase/auth/command/reset_password.go` | ResetPasswordCommand |
| UseCase | `internal/usecase/auth/command/change_password.go` | ChangePasswordCommand |
| Interface | `internal/interface/handler/auth_handler.go` | ForgotPassword, ResetPassword, ChangePassword |
| Interface | `internal/interface/dto/request/auth.go` | Request DTOs |
| Interface | `internal/interface/dto/response/auth.go` | Response DTOs |
| Infra | `internal/infrastructure/database/password_reset_token_repository.go` | Repo impl |

### Changed Files (Frontend)

| Category | File | Change |
|----------|------|--------|
| Route | `src/app/routes/auth/forgot-password.tsx` | リセット要求ページ |
| Route | `src/app/routes/auth/reset-password.tsx` | リセット実行ページ |
| Component | `src/features/settings/change-password-form.tsx` | パスワード変更フォーム |
| Component | `src/components/auth/password-strength.tsx` | パスワード強度表示（共有） |
| API | `src/lib/api/auth.ts` | forgotPassword, resetPassword, changePassword |

### Migration
```sql
-- password_reset_tokens table
-- See existing migrations
```

### Considerations
- **Security**: リセットトークン1時間有効、列挙攻撃対策、bcrypt cost 12
- **Security**: リセット完了後にトークンを使用済みマーク（再利用防止）
- **UX**: 存在しないメールでも成功メッセージ表示（セキュリティ優先）
