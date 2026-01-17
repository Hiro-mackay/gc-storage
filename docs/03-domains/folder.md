# Folder ドメイン

## 概要

Folderドメインは、ファイルを整理するための階層構造を持つフォルダの作成、移動、削除を担当します。
Storage Contextの一部として、ファイルの論理的な配置とナビゲーションの基盤を提供します。

---

## エンティティ

### Folder（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | フォルダの一意識別子 |
| name | string | Yes | フォルダ名 (1-255文字) |
| parent_id | UUID | No | 親フォルダID（ルートフォルダはNULL） |
| owner_id | UUID | Yes | 所有者ID（ユーザーまたはグループ） |
| owner_type | OwnerType | Yes | 所有者種別（user/group） |
| path | string | Yes | 正規化されたパス（検索用） |
| depth | int | Yes | 階層の深さ（ルート=0） |
| status | FolderStatus | Yes | フォルダ状態 |
| trashed_at | timestamp | No | ゴミ箱移動日時 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ビジネスルール:**
- R-F001: nameは空文字不可、1-255文字
- R-F002: nameに禁止文字（/ \ : * ? " < > |）を含まない
- R-F003: 同一親フォルダ内でnameは一意
- R-F004: ルートフォルダのparent_idはNULL
- R-F005: 自身または子孫フォルダへの移動は不可（循環参照防止）
- R-F006: 階層の最大深さは20
- R-F007: pathは`/{owner_type}/{owner_id}/...`形式で正規化

**ステータス遷移:**
```
┌─────────┐       ┌─────────┐       ┌─────────┐
│  active │──────▶│ trashed │──────▶│ deleted │
└────┬────┘       └────┬────┘       └─────────┘
     │                 │
     │                 │ restore
     │◀────────────────┘
     │
```

| ステータス | 説明 |
|-----------|------|
| active | アクティブ（通常表示） |
| trashed | ゴミ箱（復元可能） |
| deleted | 完全削除済み |

### FolderPath（値オブジェクト + 非正規化）

フォルダ階層のナビゲーション効率化のため、マテリアライズドパスを保持します。

| 属性 | 型 | 説明 |
|-----|-----|------|
| folder_id | UUID | フォルダID |
| ancestor_id | UUID | 祖先フォルダID |
| depth | int | この祖先からの距離 |

**用途:**
- 祖先フォルダの高速取得
- 権限継承の解決
- パンくずリストの生成

---

## 値オブジェクト

### FolderName

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | フォルダ名文字列 |

**バリデーション:**
- 1-255文字
- 禁止文字（/ \ : * ? " < > |）を含まない
- 先頭・末尾の空白はトリム
- 「.」「..」は使用不可

```go
type FolderName struct {
    value string
}

func NewFolderName(value string) (FolderName, error) {
    trimmed := strings.TrimSpace(value)

    if len(trimmed) == 0 {
        return FolderName{}, errors.New("folder name cannot be empty")
    }
    if len(trimmed) > 255 {
        return FolderName{}, errors.New("folder name must not exceed 255 characters")
    }

    invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
    if invalidChars.MatchString(trimmed) {
        return FolderName{}, errors.New("folder name contains invalid characters")
    }

    if trimmed == "." || trimmed == ".." {
        return FolderName{}, errors.New("folder name cannot be . or ..")
    }

    return FolderName{value: trimmed}, nil
}
```

### OwnerType

| 値 | 説明 |
|-----|------|
| user | ユーザー所有 |
| group | グループ所有 |

### FolderStatus

| 値 | 説明 |
|-----|------|
| active | アクティブ |
| trashed | ゴミ箱 |
| deleted | 削除済み |

---

## ドメインサービス

### FolderHierarchyService

**責務:** フォルダ階層の整合性管理と操作

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| CreateFolder | parentId, name, ownerType, ownerId | Folder | フォルダ作成 |
| MoveFolder | folderId, newParentId | Folder | フォルダ移動 |
| RenameFolder | folderId, newName | Folder | フォルダ名変更 |
| GetAncestors | folderId | []Folder | 祖先フォルダ取得 |
| GetDescendants | folderId | []Folder | 子孫フォルダ取得 |
| ValidatePath | folderId, targetParentId | bool | 移動可能性チェック |

