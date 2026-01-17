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

## エンティティ

### PermissionGrant（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 権限付与の一意識別子 |
| resource_type | ResourceType | Yes | リソース種別（file/folder/group） |
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

### Relationship（集約ルート）

Google Zanzibar スタイルの関係性タプルを管理します。

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 関係性の一意識別子 |
| subject_type | string | Yes | 主体の種別（user/group/folder） |
| subject_id | UUID | Yes | 主体のID |
| relation | RelationType | Yes | 関係性の種類 |
| object_type | string | Yes | 対象の種別（file/folder/group） |
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
| file:delete | ファイルの削除（ゴミ箱へ） |
| file:restore | ゴミ箱からの復元 |
| file:permanent_delete | 完全削除 |
| file:move | ファイルの移動 |
| file:rename | ファイル名の変更 |
| file:share | 共有リンクの作成 |

**フォルダ操作:**
| Permission | 説明 |
|------------|------|
| folder:read | フォルダ内容の閲覧 |
| folder:create | サブフォルダの作成 |
| folder:delete | フォルダの削除 |
| folder:move | フォルダの移動 |
| folder:rename | フォルダ名の変更 |
| folder:share | 共有リンクの作成 |

**権限管理:**
| Permission | 説明 |
|------------|------|
| permission:read | リソースの権限設定を閲覧 |
| permission:grant | 他ユーザー/グループへの権限付与 |
| permission:revoke | 権限の取り消し |

**グループ管理:**
| Permission | 説明 |
|------------|------|
| group:read | グループ情報の閲覧 |
| group:update | グループ設定の変更 |
| group:delete | グループの削除 |
| group:member:read | メンバー一覧の閲覧 |
| group:member:add | メンバーの追加 |
| group:member:remove | メンバーの削除 |
| group:member:role | メンバーのロール変更 |

```go
type Permission string

const (
    // File permissions
    PermFileRead            Permission = "file:read"
    PermFileWrite           Permission = "file:write"
    PermFileDelete          Permission = "file:delete"
    PermFileRestore         Permission = "file:restore"
    PermFilePermanentDelete Permission = "file:permanent_delete"
    PermFileMove            Permission = "file:move"
    PermFileRename          Permission = "file:rename"
    PermFileShare           Permission = "file:share"

    // Folder permissions
    PermFolderRead   Permission = "folder:read"
    PermFolderCreate Permission = "folder:create"
    PermFolderDelete Permission = "folder:delete"
    PermFolderMove   Permission = "folder:move"
    PermFolderRename Permission = "folder:rename"
    PermFolderShare  Permission = "folder:share"

    // Permission management
    PermPermissionRead   Permission = "permission:read"
    PermPermissionGrant  Permission = "permission:grant"
    PermPermissionRevoke Permission = "permission:revoke"

    // Group permissions
    PermGroupRead         Permission = "group:read"
    PermGroupUpdate       Permission = "group:update"
    PermGroupDelete       Permission = "group:delete"
    PermGroupMemberRead   Permission = "group:member:read"
    PermGroupMemberAdd    Permission = "group:member:add"
    PermGroupMemberRemove Permission = "group:member:remove"
    PermGroupMemberRole   Permission = "group:member:role"
)
```

### Role

Permissionの集合として定義されるロール。

**リソースロール（ファイル/フォルダ）:**

| Role | Permissions |
|------|-------------|
| viewer | file:read, folder:read |
| editor | viewer + file:write, file:rename, file:move, folder:create, folder:rename, folder:move |
| manager | editor + file:delete, file:restore, file:share, folder:delete, folder:share, permission:read, permission:grant, permission:revoke |
| owner | manager + file:permanent_delete + 完全制御 |

```go
type Role string

const (
    RoleViewer  Role = "viewer"
    RoleEditor  Role = "editor"
    RoleManager Role = "manager"
    RoleOwner   Role = "owner"
)

func (r Role) Permissions() []Permission {
    switch r {
    case RoleViewer:
        return []Permission{
            PermFileRead,
            PermFolderRead,
        }
    case RoleEditor:
        return append(RoleViewer.Permissions(),
            PermFileWrite,
            PermFileRename,
            PermFileMove,
            PermFolderCreate,
            PermFolderRename,
            PermFolderMove,
        )
    case RoleManager:
        return append(RoleEditor.Permissions(),
            PermFileDelete,
            PermFileRestore,
            PermFileShare,
            PermFolderDelete,
            PermFolderShare,
            PermPermissionRead,
            PermPermissionGrant,
            PermPermissionRevoke,
        )
    case RoleOwner:
        return append(RoleManager.Permissions(),
            PermFilePermanentDelete,
        )
    default:
        return nil
    }
}

func (r Role) Includes(other Role) bool {
    hierarchy := map[Role]int{
        RoleViewer:  1,
        RoleEditor:  2,
        RoleManager: 3,
        RoleOwner:   4,
    }
    return hierarchy[r] >= hierarchy[other]
}
```

