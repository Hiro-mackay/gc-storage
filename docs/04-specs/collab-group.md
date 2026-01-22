# Collaboration Group 詳細設計

## 概要

Collaboration Groupは、グループの作成・管理、メンバーシップ管理、招待フローを担当するモジュールです。
グループ単位でのファイル共有・権限管理の基盤を提供します。

**スコープ:**
- グループ CRUD（作成、取得、更新、削除）
- メンバーシップ管理（参加、脱退、ロール変更）
- 招待フロー（招待、承諾、辞退、取消）
- 所有権譲渡

**参照ドキュメント:**
- [グループドメイン](../03-domains/group.md)
- [セキュリティ設計](../02-architecture/SECURITY.md)
- [Email仕様](./infra-email.md)

---

## 1. パッケージ構成

```
backend/internal/
├── domain/
│   ├── entity/
│   │   ├── group.go           # Groupエンティティ
│   │   ├── membership.go      # Membershipエンティティ
│   │   └── invitation.go      # Invitationエンティティ
│   ├── valueobject/
│   │   ├── group_name.go      # GroupName値オブジェクト
│   │   ├── group_role.go      # GroupRole値オブジェクト
│   │   ├── group_status.go    # GroupStatus値オブジェクト
│   │   └── invitation_status.go # InvitationStatus値オブジェクト
│   └── repository/
│       ├── group_repository.go      # GroupRepositoryインターフェース
│       ├── membership_repository.go # MembershipRepositoryインターフェース
│       └── invitation_repository.go # InvitationRepositoryインターフェース
├── usecase/
│   └── collaboration/
│       ├── command/
│       │   ├── create_group.go
│       │   ├── update_group.go
│       │   ├── delete_group.go
│       │   ├── invite_member.go
│       │   ├── accept_invitation.go
│       │   ├── decline_invitation.go
│       │   ├── cancel_invitation.go
│       │   ├── remove_member.go
│       │   ├── leave_group.go
│       │   ├── change_role.go
│       │   └── transfer_ownership.go
│       └── query/
│           ├── get_group.go
│           ├── list_my_groups.go
│           ├── list_members.go
│           ├── list_invitations.go
│           └── list_pending_invitations.go
├── interface/
│   ├── handler/
│   │   └── group_handler.go
│   └── dto/
│       ├── request/
│       │   └── group.go
│       └── response/
│           └── group.go
└── infrastructure/
    └── repository/
        ├── group_repository.go
        ├── membership_repository.go
        └── invitation_repository.go
```

---

## 2. ユースケース（Command）

### 2.1 グループ作成

```go
// backend/internal/usecase/collaboration/command/create_group.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// CreateGroupInput はグループ作成の入力を定義します
type CreateGroupInput struct {
    Name        string
    Description string
    OwnerID     uuid.UUID
}

// CreateGroupOutput はグループ作成の出力を定義します
type CreateGroupOutput struct {
    Group      *entity.Group
    Membership *entity.Membership
}

// CreateGroupCommand はグループ作成コマンドです
// Note: グループとフォルダは分離されているため、グループ作成時にフォルダは作成しない
type CreateGroupCommand struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
    txManager      repository.TransactionManager
}

// NewCreateGroupCommand は新しいCreateGroupCommandを作成します
func NewCreateGroupCommand(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
    txManager repository.TransactionManager,
) *CreateGroupCommand {
    return &CreateGroupCommand{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        txManager:      txManager,
    }
}

// Execute はグループ作成を実行します
// Note: グループとフォルダは分離されているため、グループ作成時にフォルダは作成しない
// グループはリソース共有の「受け皿」として機能し、フォルダ/ファイルへのロールをPermissionGrantで付与して共有を実現する
func (c *CreateGroupCommand) Execute(ctx context.Context, input CreateGroupInput) (*CreateGroupOutput, error) {
    // 1. グループ名のバリデーション
    groupName, err := valueobject.NewGroupName(input.Name)
    if err != nil {
        return nil, apperror.NewValidationError(err.Error(), nil)
    }

    // 2. グループを作成
    group, err := entity.NewGroup(groupName, input.Description, input.OwnerID)
    if err != nil {
        return nil, apperror.NewValidationError(err.Error(), nil)
    }

    // 3. オーナーメンバーシップを作成
    membership := entity.NewOwnerMembership(group.ID, input.OwnerID)

    // 4. トランザクションで保存
    err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        if err := c.groupRepo.Create(ctx, group); err != nil {
            return err
        }
        if err := c.membershipRepo.Create(ctx, membership); err != nil {
            return err
        }
        return nil
    })

    if err != nil {
        return nil, err
    }

    return &CreateGroupOutput{
        Group:      group,
        Membership: membership,
    }, nil
}
```

### 2.2 メンバー招待