```go
type FolderHierarchyService interface {
    CreateFolder(ctx context.Context, cmd CreateFolderCommand) (*Folder, error)
    MoveFolder(ctx context.Context, folderID, newParentID uuid.UUID) (*Folder, error)
    RenameFolder(ctx context.Context, folderID uuid.UUID, newName FolderName) (*Folder, error)
    GetAncestors(ctx context.Context, folderID uuid.UUID) ([]*Folder, error)
    GetDescendants(ctx context.Context, folderID uuid.UUID) ([]*Folder, error)
    ValidatePath(ctx context.Context, folderID, targetParentID uuid.UUID) error
}
```

**フォルダ移動のバリデーション:**
```go
func (s *FolderHierarchyServiceImpl) MoveFolder(
    ctx context.Context,
    folderID, newParentID uuid.UUID,
) (*Folder, error) {
    return s.txManager.WithTransaction(ctx, func(ctx context.Context) (*Folder, error) {
        // 1. フォルダ存在確認
        folder, err := s.folderRepo.FindByID(ctx, folderID)
        if err != nil {
            return nil, err
        }
        if folder.Status != FolderStatusActive {
            return nil, errors.New("folder is not active")
        }

        // 2. ルートフォルダは移動不可
        if folder.ParentID == nil {
            return nil, errors.New("cannot move root folder")
        }

        // 3. 移動先フォルダの確認
        var newParent *Folder
        if newParentID != uuid.Nil {
            newParent, err = s.folderRepo.FindByID(ctx, newParentID)
            if err != nil {
                return nil, errors.New("target folder not found")
            }
            if newParent.Status != FolderStatusActive {
                return nil, errors.New("target folder is not active")
            }
        }

        // 4. 循環参照チェック
        if err := s.ValidatePath(ctx, folderID, newParentID); err != nil {
            return nil, err
        }

        // 5. 同名フォルダチェック
        exists, err := s.folderRepo.ExistsByNameAndParent(ctx, folder.Name, newParentID)
        if err != nil {
            return nil, err
        }
        if exists {
            return nil, errors.New("folder with same name already exists in target")
        }

        // 6. 階層深さチェック
        descendants, err := s.GetDescendants(ctx, folderID)
        if err != nil {
            return nil, err
        }
        maxDescendantDepth := 0
        for _, d := range descendants {
            relativeDepth := d.Depth - folder.Depth
            if relativeDepth > maxDescendantDepth {
                maxDescendantDepth = relativeDepth
            }
        }
        newDepth := 0
        if newParent != nil {
            newDepth = newParent.Depth + 1
        }
        if newDepth+maxDescendantDepth > 20 {
            return nil, errors.New("folder hierarchy would exceed maximum depth")
        }

        // 7. フォルダ更新
        folder.ParentID = &newParentID
        folder.Depth = newDepth
        folder.Path = s.buildPath(ctx, folder)
        folder.UpdatedAt = time.Now()

        if err := s.folderRepo.Update(ctx, folder); err != nil {
            return nil, err
        }

        // 8. 子孫フォルダのパス・深さ更新
        if err := s.updateDescendantPaths(ctx, folder, descendants); err != nil {
            return nil, err
        }

        // 9. イベント発行
        s.eventPublisher.Publish(FolderMovedEvent{
            FolderID:      folderID,
            OldParentID:   folder.ParentID,
            NewParentID:   newParentID,
        })

        return folder, nil
    })
}

func (s *FolderHierarchyServiceImpl) ValidatePath(
    ctx context.Context,
    folderID, targetParentID uuid.UUID,
) error {
    // 自身への移動チェック
    if folderID == targetParentID {
        return errors.New("cannot move folder into itself")
    }

    // 子孫への移動チェック
    descendants, err := s.GetDescendants(ctx, folderID)
    if err != nil {
        return err
    }
    for _, d := range descendants {
        if d.ID == targetParentID {
            return errors.New("cannot move folder into its descendant")
        }
    }

    return nil
}
```

### FolderTrashService

**責務:** フォルダのゴミ箱管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| TrashFolder | folderId | void | ゴミ箱へ移動 |
| RestoreFolder | folderId | Folder | 復元 |
| PermanentlyDelete | folderId | void | 完全削除 |
| CleanupTrash | olderThan | int64 | 古いゴミ箱アイテム削除 |

