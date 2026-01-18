# Share Link 詳細設計

## 概要

Share Linkは、ファイルやフォルダへの外部アクセスを可能にする共有リンクの作成、管理、アクセス検証を担当するモジュールです。
認証なしでもリソースにアクセスできる仕組みを提供します。

**スコープ:**
- 共有リンク作成（パスワード、有効期限、アクセス回数制限）
- 共有リンクアクセス（検証、リソース取得）
- 共有リンク管理（更新、無効化）
- アクセス履歴

**参照ドキュメント:**
- [共有ドメイン](../03-domains/sharing.md)
- [セキュリティ設計](../02-architecture/SECURITY.md)
- [MinIO仕様](./infra-minio.md)

---

## 1. エンティティ定義

### 1.1 ShareLink

```go
// internal/domain/sharing/share_link.go

package sharing

import (
    "crypto/rand"
    "time"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
)

type ShareLinkStatus string

const (
    ShareLinkStatusActive  ShareLinkStatus = "active"
    ShareLinkStatusRevoked ShareLinkStatus = "revoked"
    ShareLinkStatusExpired ShareLinkStatus = "expired"
)

type SharePermission string

const (
    SharePermissionRead  SharePermission = "read"
    SharePermissionWrite SharePermission = "write"
)

type ResourceType string

const (
    ResourceTypeFile   ResourceType = "file"
    ResourceTypeFolder ResourceType = "folder"
)

type ShareLink struct {
    ID             uuid.UUID
    Token          string
    ResourceType   ResourceType
    ResourceID     uuid.UUID
    CreatedBy      uuid.UUID
    Permission     SharePermission
    PasswordHash   *string
    ExpiresAt      *time.Time
    MaxAccessCount *int
    AccessCount    int
    Status         ShareLinkStatus
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// IsActive returns true if link is active and valid
func (l *ShareLink) IsActive() bool {
    if l.Status != ShareLinkStatusActive {
        return false
    }
    if l.ExpiresAt != nil && l.ExpiresAt.Before(time.Now()) {
        return false
    }
    if l.MaxAccessCount != nil && l.AccessCount >= *l.MaxAccessCount {
        return false
    }
    return true
}

// IsExpired returns true if link has expired
func (l *ShareLink) IsExpired() bool {
    return l.ExpiresAt != nil && l.ExpiresAt.Before(time.Now())
}

// HasReachedAccessLimit returns true if access limit is reached
func (l *ShareLink) HasReachedAccessLimit() bool {
    return l.MaxAccessCount != nil && l.AccessCount >= *l.MaxAccessCount
}

// RequiresPassword returns true if password is required
func (l *ShareLink) RequiresPassword() bool {
    return l.PasswordHash != nil
}

// ValidatePassword checks if the password is correct
func (l *ShareLink) ValidatePassword(password string) bool {
    if l.PasswordHash == nil {
        return true
    }
    err := bcrypt.CompareHashAndPassword([]byte(*l.PasswordHash), []byte(password))
    return err == nil
}

// IncrementAccessCount increments the access count
func (l *ShareLink) IncrementAccessCount() {
    l.AccessCount++
    l.UpdatedAt = time.Now()
}
```

### 1.2 ShareToken

```go
// internal/domain/sharing/share_token.go

package sharing

import (
    "crypto/rand"
    "errors"
    "regexp"
)

const (
    TokenLength = 32
    TokenCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var validTokenPattern = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

type ShareToken struct {
    value string
}

// NewShareToken generates a new secure random token
func NewShareToken() (ShareToken, error) {
    b := make([]byte, TokenLength)
    if _, err := rand.Read(b); err != nil {
        return ShareToken{}, err
    }

    for i := range b {
        b[i] = TokenCharset[int(b[i])%len(TokenCharset)]
    }

    return ShareToken{value: string(b)}, nil
}

// ParseShareToken validates and parses an existing token
func ParseShareToken(value string) (ShareToken, error) {
    if len(value) < TokenLength {
        return ShareToken{}, errors.New("token too short")
    }
    if !validTokenPattern.MatchString(value) {
        return ShareToken{}, errors.New("invalid token characters")
    }
    return ShareToken{value: value}, nil
}

func (t ShareToken) String() string {
    return t.value
}
```

### 1.3 ShareLinkAccess

```go
// internal/domain/sharing/share_link_access.go

package sharing

import (
    "time"
    "github.com/google/uuid"
)

type AccessAction string

const (
    AccessActionView     AccessAction = "view"
    AccessActionDownload AccessAction = "download"
    AccessActionUpload   AccessAction = "upload"
)

type ShareLinkAccess struct {
    ID          uuid.UUID
    ShareLinkID uuid.UUID
    AccessedAt  time.Time
    IPAddress   *string
    UserAgent   *string
    UserID      *uuid.UUID  // If user was authenticated
    Action      AccessAction
}
```

### 1.4 ShareLinkOptions

```go
// internal/domain/sharing/share_link_options.go

package sharing

import (
    "errors"
    "time"
)

type ShareLinkOptions struct {
    Permission     SharePermission
    Password       *string
    ExpiresAt      *time.Time
    MaxAccessCount *int
}

func (o ShareLinkOptions) Validate() error {
    // Permission validation
    if o.Permission != SharePermissionRead && o.Permission != SharePermissionWrite {
        return errors.New("invalid permission")
    }

    // Expiration must be in the future
    if o.ExpiresAt != nil && o.ExpiresAt.Before(time.Now()) {
        return errors.New("expiration must be in the future")
    }

    // Max access count must be positive
    if o.MaxAccessCount != nil && *o.MaxAccessCount < 1 {
        return errors.New("max access count must be at least 1")
    }

    // Password strength (if set)
    if o.Password != nil && len(*o.Password) < 4 {
        return errors.New("password must be at least 4 characters")
    }

    return nil
}
```

