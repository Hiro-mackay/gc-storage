# 認証・アイデンティティ仕様書

## 概要

本ドキュメントでは、GC Storageにおけるユーザー登録、認証、セッション管理、OAuth連携の実装仕様を定義します。

**関連アーキテクチャ:**
- [SECURITY.md](../02-architecture/SECURITY.md) - セキュリティ設計
- [user.md](../03-domains/user.md) - ユーザードメイン定義
- [infra-redis.md](./infra-redis.md) - セッションストア

---

## 1. JWT認証

### 1.1 トークン仕様

| トークン | 有効期限 | 保存場所 | 用途 |
|---------|---------|---------|------|
| Access Token | 15分 | メモリ / LocalStorage | API認証 |
| Refresh Token | 7日 | HttpOnly Cookie / Redis | Access Token再発行 |

### 1.2 JWT Claims構造

```go
// pkg/jwt/claims.go

package jwt

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

// AccessTokenClaims はアクセストークンのクレームを定義します
type AccessTokenClaims struct {
    jwt.RegisteredClaims
    UserID    uuid.UUID `json:"uid"`
    Email     string    `json:"email"`
    SessionID string    `json:"sid"`
}

// RefreshTokenClaims はリフレッシュトークンのクレームを定義します
type RefreshTokenClaims struct {
    jwt.RegisteredClaims
    UserID    uuid.UUID `json:"uid"`
    SessionID string    `json:"sid"`
}

// Config はJWT設定を定義します
type Config struct {
    SecretKey          string        // HMAC署名用シークレットキー
    Issuer             string        // 発行者
    Audience           []string      // 対象者
    AccessTokenExpiry  time.Duration // アクセストークン有効期限
    RefreshTokenExpiry time.Duration // リフレッシュトークン有効期限
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
    return Config{
        Issuer:             "gc-storage",
        Audience:           []string{"gc-storage-api"},
        AccessTokenExpiry:  15 * time.Minute,
        RefreshTokenExpiry: 7 * 24 * time.Hour,
    }
}
```

### 1.3 JWTサービス

```go
// pkg/jwt/service.go

package jwt

import (
    "context"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

// JWTService はJWT操作を提供します
type JWTService struct {
    config Config
}

// NewJWTService は新しいJWTServiceを作成します
func NewJWTService(cfg Config) *JWTService {
    return &JWTService{config: cfg}
}

// GenerateTokenPair はアクセストークンとリフレッシュトークンのペアを生成します
func (s *JWTService) GenerateTokenPair(userID uuid.UUID, email, sessionID string) (accessToken, refreshToken string, err error) {
    now := time.Now()

    // Access Token
    accessClaims := AccessTokenClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.config.Issuer,
            Subject:   userID.String(),
            Audience:  s.config.Audience,
            ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenExpiry)),
            IssuedAt:  jwt.NewNumericDate(now),
            ID:        uuid.New().String(),
        },
        UserID:    userID,
        Email:     email,
        SessionID: sessionID,
    }

    accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.config.SecretKey))
    if err != nil {
        return "", "", fmt.Errorf("failed to sign access token: %w", err)
    }

    // Refresh Token
    refreshClaims := RefreshTokenClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.config.Issuer,
            Subject:   userID.String(),
            Audience:  s.config.Audience,
            ExpiresAt: jwt.NewNumericDate(now.Add(s.config.RefreshTokenExpiry)),
            IssuedAt:  jwt.NewNumericDate(now),
            ID:        uuid.New().String(),
        },
        UserID:    userID,
        SessionID: sessionID,
    }

    refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.config.SecretKey))
    if err != nil {
        return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
    }

    return accessToken, refreshToken, nil
}

// ValidateAccessToken はアクセストークンを検証します
func (s *JWTService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(s.config.SecretKey), nil
    })

    if err != nil {
        return nil, fmt.Errorf("failed to parse access token: %w", err)
    }

    claims, ok := token.Claims.(*AccessTokenClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid access token")
    }

    return claims, nil
}

// ValidateRefreshToken はリフレッシュトークンを検証します
func (s *JWTService) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(s.config.SecretKey), nil
    })

    if err != nil {
        return nil, fmt.Errorf("failed to parse refresh token: %w", err)
    }

    claims, ok := token.Claims.(*RefreshTokenClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid refresh token")
    }

    return claims, nil
}

// GetAccessTokenExpiry はアクセストークンの有効期限を返します
func (s *JWTService) GetAccessTokenExpiry() time.Duration {
    return s.config.AccessTokenExpiry
}

// GetRefreshTokenExpiry はリフレッシュトークンの有効期限を返します
func (s *JWTService) GetRefreshTokenExpiry() time.Duration {
    return s.config.RefreshTokenExpiry
}
```

