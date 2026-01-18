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

### 2.2 登録ユースケース

```go
// internal/usecase/auth/register.go

package auth

import (
    "context"
    "fmt"

    "github.com/google/uuid"

    "gc-storage/internal/domain/entity"
    "gc-storage/internal/domain/repository"
    "gc-storage/internal/domain/service"
    "gc-storage/internal/domain/valueobject"
    "gc-storage/pkg/apperror"
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

// RegisterUseCase はユーザー登録ユースケースです
type RegisterUseCase struct {
    userRepo             repository.UserRepository
    verificationTokenRepo repository.VerificationTokenRepository
    emailService         service.EmailService
    txManager            repository.TransactionManager
}

// NewRegisterUseCase は新しいRegisterUseCaseを作成します
func NewRegisterUseCase(
    userRepo repository.UserRepository,
    verificationTokenRepo repository.VerificationTokenRepository,
    emailService service.EmailService,
    txManager repository.TransactionManager,
) *RegisterUseCase {
    return &RegisterUseCase{
        userRepo:              userRepo,
        verificationTokenRepo: verificationTokenRepo,
        emailService:          emailService,
        txManager:             txManager,
    }
}

// Execute はユーザー登録を実行します
func (uc *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
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
    exists, err := uc.userRepo.Exists(ctx, email)
    if err != nil {
        return nil, apperror.NewInternalError(err)
    }
    if exists {
        return nil, apperror.NewConflictError("email already exists")
    }

    var user *entity.User

    // 4. トランザクションでユーザー作成
    err = uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // ユーザー作成
        user = &entity.User{
            ID:            uuid.New(),
            Email:         email,
            Name:          input.Name,
            PasswordHash:  password.Hash(),
            Status:        entity.UserStatusPending,
            EmailVerified: false,
            CreatedAt:     time.Now(),
            UpdatedAt:     time.Now(),
        }

        if err := uc.userRepo.Create(ctx, user); err != nil {
            return err
        }

        // 確認トークン作成
        token := &entity.VerificationToken{
            ID:        uuid.New(),
            UserID:    user.ID,
            Token:     generateSecureToken(),
            ExpiresAt: time.Now().Add(24 * time.Hour),
            CreatedAt: time.Now(),
        }

        if err := uc.verificationTokenRepo.Create(ctx, token); err != nil {
            return err
        }

        // 確認メール送信（非同期）
        verificationURL := fmt.Sprintf("https://app.gc-storage.example.com/verify?token=%s", token.Token)
        go uc.emailService.SendVerificationEmail(ctx, input.Email, input.Name, verificationURL)

        return nil
    })

    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &RegisterOutput{UserID: user.ID}, nil
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

### 3.2 確認ユースケース

```go
// internal/usecase/auth/verify_email.go

package auth

import (
    "context"
    "time"

    "gc-storage/internal/domain/entity"
    "gc-storage/internal/domain/repository"
    "gc-storage/pkg/apperror"
)

// VerifyEmailUseCase はメール確認ユースケースです
type VerifyEmailUseCase struct {
    userRepo              repository.UserRepository
    verificationTokenRepo repository.VerificationTokenRepository
    txManager             repository.TransactionManager
}

// NewVerifyEmailUseCase は新しいVerifyEmailUseCaseを作成します
func NewVerifyEmailUseCase(
    userRepo repository.UserRepository,
    verificationTokenRepo repository.VerificationTokenRepository,
    txManager repository.TransactionManager,
) *VerifyEmailUseCase {
    return &VerifyEmailUseCase{
        userRepo:              userRepo,
        verificationTokenRepo: verificationTokenRepo,
        txManager:             txManager,
    }
}

// Execute はメール確認を実行します
func (uc *VerifyEmailUseCase) Execute(ctx context.Context, token string) error {
    // 1. トークンを検索
    verificationToken, err := uc.verificationTokenRepo.FindByToken(ctx, token)
    if err != nil {
        return apperror.NewNotFoundError("verification token")
    }

    // 2. 有効期限チェック
    if verificationToken.ExpiresAt.Before(time.Now()) {
        return apperror.NewValidationError("verification token expired", nil)
    }

    // 3. ユーザーを取得
    user, err := uc.userRepo.FindByID(ctx, verificationToken.UserID)
    if err != nil {
        return apperror.NewNotFoundError("user")
    }

    // 4. すでに確認済みの場合
    if user.EmailVerified {
        return nil
    }

    // 5. トランザクションで更新
    return uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        user.EmailVerified = true
        user.Status = entity.UserStatusActive
        user.UpdatedAt = time.Now()

        if err := uc.userRepo.Update(ctx, user); err != nil {
            return err
        }

        // トークン削除
        return uc.verificationTokenRepo.Delete(ctx, verificationToken.ID)
    })
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

