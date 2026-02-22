# File Upload - シングルパート & マルチパートアップロード

## Meta

| Item | Value |
|------|-------|
| Status | Draft |
| Priority | High |
| Tier | 2 (Storage) |
| Spec IDs | 2B.1 file-upload, 2B.2 file-upload-multipart |
| Domain Refs | `03-domains/file.md` |
| Depends On | `features/folder-management.md` |

---

## 1. User Stories

**Primary:**
> As a user, I want to upload files to my cloud storage so that I can access them from anywhere.

**Secondary:**
> As a user, I want to see upload progress and be able to cancel or retry uploads so that I have full control over the upload process.

### Context

アップロードは Presigned URL 方式を採用。クライアントが MinIO へ直接アップロードし、API サーバーはファイルデータを経由しない。アップロード完了は MinIO Bucket Notification (Webhook) でサーバーが検知し、クライアントはポーリングで完了を確認する。マルチパート閾値は 5MB。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-FL001 | storage_key は file_id から生成 | `03-domains/file.md` |
| R-FL002 | 同一フォルダ内で name は一意 | `03-domains/file.md` |
| R-FL005 | uploading → active 遷移はアップロード完了時のみ | `03-domains/file.md` |
| R-FL007 | folder_id は必須（ファイルは必ずフォルダに所属） | `03-domains/file.md` |
| R-FL008 | 新規作成時 owner_id = created_by = 作成者 | `03-domains/file.md` |
| R-US001 | セッション有効期限は作成から 24 時間 | `03-domains/file.md` |
| R-US002 | 期限切れセッションは自動キャンセル | `03-domains/file.md` |
| R-US004 | completed/aborted 後は状態変更不可 | `03-domains/file.md` |
| R-UP001 | part_number は session 内で一意 | `03-domains/file.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-UP001 | 5MB 未満: シングルパート、5MB 以上: マルチパート自動判定 |
| FS-UP002 | クライアントポーリング間隔 1 秒、タイムアウト 30 秒 |
| FS-UP003 | Webhook 通知の冪等性保証（重複処理防止） |
| FS-UP004 | MinIO Presigned URL の有効期限はセッション有効期限と同一 |

### State Transitions

```
UploadSession:
  pending → in_progress → completed
  pending → aborted
  in_progress → aborted
  pending/in_progress → expired (auto)