---

## 2. ユーザー登録

### 2.1 登録フロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          User Registration Flow                              │
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
     │                   │  3. Create User (status: pending)        │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  4. Create verification token            │
     │                   │────────────────────▶│                    │
     │                   │                     │                    │
     │                   │  5. Send verification email              │
     │                   │─────────────────────────────────────────▶│
     │                   │                     │                    │
     │  6. 201 Created   │                     │                    │
     │  {user_id, message: "Please verify..."}                      │
     │◀──────────────────│                     │                    │
```

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
    txManager                  repository.TransactionManager
    emailSender                service.EmailSender
    appURL                     string
}

// NewRegisterCommand は新しいRegisterCommandを作成します
func NewRegisterCommand(
    userRepo repository.UserRepository,
    emailVerificationTokenRepo repository.EmailVerificationTokenRepository,
    txManager repository.TransactionManager,
    emailSender service.EmailSender,
    appURL string,
) *RegisterCommand {
    return &RegisterCommand{
        userRepo:                   userRepo,
        emailVerificationTokenRepo: emailVerificationTokenRepo,
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

        // ユーザー作成
        user = &entity.User{
            ID:            uuid.New(),
            Email:         email,
            Name:          input.Name,
            PasswordHash:  password.Hash(),
            Status:        entity.UserStatusPending,
            EmailVerified: false,
            CreatedAt:     now,
            UpdatedAt:     now,
        }

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
     │                   │  5. Generate tokens │                    │
     │                   │  (access + refresh) │                    │
     │                   │                     │                    │
     │                   │  6. Create session  │                    │
     │                   │─────────────────────────────────────────▶│
     │                   │                     │                    │
     │  7. Return tokens │                     │                    │
     │  Set-Cookie: refresh_token              │                    │
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
    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
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
    AccessToken  string
    RefreshToken string
    ExpiresIn    int // seconds
    User         *entity.User
}

// LoginCommand はログインコマンドです
type LoginCommand struct {
    userRepo     repository.UserRepository
    sessionStore *cache.SessionStore  // Redis SessionStore
    jwtService   *jwt.JWTService
}

// NewLoginCommand は新しいLoginCommandを作成します
func NewLoginCommand(
    userRepo repository.UserRepository,
    sessionStore *cache.SessionStore,
    jwtService *jwt.JWTService,
) *LoginCommand {
    return &LoginCommand{
        userRepo:     userRepo,
        sessionStore: sessionStore,
        jwtService:   jwtService,
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

    // 4. セッション作成（Redisに保存）
    sessionID := uuid.New().String()
    now := time.Now()
    expiresAt := now.Add(c.jwtService.GetRefreshTokenExpiry())

    accessToken, refreshToken, err := c.jwtService.GenerateTokenPair(user.ID, user.Email.String(), sessionID)
    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    session := &cache.Session{
        ID:           sessionID,
        UserID:       user.ID,
        RefreshToken: refreshToken,
        UserAgent:    input.UserAgent,
        IPAddress:    input.IPAddress,
        ExpiresAt:    expiresAt,
        CreatedAt:    now,
        LastUsedAt:   now,
    }

    if err := c.sessionStore.Save(ctx, session); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &LoginOutput{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    int(c.jwtService.GetAccessTokenExpiry().Seconds()),
        User:         user,
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
    "time"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/database"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
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
    AccessToken  string
    RefreshToken string
    ExpiresIn    int
    User         *entity.User
    IsNewUser    bool
}

// OAuthLoginCommand はOAuthログインコマンドです
type OAuthLoginCommand struct {
    userRepo         repository.UserRepository
    oauthAccountRepo repository.OAuthAccountRepository
    oauthFactory     service.OAuthClientFactory
    txManager        *database.TxManager
    sessionStore     *cache.SessionStore
    jwtService       *jwt.JWTService
}

// NewOAuthLoginCommand は新しいOAuthLoginCommandを作成します
func NewOAuthLoginCommand(
    userRepo repository.UserRepository,
    oauthAccountRepo repository.OAuthAccountRepository,
    oauthFactory service.OAuthClientFactory,
    txManager *database.TxManager,
    sessionStore *cache.SessionStore,
    jwtService *jwt.JWTService,
) *OAuthLoginCommand {
    return &OAuthLoginCommand{
        userRepo:         userRepo,
        oauthAccountRepo: oauthAccountRepo,
        oauthFactory:     oauthFactory,
        txManager:        txManager,
        sessionStore:     sessionStore,
        jwtService:       jwtService,
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
        user = &entity.User{
            ID:            uuid.New(),
            Email:         email,
            Name:          userInfo.Name,
            PasswordHash:  "", // OAuthユーザーはパスワードなし
            Status:        entity.UserStatusActive,
            EmailVerified: true,
            AvatarURL:     userInfo.AvatarURL,
            CreatedAt:     now,
            UpdatedAt:     now,
        }

        if txErr = c.userRepo.Create(ctx, user); txErr != nil {
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

    // 7. セッション作成
    sessionID := uuid.New().String()
    now := time.Now()
    expiresAt := now.Add(c.jwtService.GetRefreshTokenExpiry())

    accessToken, refreshToken, err := c.jwtService.GenerateTokenPair(user.ID, user.Email.String(), sessionID)
    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    session := &cache.Session{
        ID:           sessionID,
        UserID:       user.ID,
        RefreshToken: refreshToken,
        UserAgent:    input.UserAgent,
        IPAddress:    input.IPAddress,
        ExpiresAt:    expiresAt,
        CreatedAt:    now,
        LastUsedAt:   now,
    }

    if err := c.sessionStore.Save(ctx, session); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &OAuthLoginOutput{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    int(c.jwtService.GetAccessTokenExpiry().Seconds()),
        User:         user,
        IsNewUser:    isNewUser,
    }, nil
}
```