### 4.2 ログインユースケース

```go
// internal/usecase/auth/login.go

package auth

import (
    "context"
    "time"

    "github.com/google/uuid"

    "gc-storage/internal/domain/entity"
    "gc-storage/internal/domain/repository"
    "gc-storage/internal/domain/valueobject"
    "gc-storage/pkg/apperror"
    "gc-storage/pkg/jwt"
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

// LoginUseCase はログインユースケースです
type LoginUseCase struct {
    userRepo    repository.UserRepository
    sessionRepo repository.SessionRepository
    jwtService  *jwt.JWTService
}

// NewLoginUseCase は新しいLoginUseCaseを作成します
func NewLoginUseCase(
    userRepo repository.UserRepository,
    sessionRepo repository.SessionRepository,
    jwtService *jwt.JWTService,
) *LoginUseCase {
    return &LoginUseCase{
        userRepo:    userRepo,
        sessionRepo: sessionRepo,
        jwtService:  jwtService,
    }
}

// Execute はログインを実行します
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
    // 1. メールアドレスでユーザーを検索
    email, err := valueobject.NewEmail(input.Email)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("invalid credentials")
    }

    user, err := uc.userRepo.FindByEmail(ctx, email)
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

    // 4. セッション作成
    sessionID := uuid.New().String()
    accessToken, refreshToken, err := uc.jwtService.GenerateTokenPair(user.ID, user.Email.String(), sessionID)
    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    session := &entity.Session{
        ID:               sessionID,
        UserID:           user.ID,
        RefreshToken:     refreshToken,
        UserAgent:        input.UserAgent,
        IPAddress:        input.IPAddress,
        ExpiresAt:        time.Now().Add(uc.jwtService.GetRefreshTokenExpiry()),
        CreatedAt:        time.Now(),
        LastUsedAt:       time.Now(),
    }

    if err := uc.sessionRepo.Save(ctx, session); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &LoginOutput{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    int(uc.jwtService.GetAccessTokenExpiry().Seconds()),
        User:         user,
    }, nil
}
```

---

## 5. OAuth認証

### 5.1 OAuthフロー

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          OAuth 2.0 Authorization Code Flow                   │
└─────────────────────────────────────────────────────────────────────────────┘

┌────────┐     ┌──────────┐    ┌───────────────┐     ┌────────────────┐
│ Client │     │   API    │    │ OAuth Provider│     │    Database    │
└────┬───┘     └────┬─────┘    │(Google/GitHub)│     └───────┬────────┘
     │              │          └───────┬───────┘             │
     │  1. GET      │                  │                     │
     │  /oauth/google                  │                     │
     │─────────────▶│                  │                     │
     │              │                  │                     │
     │  2. 302 Redirect to Google      │                     │
     │◀─────────────│                  │                     │
     │              │                  │                     │
     │  3. User authorizes on Google   │                     │
     │─────────────────────────────────▶                     │
     │              │                  │                     │
     │  4. Redirect to callback with code                    │
     │◀────────────────────────────────│                     │
     │              │                  │                     │
     │  5. GET /oauth/google/callback?code=xxx               │
     │─────────────▶│                  │                     │
     │              │                  │                     │
     │              │  6. Exchange code for token            │
     │              │─────────────────▶│                     │
     │              │                  │                     │
     │              │  7. Access token │                     │
     │              │◀─────────────────│                     │
     │              │                  │                     │
     │              │  8. Get user info│                     │
     │              │─────────────────▶│                     │
     │              │                  │                     │
     │              │  9. User profile │                     │
     │              │◀─────────────────│                     │
     │              │                  │                     │
     │              │  10. Find/Create user                  │
     │              │────────────────────────────────────────▶
     │              │                  │                     │
     │              │  11. Generate JWT                      │
     │              │                  │                     │
     │  12. Redirect with tokens       │                     │
     │◀─────────────│                  │                     │
```

### 5.2 OAuthプロバイダー設定

```go
// internal/infrastructure/external/oauth/config.go

package oauth

import (
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/github"
    "golang.org/x/oauth2/google"
)

