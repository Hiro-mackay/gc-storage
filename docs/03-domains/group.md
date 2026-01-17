# Group ドメイン

## 概要

Groupドメインは、ユーザーの集まりであるグループの作成、メンバー管理、ロール管理を担当します。
Collaboration Contextの中核となるドメインで、グループ単位でのファイル共有・権限管理の基盤を提供します。

---

## エンティティ

### Group（集約ルート）

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | グループの一意識別子 |
| name | string | Yes | グループ名 (1-100文字) |
| description | string | No | グループの説明 (最大500文字) |
| owner_id | UUID | Yes | オーナーのユーザーID |
| status | GroupStatus | Yes | グループ状態 |
| created_at | timestamp | Yes | 作成日時 |
| updated_at | timestamp | Yes | 更新日時 |

**ビジネスルール:**
- R-G001: グループには必ず1人のownerが存在する
- R-G002: nameは空文字不可、1-100文字
- R-G003: descriptionは最大500文字
- R-G004: ownerはグループから脱退できない（所有権譲渡が必要）
- R-G005: statusがdeletedのグループは操作不可

**ステータス遷移:**
```
┌─────────┐       ┌─────────┐
│  active │──────▶│ deleted │
└─────────┘       └─────────┘
```

| ステータス | 説明 |
|-----------|------|
| active | アクティブ |
| deleted | 削除済み（論理削除） |

### Membership

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | メンバーシップの一意識別子 |
| group_id | UUID | Yes | グループID |
| user_id | UUID | Yes | ユーザーID |
| role | GroupRole | Yes | グループ内ロール |
| joined_at | timestamp | Yes | 参加日時 |

**ビジネスルール:**
- R-M001: 同一ユーザーは同一グループに1つのMembershipのみ
- R-M002: グループオーナーのMembershipはrole=ownerで固定
- R-M003: ownerロールのMembershipは削除不可（所有権譲渡でのみ変更）

### Invitation

| 属性 | 型 | 必須 | 説明 |
|-----|-----|------|------|
| id | UUID | Yes | 招待の一意識別子 |
| group_id | UUID | Yes | グループID |
| email | Email | Yes | 招待先メールアドレス |
| token | string | Yes | 招待トークン（一意） |
| role | GroupRole | Yes | 付与予定のロール |
| invited_by | UUID | Yes | 招待者のユーザーID |
| expires_at | timestamp | Yes | 有効期限 |
| status | InvitationStatus | Yes | 招待状態 |
| created_at | timestamp | Yes | 作成日時 |

**ビジネスルール:**
- R-I001: tokenは全招待で一意
- R-I002: expires_atを過ぎた招待は自動でexpired
- R-I003: 既にメンバーのユーザーへの招待は不可
- R-I004: 同一グループ・同一メールへの有効な招待は1つのみ
- R-I005: roleにownerは指定不可（所有権譲渡は別フロー）

**ステータス遷移:**
```
┌─────────┐     ┌──────────┐
│ pending │────▶│ accepted │
└────┬────┘     └──────────┘
     │
     ├─────────▶┌──────────┐
     │          │ declined │
     │          └──────────┘
     │
     └─────────▶┌──────────┐
                │ expired  │
                └──────────┘
```

| ステータス | 説明 |
|-----------|------|
| pending | 招待中 |
| accepted | 承諾済み |
| declined | 辞退済み |
| expired | 期限切れ |

---

## 値オブジェクト

### GroupRole

| 値 | 説明 | 権限 |
|-----|------|------|
| member | 一般メンバー | グループ情報閲覧、共有リソースアクセス |
| admin | 管理者 | メンバー招待・削除、グループ設定変更 |
| owner | オーナー | 全権限、グループ削除、オーナー譲渡 |

**ロール階層:**
```
owner > admin > member
```

```go
type GroupRole string

const (
    GroupRoleMember GroupRole = "member"
    GroupRoleAdmin  GroupRole = "admin"
    GroupRoleOwner  GroupRole = "owner"
)

func (r GroupRole) CanInviteMembers() bool {
    return r == GroupRoleAdmin || r == GroupRoleOwner
}

func (r GroupRole) CanRemoveMembers() bool {
    return r == GroupRoleAdmin || r == GroupRoleOwner
}

func (r GroupRole) CanUpdateGroup() bool {
    return r == GroupRoleAdmin || r == GroupRoleOwner
}

func (r GroupRole) CanDeleteGroup() bool {
    return r == GroupRoleOwner
}

func (r GroupRole) CanTransferOwnership() bool {
    return r == GroupRoleOwner
}

func (r GroupRole) CanChangeRole(targetRole GroupRole) bool {
    // ownerロールの付与は不可（所有権譲渡で行う）
    if targetRole == GroupRoleOwner {
        return false
    }
    // adminはadmin/memberロールを付与可能
    if r == GroupRoleAdmin {
        return targetRole == GroupRoleMember || targetRole == GroupRoleAdmin
    }
    // ownerは全ロール（owner除く）を付与可能
    return r == GroupRoleOwner
}
```

