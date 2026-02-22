# User ドメイン

## 概要

Userドメインは、GC Storageにおけるユーザーの登録、認証、プロファイル管理を担当します。
Identity Contextの中核となるドメインで、他のすべてのコンテキストがUserIDを参照します。

---

## エンティティ

### User（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | ユーザーの一意識別子 |
| email | Email | Yes | メールアドレス（一意） |
| name | string | Yes | 表示名 (1-100文字) |
| password_hash | string | No | bcryptハッシュ化されたパスワード |
| personal_folder_id | UUID | No | Personal FolderのID（1:1関係、登録後に設定） |
| status | UserStatus | Yes | アカウント状態 |
| email_verified | boolean | Yes | メール確認完了フラグ |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ビジネスルール:**
- R-U001: emailは全ユーザーで一意でなければならない
- R-U002: password_hashはOAuth専用ユーザーの場合のみNULL許容
- R-U003: nameは空文字不可、1-100文字
- R-U004: statusがsuspendedまたはdeactivatedの場合、ログイン不可。pendingはログイン可能（即座にアプリ利用開始できるようにする）
- R-U005: email_verifiedがfalseの場合、重要操作（共有、チーム招待等）に制限。基本操作（ファイル閲覧・アップロード等）は許可
- R-U006: personal_folder_idはユーザー登録処理完了後に設定（登録時に自動作成）
- R-U007: UserとPersonal Folderは1対1の関係

**ステータス遷移:**
```
                    ┌─────────┐
        ┌──────────▶│ active  │◀──────────┐
        │           └────┬────┘           │
        │                │                │
        │                ▼                │
   ┌────┴────┐     ┌──────────┐     ┌────┴────┐
   │ pending │     │ suspended │     │ verified │
   └─────────┘     └────┬─────┘     └──────────┘
                        │
                        ▼
                  ┌───────────┐
                  │deactivated│
                  └───────────┘
```

| ステータス | 説明 |
|-----------|------|
| pending | メール確認待ち |
| active | 通常利用可能 |
| suspended | 管理者による一時停止 |
| deactivated | アカウント無効化（ユーザー要求） |

### OAuthAccount

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 一意識別子 |
| user_id | UUID | Yes | 紐付くユーザーID |
| provider | OAuthProvider | Yes | プロバイダー種別 |
| provider_user_id | string | Yes | プロバイダー側のユーザーID |
| email | string | Yes | プロバイダーから取得したメールアドレス |
| access_token | string | No | アクセストークン（暗号化保存） |
| refresh_token | string | No | リフレッシュトークン（暗号化保存） |
| token_expires_at | timestamp | No | トークン有効期限 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ビジネスルール:**
- R-OA001: 同一userに対して同一providerのアカウントは1つのみ
- R-OA002: provider + provider_user_idの組み合わせは一意
- R-OA003: access_token, refresh_tokenは暗号化して保存

### Session

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | string | Yes | セッションID（UUID文字列） |
| user_id | UUID | Yes | ユーザーID |
| refresh_token | string | Yes | リフレッシュトークン |
| user_agent | string | No | クライアントのUser-Agent |
| ip_address | string | No | 接続元IPアドレス |
| expires_at | timestamp | Yes | 有効期限 |
| created_at | timestamp | Yes | 作成日時 |
| last_used_at | timestamp | Yes | 最終使用日時 |

**ビジネスルール:**
- R-S001: expires_atを過ぎたセッションは無効（Redis TTLにより自動削除）
- R-S002: 同一ユーザーの有効セッションは最大10個まで
- R-S003: 新規セッション作成時、最古のセッションを自動失効

**注記:** セッションはRedisに保存され、expires_at到達時にTTLにより自動削除されます。

### UserProfile

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 一意識別子 |
| user_id | UUID | Yes | ユーザーID（FK） |
| avatar_url | string | No | アバター画像URL |
| bio | string | No | 自己紹介 (最大500文字) |
| locale | string | Yes | 言語設定 (default: ja) |
| timezone | string | Yes | タイムゾーン (default: Asia/Tokyo) |
| theme | string | Yes | テーマ設定 (system/light/dark, default: system) |
| notification_preferences | jsonb | Yes | 通知設定 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**NotificationPreferences構造:**
```json
{
  "email_enabled": true,
  "push_enabled": true
}
```

**ビジネスルール:**
- R-UP001: bioは最大500文字
- R-UP002: avatar_urlは有効なURL形式
- R-UP003: themeは"system", "light", "dark"のいずれか

**注記:** ユーザーの表示名(display_name)はusersテーブルのname列で管理されます。

---

## 値オブジェクト

### Email

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | メールアドレス文字列 |

**バリデーション:**
- RFC 5322に準拠した形式
- 最大255文字
- 小文字正規化

```go
type Email struct {
    value string
}

func NewEmail(value string) (Email, error) {
    normalized := strings.ToLower(strings.TrimSpace(value))
    if len(normalized) > 255 {
        return Email{}, errors.New("email must not exceed 255 characters")
    }
    if !isValidEmailFormat(normalized) {
        return Email{}, errors.New("invalid email format")
    }
    return Email{value: normalized}, nil
}
```

