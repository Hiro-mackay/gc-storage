# Permission ドメイン

## 概要

Permissionドメインは、リソースに対するアクセス権限の付与、取り消し、継承解決を担当します。
Authorization Contextの中核として、PBAC（Policy-Based）とReBAC（Relationship-Based）のハイブリッドモデルを実装します。

---

## 認可モデル概要

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Hybrid Authorization Model                                │
│                      PBAC + ReBAC                                           │
└─────────────────────────────────────────────────────────────────────────────┘

【設計原則】
• PBAC: 「このPermissionを持っているか？」で最終判定
• ReBAC: 「どの関係性を通じてPermissionを得ているか？」を解決
• Role = Permission の集合（割り当ての便宜のため）
• 関係性の連鎖を辿ってPermissionを収集し、最終的にPermissionで判定

【認可の流れ】
1. ユーザーがリソースに対する操作を要求
2. 関係性を辿って適用可能な全Permissionを収集
   - 直接付与されたPermission
   - Role経由のPermission
   - 親フォルダからの継承Permission
   - グループメンバーシップ経由のPermission
3. 要求されたPermissionが収集セットに含まれるかチェック
4. 含まれていれば許可、なければ拒否
```

---

## ロールモデル

### リソースロール（4段階）

```
Owner > Content Manager > Contributor > Viewer
```

| ロール | 閲覧 | 作成/編集/削除 | 共有設定 | 移動IN | 移動OUT | ルートフォルダ削除 |
|--------|:----:|:--------------:|:--------:|:------:|:-------:|:------------------:|
| Viewer | Yes | No | No | No | No | No |
| Contributor | Yes | Yes | Yes | Yes | No | No |
| Content Manager | Yes | Yes | Yes | Yes | Yes | No |
| Owner | Yes | Yes | Yes | Yes | Yes | Yes |

**権限継承**: 各ロールはサブディレクトリに対して同じ権限を継承する

### 共有時のロール付与制限

| 操作者のロール | 付与可能なロール |
|----------------|------------------|
| Owner | Viewer, Contributor, Content Manager |
| Content Manager | Viewer, Contributor, Content Manager |
| Contributor | Viewer, Contributor |
| Viewer | なし（共有不可） |

**重要ルール:**
- Ownerロールは直接付与不可（所有権譲渡で移転）
- 自分のロール以下のロールのみ付与可能

---

## エンティティ

### PermissionGrant（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 権限付与の一意識別子 |
| resource_type | ResourceType | Yes | リソース種別（file/folder） |
| resource_id | UUID | Yes | リソースID |
| grantee_type | GranteeType | Yes | 付与先種別（user/group） |
| grantee_id | UUID | Yes | 付与先ID |
| role | Role | No | 付与ロール（NULLの場合はPermission直接付与） |
| permission | Permission | No | 直接付与Permission（NULLの場合はRole経由） |
| granted_by | UUID | Yes | 付与者のユーザーID |
| granted_at | timestamp | Yes | 付与日時 |

**ビジネスルール:**
- R-PG001: roleまたはpermissionのいずれかは必須
- R-PG002: 同一(resource, grantee, role, permission)の組み合わせは一意
- R-PG003: ownerロールは直接付与不可（所有権譲渡で管理）
- R-PG004: 付与者は対象リソースに対してpermission:grant権限を持つ必要がある
- R-PG005: 付与者は自分のロール以下のロールのみ付与可能

### Relationship（集約ルート）

Google Zanzibar スタイルの関係性タプルを管理します。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 関係性の一意識別子 |
| subject_type | string | Yes | 主体の種別（user/group/folder） |
| subject_id | UUID | Yes | 主体のID |
| relation | RelationType | Yes | 関係性の種類 |
| object_type | string | Yes | 対象の種別（file/folder） |
| object_id | UUID | Yes | 対象のID |
| created_at | timestamp | Yes | 作成日時 |

**ビジネスルール:**
- R-R001: (subject_type, subject_id, relation, object_type, object_id)の組み合わせは一意
- R-R002: ownerリレーションは1リソースにつき1つのみ
- R-R003: parentリレーションは階層構造を形成（循環参照不可）

---

## 値オブジェクト

### Permission

`{resource}:{action}` 形式で定義されるアクセス権限。

**ファイル操作:**
| Permission | 説明 |
|------------|------|
| file:read | ファイルの閲覧・ダウンロード |
| file:write | ファイルのアップロード・更新 |
| file:rename | ファイル名の変更 |
| file:delete | ファイルの削除（ゴミ箱へ） |
| file:restore | ゴミ箱からの復元 |
| file:permanent_delete | 完全削除 |
| file:move_in | ファイルをフォルダ内へ移動（移動先権限） |
| file:move_out | ファイルをフォルダ外へ移動（移動元権限） |
| file:share | 共有リンクの作成 |

**フォルダ操作:**
| Permission | 説明 |
|------------|------|
| folder:read | フォルダ内容の閲覧 |
| folder:create | サブフォルダ・ファイルの作成 |
| folder:rename | フォルダ名の変更 |
| folder:delete | フォルダの削除 |
| folder:move_in | フォルダをこのフォルダ内へ移動（移動先権限） |
| folder:move_out | フォルダをこのフォルダ外へ移動（移動元権限） |
| folder:share | 共有リンクの作成 |

**権限管理:**
| Permission | 説明 |
|------------|------|
| permission:read | リソースの権限一覧の閲覧 |
| permission:grant | 他ユーザー/グループへの権限付与 |
| permission:revoke | 他ユーザー/グループからの権限取り消し |

**ルート操作:**
| Permission | 説明 |
|------------|------|
| root:delete | ルートレベルの Shared Folder を削除（Ownerのみ） |

### Role

Permissionの集合として定義されるロール。

**リソースロール（ファイル/フォルダ）:**

| Role | Permissions |
|------|-------------|
| viewer | file:read, folder:read |
| contributor | viewer + file:write, file:rename, file:delete, file:restore, file:move_in, folder:create, folder:rename, folder:delete, folder:move_in, file:share, folder:share, permission:read, permission:grant, permission:revoke |
| content_manager | contributor + file:move_out, folder:move_out |
| owner | content_manager + file:permanent_delete, root:delete + 完全制御 |

**ロールとPermissionのマッピング:**

| Permission | Viewer | Contributor | Content Manager | Owner |
|------------|:------:|:-----------:|:---------------:|:-----:|
| file:read | Yes | Yes | Yes | Yes |
| folder:read | Yes | Yes | Yes | Yes |
| file:write | No | Yes | Yes | Yes |
| file:rename | No | Yes | Yes | Yes |
| file:delete | No | Yes | Yes | Yes |
| file:restore | No | Yes | Yes | Yes |
| file:move_in | No | Yes | Yes | Yes |
| file:move_out | No | No | Yes | Yes |
| file:share | No | Yes | Yes | Yes |
| folder:create | No | Yes | Yes | Yes |
| folder:rename | No | Yes | Yes | Yes |
| folder:delete | No | Yes | Yes | Yes |
| folder:move_in | No | Yes | Yes | Yes |
| folder:move_out | No | No | Yes | Yes |
| folder:share | No | Yes | Yes | Yes |
| permission:read | No | Yes | Yes | Yes |
| permission:grant | No | Yes | Yes | Yes |
| permission:revoke | No | Yes | Yes | Yes |
| file:permanent_delete | No | No | No | Yes |
| root:delete | No | No | No | Yes |

### RelationType

関係性の種類。

| Relation | 説明 | 例 |
|----------|------|-----|
| owner | 所有者 | user ──owner──▶ file |
| member | メンバー | user ──member──▶ group |
| parent | 親子関係 | folder ──parent──▶ file |
| viewer | 閲覧者ロール | group ──viewer──▶ folder |
| contributor | 投稿者ロール | user ──contributor──▶ folder |
| content_manager | コンテンツ管理者ロール | user ──content_manager──▶ folder |

### ResourceType

| 値 | 説明 |
|-----|------|
| file | ファイル |
| folder | フォルダ |

### GranteeType

| 値 | 説明 |
|-----|------|
| user | ユーザー |
| group | グループ |

---

## ドメインサービス

### PermissionResolver

**責務:** 関係性を辿ってPermissionを収集・判定

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| HasPermission | userId, resourceType, resourceId, permission | bool | 権限判定 |
| CollectPermissions | userId, resourceType, resourceId | PermissionSet | 全権限収集 |
| GetEffectiveRole | userId, resourceType, resourceId | Role | 有効なロール取得 |

**権限収集アルゴリズム:**

1. **所有者チェック**: user ──owner──▶ resource の場合、全権限を付与
2. **直接付与された権限**: user ──{role}──▶ resource
3. **グループ経由の権限**: user ──member──▶ group ──{role}──▶ resource
4. **階層経由の権限**: resource ◀──parent── ancestor（サブディレクトリへの継承）

### PermissionGrantService

**責務:** 権限の付与・取り消し

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| GrantRole | cmd | PermissionGrant | ロール付与 |
| GrantPermission | cmd | PermissionGrant | Permission直接付与 |
| Revoke | grantId | void | 権限取り消し |
| RevokeAll | resourceType, resourceId, granteeType, granteeId | void | 全権限取り消し |

**ロール付与の制約:**
1. 付与者はpermission:grant権限を持つ必要がある
2. 付与者は自分のロール以下のロールのみ付与可能
3. ownerロールは直接付与不可（所有権譲渡で管理）

### RelationshipService

**責務:** 関係性の管理（所有権、階層、メンバーシップ）

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| SetOwner | resourceType, resourceId, ownerId | void | 所有者設定 |
| TransferOwnership | resourceType, resourceId, newOwnerId | void | 所有権譲渡 |
| SetParent | childType, childId, parentType, parentId | void | 親子関係設定 |
| AddMember | userId, groupId | void | グループメンバー追加 |
| RemoveMember | userId, groupId | void | グループメンバー削除 |

---

## リポジトリ

### PermissionGrantRepository

| 操作 | 説明 |
|-----|------|
| Create | 権限付与作成 |
| FindByID | ID検索 |
| FindByResource | リソースへの権限一覧 |
| FindByResourceAndGrantee | リソース・付与先での検索 |
| FindByResourceGranteeAndRole | リソース・付与先・ロールでの検索 |
| FindByGrantee | 付与先への権限一覧 |
| Delete | 削除 |
| DeleteByResourceAndGrantee | リソース・付与先の全権限削除 |

### RelationshipRepository

| 操作 | 説明 |
|-----|------|
| Create | 関係性作成 |
| Delete | 削除 |
| DeleteByTuple | タプルで削除 |
| Exists | 存在チェック |
| FindRelated | 主体から対象を検索 |
| FindRelatedReverse | 対象から主体を検索 |
| FindByObject | オブジェクトへの全関係性 |
| FindBySubject | 主体からの全関係性 |

---

## 関係性

### エンティティ関係図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Permission Domain ERD                                 │
└─────────────────────────────────────────────────────────────────────────────┘

      ┌──────────────────┐          ┌──────────────────┐
      │      users       │          │     groups       │
      │    (external)    │          │    (external)    │
      └────────┬─────────┘          └────────┬─────────┘
               │                             │
               │ grantee_id, granted_by      │ grantee_id
               │                             │
               └──────────────┬──────────────┘
                              │
                              ▼
                   ┌──────────────────────┐
                   │  permission_grants   │
                   ├──────────────────────┤
                   │ id                   │
                   │ resource_type        │
                   │ resource_id          │
                   │ grantee_type         │
                   │ grantee_id           │
                   │ role                 │
                   │ permission           │
                   │ granted_by (FK)      │
                   │ granted_at           │
                   └──────────────────────┘

                   ┌──────────────────────┐
                   │    relationships     │ (Zanzibar-style Tuples)
                   ├──────────────────────┤
                   │ id                   │
                   │ subject_type         │
                   │ subject_id           │
                   │ relation             │
                   │ object_type          │
                   │ object_id            │
                   │ created_at           │
                   └──────────────────────┘
```

