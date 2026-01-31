# 認証・アイデンティティ仕様書

## 概要

本ドキュメントでは、GC Storageにおけるユーザー登録、認証、セッション管理、OAuth連携の実装仕様を定義します。

**関連アーキテクチャ:**
- [SECURITY.md](../02-architecture/SECURITY.md) - セキュリティ設計
- [user.md](../03-domains/user.md) - ユーザードメイン定義
- [infra-redis.md](./infra-redis.md) - セッションストア

---

## 1. セッションベース認証

### 1.1 認証方式の概要

本システムでは**Session IDベースの認証**を採用します。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Session-Based Authentication                        │
└─────────────────────────────────────────────────────────────────────────────┘

┌────────┐          ┌──────────┐          ┌────────┐
│ Client │          │   API    │          │ Redis  │
└────┬───┘          └────┬─────┘          └───┬────┘
     │                   │                    │
     │  POST /login      │                    │
     │──────────────────▶│                    │
     │                   │                    │
     │                   │  Save Session      │
     │                   │───────────────────▶│
     │                   │                    │
     │  Set-Cookie:      │                    │
     │  session_id=xxx   │                    │
     │◀──────────────────│                    │
     │                   │                    │
     │  GET /api/xxx     │                    │
     │  Cookie: session_id=xxx               │
     │──────────────────▶│                    │
     │                   │  Validate Session  │
     │                   │───────────────────▶│
     │                   │                    │
     │  Response         │                    │
     │◀──────────────────│                    │
```

### 1.2 セッション仕様

| 項目 | 値 | 説明 |
|------|-----|------|
| セッションID | UUID v4 | 暗号学的に安全なランダムID |
| 有効期限 | 7日 | スライディングウィンドウで延長 |
| 保存場所（サーバー） | Redis | TTL付きで自動期限切れ |
| 保存場所（クライアント） | HttpOnly Cookie | フロントエンドからアクセス不可 |
| 最大セッション数 | 10 | 11個目作成時に最古を自動削除 |

**重要**: Session IDはHttpOnly Cookieで管理し、フロントエンドからはアクセス不可とする。Cookieが認証の唯一の真の情報源（Single Source of Truth）となる。

### 1.3 セッションデータ構造

```go
// internal/domain/entity/session.go

package entity

import (
    "time"

    "github.com/google/uuid"
)

const (
    // MaxActiveSessionsPerUser は1ユーザーあたりの最大アクティブセッション数
    MaxActiveSessionsPerUser = 10
    // SessionTTL はセッションのデフォルト有効期限
    SessionTTL = 7 * 24 * time.Hour
)

// Session はユーザーセッションを表します
type Session struct {
    ID         string    // セッションID (UUID)
    UserID     uuid.UUID // ユーザーID
    UserAgent  string    // ブラウザ/クライアント情報
    IPAddress  string    // IPアドレス
    ExpiresAt  time.Time // 有効期限
    CreatedAt  time.Time // 作成日時
    LastUsedAt time.Time // 最終使用日時
}

// IsExpired はセッションが期限切れかどうかを返します
func (s *Session) IsExpired() bool {
    return time.Now().After(s.ExpiresAt)
}

// Refresh はセッションの有効期限を延長します（スライディングウィンドウ）
func (s *Session) Refresh() {
    s.LastUsedAt = time.Now()
    s.ExpiresAt = time.Now().Add(SessionTTL)
}
```

### 1.4 Cookie設定

```go
// Cookie設定
&http.Cookie{
    Name:     "session_id",
    Value:    sessionID,
    Path:     "/",           // すべてのパスで送信
    HttpOnly: true,          // JavaScriptからアクセス不可
    Secure:   true,          // HTTPSのみ
    SameSite: http.SameSiteLaxMode, // CSRF対策（GETは許可）
    MaxAge:   7 * 24 * 60 * 60,     // 7日間
}
```

**SameSite設定の理由:**
- `Lax`: OAuth認証のリダイレクト時にCookieが送信されるようにするため
- `Strict`だとOAuthコールバック時にCookieが送信されない

---

## 2. ユーザー登録

### 2.1 登録フロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          User Registration Flow                              │
│                    (Personal Folder 自動作成を含む)                           │
└─────────────────────────────────────────────────────────────────────────────┘

┌────────┐          ┌──────────┐          ┌──────────┐          ┌───────┐
│ Client │          │   API    │          │   DB     │          │ Email │
└────┬───┘          └────┬─────┘          └────┬─────┘          └───┬───┘
     │                   │                     │                    │
     │  1. POST /register                      │                    │
     │  {email, password, name}                │                    │
     │──────────────────▶│                     │                    │
     │                   │                     │                    │
     │                   │  2. Check email exists                   │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  3. Create Personal Folder (1:1)         │
     │                   │     (name: ユーザー名)                    │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  4. Create Folder Closure (self-ref)     │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  5. Create User (status: pending)        │
     │                   │     (personal_folder_id = folder.id)     │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  6. Create verification token            │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  7. Send verification email              │
     │                   │─────────────────────────────────────────▶│
     │                   │                     │                    │
     │  8. 201 Created   │                     │                    │
     │  {user_id, message: "Please verify..."}                      │
     │◀──────────────────│                     │                    │
```

**Personal Folder 作成ルール:**
- ユーザー登録時に必ずPersonal Folderを自動作成（1:1関係）
- フォルダ名はユーザー名を初期値とし、後からユーザーが自由に変更可能
- `user.personal_folder_id` でPersonal Folderを判定
- Personal Folderは削除不可（アカウント削除時のみ削除）

### 2.2 登録コマンド（CQRS）