---

## 2. リポジトリインターフェース

### 2.1 ShareLinkRepository

```go
// internal/domain/sharing/share_link_repository.go

package sharing

import (
    "context"
    "time"
    "github.com/google/uuid"
)

type ShareLinkRepository interface {
    // CRUD
    Create(ctx context.Context, link *ShareLink) error
    FindByID(ctx context.Context, id uuid.UUID) (*ShareLink, error)
    FindByToken(ctx context.Context, token string) (*ShareLink, error)
    Update(ctx context.Context, link *ShareLink) error
    Delete(ctx context.Context, id uuid.UUID) error

    // Queries
    FindByResource(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID) ([]*ShareLink, error)
    FindByCreator(ctx context.Context, creatorID uuid.UUID) ([]*ShareLink, error)
    FindActiveByResource(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID) ([]*ShareLink, error)

    // Batch operations
    FindExpired(ctx context.Context) ([]*ShareLink, error)
    UpdateStatusBatch(ctx context.Context, ids []uuid.UUID, status ShareLinkStatus) (int64, error)
    DeleteByResource(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID) error
}
```

### 2.2 ShareLinkAccessRepository

```go
// internal/domain/sharing/share_link_access_repository.go

package sharing

import (
    "context"
    "time"
    "github.com/google/uuid"
)

type ShareLinkAccessRepository interface {
    Create(ctx context.Context, access *ShareLinkAccess) error
    FindByLinkID(ctx context.Context, linkID uuid.UUID, limit, offset int) ([]*ShareLinkAccess, error)
    CountByLinkID(ctx context.Context, linkID uuid.UUID) (int64, error)
    DeleteOlderThan(ctx context.Context, threshold time.Time) (int64, error)
    AnonymizeOlderThan(ctx context.Context, threshold time.Time) (int64, error)
}
```

---

## 3. ユースケース

### 3.1 共有リンク作成

```go
// internal/usecase/sharing/create_share_link.go

package sharing

import (
    "context"
    "time"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "gc-storage/internal/domain/authz"
    "gc-storage/internal/domain/sharing"
    "gc-storage/internal/domain/storage"
    "gc-storage/pkg/apperror"
)

type CreateShareLinkInput struct {
    ResourceType sharing.ResourceType
    ResourceID   uuid.UUID
    Options      sharing.ShareLinkOptions
    CreatorID    uuid.UUID
}

type CreateShareLinkOutput struct {
    ShareLink *sharing.ShareLink
    URL       string
}

type CreateShareLinkUseCase struct {
    linkRepo    sharing.ShareLinkRepository
    fileRepo    storage.FileRepository
    folderRepo  storage.FolderRepository
    resolver    authz.PermissionResolver
    baseURL     string
}

func NewCreateShareLinkUseCase(
    linkRepo sharing.ShareLinkRepository,
    fileRepo storage.FileRepository,
    folderRepo storage.FolderRepository,
    resolver authz.PermissionResolver,
    baseURL string,
) *CreateShareLinkUseCase {
    return &CreateShareLinkUseCase{
        linkRepo:   linkRepo,
        fileRepo:   fileRepo,
        folderRepo: folderRepo,
        resolver:   resolver,
        baseURL:    baseURL,
    }
}

func (uc *CreateShareLinkUseCase) Execute(ctx context.Context, input CreateShareLinkInput) (*CreateShareLinkOutput, error) {
    // 1. Validate options
    if err := input.Options.Validate(); err != nil {
        return nil, apperror.NewValidation("invalid options", err)
    }

    // 2. Verify resource exists
    exists, err := uc.resourceExists(ctx, input.ResourceType, input.ResourceID)
    if err != nil || !exists {
        return nil, apperror.NewNotFound("resource not found", err)
    }

    // 3. Check permission
    var permission authz.Permission
    if input.ResourceType == sharing.ResourceTypeFile {
        permission = authz.PermFileShare
    } else {
        permission = authz.PermFolderShare
    }
    hasPermission, err := uc.resolver.HasPermission(
        ctx, input.CreatorID, authz.ResourceType(input.ResourceType), input.ResourceID, permission,
    )
    if err != nil {
        return nil, err
    }
    if !hasPermission {
        return nil, apperror.NewForbidden("insufficient permission to create share link", nil)
    }

    // 4. Generate token
    token, err := sharing.NewShareToken()
    if err != nil {
        return nil, apperror.NewInternal("failed to generate token", err)
    }

    // 5. Hash password if provided
    var passwordHash *string
    if input.Options.Password != nil {
        hash, err := bcrypt.GenerateFromPassword([]byte(*input.Options.Password), 12)
        if err != nil {
            return nil, apperror.NewInternal("failed to hash password", err)
        }
        hashStr := string(hash)
        passwordHash = &hashStr
    }

    // 6. Create share link
    now := time.Now()
    link := &sharing.ShareLink{
        ID:             uuid.New(),
        Token:          token.String(),
        ResourceType:   input.ResourceType,
        ResourceID:     input.ResourceID,
        CreatedBy:      input.CreatorID,
        Permission:     input.Options.Permission,
        PasswordHash:   passwordHash,
        ExpiresAt:      input.Options.ExpiresAt,
        MaxAccessCount: input.Options.MaxAccessCount,
        AccessCount:    0,
        Status:         sharing.ShareLinkStatusActive,
        CreatedAt:      now,
        UpdatedAt:      now,
    }

    if err := uc.linkRepo.Create(ctx, link); err != nil {
        return nil, err
    }

    // 7. Build URL
    url := uc.baseURL + "/share/" + link.Token

    return &CreateShareLinkOutput{
        ShareLink: link,
        URL:       url,
    }, nil
}

func (uc *CreateShareLinkUseCase) resourceExists(ctx context.Context, resourceType sharing.ResourceType, resourceID uuid.UUID) (bool, error) {
    if resourceType == sharing.ResourceTypeFile {
        file, err := uc.fileRepo.FindByID(ctx, resourceID)
        return file != nil && file.Status == storage.FileStatusActive, err
    }
    folder, err := uc.folderRepo.FindByID(ctx, resourceID)
    return folder != nil && folder.Status == storage.FolderStatusActive, err
}
```

