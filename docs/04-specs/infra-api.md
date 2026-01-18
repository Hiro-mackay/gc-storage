# API インフラストラクチャ仕様書

## 概要

本ドキュメントでは、GC StorageにおけるEchoフレームワークの設定、ミドルウェア、バリデーション、エラーハンドリングの実装仕様を定義します。

**関連アーキテクチャ:**
- [API.md](../02-architecture/API.md) - API設計方針
- [BACKEND.md](../02-architecture/BACKEND.md) - バックエンド設計
- [SECURITY.md](../02-architecture/SECURITY.md) - セキュリティ設計

---

## 1. Echoフレームワーク設定

### 1.1 サーバー構成

```go
// backend/internal/interface/server/server.go

package server

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

// Config はサーバー設定を定義します
type Config struct {
    Host            string        // ホスト (default: "")
    Port            int           // ポート (default: 8080)
    ReadTimeout     time.Duration // 読み取りタイムアウト (default: 30s)
    WriteTimeout    time.Duration // 書き込みタイムアウト (default: 30s)
    ShutdownTimeout time.Duration // シャットダウンタイムアウト (default: 10s)
    BodyLimit       string        // リクエストボディ制限 (default: "10MB")
    Debug           bool          // デバッグモード
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() Config {
    return Config{
        Host:            "",
        Port:            8080,
        ReadTimeout:     30 * time.Second,
        WriteTimeout:    30 * time.Second,
        ShutdownTimeout: 10 * time.Second,
        BodyLimit:       "10MB",
        Debug:           false,
    }
}

// Server はHTTPサーバーを提供します
type Server struct {
    echo   *echo.Echo
    config Config
}

// NewServer は新しいServerを作成します
func NewServer(cfg Config) *Server {
    e := echo.New()

    // 基本設定
    e.Debug = cfg.Debug
    e.HideBanner = true
    e.HidePort = true

    // サーバーのタイムアウト設定
    e.Server.ReadTimeout = cfg.ReadTimeout
    e.Server.WriteTimeout = cfg.WriteTimeout

    return &Server{
        echo:   e,
        config: cfg,
    }
}

// Echo は内部のecho.Echoを返します
func (s *Server) Echo() *echo.Echo {
    return s.echo
}

// Start はサーバーを開始します
func (s *Server) Start() error {
    addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
    return s.echo.Start(addr)
}

// Shutdown はサーバーを停止します
func (s *Server) Shutdown(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
    defer cancel()
    return s.echo.Shutdown(ctx)
}
```

### 1.2 ディレクトリ構成

```
backend/internal/interface/
├── server/
│   └── server.go             # サーバー設定
├── router/
│   ├── router.go             # ルーター設定
│   └── routes.go             # ルート定義
├── handler/
│   ├── auth_handler.go       # 認証ハンドラー
│   ├── file_handler.go       # ファイルハンドラー
│   ├── folder_handler.go     # フォルダハンドラー
│   ├── group_handler.go      # グループハンドラー
│   └── health_handler.go     # ヘルスチェック
├── middleware/
│   ├── auth.go               # 認証ミドルウェア
│   ├── cors.go               # CORSミドルウェア
│   ├── error_handler.go      # エラーハンドラー
│   ├── logger.go             # ロギングミドルウェア
│   ├── rate_limit.go         # レート制限
│   ├── request_id.go         # リクエストID
│   ├── recover.go            # パニックリカバリー
│   └── security.go           # セキュリティヘッダー
├── presenter/
│   └── response.go           # レスポンス生成
├── dto/
│   ├── request/              # リクエストDTO
│   └── response/             # レスポンスDTO
└── validator/
    └── validator.go          # カスタムバリデーター
```

---

## 2. ルーター設定

### 2.1 ルーター構成