---

## 6. トークンリフレッシュ

### 6.1 リフレッシュコマンド（CQRS）

```go
// internal/usecase/auth/command/refresh_token.go

package command

import (
    "context"
    "time"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// RefreshTokenInput はトークンリフレッシュの入力を定義します
type RefreshTokenInput struct {
    RefreshToken string
}

// RefreshTokenOutput はトークンリフレッシュの出力を定義します
type RefreshTokenOutput struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int
}

// RefreshTokenCommand はトークンリフレッシュコマンドです
type RefreshTokenCommand struct {
    userRepo     repository.UserRepository
    sessionStore *cache.SessionStore
    jwtService   *jwt.JWTService
    jwtBlacklist *cache.JWTBlacklist
}

// NewRefreshTokenCommand は新しいRefreshTokenCommandを作成します
func NewRefreshTokenCommand(
    userRepo repository.UserRepository,
    sessionStore *cache.SessionStore,
    jwtService *jwt.JWTService,
    jwtBlacklist *cache.JWTBlacklist,
) *RefreshTokenCommand {
    return &RefreshTokenCommand{
        userRepo:     userRepo,
        sessionStore: sessionStore,
        jwtService:   jwtService,
        jwtBlacklist: jwtBlacklist,
    }
}

// Execute はトークンリフレッシュを実行します
func (c *RefreshTokenCommand) Execute(ctx context.Context, input RefreshTokenInput) (*RefreshTokenOutput, error) {
    // 1. リフレッシュトークンを検証
    claims, err := c.jwtService.ValidateRefreshToken(input.RefreshToken)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("invalid refresh token")
    }

    // 2. セッションを検索（Redis）
    session, err := c.sessionStore.FindByID(ctx, claims.SessionID)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("session not found")
    }

    // 3. セッション有効性チェック
    if session.ExpiresAt.Before(time.Now()) {
        return nil, apperror.NewUnauthorizedError("session expired")
    }

    if session.RefreshToken != input.RefreshToken {
        // トークンが一致しない = トークン再利用攻撃の可能性
        c.sessionStore.DeleteByUserID(ctx, session.UserID)
        return nil, apperror.NewUnauthorizedError("token reuse detected")
    }

    // 4. ユーザー取得・状態チェック
    user, err := c.userRepo.FindByID(ctx, session.UserID)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("user not found")
    }

    if user.Status != entity.UserStatusActive {
        return nil, apperror.NewUnauthorizedError("account is not active")
    }

    // 5. 新しいトークンペアを生成（トークンローテーション）
    newAccessToken, newRefreshToken, err := c.jwtService.GenerateTokenPair(
        user.ID,
        user.Email.String(),
        session.ID,
    )
    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    // 6. セッションを更新
    session.RefreshToken = newRefreshToken
    session.LastUsedAt = time.Now()
    session.ExpiresAt = time.Now().Add(c.jwtService.GetRefreshTokenExpiry())

    if err := c.sessionStore.Save(ctx, session); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    // 7. 古いアクセストークンをブラックリストに追加
    c.jwtBlacklist.Add(ctx, claims.ID, claims.ExpiresAt.Time)

    return &RefreshTokenOutput{
        AccessToken:  newAccessToken,
        RefreshToken: newRefreshToken,
        ExpiresIn:    int(c.jwtService.GetAccessTokenExpiry().Seconds()),
    }, nil
}
```

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
    "time"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/infrastructure/cache"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/jwt"
)