### 3.2 共有リンクアクセス

```go
// internal/usecase/sharing/access_share_link.go

package sharing

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/sharing"
    "gc-storage/internal/domain/storage"
    "gc-storage/internal/infrastructure/minio"
    "gc-storage/pkg/apperror"
)

type ClientInfo struct {
    IPAddress *string
    UserAgent *string
    UserID    *uuid.UUID
}

type ResourceInfo struct {
    ID       uuid.UUID `json:"id"`
    Name     string    `json:"name"`
    Type     string    `json:"type"`  // "file" or "folder"
    Size     int64     `json:"size,omitempty"`
    MimeType string    `json:"mime_type,omitempty"`
}

type AccessShareLinkInput struct {
    Token      string
    Password   *string
    ClientInfo ClientInfo
}

type AccessShareLinkOutput struct {
    ResourceType   string
    ResourceID     uuid.UUID
    ResourceName   string
    Permission     sharing.SharePermission
    PresignedURL   *string        // For file
    Contents       []*ResourceInfo // For folder
    RequiresAction string         // "password_required" or empty
}

const PresignedURLExpiry = 15 * time.Minute

type AccessShareLinkUseCase struct {
    linkRepo    sharing.ShareLinkRepository
    accessRepo  sharing.ShareLinkAccessRepository
    fileRepo    storage.FileRepository
    folderRepo  storage.FolderRepository
    minioClient minio.Client
    txManager   TransactionManager
}

func NewAccessShareLinkUseCase(
    linkRepo sharing.ShareLinkRepository,
    accessRepo sharing.ShareLinkAccessRepository,
    fileRepo storage.FileRepository,
    folderRepo storage.FolderRepository,
    minioClient minio.Client,
    txManager TransactionManager,
) *AccessShareLinkUseCase {
    return &AccessShareLinkUseCase{
        linkRepo:    linkRepo,
        accessRepo:  accessRepo,
        fileRepo:    fileRepo,
        folderRepo:  folderRepo,
        minioClient: minioClient,
        txManager:   txManager,
    }
}

func (uc *AccessShareLinkUseCase) Execute(ctx context.Context, input AccessShareLinkInput) (*AccessShareLinkOutput, error) {
    // 1. Parse and validate token
    token, err := sharing.ParseShareToken(input.Token)
    if err != nil {
        return nil, apperror.NewBadRequest("invalid token", err)
    }

    // 2. Find share link
    link, err := uc.linkRepo.FindByToken(ctx, token.String())
    if err != nil {
        return nil, apperror.NewNotFound("share link not found", err)
    }

    // 3. Check status
    if link.Status == sharing.ShareLinkStatusRevoked {
        return nil, apperror.NewGone("share link has been revoked", nil)
    }

    // 4. Check expiration
    if link.IsExpired() {
        // Update status to expired
        link.Status = sharing.ShareLinkStatusExpired
        _ = uc.linkRepo.Update(ctx, link)
        return nil, apperror.NewGone("share link has expired", nil)
    }

    // 5. Check access limit
    if link.HasReachedAccessLimit() {
        return nil, apperror.NewGone("share link access limit reached", nil)
    }

    // 6. Check password
    if link.RequiresPassword() {
        if input.Password == nil {
            return &AccessShareLinkOutput{
                RequiresAction: "password_required",
            }, nil
        }
        if !link.ValidatePassword(*input.Password) {
            return nil, apperror.NewUnauthorized("invalid password", nil)
        }
    }

    var output *AccessShareLinkOutput

    err = uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 7. Increment access count
        link.IncrementAccessCount()
        if err := uc.linkRepo.Update(ctx, link); err != nil {
            return err
        }

        // 8. Log access
        access := &sharing.ShareLinkAccess{
            ID:          uuid.New(),
            ShareLinkID: link.ID,
            AccessedAt:  time.Now(),
            IPAddress:   input.ClientInfo.IPAddress,
            UserAgent:   input.ClientInfo.UserAgent,
            UserID:      input.ClientInfo.UserID,
            Action:      sharing.AccessActionView,
        }
        if err := uc.accessRepo.Create(ctx, access); err != nil {
            return err
        }

        // 9. Get resource info
        output = &AccessShareLinkOutput{
            ResourceType: string(link.ResourceType),
            ResourceID:   link.ResourceID,
            Permission:   link.Permission,
        }

        if link.ResourceType == sharing.ResourceTypeFile {
            file, err := uc.fileRepo.FindByID(ctx, link.ResourceID)
            if err != nil {
                return apperror.NewNotFound("file not found", err)
            }
            if file.Status != storage.FileStatusActive {
                return apperror.NewGone("file is not available", nil)
            }
            output.ResourceName = file.Name.String()

            // Generate presigned URL
            url, err := uc.minioClient.GenerateGetURL(ctx, file.StorageKey, PresignedURLExpiry)
            if err != nil {
                return apperror.NewInternal("failed to generate download URL", err)
            }
            output.PresignedURL = &url
        } else {
            folder, err := uc.folderRepo.FindByID(ctx, link.ResourceID)
            if err != nil {
                return apperror.NewNotFound("folder not found", err)
            }
            if folder.Status != storage.FolderStatusActive {
                return apperror.NewGone("folder is not available", nil)
            }
            output.ResourceName = folder.Name.String()

            // Get folder contents
            contents, err := uc.getFolderContents(ctx, link.ResourceID)
            if err != nil {
                return err
            }
            output.Contents = contents
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return output, nil
}

func (uc *AccessShareLinkUseCase) getFolderContents(ctx context.Context, folderID uuid.UUID) ([]*ResourceInfo, error) {
    var contents []*ResourceInfo

    // Get subfolders
    folders, err := uc.folderRepo.FindByParentID(ctx, folderID, storage.FolderStatusActive)
    if err != nil {
        return nil, err
    }
    for _, f := range folders {
        contents = append(contents, &ResourceInfo{
            ID:   f.ID,
            Name: f.Name.String(),
            Type: "folder",
        })
    }

    // Get files
    files, err := uc.fileRepo.FindByFolderID(ctx, folderID, storage.FileStatusActive)
    if err != nil {
        return nil, err
    }
    for _, f := range files {
        contents = append(contents, &ResourceInfo{
            ID:       f.ID,
            Name:     f.Name.String(),
            Type:     "file",
            Size:     f.Size,
            MimeType: f.MimeType.String(),
        })
    }

    return contents, nil
}
```