// ProviderConfig はOAuthプロバイダー設定を定義します
type ProviderConfig struct {
    Google GoogleConfig
    GitHub GitHubConfig
}

// GoogleConfig はGoogle OAuth設定を定義します
type GoogleConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
}

// GitHubConfig はGitHub OAuth設定を定義します
type GitHubConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
}

// GetGoogleOAuthConfig はGoogle OAuth設定を返します
func GetGoogleOAuthConfig(cfg GoogleConfig) *oauth2.Config {
    return &oauth2.Config{
        ClientID:     cfg.ClientID,
        ClientSecret: cfg.ClientSecret,
        RedirectURL:  cfg.RedirectURL,
        Scopes: []string{
            "openid",
            "email",
            "profile",
        },
        Endpoint: google.Endpoint,
    }
}

// GetGitHubOAuthConfig はGitHub OAuth設定を返します
func GetGitHubOAuthConfig(cfg GitHubConfig) *oauth2.Config {
    return &oauth2.Config{
        ClientID:     cfg.ClientID,
        ClientSecret: cfg.ClientSecret,
        RedirectURL:  cfg.RedirectURL,
        Scopes: []string{
            "user:email",
            "read:user",
        },
        Endpoint: github.Endpoint,
    }
}
```

### 5.3 OAuthユースケース

```go
// internal/usecase/auth/oauth_login.go

package auth

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"

    "gc-storage/internal/domain/entity"
    "gc-storage/internal/domain/repository"
    "gc-storage/internal/domain/valueobject"
    "gc-storage/internal/infrastructure/external/oauth"
    "gc-storage/pkg/apperror"
    "gc-storage/pkg/jwt"
)

// OAuthLoginInput はOAuthログインの入力を定義します
type OAuthLoginInput struct {
    Provider   string
    Code       string
    State      string
    UserAgent  string
    IPAddress  string
}

// OAuthLoginOutput はOAuthログインの出力を定義します
type OAuthLoginOutput struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int
    User         *entity.User
    IsNewUser    bool
}

// OAuthLoginUseCase はOAuthログインユースケースです
type OAuthLoginUseCase struct {
    userRepo         repository.UserRepository
    oauthAccountRepo repository.OAuthAccountRepository
    sessionRepo      repository.SessionRepository
    oauthService     *oauth.OAuthService
    jwtService       *jwt.JWTService
    txManager        repository.TransactionManager
}

// Execute はOAuthログインを実行します
func (uc *OAuthLoginUseCase) Execute(ctx context.Context, input OAuthLoginInput) (*OAuthLoginOutput, error) {
    // 1. プロバイダーからユーザー情報を取得
    provider := valueobject.OAuthProvider(input.Provider)
    if !provider.IsValid() {
        return nil, apperror.NewValidationError("invalid oauth provider", nil)
    }

    oauthUser, err := uc.oauthService.GetUserInfo(ctx, provider, input.Code)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("failed to authenticate with provider")
    }

    var user *entity.User
    var isNewUser bool

    err = uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 2. 既存のOAuthアカウントを検索
        oauthAccount, err := uc.oauthAccountRepo.FindByProviderAndUserID(ctx, provider, oauthUser.ID)
        if err == nil && oauthAccount != nil {
            // 既存ユーザー
            user, err = uc.userRepo.FindByID(ctx, oauthAccount.UserID)
            if err != nil {
                return err
            }
            return nil
        }

        // 3. メールアドレスで既存ユーザーを検索
        email, _ := valueobject.NewEmail(oauthUser.Email)
        existingUser, err := uc.userRepo.FindByEmail(ctx, email)
        if err == nil && existingUser != nil {
            // 既存ユーザーにOAuthアカウントを追加
            user = existingUser
            oauthAccount = &entity.OAuthAccount{
                ID:             uuid.New(),
                UserID:         user.ID,
                Provider:       provider,
                ProviderUserID: oauthUser.ID,
                CreatedAt:      time.Now(),
                UpdatedAt:      time.Now(),
            }
            return uc.oauthAccountRepo.Create(ctx, oauthAccount)
        }

        // 4. 新規ユーザー作成
        isNewUser = true
        user = &entity.User{
            ID:            uuid.New(),
            Email:         email,
            Name:          oauthUser.Name,
            Status:        entity.UserStatusActive, // OAuthは即座にactive
            EmailVerified: true,                    // OAuthは確認済み
            CreatedAt:     time.Now(),
            UpdatedAt:     time.Now(),
        }

        if err := uc.userRepo.Create(ctx, user); err != nil {
            return err
        }

        oauthAccount = &entity.OAuthAccount{
            ID:             uuid.New(),
            UserID:         user.ID,
            Provider:       provider,
            ProviderUserID: oauthUser.ID,
            CreatedAt:      time.Now(),
            UpdatedAt:      time.Now(),
        }

        return uc.oauthAccountRepo.Create(ctx, oauthAccount)
    })

    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    // 5. ユーザー状態チェック
    if user.Status != entity.UserStatusActive {
        return nil, apperror.NewUnauthorizedError("account is not active")
    }

    // 6. セッション作成
    sessionID := uuid.New().String()
    accessToken, refreshToken, err := uc.jwtService.GenerateTokenPair(user.ID, user.Email.String(), sessionID)
    if err != nil {
        return nil, apperror.NewInternalError(err)
    }

    session := &entity.Session{
        ID:           sessionID,
        UserID:       user.ID,
        RefreshToken: refreshToken,
        UserAgent:    input.UserAgent,
        IPAddress:    input.IPAddress,
        ExpiresAt:    time.Now().Add(uc.jwtService.GetRefreshTokenExpiry()),
        CreatedAt:    time.Now(),
        LastUsedAt:   time.Now(),
    }

    if err := uc.sessionRepo.Save(ctx, session); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    return &OAuthLoginOutput{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    int(uc.jwtService.GetAccessTokenExpiry().Seconds()),
        User:         user,
        IsNewUser:    isNewUser,
    }, nil
}
```

---

## 6. トークンリフレッシュ

### 6.1 リフレッシュユースケース

```go
// internal/usecase/auth/refresh_token.go

