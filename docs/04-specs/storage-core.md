# Storage Core 詳細設計

## 概要

Storage Coreは、フォルダ管理、ファイル管理、ゴミ箱管理、バージョン管理を担当するコアモジュールです。
MinIO（オブジェクトストレージ）とPostgreSQL（メタデータ）の整合性を保ちながら、階層的なファイル構造を提供します。

**スコープ:**
- フォルダ CRUD（作成、取得、更新、削除）
- ファイルアップロード/ダウンロード
- ファイル操作（移動、コピー、名前変更）
- ゴミ箱（ソフト削除、復元、完全削除）
- バージョン管理

**参照ドキュメント:**
- [フォルダドメイン](../03-domains/folder.md)
- [ファイルドメイン](../03-domains/file.md)
- [MinIO仕様](./infra-minio.md)
- [データベース設計](../02-architecture/DATABASE.md)

---

## 1. エンティティ定義

### 1.1 Folder

```go
// internal/domain/storage/folder.go

package storage

import (
    "time"
    "github.com/google/uuid"
)

type OwnerType string

const (
    OwnerTypeUser  OwnerType = "user"
    OwnerTypeGroup OwnerType = "group"
)

type FolderStatus string

const (
    FolderStatusActive  FolderStatus = "active"
    FolderStatusTrashed FolderStatus = "trashed"
    FolderStatusDeleted FolderStatus = "deleted"
)

type Folder struct {
    ID        uuid.UUID
    Name      FolderName
    ParentID  *uuid.UUID  // nil for root folder
    OwnerID   uuid.UUID
    OwnerType OwnerType
    Path      string      // Materialized path: /{owner_type}/{owner_id}/...
    Depth     int         // 0 for root
    Status    FolderStatus
    TrashedAt *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
}

// IsRoot returns true if folder is a root folder
func (f *Folder) IsRoot() bool {
    return f.ParentID == nil
}

// CanTrash returns true if folder can be moved to trash
func (f *Folder) CanTrash() bool {
    return !f.IsRoot() && f.Status == FolderStatusActive
}

// CanRestore returns true if folder can be restored
func (f *Folder) CanRestore() bool {
    return f.Status == FolderStatusTrashed
}
```

### 1.2 FolderName（値オブジェクト）

```go
// internal/domain/storage/folder_name.go

package storage

import (
    "errors"
    "regexp"
    "strings"
)

var (
    ErrFolderNameEmpty      = errors.New("folder name cannot be empty")
    ErrFolderNameTooLong    = errors.New("folder name must not exceed 255 characters")
    ErrFolderNameInvalid    = errors.New("folder name contains invalid characters")
    ErrFolderNameReserved   = errors.New("folder name cannot be . or ..")
)

var invalidFolderChars = regexp.MustCompile(`[/\\:*?"<>|]`)

type FolderName struct {
    value string
}

func NewFolderName(value string) (FolderName, error) {
    trimmed := strings.TrimSpace(value)

    if len(trimmed) == 0 {
        return FolderName{}, ErrFolderNameEmpty
    }
    if len(trimmed) > 255 {
        return FolderName{}, ErrFolderNameTooLong
    }
    if invalidFolderChars.MatchString(trimmed) {
        return FolderName{}, ErrFolderNameInvalid
    }
    if trimmed == "." || trimmed == ".." {
        return FolderName{}, ErrFolderNameReserved
    }

    return FolderName{value: trimmed}, nil
}

func (n FolderName) String() string {
    return n.value
}

func (n FolderName) Equals(other FolderName) bool {
    return n.value == other.value
}
```

### 1.3 File

```go
// internal/domain/storage/file.go

package storage

import (
    "time"
    "github.com/google/uuid"
)

type FileStatus string

const (
    FileStatusPending FileStatus = "pending"
    FileStatusActive  FileStatus = "active"
    FileStatusTrashed FileStatus = "trashed"
    FileStatusDeleted FileStatus = "deleted"
    FileStatusFailed  FileStatus = "failed"
)

type File struct {
    ID             uuid.UUID
    Name           FileName
    FolderID       uuid.UUID
    OwnerID        uuid.UUID
    Size           int64
    MimeType       MimeType
    StorageKey     string
    CurrentVersion int
    Status         FileStatus
    TrashedAt      *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// CanDownload returns true if file can be downloaded
func (f *File) CanDownload() bool {
    return f.Status == FileStatusActive
}

// CanTrash returns true if file can be moved to trash
func (f *File) CanTrash() bool {
    return f.Status == FileStatusActive
}

// CanRestore returns true if file can be restored
func (f *File) CanRestore() bool {
    return f.Status == FileStatusTrashed
}

// IsPreviewable returns true if file can be previewed
func (f *File) IsPreviewable() bool {
    return f.MimeType.IsPreviewable()
}
```

### 1.4 FileName（値オブジェクト）

```go
// internal/domain/storage/file_name.go

package storage

import (
    "errors"
    "regexp"
    "strings"
)

var (
    ErrFileNameEmpty    = errors.New("file name cannot be empty")
    ErrFileNameTooLong  = errors.New("file name must not exceed 255 characters")
    ErrFileNameInvalid  = errors.New("file name contains invalid characters")
)

var invalidFileChars = regexp.MustCompile(`[/\\:*?"<>|]`)

type FileName struct {
    value string
}

func NewFileName(value string) (FileName, error) {
    trimmed := strings.TrimSpace(value)

    if len(trimmed) == 0 {
        return FileName{}, ErrFileNameEmpty
    }
    if len(trimmed) > 255 {
        return FileName{}, ErrFileNameTooLong
    }
    if invalidFileChars.MatchString(trimmed) {
        return FileName{}, ErrFileNameInvalid
    }

    return FileName{value: trimmed}, nil
}

func (n FileName) String() string {
    return n.value
}

func (n FileName) Extension() string {
    idx := strings.LastIndex(n.value, ".")
    if idx == -1 || idx == len(n.value)-1 {
        return ""
    }
    return strings.ToLower(n.value[idx+1:])
}

func (n FileName) BaseName() string {
    idx := strings.LastIndex(n.value, ".")
    if idx == -1 {
        return n.value
    }
    return n.value[:idx]
}
```

### 1.5 FileVersion

```go
// internal/domain/storage/file_version.go

package storage

import (
    "time"
    "github.com/google/uuid"
)

type FileVersion struct {
    ID             uuid.UUID
    FileID         uuid.UUID
    VersionNumber  int
    Size           int64
    StorageKey     string
    ChecksumMD5    string
    ChecksumSHA256 string
    CreatedBy      uuid.UUID
    CreatedAt      time.Time
}
```

### 1.6 UploadSession

```go
// internal/domain/storage/upload_session.go

package storage

import (
    "time"
    "github.com/google/uuid"
)

type UploadSessionStatus string

const (
    UploadSessionStatusInProgress UploadSessionStatus = "in_progress"
    UploadSessionStatusCompleted  UploadSessionStatus = "completed"
    UploadSessionStatusCancelled  UploadSessionStatus = "cancelled"
    UploadSessionStatusExpired    UploadSessionStatus = "expired"
)

type UploadSession struct {
    ID            uuid.UUID
    FileID        uuid.UUID
    UploadID      *string   // MinIO multipart upload ID
    IsMultipart   bool
    TotalParts    *int
    UploadedParts int
    ExpiresAt     time.Time
    Status        UploadSessionStatus
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// IsExpired returns true if session has expired
func (s *UploadSession) IsExpired() bool {
    return time.Now().After(s.ExpiresAt)
}

// CanComplete returns true if session can be completed
func (s *UploadSession) CanComplete() bool {
    return s.Status == UploadSessionStatusInProgress && !s.IsExpired()
}
```

---

## 2. リポジトリインターフェース

### 2.1 FolderRepository

```go
// internal/domain/storage/folder_repository.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
)

type FolderRepository interface {
    // CRUD
    Create(ctx context.Context, folder *Folder) error
    FindByID(ctx context.Context, id uuid.UUID) (*Folder, error)
    Update(ctx context.Context, folder *Folder) error
    Delete(ctx context.Context, id uuid.UUID) error

    // Queries
    FindByParentID(ctx context.Context, parentID uuid.UUID, status FolderStatus) ([]*Folder, error)
    FindRootByOwner(ctx context.Context, ownerType OwnerType, ownerID uuid.UUID) (*Folder, error)
    FindByOwner(ctx context.Context, ownerType OwnerType, ownerID uuid.UUID, status FolderStatus) ([]*Folder, error)
    ExistsByNameAndParent(ctx context.Context, name FolderName, parentID uuid.UUID) (bool, error)

    // Hierarchy
    FindAncestors(ctx context.Context, folderID uuid.UUID) ([]*Folder, error)
    FindDescendants(ctx context.Context, folderID uuid.UUID) ([]*Folder, error)
    FindDescendantIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error)

    // Trash
    FindTrashedByOwner(ctx context.Context, ownerType OwnerType, ownerID uuid.UUID) ([]*Folder, error)
    FindTrashedOlderThan(ctx context.Context, threshold time.Time) ([]*Folder, error)

    // Batch
    UpdateStatusBatch(ctx context.Context, ids []uuid.UUID, status FolderStatus, trashedAt *time.Time) error
}
```

### 2.2 FileRepository

```go
// internal/domain/storage/file_repository.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
)