### Relationship Tupleの例

```
【所有関係】
(user:alice, owner, file:report.pdf)
(user:alice, owner, folder:my-documents)

【メンバーシップ】
(user:alice, member, group:engineering)
(user:bob, member, group:engineering)

【階層関係】
(folder:team-docs, parent, folder:projects)
(folder:projects, parent, file:spec.pdf)

【権限付与】
(group:engineering, viewer, folder:shared)
(user:charlie, contributor, folder:projects)
(user:dave, content_manager, folder:archive)
```

---

## 不変条件

1. **所有権制約**
   - 各リソースには必ず1人の所有者が存在
   - ownerリレーションは1リソースにつき1つのみ
   - 所有権の移転は明示的な譲渡操作でのみ可能

2. **権限付与制約**
   - 自分より高いロールは付与不可
   - ownerロールは直接付与不可（所有権譲渡で管理）
   - 同一(resource, grantee, role)の重複付与は不可

3. **階層整合性**
   - parentリレーションは循環参照不可
   - 親が削除された場合、継承権限は無効化

4. **グループメンバーシップ**
   - グループ削除時、全メンバーシップリレーションも削除
   - ユーザー無効化時、メンバーシップは維持（再有効化時に復活）

---

## ユースケース概要

| ユースケース | アクター | 概要 |
|------------|--------|------|
| CheckPermission | System | ユーザーの権限チェック |
| GrantRole | Contributor/Content Manager/Owner | ロール付与 |
| GrantPermission | Contributor/Content Manager/Owner | Permission直接付与 |
| RevokePermission | Contributor/Content Manager/Owner | 権限取り消し |
| ListPermissions | User | リソースの権限一覧 |
| TransferOwnership | Owner | 所有権譲渡 |
| GetMyPermissions | User | 自分の権限確認 |
| ListAccessibleResources | User | アクセス可能なリソース一覧 |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| PermissionGranted | 権限付与 | resourceType, resourceId, granteeType, granteeId, role/permission, grantedBy |
| PermissionRevoked | 権限取消 | resourceType, resourceId, granteeType, granteeId |
| RoleAssigned | ロール割当 | resourceType, resourceId, granteeType, granteeId, role |
| RoleRemoved | ロール削除 | resourceType, resourceId, granteeType, granteeId, role |
| OwnershipTransferred | 所有権譲渡 | resourceType, resourceId, previousOwnerId, newOwnerId |
| PermissionInherited | 権限継承 | resourceType, resourceId, inheritedFrom |

---

## 他コンテキストとの連携

### Identity Context（上流）
- UserIDの参照（付与者、被付与者）

### Collaboration Context（上流）
- GroupIDの参照（グループ経由の権限）
- メンバーシップ情報の取得

### Storage Context（上流）
- FileID, FolderIDの参照
- 親フォルダ階層の取得

### Sharing Context（下流）
- 共有リンク作成時のfile:share権限チェック

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [セキュリティ設計](../02-architecture/SECURITY.md) - 認可モデルの詳細
- [グループドメイン](./group.md) - グループロール
- [ファイルドメイン](./file.md) - ファイル権限
- [フォルダドメイン](./folder.md) - フォルダ権限