### 3.3 ファイルダウンロード（共有リンク経由）

```go
// internal/usecase/sharing/download_via_share.go

package sharing

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/sharing"
    "gc-storage/internal/domain/storage"
    "gc-storage/internal/infrastructure/minio"
    "gc-storage/pkg/apperror"
)

type DownloadViaShareInput struct {
    Token      string
    FileID     uuid.UUID  // For folder share, specify which file
    Password   *string
    ClientInfo ClientInfo
}

type DownloadViaShareOutput struct {
    URL      string
    FileName string
    MimeType string
    Size     int64
}

type DownloadViaShareUseCase struct {
    linkRepo    sharing.ShareLinkRepository
    accessRepo  sharing.ShareLinkAccessRepository
    fileRepo    storage.FileRepository
    folderRepo  storage.FolderRepository
    minioClient minio.Client
}

func NewDownloadViaShareUseCase(
    linkRepo sharing.ShareLinkRepository,
    accessRepo sharing.ShareLinkAccessRepository,
    fileRepo storage.FileRepository,
    folderRepo storage.FolderRepository,
    minioClient minio.Client,
) *DownloadViaShareUseCase {
    return &DownloadViaShareUseCase{
        linkRepo:    linkRepo,
        accessRepo:  accessRepo,
        fileRepo:    fileRepo,
        folderRepo:  folderRepo,
        minioClient: minioClient,
    }
}

func (uc *DownloadViaShareUseCase) Execute(ctx context.Context, input DownloadViaShareInput) (*DownloadViaShareOutput, error) {
    // 1. Validate and get share link
    link, err := uc.validateShareLink(ctx, input.Token, input.Password)
    if err != nil {
        return nil, err
    }

    // 2. Determine file to download
    var fileID uuid.UUID
    if link.ResourceType == sharing.ResourceTypeFile {
        fileID = link.ResourceID
    } else {
        // Folder share - verify file is within shared folder
        if input.FileID == uuid.Nil {
            return nil, apperror.NewBadRequest("file_id is required for folder share", nil)
        }
        isChild, err := uc.isFileInFolder(ctx, input.FileID, link.ResourceID)
        if err != nil || !isChild {
            return nil, apperror.NewForbidden("file is not in shared folder", nil)
        }
        fileID = input.FileID
    }

    // 3. Get file
    file, err := uc.fileRepo.FindByID(ctx, fileID)
    if err != nil {
        return nil, apperror.NewNotFound("file not found", err)
    }
    if file.Status != storage.FileStatusActive {
        return nil, apperror.NewGone("file is not available", nil)
    }

    // 4. Generate download URL
    url, err := uc.minioClient.GenerateDownloadURL(ctx, file.StorageKey, file.Name.String(), PresignedURLExpiry)
    if err != nil {
        return nil, apperror.NewInternal("failed to generate download URL", err)
    }

    // 5. Log download action
    access := &sharing.ShareLinkAccess{
        ID:          uuid.New(),
        ShareLinkID: link.ID,
        AccessedAt:  time.Now(),
        IPAddress:   input.ClientInfo.IPAddress,
        UserAgent:   input.ClientInfo.UserAgent,
        UserID:      input.ClientInfo.UserID,
        Action:      sharing.AccessActionDownload,
    }
    _ = uc.accessRepo.Create(ctx, access)

    return &DownloadViaShareOutput{
        URL:      url,
        FileName: file.Name.String(),
        MimeType: file.MimeType.String(),
        Size:     file.Size,
    }, nil
}

func (uc *DownloadViaShareUseCase) validateShareLink(ctx context.Context, token string, password *string) (*sharing.ShareLink, error) {
    parsedToken, err := sharing.ParseShareToken(token)
    if err != nil {
        return nil, apperror.NewBadRequest("invalid token", err)
    }

    link, err := uc.linkRepo.FindByToken(ctx, parsedToken.String())
    if err != nil {
        return nil, apperror.NewNotFound("share link not found", err)
    }

    if !link.IsActive() {
        if link.IsExpired() {
            return nil, apperror.NewGone("share link has expired", nil)
        }
        if link.HasReachedAccessLimit() {
            return nil, apperror.NewGone("share link access limit reached", nil)
        }
        return nil, apperror.NewGone("share link is not active", nil)
    }

    if link.RequiresPassword() {
        if password == nil || !link.ValidatePassword(*password) {
            return nil, apperror.NewUnauthorized("invalid password", nil)
        }
    }

    return link, nil
}

func (uc *DownloadViaShareUseCase) isFileInFolder(ctx context.Context, fileID, folderID uuid.UUID) (bool, error) {
    file, err := uc.fileRepo.FindByID(ctx, fileID)
    if err != nil {
        return false, err
    }

    // Check if file is directly in folder
    if file.FolderID == folderID {
        return true, nil
    }

    // Check if file is in a subfolder
    currentFolderID := file.FolderID
    for {
        folder, err := uc.folderRepo.FindByID(ctx, currentFolderID)
        if err != nil {
            return false, err
        }
        if folder.ParentID == nil {
            return false, nil // Reached root without finding target folder
        }
        if *folder.ParentID == folderID {
            return true, nil
        }
        currentFolderID = *folder.ParentID
    }
}
```

