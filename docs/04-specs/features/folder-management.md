# Folder Management - フォルダ CRUD & 階層ナビゲーション

## Meta

| Item | Value |
|------|-------|
| Status | Draft |
| Priority | High |
| Tier | 2 (Storage) |
| Spec IDs | 2A.1 folder-create, 2A.2 folder-list, 2A.3 folder-rename, 2A.4 folder-move |
| Domain Refs | `03-domains/folder.md` |
| Depends On | `features/` (auth prerequisite) |

---

## 1. User Stories

**Primary:**
> As a user, I want to create, rename, move, and delete folders so that I can organize my files in a hierarchical structure.

**Secondary:**
> As a user, I want to navigate folder hierarchies with breadcrumbs so that I always know my current location and can move between levels easily.

### Context

フォルダはファイル整理の基盤。閉包テーブル（Closure Table）で階層構造を管理し、効率的な祖先・子孫クエリを実現する。Personal Folder はユーザー登録時に自動作成され、Shared Folder はユーザーが明示的に作成する。

---

## 2. Domain Behaviors

### Referenced Domain Rules

| Rule ID | Summary | Domain File |
|---------|---------|-------------|
| R-FD001 | 同一親フォルダ内で name は一意 | `03-domains/folder.md` |
| R-FD003 | 自身/子孫への移動不可（循環参照防止） | `03-domains/folder.md` |
| R-FD004 | 階層の最大深さは 20 | `03-domains/folder.md` |
| R-FD005 | 削除時、配下ファイルは ArchivedFile へ移動 | `03-domains/folder.md` |
| R-FD006 | 削除時、配下サブフォルダも再帰的に削除 | `03-domains/folder.md` |
| R-FD007 | 新規作成時 owner_id = created_by = 作成者 | `03-domains/folder.md` |
| R-FD009 | Personal Folder は削除不可 | `03-domains/folder.md` |
| R-FN001 | フォルダ名 1-255 バイト（UTF-8） | `03-domains/folder.md` |
| R-FN002 | 禁止文字 `/ \ : * ? " < > \|` を含まない | `03-domains/folder.md` |
| R-FC001 | 各フォルダは自己参照エントリを持つ | `03-domains/folder.md` |
| R-FC002 | 作成時、自己参照と全祖先への参照を挿入 | `03-domains/folder.md` |
| R-FC003 | 移動時、旧パス削除→新パス挿入 | `03-domains/folder.md` |

### Feature-Specific Rules

| Rule ID | Description |
|---------|-------------|
| FS-FM001 | ルートレベルにはフォルダのみ存在（ファイルは必ずフォルダ内） |
| FS-FM002 | Personal Folder は user.personal_folder_id で判定 |
| FS-FM003 | フォルダ削除はゴミ箱を経由せず即時削除 |

### State Transitions

```
Folder: active (単一状態、ゴミ箱なし)

削除フロー:
  active → 即時削除（配下ファイルは archived_files へ移動）
```

---