```go
// backend/internal/usecase/collaboration/command/invite_member.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// InviteMemberInput はメンバー招待の入力を定義します
type InviteMemberInput struct {
    GroupID   uuid.UUID
    Email     string
    Role      string
    InviterID uuid.UUID
}

// InviteMemberOutput はメンバー招待の出力を定義します
type InviteMemberOutput struct {
    Invitation *entity.Invitation
}

// InviteMemberCommand はメンバー招待コマンドです
type InviteMemberCommand struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
    invitationRepo repository.InvitationRepository
    userRepo       repository.UserRepository
    emailSender    service.EmailSender
    baseURL        string
}

// NewInviteMemberCommand は新しいInviteMemberCommandを作成します
func NewInviteMemberCommand(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
    invitationRepo repository.InvitationRepository,
    userRepo repository.UserRepository,
    emailSender service.EmailSender,
    baseURL string,
) *InviteMemberCommand {
    return &InviteMemberCommand{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        invitationRepo: invitationRepo,
        userRepo:       userRepo,
        emailSender:    emailSender,
        baseURL:        baseURL,
    }
}

// Execute はメンバー招待を実行します
func (c *InviteMemberCommand) Execute(ctx context.Context, input InviteMemberInput) (*InviteMemberOutput, error) {
    // 1. ロールのバリデーション
    role, err := valueobject.NewGroupRole(input.Role)
    if err != nil {
        return nil, apperror.NewValidationError(err.Error(), nil)
    }

    // 2. ownerロールでの招待は不可（所有権譲渡を使用）
    if role == valueobject.GroupRoleOwner {
        return nil, apperror.NewValidationError("cannot invite with owner role, use ownership transfer", nil)
    }

    // 3. グループの存在確認
    group, err := c.groupRepo.FindByID(ctx, input.GroupID)
    if err != nil {
        return nil, apperror.NewNotFoundError("group not found")
    }
    if !group.IsActive() {
        return nil, apperror.NewValidationError("group is not active", nil)
    }

    // 4. 招待者の権限確認
    inviter, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.InviterID)
    if err != nil {
        return nil, apperror.NewForbiddenError("not a member of this group")
    }
    if !inviter.Role.CanInviteMembers() {
        return nil, apperror.NewForbiddenError("insufficient permission to invite members")
    }

    // 5. 招待者は自分のロール以下のみ付与可能
    if !inviter.Role.CanInviteWithRole(role) {
        return nil, apperror.NewForbiddenError("cannot invite with a role higher than your own")
    }

    // 6. 既存メンバーチェック
    existingUser, _ := c.userRepo.FindByEmail(ctx, input.Email)
    if existingUser != nil {
        exists, _ := c.membershipRepo.Exists(ctx, input.GroupID, existingUser.ID)
        if exists {
            return nil, apperror.NewConflictError("user is already a member")
        }
    }

    // 7. 既存の有効な招待チェック
    existingInvite, _ := c.invitationRepo.FindPendingByGroupAndEmail(ctx, input.GroupID, input.Email)
    if existingInvite != nil {
        return nil, apperror.NewConflictError("invitation already exists for this email")
    }

    // 8. 招待を作成
    invitation, err := entity.NewInvitation(input.GroupID, input.Email, role, input.InviterID)
    if err != nil {
        return nil, apperror.NewValidationError(err.Error(), nil)
    }

    if err := c.invitationRepo.Create(ctx, invitation); err != nil {
        return nil, err
    }

    // 9. 招待メール送信（非同期）
    inviteURL := c.baseURL + "/invite/" + invitation.Token
    go c.emailSender.SendGroupInvitation(context.Background(), service.GroupInvitationData{
        To:        input.Email,
        GroupName: group.Name.String(),
        InviteURL: inviteURL,
        ExpiresAt: invitation.ExpiresAt,
    })

    return &InviteMemberOutput{Invitation: invitation}, nil
}
```

### 2.3 招待承諾

```go
// backend/internal/usecase/collaboration/command/accept_invitation.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// AcceptInvitationInput は招待承諾の入力を定義します
type AcceptInvitationInput struct {
    Token  string
    UserID uuid.UUID
}

// AcceptInvitationOutput は招待承諾の出力を定義します
type AcceptInvitationOutput struct {
    Membership *entity.Membership
    Group      *entity.Group
}

// AcceptInvitationCommand は招待承諾コマンドです
type AcceptInvitationCommand struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
    invitationRepo repository.InvitationRepository
    userRepo       repository.UserRepository
    txManager      repository.TransactionManager
}

// NewAcceptInvitationCommand は新しいAcceptInvitationCommandを作成します
func NewAcceptInvitationCommand(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
    invitationRepo repository.InvitationRepository,
    userRepo repository.UserRepository,
    txManager repository.TransactionManager,
) *AcceptInvitationCommand {
    return &AcceptInvitationCommand{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        invitationRepo: invitationRepo,
        userRepo:       userRepo,
        txManager:      txManager,
    }
}

// Execute は招待承諾を実行します
func (c *AcceptInvitationCommand) Execute(ctx context.Context, input AcceptInvitationInput) (*AcceptInvitationOutput, error) {
    var output *AcceptInvitationOutput

    err := c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. 招待をトークンで検索
        invitation, err := c.invitationRepo.FindByToken(ctx, input.Token)
        if err != nil {
            return apperror.NewNotFoundError("invitation not found")
        }

        // 2. 招待のバリデーション
        if !invitation.CanAccept() {
            if invitation.IsExpired() {
                return apperror.NewValidationError("invitation has expired", nil)
            }
            return apperror.NewValidationError("invitation is no longer valid", nil)
        }

        // 3. ユーザーの取得
        user, err := c.userRepo.FindByID(ctx, input.UserID)
        if err != nil {
            return apperror.NewNotFoundError("user not found")
        }

        // 4. メールアドレスの一致確認
        if user.Email != invitation.Email {
            return apperror.NewForbiddenError("email does not match invitation")
        }

        // 5. グループの取得
        group, err := c.groupRepo.FindByID(ctx, invitation.GroupID)
        if err != nil {
            return apperror.NewNotFoundError("group not found")
        }
        if !group.IsActive() {
            return apperror.NewValidationError("group is no longer active", nil)
        }

        // 6. 既存メンバーチェック
        exists, _ := c.membershipRepo.Exists(ctx, invitation.GroupID, input.UserID)
        if exists {
            return apperror.NewConflictError("already a member of this group")
        }

        // 7. メンバーシップを作成
        membership := entity.NewMembership(invitation.GroupID, input.UserID, invitation.Role)
        if err := c.membershipRepo.Create(ctx, membership); err != nil {
            return err
        }

        // 8. 招待ステータスを更新
        if err := invitation.Accept(); err != nil {
            return err
        }
        if err := c.invitationRepo.Update(ctx, invitation); err != nil {
            return err
        }

        output = &AcceptInvitationOutput{
            Membership: membership,
            Group:      group,
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return output, nil
}
```

### 2.4 メンバー削除