### GroupStatus

| 値 | 説明 |
|-----|------|
| active | アクティブ |
| deleted | 削除済み |

### InvitationStatus

| 値 | 説明 |
|-----|------|
| pending | 招待中 |
| accepted | 承諾済み |
| declined | 辞退済み |
| expired | 期限切れ |

---

## ドメインサービス

### GroupMembershipService

**責務:** グループメンバーシップのライフサイクル管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| InviteMember | groupId, email, role, invitedBy | Invitation | メンバー招待 |
| AcceptInvitation | token, userId | Membership | 招待承諾 |
| DeclineInvitation | token, userId | void | 招待辞退 |
| RemoveMember | groupId, userId, removedBy | void | メンバー削除 |
| LeaveGroup | groupId, userId | void | グループ脱退 |
| ChangeRole | groupId, userId, newRole, changedBy | Membership | ロール変更 |

```go
type GroupMembershipService interface {
    InviteMember(ctx context.Context, groupID uuid.UUID, email string, role GroupRole, invitedBy uuid.UUID) (*Invitation, error)
    AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*Membership, error)
    DeclineInvitation(ctx context.Context, token string, userID uuid.UUID) error
    RemoveMember(ctx context.Context, groupID, userID, removedBy uuid.UUID) error
    LeaveGroup(ctx context.Context, groupID, userID uuid.UUID) error
    ChangeRole(ctx context.Context, groupID, userID uuid.UUID, newRole GroupRole, changedBy uuid.UUID) (*Membership, error)
}
```

**InviteMemberのバリデーション:**
```go
func (s *GroupMembershipServiceImpl) InviteMember(
    ctx context.Context,
    groupID uuid.UUID,
    email string,
    role GroupRole,
    invitedBy uuid.UUID,
) (*Invitation, error) {
    // 1. グループ存在確認
    group, err := s.groupRepo.FindByID(ctx, groupID)
    if err != nil {
        return nil, err
    }
    if group.Status != GroupStatusActive {
        return nil, errors.New("group is not active")
    }

    // 2. 招待者の権限確認
    inviter, err := s.membershipRepo.FindByGroupAndUser(ctx, groupID, invitedBy)
    if err != nil {
        return nil, err
    }
    if !inviter.Role.CanInviteMembers() {
        return nil, errors.New("insufficient permission to invite members")
    }

    // 3. ownerロールでの招待は不可
    if role == GroupRoleOwner {
        return nil, errors.New("cannot invite with owner role")
    }

    // 4. 既存メンバーチェック
    user, _ := s.userRepo.FindByEmail(ctx, email)
    if user != nil {
        exists, _ := s.membershipRepo.Exists(ctx, groupID, user.ID)
        if exists {
            return nil, errors.New("user is already a member")
        }
    }

    // 5. 既存の有効な招待チェック
    existingInvite, _ := s.invitationRepo.FindPendingByGroupAndEmail(ctx, groupID, email)
    if existingInvite != nil {
        return nil, errors.New("invitation already exists")
    }

    // 6. 招待作成
    invitation := &Invitation{
        ID:        uuid.New(),
        GroupID:   groupID,
        Email:     email,
        Token:     generateSecureToken(),
        Role:      role,
        InvitedBy: invitedBy,
        ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7日間有効
        Status:    InvitationStatusPending,
        CreatedAt: time.Now(),
    }

    if err := s.invitationRepo.Create(ctx, invitation); err != nil {
        return nil, err
    }

    // 7. 招待イベント発行
    s.eventPublisher.Publish(MemberInvitedEvent{
        GroupID:   groupID,
        Email:     email,
        Role:      role,
        InvitedBy: invitedBy,
    })

    return invitation, nil
}
```

### GroupOwnershipService

**責務:** グループ所有権の管理

| 操作 | 入力 | 出力 | 説明 |
|-----|------|------|------|
| TransferOwnership | groupId, currentOwner, newOwner | void | 所有権譲渡 |

