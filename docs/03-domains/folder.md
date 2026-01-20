# Folder ドメイン

## 概要

Folderドメインは、ファイルを整理するための階層構造を持つフォルダの作成、移動、削除を担当します。
Storage Contextの一部として、ファイルの論理的な配置とナビゲーションの基盤を提供します。

### 設計方針

- **閉包テーブル（Closure Table）**: 階層構造の効率的なクエリのため閉包テーブルを使用
- **暗黙的ルート**: parent_id=nullが所有者のルートレベル（明示的なルートフォルダは作成しない）
- **ゴミ箱なし**: フォルダは直接削除、中のファイルのみアーカイブテーブルへ移動
- **グループ所有対応**: フォルダはユーザーまたはグループが所有可能

---

## エンティティ

### Folder（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | フォルダの一意識別子 |
| name | FolderName | Yes | フォルダ名（値オブジェクト） |
| parent_id | UUID | No | 親フォルダID（nullならルートレベル） |
| owner_id | UUID | Yes | 所有者ID（ユーザーまたはグループ） |
| owner_type | OwnerType | Yes | 所有者種別（user/group） |
| depth | int | Yes | 階層の深さ（ルートレベル=0） |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-FD001 | 同一親フォルダ内でnameは一意 |
| R-FD002 | 同一所有者のルートレベル（parent_id=null）でnameは一意 |
| R-FD003 | 自身または子孫フォルダへの移動は不可（循環参照防止） |
| R-FD004 | 階層の最大深さは20 |
| R-FD005 | 削除時、配下のファイルはArchivedFileへ移動 |
| R-FD006 | 削除時、配下のサブフォルダも再帰的に削除 |

---

### FolderClosure（閉包テーブル）

フォルダ階層の関係を表現する閉包テーブル。祖先・子孫の関係を効率的にクエリ可能。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| ancestor_id | UUID | Yes | 祖先フォルダID |
| descendant_id | UUID | Yes | 子孫フォルダID |
| path_length | int | Yes | 祖先から子孫までの距離（自己参照は0） |

**ビジネスルール:**

| ID | ルール |
|----|--------|
| R-FC001 | 各フォルダは自己参照エントリを持つ（ancestor_id = descendant_id, path_length = 0） |
| R-FC002 | フォルダ作成時、自己参照と全祖先への参照を挿入 |
| R-FC003 | フォルダ移動時、旧パスの参照を削除し新パスの参照を挿入 |
| R-FC004 | フォルダ削除時、関連する全エントリを削除 |

**閉包テーブルの例:**

フォルダ構造:
```
Root (A)
└── Documents (B)
    └── Work (C)
        └── Reports (D)
```

閉包テーブルエントリ:
| ancestor_id | descendant_id | path_length |
|-------------|---------------|-------------|
| A | A | 0 |
| A | B | 1 |
| A | C | 2 |
| A | D | 3 |
| B | B | 0 |
| B | C | 1 |
| B | D | 2 |
| C | C | 0 |
| C | D | 1 |
| D | D | 0 |

---

## 値オブジェクト

### FolderName

| 属性 | 型 | 説明 |
|-----|-----|------|
| value | string | フォルダ名文字列 |

**要件:**

| ID | 要件 |
|----|------|
| R-FN001 | 1-255バイト（UTF-8） |
| R-FN002 | 禁止文字（`/ \ : * ? " < > |`）を含まない |
| R-FN003 | 先頭・末尾の空白はトリム |
| R-FN004 | 空文字は不可 |
| R-FN005 | 「.」「..」は使用不可 |

---

### OwnerType

所有者タイプを表す値オブジェクト。

| 値 | 説明 |
|-----|------|
| user | 個人ユーザー所有 |
| group | グループ所有 |

---

## 定数

| 定数名 | 値 | 説明 |
|--------|-----|------|
| MaxFolderDepth | 20 | フォルダ階層の最大深さ |
| FolderNameMaxBytes | 255 | フォルダ名最大長 |

---

## 操作フロー

### フォルダ作成

```
1. クライアント → API: CreateFolder（name, parent_id, owner_id, owner_type）
2. API:
   - parent_id検証（存在確認、所有者一致確認）
   - 同一親/ルートレベルでの名前重複チェック
   - 階層深さチェック（parent.depth + 1 <= 20）
   - Folder作成（depth = parent.depth + 1、またはparent_id=nullなら0）
   - FolderClosure挿入（自己参照 + 全祖先への参照）
3. API → クライアント: 作成されたFolder返却
```

### フォルダ移動

```
1. クライアント → API: MoveFolder（folder_id, new_parent_id）
2. API:
   - フォルダ存在確認
   - 移動先フォルダ存在確認、所有者一致確認
   - 循環参照チェック（移動先が自身または子孫でないこと）
   - 階層深さチェック（移動後の最深部が20以下）
   - 同一親での名前重複チェック
3. トランザクション内:
   - FolderClosureから旧パスのエントリ削除
   - FolderClosureに新パスのエントリ挿入
   - Folder.parent_id, depth更新
   - 子孫フォルダのdepth更新
4. API → クライアント: 更新されたFolder返却
```

### フォルダ削除

フォルダはゴミ箱を経由せず直接削除。配下のファイルはアーカイブテーブルへ移動。

```
1. クライアント → API: DeleteFolder（folder_id）
2. API:
   - フォルダ存在確認
   - FolderClosureから子孫フォルダID一覧取得
3. トランザクション内:
   - 子孫フォルダ内の全ファイルをArchivedFileへ移動（Fileドメインと連携）
   - 対象フォルダ内の全ファイルをArchivedFileへ移動
   - 子孫フォルダのFolderClosureエントリ削除
   - 子孫フォルダ削除（深い順）
   - 対象フォルダのFolderClosureエントリ削除
   - 対象フォルダ削除
4. API → クライアント: 成功レスポンス
```