type FileSearchQuery struct {
    OwnerID   uuid.UUID
    Query     string       // Full-text search
    FolderID  *uuid.UUID
    MimeTypes []string
    Status    FileStatus
    Limit     int
    Offset    int
}

type FileRepository interface {
    // CRUD
    Create(ctx context.Context, file *File) error
    FindByID(ctx context.Context, id uuid.UUID) (*File, error)
    Update(ctx context.Context, file *File) error
    Delete(ctx context.Context, id uuid.UUID) error

    // Queries
    FindByFolderID(ctx context.Context, folderID uuid.UUID, status FileStatus) ([]*File, error)
    FindByNameAndFolder(ctx context.Context, name FileName, folderID uuid.UUID) (*File, error)
    FindByOwnerID(ctx context.Context, ownerID uuid.UUID, status FileStatus) ([]*File, error)
    ExistsByNameAndFolder(ctx context.Context, name FileName, folderID uuid.UUID) (bool, error)

    // Search
    Search(ctx context.Context, query FileSearchQuery) ([]*File, int64, error)

    // Trash
    FindTrashedByOwner(ctx context.Context, ownerID uuid.UUID) ([]*File, error)
    FindTrashedOlderThan(ctx context.Context, threshold time.Time) ([]*File, error)
    FindByFolderIDs(ctx context.Context, folderIDs []uuid.UUID, status FileStatus) ([]*File, error)

    // Batch
    UpdateStatusBatch(ctx context.Context, ids []uuid.UUID, status FileStatus, trashedAt *time.Time) error
}
```

### 2.3 FileVersionRepository

```go
// internal/domain/storage/file_version_repository.go

package storage

import (
    "context"
    "github.com/google/uuid"
)

type FileVersionRepository interface {
    Create(ctx context.Context, version *FileVersion) error
    FindByID(ctx context.Context, id uuid.UUID) (*FileVersion, error)
    FindByFileID(ctx context.Context, fileID uuid.UUID) ([]*FileVersion, error)
    FindByFileAndVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) (*FileVersion, error)
    FindLatestByFileID(ctx context.Context, fileID uuid.UUID) (*FileVersion, error)
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteByFileID(ctx context.Context, fileID uuid.UUID) (int64, error)
    DeleteOlderVersions(ctx context.Context, fileID uuid.UUID, keepCount int) ([]string, error) // returns deleted storage keys
    CountByFileID(ctx context.Context, fileID uuid.UUID) (int, error)
}
```

### 2.4 UploadSessionRepository

```go
// internal/domain/storage/upload_session_repository.go

package storage

import (
    "context"
    "github.com/google/uuid"
)

type UploadSessionRepository interface {
    Create(ctx context.Context, session *UploadSession) error
    FindByID(ctx context.Context, id uuid.UUID) (*UploadSession, error)
    Update(ctx context.Context, session *UploadSession) error
    Delete(ctx context.Context, id uuid.UUID) error
    FindByFileID(ctx context.Context, fileID uuid.UUID) ([]*UploadSession, error)
    FindExpired(ctx context.Context) ([]*UploadSession, error)
    DeleteExpired(ctx context.Context) (int64, error)
}
```

---

## 3. ユースケース

### 3.1 フォルダ作成

```go
// internal/usecase/storage/create_folder.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/storage"
    "gc-storage/pkg/apperror"
)

type CreateFolderInput struct {
    Name      string
    ParentID  *uuid.UUID
    OwnerID   uuid.UUID
    OwnerType storage.OwnerType
}

type CreateFolderOutput struct {
    Folder *storage.Folder
}

type CreateFolderUseCase struct {
    folderRepo storage.FolderRepository
    txManager  TransactionManager
}

func NewCreateFolderUseCase(
    folderRepo storage.FolderRepository,
    txManager TransactionManager,
) *CreateFolderUseCase {
    return &CreateFolderUseCase{
        folderRepo: folderRepo,
        txManager:  txManager,
    }
}

func (uc *CreateFolderUseCase) Execute(ctx context.Context, input CreateFolderInput) (*CreateFolderOutput, error) {
    // 1. Validate folder name
    name, err := storage.NewFolderName(input.Name)
    if err != nil {
        return nil, apperror.NewValidation("invalid folder name", err)
    }

    var folder *storage.Folder

    err = uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 2. Get parent folder (or validate root creation)
        var parentFolder *storage.Folder
        var depth int
        var path string

        if input.ParentID != nil {
            var err error
            parentFolder, err = uc.folderRepo.FindByID(ctx, *input.ParentID)
            if err != nil {
                return apperror.NewNotFound("parent folder not found", err)
            }
            if parentFolder.Status != storage.FolderStatusActive {
                return apperror.NewBadRequest("parent folder is not active", nil)
            }
            // Verify ownership
            if parentFolder.OwnerID != input.OwnerID || parentFolder.OwnerType != input.OwnerType {
                return apperror.NewForbidden("cannot create folder in another owner's folder", nil)
            }
            depth = parentFolder.Depth + 1
            path = parentFolder.Path + "/" + name.String()
        } else {
            // Root folder creation
            depth = 0
            path = "/" + string(input.OwnerType) + "/" + input.OwnerID.String()
        }

        // 3. Check max depth
        if depth > 20 {
            return apperror.NewBadRequest("folder hierarchy exceeds maximum depth (20)", nil)
        }

        // 4. Check duplicate name
        exists, err := uc.folderRepo.ExistsByNameAndParent(ctx, name, *input.ParentID)
        if err != nil {
            return err
        }
        if exists {
            return apperror.NewConflict("folder with same name already exists", nil)
        }

        // 5. Create folder
        now := time.Now()
        folder = &storage.Folder{
            ID:        uuid.New(),
            Name:      name,
            ParentID:  input.ParentID,
            OwnerID:   input.OwnerID,
            OwnerType: input.OwnerType,
            Path:      path,
            Depth:     depth,
            Status:    storage.FolderStatusActive,
            CreatedAt: now,
            UpdatedAt: now,
        }

        return uc.folderRepo.Create(ctx, folder)
    })

    if err != nil {
        return nil, err
    }

    return &CreateFolderOutput{Folder: folder}, nil
}
```

### 3.2 フォルダ移動

```go
// internal/usecase/storage/move_folder.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/storage"
    "gc-storage/pkg/apperror"
)

