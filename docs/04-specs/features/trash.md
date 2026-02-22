# Trash - ゴミ箱ライフサイクル

## Meta

| Item | Value |
|------|-------|
| Status | Draft |
| Priority | High |
| Tier | 2 (Storage) |
| Spec IDs | 2A.5-2A.7, 2B.9-2B.11, 2C.1-2C.3 |
| Domain Refs | `03-domains/file.md`, `03-domains/folder.md` |
| Depends On | `features/file-upload.md`, `features/folder-management.md` |

---

## 1. User Stories

**Primary:**
> As a user, I want to move files to trash and restore them so that I can recover accidentally deleted files within 30 days.

**Secondary:**
> As a user, I want to permanently delete files or empty the trash so that I can free up storage and remove sensitive data completely.

### Context

ゴミ箱はアーカイブテーブル方式を採用。論理削除（trashed_at フラグ）ではなく、files → archived_files への物理的なデータ移動で実現する。フォルダにはゴミ箱がなく直接削除され、配下のファイルのみが archived_files へ移動する。30 日後に自動削除バッチが実行される。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-AF001 | expires_at は archived_at から 30 日後 | `03-domains/file.md` |
| R-AF002 | expires_at 経過後はバッチで自動削除 | `03-domains/file.md` |
| R-AF003 | 復元時、元フォルダが存在しない場合は Personal Folder に復元 | `03-domains/file.md` |
| R-AF004 | 完全削除時は MinIO の全バージョンも削除 | `03-domains/file.md` |
| R-AFV001 | ArchivedFile 削除時、関連 ArchivedFileVersion も削除 | `03-domains/file.md` |
| R-AFV002 | 復元時、全バージョンを FileVersion として復元 | `03-domains/file.md` |
| R-FD005 | フォルダ削除時、配下ファイルは ArchivedFile へ移動 | `03-domains/folder.md` |
| R-FD006 | フォルダ削除時、サブフォルダも再帰的に削除 | `03-domains/folder.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-TR001 | フォルダにはゴミ箱がない（直接削除、ファイルのみアーカイブ） |
| FS-TR002 | ゴミ箱移動時、MinIO オブジェクトは削除しない |
| FS-TR003 | 復元時、全バージョンが復元される |
| FS-TR004 | 完全削除時、MinIO 全バージョン + DB レコード削除 |
| FS-TR005 | ゴミ箱を空にする操作は非同期（202 Accepted） |

### State Transitions

```
File (active)
    |
    | POST /files/:id/trash
    v
ArchivedFile (archived)
    |                    |
    | POST /restore      | DELETE (permanent)
    v                    v