```go
// internal/usecase/auth/command/register.go

package command

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "log/slog"
    "time"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RegisterInput は登録の入力を定義します
type RegisterInput struct {
    Email    string
    Password string
    Name     string
}

// RegisterOutput は登録の出力を定義します
type RegisterOutput struct {
    UserID uuid.UUID
}

// RegisterCommand はユーザー登録コマンドです
type RegisterCommand struct {
    userRepo                   repository.UserRepository
    emailVerificationTokenRepo repository.EmailVerificationTokenRepository
    folderRepo                 repository.FolderRepository
    folderClosureRepo          repository.FolderClosureRepository
    txManager                  repository.TransactionManager
    emailSender                service.EmailSender
    appURL                     string
}

// NewRegisterCommand は新しいRegisterCommandを作成します
func NewRegisterCommand(
    userRepo repository.UserRepository,
    emailVerificationTokenRepo repository.EmailVerificationTokenRepository,
    folderRepo repository.FolderRepository,
    folderClosureRepo repository.FolderClosureRepository,
    txManager repository.TransactionManager,
    emailSender service.EmailSender,
    appURL string,
) *RegisterCommand {
    return &RegisterCommand{
        userRepo:                   userRepo,
        emailVerificationTokenRepo: emailVerificationTokenRepo,
        folderRepo:                 folderRepo,
        folderClosureRepo:          folderClosureRepo,
        txManager:                  txManager,
        emailSender:                emailSender,
        appURL:                     appURL,
    }
}

// Execute はユーザー登録を実行します
func (c *RegisterCommand) Execute(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
    // 1. メールアドレスのバリデーション
    email, err := valueobject.NewEmail(input.Email)
    if err != nil {
        return nil, apperror.NewValidationError(err.Error(), nil)
    }

    // 2. パスワードのバリデーション
    password, err := valueobject.NewPassword(input.Password, input.Email)
    if err != nil {
        return nil, apperror.NewValidationError(err.Error(), nil)
    }

    // 3. メールアドレスの重複チェック
    exists, err := c.userRepo.Exists(ctx, email)
    if err != nil {
        return nil, apperror.NewInternalError(err)
    }
    if exists {
        return nil, apperror.NewConflictError("email already exists")
    }

    var user *entity.User
    var verificationToken *entity.EmailVerificationToken

    // 4. トランザクションでユーザー作成
    err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        now := time.Now()
        userID := uuid.New()
        personalFolderID := uuid.New()

        // Personal Folder を作成（ユーザーと1:1の関係）
        folderName, _ := valueobject.NewFolderName(input.Name) // 初期名はユーザー名
        personalFolder := &entity.Folder{
            ID:        personalFolderID,
            Name:      folderName,
            ParentID:  nil,        // ルートレベル
            OwnerID:   userID,     // 作成者がオーナー
            CreatedBy: userID,     // 作成者
            Depth:     0,
            Status:    valueobject.FolderStatusActive,
            CreatedAt: now,
            UpdatedAt: now,
        }

        if err := c.folderRepo.Create(ctx, personalFolder); err != nil {
            return fmt.Errorf("failed to create personal folder: %w", err)
        }

        // フォルダのClosure Table エントリを作成（自己参照）
        if err := c.folderClosureRepo.CreateSelfReference(ctx, personalFolderID); err != nil {
            return fmt.Errorf("failed to create folder closure: %w", err)
        }

        // ユーザー作成
        user = &entity.User{
            ID:            userID,
            Email:         email,
            Name:          input.Name,
            PasswordHash:  password.Hash(),
            Status:        entity.UserStatusPending,
            EmailVerified: false,
            CreatedAt:     now,
            UpdatedAt:     now,
        }
        // Personal Folder への参照を設定（ポインタ型）
        user.SetPersonalFolder(personalFolderID)

        if err := c.userRepo.Create(ctx, user); err != nil {
            return err
        }

        // 確認トークン作成
        if c.emailVerificationTokenRepo != nil {
            verificationToken = &entity.EmailVerificationToken{
                ID:        uuid.New(),
                UserID:    user.ID,
                Token:     generateSecureToken(),
                ExpiresAt: now.Add(24 * time.Hour),
                CreatedAt: now,
            }

            if err := c.emailVerificationTokenRepo.Create(ctx, verificationToken); err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    // 5. 確認メール送信（トランザクション外で実行、失敗しても登録は成功扱い）
    if c.emailSender != nil && verificationToken != nil {
        verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", c.appURL, verificationToken.Token)
        if err := c.emailSender.SendEmailVerification(ctx, user.Email.String(), user.Name, verifyURL); err != nil {
            slog.Error("failed to send verification email", "error", err, "user_id", user.ID)
        }
    }

    return &RegisterOutput{UserID: user.ID}, nil
}

// generateSecureToken はセキュアなトークンを生成します
func generateSecureToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}
```

---

## 3. メール確認

### 3.1 確認フロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Email Verification Flow                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌────────┐          ┌──────────┐          ┌──────────┐
│ Client │          │   API    │          │   DB     │
└────┬───┘          └────┬─────┘          └────┬─────┘
     │                   │                     │
     │  1. GET /verify?token=xxx               │
     │──────────────────▶│                     │
     │                   │                     │
     │                   │  2. Find token      │
     │                   │────────────────────▶│
     │                   │                     │
     │                   │  3. Check expiry    │
     │                   │                     │
     │                   │  4. Update user     │
     │                   │  (email_verified=true, status=active)
     │                   │────────────────────▶│
     │                   │                     │
     │                   │  5. Delete token    │
     │                   │────────────────────▶│
     │                   │                     │
     │  6. 200 OK        │                     │
     │◀──────────────────│                     │
```

### 3.2 確認コマンド（CQRS）

```go
// internal/usecase/auth/command/verify_email.go

package command