### Password

| 属性 | 型 | 説明 |
|-----|-----|------|
| hash | string | bcryptハッシュ値 |

**バリデーション（平文パスワード）:**
- 最小8文字、最大256文字
- 英大文字、英小文字、数字のうち2種以上を含む
- メールアドレスを含まない
- 一般的なパスワードリストに含まれない

```go
type Password struct {
    hash string
}

func NewPassword(plaintext string, email string) (Password, error) {
    if err := validatePasswordPolicy(plaintext, email); err != nil {
        return Password{}, err
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
    if err != nil {
        return Password{}, err
    }
    return Password{hash: string(hash)}, nil
}

func (p Password) Verify(plaintext string) bool {
    return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(plaintext)) == nil
}
```

### OAuthProvider

| 値 | 説明 |
|-----|------|
| google | Google OAuth 2.0 |
| github | GitHub OAuth |

### UserStatus

| 値 | 説明 |
|-----|------|
| pending | メール確認待ち |
| active | アクティブ |
| suspended | 停止中 |
| deactivated | 無効化済み |

---

## ドメインサービス

### AuthenticationService

**責務:** ユーザー認証の検証とセッション管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| AuthenticateWithCredentials | email, password | (User, Session) | メール/パスワード認証 |
| AuthenticateWithOAuth | provider, code | (User, Session) | OAuth認証 |
| RefreshSession | refreshToken | (accessToken, refreshToken) | セッション更新 |
| RevokeSession | sessionId | void | セッション失効 |
| RevokeAllSessions | userId | void | 全セッション失効 |

```go
type AuthenticationService interface {
    AuthenticateWithCredentials(ctx context.Context, email, password string) (*User, *Session, error)
    AuthenticateWithOAuth(ctx context.Context, provider OAuthProvider, code string) (*User, *Session, error)
    RefreshSession(ctx context.Context, refreshToken string) (string, string, error)
    RevokeSession(ctx context.Context, sessionID uuid.UUID) error
    RevokeAllSessions(ctx context.Context, userID uuid.UUID) error
}
```

### PasswordResetService

**責務:** パスワードリセットフローの管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| RequestReset | email | void | リセットトークン発行・メール送信 |
| ValidateToken | token | User | トークン検証 |
| ResetPassword | token, newPassword | void | パスワード更新 |

```go
type PasswordResetService interface {
    RequestReset(ctx context.Context, email string) error
    ValidateToken(ctx context.Context, token string) (*User, error)
    ResetPassword(ctx context.Context, token string, newPassword string) error
}
```

### EmailVerificationService

**責務:** メールアドレス確認フローの管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| SendVerification | userId | void | 確認メール送信 |
| Verify | token | void | メール確認完了 |

```go
type EmailVerificationService interface {
    SendVerification(ctx context.Context, userID uuid.UUID) error
    Verify(ctx context.Context, token string) error
}
```

---

## リポジトリ

### UserRepository

```go
type UserRepository interface {
    // 基本CRUD
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id uuid.UUID) (*User, error)
    FindByEmail(ctx context.Context, email Email) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id uuid.UUID) error

    // 検索
    Exists(ctx context.Context, email Email) (bool, error)
}
```

**注記:** OAuth経由のユーザー検索はOAuthAccountRepositoryのFindByProviderAndUserIDを使用します。

### SessionRepository

```go
type SessionRepository interface {
    // Save はセッションを保存します
    Save(ctx context.Context, session *Session) error

    // FindByID はIDでセッションを検索します
    FindByID(ctx context.Context, sessionID string) (*Session, error)

    // FindByUserID はユーザーIDでセッション一覧を取得します
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error)

    // Delete はセッションを削除します
    Delete(ctx context.Context, sessionID string) error

    // DeleteByUserID はユーザーの全セッションを削除します
    DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
```

**注記:** セッションはRedisに保存されるため、expiredセッションはTTLにより自動削除されます。DeleteExpiredメソッドは不要です。

### OAuthAccountRepository

```go
type OAuthAccountRepository interface {
    Create(ctx context.Context, account *OAuthAccount) error
    FindByProviderAndUserID(ctx context.Context, provider OAuthProvider, providerUserID string) (*OAuthAccount, error)
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]*OAuthAccount, error)
    Update(ctx context.Context, account *OAuthAccount) error
    Delete(ctx context.Context, userID uuid.UUID, provider OAuthProvider) error
}
```

---

## 関係性

