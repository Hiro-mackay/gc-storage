# Storage Folder - 詳細設計

## メタ情報

| 項目 | 内容 |
|------|------|
| ドキュメントID | SPEC-003 |
| バージョン | 1.0.0 |
| 最終更新日 | 2026-01-20 |
| ステータス | Draft |
| 関連ドメイン | [Folder Domain](../03-domains/folder.md) |

---

## ユーザーストーリー

### US-FD001: フォルダ作成

**As a** ユーザー
**I want to** フォルダを作成する
**So that** ファイルを整理できる

**受け入れ条件:**
- ルートレベルまたは既存フォルダ内にフォルダを作成できる
- 同一親フォルダ内でフォルダ名は一意
- 階層の最大深さは20

### US-FD002: フォルダ名変更

**As a** ユーザー
**I want to** フォルダ名を変更する
**So that** フォルダを適切に整理できる

**受け入れ条件:**
- フォルダ名を変更できる
- 同一親フォルダ内で重複名は不可

### US-FD003: フォルダ移動

**As a** ユーザー
**I want to** フォルダを別の場所へ移動する
**So that** フォルダ構造を整理できる

**受け入れ条件:**
- 別のフォルダ内へ移動できる
- ルートレベルへ移動できる
- 自身または子孫フォルダへの移動は不可（循環参照防止）
- 移動後の深さが20を超える場合は不可

### US-FD004: フォルダ削除

**As a** ユーザー
**I want to** フォルダを削除する
**So that** 不要なフォルダを整理できる

**受け入れ条件:**
- フォルダは即座に削除される（ゴミ箱なし）
- 配下のファイルはゴミ箱（archived_files）へ移動
- 配下のサブフォルダも再帰的に削除

### US-FD005: フォルダ内容一覧

**As a** ユーザー
**I want to** フォルダの内容を一覧表示する
**So that** ファイルやフォルダを確認できる

**受け入れ条件:**
- フォルダ内のサブフォルダとファイルを取得できる
- ルートレベルの内容を取得できる

### US-FD006: パンくずリスト取得

**As a** ユーザー
**I want to** フォルダの階層パスを取得する
**So that** 現在の場所を把握できる

**受け入れ条件:**
- 祖先フォルダを順序付きで取得できる
- ルートからの完全なパスが分かる

---

## API仕様

### POST /api/v1/folders

フォルダを作成する。

**Request:**
```json
{
  "name": "Documents",
  "parent_id": "uuid | null"
}
```

