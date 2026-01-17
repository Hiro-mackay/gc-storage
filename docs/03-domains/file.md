# File ドメイン

## 概要

Fileドメインは、ファイルのアップロード、ダウンロード、バージョン管理、メタデータ管理を担当します。
Storage Contextの中核として、ファイルの実体（MinIO）とメタデータ（PostgreSQL）の整合性を保証します。

---

## エンティティ

### File（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | ファイルの一意識別子 |
| name | string | Yes | ファイル名 (1-255文字) |
| folder_id | UUID | Yes | 所属フォルダID |
| owner_id | UUID | Yes | 所有者ID |
| size | int64 | Yes | ファイルサイズ（バイト） |
| mime_type | string | Yes | MIMEタイプ |
| storage_key | string | Yes | MinIO内のオブジェクトキー |
| current_version | int | Yes | 現在のバージョン番号 |
| status | FileStatus | Yes | ファイル状態 |
| trashed_at | timestamp | No | ゴミ箱移動日時 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ビジネスルール:**
- R-FL001: nameは空文字不可、1-255文字
- R-FL002: nameに禁止文字（/ \ : * ? " < > |）を含まない
- R-FL003: storage_keyは全ファイルで一意
- R-FL004: 同一フォルダ内でnameは一意
- R-FL005: sizeは0以上（0バイトファイルは許容）
- R-FL006: current_versionは1以上の連番
- R-FL007: statusがpendingからactiveへの遷移はアップロード完了時のみ

**ステータス遷移:**
```
┌─────────┐       ┌─────────┐       ┌─────────┐       ┌─────────┐
│ pending │──────▶│  active │──────▶│ trashed │──────▶│ deleted │
└─────────┘       └────┬────┘       └────┬────┘       └─────────┘
     │                 │                 │
     │  upload failed  │                 │ restore
     ▼                 │◀────────────────┘
┌─────────┐            │
│  failed │            │
└─────────┘            │
```

| ステータス | 説明 |
|-----------|------|
| pending | アップロード中 |
| active | アクティブ（通常利用可能） |
| trashed | ゴミ箱（復元可能） |
| deleted | 完全削除済み |
| failed | アップロード失敗 |

### FileVersion

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | バージョンの一意識別子 |
| file_id | UUID | Yes | ファイルID |
| version_number | int | Yes | バージョン番号 |
| size | int64 | Yes | このバージョンのサイズ |
| storage_key | string | Yes | MinIO内のオブジェクトキー |
| checksum_md5 | string | Yes | MD5チェックサム |
| checksum_sha256 | string | Yes | SHA256チェックサム |
| created_by | UUID | Yes | アップロードしたユーザーID |
| created_at | timestamp | Yes | 作成日時 |

**ビジネスルール:**
- R-FV001: version_numberは同一file_id内で一意かつ連番
- R-FV002: storage_keyは全バージョンで一意
- R-FV003: バージョン削除は最新バージョン以外のみ可能
- R-FV004: 保持バージョン数は設定可能（デフォルト10）

### FileMetadata

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| file_id | UUID | Yes | ファイルID（主キー） |
| checksum_md5 | string | Yes | MD5チェックサム |
| checksum_sha256 | string | Yes | SHA256チェックサム |
| width | int | No | 画像/動画の幅（ピクセル） |
| height | int | No | 画像/動画の高さ（ピクセル） |
| duration | int | No | 音声/動画の長さ（秒） |
| exif_data | jsonb | No | EXIF情報（画像） |
| custom_properties | jsonb | No | カスタムメタデータ |
| extracted_at | timestamp | Yes | メタデータ抽出日時 |

**ビジネスルール:**
- R-FM001: checksumはアップロード完了時に必ず計算
- R-FM002: width/heightは画像・動画ファイルのみ
- R-FM003: durationは音声・動画ファイルのみ