### 3.4 共有リンク無効化

```go
// internal/usecase/sharing/revoke_share_link.go

package sharing

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/sharing"
    "gc-storage/pkg/apperror"
)

type RevokeShareLinkInput struct {
    LinkID  uuid.UUID
    ActorID uuid.UUID
}

type RevokeShareLinkUseCase struct {
    linkRepo sharing.ShareLinkRepository
}

func NewRevokeShareLinkUseCase(
    linkRepo sharing.ShareLinkRepository,
) *RevokeShareLinkUseCase {
    return &RevokeShareLinkUseCase{
        linkRepo: linkRepo,
    }
}

func (uc *RevokeShareLinkUseCase) Execute(ctx context.Context, input RevokeShareLinkInput) error {
    // 1. Get share link
    link, err := uc.linkRepo.FindByID(ctx, input.LinkID)
    if err != nil {
        return apperror.NewNotFound("share link not found", err)
    }

    // 2. Verify actor is the creator
    if link.CreatedBy != input.ActorID {
        return apperror.NewForbidden("only the creator can revoke the share link", nil)
    }

    // 3. Check if already revoked
    if link.Status == sharing.ShareLinkStatusRevoked {
        return apperror.NewBadRequest("share link is already revoked", nil)
    }

    // 4. Revoke
    link.Status = sharing.ShareLinkStatusRevoked
    link.UpdatedAt = time.Now()

    return uc.linkRepo.Update(ctx, link)
}
```

### 3.5 共有リンク更新

```go
// internal/usecase/sharing/update_share_link.go

package sharing

import (
    "context"
    "time"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "gc-storage/internal/domain/sharing"
    "gc-storage/pkg/apperror"
)

type UpdateShareLinkInput struct {
    LinkID         uuid.UUID
    ActorID        uuid.UUID
    ExpiresAt      **time.Time  // nil = no change, *nil = remove, *value = set
    MaxAccessCount **int        // nil = no change, *nil = remove, *value = set
    Password       **string     // nil = no change, *nil = remove, *value = set
}

type UpdateShareLinkOutput struct {
    ShareLink *sharing.ShareLink
}

type UpdateShareLinkUseCase struct {
    linkRepo sharing.ShareLinkRepository
}

func NewUpdateShareLinkUseCase(
    linkRepo sharing.ShareLinkRepository,
) *UpdateShareLinkUseCase {
    return &UpdateShareLinkUseCase{
        linkRepo: linkRepo,
    }
}

func (uc *UpdateShareLinkUseCase) Execute(ctx context.Context, input UpdateShareLinkInput) (*UpdateShareLinkOutput, error) {
    // 1. Get share link
    link, err := uc.linkRepo.FindByID(ctx, input.LinkID)
    if err != nil {
        return nil, apperror.NewNotFound("share link not found", err)
    }

    // 2. Verify actor is the creator
    if link.CreatedBy != input.ActorID {
        return nil, apperror.NewForbidden("only the creator can update the share link", nil)
    }

    // 3. Cannot update revoked link
    if link.Status == sharing.ShareLinkStatusRevoked {
        return nil, apperror.NewBadRequest("cannot update revoked share link", nil)
    }

    // 4. Apply updates
    if input.ExpiresAt != nil {
        if *input.ExpiresAt != nil && (*input.ExpiresAt).Before(time.Now()) {
            return nil, apperror.NewValidation("expiration must be in the future", nil)
        }
        link.ExpiresAt = *input.ExpiresAt
    }

    if input.MaxAccessCount != nil {
        if *input.MaxAccessCount != nil && **input.MaxAccessCount < 1 {
            return nil, apperror.NewValidation("max access count must be at least 1", nil)
        }
        link.MaxAccessCount = *input.MaxAccessCount
    }

    if input.Password != nil {
        if *input.Password == nil {
            link.PasswordHash = nil
        } else {
            hash, err := bcrypt.GenerateFromPassword([]byte(**input.Password), 12)
            if err != nil {
                return nil, apperror.NewInternal("failed to hash password", err)
            }
            hashStr := string(hash)
            link.PasswordHash = &hashStr
        }
    }

    link.UpdatedAt = time.Now()

    if err := uc.linkRepo.Update(ctx, link); err != nil {
        return nil, err
    }

    return &UpdateShareLinkOutput{ShareLink: link}, nil
}
```

### 3.6 共有リンク一覧