```go
// backend/internal/usecase/collaboration/command/remove_member.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// RemoveMemberInput はメンバー削除の入力を定義します
type RemoveMemberInput struct {
    GroupID  uuid.UUID
    TargetID uuid.UUID // 削除対象のユーザーID
    ActorID  uuid.UUID // 操作実行者のユーザーID
}

// RemoveMemberCommand はメンバー削除コマンドです
type RemoveMemberCommand struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
    txManager      repository.TransactionManager
}

// NewRemoveMemberCommand は新しいRemoveMemberCommandを作成します
func NewRemoveMemberCommand(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
    txManager repository.TransactionManager,
) *RemoveMemberCommand {
    return &RemoveMemberCommand{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        txManager:      txManager,
    }
}

// Execute はメンバー削除を実行します
func (c *RemoveMemberCommand) Execute(ctx context.Context, input RemoveMemberInput) error {
    return c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. グループの取得
        group, err := c.groupRepo.FindByID(ctx, input.GroupID)
        if err != nil {
            return apperror.NewNotFoundError("group not found")
        }
        if !group.IsActive() {
            return apperror.NewValidationError("group is not active", nil)
        }

        // 2. 操作実行者のメンバーシップを取得
        actorMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.ActorID)
        if err != nil {
            return apperror.NewForbiddenError("not a member of this group")
        }

        // 3. 削除対象のメンバーシップを取得
        targetMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.TargetID)
        if err != nil {
            return apperror.NewNotFoundError("target user is not a member")
        }

        // 4. 権限チェック（ownerのみ削除可能）
        if !actorMembership.IsOwner() {
            return apperror.NewForbiddenError("only owner can remove members")
        }

        // 5. オーナーは削除不可
        if targetMembership.IsOwner() {
            return apperror.NewValidationError("cannot remove group owner", nil)
        }

        // 6. メンバーシップを削除
        return c.membershipRepo.Delete(ctx, targetMembership.ID)
    })
}
```

### 2.5 グループ脱退

```go
// backend/internal/usecase/collaboration/command/leave_group.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// LeaveGroupInput はグループ脱退の入力を定義します
type LeaveGroupInput struct {
    GroupID uuid.UUID
    UserID  uuid.UUID
}

// LeaveGroupCommand はグループ脱退コマンドです
type LeaveGroupCommand struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
}

// NewLeaveGroupCommand は新しいLeaveGroupCommandを作成します
func NewLeaveGroupCommand(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
) *LeaveGroupCommand {
    return &LeaveGroupCommand{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
    }
}

// Execute はグループ脱退を実行します
func (c *LeaveGroupCommand) Execute(ctx context.Context, input LeaveGroupInput) error {
    // 1. グループの取得
    group, err := c.groupRepo.FindByID(ctx, input.GroupID)
    if err != nil {
        return apperror.NewNotFoundError("group not found")
    }
    if !group.IsActive() {
        return apperror.NewValidationError("group is not active", nil)
    }

    // 2. メンバーシップの取得
    membership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.UserID)
    if err != nil {
        return apperror.NewNotFoundError("not a member of this group")
    }

    // 3. 脱退可能かチェック
    if !membership.CanLeave() {
        return apperror.NewValidationError("owner cannot leave the group, transfer ownership first", nil)
    }

    // 4. メンバーシップを削除
    return c.membershipRepo.Delete(ctx, membership.ID)
}
```

### 2.6 所有権譲渡

```go
// backend/internal/usecase/collaboration/command/transfer_ownership.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// TransferOwnershipInput は所有権譲渡の入力を定義します
type TransferOwnershipInput struct {
    GroupID      uuid.UUID
    CurrentOwner uuid.UUID
    NewOwner     uuid.UUID
}

// TransferOwnershipOutput は所有権譲渡の出力を定義します
type TransferOwnershipOutput struct {
    Group *entity.Group
}

// TransferOwnershipCommand は所有権譲渡コマンドです
type TransferOwnershipCommand struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
    txManager      repository.TransactionManager
}

// NewTransferOwnershipCommand は新しいTransferOwnershipCommandを作成します
func NewTransferOwnershipCommand(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
    txManager repository.TransactionManager,
) *TransferOwnershipCommand {
    return &TransferOwnershipCommand{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        txManager:      txManager,
    }
}

// Execute は所有権譲渡を実行します
func (c *TransferOwnershipCommand) Execute(ctx context.Context, input TransferOwnershipInput) (*TransferOwnershipOutput, error) {
    var output *TransferOwnershipOutput

    err := c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. グループの取得
        group, err := c.groupRepo.FindByID(ctx, input.GroupID)
        if err != nil {
            return apperror.NewNotFoundError("group not found")
        }
        if !group.IsActive() {
            return apperror.NewValidationError("group is not active", nil)
        }

        // 2. 現在のオーナー確認
        if !group.IsOwnedBy(input.CurrentOwner) {
            return apperror.NewForbiddenError("not the current owner")
        }

        // 3. 自分自身への譲渡は不可
        if input.CurrentOwner == input.NewOwner {
            return apperror.NewValidationError("cannot transfer ownership to yourself", nil)
        }

        // 4. 新オーナーがメンバーであることを確認
        newOwnerMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.NewOwner)
        if err != nil {
            return apperror.NewValidationError("new owner must be a group member", nil)
        }

        // 5. 現オーナーのメンバーシップを取得
        currentOwnerMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.CurrentOwner)
        if err != nil {
            return err
        }

        // 6. 新オーナーをownerに昇格
        newOwnerMembership.PromoteToOwner()
        if err := c.membershipRepo.Update(ctx, newOwnerMembership); err != nil {
            return err
        }

        // 7. 現オーナーをcontributorに降格
        currentOwnerMembership.DemoteToContributor()
        if err := c.membershipRepo.Update(ctx, currentOwnerMembership); err != nil {
            return err
        }

        // 8. グループのowner_idを更新
        group.TransferOwnership(input.NewOwner)
        if err := c.groupRepo.Update(ctx, group); err != nil {
            return err
        }

        output = &TransferOwnershipOutput{Group: group}
        return nil
    })

    if err != nil {
        return nil, err
    }

    return output, nil
}
```