type MoveFolderInput struct {
    FolderID    uuid.UUID
    NewParentID uuid.UUID
    ActorID     uuid.UUID
}

type MoveFolderOutput struct {
    Folder *storage.Folder
}

type MoveFolderUseCase struct {
    folderRepo storage.FolderRepository
    txManager  TransactionManager
}

func NewMoveFolderUseCase(
    folderRepo storage.FolderRepository,
    txManager TransactionManager,
) *MoveFolderUseCase {
    return &MoveFolderUseCase{
        folderRepo: folderRepo,
        txManager:  txManager,
    }
}

func (uc *MoveFolderUseCase) Execute(ctx context.Context, input MoveFolderInput) (*MoveFolderOutput, error) {
    var folder *storage.Folder

    err := uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. Get folder
        var err error
        folder, err = uc.folderRepo.FindByID(ctx, input.FolderID)
        if err != nil {
            return apperror.NewNotFound("folder not found", err)
        }
        if folder.Status != storage.FolderStatusActive {
            return apperror.NewBadRequest("folder is not active", nil)
        }
        if folder.IsRoot() {
            return apperror.NewBadRequest("cannot move root folder", nil)
        }

        // 2. Get new parent
        newParent, err := uc.folderRepo.FindByID(ctx, input.NewParentID)
        if err != nil {
            return apperror.NewNotFound("target folder not found", err)
        }
        if newParent.Status != storage.FolderStatusActive {
            return apperror.NewBadRequest("target folder is not active", nil)
        }

        // 3. Verify ownership
        if folder.OwnerID != newParent.OwnerID || folder.OwnerType != newParent.OwnerType {
            return apperror.NewForbidden("cannot move folder to another owner's folder", nil)
        }

        // 4. Check circular reference
        if input.FolderID == input.NewParentID {
            return apperror.NewBadRequest("cannot move folder into itself", nil)
        }
        descendantIDs, err := uc.folderRepo.FindDescendantIDs(ctx, input.FolderID)
        if err != nil {
            return err
        }
        for _, id := range descendantIDs {
            if id == input.NewParentID {
                return apperror.NewBadRequest("cannot move folder into its descendant", nil)
            }
        }

        // 5. Check max depth
        descendants, err := uc.folderRepo.FindDescendants(ctx, input.FolderID)
        if err != nil {
            return err
        }
        maxDescendantDepth := 0
        for _, d := range descendants {
            relativeDepth := d.Depth - folder.Depth
            if relativeDepth > maxDescendantDepth {
                maxDescendantDepth = relativeDepth
            }
        }
        newDepth := newParent.Depth + 1
        if newDepth+maxDescendantDepth > 20 {
            return apperror.NewBadRequest("folder hierarchy would exceed maximum depth (20)", nil)
        }

        // 6. Check duplicate name
        exists, err := uc.folderRepo.ExistsByNameAndParent(ctx, folder.Name, input.NewParentID)
        if err != nil {
            return err
        }
        if exists {
            return apperror.NewConflict("folder with same name already exists in target", nil)
        }

        // 7. Update folder
        now := time.Now()
        oldPath := folder.Path
        folder.ParentID = &input.NewParentID
        folder.Depth = newDepth
        folder.Path = newParent.Path + "/" + folder.Name.String()
        folder.UpdatedAt = now

        if err := uc.folderRepo.Update(ctx, folder); err != nil {
            return err
        }

        // 8. Update descendants' path and depth
        depthDiff := newDepth - (folder.Depth - (newDepth - folder.Depth))
        for _, d := range descendants {
            d.Depth += depthDiff
            d.Path = folder.Path + d.Path[len(oldPath):]
            d.UpdatedAt = now
            if err := uc.folderRepo.Update(ctx, d); err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return &MoveFolderOutput{Folder: folder}, nil
}
```

### 3.3 ファイルアップロード開始

```go
// internal/usecase/storage/initiate_upload.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/storage"
    "gc-storage/internal/infrastructure/minio"
    "gc-storage/pkg/apperror"
)

type InitiateUploadInput struct {
    FileName  string
    FolderID  uuid.UUID
    Size      int64
    MimeType  string
    ActorID   uuid.UUID
}

type InitiateUploadOutput struct {
    File        *storage.File
    Session     *storage.UploadSession
    UploadURL   string
    PartURLs    []string  // For multipart upload
}

const (
    MultipartThreshold = 100 * 1024 * 1024 // 100MB
    PartSize           = 10 * 1024 * 1024  // 10MB per part
    UploadURLExpiry    = 15 * time.Minute
    SessionExpiry      = 1 * time.Hour
)

type InitiateUploadUseCase struct {
    fileRepo    storage.FileRepository
    folderRepo  storage.FolderRepository
    sessionRepo storage.UploadSessionRepository
    minioClient minio.Client
    txManager   TransactionManager
}

func NewInitiateUploadUseCase(
    fileRepo storage.FileRepository,
    folderRepo storage.FolderRepository,
    sessionRepo storage.UploadSessionRepository,
    minioClient minio.Client,
    txManager TransactionManager,
) *InitiateUploadUseCase {
    return &InitiateUploadUseCase{
        fileRepo:    fileRepo,
        folderRepo:  folderRepo,
        sessionRepo: sessionRepo,
        minioClient: minioClient,
        txManager:   txManager,
    }
}