File (active)        Deleted (物理削除 + MinIO 削除)
                         |
                         | 30 日自動 (batch)
                         v
                     Deleted (同上)
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/files/{file_id}/trash` | Cookie(session_id) | ゴミ箱へ移動 |
| POST | `/api/v1/trash/files/{archived_file_id}/restore` | Cookie(session_id) | 復元 |
| DELETE | `/api/v1/trash/files/{archived_file_id}` | Cookie(session_id) | 完全削除 |
| GET | `/api/v1/trash` | Cookie(session_id) | ゴミ箱一覧 |
| DELETE | `/api/v1/trash` | Cookie(session_id) | ゴミ箱を空にする |

### Request / Response Details

#### `POST /api/v1/files/{file_id}/trash` - ゴミ箱へ移動

**Success Response (200):**
```json
{
  "archived_file_id": "uuid",
  "expires_at": "2026-02-19T00:00:00Z"
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 403 | ファイル削除権限なし | `FORBIDDEN` |
| 404 | ファイルが存在しない | `NOT_FOUND` |

**Internal flow:**
1. files → archived_files へデータコピー
2. file_versions → archived_file_versions へ全バージョンコピー
3. file_versions 削除（CASCADE）
4. files から削除
5. MinIO オブジェクトは削除しない

#### `POST /api/v1/trash/files/{archived_file_id}/restore` - 復元

**Success Response (200):**
```json
{
  "file_id": "uuid",
  "folder_id": "uuid | null",
  "name": "report.pdf"
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 403 | 復元権限なし | `FORBIDDEN` |
| 404 | アーカイブファイルが存在しない | `NOT_FOUND` |
| 409 | 復元先に同名ファイル存在 | `CONFLICT` |

**Internal flow:**
1. archived_files → files へデータコピー
2. archived_file_versions → file_versions へ全バージョンコピー
3. archived_file_versions 削除（CASCADE）
4. archived_files から削除
5. 元フォルダが存在しない場合は Personal Folder に復元

#### `DELETE /api/v1/trash/files/{archived_file_id}` - 完全削除

**Success Response (204 No Content)**

| Code | Condition | Error Code |
|------|-----------|------------|
| 403 | 削除権限なし | `FORBIDDEN` |
| 404 | アーカイブファイルが存在しない | `NOT_FOUND` |

**Internal flow:**
1. MinIO から全バージョン削除
2. archived_file_versions 削除（CASCADE）
3. archived_files から削除

#### `GET /api/v1/trash` - ゴミ箱一覧

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| limit | int | 取得件数（default: 50） |
| cursor | string | ページネーションカーソル |

**Success Response (200):**
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

#### `DELETE /api/v1/trash` - ゴミ箱を空にする

**Success Response (202 Accepted):**
```json
{
  "message": "Trash emptying started",
  "deleted_count": 15
}
```

---

## 4. Frontend UI

### Layout / Wireframe

```
Trash page:
+------------------------------------------------------------------+
| Trash                                            [Empty trash]   |
+------------------------------------------------------------------+
| Items in trash will be automatically deleted after 30 days.      |
+------------------------------------------------------------------+
| [x] | Name              | Deleted         | Size                |
+------------------------------------------------------------------+
| [ ] | old_report.pdf    | Jan 25, 2026    | 456 KB              |
| [ ] | backup_data.csv   | Jan 20, 2026    | 1.2 MB              |
+------------------------------------------------------------------+

Selection toolbar:
+------------------------------------------------------------------+
| 2 items selected         [Restore]  [Delete permanently]         |
+------------------------------------------------------------------+

Delete confirmation dialog:
+---------------------------------------+
| Delete permanently?               [x] |
|---------------------------------------|
| This action cannot be undone.         |
| "old_report.pdf" will be permanently  |
| deleted and cannot be recovered.      |
|                                       |
|              [Cancel]  [Delete]       |
+---------------------------------------+
```

### Components

| Component | Type | Description |
|-----------|------|-------------|
| TrashPage | Page | ゴミ箱画面（/trash） |
| TrashList | UI | アーカイブファイル一覧（テーブル表示） |
| TrashToolbar | UI | 選択時アクション（復元/完全削除） |
| EmptyTrashButton | UI | ゴミ箱を空にするボタン（確認付き） |
| RestoreConfirmDialog | Modal | 復元確認ダイアログ |
| PermanentDeleteDialog | Modal | 完全削除確認ダイアログ |
| EmptyTrashDialog | Modal | ゴミ箱空にする確認ダイアログ |

### State Management

| State | Store | Type | Description |
|-------|-------|------|-------------|
| Trash items | TanStack Query | Server | ゴミ箱一覧（cursor pagination） |
| Selected items | useState | Local | 選択中アイテム |
| Empty in progress | useState | Local | 空にする処理中フラグ |

### User Interactions

1. サイドバーの "Trash" クリック → /trash へ遷移
2. アイテム選択 → ツールバーに Restore / Delete permanently 表示
3. Restore クリック → POST /restore → 復元 → 一覧リフレッシュ + toast
4. Delete permanently クリック → 確認ダイアログ → DELETE → 完全削除 + toast
5. Empty trash クリック → 確認ダイアログ → DELETE /trash → 全件削除 + toast
6. ファイルブラウザで "Move to trash" → POST /files/:id/trash → 一覧更新

---

## 5. Integration Flow

### ゴミ箱移動

```
Client          Frontend        API             DB              MinIO
  |                |              |                |              |
  |-- trash ------>|              |                |              |
  |                |-- POST ----->|                |              |
  |                |  /trash      |-- copy to ---->|              |
  |                |              |  archived_files|              |
  |                |              |-- copy ver --->|              |
  |                |              |  archived_vers |              |
  |                |              |-- delete ----->|              |
  |                |              |  files record  |              |
  |                |              |<-- ok ---------|              |
  |                |<-- 200 ------|                |  (no change) |
  |<-- refresh ----|              |                |              |
```

### 完全削除

```
Client          Frontend        API             DB              MinIO
  |                |              |                |              |
  |-- delete ----->|              |                |              |
  |                |-- DELETE --->|                |              |
  |                |              |-- delete ----->|              |
  |                |              |  all versions  |  (MinIO del) |
  |                |              |-- delete ----->|              |
  |                |              |  archived rec  |              |
  |                |              |<-- ok ---------|              |
  |                |<-- 204 ------|                |              |
  |<-- refresh ----|              |                |              |
```

### Error Handling Flow

- 復元先に同名ファイル → 409 エラー → toast "同名のファイルが存在します"
- 元フォルダが存在しない → Personal Folder に自動復元 → toast で通知
- MinIO 削除失敗 → ログ記録、リトライキュー（バッチ処理で再試行）
- ゴミ箱空にする → 非同期処理のため即座に 202 → 完了は後続バッチ

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: ファイルをゴミ箱へ移動できる
- [ ] AC-02: ゴミ箱からファイルを復元できる（元フォルダに復元）
- [ ] AC-03: ファイルを完全削除できる（DB + MinIO）
- [ ] AC-04: ゴミ箱一覧を表示できる（削除日、有効期限表示）
- [ ] AC-05: ゴミ箱を空にできる
- [ ] AC-06: フォルダ削除時に配下ファイルがゴミ箱に移動する

### Validation Errors
- [ ] AC-10: 復元先に同名ファイル存在時に 409 エラー

### Authorization
- [ ] AC-20: 削除権限なしのゴミ箱移動で 403 エラー
- [ ] AC-21: 復元権限なしの復元で 403 エラー

### Edge Cases
- [ ] AC-30: 元フォルダ削除済みの場合、Personal Folder に復元
- [ ] AC-31: 全バージョンが復元される
- [ ] AC-32: 完全削除時に MinIO の全バージョンが削除される
- [ ] AC-33: 30 日経過後にバッチで自動削除される
- [ ] AC-34: フォルダ削除→ファイル復元→元フォルダなし→Personal Folder 復元

---

## 7. Test Plan

### Backend Unit Tests

| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| File.CanArchive active | entity.File | status=active で true |
| File.CanArchive uploading | entity.File | status=uploading で false |
| File.ToArchived | entity.File | ArchivedFile 正しく生成、expires_at = +30 days |
| Archive service | FileArchiveService | File+Versions → Archived+ArchivedVersions |
| Restore service | FileArchiveService | Archived → File+Versions |

### Backend Integration Tests

| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| Trash file | POST /files/:id/trash | active file | 200, archived_files に移動 |
| Restore file | POST /trash/files/:id/restore | archived file | 200, files に復元 |
| Restore missing folder | POST /trash/files/:id/restore | folder deleted | 200, personal folder |
| Permanent delete | DELETE /trash/files/:id | archived file | 204, MinIO+DB 削除 |
| List trash | GET /trash | archived files | 200, items returned |
| Empty trash | DELETE /trash | archived files | 202, all deleted |
| Folder delete archives | DELETE /folders/:id | folder with files | files in archived |

### Frontend Tests

| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| Trash list | TrashList | Unit | アイテム一覧表示、削除日・期限表示 |
| Restore action | TrashToolbar | Integration | 選択→復元→一覧更新 |
| Permanent delete | PermanentDeleteDialog | Integration | 確認→削除→一覧更新 |
| Empty trash | EmptyTrashDialog | Integration | 確認→全削除→空表示 |
| Auto-delete notice | TrashPage | Unit | 30 日自動削除通知表示 |

### E2E Tests (future)

| Test | Flow | Assertions |
|------|------|------------|
| Full trash lifecycle | upload → trash → restore → verify | ファイル復元成功 |
| Permanent delete | upload → trash → permanent delete → verify gone | 完全削除 |
| Folder delete + restore | create folder → add files → delete folder → restore files | ファイル復元 |

---

## 8. Implementation Notes

### Changed Files (Backend)

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/archived_file.go` | ArchivedFile entity |
| Domain | `internal/domain/entity/file.go` | ToArchived, CanArchive methods |
| Domain | `internal/domain/service/file_archive_service.go` | Archive/Restore logic |
| Domain | `internal/domain/repository/file_repository.go` | ArchiveWithVersions, RestoreFromArchive |
| UseCase | `internal/usecase/storage/command/trash_file.go` | TrashFileCommand |
| UseCase | `internal/usecase/storage/command/restore_file.go` | RestoreFileCommand |
| UseCase | `internal/usecase/storage/command/permanent_delete_file.go` | PermanentDeleteFileCommand |
| UseCase | `internal/usecase/storage/command/empty_trash.go` | EmptyTrashCommand |
| UseCase | `internal/usecase/storage/query/list_trash.go` | ListTrashQuery |
| Interface | `internal/interface/handler/trash_handler.go` | Trash endpoints |
| Infra | `internal/infrastructure/repository/file_repository.go` | Archive table operations |

### Changed Files (Frontend)

| Category | File | Change |
|----------|------|--------|
| Route | `src/app/routes/trash.tsx` | ゴミ箱画面 |
| Component | `src/components/trash/trash-list.tsx` | ゴミ箱一覧表示 |
| Component | `src/components/trash/trash-toolbar.tsx` | 復元/削除アクション |
| Component | `src/components/dialogs/permanent-delete-dialog.tsx` | 完全削除確認 |
| Component | `src/components/dialogs/empty-trash-dialog.tsx` | 全削除確認 |
| API | `src/lib/api/trash.ts` | Trash API client |

### Migration

```sql
CREATE TABLE archived_files ( ... );           -- see storage-file.md
CREATE TABLE archived_file_versions ( ... );   -- see storage-file.md
```

### Considerations

- **Performance**: ゴミ箱空にする操作は非同期（202 Accepted）で大量削除に対応
- **Security**: 完全削除は不可逆操作のため確認ダイアログ必須
- **Data Integrity**: アーカイブ↔復元はトランザクション内で実行し整合性を保証
- **Batch Processing**: 期限切れ自動削除は日次バッチ（cron）で実行
