# Storage File - 詳細設計

## メタ情報

| 項目 | 内容 |
|------|------|
| ドキュメントID | SPEC-002 |
| バージョン | 1.0.0 |
| 最終更新日 | 2026-01-20 |
| ステータス | Draft |
| 関連ドメイン | [File Domain](../03-domains/file.md) |

---

## ユーザーストーリー

### US-F001: ファイルアップロード

**As a** ユーザー
**I want to** ファイルをアップロードする
**So that** クラウドストレージにファイルを保存できる

**受け入れ条件:**
- 5MB未満のファイルはシングルパートでアップロードできる
- 5MB以上のファイルはマルチパートでアップロードできる
- アップロード完了はサーバーが自動検知する
- クライアントはポーリングで完了を確認できる

### US-F002: ファイルダウンロード

**As a** ユーザー
**I want to** ファイルをダウンロードする
**So that** ストレージのファイルをローカルで使用できる

**受け入れ条件:**
- Presigned URLでダウンロードできる
- 特定バージョンをダウンロードできる
- アップロード中のファイルはダウンロードできない

### US-F003: ファイルをゴミ箱へ移動

**As a** ユーザー
**I want to** ファイルをゴミ箱へ移動する
**So that** 誤削除時に復元できる

**受け入れ条件:**
- ファイルと全バージョンがアーカイブテーブルに移動
- 30日後に自動削除される
- MinIOオブジェクトは削除されない

### US-F004: ファイル復元

**As a** ユーザー
**I want to** ゴミ箱からファイルを復元する
**So that** 誤削除したファイルを回復できる

**受け入れ条件:**
- 元のフォルダに復元される
- フォルダが存在しない場合はルートに復元される
- 全バージョンが復元される

### US-F005: ファイル完全削除

**As a** ユーザー
**I want to** ファイルを完全に削除する
**So that** 不要なファイルを完全に消去できる

**受け入れ条件:**
- MinIOオブジェクトが全バージョン削除される
- データベースから完全に削除される
- 復元不可能になる

---

## API仕様

### アップロードアーキテクチャ

ファイルアップロードはPresigned URLを使用し、クライアントがMinIOへ直接アップロードする方式を採用。APIサーバーはファイルデータを経由しない。

```
┌─────────┐          ┌─────────┐          ┌─────────┐
│ Client  │          │   API   │          │  MinIO  │
└────┬────┘          └────┬────┘          └────┬────┘
     │                    │                    │
     │ 1. InitiateUpload  │                    │
     │ ─────────────────> │                    │
     │                    │ 2. Presigned URL生成│
     │                    │ ─────────────────> │
     │ 3. presigned_url   │                    │
     │ <───────────────── │                    │
     │                    │                    │
     │ 4. PUT (直接アップロード)                │
     │ ─────────────────────────────────────> │
     │                    │                    │
     │                    │ 5. Webhook通知     │
     │                    │ <───────────────── │
     │                    │                    │
     │ 6. GetUploadStatus │                    │
     │ ─────────────────> │                    │
     │ 7. completed       │                    │
     │ <───────────────── │                    │
```

| ステップ | 実行者 | 処理内容 |
|---------|--------|---------|
| 1 | Client → API | アップロード開始リクエスト |
| 2 | API | File/UploadSession作成、MinIO Presigned URL生成 |
| 3 | API → Client | `presigned_url` を返却 |
| 4 | Client → MinIO | **クライアントが直接MinIOへHTTP PUT** |
| 5 | MinIO → API | Bucket Notification（Webhook）で完了通知 |
| 6-7 | Client → API | ポーリングで完了確認 |

**設計理由:**
- **帯域効率**: ファイルデータがAPIサーバーを経由しない
- **スケーラビリティ**: APIサーバーの負荷軽減
- **大容量対応**: マルチパートアップロードでGBサイズ対応可能

---

### POST /api/v1/files/upload/initiate

アップロードセッションを開始し、Presigned URLを取得する。このエンドポイントは**アップロードの準備のみ**を行い、実際のファイルアップロードは実行しない。