```go
// internal/usecase/sharing/list_share_links.go

package sharing

import (
    "context"
    "github.com/google/uuid"
    "gc-storage/internal/domain/authz"
    "gc-storage/internal/domain/sharing"
    "gc-storage/pkg/apperror"
)

type ListShareLinksInput struct {
    ResourceType sharing.ResourceType
    ResourceID   uuid.UUID
    ActorID      uuid.UUID
}

type ListShareLinksOutput struct {
    Links []*sharing.ShareLink
}

type ListShareLinksUseCase struct {
    linkRepo sharing.ShareLinkRepository
    resolver authz.PermissionResolver
}

func NewListShareLinksUseCase(
    linkRepo sharing.ShareLinkRepository,
    resolver authz.PermissionResolver,
) *ListShareLinksUseCase {
    return &ListShareLinksUseCase{
        linkRepo: linkRepo,
        resolver: resolver,
    }
}

func (uc *ListShareLinksUseCase) Execute(ctx context.Context, input ListShareLinksInput) (*ListShareLinksOutput, error) {
    // 1. Verify actor has permission to view share links
    var permission authz.Permission
    if input.ResourceType == sharing.ResourceTypeFile {
        permission = authz.PermFileShare
    } else {
        permission = authz.PermFolderShare
    }

    hasPermission, err := uc.resolver.HasPermission(
        ctx, input.ActorID, authz.ResourceType(input.ResourceType), input.ResourceID, permission,
    )
    if err != nil {
        return nil, err
    }
    if !hasPermission {
        return nil, apperror.NewForbidden("insufficient permission to view share links", nil)
    }

    // 2. Get share links
    links, err := uc.linkRepo.FindByResource(ctx, input.ResourceType, input.ResourceID)
    if err != nil {
        return nil, err
    }

    return &ListShareLinksOutput{Links: links}, nil
}
```

---

## 4. ハンドラー

```go
// internal/interface/handler/share_handler.go

package handler

import (
    "net/http"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "gc-storage/internal/domain/sharing"
    "gc-storage/internal/interface/dto"
    "gc-storage/internal/interface/middleware"
    usecase "gc-storage/internal/usecase/sharing"
)

type ShareHandler struct {
    createShareLink   *usecase.CreateShareLinkUseCase
    accessShareLink   *usecase.AccessShareLinkUseCase
    downloadViaShare  *usecase.DownloadViaShareUseCase
    revokeShareLink   *usecase.RevokeShareLinkUseCase
    updateShareLink   *usecase.UpdateShareLinkUseCase
    listShareLinks    *usecase.ListShareLinksUseCase
    getAccessHistory  *usecase.GetAccessHistoryUseCase
}

// POST /api/v1/files/:id/share or /api/v1/folders/:id/share
func (h *ShareHandler) Create(c echo.Context) error {
    resourceType := sharing.ResourceType(c.Param("type"))
    resourceID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid resource id")
    }

    var req dto.CreateShareLinkRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.createShareLink.Execute(c.Request().Context(), usecase.CreateShareLinkInput{
        ResourceType: resourceType,
        ResourceID:   resourceID,
        Options: sharing.ShareLinkOptions{
            Permission:     sharing.SharePermission(req.Permission),
            Password:       req.Password,
            ExpiresAt:      req.ExpiresAt,
            MaxAccessCount: req.MaxAccessCount,
        },
        CreatorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, dto.ShareLinkResponse{
        ID:             output.ShareLink.ID,
        Token:          output.ShareLink.Token,
        URL:            output.URL,
        Permission:     string(output.ShareLink.Permission),
        HasPassword:    output.ShareLink.RequiresPassword(),
        ExpiresAt:      output.ShareLink.ExpiresAt,
        MaxAccessCount: output.ShareLink.MaxAccessCount,
        AccessCount:    output.ShareLink.AccessCount,
        CreatedAt:      output.ShareLink.CreatedAt,
    })
}

// GET /api/v1/share/:token (public endpoint)
func (h *ShareHandler) GetInfo(c echo.Context) error {
    token := c.Param("token")

    output, err := h.accessShareLink.Execute(c.Request().Context(), usecase.AccessShareLinkInput{
        Token:    token,
        Password: nil, // Just check if password is required
        ClientInfo: usecase.ClientInfo{
            IPAddress: getClientIP(c),
            UserAgent: getClientUserAgent(c),
        },
    })
    if err != nil {
        return err
    }

    if output.RequiresAction == "password_required" {
        return c.JSON(http.StatusOK, dto.ShareLinkInfoResponse{
            RequiresPassword: true,
        })
    }

    return c.JSON(http.StatusOK, dto.ShareLinkInfoResponse{
        RequiresPassword: false,
        ResourceType:     output.ResourceType,
        ResourceName:     output.ResourceName,
        Permission:       string(output.Permission),
    })
}

// POST /api/v1/share/:token/access (public endpoint)
func (h *ShareHandler) Access(c echo.Context) error {
    token := c.Param("token")

    var req dto.AccessShareLinkRequest
    if err := c.Bind(&req); err != nil {
        return err
    }

    var userID *uuid.UUID
    if claims := middleware.GetClaimsOptional(c); claims != nil {
        userID = &claims.UserID
    }

    output, err := h.accessShareLink.Execute(c.Request().Context(), usecase.AccessShareLinkInput{
        Token:    token,
        Password: req.Password,
        ClientInfo: usecase.ClientInfo{
            IPAddress: getClientIP(c),
            UserAgent: getClientUserAgent(c),
            UserID:    userID,
        },
    })
    if err != nil {
        return err
    }

    if output.RequiresAction == "password_required" {
        return echo.NewHTTPError(http.StatusUnauthorized, "password required")
    }

    return c.JSON(http.StatusOK, dto.ShareLinkAccessResponse{
        ResourceType: output.ResourceType,
        ResourceID:   output.ResourceID,
        ResourceName: output.ResourceName,
        Permission:   string(output.Permission),
        PresignedURL: output.PresignedURL,
        Contents:     dto.ToResourceInfos(output.Contents),
    })
}

// GET /api/v1/share/:token/download (public endpoint)
func (h *ShareHandler) Download(c echo.Context) error {
    token := c.Param("token")
    fileIDStr := c.QueryParam("file_id")

    var fileID uuid.UUID
    if fileIDStr != "" {
        var err error
        fileID, err = uuid.Parse(fileIDStr)
        if err != nil {
            return echo.NewHTTPError(http.StatusBadRequest, "invalid file_id")
        }
    }

    password := c.QueryParam("password")
    var passwordPtr *string
    if password != "" {
        passwordPtr = &password
    }

    var userID *uuid.UUID
    if claims := middleware.GetClaimsOptional(c); claims != nil {
        userID = &claims.UserID
    }

    output, err := h.downloadViaShare.Execute(c.Request().Context(), usecase.DownloadViaShareInput{
        Token:    token,
        FileID:   fileID,
        Password: passwordPtr,
        ClientInfo: usecase.ClientInfo{
            IPAddress: getClientIP(c),
            UserAgent: getClientUserAgent(c),
            UserID:    userID,
        },
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.DownloadResponse{
        URL:      output.URL,
        FileName: output.FileName,
        MimeType: output.MimeType,
        Size:     output.Size,
    })
}

// DELETE /api/v1/share-links/:id
func (h *ShareHandler) Revoke(c echo.Context) error {
    linkID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid link id")
    }

    claims := middleware.GetClaims(c)
    err = h.revokeShareLink.Execute(c.Request().Context(), usecase.RevokeShareLinkInput{
        LinkID:  linkID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// PATCH /api/v1/share-links/:id
func (h *ShareHandler) Update(c echo.Context) error {
    linkID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid link id")
    }

    var req dto.UpdateShareLinkRequest
    if err := c.Bind(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.updateShareLink.Execute(c.Request().Context(), usecase.UpdateShareLinkInput{
        LinkID:         linkID,
        ActorID:        claims.UserID,
        ExpiresAt:      req.ExpiresAt,
        MaxAccessCount: req.MaxAccessCount,
        Password:       req.Password,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.ShareLinkResponse{
        ID:             output.ShareLink.ID,
        Token:          output.ShareLink.Token,
        Permission:     string(output.ShareLink.Permission),
        HasPassword:    output.ShareLink.RequiresPassword(),
        ExpiresAt:      output.ShareLink.ExpiresAt,
        MaxAccessCount: output.ShareLink.MaxAccessCount,
        AccessCount:    output.ShareLink.AccessCount,
        CreatedAt:      output.ShareLink.CreatedAt,
    })
}

// GET /api/v1/files/:id/share-links or /api/v1/folders/:id/share-links
func (h *ShareHandler) List(c echo.Context) error {
    resourceType := sharing.ResourceType(c.Param("type"))
    resourceID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid resource id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.listShareLinks.Execute(c.Request().Context(), usecase.ListShareLinksInput{
        ResourceType: resourceType,
        ResourceID:   resourceID,
        ActorID:      claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.ShareLinkListResponse{
        Links: dto.ToShareLinkResponses(output.Links),
    })
}

func getClientIP(c echo.Context) *string {
    ip := c.RealIP()
    return &ip
}

func getClientUserAgent(c echo.Context) *string {
    ua := c.Request().UserAgent()
    return &ua
}
```