```go
// backend/internal/interface/router/router.go

package router

import (
    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/handler"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
)

// Config はルーター設定を定義します
type Config struct {
    AuthMiddleware        echo.MiddlewareFunc
    RateLimitMiddleware   *middleware.RateLimitMiddleware
    PermissionMiddleware  *middleware.PermissionMiddleware
}

// Router はルート設定を管理します
type Router struct {
    echo    *echo.Echo
    config  Config
    handlers *Handlers
}

// Handlers はすべてのハンドラーを保持します
type Handlers struct {
    Health     *handler.HealthHandler
    Auth       *handler.AuthHandler
    File       *handler.FileHandler
    Folder     *handler.FolderHandler
    Group      *handler.GroupHandler
    Permission *handler.PermissionHandler
    Share      *handler.ShareHandler
}

// NewRouter は新しいRouterを作成します
func NewRouter(e *echo.Echo, cfg Config, h *Handlers) *Router {
    return &Router{
        echo:     e,
        config:   cfg,
        handlers: h,
    }
}

// Setup はルートを設定します
func (r *Router) Setup() {
    // グローバルミドルウェア
    r.setupGlobalMiddleware()

    // APIバージョングループ
    api := r.echo.Group("/api/v1")

    // 公開エンドポイント
    r.setupPublicRoutes(api)

    // 認証必須エンドポイント
    r.setupAuthenticatedRoutes(api)
}

func (r *Router) setupGlobalMiddleware() {
    // ミドルウェアの順序は重要
    r.echo.Use(middleware.RequestID())
    r.echo.Use(middleware.Logger())
    r.echo.Use(middleware.Recover())
    r.echo.Use(middleware.SecurityHeaders())
    r.echo.Use(middleware.CORS())
}

func (r *Router) setupPublicRoutes(api *echo.Group) {
    // ヘルスチェック
    r.echo.GET("/health", r.handlers.Health.Check)
    r.echo.GET("/ready", r.handlers.Health.Ready)

    // 認証
    auth := api.Group("/auth")
    auth.POST("/register", r.handlers.Auth.Register,
        r.config.RateLimitMiddleware.ByIP(middleware.RateLimitAuthSignup))
    auth.POST("/login", r.handlers.Auth.Login,
        r.config.RateLimitMiddleware.ByIP(middleware.RateLimitAuthLogin))
    auth.POST("/refresh", r.handlers.Auth.Refresh)
    auth.GET("/oauth/:provider", r.handlers.Auth.OAuthRedirect)
    auth.GET("/oauth/:provider/callback", r.handlers.Auth.OAuthCallback)

    // 共有リンクアクセス
    share := api.Group("/share")
    share.GET("/:token", r.handlers.Share.Access,
        r.config.RateLimitMiddleware.ByIP(middleware.RateLimitShareAccess))
    share.POST("/:token/verify", r.handlers.Share.VerifyPassword)
}

func (r *Router) setupAuthenticatedRoutes(api *echo.Group) {
    // 認証ミドルウェアを適用
    authed := api.Group("", r.config.AuthMiddleware)
    authed.Use(r.config.RateLimitMiddleware.ByUser(middleware.RateLimitAPIDefault))

    // 認証
    authed.POST("/auth/logout", r.handlers.Auth.Logout)
    authed.DELETE("/auth/sessions/:id", r.handlers.Auth.RevokeSession)
    authed.GET("/auth/sessions", r.handlers.Auth.ListSessions)

    // ユーザー
    authed.GET("/me", r.handlers.Auth.Me)
    authed.PATCH("/me", r.handlers.Auth.UpdateProfile)
    authed.PATCH("/me/password", r.handlers.Auth.UpdatePassword)

    // ファイル
    r.setupFileRoutes(authed)

    // フォルダ
    r.setupFolderRoutes(authed)

    // グループ
    r.setupGroupRoutes(authed)

    // 検索
    authed.GET("/search", r.handlers.File.Search,
        r.config.RateLimitMiddleware.ByUser(middleware.RateLimitAPISearch))
}

func (r *Router) setupFileRoutes(g *echo.Group) {
    files := g.Group("/files")
    perm := r.config.PermissionMiddleware

    // アップロード
    files.POST("/upload", r.handlers.File.InitUpload,
        r.config.RateLimitMiddleware.ByUser(middleware.RateLimitAPIUpload))
    files.POST("/upload/multipart", r.handlers.File.InitMultipartUpload,
        r.config.RateLimitMiddleware.ByUser(middleware.RateLimitAPIUpload))
    files.GET("/upload/:sessionId/part-url", r.handlers.File.GetPartURL)
    files.POST("/upload/:sessionId/complete", r.handlers.File.CompleteUpload)
    files.DELETE("/upload/:sessionId", r.handlers.File.CancelUpload)

    // CRUD
    files.GET("/:id", r.handlers.File.Get,
        perm.RequirePermission("file", "file:read", "id"))
    files.GET("/:id/download", r.handlers.File.Download,
        perm.RequirePermission("file", "file:read", "id"))
    files.GET("/:id/preview", r.handlers.File.Preview,
        perm.RequirePermission("file", "file:read", "id"))
    files.PATCH("/:id/rename", r.handlers.File.Rename,
        perm.RequirePermission("file", "file:rename", "id"))
    files.POST("/:id/move", r.handlers.File.Move,
        perm.RequirePermission("file", "file:move", "id"))
    files.POST("/:id/copy", r.handlers.File.Copy,
        perm.RequirePermission("file", "file:read", "id"))
    files.DELETE("/:id", r.handlers.File.Trash,
        perm.RequirePermission("file", "file:delete", "id"))
    files.POST("/:id/restore", r.handlers.File.Restore,
        perm.RequirePermission("file", "file:restore", "id"))
    files.DELETE("/:id/permanent", r.handlers.File.PermanentDelete,
        perm.RequirePermission("file", "file:permanent_delete", "id"))

    // バージョン
    files.GET("/:id/versions", r.handlers.File.ListVersions,
        perm.RequirePermission("file", "file:read", "id"))
    files.POST("/:id/versions/:version/restore", r.handlers.File.RestoreVersion,
        perm.RequirePermission("file", "file:write", "id"))

    // 共有
    files.POST("/:id/share", r.handlers.Share.CreateFileLink,
        perm.RequirePermission("file", "file:share", "id"))

    // 権限
    files.GET("/:id/permissions", r.handlers.Permission.ListFilePermissions,
        perm.RequirePermission("file", "permission:read", "id"))
    files.POST("/:id/permissions", r.handlers.Permission.GrantFilePermission,
        perm.RequirePermission("file", "permission:grant", "id"))
    files.DELETE("/:id/permissions/:permissionId", r.handlers.Permission.RevokeFilePermission,
        perm.RequirePermission("file", "permission:revoke", "id"))
}

func (r *Router) setupFolderRoutes(g *echo.Group) {
    folders := g.Group("/folders")
    perm := r.config.PermissionMiddleware

    folders.POST("", r.handlers.Folder.Create)
    folders.GET("/:id", r.handlers.Folder.Get,
        perm.RequirePermission("folder", "folder:read", "id"))
    folders.GET("/:id/contents", r.handlers.Folder.ListContents,
        perm.RequirePermission("folder", "folder:read", "id"))
    folders.PATCH("/:id/rename", r.handlers.Folder.Rename,
        perm.RequirePermission("folder", "folder:rename", "id"))
    folders.POST("/:id/move", r.handlers.Folder.Move,
        perm.RequirePermission("folder", "folder:move", "id"))
    folders.DELETE("/:id", r.handlers.Folder.Trash,
        perm.RequirePermission("folder", "folder:delete", "id"))
    folders.POST("/:id/restore", r.handlers.Folder.Restore,
        perm.RequirePermission("folder", "folder:restore", "id"))

    // 共有
    folders.POST("/:id/share", r.handlers.Share.CreateFolderLink,
        perm.RequirePermission("folder", "folder:share", "id"))

    // 権限
    folders.GET("/:id/permissions", r.handlers.Permission.ListFolderPermissions,
        perm.RequirePermission("folder", "permission:read", "id"))
    folders.POST("/:id/permissions", r.handlers.Permission.GrantFolderPermission,
        perm.RequirePermission("folder", "permission:grant", "id"))
}

func (r *Router) setupGroupRoutes(g *echo.Group) {
    groups := g.Group("/groups")
    perm := r.config.PermissionMiddleware

    groups.POST("", r.handlers.Group.Create)
    groups.GET("", r.handlers.Group.List)
    groups.GET("/:id", r.handlers.Group.Get,
        perm.RequireGroupPermission("group:read", "id"))
    groups.PATCH("/:id", r.handlers.Group.Update,
        perm.RequireGroupPermission("group:update", "id"))
    groups.DELETE("/:id", r.handlers.Group.Delete,
        perm.RequireGroupPermission("group:delete", "id"))

    // メンバー
    groups.GET("/:id/members", r.handlers.Group.ListMembers,
        perm.RequireGroupPermission("group:member:read", "id"))
    groups.POST("/:id/members", r.handlers.Group.AddMember,
        perm.RequireGroupPermission("group:member:add", "id"))
    groups.DELETE("/:id/members/:userId", r.handlers.Group.RemoveMember,
        perm.RequireGroupPermission("group:member:remove", "id"))
    groups.PATCH("/:id/members/:userId/role", r.handlers.Group.UpdateMemberRole,
        perm.RequireGroupPermission("group:member:role", "id"))

    // 招待
    groups.POST("/:id/invitations", r.handlers.Group.CreateInvitation,
        perm.RequireGroupPermission("group:member:add", "id"))
    groups.DELETE("/:id/invitations/:invitationId", r.handlers.Group.CancelInvitation,
        perm.RequireGroupPermission("group:member:add", "id"))
}
```