### UploadSession

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | セッションID |
| file_id | UUID | Yes | 対象ファイルID |
| upload_id | string | No | MinIOのマルチパートアップロードID |
| is_multipart | boolean | Yes | マルチパートアップロードかどうか |
| total_parts | int | No | マルチパートの総パート数 |
| uploaded_parts | int | No | アップロード済みパート数 |
| expires_at | timestamp | Yes | セッション有効期限 |
| status | UploadSessionStatus | Yes | セッション状態 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ビジネスルール:**
- R-US001: expires_atを過ぎたセッションは自動キャンセル
- R-US002: is_multipart=trueの場合、upload_idは必須
- R-US003: セッション完了後は状態変更不可

---

## 値オブジェクト

### FileName

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | ファイル名文字列 |

**バリデーション:**
- 1-255文字
- 禁止文字（/ \ : * ? " < > |）を含まない
- 先頭・末尾の空白はトリム

```go
type FileName struct {
    value string
}

func NewFileName(value string) (FileName, error) {
    trimmed := strings.TrimSpace(value)

    if len(trimmed) == 0 {
        return FileName{}, errors.New("file name cannot be empty")
    }
    if len(trimmed) > 255 {
        return FileName{}, errors.New("file name must not exceed 255 characters")
    }

    invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
    if invalidChars.MatchString(trimmed) {
        return FileName{}, errors.New("file name contains invalid characters")
    }

    return FileName{value: trimmed}, nil
}

func (f FileName) Extension() string {
    idx := strings.LastIndex(f.value, ".")
    if idx == -1 || idx == len(f.value)-1 {
        return ""
    }
    return strings.ToLower(f.value[idx+1:])
}
```

### StorageKey

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | MinIO内のオブジェクトキー |

**形式:** `{owner_type}/{owner_id}/{file_id}/{version}`

```go
type StorageKey struct {
    value string
}

func NewStorageKey(ownerType string, ownerID, fileID uuid.UUID, version int) StorageKey {
    return StorageKey{
        value: fmt.Sprintf("%s/%s/%s/v%d", ownerType, ownerID, fileID, version),
    }
}
```

### MimeType

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | MIMEタイプ文字列 |

```go
type MimeType struct {
    value string
}

func NewMimeType(value string) MimeType {
    return MimeType{value: strings.ToLower(value)}
}

func (m MimeType) IsImage() bool {
    return strings.HasPrefix(m.value, "image/")
}

func (m MimeType) IsVideo() bool {
    return strings.HasPrefix(m.value, "video/")
}

func (m MimeType) IsAudio() bool {
    return strings.HasPrefix(m.value, "audio/")
}

func (m MimeType) IsPreviewable() bool {
    previewable := []string{
        "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml",
        "application/pdf", "text/plain", "text/markdown",
    }
    for _, p := range previewable {
        if m.value == p {
            return true
        }
    }
    return false
}
```

### FileStatus

| 値 | 説明 |
|-----|------|
| pending | アップロード中 |
| active | アクティブ |
| trashed | ゴミ箱 |
| deleted | 削除済み |
| failed | アップロード失敗 |

### UploadSessionStatus

| 値 | 説明 |
|-----|------|
| in_progress | アップロード中 |
| completed | 完了 |
| cancelled | キャンセル |
| expired | 期限切れ |

---

## ドメインサービス

### FileUploadService

**責務:** ファイルアップロードのライフサイクル管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| InitiateUpload | cmd | (File, UploadSession, PresignedURL) | アップロード開始 |
| InitiateMultipartUpload | cmd | (File, UploadSession, PartURLs) | マルチパート開始 |
| GetPartUploadURL | sessionId, partNumber | PresignedURL | パートURL取得 |
| CompleteUpload | sessionId | File | アップロード完了 |
| CompleteMultipartUpload | sessionId, parts | File | マルチパート完了 |
| CancelUpload | sessionId | void | アップロードキャンセル |

