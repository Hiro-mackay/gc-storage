# Sharing ドメイン

## 概要

Sharingドメインは、ファイルやフォルダへの外部アクセスを可能にする共有リンクの作成、管理、アクセス検証を担当します。
Sharing Contextとして、認証なしでもリソースにアクセスできる仕組みを提供します。

---

## エンティティ

### ShareLink（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 共有リンクの一意識別子 |
| token | string | Yes | アクセストークン（URL-safe、一意） |
| resource_type | ResourceType | Yes | リソース種別（file/folder） |
| resource_id | UUID | Yes | リソースID |
| created_by | UUID | Yes | 作成者のユーザーID |
| permission | SharePermission | Yes | 付与される権限レベル |
| password_hash | string | No | パスワードハッシュ（bcrypt） |
| expires_at | timestamp | No | 有効期限（NULLは無期限） |
| max_access_count | int | No | 最大アクセス回数（NULLは無制限） |
| access_count | int | Yes | 現在のアクセス回数 |
| status | ShareLinkStatus | Yes | リンク状態 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ビジネスルール:**
- R-SL001: tokenは全共有リンクで一意
- R-SL002: tokenはURL-safe（Base62または類似）、32文字以上
- R-SL003: expires_at到達後はアクセス不可
- R-SL004: access_countがmax_access_countに達したらアクセス不可
- R-SL005: statusがrevokedの場合はアクセス不可
- R-SL006: 作成者はリソースに対してfile:shareまたはfolder:share権限が必要

**ステータス遷移:**
```
┌─────────┐       ┌─────────┐
│  active │──────▶│ revoked │
└────┬────┘       └─────────┘
     │
     └───────────▶┌─────────┐
                  │ expired │ (自動遷移)
                  └─────────┘
```

| ステータス | 説明 |
|-----------|------|
| active | アクティブ（利用可能） |
| revoked | 手動で無効化 |
| expired | 有効期限切れ |

### ShareLinkAccess

共有リンクへのアクセス履歴を記録します。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | アクセスログID |
| share_link_id | UUID | Yes | 共有リンクID |
| accessed_at | timestamp | Yes | アクセス日時 |
| ip_address | string | No | アクセス元IPアドレス |
| user_agent | string | No | User-Agent |
| user_id | UUID | No | 認証済みユーザーの場合のID |
| action | AccessAction | Yes | 実行されたアクション |

**ビジネスルール:**
- R-SLA001: 監査目的でアクセス履歴を記録
- R-SLA002: 個人情報（IP）は一定期間後に匿名化

---

## 値オブジェクト

### ShareToken

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | URL-safeなトークン文字列 |

**生成ルール:**
- 暗号学的に安全な乱数を使用
- Base62エンコード（a-zA-Z0-9）
- 最小32文字（約190ビットのエントロピー）

```go
type ShareToken struct {
    value string
}

func NewShareToken() ShareToken {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    const length = 32

    b := make([]byte, length)
    if _, err := rand.Read(b); err != nil {
        panic(err)
    }

    for i := range b {
        b[i] = charset[int(b[i])%len(charset)]
    }

    return ShareToken{value: string(b)}
}

func (t ShareToken) String() string {
    return t.value
}

func ParseShareToken(value string) (ShareToken, error) {
    if len(value) < 32 {
        return ShareToken{}, errors.New("invalid token length")
    }
    // URL-safe文字のみ許可
    validChars := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
    if !validChars.MatchString(value) {
        return ShareToken{}, errors.New("invalid token characters")
    }
    return ShareToken{value: value}, nil
}
```

### SharePermission

共有リンク経由で付与される権限レベル。

| 値 | 説明 | 操作 |
|-----|------|------|
| read | 閲覧のみ | ファイル閲覧、ダウンロード、フォルダ内容閲覧 |
| write | 編集可能 | read + ファイルアップロード、名前変更 |