### フォルダ名変更

```
1. クライアント → API: RenameFolder（folder_id, new_name）
2. API:
   - フォルダ存在確認
   - 同一親/ルートレベルでの名前重複チェック
   - Folder.name更新
3. API → クライアント: 更新されたFolder返却
```

### 祖先フォルダ取得（パンくずリスト）

```
1. クライアント → API: GetAncestors（folder_id）
2. API:
   - FolderClosureから祖先ID取得（path_length > 0）
   - 祖先フォルダ情報取得（path_lengthの降順でソート）
3. API → クライアント: 祖先フォルダ一覧返却
```

### フォルダ内容一覧

```
1. クライアント → API: ListFolderContents（folder_id または owner_id+owner_type）
2. API:
   - folder_id指定: 該当フォルダの直下を取得
   - folder_id=null: 所有者のルートレベルを取得
   - サブフォルダ一覧取得（parent_id = folder_id）
   - ファイル一覧取得（folder_id = folder_id）
3. API → クライアント: フォルダとファイルの一覧返却
```

---

## リポジトリ

### FolderRepository

| 操作 | 説明 |
|-----|------|
| Create | フォルダ作成 |
| FindByID | ID検索 |
| Update | 更新 |
| Delete | 物理削除 |
| FindByParentID | 親フォルダIDで子フォルダ取得 |
| FindRootByOwner | 所有者のルートレベルフォルダ取得 |
| FindByOwner | 所有者の全フォルダ取得 |
| ExistsByNameAndParent | 親フォルダ内での名前重複チェック |
| ExistsByNameAndOwnerRoot | 所有者ルートレベルでの名前重複チェック |
| UpdateDepth | 深さ更新 |
| BulkDelete | 一括削除 |

### FolderClosureRepository

| 操作 | 説明 |
|-----|------|
| InsertSelfReference | 自己参照エントリ挿入 |
| InsertAncestorPaths | 祖先パスエントリ一括挿入 |
| FindAncestorIDs | 祖先ID一覧取得 |
| FindDescendantIDs | 子孫ID一覧取得 |
| FindDescendantsWithDepth | 子孫とpath_length取得 |
| DeleteByDescendant | 子孫IDで関連エントリ削除 |
| DeleteSubtreePaths | サブツリーの全パスエントリ削除 |
| MoveSubtree | サブツリー移動（旧パス削除→新パス挿入） |

---

## 不変条件

### 階層制約

| ID | 不変条件 |
|----|---------|
| I-FH001 | 自身または子孫への移動は不可（循環参照防止） |
| I-FH002 | 階層の最大深さは20 |
| I-FH003 | 移動後の全子孫が深さ制限を超えないこと |

### 命名制約

| ID | 不変条件 |
|----|---------|
| I-FN001 | 同一親フォルダ内で名前は一意 |
| I-FN002 | 同一所有者のルートレベルで名前は一意 |
| I-FN003 | フォルダ名に禁止文字を含まない |

### 所有権制約

| ID | 不変条件 |
|----|---------|
| I-FO001 | フォルダは必ず所有者（ユーザーまたはグループ）を持つ |
| I-FO002 | フォルダ移動は同一所有者内のみ可能 |
| I-FO003 | 所有者の変更は所有権譲渡によってのみ可能 |

### 削除制約

| ID | 不変条件 |
|----|---------|
| I-FD001 | フォルダ削除時、配下のファイルはArchivedFileへ移動 |
| I-FD002 | フォルダ削除時、配下のサブフォルダも再帰的に削除 |
| I-FD003 | 閉包テーブルエントリはフォルダ削除時に必ず削除 |

### 閉包テーブル整合性

| ID | 不変条件 |
|----|---------|
| I-FC001 | 各フォルダは必ず自己参照エントリを持つ |
| I-FC002 | 祖先→子孫の全パスが閉包テーブルに存在する |
| I-FC003 | 孤立したエントリ（参照先フォルダが存在しない）を持たない |

---

## ユースケース

| ユースケース | アクター | 概要 |
|------------|--------|------|
| CreateFolder | User | フォルダ作成 |
| RenameFolder | User | フォルダ名変更 |
| MoveFolder | User | フォルダ移動 |
| DeleteFolder | User | フォルダ削除（ファイルはアーカイブ） |
| ListFolderContents | User | フォルダ内容一覧 |
| GetAncestors | User | パンくずリスト取得 |
| GetFolderInfo | User | フォルダ情報取得 |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| FolderCreated | フォルダ作成 | folderId, name, parentId, ownerType, ownerId |
| FolderRenamed | 名前変更 | folderId, oldName, newName |
| FolderMoved | 移動 | folderId, oldParentId, newParentId |
| FolderDeleted | 削除 | folderId, archivedFileIds（アーカイブされたファイルID一覧） |

---

## 他コンテキストとの連携

### Identity Context（上流）

- UserIDの参照（owner_type = userの場合）

### Collaboration Context（上流）

- GroupIDの参照（owner_type = groupの場合）
- グループ作成時にグループの初期フォルダ作成は不要（暗黙的ルート）

### File Domain（同一コンテキスト）

- フォルダにはファイルが含まれる（folder_id参照）
- フォルダ削除時、配下のファイルをArchivedFileへ移動

### Authorization Context（下流）

- フォルダに対する権限付与
- 親フォルダからの権限継承
- グループ所有フォルダの管理権限はグループ管理者が持つ

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [ファイルドメイン](./file.md) - ファイル管理
- [権限ドメイン](./permission.md) - 権限管理