```go
type FileUploadService interface {
    InitiateUpload(ctx context.Context, cmd InitiateUploadCommand) (*File, *UploadSession, string, error)
    InitiateMultipartUpload(ctx context.Context, cmd InitiateMultipartUploadCommand) (*File, *UploadSession, []string, error)
    GetPartUploadURL(ctx context.Context, sessionID uuid.UUID, partNumber int) (string, error)
    CompleteUpload(ctx context.Context, sessionID uuid.UUID) (*File, error)
    CompleteMultipartUpload(ctx context.Context, sessionID uuid.UUID, parts []CompletedPart) (*File, error)
    CancelUpload(ctx context.Context, sessionID uuid.UUID) error
}
```

**アップロード開始の処理:**
```go
func (s *FileUploadServiceImpl) InitiateUpload(
    ctx context.Context,
    cmd InitiateUploadCommand,
) (*File, *UploadSession, string, error) {
    // 1. フォルダ存在・権限チェック
    folder, err := s.folderRepo.FindByID(ctx, cmd.FolderID)
    if err != nil {
        return nil, nil, "", err
    }
    if folder.Status != FolderStatusActive {
        return nil, nil, "", errors.New("folder is not active")
    }

    // 2. 同名ファイルチェック（バージョン作成 or エラー）
    existingFile, _ := s.fileRepo.FindByNameAndFolder(ctx, cmd.Name, cmd.FolderID)
    var file *File

    if existingFile != nil {
        // 既存ファイルの新バージョンとして処理
        file = existingFile
        file.CurrentVersion++
        file.UpdatedAt = time.Now()
    } else {
        // 新規ファイル作成
        file = &File{
            ID:             uuid.New(),
            Name:           cmd.Name,
            FolderID:       cmd.FolderID,
            OwnerID:        cmd.OwnerID,
            Size:           cmd.Size,
            MimeType:       cmd.MimeType,
            CurrentVersion: 1,
            Status:         FileStatusPending,
            CreatedAt:      time.Now(),
            UpdatedAt:      time.Now(),
        }
    }

    // 3. StorageKey生成
    storageKey := NewStorageKey(
        string(folder.OwnerType),
        folder.OwnerID,
        file.ID,
        file.CurrentVersion,
    )
    file.StorageKey = storageKey.String()

    // 4. ファイルレコード作成/更新
    if existingFile != nil {
        if err := s.fileRepo.Update(ctx, file); err != nil {
            return nil, nil, "", err
        }
    } else {
        if err := s.fileRepo.Create(ctx, file); err != nil {
            return nil, nil, "", err
        }
    }

    // 5. アップロードセッション作成
    session := &UploadSession{
        ID:          uuid.New(),
        FileID:      file.ID,
        IsMultipart: false,
        ExpiresAt:   time.Now().Add(1 * time.Hour),
        Status:      UploadSessionStatusInProgress,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    if err := s.sessionRepo.Create(ctx, session); err != nil {
        return nil, nil, "", err
    }

    // 6. Presigned URL生成
    presignedURL, err := s.storageClient.GeneratePutURL(ctx, storageKey.String(), 15*time.Minute)
    if err != nil {
        return nil, nil, "", err
    }

    // 7. イベント発行
    s.eventPublisher.Publish(FileUploadStartedEvent{
        FileID:    file.ID,
        SessionID: session.ID,
        Name:      file.Name.String(),
        Size:      file.Size,
    })

    return file, session, presignedURL, nil
}
```