import (
    "context"
    "time"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// VerifyEmailInput はメール確認の入力を定義します
type VerifyEmailInput struct {
    Token string
}

// VerifyEmailOutput はメール確認の出力を定義します
type VerifyEmailOutput struct {
    Message string
}

// VerifyEmailCommand はメール確認コマンドです
type VerifyEmailCommand struct {
    userRepo                   repository.UserRepository
    emailVerificationTokenRepo repository.EmailVerificationTokenRepository
    txManager                  repository.TransactionManager
}

// NewVerifyEmailCommand は新しいVerifyEmailCommandを作成します
func NewVerifyEmailCommand(
    userRepo repository.UserRepository,
    emailVerificationTokenRepo repository.EmailVerificationTokenRepository,
    txManager repository.TransactionManager,
) *VerifyEmailCommand {
    return &VerifyEmailCommand{
        userRepo:                   userRepo,
        emailVerificationTokenRepo: emailVerificationTokenRepo,
        txManager:                  txManager,
    }
}

// Execute はメール確認を実行します
func (c *VerifyEmailCommand) Execute(ctx context.Context, input VerifyEmailInput) (*VerifyEmailOutput, error) {
    // 1. トークンを検索
    verificationToken, err := c.emailVerificationTokenRepo.FindByToken(ctx, input.Token)
    if err != nil {
        return nil, apperror.NewValidationError("invalid or expired verification token", nil)
    }

    // 2. 有効期限チェック
    if verificationToken.IsExpired() {
        return nil, apperror.NewValidationError("verification token expired", nil)
    }

    // 3. ユーザーを取得
    user, err := c.userRepo.FindByID(ctx, verificationToken.UserID)
    if err != nil {
        return nil, apperror.NewNotFoundError("user")
    }

    // 4. すでに確認済みの場合
    if user.EmailVerified {
        return &VerifyEmailOutput{Message: "Email already verified"}, nil
    }

    // 5. トランザクションで更新
    err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        user.EmailVerified = true
        user.Status = entity.UserStatusActive
        user.UpdatedAt = time.Now()

        if err := c.userRepo.Update(ctx, user); err != nil {
            return err
        }

        // トークン削除
        return c.emailVerificationTokenRepo.Delete(ctx, verificationToken.ID)
    })

    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &VerifyEmailOutput{Message: "Email verified successfully"}, nil
}
```

---

## 4. ログイン

### 4.1 パスワードログインフロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Password Login Flow                                 │
└─────────────────────────────────────────────────────────────────────────────┘

┌────────┐          ┌──────────┐          ┌──────────┐          ┌────────┐
│ Client │          │   API    │          │   DB     │          │ Redis  │
└────┬───┘          └────┬─────┘          └────┬─────┘          └───┬────┘
     │                   │                     │                    │
     │  1. POST /login   │                     │                    │
     │  {email, password}│                     │                    │
     │──────────────────▶│                     │                    │
     │                   │                     │                    │
     │                   │  2. Find user       │                    │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  3. Verify password │                    │
     │                   │  (bcrypt)           │                    │
     │                   │                     │                    │
     │                   │  4. Check user status                    │
     │                   │  (must be active)   │                    │
     │                   │                     │                    │
     │                   │  5. Enforce session limit (max 10)       │
     │                   │  (delete oldest if needed)               │
     │                   │─────────────────────────────────────────▶│
     │                   │                     │                    │
     │                   │  6. Create session  │                    │
     │                   │─────────────────────────────────────────▶│
     │                   │                     │                    │
     │  7. Set-Cookie: session_id=xxx          │                    │
     │  + User info in body                    │                    │
     │◀──────────────────│                     │                    │
```

### 4.2 ログインコマンド（CQRS）

```go
// internal/usecase/auth/command/login.go

package command

import (
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// LoginInput はログインの入力を定義します
type LoginInput struct {
    Email     string
    Password  string
    UserAgent string
    IPAddress string
}

// LoginOutput はログインの出力を定義します
type LoginOutput struct {
    SessionID string
    User      *entity.User
}

// LoginCommand はログインコマンドです
type LoginCommand struct {
    userRepo    repository.UserRepository
    sessionRepo repository.SessionRepository
}

// NewLoginCommand は新しいLoginCommandを作成します
func NewLoginCommand(
    userRepo repository.UserRepository,
    sessionRepo repository.SessionRepository,
) *LoginCommand {
    return &LoginCommand{
        userRepo:    userRepo,
        sessionRepo: sessionRepo,
    }
}

// Execute はログインを実行します
func (c *LoginCommand) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
    // 1. メールアドレスでユーザーを検索
    email, err := valueobject.NewEmail(input.Email)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("invalid credentials")
    }

    user, err := c.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("invalid credentials")
    }

    // 2. パスワード検証
    if user.PasswordHash == "" {
        // OAuth専用ユーザー
        return nil, apperror.NewUnauthorizedError("please use OAuth to login")
    }

    password := valueobject.PasswordFromHash(user.PasswordHash)
    if !password.Verify(input.Password) {
        return nil, apperror.NewUnauthorizedError("invalid credentials")
    }

    // 3. ユーザー状態チェック
    if user.Status != entity.UserStatusActive {
        switch user.Status {
        case entity.UserStatusPending:
            return nil, apperror.NewUnauthorizedError("please verify your email first")
        case entity.UserStatusSuspended:
            return nil, apperror.NewUnauthorizedError("account suspended")
        case entity.UserStatusDeactivated:
            return nil, apperror.NewUnauthorizedError("account deactivated")
        }
    }

    // 4. セッション上限チェック（最大10セッション）
    sessions, err := c.sessionRepo.FindAllByUserID(ctx, user.ID)
    if err == nil && len(sessions) >= entity.MaxActiveSessionsPerUser {
        // 最古のセッションを削除
        if err := c.sessionRepo.DeleteOldest(ctx, user.ID); err != nil {
            return nil, apperror.NewInternalError(err)
        }
    }

    // 5. セッション作成（Redisに保存）
    sessionID := uuid.New().String()
    now := time.Now()

    session := &entity.Session{
        ID:         sessionID,
        UserID:     user.ID,
        UserAgent:  input.UserAgent,
        IPAddress:  input.IPAddress,
        ExpiresAt:  now.Add(entity.SessionTTL),
        CreatedAt:  now,
        LastUsedAt: now,
    }

    if err := c.sessionRepo.Save(ctx, session); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &LoginOutput{
        SessionID: sessionID,
        User:      user,
    }, nil
}
```

---

## 5. OAuth認証

### 5.1 OAuthフロー

フロントエンドがOAuthプロバイダーへのリダイレクトとコールバックを処理し、取得した認可コードをバックエンドAPIに送信するフローを採用しています。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    OAuth 2.0 Authorization Code Flow (SPA)                   │
└─────────────────────────────────────────────────────────────────────────────┘

