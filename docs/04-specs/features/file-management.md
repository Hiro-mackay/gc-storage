# File Management - ダウンロード、リネーム、移動

## Meta

| Item | Value |
|------|-------|
| Status | Draft |
| Priority | High |
| Tier | 2 (Storage) |
| Spec IDs | 2B.3 file-download, 2B.4 file-list, 2B.6 file-rename, 2B.7 file-move |
| Domain Refs | `03-domains/file.md` |
| Depends On | `features/file-upload.md`, `features/folder-management.md` |

---

## 1. User Stories

**Primary:**
> As a user, I want to download, rename, and move files so that I can access and organize my stored files.

**Secondary:**
> As a user, I want to download specific versions of a file so that I can access previous versions when needed.

### Context

ファイルのダウンロードは MinIO の Presigned GET URL を使用。リネーム・移動は API 経由でメタデータのみ変更し、MinIO 上のオブジェクトは移動しない（storage_key は file_id ベースで不変）。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-FL001 | storage_key は file_id から生成（不変） | `03-domains/file.md` |
| R-FL002 | 同一フォルダ内で name は一意 | `03-domains/file.md` |
| R-FL005 | uploading → active 遷移完了後のみダウンロード可 | `03-domains/file.md` |
| R-FL007 | folder_id は必須（ファイルは必ずフォルダに所属） | `03-domains/file.md` |
| R-FV004 | 特定バージョン取得は storage_key + minio_version_id | `03-domains/file.md` |
| R-FN001 | ファイル名 1-255 バイト（UTF-8） | `03-domains/file.md` |
| R-FN002 | 禁止文字を含まない | `03-domains/file.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-FM001 | ダウンロード URL は Presigned GET URL（有効期限付き） |
| FS-FM002 | uploading 状態のファイルはダウンロード不可（409 エラー） |
| FS-FM003 | 移動時、移動元に move_out、移動先に move_in 権限が必要 |
| FS-FM004 | リネーム・移動は MinIO オブジェクトに影響しない（DB のみ変更） |

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/files/{file_id}/download` | Cookie(session_id) | ダウンロード URL 取得 |
| PUT | `/api/v1/files/{file_id}/name` | Cookie(session_id) | ファイル名変更 |
| PUT | `/api/v1/files/{file_id}/folder` | Cookie(session_id) | ファイル移動 |
| GET | `/api/v1/files/{file_id}/versions` | Cookie(session_id) | バージョン一覧 |

### Request / Response Details

#### `GET /api/v1/files/{file_id}/download` - ダウンロード URL 取得

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| version | int | バージョン番号（省略時は最新） |

**Success Response (200):**
```json
{
  "download_url": "https://minio.example.com/bucket/...",
  "expires_at": "2026-01-20T01:00:00Z",
  "file_name": "report.pdf",
  "mime_type": "application/pdf",
  "size": 10485760
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 403 | ファイルへのアクセス権限なし | `FORBIDDEN` |
| 404 | ファイル/バージョンが存在しない | `NOT_FOUND` |
| 409 | ファイルがアップロード中 | `CONFLICT` |

#### `PUT /api/v1/files/{file_id}/name` - ファイル名変更

**Request Body:**
```json
{ "name": "new_report.pdf" }
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| name | string | Yes | 1-255 bytes, no forbidden chars | 新しいファイル名 |

**Success Response (200):**
```json
{ "id": "uuid", "name": "new_report.pdf", "updated_at": "..." }
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | 無効なファイル名 | `VALIDATION_ERROR` |
| 403 | 書き込み権限なし | `FORBIDDEN` |
| 404 | ファイルが存在しない | `NOT_FOUND` |
| 409 | 同名ファイル存在 | `CONFLICT` |

#### `PUT /api/v1/files/{file_id}/folder` - ファイル移動

**Request Body:**
```json
{ "folder_id": "uuid" }
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| folder_id | uuid | Yes | valid UUID | 移動先フォルダ ID |