// LogoutCommand はログアウトコマンドです
type LogoutCommand struct {
    sessionStore *cache.SessionStore
    jwtBlacklist *cache.JWTBlacklist
}

// NewLogoutCommand は新しいLogoutCommandを作成します
func NewLogoutCommand(
    sessionStore *cache.SessionStore,
    jwtBlacklist *cache.JWTBlacklist,
) *LogoutCommand {
    return &LogoutCommand{
        sessionStore: sessionStore,
        jwtBlacklist: jwtBlacklist,
    }
}

// Execute はログアウトを実行します
func (c *LogoutCommand) Execute(ctx context.Context, sessionID string, accessTokenClaims *jwt.AccessTokenClaims) error {
    // 1. セッションを削除
    if err := c.sessionStore.Delete(ctx, sessionID); err != nil {
        // エラーでも続行
    }

    // 2. アクセストークンをブラックリストに追加
    if accessTokenClaims != nil && c.jwtBlacklist != nil {
        if accessTokenClaims.ExpiresAt != nil {
            c.jwtBlacklist.Add(ctx, accessTokenClaims.ID, accessTokenClaims.ExpiresAt.Time)
        } else {
            // 有効期限がない場合は15分後に設定
            c.jwtBlacklist.Add(ctx, accessTokenClaims.ID, time.Now().Add(15*time.Minute))
        }
    }

    return nil
}