```go
type FolderTrashService interface {
    TrashFolder(ctx context.Context, folderID uuid.UUID) error
    RestoreFolder(ctx context.Context, folderID uuid.UUID) (*Folder, error)
    PermanentlyDelete(ctx context.Context, folderID uuid.UUID) error
    CleanupTrash(ctx context.Context, olderThan time.Duration) (int64, error)
}
```

**ゴミ箱移動の処理:**
```go
func (s *FolderTrashServiceImpl) TrashFolder(
    ctx context.Context,
    folderID uuid.UUID,
) error {
    return s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        folder, err := s.folderRepo.FindByID(ctx, folderID)
        if err != nil {
            return err
        }

        // ルートフォルダはゴミ箱不可
        if folder.ParentID == nil {
            return errors.New("cannot trash root folder")
        }

        // 子孫フォルダも連動してゴミ箱へ
        descendants, err := s.hierarchyService.GetDescendants(ctx, folderID)
        if err != nil {
            return err
        }

        now := time.Now()

        // フォルダをゴミ箱へ
        folder.Status = FolderStatusTrashed
        folder.TrashedAt = &now
        if err := s.folderRepo.Update(ctx, folder); err != nil {
            return err
        }

        // 子孫フォルダもゴミ箱へ
        for _, d := range descendants {
            d.Status = FolderStatusTrashed
            d.TrashedAt = &now
            if err := s.folderRepo.Update(ctx, d); err != nil {
                return err
            }
        }

        // フォルダ内のファイルもゴミ箱へ
        if err := s.fileService.TrashByFolderID(ctx, folderID); err != nil {
            return err
        }

        s.eventPublisher.Publish(FolderTrashedEvent{FolderID: folderID})
        return nil
    })
}
```

---

## リポジトリ

### FolderRepository

```go
type FolderRepository interface {
    // 基本CRUD
    Create(ctx context.Context, folder *Folder) error
    FindByID(ctx context.Context, id uuid.UUID) (*Folder, error)
    Update(ctx context.Context, folder *Folder) error
    Delete(ctx context.Context, id uuid.UUID) error

    // 検索
    FindByParentID(ctx context.Context, parentID uuid.UUID, status FolderStatus) ([]*Folder, error)
    FindRootByOwner(ctx context.Context, ownerType OwnerType, ownerID uuid.UUID) (*Folder, error)
    FindByOwner(ctx context.Context, ownerType OwnerType, ownerID uuid.UUID, status FolderStatus) ([]*Folder, error)

    // 階層関連
    FindAncestors(ctx context.Context, folderID uuid.UUID) ([]*Folder, error)
    FindDescendants(ctx context.Context, folderID uuid.UUID) ([]*Folder, error)

    // 存在チェック
    ExistsByNameAndParent(ctx context.Context, name FolderName, parentID uuid.UUID) (bool, error)

    // ゴミ箱
    FindTrashedByOwner(ctx context.Context, ownerType OwnerType, ownerID uuid.UUID) ([]*Folder, error)
    FindTrashedOlderThan(ctx context.Context, threshold time.Time) ([]*Folder, error)
}
```

### FolderPathRepository

```go
type FolderPathRepository interface {
    // パス管理
    CreatePath(ctx context.Context, folderID, ancestorID uuid.UUID, depth int) error
    DeletePaths(ctx context.Context, folderID uuid.UUID) error
    FindAncestorIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error)
    FindDescendantIDs(ctx context.Context, folderID uuid.UUID) ([]uuid.UUID, error)

    // バルク更新
    UpdatePathsForMove(ctx context.Context, folderID, oldParentID, newParentID uuid.UUID) error
}
```

---

## 関係性