┌────────────┐   ┌───────────────┐   ┌──────────┐   ┌──────────┐   ┌────────┐
│  Frontend  │   │ OAuth Provider│   │   API    │   │    DB    │   │ Redis  │
└─────┬──────┘   │(Google/GitHub)│   └────┬─────┘   └────┬─────┘   └───┬────┘
      │          └───────┬───────┘        │              │             │
      │                  │                │              │             │
      │  1. Redirect to OAuth provider    │              │             │
      │─────────────────▶│                │              │             │
      │                  │                │              │             │
      │  2. User authorizes               │              │             │
      │  3. Redirect with code            │              │             │
      │◀─────────────────│                │              │             │
      │                  │                │              │             │
      │  4. POST /api/v1/auth/oauth/:provider             │             │
      │  {code}          │                │              │             │
      │──────────────────────────────────▶│              │             │
      │                  │                │              │             │
      │                  │  5. Exchange code              │             │
      │                  │◀───────────────│              │             │
      │                  │  6. Access token               │             │
      │                  │───────────────▶│              │             │
      │                  │                │              │             │
      │                  │  7. Get user info              │             │
      │                  │◀───────────────│              │             │
      │                  │                │              │             │
      │                  │                │  8. Find/Create user        │
      │                  │                │─────────────▶│             │
      │                  │                │              │             │
      │                  │                │  9. Save session            │
      │                  │                │─────────────────────────────▶
      │                  │                │              │             │
      │  10. Return tokens + user         │              │             │
      │  Set-Cookie: refresh_token        │              │             │
      │◀──────────────────────────────────│              │             │
```

### 5.2 OAuthクライアントインターフェース

```go
// internal/domain/service/oauth_client.go

package service

import (
    "context"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// OAuthTokens はOAuthトークンを定義します
type OAuthTokens struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int
}

// OAuthUserInfo はOAuthユーザー情報を定義します
type OAuthUserInfo struct {
    ProviderUserID string
    Email          string
    Name           string
    AvatarURL      string
}

// OAuthClient はOAuthクライアントインターフェースを定義します
type OAuthClient interface {
    ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error)
    GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
    Provider() valueobject.OAuthProvider
}

// OAuthClientFactory はOAuthクライアントファクトリーインターフェースを定義します
type OAuthClientFactory interface {
    GetClient(provider valueobject.OAuthProvider) (OAuthClient, error)
}
```

### 5.3 OAuthLoginコマンド（CQRS）

```go
// internal/usecase/auth/command/oauth_login.go

package command

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// OAuthLoginInput はOAuthログインの入力を定義します
type OAuthLoginInput struct {
    Provider  string
    Code      string
    UserAgent string
    IPAddress string
}

// OAuthLoginOutput はOAuthログインの出力を定義します
type OAuthLoginOutput struct {
    SessionID string
    User      *entity.User
    IsNewUser bool
}

// OAuthLoginCommand はOAuthログインコマンドです
type OAuthLoginCommand struct {
    userRepo          repository.UserRepository
    profileRepo       repository.UserProfileRepository
    oauthAccountRepo  repository.OAuthAccountRepository
    folderRepo        repository.FolderRepository
    folderClosureRepo repository.FolderClosureRepository
    oauthFactory      service.OAuthClientFactory
    txManager         *database.TxManager
    sessionRepo       repository.SessionRepository
}

// NewOAuthLoginCommand は新しいOAuthLoginCommandを作成します
func NewOAuthLoginCommand(
    userRepo repository.UserRepository,
    profileRepo repository.UserProfileRepository,
    oauthAccountRepo repository.OAuthAccountRepository,
    folderRepo repository.FolderRepository,
    folderClosureRepo repository.FolderClosureRepository,
    oauthFactory service.OAuthClientFactory,
    txManager *database.TxManager,
    sessionRepo repository.SessionRepository,
) *OAuthLoginCommand {
    return &OAuthLoginCommand{
        userRepo:          userRepo,
        profileRepo:       profileRepo,
        oauthAccountRepo:  oauthAccountRepo,
        folderRepo:        folderRepo,
        folderClosureRepo: folderClosureRepo,
        oauthFactory:      oauthFactory,
        txManager:         txManager,
        sessionRepo:       sessionRepo,
    }
}