// ExecuteAll は全セッションからログアウトを実行します
func (c *LogoutCommand) ExecuteAll(ctx context.Context, userID uuid.UUID) error {
    return c.sessionStore.DeleteByUserID(ctx, userID)
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

    "github.com/google/uuid"
    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
    authcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
    authqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/query"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// AuthHandler は認証関連のHTTPハンドラーです
type AuthHandler struct {
    // Commands
    registerCommand                *authcmd.RegisterCommand
    loginCommand                   *authcmd.LoginCommand
    refreshTokenCommand            *authcmd.RefreshTokenCommand
    logoutCommand                  *authcmd.LogoutCommand
    verifyEmailCommand             *authcmd.VerifyEmailCommand
    resendEmailVerificationCommand *authcmd.ResendEmailVerificationCommand
    forgotPasswordCommand          *authcmd.ForgotPasswordCommand
    resetPasswordCommand           *authcmd.ResetPasswordCommand
    changePasswordCommand          *authcmd.ChangePasswordCommand
    oauthLoginCommand              *authcmd.OAuthLoginCommand

    // Queries
    getUserQuery *authqry.GetUserQuery
}

// NewAuthHandler は新しいAuthHandlerを作成します
func NewAuthHandler(
    registerCommand *authcmd.RegisterCommand,
    loginCommand *authcmd.LoginCommand,
    refreshTokenCommand *authcmd.RefreshTokenCommand,
    logoutCommand *authcmd.LogoutCommand,
    verifyEmailCommand *authcmd.VerifyEmailCommand,
    resendEmailVerificationCommand *authcmd.ResendEmailVerificationCommand,
    forgotPasswordCommand *authcmd.ForgotPasswordCommand,
    resetPasswordCommand *authcmd.ResetPasswordCommand,
    changePasswordCommand *authcmd.ChangePasswordCommand,
    oauthLoginCommand *authcmd.OAuthLoginCommand,
    getUserQuery *authqry.GetUserQuery,
) *AuthHandler {
    return &AuthHandler{
        registerCommand:                registerCommand,
        loginCommand:                   loginCommand,
        refreshTokenCommand:            refreshTokenCommand,
        logoutCommand:                  logoutCommand,
        verifyEmailCommand:             verifyEmailCommand,
        resendEmailVerificationCommand: resendEmailVerificationCommand,
        forgotPasswordCommand:          forgotPasswordCommand,
        resetPasswordCommand:           resetPasswordCommand,
        changePasswordCommand:          changePasswordCommand,
        oauthLoginCommand:              oauthLoginCommand,
        getUserQuery:                   getUserQuery,
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

    // リフレッシュトークンをHttpOnly Cookieに設定
    h.setRefreshTokenCookie(c, output.RefreshToken)

    return presenter.OK(c, response.LoginResponse{
        AccessToken: output.AccessToken,
        ExpiresIn:   output.ExpiresIn,
        User:        response.ToUserResponse(output.User),
    })
}

// Refresh はトークンリフレッシュを処理します
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c echo.Context) error {
    // Cookieからリフレッシュトークンを取得
    cookie, err := c.Cookie("refresh_token")
    if err != nil {
        return apperror.NewUnauthorizedError("refresh token not found")
    }

    output, err := h.refreshTokenCommand.Execute(c.Request().Context(), authcmd.RefreshTokenInput{
        RefreshToken: cookie.Value,
    })
    if err != nil {
        return err
    }

    // 新しいリフレッシュトークンをCookieに設定
    h.setRefreshTokenCookie(c, output.RefreshToken)

    return presenter.OK(c, response.RefreshResponse{
        AccessToken: output.AccessToken,
        ExpiresIn:   output.ExpiresIn,
    })
}

// Logout はログアウトを処理します
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
    sessionID := middleware.GetSessionID(c)
    accessClaims := middleware.GetAccessClaims(c)

    if err := h.logoutCommand.Execute(c.Request().Context(), sessionID, accessClaims); err != nil {
        // エラーでも成功扱い
    }

    // Cookieを削除
    h.clearRefreshTokenCookie(c)

    return presenter.OK(c, map[string]string{"message": "logged out successfully"})
}