**アップロード完了の処理:**
```go
func (s *FileUploadServiceImpl) CompleteUpload(
    ctx context.Context,
    sessionID uuid.UUID,
) (*File, error) {
    return s.txManager.WithTransaction(ctx, func(ctx context.Context) (*File, error) {
        // 1. セッション取得
        session, err := s.sessionRepo.FindByID(ctx, sessionID)
        if err != nil {
            return nil, err
        }
        if session.Status != UploadSessionStatusInProgress {
            return nil, errors.New("upload session is not in progress")
        }
        if session.ExpiresAt.Before(time.Now()) {
            return nil, errors.New("upload session expired")
        }

        // 2. ファイル取得
        file, err := s.fileRepo.FindByID(ctx, session.FileID)
        if err != nil {
            return nil, err
        }

        // 3. オブジェクト存在確認
        exists, err := s.storageClient.ObjectExists(ctx, file.StorageKey)
        if err != nil || !exists {
            return nil, errors.New("uploaded object not found")
        }

        // 4. メタデータ取得・検証
        objectInfo, err := s.storageClient.GetObjectInfo(ctx, file.StorageKey)
        if err != nil {
            return nil, err
        }

        // 5. ファイルサイズ更新（実際のサイズに合わせる）
        file.Size = objectInfo.Size
        file.Status = FileStatusActive
        file.UpdatedAt = time.Now()

        // 6. バージョンレコード作成
        version := &FileVersion{
            ID:             uuid.New(),
            FileID:         file.ID,
            VersionNumber:  file.CurrentVersion,
            Size:           file.Size,
            StorageKey:     file.StorageKey,
            ChecksumMD5:    objectInfo.MD5,
            ChecksumSHA256: objectInfo.SHA256,
            CreatedBy:      file.OwnerID,
            CreatedAt:      time.Now(),
        }
        if err := s.versionRepo.Create(ctx, version); err != nil {
            return nil, err
        }

        // 7. メタデータ抽出・保存
        metadata := s.extractMetadata(ctx, file, objectInfo)
        if err := s.metadataRepo.Upsert(ctx, metadata); err != nil {
            // メタデータ抽出失敗は警告のみ
            log.Warn("failed to extract metadata", "error", err)
        }

        // 8. ファイル更新
        if err := s.fileRepo.Update(ctx, file); err != nil {
            return nil, err
        }

        // 9. セッション完了
        session.Status = UploadSessionStatusCompleted
        session.UpdatedAt = time.Now()
        if err := s.sessionRepo.Update(ctx, session); err != nil {
            return nil, err
        }

        // 10. イベント発行
        s.eventPublisher.Publish(FileUploadedEvent{
            FileID:        file.ID,
            Name:          file.Name.String(),
            Size:          file.Size,
            VersionNumber: file.CurrentVersion,
        })

        return file, nil
    })
}
```

### FileDownloadService

**責務:** ファイルダウンロードの管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| GetDownloadURL | fileId, versionNumber | PresignedURL | ダウンロードURL取得 |
| GetPreviewURL | fileId | PresignedURL | プレビューURL取得 |

```go
type FileDownloadService interface {
    GetDownloadURL(ctx context.Context, fileID uuid.UUID, versionNumber *int) (string, error)
    GetPreviewURL(ctx context.Context, fileID uuid.UUID) (string, error)
}
```

### FileVersionService

**責務:** ファイルバージョンの管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| ListVersions | fileId | []FileVersion | バージョン一覧 |
| RestoreVersion | fileId, versionNumber | File | 特定バージョンを最新に |
| DeleteVersion | fileId, versionNumber | void | バージョン削除 |
| CleanupOldVersions | fileId, keepCount | int | 古いバージョン削除 |

```go
type FileVersionService interface {
    ListVersions(ctx context.Context, fileID uuid.UUID) ([]*FileVersion, error)
    RestoreVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) (*File, error)
    DeleteVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) error
    CleanupOldVersions(ctx context.Context, fileID uuid.UUID, keepCount int) (int, error)
}
```

### FileTrashService

**責務:** ファイルのゴミ箱管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| TrashFile | fileId | void | ゴミ箱へ移動 |
| RestoreFile | fileId | File | 復元 |
| PermanentlyDelete | fileId | void | 完全削除 |
| TrashByFolderID | folderId | void | フォルダ内ファイル一括ゴミ箱 |

```go
type FileTrashService interface {
    TrashFile(ctx context.Context, fileID uuid.UUID) error
    RestoreFile(ctx context.Context, fileID uuid.UUID) (*File, error)
    PermanentlyDelete(ctx context.Context, fileID uuid.UUID) error
    TrashByFolderID(ctx context.Context, folderID uuid.UUID) error
}
```

---

## リポジトリ

### FileRepository