**Request:**
```json
{
  "folder_id": "uuid",
  "name": "report.pdf",
  "mime_type": "application/pdf",
  "size": 10485760
}
```

**Note:**
- `folder_id` は必須（ファイルは必ずフォルダに所属）

**Response (201 Created):**
```json
{
  "session_id": "uuid",
  "file_id": "uuid",
  "is_multipart": true,
  "upload_urls": [
    {
      "part_number": 1,
      "url": "https://minio.example.com/bucket/...",
      "expires_at": "2026-01-21T00:00:00Z"
    }
  ],
  "expires_at": "2026-01-21T00:00:00Z"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 400 | 無効なファイル名、MIMEタイプ |
| 403 | フォルダへの書き込み権限なし |
| 404 | フォルダが存在しない |
| 409 | 同一フォルダ内に同名ファイルが存在 |

---

### GET /api/v1/files/upload/{session_id}/status

アップロードセッションの状態を取得する（ポーリング用）。

**Response (200 OK):**
```json
{
  "session_id": "uuid",
  "status": "completed",
  "file_id": "uuid",
  "progress": {
    "uploaded_parts": 3,
    "total_parts": 3
  }
}
```

**ステータス値:**
| 値 | 説明 |
|-----|------|
| pending | 初期化済み、アップロード未開始 |
| in_progress | アップロード中（マルチパート） |
| completed | 完了 |
| aborted | 中断 |
| expired | 期限切れ |

---

### POST /api/v1/files/upload/{session_id}/abort

アップロードをキャンセルする。

**Response (204 No Content)**

**エラー:**
| コード | 条件 |
|--------|------|
| 404 | セッションが存在しない |
| 409 | 既に完了またはキャンセル済み |

---

### GET /api/v1/files/{file_id}/download

ファイルのダウンロードURLを取得する。

**Query Parameters:**
| パラメータ | 型 | 説明 |
|-----------|-----|------|
| version | int | バージョン番号（省略時は最新） |

**Response (200 OK):**
```json
{
  "download_url": "https://minio.example.com/bucket/...",
  "expires_at": "2026-01-20T01:00:00Z",
  "file_name": "report.pdf",
  "mime_type": "application/pdf",
  "size": 10485760
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | ファイルへのアクセス権限なし |
| 404 | ファイルまたはバージョンが存在しない |
| 409 | ファイルがアップロード中 |

---

### PUT /api/v1/files/{file_id}/name

ファイル名を変更する。

**Request:**
```json
{
  "name": "new_report.pdf"
}
```

**Response (200 OK):**
```json
{
  "id": "uuid",
  "name": "new_report.pdf",
  "updated_at": "2026-01-20T00:00:00Z"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 400 | 無効なファイル名 |
| 403 | ファイルへの書き込み権限なし |
| 404 | ファイルが存在しない |
| 409 | 同一フォルダ内に同名ファイルが存在 |

---

### PUT /api/v1/files/{file_id}/folder

ファイルを別フォルダへ移動する。

**Request:**
```json
{
  "folder_id": "uuid"
}
```

**Note:**
- `folder_id` は必須（ファイルは必ずフォルダに所属）

**Response (200 OK):**
```json
{
  "id": "uuid",
  "folder_id": "uuid",
  "updated_at": "2026-01-20T00:00:00Z"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | ファイルまたはフォルダへの権限なし |
| 404 | ファイルまたはフォルダが存在しない |
| 409 | 移動先に同名ファイルが存在 |

---

### POST /api/v1/files/{file_id}/trash

ファイルをゴミ箱へ移動する。

**Response (200 OK):**
```json
{
  "archived_file_id": "uuid",
  "expires_at": "2026-02-19T00:00:00Z"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | ファイルへの削除権限なし |
| 404 | ファイルが存在しない |

---

### POST /api/v1/trash/files/{archived_file_id}/restore

ゴミ箱からファイルを復元する。

**Response (200 OK):**
```json
{
  "file_id": "uuid",
  "folder_id": "uuid | null",
  "name": "report.pdf"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | 復元権限なし |
| 404 | アーカイブファイルが存在しない |
| 409 | 復元先に同名ファイルが存在 |

---

### DELETE /api/v1/trash/files/{archived_file_id}

ファイルを完全に削除する。

**Response (204 No Content)**

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | 削除権限なし |
| 404 | アーカイブファイルが存在しない |

---

### GET /api/v1/files/{file_id}/versions

ファイルのバージョン一覧を取得する。

**Response (200 OK):**
```json
{
  "versions": [
    {
      "version_number": 2,
      "size": 10485760,
      "checksum": "sha256:...",
      "uploaded_by": "uuid",
      "created_at": "2026-01-20T00:00:00Z"
    },
    {
      "version_number": 1,
      "size": 8388608,
      "checksum": "sha256:...",
      "uploaded_by": "uuid",
      "created_at": "2026-01-15T00:00:00Z"
    }
  ]
}
```

---

### GET /api/v1/trash

ゴミ箱の内容を取得する。

**Query Parameters:**
| パラメータ | 型 | 説明 |
|-----------|-----|------|
| limit | int | 取得件数（デフォルト: 50） |
| cursor | string | ページネーションカーソル |

**Response (200 OK):**
```json
{
  "items": [
    {
      "id": "uuid",
      "type": "file",
      "name": "report.pdf",
      "original_path": "/documents/report.pdf",
      "size": 10485760,
      "archived_at": "2026-01-20T00:00:00Z",
      "expires_at": "2026-02-19T00:00:00Z"
    }
  ],
  "next_cursor": "string | null"
}
```

---

### DELETE /api/v1/trash

ゴミ箱を空にする。

**Response (202 Accepted):**
```json
{
  "message": "Trash emptying started",
  "deleted_count": 15
}
```

---

### POST /internal/webhooks/minio

MinIOからのWebhook通知を受信する内部エンドポイント。

**Request (MinIO S3 Event):**
```json
{
  "Records": [
    {
      "eventVersion": "2.0",
      "eventSource": "minio:s3",
      "eventName": "s3:ObjectCreated:Put",
      "s3": {
        "bucket": {
          "name": "files"
        },
        "object": {
          "key": "uuid-file-id",
          "size": 10485760,
          "eTag": "etag-value",
          "versionId": "minio-version-id"
        }
      }
    }
  ]
}
```

**Response (200 OK)**

---

## データ変更

### 新規テーブル

#### files

```sql
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    folder_id UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,  -- 必須（ファイルは必ずフォルダに所属）
    owner_id UUID NOT NULL REFERENCES users(id),      -- 現在の所有者（所有権譲渡で変更可能）
    created_by UUID NOT NULL REFERENCES users(id),    -- 最初の作成者（不変、履歴追跡用）
    name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL CHECK (size >= 0),
    storage_key VARCHAR(1024) NOT NULL UNIQUE,
    current_version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'uploading' CHECK (status IN ('uploading', 'active', 'upload_failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_files_name_folder UNIQUE (folder_id, name)
);

-- Note: owner_type は削除（ファイルは常にユーザーが作成者）
-- Note: folder_id は必須（NOT NULL）- ファイルは必ずフォルダに所属
-- Note: グループはPermissionGrantで関連付けてアクセス

CREATE INDEX idx_files_folder ON files(folder_id);
CREATE INDEX idx_files_owner ON files(owner_id);
CREATE INDEX idx_files_created_by ON files(created_by);
CREATE INDEX idx_files_status ON files(status);
CREATE INDEX idx_files_storage_key ON files(storage_key);
```

#### file_versions

```sql
CREATE TABLE file_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    minio_version_id VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    checksum VARCHAR(128) NOT NULL,
    uploaded_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_file_versions_file_version UNIQUE (file_id, version_number)
);

CREATE INDEX idx_file_versions_file ON file_versions(file_id);
```

#### archived_files

```sql
CREATE TABLE archived_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_file_id UUID NOT NULL,
    original_folder_id UUID NOT NULL,   -- 復元先フォルダID（ファイルは必ずフォルダに所属）
    original_path VARCHAR(4096) NOT NULL,
    name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id),      -- 現在の所有者
    created_by UUID NOT NULL REFERENCES users(id),    -- 最初の作成者
    storage_key VARCHAR(1024) NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Note: owner_type は削除

CREATE INDEX idx_archived_files_owner ON archived_files(owner_id);
CREATE INDEX idx_archived_files_created_by ON archived_files(created_by);
CREATE INDEX idx_archived_files_expires ON archived_files(expires_at);
```

#### archived_file_versions

```sql
CREATE TABLE archived_file_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    archived_file_id UUID NOT NULL REFERENCES archived_files(id) ON DELETE CASCADE,
    original_version_id UUID NOT NULL,
    version_number INTEGER NOT NULL,
    minio_version_id VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    checksum VARCHAR(128) NOT NULL,
    uploaded_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_archived_file_versions_archived_file ON archived_file_versions(archived_file_id);
```

#### upload_sessions

```sql
CREATE TABLE upload_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id),      -- 所有者ID
    created_by UUID NOT NULL REFERENCES users(id),    -- 作成者ID（アップロード者）
    folder_id UUID NOT NULL REFERENCES folders(id),   -- アップロード先フォルダID（必須）
    file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    total_size BIGINT NOT NULL,
    storage_key VARCHAR(1024) NOT NULL,
    minio_upload_id VARCHAR(255),
    is_multipart BOOLEAN NOT NULL DEFAULT FALSE,
    total_parts INTEGER NOT NULL DEFAULT 1,
    uploaded_parts INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'aborted', 'expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Note: owner_type は削除
-- Note: folder_id は必須（NOT NULL）

CREATE INDEX idx_upload_sessions_file ON upload_sessions(file_id);
CREATE INDEX idx_upload_sessions_folder ON upload_sessions(folder_id);
CREATE INDEX idx_upload_sessions_storage_key ON upload_sessions(storage_key);
CREATE INDEX idx_upload_sessions_status ON upload_sessions(status);
CREATE INDEX idx_upload_sessions_expires ON upload_sessions(expires_at);
```

#### upload_parts

```sql
CREATE TABLE upload_parts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES upload_sessions(id) ON DELETE CASCADE,
    part_number INTEGER NOT NULL,
    size BIGINT NOT NULL,
    etag VARCHAR(255) NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_upload_parts_session_part UNIQUE (session_id, part_number)
);

CREATE INDEX idx_upload_parts_session ON upload_parts(session_id);
```

---

## 実装ノート

### 設計原則

**問題点（アンチパターン）:**
- UseCaseに手続き的なロジックが集中
- ドメインロジックがUseCase層にリーク
- エンティティが貧血モデル（データのみ、振る舞いなし）

**解決方針:**
- **リッチドメインモデル**: エンティティに振る舞いを持たせる
- **ドメインサービス**: 複数エンティティにまたがるビジネスロジックをカプセル化
- **リポジトリの責務拡大**: 永続化の詳細を隠蔽し、ドメイン操作を提供

---

### パッケージ構成

```
backend/internal/
├── domain/
│   ├── entity/
│   │   ├── file.go                    # File（リッチモデル）
│   │   ├── file_version.go            # FileVersion
│   │   ├── archived_file.go           # ArchivedFile
│   │   └── upload_session.go          # UploadSession（リッチモデル）
│   ├── valueobject/
│   │   ├── storage_key.go
│   │   ├── mime_type.go
│   │   └── file_name.go
│   ├── service/                       # ドメインサービス
│   │   └── file_archive_service.go    # アーカイブロジック
│   └── repository/
│       ├── file_repository.go         # ドメイン操作を含む
│       └── upload_session_repository.go
│
├── usecase/storage/
│   ├── command/
│   │   ├── initiate_upload.go         # シンプルなオーケストレーション
│   │   ├── complete_upload.go         # Webhook処理
│   │   ├── trash_file.go
│   │   └── restore_file.go
│   └── query/
│       ├── get_upload_status.go
│       └── get_download_url.go
│
└── infrastructure/
    ├── repository/
    │   └── file_repository.go         # ドメインリポジトリ実装
    └── storage/
        └── minio_client.go
```

---

### エンティティ設計（リッチドメインモデル）

#### File エンティティ

```go
// internal/domain/entity/file.go

type File struct {
    id             uuid.UUID
    folderID       uuid.UUID                   // 必須（ファイルは必ずフォルダに所属）
    ownerID        uuid.UUID                   // 現在の所有者（所有権譲渡で変更可能）
    createdBy      uuid.UUID                   // 最初の作成者（不変、履歴追跡用）
    name           valueobject.FileName
    mimeType       valueobject.MimeType
    size           int64
    storageKey     valueobject.StorageKey
    currentVersion int
    status         FileStatus
    createdAt      time.Time
    updatedAt      time.Time
}

// Note: owner_type は削除（ファイルは常にユーザーが作成者）
// Note: folder_id は必須（NOT NULL）- ファイルは必ずフォルダに所属
// Note: グループはPermissionGrantで関連付けてアクセス

// ファクトリメソッド - 不変条件を保証
// 新規作成時は owner_id = created_by = 作成者
func NewFile(
    folderID uuid.UUID,      // 必須
    creatorID uuid.UUID,     // 作成者ID（owner_id と created_by の両方に設定）
    name valueobject.FileName,
    mimeType valueobject.MimeType,
    size int64,
) (*File, error) {
    id := uuid.New()
    storageKey, err := valueobject.NewStorageKey(id)
    if err != nil {
        return nil, err
    }

    return &File{
        id:             id,
        folderID:       folderID,       // 必須
        ownerID:        creatorID,      // 新規作成時は作成者がオーナー
        createdBy:      creatorID,      // 作成者は不変
        name:           name,
        mimeType:       mimeType,
        size:           size,
        storageKey:     storageKey,
        currentVersion: 0,  // アップロード完了時に1になる
        status:         FileStatusUploading,
        createdAt:      time.Now(),
        updatedAt:      time.Now(),
    }, nil
}

// ステータス遷移メソッド - ビジネスルールをカプセル化
func (f *File) Activate(versionNumber int) error {
    if f.status != FileStatusUploading {
        return ErrInvalidStatusTransition
    }
    f.status = FileStatusActive
    f.currentVersion = versionNumber
    f.updatedAt = time.Now()
    return nil
}

func (f *File) MarkUploadFailed() error {
    if f.status != FileStatusUploading {
        return ErrInvalidStatusTransition
    }
    f.status = FileStatusUploadFailed
    f.updatedAt = time.Now()
    return nil
}

// ドメインロジック
func (f *File) CanDownload() bool {
    return f.status == FileStatusActive
}

func (f *File) CanArchive() bool {
    return f.status == FileStatusActive
}

func (f *File) Rename(newName valueobject.FileName) {
    f.name = newName
    f.updatedAt = time.Now()
}

func (f *File) MoveTo(folderID uuid.UUID) {
    f.folderID = folderID
    f.updatedAt = time.Now()
}

// 所有権譲渡
func (f *File) TransferOwnership(newOwnerID uuid.UUID) {
    f.ownerID = newOwnerID
    f.updatedAt = time.Now()
}

// Getters
func (f *File) FolderID() uuid.UUID   { return f.folderID }
func (f *File) OwnerID() uuid.UUID    { return f.ownerID }
func (f *File) CreatedBy() uuid.UUID  { return f.createdBy }

// ArchivedFile生成 - 変換ロジックをエンティティに
func (f *File) ToArchived(archivedBy uuid.UUID, originalPath string) *ArchivedFile {
    return &ArchivedFile{
        id:               uuid.New(),
        originalFileID:   f.id,
        originalFolderID: f.folderID,     // 必須
        originalPath:     originalPath,
        name:             f.name,
        mimeType:         f.mimeType,
        size:             f.size,
        ownerID:          f.ownerID,
        createdBy:        f.createdBy,    // 作成者情報を保持
        storageKey:       f.storageKey,
        archivedAt:       time.Now(),
        archivedBy:       archivedBy,
        expiresAt:        time.Now().AddDate(0, 0, TrashRetentionDays),
    }
}
```

#### UploadSession エンティティ

```go
// internal/domain/entity/upload_session.go

type UploadSession struct {
    id            uuid.UUID
    fileID        uuid.UUID
    storageKey    valueobject.StorageKey
    isMultipart   bool
    totalParts    int
    uploadedParts int
    status        UploadStatus
    expiresAt     time.Time
    // ...
}

// ファクトリメソッド
func NewUploadSession(file *File, totalSize int64) *UploadSession {
    isMultipart := totalSize >= MultipartThreshold
    totalParts := 1
    if isMultipart {
        totalParts = int(math.Ceil(float64(totalSize) / float64(MinPartSize)))
    }

    return &UploadSession{
        id:            uuid.New(),
        fileID:        file.ID(),
        storageKey:    file.StorageKey(),
        isMultipart:   isMultipart,
        totalParts:    totalParts,
        uploadedParts: 0,
        status:        UploadStatusPending,
        expiresAt:     time.Now().Add(UploadSessionTTLHours * time.Hour),
    }
}

// パーツアップロード記録
func (s *UploadSession) RecordPartUpload() error {
    if s.status == UploadStatusCompleted || s.status == UploadStatusAborted {
        return ErrSessionAlreadyFinished
    }
    s.uploadedParts++
    s.status = UploadStatusInProgress
    return nil
}

// 全パーツ完了チェック
func (s *UploadSession) IsAllPartsUploaded() bool {
    return s.uploadedParts >= s.totalParts
}

// 完了
func (s *UploadSession) Complete() error {
    if s.status == UploadStatusCompleted {
        return ErrSessionAlreadyFinished
    }
    s.status = UploadStatusCompleted
    return nil
}

// 中断
func (s *UploadSession) Abort() error {
    if s.status == UploadStatusCompleted {
        return ErrCannotAbortCompletedSession
    }
    s.status = UploadStatusAborted
    return nil
}

// 期限切れチェック
func (s *UploadSession) IsExpired() bool {
    return time.Now().After(s.expiresAt)
}
```

---

### ドメインサービス

複数エンティティにまたがる操作や、エンティティに持たせるには不自然なロジックをカプセル化。

```go
// internal/domain/service/file_archive_service.go

type FileArchiveService interface {
    // ファイルをアーカイブ（File + FileVersions → ArchivedFile + ArchivedFileVersions）
    Archive(file *entity.File, versions []*entity.FileVersion, archivedBy uuid.UUID, path string) (*entity.ArchivedFile, []*entity.ArchivedFileVersion)

    // アーカイブから復元
    Restore(archived *entity.ArchivedFile, archivedVersions []*entity.ArchivedFileVersion) (*entity.File, []*entity.FileVersion)
}

type fileArchiveService struct{}

func (s *fileArchiveService) Archive(
    file *entity.File,
    versions []*entity.FileVersion,
    archivedBy uuid.UUID,
    path string,
) (*entity.ArchivedFile, []*entity.ArchivedFileVersion) {
    // File → ArchivedFile 変換（エンティティメソッド使用）
    archivedFile := file.ToArchived(archivedBy, path)

    // FileVersion → ArchivedFileVersion 変換
    archivedVersions := make([]*entity.ArchivedFileVersion, len(versions))
    for i, v := range versions {
        archivedVersions[i] = v.ToArchived(archivedFile.ID())
    }

    return archivedFile, archivedVersions
}
```

---

### リポジトリ設計

リポジトリはCRUDだけでなく、ドメイン操作をカプセル化。

```go
// internal/domain/repository/file_repository.go

type FileRepository interface {
    // 基本操作
    Create(ctx context.Context, file *entity.File) error
    FindByID(ctx context.Context, id uuid.UUID) (*entity.File, error)
    Update(ctx context.Context, file *entity.File) error
    Delete(ctx context.Context, id uuid.UUID) error

    // ドメイン操作（トランザクション含む）
    ArchiveWithVersions(ctx context.Context, archivedFile *entity.ArchivedFile, archivedVersions []*entity.ArchivedFileVersion, originalFileID uuid.UUID) error
    RestoreFromArchive(ctx context.Context, file *entity.File, versions []*entity.FileVersion, archivedFileID uuid.UUID) error

    // クエリ
    FindByFolderID(ctx context.Context, folderID uuid.UUID) ([]*entity.File, error)
    ExistsByNameInFolder(ctx context.Context, name valueobject.FileName, folderID uuid.UUID) (bool, error)
}
```

---

### UseCase実装（シンプルなオーケストレーション）

UseCaseはドメインオブジェクトの組み合わせと永続化の調整のみを行う。

```go
// internal/usecase/file/command/trash_file.go

type TrashFileCommand struct {
    fileRepo       repository.FileRepository
    fileVersionRepo repository.FileVersionRepository
    archiveService service.FileArchiveService
    pathResolver   service.PathResolver  // パス解決サービス
}

type TrashFileInput struct {
    FileID uuid.UUID
    UserID uuid.UUID
}

func (c *TrashFileCommand) Execute(ctx context.Context, input TrashFileInput) error {
    // 1. ファイル取得
    file, err := c.fileRepo.FindByID(ctx, input.FileID)
    if err != nil {
        return err
    }

    // 2. ドメインバリデーション（エンティティメソッド）
    if !file.CanArchive() {
        return apperror.NewValidationError("file cannot be archived", nil)
    }

    // 3. バージョン取得
    versions, err := c.fileVersionRepo.FindByFileID(ctx, file.ID())
    if err != nil {
        return err
    }

    // 4. パス解決
    path := c.pathResolver.ResolvePath(ctx, file)

    // 5. ドメインサービスでアーカイブ変換
    archivedFile, archivedVersions := c.archiveService.Archive(file, versions, input.UserID, path)

    // 6. リポジトリでアトミックに永続化（トランザクションはリポジトリ内）
    return c.fileRepo.ArchiveWithVersions(ctx, archivedFile, archivedVersions, file.ID())
}
```

```go
// internal/usecase/file/command/complete_upload.go

type CompleteUploadCommand struct {
    sessionRepo repository.UploadSessionRepository
    fileRepo    repository.FileRepository
    versionRepo repository.FileVersionRepository
}

type CompleteUploadInput struct {
    StorageKey     string
    MinioVersionID string
    Size           int64
    Checksum       string
}

func (c *CompleteUploadCommand) Execute(ctx context.Context, input CompleteUploadInput) error {
    // 1. セッション取得
    session, err := c.sessionRepo.FindByStorageKey(ctx, input.StorageKey)
    if err != nil {
        return err
    }

    // 2. マルチパートの場合、パーツ記録
    if session.IsMultipart() {
        if err := session.RecordPartUpload(); err != nil {
            return err
        }

        // 全パーツ完了でなければ終了
        if !session.IsAllPartsUploaded() {
            return c.sessionRepo.Update(ctx, session)
        }
    }

    // 3. ファイル取得
    file, err := c.fileRepo.FindByID(ctx, session.FileID())
    if err != nil {
        return err
    }

    // 4. FileVersion作成
    version := entity.NewFileVersion(file.ID(), file.CurrentVersion()+1, input.MinioVersionID, input.Size, input.Checksum)

    // 5. File活性化（エンティティメソッド）
    if err := file.Activate(version.VersionNumber()); err != nil {
        return err
    }

    // 6. Session完了（エンティティメソッド）
    if err := session.Complete(); err != nil {
        return err
    }

    // 7. 永続化
    if err := c.versionRepo.Create(ctx, version); err != nil {
        return err
    }
    if err := c.fileRepo.Update(ctx, file); err != nil {
        return err
    }
    return c.sessionRepo.Update(ctx, session)
}
```

---

### 定数定義

```go
// internal/domain/entity/file.go

const (
    FileStatusUploading    FileStatus = "uploading"
    FileStatusActive       FileStatus = "active"
    FileStatusUploadFailed FileStatus = "upload_failed"

    TrashRetentionDays    = 30
    UploadSessionTTLHours = 24
    MultipartThreshold    = 5 * 1024 * 1024  // 5MB
    MinPartSize           = 5 * 1024 * 1024  // 5MB
    MaxPartSize           = 5 * 1024 * 1024 * 1024  // 5GB
)

// ドメインエラー
var (
    ErrInvalidStatusTransition    = errors.New("invalid status transition")
    ErrSessionAlreadyFinished     = errors.New("session already finished")
    ErrCannotAbortCompletedSession = errors.New("cannot abort completed session")
)
```

---

## テスト計画

### ユニットテスト

| テスト対象 | テストケース |
|-----------|-------------|
| StorageKey | 有効なUUID、無効な形式、最大長 |
| MimeType | 有効なMIMEタイプ、カテゴリ判定、無効な形式 |
| FileName | 有効なファイル名、禁止文字、最大長、拡張子取得 |
| File Entity | ステータス遷移、バリデーション |
| UploadSession | ステータス遷移、有効期限チェック |

### 統合テスト

| テストケース | 概要 |
|-------------|------|
| シングルパートアップロード | 5MB未満ファイルのアップロード完了フロー |
| マルチパートアップロード | 5MB以上ファイルのアップロード完了フロー |
| アップロードキャンセル | マルチパートアップロードの中断 |
| ゴミ箱移動・復元 | ファイルと全バージョンの移動・復元 |
| 完全削除 | MinIOオブジェクト含む完全削除 |
| 期限切れセッション | 自動キャンセル処理 |
| 期限切れゴミ箱 | 自動削除処理 |

### E2Eテスト

| シナリオ | 概要 |
|---------|------|
| ファイルアップロード→ダウンロード | アップロードしたファイルをダウンロードできる |
| ファイルバージョン管理 | 複数バージョンをアップロードし、特定バージョンをダウンロードできる |
| ゴミ箱操作 | ゴミ箱への移動、一覧表示、復元、完全削除 |

---

## 受け入れ基準

### 機能要件

- [ ] シングルパートアップロード（5MB未満）が動作する
- [ ] マルチパートアップロード（5MB以上）が動作する
- [ ] アップロード完了がWebhookで検知される
- [ ] クライアントがポーリングで完了を確認できる
- [ ] ファイルをダウンロードできる
- [ ] 特定バージョンをダウンロードできる
- [ ] ファイル名を変更できる
- [ ] ファイルを別フォルダへ移動できる（移動元にmove_out、移動先にmove_in権限が必要）
- [ ] ファイルをゴミ箱へ移動できる
- [ ] ゴミ箱からファイルを復元できる
- [ ] ファイルを完全に削除できる
- [ ] ゴミ箱を空にできる

### フォルダ制約

- [ ] ファイルは必ずフォルダに所属する（folder_id NOT NULL）
- [ ] ルートレベルのファイルは存在しない（必ず何らかのフォルダに所属）

### 所有権

- [ ] `owner_id` は現在の所有者を表す（所有権譲渡で変更可能）
- [ ] `created_by` は最初の作成者を表す（不変）
- [ ] 新規作成時は `owner_id = created_by = 作成者`
- [ ] `created_by` は所有権譲渡後も変更されない
- [ ] グループはPermissionGrantで関連付けてアクセス（owner_typeは廃止）

### 非機能要件

- [ ] MinIO Bucket Notificationが設定されている
- [ ] Webhook通知の冪等性が保証されている
- [ ] 期限切れゴミ箱の自動削除バッチが動作する
- [ ] 期限切れアップロードセッションの自動キャンセルバッチが動作する

---

## 関連ドキュメント

- [File Domain](../03-domains/file.md)
- [Folder Domain](../03-domains/folder.md)
- [Storage Folder Spec](./storage-folder.md)