func (uc *InitiateUploadUseCase) Execute(ctx context.Context, input InitiateUploadInput) (*InitiateUploadOutput, error) {
    // 1. Validate file name
    fileName, err := storage.NewFileName(input.FileName)
    if err != nil {
        return nil, apperror.NewValidation("invalid file name", err)
    }

    // 2. Validate MIME type
    mimeType := storage.NewMimeType(input.MimeType)

    var output *InitiateUploadOutput

    err = uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 3. Get and validate folder
        folder, err := uc.folderRepo.FindByID(ctx, input.FolderID)
        if err != nil {
            return apperror.NewNotFound("folder not found", err)
        }
        if folder.Status != storage.FolderStatusActive {
            return apperror.NewBadRequest("folder is not active", nil)
        }

        // 4. Check for existing file (versioning)
        existingFile, _ := uc.fileRepo.FindByNameAndFolder(ctx, fileName, input.FolderID)

        now := time.Now()
        var file *storage.File
        var version int

        if existingFile != nil && existingFile.Status == storage.FileStatusActive {
            // Create new version
            file = existingFile
            file.CurrentVersion++
            file.Size = input.Size
            file.MimeType = mimeType
            file.UpdatedAt = now
            version = file.CurrentVersion
        } else {
            // Create new file
            file = &storage.File{
                ID:             uuid.New(),
                Name:           fileName,
                FolderID:       input.FolderID,
                OwnerID:        input.ActorID,
                Size:           input.Size,
                MimeType:       mimeType,
                CurrentVersion: 1,
                Status:         storage.FileStatusPending,
                CreatedAt:      now,
                UpdatedAt:      now,
            }
            version = 1
        }

        // 5. Generate storage key
        storageKey := minio.NewStorageKey(
            string(folder.OwnerType),
            folder.OwnerID,
            file.ID,
            version,
        )
        file.StorageKey = storageKey.String()

        // 6. Save/update file
        if existingFile != nil {
            if err := uc.fileRepo.Update(ctx, file); err != nil {
                return err
            }
        } else {
            if err := uc.fileRepo.Create(ctx, file); err != nil {
                return err
            }
        }

        // 7. Determine upload method
        isMultipart := input.Size > MultipartThreshold

        session := &storage.UploadSession{
            ID:          uuid.New(),
            FileID:      file.ID,
            IsMultipart: isMultipart,
            ExpiresAt:   now.Add(SessionExpiry),
            Status:      storage.UploadSessionStatusInProgress,
            CreatedAt:   now,
            UpdatedAt:   now,
        }

        output = &InitiateUploadOutput{
            File:    file,
            Session: session,
        }

        if isMultipart {
            // 8a. Multipart upload
            totalParts := int((input.Size + PartSize - 1) / PartSize)
            session.TotalParts = &totalParts

            uploadID, err := uc.minioClient.InitiateMultipartUpload(ctx, storageKey.String())
            if err != nil {
                return apperror.NewInternal("failed to initiate multipart upload", err)
            }
            session.UploadID = &uploadID

            // Generate part URLs
            partURLs := make([]string, totalParts)
            for i := 1; i <= totalParts; i++ {
                url, err := uc.minioClient.GetPartUploadURL(ctx, storageKey.String(), uploadID, i, UploadURLExpiry)
                if err != nil {
                    // Abort multipart upload on failure
                    _ = uc.minioClient.AbortMultipartUpload(ctx, storageKey.String(), uploadID)
                    return apperror.NewInternal("failed to generate part upload URL", err)
                }
                partURLs[i-1] = url
            }
            output.PartURLs = partURLs
        } else {
            // 8b. Simple upload
            uploadURL, err := uc.minioClient.GeneratePutURL(ctx, storageKey.String(), UploadURLExpiry)
            if err != nil {
                return apperror.NewInternal("failed to generate upload URL", err)
            }
            output.UploadURL = uploadURL
        }

        // 9. Save session
        if err := uc.sessionRepo.Create(ctx, session); err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return output, nil
}
```

### 3.4 ファイルアップロード完了

```go
// internal/usecase/storage/complete_upload.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/storage"
    "gc-storage/internal/infrastructure/minio"
    "gc-storage/pkg/apperror"
)

type CompleteUploadInput struct {
    SessionID uuid.UUID
    Parts     []minio.CompletedPart  // For multipart upload
    ActorID   uuid.UUID
}

type CompleteUploadOutput struct {
    File    *storage.File
    Version *storage.FileVersion
}

type CompleteUploadUseCase struct {
    fileRepo    storage.FileRepository
    versionRepo storage.FileVersionRepository
    sessionRepo storage.UploadSessionRepository
    minioClient minio.Client
    txManager   TransactionManager
}

func NewCompleteUploadUseCase(
    fileRepo storage.FileRepository,
    versionRepo storage.FileVersionRepository,
    sessionRepo storage.UploadSessionRepository,
    minioClient minio.Client,
    txManager TransactionManager,
) *CompleteUploadUseCase {
    return &CompleteUploadUseCase{
        fileRepo:    fileRepo,
        versionRepo: versionRepo,
        sessionRepo: sessionRepo,
        minioClient: minioClient,
        txManager:   txManager,
    }
}

func (uc *CompleteUploadUseCase) Execute(ctx context.Context, input CompleteUploadInput) (*CompleteUploadOutput, error) {
    var output *CompleteUploadOutput

    err := uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. Get and validate session
        session, err := uc.sessionRepo.FindByID(ctx, input.SessionID)
        if err != nil {
            return apperror.NewNotFound("upload session not found", err)
        }
        if !session.CanComplete() {
            if session.IsExpired() {
                return apperror.NewBadRequest("upload session expired", nil)
            }
            return apperror.NewBadRequest("upload session is not in progress", nil)
        }

        // 2. Get file
        file, err := uc.fileRepo.FindByID(ctx, session.FileID)
        if err != nil {
            return apperror.NewNotFound("file not found", err)
        }

        // 3. Complete multipart upload if needed
        if session.IsMultipart {
            if session.UploadID == nil {
                return apperror.NewInternal("multipart upload ID is missing", nil)
            }
            if err := uc.minioClient.CompleteMultipartUpload(ctx, file.StorageKey, *session.UploadID, input.Parts); err != nil {
                return apperror.NewInternal("failed to complete multipart upload", err)
            }
        }

        // 4. Verify object exists in MinIO
        exists, err := uc.minioClient.ObjectExists(ctx, file.StorageKey)
        if err != nil || !exists {
            return apperror.NewBadRequest("uploaded object not found", err)
        }

        // 5. Get object info
        objectInfo, err := uc.minioClient.GetObjectInfo(ctx, file.StorageKey)
        if err != nil {
            return apperror.NewInternal("failed to get object info", err)
        }

        now := time.Now()

        // 6. Update file
        file.Size = objectInfo.Size
        file.Status = storage.FileStatusActive
        file.UpdatedAt = now
        if err := uc.fileRepo.Update(ctx, file); err != nil {
            return err
        }

        // 7. Create version record
        version := &storage.FileVersion{
            ID:             uuid.New(),
            FileID:         file.ID,
            VersionNumber:  file.CurrentVersion,
            Size:           objectInfo.Size,
            StorageKey:     file.StorageKey,
            ChecksumMD5:    objectInfo.ETag,
            ChecksumSHA256: objectInfo.SHA256,
            CreatedBy:      input.ActorID,
            CreatedAt:      now,
        }
        if err := uc.versionRepo.Create(ctx, version); err != nil {
            return err
        }

        // 8. Update session
        session.Status = storage.UploadSessionStatusCompleted
        session.UpdatedAt = now
        if err := uc.sessionRepo.Update(ctx, session); err != nil {
            return err
        }

        output = &CompleteUploadOutput{
            File:    file,
            Version: version,
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return output, nil
}
```

### 3.5 ファイルダウンロード

```go
// internal/usecase/storage/download_file.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/storage"
    "gc-storage/internal/infrastructure/minio"
    "gc-storage/pkg/apperror"
)

type DownloadFileInput struct {
    FileID        uuid.UUID
    VersionNumber *int  // nil = current version
    ActorID       uuid.UUID
    AsAttachment  bool  // true = download, false = preview
}

type DownloadFileOutput struct {
    URL      string
    FileName string
    MimeType string
    Size     int64
}

const DownloadURLExpiry = 1 * time.Hour

type DownloadFileUseCase struct {
    fileRepo    storage.FileRepository
    versionRepo storage.FileVersionRepository
    minioClient minio.Client
}

func NewDownloadFileUseCase(
    fileRepo storage.FileRepository,
    versionRepo storage.FileVersionRepository,
    minioClient minio.Client,
) *DownloadFileUseCase {
    return &DownloadFileUseCase{
        fileRepo:    fileRepo,
        versionRepo: versionRepo,
        minioClient: minioClient,
    }
}