// Execute はOAuthログインを実行します
func (c *OAuthLoginCommand) Execute(ctx context.Context, input OAuthLoginInput) (*OAuthLoginOutput, error) {
    // 1. プロバイダーの検証
    provider := valueobject.OAuthProvider(input.Provider)
    if !provider.IsValid() {
        return nil, apperror.NewValidationError("unsupported oauth provider", nil)
    }

    // 2. OAuthクライアントの取得
    oauthClient, err := c.oauthFactory.GetClient(provider)
    if err != nil {
        return nil, apperror.NewValidationError("unsupported oauth provider", nil)
    }

    // 3. 認可コードをトークンに交換
    tokens, err := oauthClient.ExchangeCode(ctx, input.Code)
    if err != nil {
        return nil, apperror.NewValidationError("invalid authorization code", nil)
    }

    // 4. ユーザー情報の取得
    userInfo, err := oauthClient.GetUserInfo(ctx, tokens.AccessToken)
    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    // 5. トランザクション内でユーザー処理
    var user *entity.User
    var isNewUser bool

    err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 5a. OAuthアカウントで検索
        oauthAccount, txErr := c.oauthAccountRepo.FindByProviderAndUserID(ctx, provider, userInfo.ProviderUserID)
        if txErr == nil {
            // 既存のOAuthアカウントがある場合
            user, txErr = c.userRepo.FindByID(ctx, oauthAccount.UserID)
            if txErr != nil {
                return txErr
            }
            isNewUser = false
            return nil
        }

        // 5b. メールアドレスでユーザーを検索
        email, _ := valueobject.NewEmail(userInfo.Email)
        user, txErr = c.userRepo.FindByEmail(ctx, email)
        if txErr == nil {
            // 既存ユーザーにOAuthアカウントを紐付け
            oauthAccount = &entity.OAuthAccount{
                ID:             uuid.New(),
                UserID:         user.ID,
                Provider:       provider,
                ProviderUserID: userInfo.ProviderUserID,
                Email:          userInfo.Email,
                AccessToken:    tokens.AccessToken,
                RefreshToken:   tokens.RefreshToken,
                CreatedAt:      time.Now(),
                UpdatedAt:      time.Now(),
            }
            if txErr = c.oauthAccountRepo.Create(ctx, oauthAccount); txErr != nil {
                return txErr
            }

            // ユーザーがpending状態の場合、activeに変更
            if user.Status == entity.UserStatusPending {
                user.Status = entity.UserStatusActive
                user.EmailVerified = true
                user.UpdatedAt = time.Now()
                if txErr = c.userRepo.Update(ctx, user); txErr != nil {
                    return txErr
                }
            }

            isNewUser = false
            return nil
        }

        // 5c. 新規ユーザーを作成
        now := time.Now()
        userID := uuid.New()
        personalFolderID := uuid.New()

        // Personal Folder を作成（ユーザーと1:1の関係）
        folderName, _ := valueobject.NewFolderName(userInfo.Name) // 初期名はユーザー名
        personalFolder := &entity.Folder{
            ID:        personalFolderID,
            Name:      folderName,
            ParentID:  nil,        // ルートレベル
            OwnerID:   userID,     // 作成者がオーナー
            CreatedBy: userID,     // 作成者
            Depth:     0,
            Status:    valueobject.FolderStatusActive,
            CreatedAt: now,
            UpdatedAt: now,
        }

        if txErr = c.folderRepo.Create(ctx, personalFolder); txErr != nil {
            return fmt.Errorf("failed to create personal folder: %w", txErr)
        }

        // フォルダのClosure Table エントリを作成（自己参照）
        if txErr = c.folderClosureRepo.CreateSelfReference(ctx, personalFolderID); txErr != nil {
            return fmt.Errorf("failed to create folder closure: %w", txErr)
        }

        user = &entity.User{
            ID:            userID,
            Email:         email,
            Name:          userInfo.Name,
            PasswordHash:  "", // OAuthユーザーはパスワードなし
            Status:        entity.UserStatusActive,
            EmailVerified: true, // OAuthはメール確認済みとみなす
            CreatedAt:     now,
            UpdatedAt:     now,
        }
        // Personal Folder への参照を設定（ポインタ型）
        user.SetPersonalFolder(personalFolderID)

        if txErr = c.userRepo.Create(ctx, user); txErr != nil {
            return txErr
        }

        // UserProfileを作成（AvatarURLを含む）
        profile := entity.NewUserProfile(user.ID)
        profile.AvatarURL = userInfo.AvatarURL
        if txErr = c.profileRepo.Upsert(ctx, profile); txErr != nil {
            return txErr
        }

        oauthAccount = &entity.OAuthAccount{
            ID:             uuid.New(),
            UserID:         user.ID,
            Provider:       provider,
            ProviderUserID: userInfo.ProviderUserID,
            Email:          userInfo.Email,
            AccessToken:    tokens.AccessToken,
            RefreshToken:   tokens.RefreshToken,
            CreatedAt:      now,
            UpdatedAt:      now,
        }

        if txErr = c.oauthAccountRepo.Create(ctx, oauthAccount); txErr != nil {
            return txErr
        }

        isNewUser = true
        return nil
    })

    if err != nil {
        return nil, err
    }

    // 6. ユーザー状態チェック
    if user.Status != entity.UserStatusActive {
        return nil, apperror.NewUnauthorizedError("account is not active")
    }

    // 7. セッション上限チェック（最大10セッション）
    sessions, err := c.sessionRepo.FindAllByUserID(ctx, user.ID)
    if err == nil && len(sessions) >= entity.MaxActiveSessionsPerUser {
        // 最古のセッションを削除
        if err := c.sessionRepo.DeleteOldest(ctx, user.ID); err != nil {
            return nil, apperror.NewInternalError(err)
        }
    }

    // 8. セッション作成
    sessionID := uuid.New().String()
    now := time.Now()

    session := &entity.Session{
        ID:         sessionID,
        UserID:     user.ID,
        UserAgent:  input.UserAgent,
        IPAddress:  input.IPAddress,
        ExpiresAt:  now.Add(entity.SessionTTL),
        CreatedAt:  now,
        LastUsedAt: now,
    }

    if err := c.sessionRepo.Save(ctx, session); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &OAuthLoginOutput{
        SessionID: sessionID,
        User:      user,
        IsNewUser: isNewUser,
    }, nil
}
```

---

## 6. セッション管理

### 6.1 セッション検証（ミドルウェア）

すべてのAPI リクエストで、セッションの有効性を検証します。

```go
// internal/interface/middleware/auth.go

package middleware

import (
    "net/http"

    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

type AuthMiddleware struct {
    sessionRepo repository.SessionRepository
    userRepo    repository.UserRepository
}

func NewAuthMiddleware(sessionRepo repository.SessionRepository, userRepo repository.UserRepository) *AuthMiddleware {
    return &AuthMiddleware{
        sessionRepo: sessionRepo,
        userRepo:    userRepo,
    }
}

func (m *AuthMiddleware) RequireAuth() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // 1. Cookieからセッション IDを取得
            cookie, err := c.Cookie("session_id")
            if err != nil {
                return apperror.NewUnauthorizedError("session not found")
            }

            // 2. Redisからセッションを取得
            session, err := m.sessionRepo.FindByID(c.Request().Context(), cookie.Value)
            if err != nil {
                return apperror.NewUnauthorizedError("invalid session")
            }

            // 3. セッションの有効期限をチェック
            if session.IsExpired() {
                m.sessionRepo.Delete(c.Request().Context(), session.ID)
                return apperror.NewUnauthorizedError("session expired")
            }

            // 4. ユーザーの状態をチェック
            user, err := m.userRepo.FindByID(c.Request().Context(), session.UserID)
            if err != nil {
                return apperror.NewUnauthorizedError("user not found")
            }

            if user.Status != entity.UserStatusActive {
                return apperror.NewUnauthorizedError("account is not active")
            }

            // 5. セッションをリフレッシュ（スライディングウィンドウ）
            session.Refresh()
            m.sessionRepo.Save(c.Request().Context(), session)

            // 6. コンテキストにユーザー情報を設定
            c.Set("user", user)
            c.Set("session_id", session.ID)

            return next(c)
        }
    }
}

// GetUser はコンテキストからユーザーを取得します
func GetUser(c echo.Context) *entity.User {
    user, ok := c.Get("user").(*entity.User)
    if !ok {
        return nil
    }
    return user
}