---

## 3. ミドルウェア

### 3.1 リクエストIDミドルウェア

```go
// backend/internal/interface/middleware/request_id.go

package middleware

import (
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
)

const (
    HeaderRequestID = "X-Request-ID"
    ContextKeyRequestID = "request_id"
)

// RequestID はリクエストIDを生成・設定するミドルウェアを返します
func RequestID() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            requestID := c.Request().Header.Get(HeaderRequestID)
            if requestID == "" {
                requestID = uuid.New().String()
            }

            c.Set(ContextKeyRequestID, requestID)
            c.Response().Header().Set(HeaderRequestID, requestID)

            return next(c)
        }
    }
}

// GetRequestID はコンテキストからリクエストIDを取得します
func GetRequestID(c echo.Context) string {
    if id, ok := c.Get(ContextKeyRequestID).(string); ok {
        return id
    }
    return ""
}
```

### 3.2 ロギングミドルウェア

```go
// backend/internal/interface/middleware/logger.go

package middleware

import (
    "log/slog"
    "time"

    "github.com/labstack/echo/v4"
)

// Logger はリクエストロギングミドルウェアを返します
func Logger() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            start := time.Now()

            err := next(c)

            latency := time.Since(start)

            // 構造化ログ出力
            slog.Info("request",
                "request_id", GetRequestID(c),
                "method", c.Request().Method,
                "uri", c.Request().RequestURI,
                "status", c.Response().Status,
                "latency_ms", latency.Milliseconds(),
                "ip", c.RealIP(),
                "user_agent", c.Request().UserAgent(),
                "bytes_in", c.Request().ContentLength,
                "bytes_out", c.Response().Size,
            )

            return err
        }
    }
}
```

### 3.3 パニックリカバリーミドルウェア

```go
// backend/internal/interface/middleware/recover.go

package middleware

import (
    "fmt"
    "log/slog"
    "runtime"

    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// Recover はパニックをリカバーするミドルウェアを返します
func Recover() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            defer func() {
                if r := recover(); r != nil {
                    // スタックトレースを取得
                    buf := make([]byte, 4096)
                    n := runtime.Stack(buf, false)
                    stackTrace := string(buf[:n])

                    slog.Error("panic recovered",
                        "request_id", GetRequestID(c),
                        "error", fmt.Sprintf("%v", r),
                        "stack", stackTrace,
                    )

                    // 500エラーを返す
                    c.Error(apperror.NewInternalError(fmt.Errorf("internal server error")))
                }
            }()

            return next(c)
        }
    }
}
```