func (uc *DownloadFileUseCase) Execute(ctx context.Context, input DownloadFileInput) (*DownloadFileOutput, error) {
    // 1. Get file
    file, err := uc.fileRepo.FindByID(ctx, input.FileID)
    if err != nil {
        return nil, apperror.NewNotFound("file not found", err)
    }
    if !file.CanDownload() {
        return nil, apperror.NewBadRequest("file is not available for download", nil)
    }

    // 2. Get version
    var version *storage.FileVersion
    if input.VersionNumber != nil {
        version, err = uc.versionRepo.FindByFileAndVersion(ctx, input.FileID, *input.VersionNumber)
        if err != nil {
            return nil, apperror.NewNotFound("version not found", err)
        }
    } else {
        version, err = uc.versionRepo.FindLatestByFileID(ctx, input.FileID)
        if err != nil {
            return nil, apperror.NewNotFound("no versions found", err)
        }
    }

    // 3. Generate presigned URL
    var url string
    if input.AsAttachment {
        url, err = uc.minioClient.GenerateDownloadURL(ctx, version.StorageKey, file.Name.String(), DownloadURLExpiry)
    } else {
        url, err = uc.minioClient.GenerateGetURL(ctx, version.StorageKey, DownloadURLExpiry)
    }
    if err != nil {
        return nil, apperror.NewInternal("failed to generate download URL", err)
    }

    return &DownloadFileOutput{
        URL:      url,
        FileName: file.Name.String(),
        MimeType: file.MimeType.String(),
        Size:     version.Size,
    }, nil
}
```

### 3.6 ゴミ箱移動（フォルダ）

```go
// internal/usecase/storage/trash_folder.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/storage"
    "gc-storage/pkg/apperror"
)

type TrashFolderInput struct {
    FolderID uuid.UUID
    ActorID  uuid.UUID
}

type TrashFolderUseCase struct {
    folderRepo storage.FolderRepository
    fileRepo   storage.FileRepository
    txManager  TransactionManager
}

func NewTrashFolderUseCase(
    folderRepo storage.FolderRepository,
    fileRepo storage.FileRepository,
    txManager TransactionManager,
) *TrashFolderUseCase {
    return &TrashFolderUseCase{
        folderRepo: folderRepo,
        fileRepo:   fileRepo,
        txManager:  txManager,
    }
}

func (uc *TrashFolderUseCase) Execute(ctx context.Context, input TrashFolderInput) error {
    return uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. Get folder
        folder, err := uc.folderRepo.FindByID(ctx, input.FolderID)
        if err != nil {
            return apperror.NewNotFound("folder not found", err)
        }
        if !folder.CanTrash() {
            if folder.IsRoot() {
                return apperror.NewBadRequest("cannot trash root folder", nil)
            }
            return apperror.NewBadRequest("folder is not active", nil)
        }

        now := time.Now()

        // 2. Get all descendant folders
        descendantIDs, err := uc.folderRepo.FindDescendantIDs(ctx, input.FolderID)
        if err != nil {
            return err
        }

        // 3. Trash all folders (including target)
        allFolderIDs := append([]uuid.UUID{input.FolderID}, descendantIDs...)
        if err := uc.folderRepo.UpdateStatusBatch(ctx, allFolderIDs, storage.FolderStatusTrashed, &now); err != nil {
            return err
        }

        // 4. Trash all files in affected folders
        files, err := uc.fileRepo.FindByFolderIDs(ctx, allFolderIDs, storage.FileStatusActive)
        if err != nil {
            return err
        }
        if len(files) > 0 {
            fileIDs := make([]uuid.UUID, len(files))
            for i, f := range files {
                fileIDs[i] = f.ID
            }
            if err := uc.fileRepo.UpdateStatusBatch(ctx, fileIDs, storage.FileStatusTrashed, &now); err != nil {
                return err
            }
        }

        return nil
    })
}
```

### 3.7 ゴミ箱から復元

```go
// internal/usecase/storage/restore_folder.go

package storage

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/storage"
    "gc-storage/pkg/apperror"
)

type RestoreFolderInput struct {
    FolderID uuid.UUID
    ActorID  uuid.UUID
}

type RestoreFolderOutput struct {
    Folder *storage.Folder
}

type RestoreFolderUseCase struct {
    folderRepo storage.FolderRepository
    fileRepo   storage.FileRepository
    txManager  TransactionManager
}

func NewRestoreFolderUseCase(
    folderRepo storage.FolderRepository,
    fileRepo storage.FileRepository,
    txManager TransactionManager,
) *RestoreFolderUseCase {
    return &RestoreFolderUseCase{
        folderRepo: folderRepo,
        fileRepo:   fileRepo,
        txManager:  txManager,
    }
}

func (uc *RestoreFolderUseCase) Execute(ctx context.Context, input RestoreFolderInput) (*RestoreFolderOutput, error) {
    var folder *storage.Folder

    err := uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. Get folder
        var err error
        folder, err = uc.folderRepo.FindByID(ctx, input.FolderID)
        if err != nil {
            return apperror.NewNotFound("folder not found", err)
        }
        if !folder.CanRestore() {
            return apperror.NewBadRequest("folder is not in trash", nil)
        }

        // 2. Check parent folder is active
        if folder.ParentID != nil {
            parent, err := uc.folderRepo.FindByID(ctx, *folder.ParentID)
            if err != nil {
                return apperror.NewNotFound("parent folder not found", err)
            }
            if parent.Status != storage.FolderStatusActive {
                return apperror.NewBadRequest("parent folder is not active, restore it first", nil)
            }
        }

        // 3. Check for name conflict
        if folder.ParentID != nil {
            exists, err := uc.folderRepo.ExistsByNameAndParent(ctx, folder.Name, *folder.ParentID)
            if err != nil {
                return err
            }
            if exists {
                return apperror.NewConflict("folder with same name already exists", nil)
            }
        }

        now := time.Now()

        // 4. Restore folder (only this folder, not descendants)
        folder.Status = storage.FolderStatusActive
        folder.TrashedAt = nil
        folder.UpdatedAt = now
        if err := uc.folderRepo.Update(ctx, folder); err != nil {
            return err
        }

        // 5. Restore files in this folder
        files, err := uc.fileRepo.FindByFolderID(ctx, input.FolderID, storage.FileStatusTrashed)
        if err != nil {
            return err
        }
        if len(files) > 0 {
            fileIDs := make([]uuid.UUID, len(files))
            for i, f := range files {
                fileIDs[i] = f.ID
            }
            if err := uc.fileRepo.UpdateStatusBatch(ctx, fileIDs, storage.FileStatusActive, nil); err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return &RestoreFolderOutput{Folder: folder}, nil
}
```

### 3.8 完全削除

```go
// internal/usecase/storage/permanently_delete.go

package storage

import (
    "context"
    "github.com/google/uuid"
    "gc-storage/internal/domain/storage"
    "gc-storage/internal/infrastructure/minio"
    "gc-storage/pkg/apperror"
)

type PermanentlyDeleteFolderInput struct {
    FolderID uuid.UUID
    ActorID  uuid.UUID
}

type PermanentlyDeleteFolderUseCase struct {
    folderRepo  storage.FolderRepository
    fileRepo    storage.FileRepository
    versionRepo storage.FileVersionRepository
    minioClient minio.Client
    txManager   TransactionManager
}

func NewPermanentlyDeleteFolderUseCase(
    folderRepo storage.FolderRepository,
    fileRepo storage.FileRepository,
    versionRepo storage.FileVersionRepository,
    minioClient minio.Client,
    txManager TransactionManager,
) *PermanentlyDeleteFolderUseCase {
    return &PermanentlyDeleteFolderUseCase{
        folderRepo:  folderRepo,
        fileRepo:    fileRepo,
        versionRepo: versionRepo,
        minioClient: minioClient,
        txManager:   txManager,
    }
}

func (uc *PermanentlyDeleteFolderUseCase) Execute(ctx context.Context, input PermanentlyDeleteFolderInput) error {
    // 1. Get folder
    folder, err := uc.folderRepo.FindByID(ctx, input.FolderID)
    if err != nil {
        return apperror.NewNotFound("folder not found", err)
    }
    if folder.Status != storage.FolderStatusTrashed {
        return apperror.NewBadRequest("folder must be in trash to permanently delete", nil)
    }

    // 2. Get all descendant folder IDs
    descendantIDs, err := uc.folderRepo.FindDescendantIDs(ctx, input.FolderID)
    if err != nil {
        return err
    }
    allFolderIDs := append([]uuid.UUID{input.FolderID}, descendantIDs...)

    // 3. Get all files in affected folders
    files, err := uc.fileRepo.FindByFolderIDs(ctx, allFolderIDs, storage.FileStatusTrashed)
    if err != nil {
        return err
    }

    // 4. Collect all storage keys to delete
    var storageKeys []string
    for _, file := range files {
        versions, err := uc.versionRepo.FindByFileID(ctx, file.ID)
        if err != nil {
            return err
        }
        for _, v := range versions {
            storageKeys = append(storageKeys, v.StorageKey)
        }
    }

    // 5. Delete from MinIO (outside transaction)
    if len(storageKeys) > 0 {
        if err := uc.minioClient.DeleteObjects(ctx, storageKeys); err != nil {
            return apperror.NewInternal("failed to delete objects from storage", err)
        }
    }

    // 6. Delete from database (in transaction)
    return uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // Delete versions
        for _, file := range files {
            if _, err := uc.versionRepo.DeleteByFileID(ctx, file.ID); err != nil {
                return err
            }
        }

        // Delete files
        for _, file := range files {
            if err := uc.fileRepo.Delete(ctx, file.ID); err != nil {
                return err
            }
        }

        // Delete folders (descendants first, then target)
        for i := len(descendantIDs) - 1; i >= 0; i-- {
            if err := uc.folderRepo.Delete(ctx, descendantIDs[i]); err != nil {
                return err
            }
        }
        if err := uc.folderRepo.Delete(ctx, input.FolderID); err != nil {
            return err
        }

        return nil
    })
}
```

---

## 4. ハンドラー

### 4.1 フォルダハンドラー

```go
// internal/interface/handler/folder_handler.go