```go
type SharePermission string

const (
    SharePermissionRead  SharePermission = "read"
    SharePermissionWrite SharePermission = "write"
)

func (p SharePermission) ToPermissions(resourceType string) []Permission {
    switch resourceType {
    case "file":
        if p == SharePermissionRead {
            return []Permission{PermFileRead}
        }
        return []Permission{PermFileRead, PermFileWrite}
    case "folder":
        if p == SharePermissionRead {
            return []Permission{PermFolderRead, PermFileRead}
        }
        return []Permission{PermFolderRead, PermFileRead, PermFolderCreate, PermFileWrite}
    default:
        return nil
    }
}
```

### ShareLinkStatus

| 値 | 説明 |
|-----|------|
| active | アクティブ |
| revoked | 無効化済み |
| expired | 期限切れ |

### AccessAction

| 値 | 説明 |
|-----|------|
| view | 閲覧 |
| download | ダウンロード |
| upload | アップロード |

### ShareLinkOptions

共有リンク作成時のオプション。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| permission | SharePermission | Yes | 付与権限 |
| password | string | No | パスワード（平文、ハッシュ化して保存） |
| expires_at | timestamp | No | 有効期限 |
| max_access_count | int | No | 最大アクセス回数 |

```go
type ShareLinkOptions struct {
    Permission     SharePermission
    Password       *string
    ExpiresAt      *time.Time
    MaxAccessCount *int
}

func (o ShareLinkOptions) Validate() error {
    // 有効期限は現在より未来
    if o.ExpiresAt != nil && o.ExpiresAt.Before(time.Now()) {
        return errors.New("expiration must be in the future")
    }

    // 最大アクセス回数は1以上
    if o.MaxAccessCount != nil && *o.MaxAccessCount < 1 {
        return errors.New("max access count must be at least 1")
    }

    // パスワードの強度チェック（設定時）
    if o.Password != nil {
        if len(*o.Password) < 4 {
            return errors.New("password must be at least 4 characters")
        }
    }

    return nil
}
```

---

## ドメインサービス

### ShareLinkService

**責務:** 共有リンクのライフサイクル管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| Create | cmd | ShareLink | 共有リンク作成 |
| Access | token, password | (Resource, PresignedURL) | リンクアクセス |
| Update | linkId, options | ShareLink | リンク設定更新 |
| Revoke | linkId | void | リンク無効化 |
| List | resourceType, resourceId | []ShareLink | リソースの共有リンク一覧 |

```go
type ShareLinkService interface {
    Create(ctx context.Context, cmd CreateShareLinkCommand) (*ShareLink, error)
    Access(ctx context.Context, token string, password *string, clientInfo ClientInfo) (*ShareLinkAccessResult, error)
    Update(ctx context.Context, linkID uuid.UUID, options ShareLinkOptions) (*ShareLink, error)
    Revoke(ctx context.Context, linkID uuid.UUID) error
    List(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*ShareLink, error)
}
```

**共有リンク作成:**
```go
func (s *ShareLinkServiceImpl) Create(
    ctx context.Context,
    cmd CreateShareLinkCommand,
) (*ShareLink, error) {
    // 1. リソース存在確認
    exists, err := s.resourceExists(ctx, cmd.ResourceType, cmd.ResourceID)
    if err != nil || !exists {
        return nil, errors.New("resource not found")
    }

    // 2. 権限チェック
    permission := PermFileShare
    if cmd.ResourceType == "folder" {
        permission = PermFolderShare
    }
    hasPermission, err := s.permissionResolver.HasPermission(ctx, cmd.CreatedBy, cmd.ResourceType, cmd.ResourceID, permission)
    if err != nil {
        return nil, err
    }
    if !hasPermission {
        return nil, errors.New("insufficient permission to create share link")
    }

    // 3. オプションバリデーション
    if err := cmd.Options.Validate(); err != nil {
        return nil, err
    }

    // 4. トークン生成
    token := NewShareToken()

    // 5. パスワードハッシュ化
    var passwordHash *string
    if cmd.Options.Password != nil {
        hash, err := bcrypt.GenerateFromPassword([]byte(*cmd.Options.Password), 12)
        if err != nil {
            return nil, err
        }
        hashStr := string(hash)
        passwordHash = &hashStr
    }

    // 6. 共有リンク作成
    link := &ShareLink{
        ID:             uuid.New(),
        Token:          token.String(),
        ResourceType:   cmd.ResourceType,
        ResourceID:     cmd.ResourceID,
        CreatedBy:      cmd.CreatedBy,
        Permission:     cmd.Options.Permission,
        PasswordHash:   passwordHash,
        ExpiresAt:      cmd.Options.ExpiresAt,
        MaxAccessCount: cmd.Options.MaxAccessCount,
        AccessCount:    0,
        Status:         ShareLinkStatusActive,
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }

    if err := s.linkRepo.Create(ctx, link); err != nil {
        return nil, err
    }

    // 7. イベント発行
    s.eventPublisher.Publish(ShareLinkCreatedEvent{
        LinkID:       link.ID,
        Token:        link.Token,
        ResourceType: link.ResourceType,
        ResourceID:   link.ResourceID,
        CreatedBy:    link.CreatedBy,
    })

    return link, nil
}
```