### 2.7 ロール変更

```go
// backend/internal/usecase/collaboration/command/change_role.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ChangeRoleInput はロール変更の入力を定義します
type ChangeRoleInput struct {
    GroupID  uuid.UUID
    TargetID uuid.UUID
    NewRole  string
    ActorID  uuid.UUID
}

// ChangeRoleOutput はロール変更の出力を定義します
type ChangeRoleOutput struct {
    Membership *entity.Membership
}

// ChangeRoleCommand はロール変更コマンドです
type ChangeRoleCommand struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
}

// NewChangeRoleCommand は新しいChangeRoleCommandを作成します
func NewChangeRoleCommand(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
) *ChangeRoleCommand {
    return &ChangeRoleCommand{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
    }
}

// Execute はロール変更を実行します
// GroupRole: viewer < contributor < owner
func (c *ChangeRoleCommand) Execute(ctx context.Context, input ChangeRoleInput) (*ChangeRoleOutput, error) {
    // 1. 新ロールのバリデーション
    newRole, err := valueobject.NewGroupRole(input.NewRole)
    if err != nil {
        return nil, apperror.NewValidationError(err.Error(), nil)
    }

    // 2. ownerロールへの変更は不可（所有権譲渡を使用）
    if newRole == valueobject.GroupRoleOwner {
        return nil, apperror.NewValidationError("use ownership transfer to assign owner role", nil)
    }

    // 3. グループの取得
    group, err := c.groupRepo.FindByID(ctx, input.GroupID)
    if err != nil {
        return nil, apperror.NewNotFoundError("group not found")
    }
    if !group.IsActive() {
        return nil, apperror.NewValidationError("group is not active", nil)
    }

    // 4. 操作実行者のメンバーシップを取得
    actorMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.ActorID)
    if err != nil {
        return nil, apperror.NewForbiddenError("not a member of this group")
    }

    // 5. 対象のメンバーシップを取得
    targetMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.TargetID)
    if err != nil {
        return nil, apperror.NewNotFoundError("target user is not a member")
    }

    // 6. 権限チェック（ownerのみロール変更可能）
    if !actorMembership.IsOwner() {
        return nil, apperror.NewForbiddenError("only owner can change member roles")
    }

    // 7. 自分自身のロールは変更不可
    if input.ActorID == input.TargetID {
        return nil, apperror.NewValidationError("cannot change your own role", nil)
    }

    // 8. ロールを更新
    targetMembership.ChangeRole(newRole)
    if err := c.membershipRepo.Update(ctx, targetMembership); err != nil {
        return nil, err
    }

    return &ChangeRoleOutput{Membership: targetMembership}, nil
}
```

### 2.8 グループ削除

```go
// backend/internal/usecase/collaboration/command/delete_group.go

package command

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// DeleteGroupInput はグループ削除の入力を定義します
type DeleteGroupInput struct {
    GroupID uuid.UUID
    ActorID uuid.UUID
}

// DeleteGroupCommand はグループ削除コマンドです
// Note: グループとフォルダは分離されているため、グループ削除時にフォルダは削除しない
// グループに付与されていたPermissionGrantは別途削除される
type DeleteGroupCommand struct {
    groupRepo       repository.GroupRepository
    membershipRepo  repository.MembershipRepository
    invitationRepo  repository.InvitationRepository
    permGrantRepo   repository.PermissionGrantRepository
    txManager       repository.TransactionManager
}

// NewDeleteGroupCommand は新しいDeleteGroupCommandを作成します
func NewDeleteGroupCommand(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
    invitationRepo repository.InvitationRepository,
    permGrantRepo repository.PermissionGrantRepository,
    txManager repository.TransactionManager,
) *DeleteGroupCommand {
    return &DeleteGroupCommand{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        invitationRepo: invitationRepo,
        permGrantRepo:  permGrantRepo,
        txManager:      txManager,
    }
}

// Execute はグループ削除を実行します
func (c *DeleteGroupCommand) Execute(ctx context.Context, input DeleteGroupInput) error {
    return c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. グループの取得
        group, err := c.groupRepo.FindByID(ctx, input.GroupID)
        if err != nil {
            return apperror.NewNotFoundError("group not found")
        }
        if !group.IsActive() {
            return apperror.NewValidationError("group is already deleted", nil)
        }

        // 2. オーナーのみ削除可能
        if !group.IsOwnedBy(input.ActorID) {
            return apperror.NewForbiddenError("only the owner can delete the group")
        }

        // 3. 招待を全削除
        if err := c.invitationRepo.DeleteByGroupID(ctx, input.GroupID); err != nil {
            return err
        }

        // 4. メンバーシップを全削除
        if err := c.membershipRepo.DeleteByGroupID(ctx, input.GroupID); err != nil {
            return err
        }

        // 5. グループに付与されていたPermissionGrantを全削除
        // Note: グループとフォルダは分離されているため、グループ削除時にフォルダは削除しない
        if err := c.permGrantRepo.DeleteByGrantee(ctx, authz.GranteeTypeGroup, input.GroupID); err != nil {
            return err
        }

        // 6. グループを論理削除
        group.Delete()
        return c.groupRepo.Update(ctx, group)
    })
}
```

---

## 3. ユースケース（Query）

### 3.1 グループ詳細取得