### RelationType

関係性の種類。

| Relation | 説明 | 例 |
|----------|------|-----|
| owner | 所有者 | user ──owner──▶ file |
| member | メンバー | user ──member──▶ group |
| parent | 親子関係 | folder ──parent──▶ file |
| viewer | 閲覧者ロール | group ──viewer──▶ folder |
| editor | 編集者ロール | user ──editor──▶ folder |
| manager | 管理者ロール | user ──manager──▶ folder |

```go
type RelationType string

const (
    RelationOwner   RelationType = "owner"
    RelationMember  RelationType = "member"
    RelationParent  RelationType = "parent"
    RelationViewer  RelationType = "viewer"
    RelationEditor  RelationType = "editor"
    RelationManager RelationType = "manager"
)

func (r RelationType) ToRole() (Role, bool) {
    switch r {
    case RelationViewer:
        return RoleViewer, true
    case RelationEditor:
        return RoleEditor, true
    case RelationManager:
        return RoleManager, true
    case RelationOwner:
        return RoleOwner, true
    default:
        return "", false
    }
}
```

### ResourceType

| 値 | 説明 |
|-----|------|
| file | ファイル |
| folder | フォルダ |
| group | グループ |

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

```go
type PermissionResolver interface {
    HasPermission(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID, permission Permission) (bool, error)
    CollectPermissions(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID) (PermissionSet, error)
    GetEffectiveRole(ctx context.Context, userID uuid.UUID, resourceType string, resourceID uuid.UUID) (Role, error)
}
```

**権限収集アルゴリズム:**
```go
func (r *PermissionResolverImpl) CollectPermissions(
    ctx context.Context,
    userID uuid.UUID,
    resourceType string,
    resourceID uuid.UUID,
) (PermissionSet, error) {
    permissions := NewPermissionSet()

    // Step 1: 所有者チェック (user ──owner──▶ resource)
    isOwner, err := r.relationshipRepo.Exists(ctx, Tuple{
        SubjectType: "user",
        SubjectID:   userID,
        Relation:    RelationOwner,
        ObjectType:  resourceType,
        ObjectID:    resourceID,
    })
    if err != nil {
        return nil, err
    }
    if isOwner {
        permissions.AddAll(RoleOwner.Permissions())
        return permissions, nil // オーナーは全権限
    }

    // Step 2: 直接付与された権限 (user ──{role}──▶ resource)
    directGrants, err := r.grantRepo.FindByResourceAndGrantee(ctx, resourceType, resourceID, "user", userID)
    if err != nil {
        return nil, err
    }
    for _, grant := range directGrants {
        if grant.Role != "" {
            if role := Role(grant.Role); role.IsValid() {
                permissions.AddAll(role.Permissions())
            }
        }
        if grant.Permission != "" {
            permissions.Add(Permission(grant.Permission))
        }
    }

    // Step 3: グループ経由の権限 (user ──member──▶ group ──{role}──▶ resource)
    groups, err := r.relationshipRepo.FindRelated(ctx, "user", userID, RelationMember, "group")
    if err != nil {
        return nil, err
    }
    for _, groupID := range groups {
        groupGrants, err := r.grantRepo.FindByResourceAndGrantee(ctx, resourceType, resourceID, "group", groupID)
        if err != nil {
            continue
        }
        for _, grant := range groupGrants {
            if grant.Role != "" {
                if role := Role(grant.Role); role.IsValid() {
                    permissions.AddAll(role.Permissions())
                }
            }
            if grant.Permission != "" {
                permissions.Add(Permission(grant.Permission))
            }
        }
    }

    // Step 4: 階層経由の権限 (resource ◀──parent── ancestor)
    if resourceType == "file" || resourceType == "folder" {
        ancestors, err := r.getAncestors(ctx, resourceType, resourceID)
        if err != nil {
            return nil, err
        }
        for _, ancestor := range ancestors {
            // 祖先への直接権限
            ancestorGrants, err := r.grantRepo.FindByResourceAndGrantee(ctx, ancestor.Type, ancestor.ID, "user", userID)
            if err != nil {
                continue
            }
            for _, grant := range ancestorGrants {
                if grant.Role != "" {
                    if role := Role(grant.Role); role.IsValid() {
                        permissions.AddAll(role.Permissions())
                    }
                }
            }

            // 祖先へのグループ経由権限
            for _, groupID := range groups {
                groupGrants, err := r.grantRepo.FindByResourceAndGrantee(ctx, ancestor.Type, ancestor.ID, "group", groupID)
                if err != nil {
                    continue
                }
                for _, grant := range groupGrants {
                    if grant.Role != "" {
                        if role := Role(grant.Role); role.IsValid() {
                            permissions.AddAll(role.Permissions())
                        }
                    }
                }
            }
        }
    }

    return permissions, nil
}

func (r *PermissionResolverImpl) getAncestors(
    ctx context.Context,
    resourceType string,
    resourceID uuid.UUID,
) ([]Resource, error) {
    var ancestors []Resource

    currentType := resourceType
    currentID := resourceID

    for {
        // parent関係を探す
        parents, err := r.relationshipRepo.FindRelatedReverse(ctx, currentType, currentID, RelationParent)
        if err != nil || len(parents) == 0 {
            break
        }

        parent := parents[0]
        ancestors = append(ancestors, parent)
        currentType = parent.Type
        currentID = parent.ID
    }

    return ancestors, nil
}
```