**共有リンクアクセス:**
```go
type ShareLinkAccessResult struct {
    ResourceType string
    ResourceID   uuid.UUID
    ResourceName string
    Permissions  []Permission
    PresignedURL *string // ファイルの場合
    Contents     []*ResourceInfo // フォルダの場合
}

func (s *ShareLinkServiceImpl) Access(
    ctx context.Context,
    token string,
    password *string,
    clientInfo ClientInfo,
) (*ShareLinkAccessResult, error) {
    // 1. トークン検証
    parsedToken, err := ParseShareToken(token)
    if err != nil {
        return nil, errors.New("invalid token")
    }

    // 2. 共有リンク取得
    link, err := s.linkRepo.FindByToken(ctx, parsedToken.String())
    if err != nil {
        return nil, errors.New("share link not found")
    }

    // 3. ステータス検証
    if link.Status != ShareLinkStatusActive {
        return nil, errors.New("share link is not active")
    }

    // 4. 有効期限検証
    if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
        // ステータスを更新
        link.Status = ShareLinkStatusExpired
        s.linkRepo.Update(ctx, link)
        return nil, errors.New("share link has expired")
    }

    // 5. アクセス回数検証
    if link.MaxAccessCount != nil && link.AccessCount >= *link.MaxAccessCount {
        return nil, errors.New("share link access limit reached")
    }

    // 6. パスワード検証
    if link.PasswordHash != nil {
        if password == nil {
            return nil, &PasswordRequiredError{}
        }
        if err := bcrypt.CompareHashAndPassword([]byte(*link.PasswordHash), []byte(*password)); err != nil {
            return nil, errors.New("invalid password")
        }
    }

    // 7. アクセスカウント更新
    link.AccessCount++
    link.UpdatedAt = time.Now()
    if err := s.linkRepo.Update(ctx, link); err != nil {
        return nil, err
    }

    // 8. アクセスログ記録
    accessLog := &ShareLinkAccess{
        ID:          uuid.New(),
        ShareLinkID: link.ID,
        AccessedAt:  time.Now(),
        IPAddress:   clientInfo.IPAddress,
        UserAgent:   clientInfo.UserAgent,
        UserID:      clientInfo.UserID,
        Action:      AccessActionView,
    }
    s.accessRepo.Create(ctx, accessLog)

    // 9. リソース情報取得
    result := &ShareLinkAccessResult{
        ResourceType: link.ResourceType,
        ResourceID:   link.ResourceID,
        Permissions:  link.Permission.ToPermissions(link.ResourceType),
    }

    if link.ResourceType == "file" {
        file, err := s.fileRepo.FindByID(ctx, link.ResourceID)
        if err != nil {
            return nil, err
        }
        result.ResourceName = file.Name

        // Presigned URL生成
        url, err := s.storageClient.GenerateGetURL(ctx, file.StorageKey, 15*time.Minute)
        if err != nil {
            return nil, err
        }
        result.PresignedURL = &url
    } else {
        folder, err := s.folderRepo.FindByID(ctx, link.ResourceID)
        if err != nil {
            return nil, err
        }
        result.ResourceName = folder.Name

        // フォルダ内容取得
        contents, err := s.getSharedFolderContents(ctx, link.ResourceID)
        if err != nil {
            return nil, err
        }
        result.Contents = contents
    }

    // 10. イベント発行
    s.eventPublisher.Publish(ShareLinkAccessedEvent{
        LinkID:    link.ID,
        IPAddress: clientInfo.IPAddress,
        UserID:    clientInfo.UserID,
    })

    return result, nil
}
```