```go
// backend/internal/usecase/collaboration/query/get_group.go

package query

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetGroupInput はグループ詳細取得の入力を定義します
type GetGroupInput struct {
    GroupID uuid.UUID
    ActorID uuid.UUID
}

// GetGroupOutput はグループ詳細取得の出力を定義します
type GetGroupOutput struct {
    Group       *entity.Group
    Membership  *entity.Membership
    MemberCount int
}

// GetGroupQuery はグループ詳細取得クエリです
type GetGroupQuery struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
}

// NewGetGroupQuery は新しいGetGroupQueryを作成します
func NewGetGroupQuery(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
) *GetGroupQuery {
    return &GetGroupQuery{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
    }
}

// Execute はグループ詳細取得を実行します
func (q *GetGroupQuery) Execute(ctx context.Context, input GetGroupInput) (*GetGroupOutput, error) {
    // 1. グループの取得
    group, err := q.groupRepo.FindByID(ctx, input.GroupID)
    if err != nil {
        return nil, apperror.NewNotFoundError("group not found")
    }

    // 2. メンバーシップの確認（閲覧権限チェック）
    membership, err := q.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.ActorID)
    if err != nil {
        return nil, apperror.NewForbiddenError("not a member of this group")
    }

    // 3. メンバー数の取得
    memberCount, err := q.membershipRepo.CountByGroupID(ctx, input.GroupID)
    if err != nil {
        return nil, err
    }

    return &GetGroupOutput{
        Group:       group,
        Membership:  membership,
        MemberCount: memberCount,
    }, nil
}
```

### 3.2 所属グループ一覧

```go
// backend/internal/usecase/collaboration/query/list_my_groups.go

package query

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// ListMyGroupsInput は所属グループ一覧の入力を定義します
type ListMyGroupsInput struct {
    UserID uuid.UUID
}

// GroupWithMembership はメンバーシップ付きグループ
type GroupWithMembership struct {
    Group      *entity.Group
    Membership *entity.Membership
}

// ListMyGroupsOutput は所属グループ一覧の出力を定義します
type ListMyGroupsOutput struct {
    Groups []*GroupWithMembership
}

// ListMyGroupsQuery は所属グループ一覧クエリです
type ListMyGroupsQuery struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
}

// NewListMyGroupsQuery は新しいListMyGroupsQueryを作成します
func NewListMyGroupsQuery(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
) *ListMyGroupsQuery {
    return &ListMyGroupsQuery{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
    }
}

// Execute は所属グループ一覧を実行します
func (q *ListMyGroupsQuery) Execute(ctx context.Context, input ListMyGroupsInput) (*ListMyGroupsOutput, error) {
    // 1. ユーザーのメンバーシップ一覧を取得
    memberships, err := q.membershipRepo.FindByUserID(ctx, input.UserID)
    if err != nil {
        return nil, err
    }

    // 2. 各メンバーシップのグループを取得
    groups := make([]*GroupWithMembership, 0, len(memberships))
    for _, m := range memberships {
        group, err := q.groupRepo.FindByID(ctx, m.GroupID)
        if err != nil {
            continue // グループが見つからない場合はスキップ
        }
        if !group.IsActive() {
            continue // 削除済みグループはスキップ
        }
        groups = append(groups, &GroupWithMembership{
            Group:      group,
            Membership: m,
        })
    }

    return &ListMyGroupsOutput{Groups: groups}, nil
}
```

### 3.3 メンバー一覧

```go
// backend/internal/usecase/collaboration/query/list_members.go

package query

import (
    "context"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ListMembersInput はメンバー一覧の入力を定義します
type ListMembersInput struct {
    GroupID uuid.UUID
    ActorID uuid.UUID
}

// ListMembersOutput はメンバー一覧の出力を定義します
type ListMembersOutput struct {
    Members []*repository.MembershipWithUser
}

// ListMembersQuery はメンバー一覧クエリです
type ListMembersQuery struct {
    groupRepo      repository.GroupRepository
    membershipRepo repository.MembershipRepository
}

// NewListMembersQuery は新しいListMembersQueryを作成します
func NewListMembersQuery(
    groupRepo repository.GroupRepository,
    membershipRepo repository.MembershipRepository,
) *ListMembersQuery {
    return &ListMembersQuery{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
    }
}

// Execute はメンバー一覧を実行します
func (q *ListMembersQuery) Execute(ctx context.Context, input ListMembersInput) (*ListMembersOutput, error) {
    // 1. グループの存在確認
    group, err := q.groupRepo.FindByID(ctx, input.GroupID)
    if err != nil {
        return nil, apperror.NewNotFoundError("group not found")
    }
    if !group.IsActive() {
        return nil, apperror.NewValidationError("group is not active", nil)
    }

    // 2. 操作実行者のメンバーシップ確認
    _, err = q.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.ActorID)
    if err != nil {
        return nil, apperror.NewForbiddenError("not a member of this group")
    }

    // 3. メンバー一覧（ユーザー情報付き）を取得
    members, err := q.membershipRepo.FindByGroupIDWithUsers(ctx, input.GroupID)
    if err != nil {
        return nil, err
    }

    return &ListMembersOutput{Members: members}, nil
}
```

---

## 4. ハンドラー