### PermissionGrantService

**責務:** 権限の付与・取り消し

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| GrantRole | cmd | PermissionGrant | ロール付与 |
| GrantPermission | cmd | PermissionGrant | Permission直接付与 |
| Revoke | grantId | void | 権限取り消し |
| RevokeAll | resourceType, resourceId, granteeType, granteeId | void | 全権限取り消し |

```go
type PermissionGrantService interface {
    GrantRole(ctx context.Context, cmd GrantRoleCommand) (*PermissionGrant, error)
    GrantPermission(ctx context.Context, cmd GrantPermissionCommand) (*PermissionGrant, error)
    Revoke(ctx context.Context, grantID uuid.UUID) error
    RevokeAll(ctx context.Context, resourceType string, resourceID uuid.UUID, granteeType string, granteeID uuid.UUID) error
}
```

**ロール付与の処理:**
```go
func (s *PermissionGrantServiceImpl) GrantRole(
    ctx context.Context,
    cmd GrantRoleCommand,
) (*PermissionGrant, error) {
    // 1. 付与者の権限チェック
    hasGrant, err := s.resolver.HasPermission(ctx, cmd.GrantedBy, cmd.ResourceType, cmd.ResourceID, PermPermissionGrant)
    if err != nil {
        return nil, err
    }
    if !hasGrant {
        return nil, errors.New("insufficient permission to grant roles")
    }

    // 2. 付与者のロールチェック（自分より高いロールは付与不可）
    granterRole, err := s.resolver.GetEffectiveRole(ctx, cmd.GrantedBy, cmd.ResourceType, cmd.ResourceID)
    if err != nil {
        return nil, err
    }
    if !granterRole.Includes(cmd.Role) {
        return nil, errors.New("cannot grant a role higher than your own")
    }

    // 3. ownerロールは直接付与不可
    if cmd.Role == RoleOwner {
        return nil, errors.New("owner role cannot be directly granted")
    }

    // 4. 既存の付与チェック
    existing, _ := s.grantRepo.FindByResourceGranteeAndRole(ctx, cmd.ResourceType, cmd.ResourceID, cmd.GranteeType, cmd.GranteeID, cmd.Role)
    if existing != nil {
        return nil, errors.New("role already granted")
    }

    // 5. 権限付与作成
    grant := &PermissionGrant{
        ID:           uuid.New(),
        ResourceType: cmd.ResourceType,
        ResourceID:   cmd.ResourceID,
        GranteeType:  cmd.GranteeType,
        GranteeID:    cmd.GranteeID,
        Role:         cmd.Role,
        GrantedBy:    cmd.GrantedBy,
        GrantedAt:    time.Now(),
    }

    if err := s.grantRepo.Create(ctx, grant); err != nil {
        return nil, err
    }

    // 6. Relationshipタプル作成
    relationshipTuple := &Relationship{
        ID:          uuid.New(),
        SubjectType: cmd.GranteeType,
        SubjectID:   cmd.GranteeID,
        Relation:    RelationType(cmd.Role),
        ObjectType:  cmd.ResourceType,
        ObjectID:    cmd.ResourceID,
        CreatedAt:   time.Now(),
    }
    if err := s.relationshipRepo.Create(ctx, relationshipTuple); err != nil {
        return nil, err
    }

    // 7. イベント発行
    s.eventPublisher.Publish(PermissionGrantedEvent{
        ResourceType: cmd.ResourceType,
        ResourceID:   cmd.ResourceID,
        GranteeType:  cmd.GranteeType,
        GranteeID:    cmd.GranteeID,
        Role:         cmd.Role,
        GrantedBy:    cmd.GrantedBy,
    })

    return grant, nil
}
```