---

## 5. DTO定義

```go
// internal/interface/dto/share.go

package dto

import (
    "time"
    "github.com/google/uuid"
)

type CreateShareLinkRequest struct {
    Permission     string     `json:"permission" validate:"required,oneof=read write"`
    Password       *string    `json:"password" validate:"omitempty,min=4"`
    ExpiresAt      *time.Time `json:"expires_at"`
    MaxAccessCount *int       `json:"max_access_count" validate:"omitempty,min=1"`
}

type UpdateShareLinkRequest struct {
    ExpiresAt      **time.Time `json:"expires_at"`
    MaxAccessCount **int       `json:"max_access_count"`
    Password       **string    `json:"password"`
}

type AccessShareLinkRequest struct {
    Password *string `json:"password"`
}

type ShareLinkResponse struct {
    ID             uuid.UUID  `json:"id"`
    Token          string     `json:"token"`
    URL            string     `json:"url,omitempty"`
    Permission     string     `json:"permission"`
    HasPassword    bool       `json:"has_password"`
    ExpiresAt      *time.Time `json:"expires_at,omitempty"`
    MaxAccessCount *int       `json:"max_access_count,omitempty"`
    AccessCount    int        `json:"access_count"`
    Status         string     `json:"status"`
    CreatedAt      time.Time  `json:"created_at"`
}

type ShareLinkListResponse struct {
    Links []ShareLinkResponse `json:"links"`
}

type ShareLinkInfoResponse struct {
    RequiresPassword bool   `json:"requires_password"`
    ResourceType     string `json:"resource_type,omitempty"`
    ResourceName     string `json:"resource_name,omitempty"`
    Permission       string `json:"permission,omitempty"`
}

type ShareLinkAccessResponse struct {
    ResourceType string          `json:"resource_type"`
    ResourceID   uuid.UUID       `json:"resource_id"`
    ResourceName string          `json:"resource_name"`
    Permission   string          `json:"permission"`
    PresignedURL *string         `json:"presigned_url,omitempty"`
    Contents     []ResourceInfo  `json:"contents,omitempty"`
}

type ResourceInfo struct {
    ID       uuid.UUID `json:"id"`
    Name     string    `json:"name"`
    Type     string    `json:"type"`
    Size     int64     `json:"size,omitempty"`
    MimeType string    `json:"mime_type,omitempty"`
}
```