```go
type GroupOwnershipService interface {
    TransferOwnership(ctx context.Context, groupID, currentOwnerID, newOwnerID uuid.UUID) error
}
```

**所有権譲渡のフロー:**
```go
func (s *GroupOwnershipServiceImpl) TransferOwnership(
    ctx context.Context,
    groupID, currentOwnerID, newOwnerID uuid.UUID,
) error {
    return s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. グループ存在確認
        group, err := s.groupRepo.FindByID(ctx, groupID)
        if err != nil {
            return err
        }

        // 2. 現在のオーナー確認
        if group.OwnerID != currentOwnerID {
            return errors.New("not the current owner")
        }

        // 3. 新オーナーがメンバーであることを確認
        newOwnerMembership, err := s.membershipRepo.FindByGroupAndUser(ctx, groupID, newOwnerID)
        if err != nil {
            return errors.New("new owner must be a group member")
        }

        // 4. 新オーナーのロールをownerに変更
        newOwnerMembership.Role = GroupRoleOwner
        if err := s.membershipRepo.Update(ctx, newOwnerMembership); err != nil {
            return err
        }

        // 5. 現オーナーのロールをadminに変更
        currentOwnerMembership, _ := s.membershipRepo.FindByGroupAndUser(ctx, groupID, currentOwnerID)
        currentOwnerMembership.Role = GroupRoleAdmin
        if err := s.membershipRepo.Update(ctx, currentOwnerMembership); err != nil {
            return err
        }

        // 6. グループのowner_idを更新
        group.OwnerID = newOwnerID
        if err := s.groupRepo.Update(ctx, group); err != nil {
            return err
        }

        // 7. イベント発行
        s.eventPublisher.Publish(GroupOwnershipTransferredEvent{
            GroupID:         groupID,
            PreviousOwnerID: currentOwnerID,
            NewOwnerID:      newOwnerID,
        })

        return nil
    })
}
```

---

## リポジトリ

### GroupRepository

```go
type GroupRepository interface {
    Create(ctx context.Context, group *Group) error
    FindByID(ctx context.Context, id uuid.UUID) (*Group, error)
    Update(ctx context.Context, group *Group) error
    Delete(ctx context.Context, id uuid.UUID) error

    // 検索
    FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*Group, error)
    FindByMemberID(ctx context.Context, userID uuid.UUID) ([]*Group, error)
}
```

### MembershipRepository

```go
type MembershipRepository interface {
    Create(ctx context.Context, membership *Membership) error
    FindByID(ctx context.Context, id uuid.UUID) (*Membership, error)
    FindByGroupAndUser(ctx context.Context, groupID, userID uuid.UUID) (*Membership, error)
    FindByGroupID(ctx context.Context, groupID uuid.UUID) ([]*Membership, error)
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]*Membership, error)
    Update(ctx context.Context, membership *Membership) error
    Delete(ctx context.Context, id uuid.UUID) error
    Exists(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
    CountByGroupID(ctx context.Context, groupID uuid.UUID) (int, error)
}
```

### InvitationRepository

```go
type InvitationRepository interface {
    Create(ctx context.Context, invitation *Invitation) error
    FindByID(ctx context.Context, id uuid.UUID) (*Invitation, error)
    FindByToken(ctx context.Context, token string) (*Invitation, error)
    FindPendingByGroupAndEmail(ctx context.Context, groupID uuid.UUID, email string) (*Invitation, error)
    FindPendingByGroupID(ctx context.Context, groupID uuid.UUID) ([]*Invitation, error)
    FindPendingByEmail(ctx context.Context, email string) ([]*Invitation, error)
    Update(ctx context.Context, invitation *Invitation) error
    ExpireOld(ctx context.Context) (int64, error)
}
```

---

## 関係性

### エンティティ関係図

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Group Domain ERD                                    │
└─────────────────────────────────────────────────────────────────────────────┘

      ┌──────────────────┐
      │      users       │ (Identity Context)
      │    (external)    │
      └────────┬─────────┘
               │
               │ owner_id, user_id, invited_by
               │
               ▼
      ┌──────────────────┐
      │     groups       │
      ├──────────────────┤
      │ id               │
      │ name             │
      │ description      │
      │ owner_id (FK)    │──────────────┐
      │ status           │              │
      │ created_at       │              │
      │ updated_at       │              │
      └────────┬─────────┘              │
               │                        │
    ┌──────────┴──────────┐             │
    │                     │             │
    ▼                     ▼             │