File:
  uploading → active (on upload complete)
  uploading → upload_failed (on failure)
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/files/upload/initiate` | Cookie(session_id) | アップロード開始 |
| GET | `/api/v1/files/upload/{session_id}/status` | Cookie(session_id) | ステータス確認 |
| POST | `/api/v1/files/upload/{session_id}/abort` | Cookie(session_id) | アップロードキャンセル |
| POST | `/internal/webhooks/minio` | Internal | MinIO Webhook 受信 |

### Request / Response Details

#### `POST /api/v1/files/upload/initiate` - アップロード開始

**Request Body:**
```json
{
  "folder_id": "uuid",
  "name": "report.pdf",
  "mime_type": "application/pdf",
  "size": 10485760
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| folder_id | uuid | Yes | valid UUID | アップロード先フォルダ |
| name | string | Yes | 1-255 bytes, no forbidden chars | ファイル名 |
| mime_type | string | Yes | type/subtype format | MIME タイプ |
| size | int64 | Yes | >= 0 | ファイルサイズ（バイト） |

**Success Response (201):**
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

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | 無効なファイル名 / MIME タイプ | `VALIDATION_ERROR` |
| 403 | フォルダへの書き込み権限なし | `FORBIDDEN` |
| 404 | フォルダが存在しない | `NOT_FOUND` |
| 409 | 同一フォルダ内に同名ファイル存在 | `CONFLICT` |

#### `GET /api/v1/files/upload/{session_id}/status` - ステータス確認

**Success Response (200):**
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

| Status | Description |
|--------|-------------|
| pending | 初期化済み、アップロード未開始 |
| in_progress | アップロード中（マルチパート） |
| completed | 完了 |
| aborted | 中断 |
| expired | 期限切れ |

#### `POST /api/v1/files/upload/{session_id}/abort` - キャンセル

**Success Response (204 No Content)**

| Code | Condition | Error Code |
|------|-----------|------------|
| 404 | セッションが存在しない | `NOT_FOUND` |
| 409 | 既に完了/キャンセル済み | `CONFLICT` |

#### `POST /internal/webhooks/minio` - MinIO Webhook

**Request (MinIO S3 Event):**
```json
{
  "Records": [{
    "eventName": "s3:ObjectCreated:Put",
    "s3": {
      "bucket": { "name": "files" },
      "object": { "key": "uuid-file-id", "size": 10485760, "eTag": "..." }
    }
  }]
}
```

**Response (200 OK)** - 内部エンドポイント

---

## 4. Frontend UI

### Layout / Wireframe

```
Drag & Drop zone (full content area):
+------------------------------------------------------------------+
|  +------------------------------------------------------------+  |
|  |          [Upload icon]                                      |  |
|  |          Drop files to upload                               |  |
|  +------------------------------------------------------------+  |
|  (dashed border, semi-transparent overlay)                       |
+------------------------------------------------------------------+

Upload progress panel (bottom-right):
+-------------------------------------+
| Uploading 3 files          [Clear]  |
+---------+---------------------------+
| [done]  | photo1.jpg      Complete  |
| [spin]  | document.pdf       75%    |
|         | ================----      |
| [fail]  | video.mp4       Failed    |
|         | Upload failed - Retry     |
| [wait]  | data.csv        Pending   |
+-------------------------------------+
```

### Components

| Component | Type | Description |
|-----------|------|-------------|
| DropZone | UI | ドラッグ&ドロップ領域（オーバーレイ表示） |
| UploadButton | UI | ツールバーのアップロードボタン（ファイル選択ダイアログ） |
| UploadProgressPanel | UI | 画面右下のアップロード進捗パネル |
| UploadProgressItem | UI | 個別ファイルの進捗表示（完了/進行中/失敗/待機） |

### State Management

| State | Store | Type | Description |
|-------|-------|------|-------------|
| Upload queue | Zustand | Global | アップロードキュー（ページ遷移しても維持） |
| Upload progress | Zustand | Global | 各ファイルの進捗 (%) |
| Upload status polling | TanStack Query | Server | session status ポーリング |
| Drag state | useState | Local | ドラッグ中フラグ |

### User Interactions

1. ユーザーがファイルをドラッグ → DropZone オーバーレイ表示
2. ドロップ → InitiateUpload API → Presigned URL 取得
3. クライアントが MinIO へ直接 PUT（進捗表示）
4. ポーリング（1秒間隔）で完了確認 → 完了表示、一覧リフレッシュ
5. 失敗時 → エラー表示 + Retry ボタン
6. Cancel ボタン → abort API → キャンセル表示

---

## 5. Integration Flow

### シングルパートアップロード

```
Client          Frontend        API             MinIO           DB
  |                |              |                |              |
  |-- drop file -->|              |                |              |
  |                |-- POST ----->|                |              |
  |                |  initiate    |-- presigned -->|              |
  |                |              |<-- url --------|              |
  |                |              |-- insert ----->|              |>
  |                |              |  file+session  |              |
  |                |<-- 201 ------|                |              |
  |                |              |                |              |
  |                |-- PUT ------>|                |              |
  |                |  (direct)    |  MinIO storage |              |
  |                |              |                |              |
  |                |              |<-- webhook ----|              |
  |                |              |  ObjectCreated |              |
  |                |              |-- update ----->|              |>
  |                |              |  file=active   |              |
  |                |              |                |              |
  |                |-- GET ------>|                |              |
  |                |  poll status |                |              |
  |                |<-- completed-|                |              |
  |<-- refresh ----|              |                |              |
```

### Error Handling Flow

- Presigned URL 期限切れ → フロントエンドで検知 → 再度 initiate
- MinIO PUT 失敗 → フロントエンドで検知 → リトライまたは abort
- Webhook 未到着 → ポーリングタイムアウト（30秒） → エラー表示
- 重複 Webhook → 冪等性チェック（session status が completed なら無視）

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: 5MB 未満のファイルをシングルパートでアップロードできる
- [ ] AC-02: 5MB 以上のファイルをマルチパートでアップロードできる
- [ ] AC-03: アップロード完了が Webhook で検知される
- [ ] AC-04: クライアントがポーリングで完了を確認できる
- [ ] AC-05: 複数ファイルを同時にアップロードできる
- [ ] AC-06: ドラッグ&ドロップでアップロードできる
- [ ] AC-07: ファイル選択ダイアログからアップロードできる

### Validation Errors
- [ ] AC-10: 無効なファイル名でエラー
- [ ] AC-11: 同名ファイル存在時に 409 エラー

### Authorization
- [ ] AC-20: 書き込み権限なしのフォルダへアップロードで 403 エラー

### Edge Cases
- [ ] AC-30: アップロード中のキャンセルが正常に処理される
- [ ] AC-31: 期限切れセッションが自動キャンセルされる
- [ ] AC-32: Webhook の重複通知が冪等に処理される
- [ ] AC-33: ページ遷移してもアップロードが継続する

---

## 7. Test Plan

### Backend Unit Tests

| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| NewUploadSession multipart | entity.NewUploadSession | size >= 5MB で is_multipart=true |
| Session complete | entity.UploadSession | Complete() で status=completed |
| Session abort | entity.UploadSession | Abort() で status=aborted |
| File activate | entity.File | Activate() で status=active |
| FileName validation | valueobject.FileName | 禁止文字、空文字でエラー |
| MimeType category | valueobject.MimeType | image/png → category=image |

### Backend Integration Tests

| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| Initiate single upload | POST /upload/initiate | folder, size < 5MB | 201, is_multipart=false |
| Initiate multipart | POST /upload/initiate | folder, size >= 5MB | 201, is_multipart=true, urls |
| Upload status poll | GET /upload/:id/status | completed session | 200, status=completed |
| Abort upload | POST /upload/:id/abort | pending session | 204 |
| Webhook complete | POST /internal/webhooks/minio | upload session | file.status=active |

### Frontend Tests

| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| Drop zone display | DropZone | Unit | ドラッグ中にオーバーレイ表示 |
| Upload progress | UploadProgressPanel | Unit | 進捗表示、ステータス表示 |
| File select dialog | UploadButton | Integration | ファイル選択→アップロード開始 |
| Cancel upload | UploadProgressItem | Integration | キャンセル→abort API 呼出 |

### E2E Tests (future)

| Test | Flow | Assertions |
|------|------|------------|
| Upload and download | upload → poll → download | ファイル内容一致 |
| Multipart upload | large file upload → poll | 完了、ファイル一覧に表示 |

---

## 8. Implementation Notes

### Changed Files (Backend)

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/file.go` | File entity, status transitions |
| Domain | `internal/domain/entity/upload_session.go` | UploadSession entity |
| Domain | `internal/domain/valueobject/file_name.go` | FileName value object |
| Domain | `internal/domain/valueobject/mime_type.go` | MimeType value object |
| Domain | `internal/domain/valueobject/storage_key.go` | StorageKey value object |
| Domain | `internal/domain/repository/upload_session_repository.go` | UploadSessionRepository IF |
| UseCase | `internal/usecase/storage/command/initiate_upload.go` | InitiateUploadCommand |
| UseCase | `internal/usecase/storage/command/complete_upload.go` | CompleteUploadCommand (webhook) |
| UseCase | `internal/usecase/storage/query/get_upload_status.go` | GetUploadStatusQuery |
| Interface | `internal/interface/handler/upload_handler.go` | Upload endpoints |
| Interface | `internal/interface/handler/webhook_handler.go` | MinIO webhook handler |
| Infra | `internal/infrastructure/storage/minio_client.go` | Presigned URL generation |
| Infra | `internal/infrastructure/repository/upload_session_repository.go` | Repository impl |

### Changed Files (Frontend)

| Category | File | Change |
|----------|------|--------|
| Component | `src/features/files/components/upload-area.tsx` | ドラッグ&ドロップ + アップロード UI |
| Component | `src/features/files/components/upload-progress-panel.tsx` | 進捗パネル |
| Component | `src/features/files/components/upload-progress-item.tsx` | 個別進捗 |
| Store | `src/stores/upload-store.ts` | アップロードキュー管理 |
| Query | `src/features/files/api/queries.ts` | uploadStatus ポーリング |
| Mutation | `src/features/files/api/mutations.ts` | initiateUpload, abortUpload |

### Considerations

- **Performance**: クライアント→MinIO 直接アップロードにより API サーバーの帯域を消費しない
- **Security**: Presigned URL は有効期限付き、session_id cookie で認証
- **Idempotency**: Webhook の冪等性は storage_key + session status で保証