**Success Response (200):**
```json
{ "id": "uuid", "folder_id": "uuid", "updated_at": "..." }
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 403 | ファイル/フォルダ権限なし | `FORBIDDEN` |
| 404 | ファイル/フォルダ存在しない | `NOT_FOUND` |
| 409 | 移動先に同名ファイル存在 | `CONFLICT` |

#### `GET /api/v1/files/{file_id}/versions` - バージョン一覧

**Success Response (200):**
```json
{
  "versions": [
    { "version_number": 2, "size": 10485760, "checksum": "sha256:...", "uploaded_by": "uuid", "created_at": "..." },
    { "version_number": 1, "size": 8388608, "checksum": "sha256:...", "uploaded_by": "uuid", "created_at": "..." }
  ]
}
```

---

## 4. Frontend UI

### Layout / Wireframe

```
Context menu (right-click on file):
+---------------------+
| [eye]  Preview      |
| [down] Download     |
|---------------------|
| [pen]  Rename       |
| [move] Move to...   |
| [copy] Copy         |
|---------------------|
| [link] Share        |
| [info] Details      |
| [hist] Versions     |
|---------------------|
| [bin]  Move to trash|
+---------------------+

Rename dialog:
+---------------------------------------+
| Rename                            [x] |
|---------------------------------------|
| File name                             |
| +-----------------------------------+ |
| | report.pdf                        | |
| +-----------------------------------+ |
|                                       |
|              [Cancel]  [Save]         |
+---------------------------------------+

Move dialog:
+---------------------------------------+
| Move to...                        [x] |
|---------------------------------------|
| +-----------------------------------+ |
| | > My Files                        | |
| |   > Documents                     | |
| |   > Images (selected)             | |
| |   > Projects                      | |
| | > Shared Project                  | |
| +-----------------------------------+ |
|              [Cancel]  [Move]         |
+---------------------------------------+
```

### Components

| Component | Type | Description |
|-----------|------|-------------|
| ContextMenu | UI | 右クリックコンテキストメニュー（ファイル/フォルダ別） |
| RenameDialog | Modal | ファイル名変更ダイアログ |
| MoveDialog | Modal | 移動先フォルダツリー選択ダイアログ |
| FilePreview | Modal | ファイルプレビュー（画像、PDF、テキスト） |
| VersionHistoryPanel | Panel | バージョン履歴表示 |

### State Management

| State | Store | Type | Description |
|-------|-------|------|-------------|
| Context menu | useState | Local | メニュー位置、対象アイテム |
| File versions | TanStack Query | Server | バージョン一覧キャッシュ |
| Rename form | useState | Local | ダイアログ入力状態 |
| Move target | useState | Local | 移動先フォルダ選択状態 |

### User Interactions

1. ファイル右クリック → コンテキストメニュー表示
2. Download クリック → GET /files/:id/download → Presigned URL で自動ダウンロード開始
3. Rename クリック → RenameDialog → 名前入力 → PUT /files/:id/name → 一覧更新
4. Move to... クリック → MoveDialog → フォルダ選択 → PUT /files/:id/folder → 一覧更新
5. Versions クリック → VersionHistoryPanel → 特定バージョンダウンロード

---

## 5. Integration Flow

### ファイルダウンロード

```
Client          Frontend        API             MinIO
  |                |              |                |
  |-- click DL -->|              |                |
  |                |-- GET ------>|                |
  |                |  /download   |-- presigned -->|
  |                |              |<-- GET URL ----|
  |                |<-- 200 ------|                |
  |                |              |                |
  |                |-- GET ------>|                |
  |                |  presigned   |  (direct DL)   |
  |<-- file data --|              |                |
```

### ファイル移動

```
Client          Frontend        API             DB
  |                |              |                |
  |-- move to.. ->|              |                |
  |-- select dst ->|              |                |
  |-- confirm ---->|              |                |
  |                |-- PUT ------>|                |
  |                |  /folder     |-- check perm ->|
  |                |              |-- check dup -->|
  |                |              |-- update ----->|
  |                |              |<-- ok ---------|
  |                |<-- 200 ------|                |
  |<-- refresh ----|              |                |