### ShareLinkValidator

**責務:** 共有リンクの検証ロジック

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| Validate | token, password | ShareLink | リンク検証 |
| IsPasswordRequired | token | bool | パスワード必要判定 |

```go
type ShareLinkValidator interface {
    Validate(ctx context.Context, token string, password *string) (*ShareLink, error)
    IsPasswordRequired(ctx context.Context, token string) (bool, error)
}
```

### ShareLinkCleanupService

**責務:** 期限切れリンクのクリーンアップ

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| ExpireLinks | - | int64 | 期限切れリンクのステータス更新 |
| CleanupAccessLogs | olderThan | int64 | 古いアクセスログの削除/匿名化 |

```go
type ShareLinkCleanupService interface {
    ExpireLinks(ctx context.Context) (int64, error)
    CleanupAccessLogs(ctx context.Context, olderThan time.Duration) (int64, error)
}
```

---

## リポジトリ

### ShareLinkRepository

```go
type ShareLinkRepository interface {
    Create(ctx context.Context, link *ShareLink) error
    FindByID(ctx context.Context, id uuid.UUID) (*ShareLink, error)
    FindByToken(ctx context.Context, token string) (*ShareLink, error)
    FindByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*ShareLink, error)
    FindByCreator(ctx context.Context, createdBy uuid.UUID) ([]*ShareLink, error)
    Update(ctx context.Context, link *ShareLink) error
    Delete(ctx context.Context, id uuid.UUID) error

    // バッチ処理
    FindExpired(ctx context.Context) ([]*ShareLink, error)
    UpdateStatusBatch(ctx context.Context, ids []uuid.UUID, status ShareLinkStatus) (int64, error)
}
```

### ShareLinkAccessRepository

```go
type ShareLinkAccessRepository interface {
    Create(ctx context.Context, access *ShareLinkAccess) error
    FindByLinkID(ctx context.Context, linkID uuid.UUID, limit, offset int) ([]*ShareLinkAccess, error)
    CountByLinkID(ctx context.Context, linkID uuid.UUID) (int64, error)
    DeleteOlderThan(ctx context.Context, threshold time.Time) (int64, error)
    AnonymizeOlderThan(ctx context.Context, threshold time.Time) (int64, error)
}
```

---

## 関係性

### エンティティ関係図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Sharing Domain ERD                                   │
└─────────────────────────────────────────────────────────────────────────────┘

      ┌──────────────────┐          ┌──────────────────┐
      │      files       │          │     folders      │
      │    (external)    │          │    (external)    │
      └────────┬─────────┘          └────────┬─────────┘
               │                             │
               │ resource_id (file)          │ resource_id (folder)
               │                             │
               └──────────────┬──────────────┘
                              │
                              ▼
                     ┌──────────────────┐
                     │   share_links    │
                     ├──────────────────┤
                     │ id               │
                     │ token            │
                     │ resource_type    │
                     │ resource_id      │
                     │ created_by (FK)  │──────▶ users
                     │ permission       │
                     │ password_hash    │
                     │ expires_at       │
                     │ max_access_count │
                     │ access_count     │
                     │ status           │
                     │ created_at       │
                     │ updated_at       │
                     └────────┬─────────┘
                              │
                              │ 1:N
                              ▼
                  ┌────────────────────────┐
                  │   share_link_accesses  │
                  ├────────────────────────┤
                  │ id                     │
                  │ share_link_id (FK)     │
                  │ accessed_at            │
                  │ ip_address             │
                  │ user_agent             │
                  │ user_id (FK, nullable) │──────▶ users
                  │ action                 │
                  └────────────────────────┘