### 3.4 CORSミドルウェア

```go
// backend/internal/interface/middleware/cors.go

package middleware

import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

// CORSConfig はCORS設定を定義します
type CORSConfig struct {
    AllowOrigins     []string
    AllowMethods     []string
    AllowHeaders     []string
    AllowCredentials bool
    MaxAge           int
}

// DefaultCORSConfig はデフォルトCORS設定を返します
func DefaultCORSConfig() CORSConfig {
    return CORSConfig{
        AllowOrigins:     []string{"http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Content-Type", "Authorization", "X-Request-ID"},
        AllowCredentials: true,
        MaxAge:           86400, // 24時間
    }
}

// CORS はCORSミドルウェアを返します
func CORS() echo.MiddlewareFunc {
    cfg := DefaultCORSConfig()
    return CORSWithConfig(cfg)
}

// CORSWithConfig は設定付きCORSミドルウェアを返します
func CORSWithConfig(cfg CORSConfig) echo.MiddlewareFunc {
    return middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins:     cfg.AllowOrigins,
        AllowMethods:     cfg.AllowMethods,
        AllowHeaders:     cfg.AllowHeaders,
        AllowCredentials: cfg.AllowCredentials,
        MaxAge:           cfg.MaxAge,
    })
}
```

### 3.5 セキュリティヘッダーミドルウェア

```go
// backend/internal/interface/middleware/security.go

package middleware

import (
    "github.com/labstack/echo/v4"
)

// SecurityHeaders はセキュリティヘッダーを設定するミドルウェアを返します
func SecurityHeaders() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // XSS対策
            c.Response().Header().Set("X-Content-Type-Options", "nosniff")
            c.Response().Header().Set("X-XSS-Protection", "1; mode=block")

            // クリックジャッキング対策
            c.Response().Header().Set("X-Frame-Options", "DENY")

            // HTTPS強制（本番環境のみ）
            // c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

            // CSP
            c.Response().Header().Set("Content-Security-Policy", "default-src 'self'")

            // Referrer Policy
            c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

            // Permissions Policy
            c.Response().Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

            return next(c)
        }
    }
}
```

### 3.6 認証ミドルウェア

```go
// backend/internal/interface/middleware/auth.go

package middleware

import (
    "strings"

    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

const (
    ContextKeyUserID    = "user_id"
    ContextKeySessionID = "session_id"
)

// AuthMiddleware は認証ミドルウェアを提供します
type AuthMiddleware struct {
    jwtService      service.JWTService
    blacklistService service.JWTBlacklistService
}

// NewAuthMiddleware は新しいAuthMiddlewareを作成します
func NewAuthMiddleware(jwtSvc service.JWTService, blacklistSvc service.JWTBlacklistService) *AuthMiddleware {
    return &AuthMiddleware{
        jwtService:      jwtSvc,
        blacklistService: blacklistSvc,
    }
}

// Authenticate は認証ミドルウェアを返します
func (m *AuthMiddleware) Authenticate() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Authorizationヘッダーを取得
            authHeader := c.Request().Header.Get("Authorization")
            if authHeader == "" {
                return apperror.NewUnauthorizedError("authorization header required")
            }

            // Bearer トークンを抽出
            parts := strings.SplitN(authHeader, " ", 2)
            if len(parts) != 2 || parts[0] != "Bearer" {
                return apperror.NewUnauthorizedError("invalid authorization header format")
            }

            token := parts[1]

            // トークンを検証
            claims, err := m.jwtService.ValidateAccessToken(token)
            if err != nil {
                return apperror.NewUnauthorizedError("invalid or expired token")
            }

            // ブラックリストチェック
            isBlacklisted, err := m.blacklistService.IsBlacklisted(c.Request().Context(), claims.ID)
            if err != nil {
                return apperror.NewInternalError(err)
            }
            if isBlacklisted {
                return apperror.NewUnauthorizedError("token has been revoked")
            }

            // コンテキストにユーザー情報を設定
            c.Set(ContextKeyUserID, claims.UserID)
            c.Set(ContextKeySessionID, claims.SessionID)

            return next(c)
        }
    }
}

// GetUserID はコンテキストからユーザーIDを取得します
func GetUserID(c echo.Context) string {
    if id, ok := c.Get(ContextKeyUserID).(string); ok {
        return id
    }
    return ""
}

// GetSessionID はコンテキストからセッションIDを取得します
func GetSessionID(c echo.Context) string {
    if id, ok := c.Get(ContextKeySessionID).(string); ok {
        return id
    }
    return ""
}
```

---

## 4. バリデーション

### 4.1 カスタムバリデーター