// GetSessionID はコンテキストからセッションIDを取得します
func GetSessionID(c echo.Context) string {
    sessionID, ok := c.Get("session_id").(string)
    if !ok {
        return ""
    }
    return sessionID
}
```

### 6.2 セッション自動延長（スライディングウィンドウ）

- 各リクエスト時にセッションの有効期限を7日間延長
- アクティブなユーザーは自動的にログイン状態を維持
- 7日間アクセスがない場合にセッションが期限切れ

---

## 7. パスワードリセット

### 7.1 リセットフロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Password Reset Flow                                 │
└─────────────────────────────────────────────────────────────────────────────┘

【リセット要求】
┌────────┐          ┌──────────┐          ┌──────────┐          ┌───────┐
│ Client │          │   API    │          │   DB     │          │ Email │
└────┬───┘          └────┬─────┘          └────┬─────┘          └───┬───┘
     │                   │                     │                    │
     │  1. POST /password/forgot               │                    │
     │  {email}          │                     │                    │
     │──────────────────▶│                     │                    │
     │                   │                     │                    │
     │                   │  2. Find user       │                    │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  3. Create reset token                   │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  4. Send reset email                     │
     │                   │─────────────────────────────────────────▶│
     │                   │                     │                    │
     │  5. 200 OK (always, to prevent enumeration)                  │
     │◀──────────────────│                     │                    │

【パスワード更新】
     │                   │                     │                    │
     │  6. POST /password/reset                │                    │
     │  {token, newPassword}                   │                    │
     │──────────────────▶│                     │                    │
     │                   │                     │                    │
     │                   │  7. Validate token  │                    │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  8. Update password │                    │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  9. Revoke all sessions                  │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │  10. 200 OK       │                     │                    │
     │◀──────────────────│                     │                    │
```

### 7.2 パスワードリセット要求コマンド（CQRS）

```go
// internal/usecase/auth/command/forgot_password.go

package command

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// ForgotPasswordInput はパスワードリセット要求の入力を定義します
type ForgotPasswordInput struct {
    Email string
}

// ForgotPasswordOutput はパスワードリセット要求の出力を定義します
type ForgotPasswordOutput struct {
    Message string
}

// ForgotPasswordCommand はパスワードリセット要求コマンドです
type ForgotPasswordCommand struct {
    userRepo               repository.UserRepository
    passwordResetTokenRepo repository.PasswordResetTokenRepository
    emailSender            service.EmailSender
    appURL                 string
}

// NewForgotPasswordCommand は新しいForgotPasswordCommandを作成します
func NewForgotPasswordCommand(
    userRepo repository.UserRepository,
    passwordResetTokenRepo repository.PasswordResetTokenRepository,
    emailSender service.EmailSender,
    appURL string,
) *ForgotPasswordCommand {
    return &ForgotPasswordCommand{
        userRepo:               userRepo,
        passwordResetTokenRepo: passwordResetTokenRepo,
        emailSender:            emailSender,
        appURL:                 appURL,
    }
}

// Execute はパスワードリセット要求を実行します
// ユーザー列挙攻撃を防ぐため、常に成功メッセージを返す
func (c *ForgotPasswordCommand) Execute(ctx context.Context, input ForgotPasswordInput) (*ForgotPasswordOutput, error) {
    successMsg := &ForgotPasswordOutput{
        Message: "If your email is registered, you will receive a password reset link.",
    }

    email, err := valueobject.NewEmail(input.Email)
    if err != nil {
        return successMsg, nil
    }

    user, err := c.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return successMsg, nil
    }

    // pending状態のユーザーにはリセットメールを送らない
    if user.Status == entity.UserStatusPending {
        return successMsg, nil
    }

    // OAuthのみのユーザーにはリセットメールを送らない
    if user.PasswordHash == "" {
        return successMsg, nil
    }

    // リセットトークン作成
    now := time.Now()
    token := &entity.PasswordResetToken{
        ID:        uuid.New(),
        UserID:    user.ID,
        Token:     generateSecureToken(),
        ExpiresAt: now.Add(1 * time.Hour),
        CreatedAt: now,
    }

    if err := c.passwordResetTokenRepo.Create(ctx, token); err != nil {
        slog.Error("failed to create password reset token", "error", err)
        return successMsg, nil
    }

    // リセットメール送信
    if c.emailSender != nil {
        resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", c.appURL, token.Token)
        if err := c.emailSender.SendPasswordReset(ctx, user.Email.String(), user.Name, resetURL); err != nil {
            slog.Error("failed to send password reset email", "error", err)
        }
    }

    return successMsg, nil
}
```

### 7.3 パスワードリセット実行コマンド（CQRS）

```go
// internal/usecase/auth/command/reset_password.go

package command

import (
    "context"
    "time"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ResetPasswordInput はパスワードリセットの入力を定義します
type ResetPasswordInput struct {
    Token    string
    Password string
}

// ResetPasswordOutput はパスワードリセットの出力を定義します
type ResetPasswordOutput struct {
    Message string
}

// ResetPasswordCommand はパスワードリセットコマンドです
type ResetPasswordCommand struct {
    userRepo               repository.UserRepository
    passwordResetTokenRepo repository.PasswordResetTokenRepository
    txManager              repository.TransactionManager
}

// NewResetPasswordCommand は新しいResetPasswordCommandを作成します
func NewResetPasswordCommand(
    userRepo repository.UserRepository,
    passwordResetTokenRepo repository.PasswordResetTokenRepository,
    txManager repository.TransactionManager,
) *ResetPasswordCommand {
    return &ResetPasswordCommand{
        userRepo:               userRepo,
        passwordResetTokenRepo: passwordResetTokenRepo,
        txManager:              txManager,
    }
}

// Execute はパスワードリセットを実行します
func (c *ResetPasswordCommand) Execute(ctx context.Context, input ResetPasswordInput) (*ResetPasswordOutput, error) {
    // 1. トークンを検索
    resetToken, err := c.passwordResetTokenRepo.FindByToken(ctx, input.Token)
    if err != nil {
        return nil, apperror.NewValidationError("invalid or expired reset token", nil)
    }

    // 2. トークンの有効性チェック
    if !resetToken.IsValid() {
        if resetToken.IsUsed() {
            return nil, apperror.NewValidationError("reset token already used", nil)
        }
        return nil, apperror.NewValidationError("reset token expired", nil)
    }

    // 3. ユーザーを取得
    user, err := c.userRepo.FindByID(ctx, resetToken.UserID)
    if err != nil {
        return nil, apperror.NewNotFoundError("user")
    }

    // 4. 新しいパスワードのバリデーション
    password, err := valueobject.NewPassword(input.Password, user.Email.String())
    if err != nil {
        return nil, apperror.NewValidationError(err.Error(), nil)
    }

    // 5. トランザクションで更新
    err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // パスワード更新
        user.PasswordHash = password.Hash()
        user.UpdatedAt = time.Now()

        if err := c.userRepo.Update(ctx, user); err != nil {
            return err
        }

        // トークンを使用済みにする
        return c.passwordResetTokenRepo.MarkAsUsed(ctx, resetToken.ID)
    })

    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &ResetPasswordOutput{Message: "Password reset successfully"}, nil
}
```