**Response (201 Created):**
```json
{
  "id": "uuid",
  "name": "Documents",
  "parent_id": "uuid | null",
  "owner_id": "uuid",
  "owner_type": "user",
  "depth": 1,
  "created_at": "2026-01-20T00:00:00Z",
  "updated_at": "2026-01-20T00:00:00Z"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 400 | 無効なフォルダ名 |
| 403 | 親フォルダへの書き込み権限なし |
| 404 | 親フォルダが存在しない |
| 409 | 同一親フォルダ内に同名フォルダが存在 |
| 422 | 階層深さ制限（20）を超過 |

---

### GET /api/v1/folders/{folder_id}

フォルダ情報を取得する。

**Response (200 OK):**
```json
{
  "id": "uuid",
  "name": "Documents",
  "parent_id": "uuid | null",
  "owner_id": "uuid",
  "owner_type": "user",
  "depth": 1,
  "created_at": "2026-01-20T00:00:00Z",
  "updated_at": "2026-01-20T00:00:00Z"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | フォルダへのアクセス権限なし |
| 404 | フォルダが存在しない |

---

### PUT /api/v1/folders/{folder_id}/name

フォルダ名を変更する。

**Request:**
```json
{
  "name": "New Documents"
}
```

**Response (200 OK):**
```json
{
  "id": "uuid",
  "name": "New Documents",
  "updated_at": "2026-01-20T00:00:00Z"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 400 | 無効なフォルダ名 |
| 403 | フォルダへの書き込み権限なし |
| 404 | フォルダが存在しない |
| 409 | 同一親フォルダ内に同名フォルダが存在 |

---

### PUT /api/v1/folders/{folder_id}/parent

フォルダを別の場所へ移動する。

**Request:**
```json
{
  "parent_id": "uuid | null"
}
```

**Response (200 OK):**
```json
{
  "id": "uuid",
  "parent_id": "uuid | null",
  "depth": 2,
  "updated_at": "2026-01-20T00:00:00Z"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | フォルダまたは移動先への権限なし |
| 404 | フォルダまたは移動先が存在しない |
| 409 | 移動先に同名フォルダが存在 |
| 422 | 循環参照（自身または子孫への移動） |
| 422 | 階層深さ制限を超過 |

---

### DELETE /api/v1/folders/{folder_id}

フォルダを削除する。配下のファイルはゴミ箱へ移動。

**Response (200 OK):**
```json
{
  "deleted_folder_count": 5,
  "archived_file_count": 12
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | フォルダへの削除権限なし |
| 404 | フォルダが存在しない |

---

### GET /api/v1/folders/{folder_id}/contents

フォルダの内容（サブフォルダとファイル）を取得する。

**Query Parameters:**
| パラメータ | 型 | 説明 |
|-----------|-----|------|
| sort | string | ソート順（name, created_at, updated_at, size） |
| order | string | 昇順(asc)/降順(desc) |
| limit | int | 取得件数（デフォルト: 50） |
| cursor | string | ページネーションカーソル |

**Response (200 OK):**
```json
{
  "folder": {
    "id": "uuid",
    "name": "Documents",
    "parent_id": "uuid | null",
    "depth": 1
  },
  "folders": [
    {
      "id": "uuid",
      "name": "Work",
      "created_at": "2026-01-20T00:00:00Z",
      "updated_at": "2026-01-20T00:00:00Z"
    }
  ],
  "files": [
    {
      "id": "uuid",
      "name": "report.pdf",
      "mime_type": "application/pdf",
      "size": 10485760,
      "created_at": "2026-01-20T00:00:00Z",
      "updated_at": "2026-01-20T00:00:00Z"
    }
  ],
  "next_cursor": "string | null"
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | フォルダへのアクセス権限なし |
| 404 | フォルダが存在しない |

---

### GET /api/v1/folders/root/contents

ルートレベルの内容を取得する。

**Query Parameters:**
| パラメータ | 型 | 説明 |
|-----------|-----|------|
| owner_type | string | user または group（デフォルト: user） |
| owner_id | uuid | グループIDの場合に指定 |
| sort | string | ソート順 |
| order | string | 昇順/降順 |
| limit | int | 取得件数 |
| cursor | string | ページネーションカーソル |

**Response (200 OK):**
```json
{
  "folders": [
    {
      "id": "uuid",
      "name": "Documents",
      "created_at": "2026-01-20T00:00:00Z",
      "updated_at": "2026-01-20T00:00:00Z"
    }
  ],
  "files": [
    {
      "id": "uuid",
      "name": "readme.txt",
      "mime_type": "text/plain",
      "size": 1024,
      "created_at": "2026-01-20T00:00:00Z",
      "updated_at": "2026-01-20T00:00:00Z"
    }
  ],
  "next_cursor": "string | null"
}
```

---

### GET /api/v1/folders/{folder_id}/ancestors

フォルダの祖先一覧を取得する（パンくずリスト用）。

**Response (200 OK):**
```json
{
  "ancestors": [
    {
      "id": "uuid",
      "name": "Documents",
      "depth": 0
    },
    {
      "id": "uuid",
      "name": "Work",
      "depth": 1
    },
    {
      "id": "uuid",
      "name": "Reports",
      "depth": 2
    }
  ]
}
```

**エラー:**
| コード | 条件 |
|--------|------|
| 403 | フォルダへのアクセス権限なし |
| 404 | フォルダが存在しない |

---

## データ変更

### 新規テーブル

#### folders

```sql
CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL,
    owner_type VARCHAR(10) NOT NULL CHECK (owner_type IN ('user', 'group')),
    depth INTEGER NOT NULL DEFAULT 0 CHECK (depth >= 0 AND depth <= 20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_folders_name_parent UNIQUE (parent_id, name) WHERE parent_id IS NOT NULL,
    CONSTRAINT uq_folders_name_owner_root UNIQUE (owner_id, owner_type, name) WHERE parent_id IS NULL
);

CREATE INDEX idx_folders_parent ON folders(parent_id);
CREATE INDEX idx_folders_owner ON folders(owner_id, owner_type);
CREATE INDEX idx_folders_depth ON folders(depth);
```

#### folder_paths (閉包テーブル)

```sql
CREATE TABLE folder_paths (
    ancestor_id UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    descendant_id UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    path_length INTEGER NOT NULL CHECK (path_length >= 0),

    PRIMARY KEY (ancestor_id, descendant_id)
);

CREATE INDEX idx_folder_paths_ancestor ON folder_paths(ancestor_id);
CREATE INDEX idx_folder_paths_descendant ON folder_paths(descendant_id);
CREATE INDEX idx_folder_paths_length ON folder_paths(path_length);
```

---

## 実装ノート

### 設計原則

**問題点（アンチパターン）:**
- 閉包テーブル操作がUseCaseに露出（手続き的）
- 循環参照チェック、深さ計算などのドメインロジックがUseCase層にリーク
- フォルダと閉包テーブルが別々に管理され、整合性担保がUseCase責務に

**解決方針:**
- **FolderHierarchy集約**: フォルダと閉包テーブルを1つの集約として管理
- **ドメインサービス**: 階層操作のビジネスルールをカプセル化
- **リポジトリの責務拡大**: 閉包テーブル操作を隠蔽し、ドメイン操作を提供

---

### パッケージ構成

```
backend/internal/
├── domain/
│   ├── entity/
│   │   └── folder.go              # Folder（リッチモデル）
│   ├── valueobject/
│   │   ├── folder_name.go
│   │   └── owner_type.go
│   ├── service/
│   │   └── folder_hierarchy_service.go  # 階層操作ドメインサービス
│   └── repository/
│       └── folder_repository.go   # 階層操作を含むリポジトリ
│
├── usecase/storage/
│   ├── command/
│   │   ├── create_folder.go       # シンプルなオーケストレーション
│   │   ├── rename_folder.go
│   │   ├── move_folder.go
│   │   └── delete_folder.go
│   └── query/
│       ├── get_folder.go
│       ├── list_folder_contents.go
│       └── get_ancestors.go
│
└── infrastructure/
    └── repository/
        └── folder_repository.go   # 閉包テーブル操作を内包
```

---

### エンティティ設計（リッチドメインモデル）

#### Folder エンティティ

```go
// internal/domain/entity/folder.go

type Folder struct {
    id        uuid.UUID
    name      valueobject.FolderName
    parentID  *uuid.UUID
    ownerID   uuid.UUID
    ownerType valueobject.OwnerType
    depth     int
    createdAt time.Time
    updatedAt time.Time
}

// ファクトリメソッド - 不変条件を保証
func NewFolder(
    name valueobject.FolderName,
    parentID *uuid.UUID,
    ownerID uuid.UUID,
    ownerType valueobject.OwnerType,
    parentDepth int,  // 親の深さ（ルートなら-1）
) (*Folder, error) {
    depth := 0
    if parentID != nil {
        depth = parentDepth + 1
    }

    // 深さ制限チェック
    if depth > MaxFolderDepth {
        return nil, ErrMaxDepthExceeded
    }

    return &Folder{
        id:        uuid.New(),
        name:      name,
        parentID:  parentID,
        ownerID:   ownerID,
        ownerType: ownerType,
        depth:     depth,
        createdAt: time.Now(),
        updatedAt: time.Now(),
    }, nil
}

// ドメインロジック - 移動可否判定
func (f *Folder) CanMoveTo(newParentID *uuid.UUID, descendantIDs []uuid.UUID) error {
    // 自身への移動は不可
    if newParentID != nil && *newParentID == f.id {
        return ErrCannotMoveToSelf
    }

    // 子孫への移動は不可（循環参照）
    if newParentID != nil {
        for _, descendantID := range descendantIDs {
            if *newParentID == descendantID {
                return ErrCannotMoveToDescendant
            }
        }
    }

    return nil
}

// 移動後の深さを計算
func (f *Folder) CalculateNewDepth(newParentDepth int) int {
    if newParentDepth < 0 {
        return 0  // ルートへ移動
    }
    return newParentDepth + 1
}

// 移動後の深さ制限チェック
func (f *Folder) ValidateDepthAfterMove(newDepth int, maxDescendantPathLength int) error {
    if newDepth+maxDescendantPathLength > MaxFolderDepth {
        return ErrMaxDepthExceededAfterMove
    }
    return nil
}

// 移動実行（状態変更）
func (f *Folder) MoveTo(newParentID *uuid.UUID, newDepth int) {
    f.parentID = newParentID
    f.depth = newDepth
    f.updatedAt = time.Now()
}

// 名前変更
func (f *Folder) Rename(newName valueobject.FolderName) {
    f.name = newName
    f.updatedAt = time.Now()
}

// ルートフォルダ判定
func (f *Folder) IsRoot() bool {
    return f.parentID == nil
}

// Getters
func (f *Folder) ID() uuid.UUID              { return f.id }
func (f *Folder) Name() valueobject.FolderName { return f.name }
func (f *Folder) ParentID() *uuid.UUID       { return f.parentID }
func (f *Folder) OwnerID() uuid.UUID         { return f.ownerID }
func (f *Folder) Depth() int                 { return f.depth }
```

---

### ドメインサービス

階層操作に関する複雑なビジネスロジックをカプセル化。

```go
// internal/domain/service/folder_hierarchy_service.go

type FolderHierarchyService interface {
    // 移動バリデーション（循環参照、深さ制限）
    ValidateMove(ctx context.Context, folder *entity.Folder, newParentID *uuid.UUID) error

    // 削除時のファイルアーカイブ処理
    ArchiveFilesInSubtree(ctx context.Context, folderID uuid.UUID, archivedBy uuid.UUID) (archivedCount int, err error)
}

type folderHierarchyService struct {
    folderRepo      repository.FolderRepository
    fileRepo        repository.FileRepository
    archiveService  service.FileArchiveService
}

func (s *folderHierarchyService) ValidateMove(
    ctx context.Context,
    folder *entity.Folder,
    newParentID *uuid.UUID,
) error {
    // 1. 子孫ID取得
    descendantIDs, err := s.folderRepo.GetDescendantIDs(ctx, folder.ID())
    if err != nil {
        return err
    }

    // 2. 循環参照チェック（エンティティメソッド）
    if err := folder.CanMoveTo(newParentID, descendantIDs); err != nil {
        return err
    }

    // 3. 新しい親の深さ取得
    newParentDepth := -1  // ルートの場合
    if newParentID != nil {
        newParent, err := s.folderRepo.FindByID(ctx, *newParentID)
        if err != nil {
            return err
        }
        newParentDepth = newParent.Depth()
    }

    // 4. 深さ制限チェック（エンティティメソッド）
    newDepth := folder.CalculateNewDepth(newParentDepth)
    maxDescendantPathLength, err := s.folderRepo.GetMaxDescendantPathLength(ctx, folder.ID())
    if err != nil {
        return err
    }

    return folder.ValidateDepthAfterMove(newDepth, maxDescendantPathLength)
}

func (s *folderHierarchyService) ArchiveFilesInSubtree(
    ctx context.Context,
    folderID uuid.UUID,
    archivedBy uuid.UUID,
) (int, error) {
    // 子孫フォルダID取得（自身含む）
    folderIDs, err := s.folderRepo.GetDescendantIDsIncludingSelf(ctx, folderID)
    if err != nil {
        return 0, err
    }

    archivedCount := 0
    for _, fid := range folderIDs {
        files, err := s.fileRepo.FindByFolderID(ctx, fid)
        if err != nil {
            return archivedCount, err
        }

        for _, file := range files {
            if !file.CanArchive() {
                continue
            }
            // FileArchiveServiceに委譲
            // ...
            archivedCount++
        }
    }

    return archivedCount, nil
}
```

---

### リポジトリ設計

リポジトリが閉包テーブル操作を完全に隠蔽し、ドメイン操作を提供。

```go
// internal/domain/repository/folder_repository.go

type FolderRepository interface {
    // 基本操作
    FindByID(ctx context.Context, id uuid.UUID) (*entity.Folder, error)
    FindByParentID(ctx context.Context, parentID uuid.UUID) ([]*entity.Folder, error)
    FindRootByOwner(ctx context.Context, ownerID uuid.UUID, ownerType valueobject.OwnerType) ([]*entity.Folder, error)

    // ドメイン操作（閉包テーブル操作を内包）
    CreateWithHierarchy(ctx context.Context, folder *entity.Folder) error
    MoveWithHierarchy(ctx context.Context, folder *entity.Folder, newParentID *uuid.UUID) error
    DeleteWithSubtree(ctx context.Context, folderID uuid.UUID) (deletedFolderIDs []uuid.UUID, err error)

    // 階層クエリ
    GetAncestors(ctx context.Context, folderID uuid.UUID) ([]*entity.Folder, error)
    GetDescendantIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error)
    GetDescendantIDsIncludingSelf(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error)
    GetMaxDescendantPathLength(ctx context.Context, folderID uuid.UUID) (int, error)

    // 存在チェック
    ExistsByNameInParent(ctx context.Context, name valueobject.FolderName, parentID *uuid.UUID, ownerID uuid.UUID, ownerType valueobject.OwnerType) (bool, error)
}
```

#### インフラ層実装（閉包テーブル操作を隠蔽）

```go
// internal/infrastructure/repository/folder_repository.go

func (r *folderRepository) CreateWithHierarchy(ctx context.Context, folder *entity.Folder) error {
    return r.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. フォルダ作成
        if err := r.createFolder(ctx, folder); err != nil {
            return err
        }

        // 2. 自己参照エントリ挿入
        if err := r.insertSelfReference(ctx, folder.ID()); err != nil {
            return err
        }

        // 3. 祖先パス挿入
        if folder.ParentID() != nil {
            if err := r.insertAncestorPaths(ctx, folder.ID(), *folder.ParentID()); err != nil {
                return err
            }
        }

        return nil
    })
}

func (r *folderRepository) MoveWithHierarchy(ctx context.Context, folder *entity.Folder, newParentID *uuid.UUID) error {
    return r.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. サブツリーの祖先パスを削除（自己参照は保持）
        if err := r.deleteSubtreeAncestorPaths(ctx, folder.ID()); err != nil {
            return err
        }

        // 2. 新しい祖先パスを挿入
        if newParentID != nil {
            if err := r.insertSubtreeAncestorPaths(ctx, folder.ID(), *newParentID); err != nil {
                return err
            }
        }

        // 3. フォルダ更新
        if err := r.updateFolder(ctx, folder); err != nil {
            return err
        }

        // 4. 子孫フォルダの深さ更新
        return r.updateDescendantsDepth(ctx, folder.ID(), folder.Depth())
    })
}

func (r *folderRepository) DeleteWithSubtree(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error) {
    var deletedIDs []uuid.UUID

    err := r.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 子孫フォルダID取得（深い順）
        ids, err := r.getDescendantIDsOrderByDepthDesc(ctx, folderID)
        if err != nil {
            return err
        }
        deletedIDs = append(ids, folderID)

        // CASCADEで閉包テーブルも削除される
        for _, id := range deletedIDs {
            if err := r.deleteFolder(ctx, id); err != nil {
                return err
            }
        }
        return nil
    })

    return deletedIDs, err
}
```

---

### UseCase実装（シンプルなオーケストレーション）

UseCaseはドメインオブジェクトの組み合わせのみを行う。

```go
// internal/usecase/folder/command/create_folder.go

type CreateFolderCommand struct {
    folderRepo repository.FolderRepository
}

type CreateFolderInput struct {
    Name      valueobject.FolderName
    ParentID  *uuid.UUID
    OwnerID   uuid.UUID
    OwnerType valueobject.OwnerType
}

func (c *CreateFolderCommand) Execute(ctx context.Context, input CreateFolderInput) (*entity.Folder, error) {
    // 1. 名前重複チェック
    exists, err := c.folderRepo.ExistsByNameInParent(ctx, input.Name, input.ParentID, input.OwnerID, input.OwnerType)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, ErrFolderNameAlreadyExists
    }

    // 2. 親フォルダの深さ取得
    parentDepth := -1
    if input.ParentID != nil {
        parent, err := c.folderRepo.FindByID(ctx, *input.ParentID)
        if err != nil {
            return nil, err
        }
        parentDepth = parent.Depth()
    }

    // 3. フォルダ作成（エンティティファクトリで深さ制限チェック）
    folder, err := entity.NewFolder(input.Name, input.ParentID, input.OwnerID, input.OwnerType, parentDepth)
    if err != nil {
        return nil, err
    }

    // 4. 永続化（閉包テーブル操作はリポジトリ内）
    if err := c.folderRepo.CreateWithHierarchy(ctx, folder); err != nil {
        return nil, err
    }

    return folder, nil
}
```

```go
// internal/usecase/folder/command/move_folder.go

type MoveFolderCommand struct {
    folderRepo       repository.FolderRepository
    hierarchyService service.FolderHierarchyService
}

type MoveFolderInput struct {
    FolderID    uuid.UUID
    NewParentID *uuid.UUID
}

func (c *MoveFolderCommand) Execute(ctx context.Context, input MoveFolderInput) (*entity.Folder, error) {
    // 1. フォルダ取得
    folder, err := c.folderRepo.FindByID(ctx, input.FolderID)
    if err != nil {
        return nil, err
    }

    // 2. バリデーション（ドメインサービス）
    if err := c.hierarchyService.ValidateMove(ctx, folder, input.NewParentID); err != nil {
        return nil, err
    }

    // 3. 新しい深さ計算
    newParentDepth := -1
    if input.NewParentID != nil {
        newParent, _ := c.folderRepo.FindByID(ctx, *input.NewParentID)
        newParentDepth = newParent.Depth()
    }
    newDepth := folder.CalculateNewDepth(newParentDepth)

    // 4. 状態変更（エンティティメソッド）
    folder.MoveTo(input.NewParentID, newDepth)

    // 5. 永続化（閉包テーブル操作はリポジトリ内）
    if err := c.folderRepo.MoveWithHierarchy(ctx, folder, input.NewParentID); err != nil {
        return nil, err
    }

    return folder, nil
}
```

```go
// internal/usecase/folder/command/delete_folder.go

type DeleteFolderCommand struct {
    folderRepo       repository.FolderRepository
    hierarchyService service.FolderHierarchyService
}

type DeleteFolderInput struct {
    FolderID uuid.UUID
    UserID   uuid.UUID
}

type DeleteFolderOutput struct {
    DeletedFolderCount int
    ArchivedFileCount  int
}

func (c *DeleteFolderCommand) Execute(ctx context.Context, input DeleteFolderInput) (*DeleteFolderOutput, error) {
    // 1. サブツリー内のファイルをアーカイブ（ドメインサービス）
    archivedCount, err := c.hierarchyService.ArchiveFilesInSubtree(ctx, input.FolderID, input.UserID)
    if err != nil {
        return nil, err
    }

    // 2. フォルダ削除（リポジトリがサブツリー削除を処理）
    deletedIDs, err := c.folderRepo.DeleteWithSubtree(ctx, input.FolderID)
    if err != nil {
        return nil, err
    }

    return &DeleteFolderOutput{
        DeletedFolderCount: len(deletedIDs),
        ArchivedFileCount:  archivedCount,
    }, nil
}
```

---

### 定数・エラー定義

```go
// internal/domain/entity/folder.go

const (
    MaxFolderDepth     = 20
    FolderNameMaxBytes = 255
)

// ドメインエラー
var (
    ErrMaxDepthExceeded          = errors.New("max folder depth exceeded")
    ErrMaxDepthExceededAfterMove = errors.New("max depth would be exceeded after move")
    ErrCannotMoveToSelf          = errors.New("cannot move folder to itself")
    ErrCannotMoveToDescendant    = errors.New("cannot move folder to its descendant")
    ErrFolderNameAlreadyExists   = errors.New("folder name already exists in parent")
)
```

---

## テスト計画

### ユニットテスト

| テスト対象 | テストケース |
|-----------|-------------|
| FolderName | 有効なフォルダ名、禁止文字、最大長、「.」「..」 |
| Folder Entity | 深さ制限、バリデーション |

### 統合テスト

| テストケース | 概要 |
|-------------|------|
| フォルダ作成 | ルート、サブフォルダ作成と閉包テーブル検証 |
| フォルダ作成（深さ制限） | 深さ20を超える作成が拒否される |
| フォルダ名変更 | 名前変更と重複チェック |
| フォルダ移動 | 移動と閉包テーブル更新 |
| フォルダ移動（循環参照） | 自身または子孫への移動が拒否される |
| フォルダ移動（深さ制限） | 移動後に深さ20を超える場合が拒否される |
| フォルダ削除 | 削除とファイルアーカイブ |
| 祖先取得 | パンくずリストの正確性 |
| フォルダ内容一覧 | サブフォルダとファイルの取得 |

### E2Eテスト

| シナリオ | 概要 |
|---------|------|
| フォルダ階層作成 | ネストしたフォルダ構造を作成できる |
| フォルダ移動と内容確認 | 移動後も内容が保持される |
| フォルダ削除とファイル復元 | 削除されたフォルダ内のファイルをゴミ箱から復元できる |
| パンくずナビゲーション | 任意のフォルダから祖先をたどれる |

---

## 受け入れ基準

### 機能要件

- [ ] ルートレベルにフォルダを作成できる
- [ ] サブフォルダを作成できる
- [ ] フォルダ名を変更できる
- [ ] フォルダを別の場所へ移動できる
- [ ] フォルダをルートレベルへ移動できる
- [ ] 循環参照となる移動が拒否される
- [ ] 深さ20を超える操作が拒否される
- [ ] フォルダを削除できる
- [ ] 削除されたフォルダ内のファイルがゴミ箱に移動する
- [ ] フォルダの内容一覧を取得できる
- [ ] ルートレベルの内容一覧を取得できる
- [ ] 祖先フォルダ一覧を取得できる（パンくずリスト）

### 非機能要件

- [ ] 閉包テーブルが正しく維持される
- [ ] 深いフォルダ階層でもパフォーマンスが維持される
- [ ] 大量のファイルを含むフォルダ削除が完了する

---

## 関連ドキュメント

- [Folder Domain](../03-domains/folder.md)
- [File Domain](../03-domains/file.md)
- [Storage File Spec](./storage-file.md)