```go
// backend/internal/interface/validator/validator.go

package validator

import (
    "net/http"
    "regexp"
    "strings"

    "github.com/go-playground/validator/v10"
    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CustomValidator はEcho用のカスタムバリデーターです
type CustomValidator struct {
    validator *validator.Validate
}

// NewCustomValidator は新しいCustomValidatorを作成します
func NewCustomValidator() *CustomValidator {
    v := validator.New()

    // カスタムバリデーション登録
    v.RegisterValidation("filename", validateFileName)
    v.RegisterValidation("foldername", validateFolderName)
    v.RegisterValidation("password", validatePassword)

    return &CustomValidator{validator: v}
}

// Validate はリクエストを検証します
func (cv *CustomValidator) Validate(i interface{}) error {
    if err := cv.validator.Struct(i); err != nil {
        return cv.formatValidationErrors(err)
    }
    return nil
}

// formatValidationErrors はバリデーションエラーをフォーマットします
func (cv *CustomValidator) formatValidationErrors(err error) error {
    validationErrors, ok := err.(validator.ValidationErrors)
    if !ok {
        return apperror.NewValidationError(err.Error(), nil)
    }

    details := make([]apperror.FieldError, 0, len(validationErrors))
    for _, e := range validationErrors {
        details = append(details, apperror.FieldError{
            Field:   toSnakeCase(e.Field()),
            Message: getValidationMessage(e),
        })
    }

    return apperror.NewValidationError("validation failed", details)
}

// validateFileName はファイル名のバリデーション
func validateFileName(fl validator.FieldLevel) bool {
    name := fl.Field().String()
    if name == "" {
        return false
    }

    // 禁止文字チェック: / \ : * ? " < > |
    invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
    if invalidChars.MatchString(name) {
        return false
    }

    // 隠しファイルや特殊ファイル名のチェック
    if strings.HasPrefix(name, ".") || name == "." || name == ".." {
        return false
    }

    return len(name) <= 255
}

// validateFolderName はフォルダ名のバリデーション
func validateFolderName(fl validator.FieldLevel) bool {
    name := fl.Field().String()
    if name == "" {
        return false
    }

    // 禁止文字チェック
    invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
    return !invalidChars.MatchString(name) && len(name) <= 255
}

// validatePassword はパスワードのバリデーション
func validatePassword(fl validator.FieldLevel) bool {
    password := fl.Field().String()

    if len(password) < 8 || len(password) > 256 {
        return false
    }

    // 英大文字、英小文字、数字のうち2種以上
    var hasUpper, hasLower, hasDigit bool
    for _, char := range password {
        switch {
        case 'A' <= char && char <= 'Z':
            hasUpper = true
        case 'a' <= char && char <= 'z':
            hasLower = true
        case '0' <= char && char <= '9':
            hasDigit = true
        }
    }

    count := 0
    if hasUpper {
        count++
    }
    if hasLower {
        count++
    }
    if hasDigit {
        count++
    }

    return count >= 2
}

// getValidationMessage はバリデーションエラーメッセージを返します
func getValidationMessage(e validator.FieldError) string {
    switch e.Tag() {
    case "required":
        return "this field is required"
    case "email":
        return "must be a valid email address"
    case "min":
        return "must be at least " + e.Param() + " characters"
    case "max":
        return "must be at most " + e.Param() + " characters"
    case "uuid":
        return "must be a valid UUID"
    case "filename":
        return "must be a valid file name (no special characters)"
    case "foldername":
        return "must be a valid folder name (no special characters)"
    case "password":
        return "must be 8-256 characters with at least 2 of: uppercase, lowercase, digit"
    default:
        return "validation failed"
    }
}

// toSnakeCase はPascalCase/camelCaseをsnake_caseに変換します
func toSnakeCase(str string) string {
    var result []rune
    for i, r := range str {
        if i > 0 && 'A' <= r && r <= 'Z' {
            result = append(result, '_')
        }
        result = append(result, r)
    }
    return strings.ToLower(string(result))
}
```

### 4.2 リクエストDTOの例

```go
// backend/internal/interface/dto/request/file.go

package request

// InitUploadRequest はアップロード開始リクエスト
type InitUploadRequest struct {
    Name     string  `json:"name" validate:"required,filename,max=255"`
    FolderID *string `json:"folder_id" validate:"omitempty,uuid"`
    Size     int64   `json:"size" validate:"required,min=0,max=5368709120"`
    MimeType string  `json:"mime_type" validate:"required"`
}

// RenameFileRequest はファイル名変更リクエスト
type RenameFileRequest struct {
    Name string `json:"name" validate:"required,filename,max=255"`
}

// MoveFileRequest はファイル移動リクエスト
type MoveFileRequest struct {
    DestinationFolderID string `json:"destination_folder_id" validate:"required,uuid"`
}
```

```go
// backend/internal/interface/dto/request/auth.go

package request

// RegisterRequest はユーザー登録リクエスト
type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email,max=255"`
    Password string `json:"password" validate:"required,password"`
    Name     string `json:"name" validate:"required,min=1,max=100"`
}

// LoginRequest はログインリクエスト
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
```

---

## 5. エラーハンドリング

### 5.1 アプリケーションエラー

```go
// pkg/apperror/error.go

package apperror

import (
    "fmt"
    "net/http"
)

// ErrorCode はエラーコードを表します
type ErrorCode string