---

## 8. ログアウト

### 8.1 ログアウトコマンド（CQRS）

```go
// internal/usecase/auth/command/logout.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// LogoutCommand はログアウトコマンドです
type LogoutCommand struct {
    sessionRepo repository.SessionRepository
}

// NewLogoutCommand は新しいLogoutCommandを作成します
func NewLogoutCommand(sessionRepo repository.SessionRepository) *LogoutCommand {
    return &LogoutCommand{
        sessionRepo: sessionRepo,
    }
}

// Execute はログアウトを実行します
func (c *LogoutCommand) Execute(ctx context.Context, sessionID string) error {
    return c.sessionRepo.Delete(ctx, sessionID)
}

// ExecuteAll は全セッションからログアウトを実行します
func (c *LogoutCommand) ExecuteAll(ctx context.Context, userID uuid.UUID) error {
    return c.sessionRepo.DeleteByUserID(ctx, userID)
}
```

---

## 9. APIハンドラー

### 9.1 認証ハンドラー（CQRS）

```go
// internal/interface/handler/auth_handler.go

package handler

import (
    "net/http"

    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
    authcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// AuthHandler は認証関連のHTTPハンドラーです
type AuthHandler struct {
    // Commands
    registerCommand                *authcmd.RegisterCommand
    loginCommand                   *authcmd.LoginCommand
    logoutCommand                  *authcmd.LogoutCommand
    verifyEmailCommand             *authcmd.VerifyEmailCommand
    resendEmailVerificationCommand *authcmd.ResendEmailVerificationCommand
    forgotPasswordCommand          *authcmd.ForgotPasswordCommand
    resetPasswordCommand           *authcmd.ResetPasswordCommand
    changePasswordCommand          *authcmd.ChangePasswordCommand
    oauthLoginCommand              *authcmd.OAuthLoginCommand
}

// NewAuthHandler は新しいAuthHandlerを作成します
func NewAuthHandler(
    registerCommand *authcmd.RegisterCommand,
    loginCommand *authcmd.LoginCommand,
    logoutCommand *authcmd.LogoutCommand,
    verifyEmailCommand *authcmd.VerifyEmailCommand,
    resendEmailVerificationCommand *authcmd.ResendEmailVerificationCommand,
    forgotPasswordCommand *authcmd.ForgotPasswordCommand,
    resetPasswordCommand *authcmd.ResetPasswordCommand,
    changePasswordCommand *authcmd.ChangePasswordCommand,
    oauthLoginCommand *authcmd.OAuthLoginCommand,
) *AuthHandler {
    return &AuthHandler{
        registerCommand:                registerCommand,
        loginCommand:                   loginCommand,
        logoutCommand:                  logoutCommand,
        verifyEmailCommand:             verifyEmailCommand,
        resendEmailVerificationCommand: resendEmailVerificationCommand,
        forgotPasswordCommand:          forgotPasswordCommand,
        resetPasswordCommand:           resetPasswordCommand,
        changePasswordCommand:          changePasswordCommand,
        oauthLoginCommand:              oauthLoginCommand,
    }
}

// Register はユーザー登録を処理します
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c echo.Context) error {
    var req request.RegisterRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    output, err := h.registerCommand.Execute(c.Request().Context(), authcmd.RegisterInput{
        Email:    req.Email,
        Password: req.Password,
        Name:     req.Name,
    })
    if err != nil {
        return err
    }

    return presenter.Created(c, response.RegisterResponse{
        UserID:  output.UserID.String(),
        Message: "Registration successful. Please check your email to verify your account.",
    })
}

// Login はログインを処理します
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c echo.Context) error {
    var req request.LoginRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    output, err := h.loginCommand.Execute(c.Request().Context(), authcmd.LoginInput{
        Email:     req.Email,
        Password:  req.Password,
        UserAgent: c.Request().UserAgent(),
        IPAddress: c.RealIP(),
    })
    if err != nil {
        return err
    }

    // Session IDをHttpOnly Cookieに設定
    h.setSessionCookie(c, output.SessionID)

    // レスポンスボディにはユーザー情報のみ
    return presenter.OK(c, response.LoginResponse{
        User: response.ToUserResponse(output.User),
    })
}

// Logout はログアウトを処理します
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
    sessionID := middleware.GetSessionID(c)

    if err := h.logoutCommand.Execute(c.Request().Context(), sessionID); err != nil {
        // エラーでも成功扱い
    }

    // セッションCookieを削除
    h.clearSessionCookie(c)

    return presenter.OK(c, map[string]string{"message": "logged out successfully"})
}

// Me は現在のユーザー情報を取得します
// GET /api/v1/me
func (h *AuthHandler) Me(c echo.Context) error {
    user := middleware.GetUser(c)
    if user == nil {
        return apperror.NewUnauthorizedError("not authenticated")
    }

    return presenter.OK(c, response.ToUserResponse(user))
}

// VerifyEmail はメール確認を処理します
// POST /api/v1/auth/email/verify?token=xxx
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
    token := c.QueryParam("token")
    if token == "" {
        return apperror.NewValidationError("token is required", nil)
    }

    output, err := h.verifyEmailCommand.Execute(c.Request().Context(), authcmd.VerifyEmailInput{
        Token: token,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.VerifyEmailResponse{
        Message: output.Message,
    })
}

// ResendEmailVerification は確認メール再送を処理します
// POST /api/v1/auth/email/resend
func (h *AuthHandler) ResendEmailVerification(c echo.Context) error {
    var req request.ResendEmailVerificationRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    output, err := h.resendEmailVerificationCommand.Execute(c.Request().Context(), authcmd.ResendEmailVerificationInput{
        Email: req.Email,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.ResendEmailVerificationResponse{
        Message: output.Message,
    })
}

// ForgotPassword はパスワードリセットリクエストを処理します
// POST /api/v1/auth/password/forgot
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
    var req request.ForgotPasswordRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    output, err := h.forgotPasswordCommand.Execute(c.Request().Context(), authcmd.ForgotPasswordInput{
        Email: req.Email,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.ForgotPasswordResponse{
        Message: output.Message,
    })
}

// ResetPassword はパスワードリセットを処理します
// POST /api/v1/auth/password/reset
func (h *AuthHandler) ResetPassword(c echo.Context) error {
    var req request.ResetPasswordRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    output, err := h.resetPasswordCommand.Execute(c.Request().Context(), authcmd.ResetPasswordInput{
        Token:    req.Token,
        Password: req.Password,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.ResetPasswordResponse{
        Message: output.Message,
    })
}

// ChangePassword はパスワード変更を処理します（認証必須）
// POST /api/v1/auth/password/change
func (h *AuthHandler) ChangePassword(c echo.Context) error {
    user := middleware.GetUser(c)
    if user == nil {
        return apperror.NewUnauthorizedError("not authenticated")
    }

    var req request.ChangePasswordRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    output, err := h.changePasswordCommand.Execute(c.Request().Context(), authcmd.ChangePasswordInput{
        UserID:          user.ID,
        CurrentPassword: req.CurrentPassword,
        NewPassword:     req.NewPassword,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.ChangePasswordResponse{
        Message: output.Message,
    })
}

// OAuthLogin はOAuthログインを処理します
// POST /api/v1/auth/oauth/:provider
func (h *AuthHandler) OAuthLogin(c echo.Context) error {
    provider := c.Param("provider")
    if provider == "" {
        return apperror.NewValidationError("provider is required", nil)
    }

    var req request.OAuthLoginRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    output, err := h.oauthLoginCommand.Execute(c.Request().Context(), authcmd.OAuthLoginInput{
        Provider:  provider,
        Code:      req.Code,
        UserAgent: c.Request().UserAgent(),
        IPAddress: c.RealIP(),
    })
    if err != nil {
        return err
    }

    // Session IDをHttpOnly Cookieに設定
    h.setSessionCookie(c, output.SessionID)

    // レスポンスボディにはユーザー情報のみ
    return presenter.OK(c, response.OAuthLoginResponse{
        User:      response.ToUserResponse(output.User),
        IsNewUser: output.IsNewUser,
    })
}

func (h *AuthHandler) setSessionCookie(c echo.Context, sessionID string) {
    c.SetCookie(&http.Cookie{
        Name:     "session_id",
        Value:    sessionID,
        Path:     "/",  // すべてのパスで送信
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode, // OAuthリダイレクト対応
        MaxAge:   7 * 24 * 60 * 60,     // 7日
    })
}

func (h *AuthHandler) clearSessionCookie(c echo.Context) {
    c.SetCookie(&http.Cookie{
        Name:     "session_id",
        Value:    "",
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   -1,
    })
}
```