### エンティティ関係図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Folder Domain ERD                                   │
└─────────────────────────────────────────────────────────────────────────────┘

      ┌──────────────────┐          ┌──────────────────┐
      │      users       │          │     groups       │
      │    (external)    │          │    (external)    │
      └────────┬─────────┘          └────────┬─────────┘
               │                             │
               │ owner_id (when owner_type = 'user')
               │                             │
               │    owner_id (when owner_type = 'group')
               │                             │
               └──────────────┬──────────────┘
                              │
                              ▼
                     ┌──────────────────┐
                     │     folders      │
                     ├──────────────────┤
                     │ id               │◀──────────┐
                     │ name             │           │ parent_id
                     │ parent_id (FK)   │───────────┘
                     │ owner_id         │
                     │ owner_type       │
                     │ path             │
                     │ depth            │
                     │ status           │
                     │ trashed_at       │
                     │ created_at       │
                     │ updated_at       │
                     └────────┬─────────┘
                              │
                              │ 1:N
                              ▼
                     ┌──────────────────┐
                     │   folder_paths   │ (マテリアライズドパス)
                     ├──────────────────┤
                     │ folder_id (FK)   │
                     │ ancestor_id (FK) │
                     │ depth            │
                     └──────────────────┘

                              │
                              │ 1:N
                              ▼
                     ┌──────────────────┐
                     │      files       │ (File Domain)
                     │    (external)    │
                     └──────────────────┘
```

### 関係性ルール

| 関係 | カーディナリティ | 説明 |
|-----|----------------|------|
| Folder - Parent (Folder) | N:1 | 各フォルダは最大1つの親を持つ |
| Folder - Children (Folder) | 1:N | 各フォルダは複数の子フォルダを持てる |
| Folder - Owner (User/Group) | N:1 | 各フォルダは1つの所有者を持つ |
| Folder - FolderPath | 1:N | 各フォルダは複数の祖先パスエントリを持つ |
| Folder - File | 1:N | 各フォルダは複数のファイルを含める |

---

## 不変条件

1. **階層制約**
   - 自身または子孫への移動は不可（循環参照防止）
   - 階層の最大深さは20
   - ルートフォルダの移動・削除は不可

2. **命名制約**
   - 同一親フォルダ内で名前は一意
   - 禁止文字を含まない
   - 1-255文字

3. **所有権制約**
   - フォルダは必ず所有者（ユーザーまたはグループ）を持つ
   - 所有者の変更は所有権譲渡によってのみ可能

4. **ゴミ箱制約**
   - ゴミ箱移動時、子孫フォルダとファイルも連動
   - 復元時、親フォルダが存在する必要がある
   - 完全削除は30日経過後に自動実行

5. **パス整合性**
   - pathは常に正規化された形式を維持
   - 移動時、子孫のpathも更新

---

## ユースケース概要

| ユースケース | アクター | 概要 |
|------------|--------|------|
| CreateFolder | User | フォルダ作成 |
| RenameFolder | User | フォルダ名変更 |
| MoveFolder | User | フォルダ移動 |
| TrashFolder | User | ゴミ箱へ移動 |
| RestoreFolder | User | ゴミ箱から復元 |
| PermanentlyDeleteFolder | User | 完全削除 |
| ListFolderContents | User | フォルダ内容一覧 |
| GetFolderPath | User | パンくずリスト取得 |
| GetTrash | User | ゴミ箱内容表示 |
| EmptyTrash | User | ゴミ箱を空にする |
| CleanupOldTrash | System | 古いゴミ箱アイテム自動削除 |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| FolderCreated | フォルダ作成 | folderId, name, parentId, ownerType, ownerId |
| FolderRenamed | 名前変更 | folderId, oldName, newName |
| FolderMoved | 移動 | folderId, oldParentId, newParentId |
| FolderTrashed | ゴミ箱移動 | folderId |
| FolderRestored | 復元 | folderId |
| FolderPermanentlyDeleted | 完全削除 | folderId |

---

## 他コンテキストとの連携

### Identity Context（上流）
- UserIDの参照（owner_type = userの場合）

### Collaboration Context（上流）
- GroupIDの参照（owner_type = groupの場合）
- グループ作成時にグループルートフォルダを作成

### File Domain（同一コンテキスト）
- フォルダにはファイルが含まれる
- フォルダ削除時、配下のファイルも削除

### Authorization Context（下流）
- フォルダに対する権限付与
- 親フォルダからの権限継承

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [ファイルドメイン](./file.md) - ファイル管理
- [権限ドメイン](./permission.md) - 権限管理
- [データベース設計](../02-architecture/DATABASE.md) - スキーマ定義