```

### 関係性ルール

| 関係 | カーディナリティ | 説明 |
|-----|----------------|------|
| ShareLink - Resource (File/Folder) | N:1 | 各共有リンクは1つのリソースを指す |
| ShareLink - Creator (User) | N:1 | 各共有リンクは1人の作成者を持つ |
| ShareLink - ShareLinkAccess | 1:N | 各共有リンクは複数のアクセス履歴を持つ |
| ShareLinkAccess - User | N:1 | アクセス時にログイン中の場合、ユーザーを記録 |

---

## 不変条件

1. **トークン制約**
   - tokenは全共有リンクで一意
   - tokenは変更不可（新しいリンクを作成）
   - tokenは推測不可能な形式

2. **アクセス制約**
   - 期限切れリンクはアクセス不可
   - アクセス上限到達リンクはアクセス不可
   - revokedリンクはアクセス不可
   - パスワード設定リンクは正しいパスワードなしでアクセス不可

3. **リソース整合性**
   - リソース削除時、関連する共有リンクも無効化
   - リソース移動時、共有リンクは維持（リソースIDで追跡）

4. **監査制約**
   - すべてのアクセスをログに記録
   - 一定期間後にIPアドレス等を匿名化

---

## ユースケース概要

| ユースケース | アクター | 概要 |
|------------|--------|------|
| CreateShareLink | User | 共有リンク作成 |
| AccessShareLink | Guest/User | 共有リンクでリソースにアクセス |
| AccessWithPassword | Guest/User | パスワード付きリンクにアクセス |
| UpdateShareLink | User | 共有リンク設定変更 |
| RevokeShareLink | User | 共有リンク無効化 |
| ListShareLinks | User | リソースの共有リンク一覧 |
| GetAccessHistory | User | 共有リンクのアクセス履歴 |
| DownloadViaShareLink | Guest/User | 共有リンク経由でファイルダウンロード |
| BrowseFolderViaShareLink | Guest/User | 共有リンク経由でフォルダ閲覧 |
| UploadViaShareLink | Guest/User | 共有リンク経由でアップロード（write権限時） |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| ShareLinkCreated | 共有リンク作成 | linkId, token, resourceType, resourceId, createdBy |
| ShareLinkAccessed | リンクアクセス | linkId, ipAddress, userId |
| ShareLinkRevoked | リンク無効化 | linkId, revokedBy |
| ShareLinkExpired | 期限切れ | linkId |
| ShareLinkUpdated | 設定変更 | linkId, changedFields |
| ShareLinkPasswordSet | パスワード設定 | linkId |
| FileDownloadedViaShareLink | 共有リンク経由ダウンロード | linkId, fileId |

---

## 共有リンクURL形式

```
https://{domain}/share/{token}

例:
https://gc-storage.example.com/share/abc123XYZ789...
```

**APIエンドポイント:**
```
GET  /api/v1/share/{token}          # リンク情報取得（パスワード必要性確認）
POST /api/v1/share/{token}/access   # リンクアクセス（パスワード送信）
GET  /api/v1/share/{token}/download # ファイルダウンロード
GET  /api/v1/share/{token}/browse   # フォルダ内容取得
POST /api/v1/share/{token}/upload   # ファイルアップロード（write権限時）
```

---

## 他コンテキストとの連携

### Storage Context（上流）
- FileID, FolderIDの参照
- ファイルダウンロード用Presigned URL生成
- フォルダ内容取得

### Identity Context（上流）
- 作成者UserIDの参照
- アクセス時のログインユーザー情報

### Authorization Context（上流）
- 共有リンク作成時のfile:share/folder:share権限チェック

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [ファイルドメイン](./file.md) - ファイル管理
- [フォルダドメイン](./folder.md) - フォルダ管理
- [権限ドメイン](./permission.md) - 共有リンク作成権限
- [セキュリティ設計](../02-architecture/SECURITY.md) - 共有リンク認証フロー