### エンティティ関係図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          User Domain ERD                                     │
└─────────────────────────────────────────────────────────────────────────────┘

                           ┌───────────────────┐
                           │       users       │
                           ├───────────────────┤
                           │ id                │
                           │ email             │
                           │ name              │
                           │ password_hash     │
                           │ personal_folder_id│──────┐
                           │ status            │      │
                           │ email_verified    │      │
                           │ created_at        │      │
                           │ updated_at        │      │
                           └──────┬────────────┘      │
                                  │                   │
              ┌───────────────────┼───────────────────┤
              │                   │                   │
              ▼                   ▼                   ▼
    ┌──────────────────┐ ┌────────────────┐ ┌──────────────────┐
    │  oauth_accounts  │ │    sessions    │ │  user_profiles   │
    ├──────────────────┤ │   (Redis)      │ ├──────────────────┤
    │ id               │ ├────────────────┤ │ id (PK)          │
    │ user_id (FK)     │ │ id             │ │ user_id (FK)     │
    │ provider         │ │ user_id (FK)   │ │ avatar_url       │
    │ provider_user_id │ │ refresh_token  │ │ bio              │
    │ email            │ │ user_agent     │ │ locale           │
    │ access_token     │ │ ip_address     │ │ timezone         │
    │ refresh_token    │ │ expires_at     │ │ theme            │
    │ token_expires_at │ │ created_at     │ │ notification_... │
    │ created_at       │ │ last_used_at   │ │ created_at       │
    │ updated_at       │ └────────────────┘ │ updated_at       │
    └──────────────────┘                    └──────────────────┘

                           ┌──────────────────┐
                           │     folders      │ (Storage Context)
                           │  (external 1:1)  │
                           ├──────────────────┤
                           │ id               │◀─────── personal_folder_id
                           │ name             │
                           │ ...              │
                           └──────────────────┘
```

### 関係性ルール

| 関係 | カーディナリティ | 説明 |
|-----|----------------|------|
| User - OAuthAccount | 1:N | 1ユーザーは複数のOAuthアカウントを持てる（各プロバイダー1つずつ） |
| User - Session | 1:N | 1ユーザーは複数のアクティブセッションを持てる（最大10） |
| User - UserProfile | 1:1 | 1ユーザーに1つのプロファイル |
| User - Personal Folder | 1:1 | 1ユーザーに1つのPersonal Folder（ユーザー登録時に自動作成） |

---

## 不変条件

1. **一意性制約**
   - emailは全ユーザーで一意
   - (provider, provider_user_id)の組み合わせは一意
   - 同一ユーザー・同一プロバイダーのOAuthAccountは1つのみ

2. **認証要件**
   - ユーザーはpassword_hashまたはOAuthAccountの少なくとも1つを持つ
   - OAuth専用ユーザーはpassword_hashがNULL

3. **セッション制約**
   - アクティブセッションは最大10個
   - expires_at到達後は自動無効

4. **状態整合性**
   - statusがpendingの間はemail_verifiedはfalse
   - statusがdeactivatedになったらすべてのセッションを失効

5. **Personal Folder制約**
   - UserとPersonal Folderは1対1の関係
   - ユーザー登録処理でPersonal Folderを自動作成
   - personal_folder_idは登録処理完了後に設定（nullable）
   - Personal Folderの存在判定は`HasPersonalFolder()`メソッドで行う

---

## ユースケース概要

| ユースケース | アクター | 概要 |
|------------|--------|------|
| RegisterUser | Guest | 新規ユーザー登録 |
| VerifyEmail | Guest | メールアドレス確認 |
| LoginWithCredentials | Guest | メール/パスワードでログイン |
| LoginWithOAuth | Guest | OAuth認証でログイン |
| Logout | User | ログアウト |
| RefreshToken | User | アクセストークン更新 |
| ChangePassword | User | パスワード変更 |
| RequestPasswordReset | Guest | パスワードリセット要求 |
| ResetPassword | Guest | パスワードリセット実行 |
| UpdateProfile | User | プロファイル更新 |
| LinkOAuthAccount | User | OAuthアカウント連携 |
| UnlinkOAuthAccount | User | OAuthアカウント解除 |
| DeactivateAccount | User | アカウント無効化 |
| ListActiveSessions | User | アクティブセッション一覧 |
| RevokeSession | User | 特定セッション失効 |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| UserRegistered | ユーザー登録完了 | userId, email |
| UserLoggedIn | ログイン成功 | userId, sessionId, loginMethod |
| UserLoggedOut | ログアウト | userId, sessionId |
| EmailVerified | メール確認完了 | userId |
| PasswordChanged | パスワード変更 | userId |
| PasswordResetRequested | リセット要求 | email |
| PasswordReset | リセット完了 | userId |
| OAuthAccountLinked | OAuth連携追加 | userId, provider |
| OAuthAccountUnlinked | OAuth連携解除 | userId, provider |
| ProfileUpdated | プロファイル更新 | userId, changedFields |
| UserDeactivated | アカウント無効化 | userId |
| SessionRevoked | セッション失効 | sessionId |

---

## 他コンテキストとの連携

### Storage Context（下流）

- ユーザー登録時にPersonal Folderを自動作成
- personal_folder_idでPersonal Folderを参照
- ユーザー削除時にPersonal Folderも削除

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [セキュリティ設計](../02-architecture/SECURITY.md) - 認証・認可の詳細設計
- [API設計](../02-architecture/API.md) - エンドポイント定義
- [フォルダドメイン](./folder.md) - Personal Folder連携