---

## 10. 受け入れ基準

### 10.1 機能要件

| 項目 | 基準 |
|------|------|
| ユーザー登録 | メールアドレス・パスワードで登録できる |
| Personal Folder自動作成 | ユーザー登録時にPersonal Folderが自動作成される |
| User-PersonalFolder 1:1関係 | ユーザーとPersonal Folderは必ず1対1の関係を維持 |
| メール確認 | 確認メールのリンクで認証完了できる |
| 確認メール再送 | 未確認ユーザーに確認メールを再送できる |
| ログイン | メール/パスワードでログインできる |
| OAuth認証 | Google/GitHubでログインできる（POST /oauth/:provider） |
| OAuth新規ユーザー | OAuthで新規ユーザー作成時もPersonal Folderが自動作成される |
| トークンリフレッシュ | リフレッシュトークンで新しいトークンを取得できる |
| ログアウト | セッションを終了できる |
| パスワードリセット | メールでリセットリンクを受け取り、パスワードを変更できる |
| パスワード変更 | 認証済みユーザーがパスワードを変更できる |
| ユーザー情報取得 | GET /me で現在のユーザー情報を取得できる |

### 10.2 セキュリティ要件

| 項目 | 基準 |
|------|------|
| パスワードハッシュ | bcrypt cost 12 |
| セッション有効期限 | 7日（スライディングウィンドウ） |
| セッション保存 | HttpOnly Cookie + Redis |
| セッション上限 | 1ユーザー最大10セッション |
| Cookie設定 | HttpOnly, Secure, SameSite=Lax |
| レート制限 | ログイン/登録: 10 req/min/IP |

### 10.3 チェックリスト

- [x] ユーザー登録が正常に動作する
- [x] ユーザー登録時にPersonal Folderが自動作成される
- [x] UserとPersonal Folderが1:1の関係を維持する
- [x] personal_folder_idが正しく設定される
- [x] Personal Folderの初期名がユーザー名になる
- [x] メール確認が正常に動作する
- [x] 確認メール再送が正常に動作する
- [x] メール/パスワードログインが正常に動作する
- [x] Google OAuthログインが正常に動作する
- [x] GitHub OAuthログインが正常に動作する
- [x] OAuth新規ユーザー作成時にPersonal Folderが自動作成される
- [x] セッションがRedisに保存される
- [x] セッションがスライディングウィンドウで延長される
- [x] ログアウトでセッションが削除される
- [x] 11番目のセッション作成時に最古が自動削除される
- [x] パスワードリセット要求でメールが送信される
- [x] パスワードリセットでパスワードが更新される
- [x] パスワード変更が正常に動作する
- [x] GET /me でユーザー情報が取得できる
- [x] 無効なセッションIDが拒否される
- [x] 期限切れセッションが拒否される
- [x] レート制限が正しく動作する

---

## 関連ドキュメント

- [infra-redis.md](./infra-redis.md) - セッションストア・JWTブラックリスト
- [infra-email.md](./infra-email.md) - 認証メール・リセットメール
- [infra-api.md](./infra-api.md) - 認証ミドルウェア
- [SECURITY.md](../02-architecture/SECURITY.md) - セキュリティ設計
- [user.md](../03-domains/user.md) - ユーザードメイン
- [folder.md](../03-domains/folder.md) - フォルダドメイン（Personal Folder連携）
- [storage-folder.md](./storage-folder.md) - フォルダ機能仕様