---

## 6. APIエンドポイント

### 認証必要

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | /api/v1/files/:id/share | ShareHandler.Create | ファイル共有リンク作成 |
| POST | /api/v1/folders/:id/share | ShareHandler.Create | フォルダ共有リンク作成 |
| GET | /api/v1/files/:id/share-links | ShareHandler.List | ファイル共有リンク一覧 |
| GET | /api/v1/folders/:id/share-links | ShareHandler.List | フォルダ共有リンク一覧 |
| PATCH | /api/v1/share-links/:id | ShareHandler.Update | 共有リンク更新 |
| DELETE | /api/v1/share-links/:id | ShareHandler.Revoke | 共有リンク無効化 |
| GET | /api/v1/share-links/:id/history | ShareHandler.GetHistory | アクセス履歴 |

### 認証不要（パブリック）

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | /api/v1/share/:token | ShareHandler.GetInfo | リンク情報取得 |
| POST | /api/v1/share/:token/access | ShareHandler.Access | リンクアクセス |
| GET | /api/v1/share/:token/download | ShareHandler.Download | ファイルダウンロード |
| GET | /api/v1/share/:token/browse | ShareHandler.Browse | フォルダ閲覧 |

---

## 7. バックグラウンドジョブ

### 7.1 期限切れリンク処理

```go
// internal/job/share_link_expiry.go

package job

import (
    "context"
    "log/slog"
    "gc-storage/internal/domain/sharing"
)

type ShareLinkExpiryJob struct {
    linkRepo sharing.ShareLinkRepository
    logger   *slog.Logger
}

func NewShareLinkExpiryJob(
    linkRepo sharing.ShareLinkRepository,
    logger *slog.Logger,
) *ShareLinkExpiryJob {
    return &ShareLinkExpiryJob{
        linkRepo: linkRepo,
        logger:   logger,
    }
}

// Run executes every hour
func (j *ShareLinkExpiryJob) Run(ctx context.Context) error {
    expired, err := j.linkRepo.FindExpired(ctx)
    if err != nil {
        j.logger.Error("failed to find expired links", "error", err)
        return err
    }

    if len(expired) == 0 {
        return nil
    }

    ids := make([]uuid.UUID, len(expired))
    for i, link := range expired {
        ids[i] = link.ID
    }

    updated, err := j.linkRepo.UpdateStatusBatch(ctx, ids, sharing.ShareLinkStatusExpired)
    if err != nil {
        j.logger.Error("failed to update expired links", "error", err)
        return err
    }

    j.logger.Info("expired share links", "count", updated)
    return nil
}
```

### 7.2 アクセスログ匿名化

```go
// internal/job/access_log_anonymize.go

package job

import (
    "context"
    "log/slog"
    "time"
    "gc-storage/internal/domain/sharing"
)

const AccessLogRetentionDays = 90

type AccessLogAnonymizeJob struct {
    accessRepo sharing.ShareLinkAccessRepository
    logger     *slog.Logger
}

func NewAccessLogAnonymizeJob(
    accessRepo sharing.ShareLinkAccessRepository,
    logger *slog.Logger,
) *AccessLogAnonymizeJob {
    return &AccessLogAnonymizeJob{
        accessRepo: accessRepo,
        logger:     logger,
    }
}

// Run executes daily
func (j *AccessLogAnonymizeJob) Run(ctx context.Context) error {
    threshold := time.Now().AddDate(0, 0, -AccessLogRetentionDays)

    anonymized, err := j.accessRepo.AnonymizeOlderThan(ctx, threshold)
    if err != nil {
        j.logger.Error("failed to anonymize access logs", "error", err)
        return err
    }

    if anonymized > 0 {
        j.logger.Info("anonymized access logs", "count", anonymized)
    }
    return nil
}
```

---

## 8. 受け入れ基準

### 共有リンク作成
- [ ] file:share/folder:share権限を持つユーザーのみ作成可能
- [ ] read/write権限レベルを選択できる
- [ ] パスワードを設定できる
- [ ] 有効期限を設定できる
- [ ] 最大アクセス回数を設定できる
- [ ] URL-safe な32文字以上のトークンが生成される

### 共有リンクアクセス
- [ ] トークンでリソースにアクセスできる
- [ ] パスワード設定時は正しいパスワードが必要
- [ ] 有効期限切れのリンクはアクセス不可
- [ ] アクセス回数上限到達のリンクはアクセス不可
- [ ] 無効化されたリンクはアクセス不可
- [ ] ファイルはPresigned URLでダウンロード可能
- [ ] フォルダは内容一覧を取得可能
- [ ] フォルダ共有時、配下のファイルをダウンロード可能

### 共有リンク管理
- [ ] 作成者のみ更新・無効化可能
- [ ] 有効期限、アクセス回数、パスワードを変更できる
- [ ] 無効化したリンクは復活不可

### アクセス履歴
- [ ] すべてのアクセスがログに記録される
- [ ] IP、User-Agent、ユーザーID（認証時）を記録
- [ ] 90日後にIPアドレスが匿名化される

---

## 関連ドキュメント

- [共有ドメイン](../03-domains/sharing.md)
- [セキュリティ設計](../02-architecture/SECURITY.md)
- [MinIO仕様](./infra-minio.md)
- [Storage Core仕様](./storage-core.md)