```go
// backend/internal/interface/handler/group_handler.go

package handler

import (
    "net/http"

    "github.com/google/uuid"
    "github.com/labstack/echo/v4"

    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
    "github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
    "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/command"
    "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/query"
)

// GroupHandler はグループ関連のHTTPハンドラーです
type GroupHandler struct {
    createGroup       *command.CreateGroupCommand
    updateGroup       *command.UpdateGroupCommand
    deleteGroup       *command.DeleteGroupCommand
    inviteMember      *command.InviteMemberCommand
    acceptInvitation  *command.AcceptInvitationCommand
    declineInvitation *command.DeclineInvitationCommand
    cancelInvitation  *command.CancelInvitationCommand
    removeMember      *command.RemoveMemberCommand
    leaveGroup        *command.LeaveGroupCommand
    changeRole        *command.ChangeRoleCommand
    transferOwnership *command.TransferOwnershipCommand
    getGroup          *query.GetGroupQuery
    listMyGroups      *query.ListMyGroupsQuery
    listMembers       *query.ListMembersQuery
    listInvitations   *query.ListInvitationsQuery
}

// NewGroupHandler は新しいGroupHandlerを作成します
func NewGroupHandler(
    createGroup *command.CreateGroupCommand,
    updateGroup *command.UpdateGroupCommand,
    deleteGroup *command.DeleteGroupCommand,
    inviteMember *command.InviteMemberCommand,
    acceptInvitation *command.AcceptInvitationCommand,
    declineInvitation *command.DeclineInvitationCommand,
    cancelInvitation *command.CancelInvitationCommand,
    removeMember *command.RemoveMemberCommand,
    leaveGroup *command.LeaveGroupCommand,
    changeRole *command.ChangeRoleCommand,
    transferOwnership *command.TransferOwnershipCommand,
    getGroup *query.GetGroupQuery,
    listMyGroups *query.ListMyGroupsQuery,
    listMembers *query.ListMembersQuery,
    listInvitations *query.ListInvitationsQuery,
) *GroupHandler {
    return &GroupHandler{
        createGroup:       createGroup,
        updateGroup:       updateGroup,
        deleteGroup:       deleteGroup,
        inviteMember:      inviteMember,
        acceptInvitation:  acceptInvitation,
        declineInvitation: declineInvitation,
        cancelInvitation:  cancelInvitation,
        removeMember:      removeMember,
        leaveGroup:        leaveGroup,
        changeRole:        changeRole,
        transferOwnership: transferOwnership,
        getGroup:          getGroup,
        listMyGroups:      listMyGroups,
        listMembers:       listMembers,
        listInvitations:   listInvitations,
    }
}

// Create handles POST /api/v1/groups
func (h *GroupHandler) Create(c echo.Context) error {
    var req request.CreateGroupRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.createGroup.Execute(c.Request().Context(), command.CreateGroupInput{
        Name:        req.Name,
        Description: req.Description,
        OwnerID:     claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, response.ToGroupResponse(output.Group, output.Membership))
}

// Get handles GET /api/v1/groups/:id
func (h *GroupHandler) Get(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.getGroup.Execute(c.Request().Context(), query.GetGroupInput{
        GroupID: groupID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, response.ToGroupDetailResponse(output.Group, output.Membership, output.MemberCount))
}

// ListMyGroups handles GET /api/v1/groups
func (h *GroupHandler) ListMyGroups(c echo.Context) error {
    claims := middleware.GetClaims(c)
    output, err := h.listMyGroups.Execute(c.Request().Context(), query.ListMyGroupsInput{
        UserID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, response.ToGroupListResponse(output.Groups))
}

// Delete handles DELETE /api/v1/groups/:id
func (h *GroupHandler) Delete(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    claims := middleware.GetClaims(c)
    err = h.deleteGroup.Execute(c.Request().Context(), command.DeleteGroupInput{
        GroupID: groupID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// InviteMember handles POST /api/v1/groups/:id/invitations
func (h *GroupHandler) InviteMember(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    var req request.InviteMemberRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.inviteMember.Execute(c.Request().Context(), command.InviteMemberInput{
        GroupID:   groupID,
        Email:     req.Email,
        Role:      req.Role,
        InviterID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, response.ToInvitationResponse(output.Invitation))
}

// AcceptInvitation handles POST /api/v1/invitations/:token/accept
func (h *GroupHandler) AcceptInvitation(c echo.Context) error {
    token := c.Param("token")

    claims := middleware.GetClaims(c)
    output, err := h.acceptInvitation.Execute(c.Request().Context(), command.AcceptInvitationInput{
        Token:  token,
        UserID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, response.ToAcceptInvitationResponse(output.Group, output.Membership))
}

// DeclineInvitation handles POST /api/v1/invitations/:token/decline
func (h *GroupHandler) DeclineInvitation(c echo.Context) error {
    token := c.Param("token")

    claims := middleware.GetClaims(c)
    err := h.declineInvitation.Execute(c.Request().Context(), command.DeclineInvitationInput{
        Token:  token,
        UserID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// ListMembers handles GET /api/v1/groups/:id/members
func (h *GroupHandler) ListMembers(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.listMembers.Execute(c.Request().Context(), query.ListMembersInput{
        GroupID: groupID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, response.ToMemberListResponse(output.Members))
}

// RemoveMember handles DELETE /api/v1/groups/:id/members/:userId
func (h *GroupHandler) RemoveMember(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }
    targetID, err := uuid.Parse(c.Param("userId"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
    }

    claims := middleware.GetClaims(c)
    err = h.removeMember.Execute(c.Request().Context(), command.RemoveMemberInput{
        GroupID:  groupID,
        TargetID: targetID,
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// Leave handles POST /api/v1/groups/:id/leave
func (h *GroupHandler) Leave(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    claims := middleware.GetClaims(c)
    err = h.leaveGroup.Execute(c.Request().Context(), command.LeaveGroupInput{
        GroupID: groupID,
        UserID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// ChangeRole handles PATCH /api/v1/groups/:id/members/:userId/role
func (h *GroupHandler) ChangeRole(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }
    targetID, err := uuid.Parse(c.Param("userId"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
    }

    var req request.ChangeRoleRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.changeRole.Execute(c.Request().Context(), command.ChangeRoleInput{
        GroupID:  groupID,
        TargetID: targetID,
        NewRole:  req.Role,
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, response.ToMemberResponse(output.Membership, "", ""))
}

// TransferOwnership handles POST /api/v1/groups/:id/transfer
func (h *GroupHandler) TransferOwnership(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    var req request.TransferOwnershipRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.transferOwnership.Execute(c.Request().Context(), command.TransferOwnershipInput{
        GroupID:      groupID,
        CurrentOwner: claims.UserID,
        NewOwner:     req.NewOwnerID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, response.ToGroupSimpleResponse(output.Group))
}
```

---

## 5. DTO定義