package handler

import (
    "net/http"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "gc-storage/internal/interface/dto"
    "gc-storage/internal/interface/middleware"
    "gc-storage/internal/usecase/storage"
)

type FolderHandler struct {
    createFolder  *storage.CreateFolderUseCase
    moveFolder    *storage.MoveFolderUseCase
    renameFolder  *storage.RenameFolderUseCase
    trashFolder   *storage.TrashFolderUseCase
    restoreFolder *storage.RestoreFolderUseCase
    deleteFolder  *storage.PermanentlyDeleteFolderUseCase
    listContents  *storage.ListFolderContentsUseCase
    getAncestors  *storage.GetFolderAncestorsUseCase
}

// POST /api/v1/folders
func (h *FolderHandler) Create(c echo.Context) error {
    var req dto.CreateFolderRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.createFolder.Execute(c.Request().Context(), storage.CreateFolderInput{
        Name:      req.Name,
        ParentID:  req.ParentID,
        OwnerID:   claims.UserID,
        OwnerType: storage.OwnerTypeUser,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, dto.FolderResponse{
        ID:        output.Folder.ID,
        Name:      output.Folder.Name.String(),
        ParentID:  output.Folder.ParentID,
        Path:      output.Folder.Path,
        CreatedAt: output.Folder.CreatedAt,
        UpdatedAt: output.Folder.UpdatedAt,
    })
}

// PATCH /api/v1/folders/:id/move
func (h *FolderHandler) Move(c echo.Context) error {
    folderID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid folder id")
    }

    var req dto.MoveFolderRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.moveFolder.Execute(c.Request().Context(), storage.MoveFolderInput{
        FolderID:    folderID,
        NewParentID: req.ParentID,
        ActorID:     claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.FolderResponse{
        ID:        output.Folder.ID,
        Name:      output.Folder.Name.String(),
        ParentID:  output.Folder.ParentID,
        Path:      output.Folder.Path,
        CreatedAt: output.Folder.CreatedAt,
        UpdatedAt: output.Folder.UpdatedAt,
    })
}

// DELETE /api/v1/folders/:id
func (h *FolderHandler) Trash(c echo.Context) error {
    folderID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid folder id")
    }

    claims := middleware.GetClaims(c)
    err = h.trashFolder.Execute(c.Request().Context(), storage.TrashFolderInput{
        FolderID: folderID,
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// POST /api/v1/folders/:id/restore
func (h *FolderHandler) Restore(c echo.Context) error {
    folderID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid folder id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.restoreFolder.Execute(c.Request().Context(), storage.RestoreFolderInput{
        FolderID: folderID,
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.FolderResponse{
        ID:        output.Folder.ID,
        Name:      output.Folder.Name.String(),
        ParentID:  output.Folder.ParentID,
        Path:      output.Folder.Path,
        CreatedAt: output.Folder.CreatedAt,
        UpdatedAt: output.Folder.UpdatedAt,
    })
}

// DELETE /api/v1/folders/:id/permanent
func (h *FolderHandler) PermanentDelete(c echo.Context) error {
    folderID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid folder id")
    }

    claims := middleware.GetClaims(c)
    err = h.deleteFolder.Execute(c.Request().Context(), storage.PermanentlyDeleteFolderInput{
        FolderID: folderID,
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// GET /api/v1/folders/:id/contents
func (h *FolderHandler) ListContents(c echo.Context) error {
    folderID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid folder id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.listContents.Execute(c.Request().Context(), storage.ListFolderContentsInput{
        FolderID: folderID,
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.FolderContentsResponse{
        Folders: dto.ToFolderResponses(output.Folders),
        Files:   dto.ToFileResponses(output.Files),
    })
}

// GET /api/v1/folders/:id/breadcrumb
func (h *FolderHandler) GetBreadcrumb(c echo.Context) error {
    folderID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid folder id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.getAncestors.Execute(c.Request().Context(), storage.GetFolderAncestorsInput{
        FolderID: folderID,
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.BreadcrumbResponse{
        Items: dto.ToBreadcrumbItems(output.Ancestors),
    })
}
```

### 4.2 ファイルハンドラー

```go
// internal/interface/handler/file_handler.go

package handler

import (
    "net/http"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "gc-storage/internal/interface/dto"
    "gc-storage/internal/interface/middleware"
    "gc-storage/internal/usecase/storage"
)

type FileHandler struct {
    initiateUpload  *storage.InitiateUploadUseCase
    completeUpload  *storage.CompleteUploadUseCase
    cancelUpload    *storage.CancelUploadUseCase
    downloadFile    *storage.DownloadFileUseCase
    moveFile        *storage.MoveFileUseCase
    copyFile        *storage.CopyFileUseCase
    renameFile      *storage.RenameFileUseCase
    trashFile       *storage.TrashFileUseCase
    restoreFile     *storage.RestoreFileUseCase
    deleteFile      *storage.PermanentlyDeleteFileUseCase
    listVersions    *storage.ListFileVersionsUseCase
    restoreVersion  *storage.RestoreFileVersionUseCase
}

// POST /api/v1/files/upload
func (h *FileHandler) InitiateUpload(c echo.Context) error {
    var req dto.InitiateUploadRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.initiateUpload.Execute(c.Request().Context(), storage.InitiateUploadInput{
        FileName:  req.FileName,
        FolderID:  req.FolderID,
        Size:      req.Size,
        MimeType:  req.MimeType,
        ActorID:   claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, dto.InitiateUploadResponse{
        FileID:      output.File.ID,
        SessionID:   output.Session.ID,
        UploadURL:   output.UploadURL,
        PartURLs:    output.PartURLs,
        IsMultipart: output.Session.IsMultipart,
        ExpiresAt:   output.Session.ExpiresAt,
    })
}

// POST /api/v1/files/upload/:sessionId/complete
func (h *FileHandler) CompleteUpload(c echo.Context) error {
    sessionID, err := uuid.Parse(c.Param("sessionId"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid session id")
    }

    var req dto.CompleteUploadRequest
    if err := c.Bind(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.completeUpload.Execute(c.Request().Context(), storage.CompleteUploadInput{
        SessionID: sessionID,
        Parts:     dto.ToCompletedParts(req.Parts),
        ActorID:   claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.FileResponse{
        ID:        output.File.ID,
        Name:      output.File.Name.String(),
        Size:      output.File.Size,
        MimeType:  output.File.MimeType.String(),
        Version:   output.File.CurrentVersion,
        CreatedAt: output.File.CreatedAt,
        UpdatedAt: output.File.UpdatedAt,
    })
}

// DELETE /api/v1/files/upload/:sessionId
func (h *FileHandler) CancelUpload(c echo.Context) error {
    sessionID, err := uuid.Parse(c.Param("sessionId"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid session id")
    }

    claims := middleware.GetClaims(c)
    err = h.cancelUpload.Execute(c.Request().Context(), storage.CancelUploadInput{
        SessionID: sessionID,
        ActorID:   claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// GET /api/v1/files/:id/download
func (h *FileHandler) Download(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
    }

    version := c.QueryParam("version")
    var versionNumber *int
    if version != "" {
        v, err := strconv.Atoi(version)
        if err != nil {
            return echo.NewHTTPError(http.StatusBadRequest, "invalid version number")
        }
        versionNumber = &v
    }

    claims := middleware.GetClaims(c)
    output, err := h.downloadFile.Execute(c.Request().Context(), storage.DownloadFileInput{
        FileID:        fileID,
        VersionNumber: versionNumber,
        ActorID:       claims.UserID,
        AsAttachment:  true,
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

// GET /api/v1/files/:id/preview
func (h *FileHandler) Preview(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.downloadFile.Execute(c.Request().Context(), storage.DownloadFileInput{
        FileID:       fileID,
        ActorID:      claims.UserID,
        AsAttachment: false,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.PreviewResponse{
        URL:      output.URL,
        MimeType: output.MimeType,
    })
}

// DELETE /api/v1/files/:id
func (h *FileHandler) Trash(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
    }

    claims := middleware.GetClaims(c)
    err = h.trashFile.Execute(c.Request().Context(), storage.TrashFileInput{
        FileID:  fileID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// POST /api/v1/files/:id/restore
func (h *FileHandler) Restore(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.restoreFile.Execute(c.Request().Context(), storage.RestoreFileInput{
        FileID:  fileID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.FileResponse{
        ID:        output.File.ID,
        Name:      output.File.Name.String(),
        Size:      output.File.Size,
        MimeType:  output.File.MimeType.String(),
        Version:   output.File.CurrentVersion,
        CreatedAt: output.File.CreatedAt,
        UpdatedAt: output.File.UpdatedAt,
    })
}

// GET /api/v1/files/:id/versions
func (h *FileHandler) ListVersions(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.listVersions.Execute(c.Request().Context(), storage.ListFileVersionsInput{
        FileID:  fileID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.VersionListResponse{
        Versions: dto.ToVersionResponses(output.Versions),
    })
}

// POST /api/v1/files/:id/versions/:version/restore
func (h *FileHandler) RestoreVersion(c echo.Context) error {
    fileID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
    }

    versionNumber, err := strconv.Atoi(c.Param("version"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid version number")
    }

    claims := middleware.GetClaims(c)
    output, err := h.restoreVersion.Execute(c.Request().Context(), storage.RestoreFileVersionInput{
        FileID:        fileID,
        VersionNumber: versionNumber,
        ActorID:       claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.FileResponse{
        ID:        output.File.ID,
        Name:      output.File.Name.String(),
        Size:      output.File.Size,
        MimeType:  output.File.MimeType.String(),
        Version:   output.File.CurrentVersion,
        CreatedAt: output.File.CreatedAt,
        UpdatedAt: output.File.UpdatedAt,
    })
}
```

---

## 5. DTO定義

```go
// internal/interface/dto/storage.go

package dto

import (
    "time"
    "github.com/google/uuid"
)

// Folder DTOs
type CreateFolderRequest struct {
    Name     string     `json:"name" validate:"required,foldername"`
    ParentID *uuid.UUID `json:"parent_id"`
}

type MoveFolderRequest struct {
    ParentID uuid.UUID `json:"parent_id" validate:"required"`
}

type RenameFolderRequest struct {
    Name string `json:"name" validate:"required,foldername"`
}

type FolderResponse struct {
    ID        uuid.UUID  `json:"id"`
    Name      string     `json:"name"`
    ParentID  *uuid.UUID `json:"parent_id,omitempty"`
    Path      string     `json:"path"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}

type FolderContentsResponse struct {
    Folders []FolderResponse `json:"folders"`
    Files   []FileResponse   `json:"files"`
}

type BreadcrumbItem struct {
    ID   uuid.UUID `json:"id"`
    Name string    `json:"name"`
}

type BreadcrumbResponse struct {
    Items []BreadcrumbItem `json:"items"`
}

// File DTOs
type InitiateUploadRequest struct {
    FileName string    `json:"file_name" validate:"required,filename"`
    FolderID uuid.UUID `json:"folder_id" validate:"required"`
    Size     int64     `json:"size" validate:"required,min=0"`
    MimeType string    `json:"mime_type" validate:"required"`
}

type InitiateUploadResponse struct {
    FileID      uuid.UUID `json:"file_id"`
    SessionID   uuid.UUID `json:"session_id"`
    UploadURL   string    `json:"upload_url,omitempty"`
    PartURLs    []string  `json:"part_urls,omitempty"`
    IsMultipart bool      `json:"is_multipart"`
    ExpiresAt   time.Time `json:"expires_at"`
}

type CompletedPartDTO struct {
    PartNumber int    `json:"part_number"`
    ETag       string `json:"etag"`
}

type CompleteUploadRequest struct {
    Parts []CompletedPartDTO `json:"parts,omitempty"`
}

type FileResponse struct {
    ID        uuid.UUID `json:"id"`
    Name      string    `json:"name"`
    Size      int64     `json:"size"`
    MimeType  string    `json:"mime_type"`
    Version   int       `json:"version"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type DownloadResponse struct {
    URL      string `json:"url"`
    FileName string `json:"file_name"`
    MimeType string `json:"mime_type"`
    Size     int64  `json:"size"`
}

type PreviewResponse struct {
    URL      string `json:"url"`
    MimeType string `json:"mime_type"`
}

type VersionResponse struct {
    VersionNumber int       `json:"version_number"`
    Size          int64     `json:"size"`
    CreatedBy     uuid.UUID `json:"created_by"`
    CreatedAt     time.Time `json:"created_at"`
}

type VersionListResponse struct {
    Versions []VersionResponse `json:"versions"`
}

// Trash DTOs
type TrashResponse struct {
    Folders []FolderResponse `json:"folders"`
    Files   []FileResponse   `json:"files"`
}
```

---

## 6. APIエンドポイント

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | /api/v1/folders | FolderHandler.Create | フォルダ作成 |
| GET | /api/v1/folders/:id | FolderHandler.Get | フォルダ取得 |
| PATCH | /api/v1/folders/:id | FolderHandler.Rename | フォルダ名変更 |
| PATCH | /api/v1/folders/:id/move | FolderHandler.Move | フォルダ移動 |
| DELETE | /api/v1/folders/:id | FolderHandler.Trash | ゴミ箱へ移動 |
| POST | /api/v1/folders/:id/restore | FolderHandler.Restore | 復元 |
| DELETE | /api/v1/folders/:id/permanent | FolderHandler.PermanentDelete | 完全削除 |
| GET | /api/v1/folders/:id/contents | FolderHandler.ListContents | 内容一覧 |
| GET | /api/v1/folders/:id/breadcrumb | FolderHandler.GetBreadcrumb | パンくずリスト |
| POST | /api/v1/files/upload | FileHandler.InitiateUpload | アップロード開始 |
| POST | /api/v1/files/upload/:sessionId/complete | FileHandler.CompleteUpload | アップロード完了 |
| DELETE | /api/v1/files/upload/:sessionId | FileHandler.CancelUpload | アップロードキャンセル |
| GET | /api/v1/files/:id | FileHandler.Get | ファイル情報取得 |
| PATCH | /api/v1/files/:id | FileHandler.Rename | ファイル名変更 |
| PATCH | /api/v1/files/:id/move | FileHandler.Move | ファイル移動 |
| POST | /api/v1/files/:id/copy | FileHandler.Copy | ファイルコピー |
| GET | /api/v1/files/:id/download | FileHandler.Download | ダウンロードURL取得 |
| GET | /api/v1/files/:id/preview | FileHandler.Preview | プレビューURL取得 |
| DELETE | /api/v1/files/:id | FileHandler.Trash | ゴミ箱へ移動 |
| POST | /api/v1/files/:id/restore | FileHandler.Restore | 復元 |
| DELETE | /api/v1/files/:id/permanent | FileHandler.PermanentDelete | 完全削除 |
| GET | /api/v1/files/:id/versions | FileHandler.ListVersions | バージョン一覧 |
| POST | /api/v1/files/:id/versions/:v/restore | FileHandler.RestoreVersion | バージョン復元 |
| GET | /api/v1/trash | TrashHandler.List | ゴミ箱一覧 |
| DELETE | /api/v1/trash | TrashHandler.Empty | ゴミ箱を空にする |

---

## 7. バックグラウンドジョブ

### 7.1 ゴミ箱自動削除

```go
// internal/job/trash_cleanup.go

package job

import (
    "context"
    "log/slog"
    "time"
    "gc-storage/internal/usecase/storage"
)

const TrashRetentionDays = 30

type TrashCleanupJob struct {
    cleanupUseCase *storage.CleanupTrashUseCase
    logger         *slog.Logger
}

func NewTrashCleanupJob(
    cleanupUseCase *storage.CleanupTrashUseCase,
    logger *slog.Logger,
) *TrashCleanupJob {
    return &TrashCleanupJob{
        cleanupUseCase: cleanupUseCase,
        logger:         logger,
    }
}

// Run executes daily at 3:00 AM
func (j *TrashCleanupJob) Run(ctx context.Context) error {
    threshold := time.Now().AddDate(0, 0, -TrashRetentionDays)

    output, err := j.cleanupUseCase.Execute(ctx, storage.CleanupTrashInput{
        OlderThan: threshold,
    })
    if err != nil {
        j.logger.Error("trash cleanup failed", "error", err)
        return err
    }

    j.logger.Info("trash cleanup completed",
        "deleted_folders", output.DeletedFolders,
        "deleted_files", output.DeletedFiles,
        "freed_bytes", output.FreedBytes,
    )
    return nil
}
```

### 7.2 期限切れアップロードセッションクリーンアップ

```go
// internal/job/session_cleanup.go

package job

import (
    "context"
    "log/slog"
    "gc-storage/internal/usecase/storage"
)

type SessionCleanupJob struct {
    cleanupUseCase *storage.CleanupExpiredSessionsUseCase
    logger         *slog.Logger
}

func NewSessionCleanupJob(
    cleanupUseCase *storage.CleanupExpiredSessionsUseCase,
    logger *slog.Logger,
) *SessionCleanupJob {
    return &SessionCleanupJob{
        cleanupUseCase: cleanupUseCase,
        logger:         logger,
    }
}

// Run executes every hour
func (j *SessionCleanupJob) Run(ctx context.Context) error {
    output, err := j.cleanupUseCase.Execute(ctx)
    if err != nil {
        j.logger.Error("session cleanup failed", "error", err)
        return err
    }

    if output.CleanedSessions > 0 {
        j.logger.Info("session cleanup completed",
            "cleaned_sessions", output.CleanedSessions,
            "aborted_uploads", output.AbortedUploads,
        )
    }
    return nil
}
```

---

## 8. 受け入れ基準

### フォルダ管理
- [ ] フォルダを作成できる（名前、親フォルダ指定）
- [ ] フォルダ名を変更できる
- [ ] フォルダを別のフォルダに移動できる
- [ ] 循環参照になる移動は拒否される
- [ ] 最大深さ（20）を超える移動は拒否される
- [ ] 同名フォルダがある場所への移動は拒否される
- [ ] ルートフォルダは移動・削除できない

### ファイル管理
- [ ] ファイルをアップロードできる（100MB未満: 単一、以上: マルチパート）
- [ ] Presigned URLでMinIOに直接アップロードできる
- [ ] アップロード完了後にファイルがactiveになる
- [ ] ファイルをダウンロードできる（Presigned URL）
- [ ] ファイルをプレビューできる
- [ ] ファイル名を変更できる
- [ ] ファイルを別フォルダに移動できる
- [ ] ファイルをコピーできる

### バージョン管理
- [ ] 同名ファイルアップロードで新バージョン作成
- [ ] バージョン一覧を取得できる
- [ ] 特定バージョンをダウンロードできる
- [ ] 特定バージョンを最新版に復元できる
- [ ] 古いバージョンを削除できる（最新版以外）

### ゴミ箱管理
- [ ] フォルダをゴミ箱に移動できる（配下も連動）
- [ ] ファイルをゴミ箱に移動できる
- [ ] ゴミ箱から復元できる
- [ ] 親フォルダがゴミ箱にある場合、復元時にエラー
- [ ] ゴミ箱から完全削除できる（MinIOも削除）
- [ ] ゴミ箱を空にできる
- [ ] 30日経過したゴミ箱アイテムは自動削除

### 非機能要件
- [ ] アップロードセッションは1時間で期限切れ
- [ ] 期限切れセッションは自動クリーンアップ
- [ ] フォルダ操作はトランザクションで整合性確保
- [ ] MinIO削除失敗時もDB削除は実行（孤児オブジェクト許容）

---

## 関連ドキュメント

- [フォルダドメイン](../03-domains/folder.md)
- [ファイルドメイン](../03-domains/file.md)
- [MinIO仕様](./infra-minio.md)
- [API仕様](./infra-api.md)
- [データベース設計](../02-architecture/DATABASE.md)