```go
type FileRepository interface {
    // 基本CRUD
    Create(ctx context.Context, file *File) error
    FindByID(ctx context.Context, id uuid.UUID) (*File, error)
    Update(ctx context.Context, file *File) error
    Delete(ctx context.Context, id uuid.UUID) error

    // 検索
    FindByFolderID(ctx context.Context, folderID uuid.UUID, status FileStatus) ([]*File, error)
    FindByNameAndFolder(ctx context.Context, name FileName, folderID uuid.UUID) (*File, error)
    FindByOwnerID(ctx context.Context, ownerID uuid.UUID, status FileStatus) ([]*File, error)

    // 検索（全文検索対応）
    Search(ctx context.Context, query FileSearchQuery) ([]*File, int64, error)

    // ゴミ箱
    FindTrashedByOwner(ctx context.Context, ownerID uuid.UUID) ([]*File, error)
    FindTrashedOlderThan(ctx context.Context, threshold time.Time) ([]*File, error)
}
```

### FileVersionRepository

```go
type FileVersionRepository interface {
    Create(ctx context.Context, version *FileVersion) error
    FindByID(ctx context.Context, id uuid.UUID) (*FileVersion, error)
    FindByFileID(ctx context.Context, fileID uuid.UUID) ([]*FileVersion, error)
    FindByFileAndVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) (*FileVersion, error)
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteOlderVersions(ctx context.Context, fileID uuid.UUID, keepCount int) (int64, error)
}
```

### FileMetadataRepository

```go
type FileMetadataRepository interface {
    Upsert(ctx context.Context, metadata *FileMetadata) error
    FindByFileID(ctx context.Context, fileID uuid.UUID) (*FileMetadata, error)
    Delete(ctx context.Context, fileID uuid.UUID) error
}
```

### UploadSessionRepository

```go
type UploadSessionRepository interface {
    Create(ctx context.Context, session *UploadSession) error
    FindByID(ctx context.Context, id uuid.UUID) (*UploadSession, error)
    Update(ctx context.Context, session *UploadSession) error
    Delete(ctx context.Context, id uuid.UUID) error
    FindExpired(ctx context.Context) ([]*UploadSession, error)
    DeleteExpired(ctx context.Context) (int64, error)
}
```

---

## 関係性

### エンティティ関係図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           File Domain ERD                                    │
└─────────────────────────────────────────────────────────────────────────────┘

      ┌──────────────────┐          ┌──────────────────┐
      │     folders      │          │      users       │
      │    (external)    │          │    (external)    │
      └────────┬─────────┘          └────────┬─────────┘
               │                             │
               │ folder_id                   │ owner_id, created_by
               │                             │
               └──────────────┬──────────────┘
                              │
                              ▼
                     ┌──────────────────┐
                     │      files       │
                     ├──────────────────┤
                     │ id               │
                     │ name             │
                     │ folder_id (FK)   │
                     │ owner_id (FK)    │
                     │ size             │
                     │ mime_type        │
                     │ storage_key      │
                     │ current_version  │
                     │ status           │
                     │ trashed_at       │
                     │ created_at       │
                     │ updated_at       │
                     └────────┬─────────┘
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
         ▼                    ▼                    ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│  file_versions   │ │  file_metadata   │ │ upload_sessions  │
├──────────────────┤ ├──────────────────┤ ├──────────────────┤
│ id               │ │ file_id (PK,FK)  │ │ id               │
│ file_id (FK)     │ │ checksum_md5     │ │ file_id (FK)     │
│ version_number   │ │ checksum_sha256  │ │ upload_id        │
│ size             │ │ width            │ │ is_multipart     │
│ storage_key      │ │ height           │ │ total_parts      │
│ checksum_md5     │ │ duration         │ │ uploaded_parts   │
│ checksum_sha256  │ │ exif_data        │ │ expires_at       │
│ created_by (FK)  │ │ custom_properties│ │ status           │
│ created_at       │ │ extracted_at     │ │ created_at       │
└──────────────────┘ └──────────────────┘ │ updated_at       │
                                          └──────────────────┘