### RelationshipService

**責務:** 関係性の管理（所有権、階層、メンバーシップ）

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| SetOwner | resourceType, resourceId, ownerId | void | 所有者設定 |
| TransferOwnership | resourceType, resourceId, newOwnerId | void | 所有権譲渡 |
| SetParent | childType, childId, parentType, parentId | void | 親子関係設定 |
| AddMember | userId, groupId | void | グループメンバー追加 |
| RemoveMember | userId, groupId | void | グループメンバー削除 |

```go
type RelationshipService interface {
    SetOwner(ctx context.Context, resourceType string, resourceID, ownerID uuid.UUID) error
    TransferOwnership(ctx context.Context, resourceType string, resourceID, currentOwnerID, newOwnerID uuid.UUID) error
    SetParent(ctx context.Context, childType string, childID uuid.UUID, parentType string, parentID uuid.UUID) error
    AddMember(ctx context.Context, userID, groupID uuid.UUID) error
    RemoveMember(ctx context.Context, userID, groupID uuid.UUID) error
}
```

---

## リポジトリ

### PermissionGrantRepository

```go
type PermissionGrantRepository interface {
    Create(ctx context.Context, grant *PermissionGrant) error
    FindByID(ctx context.Context, id uuid.UUID) (*PermissionGrant, error)
    FindByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*PermissionGrant, error)
    FindByResourceAndGrantee(ctx context.Context, resourceType string, resourceID uuid.UUID, granteeType string, granteeID uuid.UUID) ([]*PermissionGrant, error)
    FindByResourceGranteeAndRole(ctx context.Context, resourceType string, resourceID uuid.UUID, granteeType string, granteeID uuid.UUID, role Role) (*PermissionGrant, error)
    FindByGrantee(ctx context.Context, granteeType string, granteeID uuid.UUID) ([]*PermissionGrant, error)
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteByResourceAndGrantee(ctx context.Context, resourceType string, resourceID uuid.UUID, granteeType string, granteeID uuid.UUID) error
}
```

### RelationshipRepository

```go
type RelationshipRepository interface {
    Create(ctx context.Context, rel *Relationship) error
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteByTuple(ctx context.Context, tuple Tuple) error

    Exists(ctx context.Context, tuple Tuple) (bool, error)

    // 主体から対象を探す (user ──member──▶ group の groupを探す)
    FindRelated(ctx context.Context, subjectType string, subjectID uuid.UUID, relation RelationType, objectType string) ([]uuid.UUID, error)

    // 対象から主体を探す (folder ◀──parent── file の folderを探す)
    FindRelatedReverse(ctx context.Context, objectType string, objectID uuid.UUID, relation RelationType) ([]Resource, error)

    // 特定オブジェクトへの全関係性
    FindByObject(ctx context.Context, objectType string, objectID uuid.UUID) ([]*Relationship, error)

    // 特定主体からの全関係性
    FindBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) ([]*Relationship, error)
}
```

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
(group:engineering, owner, folder:team-docs)

【メンバーシップ】
(user:alice, member, group:engineering)
(user:bob, member, group:engineering)

【階層関係】
(folder:team-docs, parent, folder:projects)
(folder:projects, parent, file:spec.pdf)

【権限付与】
(group:engineering, viewer, folder:shared)
(user:charlie, editor, folder:projects)
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
| GrantRole | Manager/Owner | ロール付与 |
| GrantPermission | Manager/Owner | Permission直接付与 |
| RevokePermission | Manager/Owner | 権限取り消し |
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