// Me は現在のユーザー情報を取得します
// GET /api/v1/me
func (h *AuthHandler) Me(c echo.Context) error {
    claims := middleware.GetAccessClaims(c)
    if claims == nil {
        return apperror.NewUnauthorizedError("invalid token")
    }

    output, err := h.getUserQuery.Execute(c.Request().Context(), authqry.GetUserInput{
        UserID: uuid.MustParse(claims.UserID.String()),
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.ToUserResponse(output.User))
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
    claims := middleware.GetAccessClaims(c)
    if claims == nil {
        return apperror.NewUnauthorizedError("invalid token")
    }

    var req request.ChangePasswordRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    output, err := h.changePasswordCommand.Execute(c.Request().Context(), authcmd.ChangePasswordInput{
        UserID:          claims.UserID,
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

    // リフレッシュトークンをHttpOnly Cookieに設定
    h.setRefreshTokenCookie(c, output.RefreshToken)

    return presenter.OK(c, response.OAuthLoginResponse{
        AccessToken: output.AccessToken,
        ExpiresIn:   output.ExpiresIn,
        User:        response.ToUserResponse(output.User),
        IsNewUser:   output.IsNewUser,
    })
}

func (h *AuthHandler) setRefreshTokenCookie(c echo.Context, token string) {
    c.SetCookie(&http.Cookie{
        Name:     "refresh_token",
        Value:    token,
        Path:     "/api/v1/auth",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   7 * 24 * 60 * 60, // 7日
    })
}

func (h *AuthHandler) clearRefreshTokenCookie(c echo.Context) {
    c.SetCookie(&http.Cookie{
        Name:     "refresh_token",
        Value:    "",
        Path:     "/api/v1/auth",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
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
| メール確認 | 確認メールのリンクで認証完了できる |
| 確認メール再送 | 未確認ユーザーに確認メールを再送できる |
| ログイン | メール/パスワードでログインできる |
| OAuth認証 | Google/GitHubでログインできる（POST /oauth/:provider） |
| トークンリフレッシュ | リフレッシュトークンで新しいトークンを取得できる |
| ログアウト | セッションを終了できる |
| パスワードリセット | メールでリセットリンクを受け取り、パスワードを変更できる |
| パスワード変更 | 認証済みユーザーがパスワードを変更できる |
| ユーザー情報取得 | GET /me で現在のユーザー情報を取得できる |

### 10.2 セキュリティ要件

| 項目 | 基準 |
|------|------|
| パスワードハッシュ | bcrypt cost 12 |
| Access Token有効期限 | 15分 |
| Refresh Token有効期限 | 7日 |
| Refresh Token保存 | HttpOnly Cookie + Redis |
| トークンローテーション | リフレッシュ時に新しいペアを発行 |
| セッション上限 | 1ユーザー最大10セッション |
| レート制限 | ログイン/登録: 10 req/min/IP |

### 10.3 チェックリスト

- [x] ユーザー登録が正常に動作する
- [x] メール確認が正常に動作する
- [x] 確認メール再送が正常に動作する
- [x] メール/パスワードログインが正常に動作する
- [x] Google OAuthログインが正常に動作する
- [x] GitHub OAuthログインが正常に動作する
- [x] トークンリフレッシュが正常に動作する
- [x] トークンローテーションが実装されている
- [x] ログアウトでセッションが削除される
- [x] パスワードリセット要求でメールが送信される
- [x] パスワードリセットでパスワードが更新される
- [x] パスワード変更が正常に動作する
- [x] GET /me でユーザー情報が取得できる
- [x] 無効なトークンが拒否される
- [x] ブラックリストに追加されたトークンが拒否される
- [x] レート制限が正しく動作する

---

## 関連ドキュメント

- [infra-redis.md](./infra-redis.md) - セッションストア・JWTブラックリスト
- [infra-email.md](./infra-email.md) - 認証メール・リセットメール
- [infra-api.md](./infra-api.md) - 認証ミドルウェア
- [SECURITY.md](../02-architecture/SECURITY.md) - セキュリティ設計
- [user.md](../03-domains/user.md) - ユーザードメイン