```

### Error Handling Flow

- 409 Conflict（同名ファイル）→ ダイアログ内にインラインエラー
- 409 Conflict（アップロード中）→ toast "ファイルはまだアップロード中です"
- 403 Forbidden → toast "権限がありません"
- 404 Not Found → toast "ファイルが見つかりません" + 一覧リフレッシュ

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: active 状態のファイルをダウンロードできる
- [ ] AC-02: 特定バージョンをダウンロードできる
- [ ] AC-03: ファイル名を変更できる
- [ ] AC-04: ファイルを別フォルダへ移動できる
- [ ] AC-05: バージョン一覧を表示できる

### Validation Errors
- [ ] AC-10: 空文字のファイル名でバリデーションエラー
- [ ] AC-11: 禁止文字を含むファイル名でバリデーションエラー
- [ ] AC-12: 同一フォルダ内に同名ファイルが存在する場合に 409 エラー

### Authorization
- [ ] AC-20: アクセス権限なしのファイルダウンロードで 403 エラー
- [ ] AC-21: 書き込み権限なしのリネームで 403 エラー
- [ ] AC-22: 移動元 move_out / 移動先 move_in 権限なしで 403 エラー

### Edge Cases
- [ ] AC-30: uploading 状態のファイルダウンロードで 409 エラー
- [ ] AC-31: 存在しないバージョンのダウンロードで 404 エラー
- [ ] AC-32: 移動先フォルダが存在しない場合に 404 エラー
- [ ] AC-33: リネーム時に拡張子変更の警告（フロントエンド）

---

## 7. Test Plan

### Backend Unit Tests

| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| CanDownload active | entity.File | status=active で true |
| CanDownload uploading | entity.File | status=uploading で false |
| Rename file | entity.File | name 変更、updated_at 更新 |
| MoveTo folder | entity.File | folder_id 変更、updated_at 更新 |

### Backend Integration Tests

| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| Download active file | GET /files/:id/download | active file | 200, presigned URL |
| Download uploading | GET /files/:id/download | uploading file | 409 |
| Download version | GET /files/:id/download?version=1 | file with versions | 200, correct version |
| Rename file | PUT /files/:id/name | existing file | 200, name updated |
| Rename duplicate | PUT /files/:id/name | same-name file exists | 409 |
| Move file | PUT /files/:id/folder | target folder | 200, folder_id updated |
| Move duplicate | PUT /files/:id/folder | same-name in target | 409 |
| List versions | GET /files/:id/versions | file with 2 versions | 200, 2 versions |

### Frontend Tests

| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| Context menu display | ContextMenu | Unit | ファイル用メニュー項目表示 |
| Rename dialog | RenameDialog | Integration | 入力→送信→一覧更新 |
| Move dialog | MoveDialog | Integration | フォルダ選択→移動→一覧更新 |
| Download trigger | ContextMenu | Integration | クリック→presigned URL→ダウンロード開始 |

### E2E Tests (future)

| Test | Flow | Assertions |
|------|------|------------|
| Rename and verify | upload → rename → verify name | 名前変更が反映 |
| Move and verify | upload → move → check both folders | 移動元から消え、移動先に表示 |

---

## 8. Implementation Notes

### Changed Files (Backend)

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/file.go` | Rename, MoveTo methods |
| Domain | `internal/domain/repository/file_repository.go` | FileRepository IF |
| UseCase | `internal/usecase/storage/command/rename_file.go` | RenameFileCommand |
| UseCase | `internal/usecase/storage/command/move_file.go` | MoveFileCommand |
| UseCase | `internal/usecase/storage/query/get_download_url.go` | GetDownloadUrlQuery |
| UseCase | `internal/usecase/storage/query/list_file_versions.go` | ListFileVersionsQuery |
| Interface | `internal/interface/handler/file_handler.go` | File endpoints |
| Infra | `internal/infrastructure/storage/minio_client.go` | Presigned GET URL |
| Infra | `internal/infrastructure/repository/file_repository.go` | Repository impl |

### Changed Files (Frontend)

| Category | File | Change |
|----------|------|--------|
| Component | `src/features/files/components/file-context-menu.tsx` | コンテキストメニュー |
| Component | `src/features/files/components/rename-dialog.tsx` | リネームダイアログ（共有） |
| Component | `src/features/files/components/move-dialog.tsx` | 移動ダイアログ |
| Component | `src/features/files/components/file-preview.tsx` | プレビューモーダル |
| Query | `src/features/files/api/queries.ts` | downloadUrl, fileVersions |
| Mutation | `src/features/files/api/mutations.ts` | renameFile, moveFile |

### Considerations

- **Performance**: Presigned URL でファイルデータは API サーバーを経由しない
- **Security**: Presigned URL は有効期限付き。権限チェックは URL 発行時に実施
- **Backward Compatibility**: storage_key は不変のため、リネーム・移動で MinIO 操作不要