### 5.1 Request DTO

```go
// backend/internal/interface/dto/request/group.go

package request

import "github.com/google/uuid"

// CreateGroupRequest はグループ作成リクエスト
type CreateGroupRequest struct {
    Name        string `json:"name" validate:"required,min=1,max=100"`
    Description string `json:"description" validate:"max=500"`
}

// UpdateGroupRequest はグループ更新リクエスト
type UpdateGroupRequest struct {
    Name        *string `json:"name" validate:"omitempty,min=1,max=100"`
    Description *string `json:"description" validate:"omitempty,max=500"`
}

// InviteMemberRequest はメンバー招待リクエスト
// GroupRole: viewer, contributor（ownerは所有権譲渡で付与）
// デフォルトロール: viewer
type InviteMemberRequest struct {
    Email string `json:"email" validate:"required,email"`
    Role  string `json:"role" validate:"omitempty,oneof=viewer contributor"`  // 省略時はviewer
}

// ChangeRoleRequest はロール変更リクエスト
// GroupRole: viewer, contributor（ownerは所有権譲渡で付与）
type ChangeRoleRequest struct {
    Role string `json:"role" validate:"required,oneof=viewer contributor"`
}

// TransferOwnershipRequest は所有権譲渡リクエスト
type TransferOwnershipRequest struct {
    NewOwnerID uuid.UUID `json:"newOwnerId" validate:"required"`
}
```

### 5.2 Response DTO

```go
// backend/internal/interface/dto/response/group.go

package response

import (
    "time"

    "github.com/google/uuid"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
    "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/query"
)

// GroupResponse はグループレスポンス
type GroupResponse struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    OwnerID     uuid.UUID `json:"ownerId"`
    Role        string    `json:"role"`
    CreatedAt   time.Time `json:"createdAt"`
}

// GroupDetailResponse はグループ詳細レスポンス
type GroupDetailResponse struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    OwnerID     uuid.UUID `json:"ownerId"`
    Role        string    `json:"role"`
    MemberCount int       `json:"memberCount"`
    CreatedAt   time.Time `json:"createdAt"`
}

// GroupSimpleResponse はシンプルなグループレスポンス
type GroupSimpleResponse struct {
    ID      uuid.UUID `json:"id"`
    Name    string    `json:"name"`
    OwnerID uuid.UUID `json:"ownerId"`
}

// GroupListResponse はグループ一覧レスポンス
type GroupListResponse struct {
    Groups []GroupResponse `json:"groups"`
}

// InvitationResponse は招待レスポンス
type InvitationResponse struct {
    ID        uuid.UUID `json:"id"`
    Email     string    `json:"email"`
    Role      string    `json:"role"`
    ExpiresAt time.Time `json:"expiresAt"`
    Status    string    `json:"status"`
}

// InvitationListResponse は招待一覧レスポンス
type InvitationListResponse struct {
    Invitations []InvitationResponse `json:"invitations"`
}

// AcceptInvitationResponse は招待承諾レスポンス
type AcceptInvitationResponse struct {
    GroupID   uuid.UUID `json:"groupId"`
    GroupName string    `json:"groupName"`
    Role      string    `json:"role"`
}

// MemberResponse はメンバーレスポンス
type MemberResponse struct {
    UserID   uuid.UUID `json:"userId"`
    UserName string    `json:"userName"`
    Email    string    `json:"email"`
    Role     string    `json:"role"`
    JoinedAt time.Time `json:"joinedAt"`
}

// MemberListResponse はメンバー一覧レスポンス
type MemberListResponse struct {
    Members []MemberResponse `json:"members"`
}

// ToGroupResponse はGroupResponseを生成します
func ToGroupResponse(group *entity.Group, membership *entity.Membership) GroupResponse {
    return GroupResponse{
        ID:          group.ID,
        Name:        group.Name.String(),
        Description: group.Description,
        OwnerID:     group.OwnerID,
        Role:        membership.Role.String(),
        CreatedAt:   group.CreatedAt,
    }
}

// ToGroupDetailResponse はGroupDetailResponseを生成します
func ToGroupDetailResponse(group *entity.Group, membership *entity.Membership, memberCount int) GroupDetailResponse {
    return GroupDetailResponse{
        ID:          group.ID,
        Name:        group.Name.String(),
        Description: group.Description,
        OwnerID:     group.OwnerID,
        Role:        membership.Role.String(),
        MemberCount: memberCount,
        CreatedAt:   group.CreatedAt,
    }
}

// ToGroupSimpleResponse はGroupSimpleResponseを生成します
func ToGroupSimpleResponse(group *entity.Group) GroupSimpleResponse {
    return GroupSimpleResponse{
        ID:      group.ID,
        Name:    group.Name.String(),
        OwnerID: group.OwnerID,
    }
}

// ToGroupListResponse はGroupListResponseを生成します
func ToGroupListResponse(groups []*query.GroupWithMembership) GroupListResponse {
    responses := make([]GroupResponse, 0, len(groups))
    for _, g := range groups {
        responses = append(responses, ToGroupResponse(g.Group, g.Membership))
    }
    return GroupListResponse{Groups: responses}
}

// ToInvitationResponse はInvitationResponseを生成します
func ToInvitationResponse(invitation *entity.Invitation) InvitationResponse {
    return InvitationResponse{
        ID:        invitation.ID,
        Email:     invitation.Email,
        Role:      invitation.Role.String(),
        ExpiresAt: invitation.ExpiresAt,
        Status:    invitation.Status.String(),
    }
}

// ToAcceptInvitationResponse はAcceptInvitationResponseを生成します
func ToAcceptInvitationResponse(group *entity.Group, membership *entity.Membership) AcceptInvitationResponse {
    return AcceptInvitationResponse{
        GroupID:   group.ID,
        GroupName: group.Name.String(),
        Role:      membership.Role.String(),
    }
}

// ToMemberResponse はMemberResponseを生成します
func ToMemberResponse(membership *entity.Membership, userName, email string) MemberResponse {
    return MemberResponse{
        UserID:   membership.UserID,
        UserName: userName,
        Email:    email,
        Role:     membership.Role.String(),
        JoinedAt: membership.JoinedAt,
    }
}

// ToMemberListResponse はMemberListResponseを生成します
func ToMemberListResponse(members []*repository.MembershipWithUser) MemberListResponse {
    responses := make([]MemberResponse, 0, len(members))
    for _, m := range members {
        responses = append(responses, MemberResponse{
            UserID:   m.UserID,
            UserName: m.UserName,
            Email:    m.UserEmail,
            Role:     m.Role.String(),
            JoinedAt: m.JoinedAt,
        })
    }
    return MemberListResponse{Members: responses}
}
```