const (
    CodeValidationError   ErrorCode = "VALIDATION_ERROR"
    CodeInvalidRequest    ErrorCode = "INVALID_REQUEST"
    CodeUnauthorized      ErrorCode = "UNAUTHORIZED"
    CodeTokenExpired      ErrorCode = "TOKEN_EXPIRED"
    CodeForbidden         ErrorCode = "FORBIDDEN"
    CodeQuotaExceeded     ErrorCode = "QUOTA_EXCEEDED"
    CodeNotFound          ErrorCode = "NOT_FOUND"
    CodeConflict          ErrorCode = "CONFLICT"
    CodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
    CodeInternalError     ErrorCode = "INTERNAL_ERROR"
    CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// AppError はアプリケーションエラーを表します
type AppError struct {
    Code       ErrorCode    `json:"code"`
    Message    string       `json:"message"`
    Details    []FieldError `json:"details,omitempty"`
    HTTPStatus int          `json:"-"`
    Err        error        `json:"-"`
}

// FieldError はフィールドエラーを表します
type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

// Error はerrorインターフェースを実装します
func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap は元のエラーを返します
func (e *AppError) Unwrap() error {
    return e.Err
}

// コンストラクタ関数

func NewValidationError(message string, details []FieldError) *AppError {
    return &AppError{
        Code:       CodeValidationError,
        Message:    message,
        Details:    details,
        HTTPStatus: http.StatusBadRequest,
    }
}

func NewUnauthorizedError(message string) *AppError {
    return &AppError{
        Code:       CodeUnauthorized,
        Message:    message,
        HTTPStatus: http.StatusUnauthorized,
    }
}

func NewForbiddenError(message string) *AppError {
    return &AppError{
        Code:       CodeForbidden,
        Message:    message,
        HTTPStatus: http.StatusForbidden,
    }
}

func NewNotFoundError(resource string) *AppError {
    return &AppError{
        Code:       CodeNotFound,
        Message:    fmt.Sprintf("%s not found", resource),
        HTTPStatus: http.StatusNotFound,
    }
}

func NewConflictError(message string) *AppError {
    return &AppError{
        Code:       CodeConflict,
        Message:    message,
        HTTPStatus: http.StatusConflict,
    }
}

func NewTooManyRequestsError(message string) *AppError {
    return &AppError{
        Code:       CodeRateLimitExceeded,
        Message:    message,
        HTTPStatus: http.StatusTooManyRequests,
    }
}

func NewInternalError(err error) *AppError {
    return &AppError{
        Code:       CodeInternalError,
        Message:    "internal server error",
        HTTPStatus: http.StatusInternalServerError,
        Err:        err,
    }
}
```

### 5.2 エラーハンドラー

```go
// backend/internal/interface/middleware/error_handler.go

package middleware

import (
    "errors"
    "log/slog"
    "net/http"

    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ErrorResponse はエラーレスポンス構造を定義します
type ErrorResponse struct {
    Error struct {
        Code    string                 `json:"code"`
        Message string                 `json:"message"`
        Details []apperror.FieldError `json:"details,omitempty"`
    } `json:"error"`
    Meta interface{} `json:"meta"`
}

// CustomHTTPErrorHandler はカスタムエラーハンドラーです
func CustomHTTPErrorHandler(err error, c echo.Context) {
    if c.Response().Committed {
        return
    }

    var appErr *apperror.AppError
    if errors.As(err, &appErr) {
        // AppErrorの場合
        response := ErrorResponse{}
        response.Error.Code = string(appErr.Code)
        response.Error.Message = appErr.Message
        response.Error.Details = appErr.Details

        // 内部エラーの場合はログ出力
        if appErr.HTTPStatus >= 500 {
            slog.Error("internal error",
                "request_id", GetRequestID(c),
                "error", appErr.Error(),
            )
        }

        c.JSON(appErr.HTTPStatus, response)
        return
    }

    // Echo HTTPErrorの場合
    var he *echo.HTTPError
    if errors.As(err, &he) {
        response := ErrorResponse{}
        response.Error.Code = http.StatusText(he.Code)
        response.Error.Message = fmt.Sprintf("%v", he.Message)

        c.JSON(he.Code, response)
        return
    }

    // 未知のエラー
    slog.Error("unknown error",
        "request_id", GetRequestID(c),
        "error", err.Error(),
    )

    response := ErrorResponse{}
    response.Error.Code = "INTERNAL_ERROR"
    response.Error.Message = "internal server error"

    c.JSON(http.StatusInternalServerError, response)
}
```

---

## 6. レスポンス生成

### 6.1 統一レスポンス形式

```go
// backend/internal/interface/presenter/response.go

package presenter

import (
    "net/http"

    "github.com/labstack/echo/v4"
)

// Response は統一レスポンス構造を定義します
type Response struct {
    Data interface{} `json:"data"`
    Meta interface{} `json:"meta"`
}

// Pagination はページネーション情報を定義します
type Pagination struct {
    Page       int  `json:"page"`
    PerPage    int  `json:"per_page"`
    TotalItems int  `json:"total_items"`
    TotalPages int  `json:"total_pages"`
    HasNext    bool `json:"has_next"`
    HasPrev    bool `json:"has_prev"`
}

// Meta はメタ情報を定義します
type Meta struct {
    Message    string      `json:"message,omitempty"`
    Pagination *Pagination `json:"pagination,omitempty"`
}

// OK は成功レスポンスを返します
func OK(c echo.Context, data interface{}) error {
    return c.JSON(http.StatusOK, Response{
        Data: data,
        Meta: nil,
    })
}

// OKWithMeta はメタ情報付き成功レスポンスを返します
func OKWithMeta(c echo.Context, data interface{}, meta interface{}) error {
    return c.JSON(http.StatusOK, Response{
        Data: data,
        Meta: meta,
    })
}

// Created は作成成功レスポンスを返します
func Created(c echo.Context, data interface{}) error {
    return c.JSON(http.StatusCreated, Response{
        Data: data,
        Meta: nil,
    })
}

// NoContent はコンテンツなしレスポンスを返します
func NoContent(c echo.Context) error {
    return c.NoContent(http.StatusNoContent)
}

// Deleted は削除成功レスポンスを返します
func Deleted(c echo.Context, message string) error {
    return c.JSON(http.StatusOK, Response{
        Data: nil,
        Meta: Meta{Message: message},
    })
}

// List はリスト取得レスポンスを返します
func List(c echo.Context, data interface{}, pagination *Pagination) error {
    return c.JSON(http.StatusOK, Response{
        Data: data,
        Meta: Meta{Pagination: pagination},
    })
}

// NewPagination はページネーション情報を作成します
func NewPagination(page, perPage, totalItems int) *Pagination {
    totalPages := (totalItems + perPage - 1) / perPage
    if totalPages == 0 {
        totalPages = 1
    }

    return &Pagination{
        Page:       page,
        PerPage:    perPage,
        TotalItems: totalItems,
        TotalPages: totalPages,
        HasNext:    page < totalPages,
        HasPrev:    page > 1,
    }
}
```

### 6.2 レスポンスDTOの例

```go
// backend/internal/interface/dto/response/file.go

package response

import (
    "time"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
)

// FileResponse はファイルレスポンス
type FileResponse struct {
    ID             string    `json:"id"`
    Name           string    `json:"name"`
    FolderID       string    `json:"folder_id,omitempty"`
    Size           int64     `json:"size"`
    MimeType       string    `json:"mime_type"`
    CurrentVersion int       `json:"current_version"`
    Status         string    `json:"status"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}

// InitUploadResponse はアップロード開始レスポンス
type InitUploadResponse struct {
    FileID    string    `json:"file_id"`
    SessionID string    `json:"session_id"`
    UploadURL string    `json:"upload_url"`
    ExpiresAt time.Time `json:"expires_at"`
}

// ToFileResponse はエンティティをレスポンスに変換します
func ToFileResponse(f *entity.File) *FileResponse {
    return &FileResponse{
        ID:             f.ID.String(),
        Name:           f.Name.String(),
        FolderID:       f.FolderID.String(),
        Size:           f.Size,
        MimeType:       f.MimeType.String(),
        CurrentVersion: f.CurrentVersion,
        Status:         string(f.Status),
        CreatedAt:      f.CreatedAt,
        UpdatedAt:      f.UpdatedAt,
    }
}

// ToFileResponseList はエンティティリストをレスポンスに変換します
func ToFileResponseList(files []*entity.File) []*FileResponse {
    result := make([]*FileResponse, len(files))
    for i, f := range files {
        result[i] = ToFileResponse(f)
    }
    return result
}
```

---

## 7. ハンドラー実装例

### 7.1 ファイルハンドラー

```go
// backend/internal/interface/handler/file_handler.go

package handler

import (
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
    filecmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/file/command"
    fileqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/file/query"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// FileHandler はファイル関連のHTTPハンドラーです（CQRSパターン）
type FileHandler struct {
    // Commands
    initiateUploadCommand *filecmd.InitiateUploadCommand
    completeUploadCommand *filecmd.CompleteUploadCommand
    moveFileCommand       *filecmd.MoveFileCommand
    trashFileCommand      *filecmd.TrashFileCommand

    // Queries
    getFileQuery        *fileqry.GetFileQuery
    getDownloadURLQuery *fileqry.GetDownloadURLQuery
}

// NewFileHandler は新しいFileHandlerを作成します
func NewFileHandler(
    initiateUploadCommand *filecmd.InitiateUploadCommand,
    completeUploadCommand *filecmd.CompleteUploadCommand,
    moveFileCommand *filecmd.MoveFileCommand,
    trashFileCommand *filecmd.TrashFileCommand,
    getFileQuery *fileqry.GetFileQuery,
    getDownloadURLQuery *fileqry.GetDownloadURLQuery,
) *FileHandler {
    return &FileHandler{
        initiateUploadCommand: initiateUploadCommand,
        completeUploadCommand: completeUploadCommand,
        moveFileCommand:       moveFileCommand,
        trashFileCommand:      trashFileCommand,
        getFileQuery:          getFileQuery,
        getDownloadURLQuery:   getDownloadURLQuery,
    }
}

// InitUpload はアップロードを開始します
// POST /api/v1/files/upload
func (h *FileHandler) InitUpload(c echo.Context) error {
    var req request.InitUploadRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    userID, err := uuid.Parse(middleware.GetUserID(c))
    if err != nil {
        return apperror.NewUnauthorizedError("invalid user id")
    }

    var folderID *uuid.UUID
    if req.FolderID != nil {
        id, err := uuid.Parse(*req.FolderID)
        if err != nil {
            return apperror.NewValidationError("invalid folder_id", nil)
        }
        folderID = &id
    }

    output, err := h.initiateUploadCommand.Execute(c.Request().Context(), filecmd.InitiateUploadInput{
        UserID:   userID,
        FolderID: folderID,
        Name:     req.Name,
        Size:     req.Size,
        MimeType: req.MimeType,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.InitUploadResponse{
        FileID:    output.FileID.String(),
        SessionID: output.SessionID.String(),
        UploadURL: output.UploadURL,
        ExpiresAt: output.ExpiresAt,
    })
}

// Get はファイル情報を取得します
// GET /api/v1/files/:id
func (h *FileHandler) Get(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return apperror.NewValidationError("invalid file id", nil)
    }

    output, err := h.getFileQuery.Execute(c.Request().Context(), fileqry.GetFileInput{
        FileID: fileID,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.ToFileResponse(output.File))
}

// Download はダウンロードURLを取得します
// GET /api/v1/files/:id/download
func (h *FileHandler) Download(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return apperror.NewValidationError("invalid file id", nil)
    }

    output, err := h.getDownloadURLQuery.Execute(c.Request().Context(), fileqry.GetDownloadURLInput{
        FileID:    fileID,
        VersionID: nil,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, map[string]string{"download_url": output.URL})
}

// Rename はファイル名を変更します
// PATCH /api/v1/files/:id/rename
func (h *FileHandler) Rename(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return apperror.NewValidationError("invalid file id", nil)
    }

    var req request.RenameFileRequest
    if err := c.Bind(&req); err != nil {
        return apperror.NewValidationError("invalid request body", nil)
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    // Note: RenameFileCommand would need to be added to the handler
    output, err := h.renameFileCommand.Execute(c.Request().Context(), filecmd.RenameFileInput{
        FileID:  fileID,
        NewName: req.Name,
    })
    if err != nil {
        return err
    }

    return presenter.OK(c, response.ToFileResponse(output.File))
}
```

---

## 8. 初期化とDI

### 8.1 サーバー初期化

```go
// backend/internal/infrastructure/di/server.go

package di

import (
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/handler"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/router"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/server"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/validator"
)

// ServerComponents はサーバー関連の依存関係を保持します
type ServerComponents struct {
    Server *server.Server
    Router *router.Router
}

// NewServerComponents はサーバー関連の依存関係を初期化します
func NewServerComponents(
    cfg server.Config,
    authMiddleware *middleware.AuthMiddleware,
    rateLimitMiddleware *middleware.RateLimitMiddleware,
    permissionMiddleware *middleware.PermissionMiddleware,
    handlers *router.Handlers,
) *ServerComponents {
    srv := server.NewServer(cfg)
    e := srv.Echo()

    // カスタムバリデーター設定
    e.Validator = validator.NewCustomValidator()

    // カスタムエラーハンドラー設定
    e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

    // ルーター設定
    r := router.NewRouter(e, router.Config{
        AuthMiddleware:       authMiddleware.Authenticate(),
        RateLimitMiddleware:  rateLimitMiddleware,
        PermissionMiddleware: permissionMiddleware,
    }, handlers)
    r.Setup()

    return &ServerComponents{
        Server: srv,
        Router: r,
    }
}
```

---

## 9. テストヘルパー

### 9.1 テスト用ヘルパー

```go
// backend/internal/interface/handler/testhelper/handler.go

package testhelper

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/validator"
)

// TestContext はテスト用のEchoコンテキストを作成します
func TestContext(t *testing.T, method, path string, body interface{}) (echo.Context, *httptest.ResponseRecorder) {
    t.Helper()

    e := echo.New()
    e.Validator = validator.NewCustomValidator()
    e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

    var req *http.Request
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            t.Fatalf("failed to marshal request body: %v", err)
        }
        req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
        req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    } else {
        req = httptest.NewRequest(method, path, nil)
    }

    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    return c, rec
}

// WithAuthUser は認証済みユーザーを設定します
func WithAuthUser(c echo.Context, userID, sessionID string) echo.Context {
    c.Set(middleware.ContextKeyUserID, userID)
    c.Set(middleware.ContextKeySessionID, sessionID)
    return c
}

// ParseResponse はレスポンスをパースします
func ParseResponse[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
    t.Helper()

    var result struct {
        Data T           `json:"data"`
        Meta interface{} `json:"meta"`
    }

    if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
        t.Fatalf("failed to parse response: %v", err)
    }

    return result.Data
}
```

---

## 10. 受け入れ基準

### 10.1 機能要件

| 項目 | 基準 |
|------|------|
| ルーティング | URL設計規則に従ったエンドポイント |
| 認証 | JWT Bearer認証が動作する |
| バリデーション | リクエストが正しく検証される |
| エラーハンドリング | 統一形式でエラーが返される |
| レスポンス形式 | 統一形式でレスポンスが返される |
| ページネーション | meta.paginationが正しく設定される |

### 10.2 非機能要件

| 項目 | 基準 |
|------|------|
| リクエストタイムアウト | 30秒 |
| ボディサイズ制限 | 10MB |
| リクエストID | 全リクエストで追跡可能 |
| 構造化ログ | JSON形式でログ出力 |
| セキュリティヘッダー | 全レスポンスに設定 |

### 10.3 チェックリスト

- [ ] Echoサーバーが起動できる
- [ ] グローバルミドルウェアが適用される
- [ ] 認証ミドルウェアが動作する
- [ ] バリデーションエラーが正しく返される
- [ ] カスタムバリデーション（filename, password）が動作する
- [ ] エラーレスポンスが統一形式で返される
- [ ] 成功レスポンスが統一形式で返される
- [ ] ページネーションが正しく計算される
- [ ] リクエストIDがログに含まれる
- [ ] CORSが正しく設定される
- [ ] セキュリティヘッダーが設定される

---

## 関連ドキュメント

- [infra-database.md](./infra-database.md) - PostgreSQL基盤仕様
- [infra-redis.md](./infra-redis.md) - Redis基盤仕様
- [auth-identity.md](./auth-identity.md) - 認証仕様
- [API.md](../02-architecture/API.md) - API設計方針
- [BACKEND.md](../02-architecture/BACKEND.md) - バックエンド設計