## 3. API Contract

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/folders` | Cookie(session_id) | フォルダ作成 |
| GET | `/api/v1/folders/{folder_id}` | Cookie(session_id) | フォルダ情報取得 |
| GET | `/api/v1/folders/{folder_id}/contents` | Cookie(session_id) | フォルダ内容一覧 |
| GET | `/api/v1/folders/root/contents` | Cookie(session_id) | ルートレベル一覧 |
| PUT | `/api/v1/folders/{folder_id}/name` | Cookie(session_id) | フォルダ名変更 |
| PUT | `/api/v1/folders/{folder_id}/parent` | Cookie(session_id) | フォルダ移動 |
| DELETE | `/api/v1/folders/{folder_id}` | Cookie(session_id) | フォルダ削除 |
| GET | `/api/v1/folders/{folder_id}/ancestors` | Cookie(session_id) | 祖先一覧（パンくず） |

### Request / Response Details

#### `POST /api/v1/folders` - フォルダ作成

**Request Body:**
```json
{
  "name": "Documents",
  "parent_id": "uuid | null"
}
```

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| name | string | Yes | 1-255 bytes, no forbidden chars | フォルダ名 |
| parent_id | uuid | No | valid UUID or null | 親フォルダ ID |

**Success Response (201):**
```json
{
  "id": "uuid",
  "name": "Documents",
  "parent_id": "uuid | null",
  "owner_id": "uuid",
  "created_by": "uuid",
  "depth": 1,
  "created_at": "2026-01-20T00:00:00Z",
  "updated_at": "2026-01-20T00:00:00Z"
}
```

**Error Responses:**

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | 無効なフォルダ名 | `VALIDATION_ERROR` |
| 403 | 親フォルダへの書き込み権限なし | `FORBIDDEN` |
| 404 | 親フォルダが存在しない | `NOT_FOUND` |
| 409 | 同名フォルダが存在 | `CONFLICT` |
| 422 | 階層深さ制限（20）超過 | `DEPTH_EXCEEDED` |

#### `GET /api/v1/folders/{folder_id}/contents` - フォルダ内容一覧

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| sort | string | name, created_at, updated_at, size |
| order | string | asc / desc |
| limit | int | 取得件数（default: 50） |
| cursor | string | ページネーションカーソル |

**Success Response (200):**
```json
{
  "folder": { "id": "uuid", "name": "Documents", "parent_id": "uuid", "depth": 1 },
  "folders": [{ "id": "uuid", "name": "Work", "created_at": "...", "updated_at": "..." }],
  "files": [{ "id": "uuid", "name": "report.pdf", "mime_type": "application/pdf", "size": 10485760, "created_at": "...", "updated_at": "..." }],
  "next_cursor": "string | null"
}
```

#### `PUT /api/v1/folders/{folder_id}/name` - フォルダ名変更

**Request Body:**
```json
{ "name": "New Documents" }
```

**Success Response (200):**
```json
{ "id": "uuid", "name": "New Documents", "updated_at": "..." }
```

| Code | Condition | Error Code |
|------|-----------|------------|
| 400 | 無効なフォルダ名 | `VALIDATION_ERROR` |
| 409 | 同名フォルダ存在 | `CONFLICT` |

#### `PUT /api/v1/folders/{folder_id}/parent` - フォルダ移動

**Request Body:**
```json
{ "parent_id": "uuid | null" }
```

**Success Response (200):**
```json
{ "id": "uuid", "parent_id": "uuid | null", "depth": 2, "updated_at": "..." }
```

| Code | Condition | Error Code |
|------|-----------|------------|
| 409 | 移動先に同名フォルダ | `CONFLICT` |
| 422 | 循環参照 or 深さ制限超過 | `INVALID_MOVE` |

#### `DELETE /api/v1/folders/{folder_id}` - フォルダ削除

**Success Response (200):**
```json
{ "deleted_folder_count": 5, "archived_file_count": 12 }
```

#### `GET /api/v1/folders/{folder_id}/ancestors` - 祖先一覧

**Success Response (200):**
```json
{
  "ancestors": [
    { "id": "uuid", "name": "Documents", "depth": 0 },
    { "id": "uuid", "name": "Work", "depth": 1 }
  ]
}
```

---

## 4. Frontend UI

### Layout / Wireframe

```
+------------------------------------------------------------------+
| Breadcrumb: Home > Documents > Projects                   [Search]|
+------------------------------------------------------------------+
| [+ New Folder] [Upload]                    [Grid|List] [Sort v]  |
+------------------------------------------------------------------+
|  [Folder]      [Folder]      [File]        [File]                |
|  Documents     Images        report.pdf    notes.txt             |
+------------------------------------------------------------------+
```

### Components

| Component | Type | Description |
|-----------|------|-------------|
| BreadcrumbNav | UI | パンくずリスト表示、各要素クリックでナビゲーション |
| FileToolbar | UI | 新規フォルダ、アップロード、表示切替、ソート |
| FolderGrid / FolderList | UI | グリッド/リスト表示でフォルダ・ファイル一覧 |
| CreateFolderDialog | Modal | 新規フォルダ名入力ダイアログ |
| RenameFolderDialog | Modal | フォルダ名変更ダイアログ |
| MoveFolderDialog | Modal | 移動先フォルダツリー選択ダイアログ |
| DeleteConfirmDialog | Modal | 削除確認ダイアログ |
| EmptyState | UI | 空フォルダ時の案内表示 |

### State Management

| State | Store | Type | Description |
|-------|-------|------|-------------|
| Folder contents | TanStack Query | Server | フォルダ内容（folders + files） |
| Ancestors | TanStack Query | Server | パンくずリスト用祖先フォルダ |
| View mode | Zustand | Global | grid / list 切替 |
| Sort option | Zustand | Global | ソート条件（name, date, size） |
| Selected items | useState | Local | 選択中アイテム一覧 |

### User Interactions

1. ユーザーがフォルダをダブルクリック → 該当フォルダへ遷移（URL 変更）
2. パンくず要素クリック → 該当フォルダへ遷移
3. [+ New Folder] クリック → CreateFolderDialog 表示 → 名前入力 → POST /api/v1/folders → 一覧リフレッシュ
4. 右クリック → コンテキストメニュー → Rename / Move / Delete 選択

---

## 5. Integration Flow

### フォルダ作成シーケンス

```
Client          Frontend        API             DB
  |                |              |                |
  |-- click ------>|              |                |
  |                |-- dialog --->|                |
  |-- submit ----->|              |                |
  |                |-- POST ----->|                |
  |                |              |-- name check ->|
  |                |              |<-- ok ---------|
  |                |              |-- insert ----->|
  |                |              |   folder +     |
  |                |              |   closure      |
  |                |              |<-- ok ---------|
  |                |<-- 201 ------|                |
  |<-- refresh ----|              |                |