---

## 6. APIエンドポイント

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | /api/v1/groups | GroupHandler.Create | グループ作成 |
| GET | /api/v1/groups | GroupHandler.ListMyGroups | 所属グループ一覧 |
| GET | /api/v1/groups/:id | GroupHandler.Get | グループ詳細 |
| PATCH | /api/v1/groups/:id | GroupHandler.Update | グループ更新 |
| DELETE | /api/v1/groups/:id | GroupHandler.Delete | グループ削除 |
| POST | /api/v1/groups/:id/invitations | GroupHandler.InviteMember | メンバー招待 |
| GET | /api/v1/groups/:id/invitations | GroupHandler.ListInvitations | 招待一覧 |
| DELETE | /api/v1/groups/:id/invitations/:invitationId | GroupHandler.CancelInvitation | 招待取消 |
| GET | /api/v1/groups/:id/members | GroupHandler.ListMembers | メンバー一覧 |
| DELETE | /api/v1/groups/:id/members/:userId | GroupHandler.RemoveMember | メンバー削除 |
| PATCH | /api/v1/groups/:id/members/:userId/role | GroupHandler.ChangeRole | ロール変更 |
| POST | /api/v1/groups/:id/leave | GroupHandler.Leave | グループ脱退 |
| POST | /api/v1/groups/:id/transfer | GroupHandler.TransferOwnership | 所有権譲渡 |
| POST | /api/v1/invitations/:token/accept | GroupHandler.AcceptInvitation | 招待承諾 |
| POST | /api/v1/invitations/:token/decline | GroupHandler.DeclineInvitation | 招待辞退 |
| GET | /api/v1/invitations/pending | GroupHandler.ListPendingInvitations | 自分への招待一覧 |

---

## 7. バックグラウンドジョブ

### 7.1 招待期限切れ処理

```go
// backend/internal/job/invitation_expiry.go

package job

import (
    "context"
    "log/slog"

    "github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// InvitationExpiryJob は招待期限切れ処理ジョブです
type InvitationExpiryJob struct {
    invitationRepo repository.InvitationRepository
    logger         *slog.Logger
}

// NewInvitationExpiryJob は新しいInvitationExpiryJobを作成します
func NewInvitationExpiryJob(
    invitationRepo repository.InvitationRepository,
    logger *slog.Logger,
) *InvitationExpiryJob {
    return &InvitationExpiryJob{
        invitationRepo: invitationRepo,
        logger:         logger,
    }
}

// Run は招待期限切れ処理を実行します（毎時実行）
func (j *InvitationExpiryJob) Run(ctx context.Context) error {
    expired, err := j.invitationRepo.ExpireOld(ctx)
    if err != nil {
        j.logger.Error("invitation expiry job failed", "error", err)
        return err
    }

    if expired > 0 {
        j.logger.Info("expired invitations", "count", expired)
    }
    return nil
}
```

---

## 8. 受け入れ基準

### グループ管理
- [ ] グループを作成できる（作成者がオーナーになる）
- [ ] グループ作成時にフォルダは作成されない（グループとフォルダは分離）
- [ ] グループ名・説明を更新できる（contributor/owner）
- [ ] グループを削除できる（ownerのみ）
- [ ] 削除時にメンバーシップ・招待・PermissionGrantが削除される
- [ ] 所属グループ一覧を取得できる

### メンバー招待
- [ ] contributor/ownerがメンバーを招待できる
- [ ] 招待メールが送信される
- [ ] デフォルト招待ロールはviewerになる
- [ ] 招待者は自分のロール以下のみ付与可能
  - Owner → Viewer, Contributor
  - Contributor → Viewer, Contributor
- [ ] 招待リンクから承諾できる
- [ ] 招待を辞退できる
- [ ] 招待を取消できる（contributor/owner）
- [ ] 同じメールへの重複招待は拒否される
- [ ] 既存メンバーへの招待は拒否される
- [ ] ownerロールでの招待は拒否される（所有権譲渡を使用）
- [ ] 7日後に招待が期限切れになる

### メンバー管理
- [ ] メンバー一覧を取得できる
- [ ] ownerがメンバーを削除できる
- [ ] メンバーが自発的に脱退できる
- [ ] ownerは脱退できない（所有権譲渡が必要）
- [ ] ownerがメンバーのロールを変更できる
- [ ] 自分のロールは変更できない

### 所有権譲渡
- [ ] ownerが他のメンバーに所有権を譲渡できる
- [ ] 譲渡後、旧ownerはcontributorになる
- [ ] 非メンバーへの譲渡は拒否される

### 権限検証（GroupRole: viewer < contributor < owner）
- [ ] viewer: グループ情報閲覧、共有リソースアクセス
- [ ] contributor: 招待（Contributor以下のロール）、メンバー削除
- [ ] owner: 全権限、グループ削除、設定変更、オーナー譲渡

---

## 関連ドキュメント

- [グループドメイン](../03-domains/group.md)
- [セキュリティ設計](../02-architecture/SECURITY.md)
- [Email仕様](./infra-email.md)
- [Storage Core仕様](./storage-core.md)