package auth

import (
    "context"
    "time"

    "gc-storage/internal/domain/repository"
    "gc-storage/pkg/apperror"
    "gc-storage/pkg/jwt"
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

// RefreshTokenUseCase はトークンリフレッシュユースケースです
type RefreshTokenUseCase struct {
    userRepo         repository.UserRepository
    sessionRepo      repository.SessionRepository
    jwtService       *jwt.JWTService
    jwtBlacklist     service.JWTBlacklistService
}

// Execute はトークンリフレッシュを実行します
func (uc *RefreshTokenUseCase) Execute(ctx context.Context, input RefreshTokenInput) (*RefreshTokenOutput, error) {
    // 1. リフレッシュトークンを検証
    claims, err := uc.jwtService.ValidateRefreshToken(input.RefreshToken)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("invalid refresh token")
    }

    // 2. セッションを検索
    session, err := uc.sessionRepo.FindByID(ctx, claims.SessionID)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("session not found")
    }

    // 3. セッション有効性チェック
    if session.ExpiresAt.Before(time.Now()) {
        return nil, apperror.NewUnauthorizedError("session expired")
    }

    if session.RefreshToken != input.RefreshToken {
        // トークンが一致しない = トークン再利用攻撃の可能性
        // 全セッションを無効化
        uc.sessionRepo.DeleteByUserID(ctx, session.UserID)
        return nil, apperror.NewUnauthorizedError("token reuse detected")
    }

    // 4. ユーザー取得・状態チェック
    user, err := uc.userRepo.FindByID(ctx, session.UserID)
    if err != nil {
        return nil, apperror.NewUnauthorizedError("user not found")
    }

    if user.Status != entity.UserStatusActive {
        return nil, apperror.NewUnauthorizedError("account is not active")
    }

    // 5. 新しいトークンペアを生成（トークンローテーション）
    newAccessToken, newRefreshToken, err := uc.jwtService.GenerateTokenPair(
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
    session.ExpiresAt = time.Now().Add(uc.jwtService.GetRefreshTokenExpiry())

    if err := uc.sessionRepo.Save(ctx, session); err != nil {
        return nil, apperror.NewInternalError(err)
    }

    // 7. 古いアクセストークンをブラックリストに追加（オプション）
    uc.jwtBlacklist.Add(ctx, claims.ID, claims.ExpiresAt.Time)

    return &RefreshTokenOutput{
        AccessToken:  newAccessToken,
        RefreshToken: newRefreshToken,
        ExpiresIn:    int(uc.jwtService.GetAccessTokenExpiry().Seconds()),
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

### 7.2 リセット要求ユースケース

```go
// internal/usecase/auth/request_password_reset.go

package auth

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"

    "gc-storage/internal/domain/entity"
    "gc-storage/internal/domain/repository"
    "gc-storage/internal/domain/service"
    "gc-storage/internal/domain/valueobject"
)

// RequestPasswordResetUseCase はパスワードリセット要求ユースケースです
type RequestPasswordResetUseCase struct {
    userRepo       repository.UserRepository
    resetTokenRepo repository.PasswordResetTokenRepository
    emailService   service.EmailService
}

// Execute はパスワードリセット要求を実行します
func (uc *RequestPasswordResetUseCase) Execute(ctx context.Context, emailStr, ipAddress string) error {
    // 常に200を返す（ユーザー列挙防止）
    email, err := valueobject.NewEmail(emailStr)
    if err != nil {
        return nil
    }

    user, err := uc.userRepo.FindByEmail(ctx, email)
    if err != nil {
        return nil // ユーザーが存在しなくてもエラーを返さない
    }

    // パスワードを持っていないユーザー（OAuth専用）の場合は何もしない
    if user.PasswordHash == "" {
        return nil
    }

    // リセットトークン作成
    token := &entity.PasswordResetToken{
        ID:        uuid.New(),
        UserID:    user.ID,
        Token:     generateSecureToken(),
        ExpiresAt: time.Now().Add(1 * time.Hour),
        CreatedAt: time.Now(),
    }

    if err := uc.resetTokenRepo.Create(ctx, token); err != nil {
        return nil // エラーでも200を返す
    }

    // リセットメール送信（非同期）
    resetURL := fmt.Sprintf("https://app.gc-storage.example.com/reset-password?token=%s", token.Token)
    go uc.emailService.SendPasswordResetEmail(ctx, emailStr, user.Name, resetURL, ipAddress)

    return nil
}
```

---

## 8. ログアウト

### 8.1 ログアウトユースケース

```go
// internal/usecase/auth/logout.go

package auth

import (
    "context"

    "gc-storage/internal/domain/repository"
    "gc-storage/internal/domain/service"
    "gc-storage/pkg/jwt"
)

// LogoutUseCase はログアウトユースケースです
type LogoutUseCase struct {
    sessionRepo  repository.SessionRepository
    jwtBlacklist service.JWTBlacklistService
}

// Execute はログアウトを実行します
func (uc *LogoutUseCase) Execute(ctx context.Context, sessionID string, accessTokenClaims *jwt.AccessTokenClaims) error {
    // 1. セッションを削除
    if err := uc.sessionRepo.Delete(ctx, sessionID); err != nil {
        // エラーでも続行
    }

    // 2. アクセストークンをブラックリストに追加
    if accessTokenClaims != nil {
        uc.jwtBlacklist.Add(ctx, accessTokenClaims.ID, accessTokenClaims.ExpiresAt.Time)
    }

    return nil
}

// ExecuteAll は全セッションからログアウトを実行します
func (uc *LogoutUseCase) ExecuteAll(ctx context.Context, userID uuid.UUID) error {
    return uc.sessionRepo.DeleteByUserID(ctx, userID)
}
```

---

## 9. APIハンドラー

### 9.1 認証ハンドラー

```go
// internal/interface/handler/auth_handler.go

package handler

import (
    "net/http"
    "time"

    "github.com/labstack/echo/v4"

    "gc-storage/internal/interface/dto/request"
    "gc-storage/internal/interface/dto/response"
    "gc-storage/internal/interface/middleware"
    "gc-storage/internal/interface/presenter"
    "gc-storage/internal/usecase/auth"
    "gc-storage/pkg/apperror"
)

// AuthHandler は認証関連のHTTPハンドラーです
type AuthHandler struct {
    registerUC          *auth.RegisterUseCase
    loginUC             *auth.LoginUseCase
    oauthLoginUC        *auth.OAuthLoginUseCase
    refreshTokenUC      *auth.RefreshTokenUseCase
    logoutUC            *auth.LogoutUseCase
    verifyEmailUC       *auth.VerifyEmailUseCase
    requestResetUC      *auth.RequestPasswordResetUseCase
    resetPasswordUC     *auth.ResetPasswordUseCase
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

    output, err := h.registerUC.Execute(c.Request().Context(), auth.RegisterInput{
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

    output, err := h.loginUC.Execute(c.Request().Context(), auth.LoginInput{
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

    output, err := h.refreshTokenUC.Execute(c.Request().Context(), auth.RefreshTokenInput{
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

    if err := h.logoutUC.Execute(c.Request().Context(), sessionID, nil); err != nil {
        // エラーでも成功扱い
    }

    // Cookieを削除
    h.clearRefreshTokenCookie(c)

    return presenter.OK(c, map[string]string{"message": "logged out successfully"})
}

// OAuthRedirect はOAuth認証へリダイレクトします
// GET /api/v1/auth/oauth/:provider
func (h *AuthHandler) OAuthRedirect(c echo.Context) error {
    provider := c.Param("provider")

    url, state, err := h.oauthLoginUC.GetAuthURL(provider)
    if err != nil {
        return err
    }

    // stateをCookieに保存（CSRF対策）
    c.SetCookie(&http.Cookie{
        Name:     "oauth_state",
        Value:    state,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   600, // 10分
    })

    return c.Redirect(http.StatusFound, url)
}

// OAuthCallback はOAuthコールバックを処理します
// GET /api/v1/auth/oauth/:provider/callback
func (h *AuthHandler) OAuthCallback(c echo.Context) error {
    provider := c.Param("provider")
    code := c.QueryParam("code")
    state := c.QueryParam("state")

    // state検証
    stateCookie, err := c.Cookie("oauth_state")
    if err != nil || stateCookie.Value != state {
        return apperror.NewUnauthorizedError("invalid state parameter")
    }

    output, err := h.oauthLoginUC.Execute(c.Request().Context(), auth.OAuthLoginInput{
        Provider:  provider,
        Code:      code,
        State:     state,
        UserAgent: c.Request().UserAgent(),
        IPAddress: c.RealIP(),
    })
    if err != nil {
        return err
    }

    // state Cookieを削除
    c.SetCookie(&http.Cookie{
        Name:   "oauth_state",
        Value:  "",
        Path:   "/",
        MaxAge: -1,
    })

    // リフレッシュトークンをCookieに設定
    h.setRefreshTokenCookie(c, output.RefreshToken)

    // フロントエンドにリダイレクト
    redirectURL := fmt.Sprintf(
        "https://app.gc-storage.example.com/oauth/callback?access_token=%s&expires_in=%d&is_new_user=%t",
        output.AccessToken,
        output.ExpiresIn,
        output.IsNewUser,
    )

    return c.Redirect(http.StatusFound, redirectURL)
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
| ログイン | メール/パスワードでログインできる |
| OAuth認証 | Google/GitHubでログインできる |
| トークンリフレッシュ | リフレッシュトークンで新しいトークンを取得できる |
| ログアウト | セッションを終了できる |
| パスワードリセット | メールでリセットリンクを受け取り、パスワードを変更できる |
| セッション管理 | アクティブセッション一覧表示・個別失効ができる |

### 10.2 セキュリティ要件

| 項目 | 基準 |
|------|------|
| パスワードハッシュ | bcrypt cost 12 |
| Access Token有効期限 | 15分 |
| Refresh Token有効期限 | 7日 |
| Refresh Token保存 | HttpOnly Cookie + Redis |
| トークンローテーション | リフレッシュ時に新しいペアを発行 |
| セッション上限 | 1ユーザー最大10セッション |
| CSRF対策 | OAuth state検証 |
| レート制限 | ログイン: 10 req/min/IP |

### 10.3 チェックリスト

- [ ] ユーザー登録が正常に動作する
- [ ] メール確認が正常に動作する
- [ ] メール/パスワードログインが正常に動作する
- [ ] Google OAuthログインが正常に動作する
- [ ] GitHub OAuthログインが正常に動作する
- [ ] トークンリフレッシュが正常に動作する
- [ ] トークンローテーションが実装されている
- [ ] ログアウトでセッションが削除される
- [ ] パスワードリセット要求でメールが送信される
- [ ] パスワードリセット完了で全セッションが失効する
- [ ] 無効なトークンが拒否される
- [ ] ブラックリストに追加されたトークンが拒否される
- [ ] レート制限が正しく動作する
- [ ] CSRF対策（state検証）が動作する

---

## 関連ドキュメント

- [infra-redis.md](./infra-redis.md) - セッションストア・JWTブラックリスト
- [infra-email.md](./infra-email.md) - 認証メール・リセットメール
- [infra-api.md](./infra-api.md) - 認証ミドルウェア
- [SECURITY.md](../02-architecture/SECURITY.md) - セキュリティ設計
- [user.md](../03-domains/user.md) - ユーザードメイン