┌──────────────┐   ┌──────────────┐     │
│  memberships │   │  invitations │     │
├──────────────┤   ├──────────────┤     │
│ id           │   │ id           │     │
│ group_id(FK) │   │ group_id(FK) │     │
│ user_id (FK) │───│ email        │     │
│ role         │   │ token        │     │
│ joined_at    │   │ role         │     │
└──────────────┘   │ invited_by   │─────┘
                   │ expires_at   │
                   │ status       │
                   │ created_at   │
                   └──────────────┘
```

### 関係性ルール

| 関係 | カーディナリティ | 説明 |
|-----|----------------|------|
| Group - Owner (User) | N:1 | 各グループは1人のオーナーを持つ |
| Group - Membership | 1:N | 1グループは複数のメンバーシップを持つ |
| Group - Invitation | 1:N | 1グループは複数の招待を持てる |
| User - Membership | 1:N | 1ユーザーは複数グループに所属可能 |
| Invitation - Inviter (User) | N:1 | 各招待は1人の招待者を持つ |

---

## 不変条件

1. **オーナー制約**
   - グループには必ず1人のownerが存在する
   - ownerロールのMembershipは削除不可
   - ownerはグループから脱退不可
   - ownerの変更は所有権譲渡によってのみ可能

2. **メンバーシップ制約**
   - 同一ユーザーは同一グループに1つのMembershipのみ
   - メンバーシップ作成時、グループがactiveであること

3. **招待制約**
   - tokenは全招待で一意
   - 同一グループ・同一メールへの有効な招待は1つのみ
   - ownerロールでの招待は不可
   - 既存メンバーへの招待は不可

4. **グループ削除制約**
   - 削除はownerのみ可能
   - 削除時、全メンバーシップと招待も削除
   - グループ所有のリソース（フォルダ・ファイル）の処理が必要

---

## ユースケース概要

| ユースケース | アクター | 概要 |
|------------|--------|------|
| CreateGroup | User | グループ作成（作成者がowner） |
| UpdateGroup | Admin/Owner | グループ名・説明の変更 |
| DeleteGroup | Owner | グループ削除 |
| InviteMember | Admin/Owner | メンバー招待 |
| AcceptInvitation | User | 招待承諾 |
| DeclineInvitation | User | 招待辞退 |
| RemoveMember | Admin/Owner | メンバー削除 |
| LeaveGroup | Member | グループ脱退 |
| ChangeRole | Owner | メンバーロール変更 |
| TransferOwnership | Owner | 所有権譲渡 |
| ListMembers | Member | メンバー一覧表示 |
| ListInvitations | Admin/Owner | 招待一覧表示 |
| CancelInvitation | Admin/Owner | 招待取消 |
| ListMyGroups | User | 所属グループ一覧 |

---

## ドメインイベント

| イベント | トリガー | ペイロード |
|---------|---------|-----------|
| GroupCreated | グループ作成 | groupId, name, ownerId |
| GroupUpdated | グループ更新 | groupId, changedFields |
| GroupDeleted | グループ削除 | groupId, deletedBy |
| MemberInvited | メンバー招待 | groupId, email, role, invitedBy |
| InvitationAccepted | 招待承諾 | invitationId, groupId, userId |
| InvitationDeclined | 招待辞退 | invitationId, groupId, userId |
| InvitationExpired | 招待期限切れ | invitationId, groupId |
| MemberJoined | メンバー参加 | groupId, userId, role |
| MemberLeft | メンバー脱退 | groupId, userId |
| MemberRemoved | メンバー削除 | groupId, userId, removedBy |
| MemberRoleChanged | ロール変更 | groupId, userId, oldRole, newRole, changedBy |
| GroupOwnershipTransferred | 所有権譲渡 | groupId, previousOwnerId, newOwnerId |

---

## 他コンテキストとの連携

### Identity Context（上流）
- UserIDの参照
- ユーザー情報の取得（表示名、メール）

### Authorization Context（下流）
- グループメンバーシップに基づく権限解決
- Relationship Tuple: `(user, member, group)` の作成・削除

### Storage Context（下流）
- グループ作成時にグループルートフォルダを作成
- グループ削除時のリソース処理

---

## 関連ドキュメント

- [イベントストーミング](./EVENT_STORMING.md) - ドメインイベント定義
- [ユーザードメイン](./user.md) - ユーザー管理
- [権限ドメイン](./permission.md) - 権限管理
- [セキュリティ設計](../02-architecture/SECURITY.md) - グループロールの権限定義