```

### Error Handling Flow

- API エラー → TanStack Query の onError → toast 通知（エラーメッセージ表示）
- 409 Conflict → ダイアログ内にインラインエラー（"同名フォルダが存在します"）
- 422 Depth exceeded → ダイアログ内にインラインエラー（"階層制限を超えています"）

---

## 6. Acceptance Criteria

### Happy Path
- [ ] AC-01: ルートレベルに Shared Folder を作成できる
- [ ] AC-02: サブフォルダを作成できる
- [ ] AC-03: フォルダ名を変更できる
- [ ] AC-04: フォルダを別のフォルダへ移動できる
- [ ] AC-05: フォルダをルートレベルへ移動できる
- [ ] AC-06: フォルダ内容一覧を取得できる（サブフォルダ + ファイル）
- [ ] AC-07: ルートレベルの一覧を取得できる（Personal + Shared）
- [ ] AC-08: 祖先一覧を取得できパンくずリスト表示に使える

### Validation Errors
- [ ] AC-10: 空文字のフォルダ名で作成するとバリデーションエラー
- [ ] AC-11: 禁止文字を含むフォルダ名で作成するとバリデーションエラー
- [ ] AC-12: 同一親フォルダ内に同名フォルダ作成で 409 エラー

### Authorization
- [ ] AC-20: 権限なしの親フォルダに作成で 403 エラー

### Edge Cases
- [ ] AC-30: 深さ 20 を超える作成で 422 エラー
- [ ] AC-31: 自身への移動で 422 エラー（循環参照防止）
- [ ] AC-32: 子孫フォルダへの移動で 422 エラー（循環参照防止）
- [ ] AC-33: 移動後に深さ 20 を超える場合に 422 エラー
- [ ] AC-34: フォルダ削除時に配下ファイルが archived_files へ移動
- [ ] AC-35: Personal Folder の削除は拒否される

---

## 7. Test Plan

### Backend Unit Tests

| Test | UseCase/Service | Key Assertions |
|------|----------------|----------------|
| NewFolder depth check | entity.NewFolder | depth > 20 で ErrMaxDepthExceeded |
| CanMoveTo self | entity.Folder | 自身 ID で ErrCannotMoveToSelf |
| CanMoveTo descendant | entity.Folder | 子孫 ID で ErrCannotMoveToDescendant |
| FolderName validation | valueobject.FolderName | 禁止文字、空文字、"."".." でエラー |
| CreateFolder duplicate | CreateFolderCommand | 同名存在時にエラー |

### Backend Integration Tests

| Test | Endpoint | Setup | Assertions |
|------|----------|-------|------------|
| Create root folder | POST /api/v1/folders | session cookie | 201, closure table に自己参照 |
| Create subfolder | POST /api/v1/folders | parent folder | 201, depth=parent+1, closure paths |
| List folder contents | GET /folders/:id/contents | folders + files | 200, folders + files 返却 |
| Rename folder | PUT /folders/:id/name | existing folder | 200, name 更新 |
| Move folder | PUT /folders/:id/parent | folder tree | 200, closure table 更新 |
| Delete folder | DELETE /folders/:id | folder with files | 200, files archived |
| Get ancestors | GET /folders/:id/ancestors | nested folders | 200, ordered ancestors |

### Frontend Tests

| Test | Component | Type | Assertions |
|------|-----------|------|------------|
| Render folder list | FolderGrid | Unit | フォルダ・ファイル表示 |
| Create folder dialog | CreateFolderDialog | Integration | 入力→送信→一覧更新 |
| Breadcrumb navigation | BreadcrumbNav | Unit | 祖先表示、クリックでナビゲーション |
| Empty state | EmptyState | Unit | 空フォルダ時に案内表示 |

### E2E Tests (future)

| Test | Flow | Assertions |
|------|------|------------|
| Full folder CRUD | create → rename → move → delete | 各操作成功、一覧に反映 |
| Breadcrumb navigation | create nested → navigate via breadcrumb | 各レベルへ正しく遷移 |

---

## 8. Implementation Notes

### Changed Files (Backend)

| Layer | File | Change |
|-------|------|--------|
| Domain | `internal/domain/entity/folder.go` | Folder entity (rich model) |
| Domain | `internal/domain/valueobject/folder_name.go` | FolderName value object |
| Domain | `internal/domain/service/folder_hierarchy_service.go` | 階層操作ドメインサービス |
| Domain | `internal/domain/repository/folder_repository.go` | FolderRepository interface |
| UseCase | `internal/usecase/storage/command/create_folder.go` | CreateFolderCommand |
| UseCase | `internal/usecase/storage/command/rename_folder.go` | RenameFolderCommand |
| UseCase | `internal/usecase/storage/command/move_folder.go` | MoveFolderCommand |
| UseCase | `internal/usecase/storage/command/delete_folder.go` | DeleteFolderCommand |
| UseCase | `internal/usecase/storage/query/list_folder_contents.go` | ListFolderContentsQuery |
| UseCase | `internal/usecase/storage/query/get_ancestors.go` | GetAncestorsQuery |
| Infra | `internal/infrastructure/repository/folder_repository.go` | 閉包テーブル操作実装 |

### Changed Files (Frontend)

| Category | File | Change |
|----------|------|--------|
| Route | `src/app/routes/files.tsx` | マイファイル画面 |
| Route | `src/app/routes/files.$folderId.tsx` | フォルダ詳細画面 |
| Component | `src/components/file-browser/breadcrumb-nav.tsx` | パンくずリスト |
| Component | `src/components/file-browser/file-toolbar.tsx` | ツールバー |
| Component | `src/components/file-browser/folder-grid.tsx` | グリッド表示 |
| Component | `src/components/file-browser/folder-list.tsx` | リスト表示 |
| Component | `src/components/dialogs/create-folder-dialog.tsx` | 新規フォルダ |
| Component | `src/components/dialogs/rename-dialog.tsx` | 名前変更 |
| Component | `src/components/dialogs/move-dialog.tsx` | 移動先選択 |

### Migration

```sql
CREATE TABLE folders ( ... );        -- see storage-folder.md
CREATE TABLE folder_paths ( ... );   -- closure table
```

### Considerations

- **Performance**: 閉包テーブルにより祖先・子孫クエリは O(1) JOIN で完了
- **Security**: 全エンドポイントで session_id cookie 認証必須、フォルダ権限チェック
- **Backward Compatibility**: 新規テーブルのため互換性問題なし