```

### 関係性ルール

| 関係 | カーディナリティ | 説明 |
|-----|----------------|------|
| File - Folder | N:1 | 各ファイルは1つのフォルダに所属 |
| File - Owner (User) | N:1 | 各ファイルは1人の所有者を持つ |
| File - FileVersion | 1:N | 各ファイルは複数のバージョンを持てる |
| File - FileMetadata | 1:1 | 各ファイルは1つのメタデータを持つ |
| File - UploadSession | 1:N | 各ファイルは複数のアップロードセッションを持てる |

---

## 不変条件

1. **ストレージ整合性**
   - storage_keyは全ファイル・全バージョンで一意
   - ファイル削除時、対応するMinIOオブジェクトも削除
   - checksumは必ず計算・保存

2. **バージョン整合性**
   - version_numberは連番（ギャップなし）
   - current_versionは存在するバージョンを指す
   - 最新バージョンは削除不可

3. **ステータス整合性**
   - pending状態のファイルはダウンロード不可
   - failed状態のファイルは自動クリーンアップ対象
   - trashed状態のファイルは30日後に自動削除

4. **命名制約**
   - 同一フォルダ内でファイル名は一意
   - 禁止文字を含まない

5. **アップロードセッション**
   - 期限切れセッションは自動キャンセル
   - マルチパートアップロードの中断時はMinIO側もクリーンアップ

---

## ユースケース概要

| ユースケース | アクター | 概要 |
|------------|--------|------|
| InitiateUpload | User | アップロード開始（Presigned URL取得） |
| CompleteUpload | User | アップロード完了通知 |
| InitiateMultipartUpload | User | 大容量ファイルのマルチパート開始 |
| CompleteMultipartUpload | User | マルチパートアップロード完了 |
| CancelUpload | User | アップロードキャンセル |
| DownloadFile | User | ファイルダウンロード |
| PreviewFile | User | ファイルプレビュー |
| RenameFile | User | ファイル名変更 |
| MoveFile | User | ファイル移動 |
| CopyFile | User | ファイルコピー |
| TrashFile | User | ゴミ箱へ移動 |
| RestoreFile | User | ゴミ箱から復元 |
| PermanentlyDeleteFile | User | 完全削除 |
| ListVersions | User | バージョン一覧 |
| RestoreVersion | User | 特定バージョン復元 |
| GetFileMetadata | User | メタデータ取得 |
| SearchFiles | User | ファイル検索 |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| FileUploadStarted | アップロード開始 | fileId, sessionId, name, size |
| FileUploaded | アップロード完了 | fileId, name, size, versionNumber |
| FileUploadFailed | アップロード失敗 | fileId, sessionId, reason |
| FileDownloaded | ダウンロード | fileId, downloadedBy, versionNumber |
| FileRenamed | 名前変更 | fileId, oldName, newName |
| FileMoved | 移動 | fileId, oldFolderId, newFolderId |
| FileCopied | コピー | sourceFileId, newFileId |
| FileTrashed | ゴミ箱移動 | fileId |
| FileRestored | 復元 | fileId |
| FilePermanentlyDeleted | 完全削除 | fileId |
| FileVersionCreated | バージョン作成 | fileId, versionNumber |
| FileVersionRestored | バージョン復元 | fileId, versionNumber |
| FileVersionDeleted | バージョン削除 | fileId, versionNumber |

---

## 他コンテキストとの連携

### Folder Domain（同一コンテキスト）
- ファイルはフォルダに所属
- フォルダ削除時、配下のファイルも削除

### Identity Context（上流）
- UserIDの参照（所有者、作成者）

### Authorization Context（下流）
- ファイルに対する権限付与
- 親フォルダからの権限継承

### Sharing Context（下流）
- ファイルの共有リンク作成

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [フォルダドメイン](./folder.md) - フォルダ管理
- [共有ドメイン](./sharing.md) - 共有機能
- [システムアーキテクチャ](../02-architecture/SYSTEM.md) - アップロード/ダウンロードフロー
