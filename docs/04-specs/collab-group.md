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

## 1. エンティティ定義

### 1.1 Group

```go
// internal/domain/group/group.go

package group

import (
    "time"
    "github.com/google/uuid"
)

type GroupStatus string

const (
    GroupStatusActive  GroupStatus = "active"
    GroupStatusDeleted GroupStatus = "deleted"
)

type Group struct {
    ID          uuid.UUID
    Name        string
    Description string
    OwnerID     uuid.UUID
    Status      GroupStatus
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// IsActive returns true if group is active
func (g *Group) IsActive() bool {
    return g.Status == GroupStatusActive
}

// CanUpdate returns true if group can be updated
func (g *Group) CanUpdate(userID uuid.UUID, role GroupRole) bool {
    return g.IsActive() && role.CanUpdateGroup()
}

// CanDelete returns true if group can be deleted
func (g *Group) CanDelete(userID uuid.UUID) bool {
    return g.IsActive() && g.OwnerID == userID
}
```

### 1.2 GroupRole

```go
// internal/domain/group/role.go

package group

type GroupRole string

const (
    GroupRoleMember GroupRole = "member"
    GroupRoleAdmin  GroupRole = "admin"
    GroupRoleOwner  GroupRole = "owner"
)

// CanInviteMembers returns true if role can invite members
func (r GroupRole) CanInviteMembers() bool {
    return r == GroupRoleAdmin || r == GroupRoleOwner
}

// CanRemoveMembers returns true if role can remove members
func (r GroupRole) CanRemoveMembers() bool {
    return r == GroupRoleAdmin || r == GroupRoleOwner
}

// CanUpdateGroup returns true if role can update group settings
func (r GroupRole) CanUpdateGroup() bool {
    return r == GroupRoleAdmin || r == GroupRoleOwner
}

// CanDeleteGroup returns true if role can delete group
func (r GroupRole) CanDeleteGroup() bool {
    return r == GroupRoleOwner
}

// CanTransferOwnership returns true if role can transfer ownership
func (r GroupRole) CanTransferOwnership() bool {
    return r == GroupRoleOwner
}

// CanChangeRole returns true if role can change target to newRole
func (r GroupRole) CanChangeRole(targetRole, newRole GroupRole) bool {
    // Cannot grant owner role (use ownership transfer)
    if newRole == GroupRoleOwner {
        return false
    }
    // Cannot change owner's role
    if targetRole == GroupRoleOwner {
        return false
    }
    // Admin can manage member/admin roles
    if r == GroupRoleAdmin {
        return newRole == GroupRoleMember || newRole == GroupRoleAdmin
    }
    // Owner can manage all roles except owner
    return r == GroupRoleOwner
}

// CanRemove returns true if role can remove target role
func (r GroupRole) CanRemove(targetRole GroupRole) bool {
    // Cannot remove owner
    if targetRole == GroupRoleOwner {
        return false
    }
    // Admin can remove members (not other admins)
    if r == GroupRoleAdmin {
        return targetRole == GroupRoleMember
    }
    // Owner can remove anyone except owner
    return r == GroupRoleOwner
}

// IsHigherThan returns true if r has higher privilege than other
func (r GroupRole) IsHigherThan(other GroupRole) bool {
    roleOrder := map[GroupRole]int{
        GroupRoleMember: 1,
        GroupRoleAdmin:  2,
        GroupRoleOwner:  3,
    }
    return roleOrder[r] > roleOrder[other]
}
```

### 1.3 Membership

```go
// internal/domain/group/membership.go

package group

import (
    "time"
    "github.com/google/uuid"
)

type Membership struct {
    ID       uuid.UUID
    GroupID  uuid.UUID
    UserID   uuid.UUID
    Role     GroupRole
    JoinedAt time.Time
}

// IsOwner returns true if this membership is for the group owner
func (m *Membership) IsOwner() bool {
    return m.Role == GroupRoleOwner
}

// CanLeave returns true if member can leave the group
func (m *Membership) CanLeave() bool {
    // Owner cannot leave (must transfer ownership first)
    return m.Role != GroupRoleOwner
}
```

### 1.4 Invitation

```go
// internal/domain/group/invitation.go

package group

import (
    "crypto/rand"
    "encoding/hex"
    "time"
    "github.com/google/uuid"
)

type InvitationStatus string

const (
    InvitationStatusPending  InvitationStatus = "pending"
    InvitationStatusAccepted InvitationStatus = "accepted"
    InvitationStatusDeclined InvitationStatus = "declined"
    InvitationStatusExpired  InvitationStatus = "expired"
)

const (
    InvitationTokenLength = 32
    InvitationExpiry      = 7 * 24 * time.Hour // 7 days
)

type Invitation struct {
    ID        uuid.UUID
    GroupID   uuid.UUID
    Email     string
    Token     string
    Role      GroupRole
    InvitedBy uuid.UUID
    ExpiresAt time.Time
    Status    InvitationStatus
    CreatedAt time.Time
}

// IsValid returns true if invitation can be used
func (i *Invitation) IsValid() bool {
    return i.Status == InvitationStatusPending && !i.IsExpired()
}

// IsExpired returns true if invitation has expired
func (i *Invitation) IsExpired() bool {
    return time.Now().After(i.ExpiresAt)
}

// CanAccept returns true if invitation can be accepted
func (i *Invitation) CanAccept() bool {
    return i.IsValid()
}

// CanDecline returns true if invitation can be declined
func (i *Invitation) CanDecline() bool {
    return i.Status == InvitationStatusPending
}

// CanCancel returns true if invitation can be cancelled
func (i *Invitation) CanCancel() bool {
    return i.Status == InvitationStatusPending
}

// GenerateToken generates a secure random token
func GenerateInvitationToken() (string, error) {
    bytes := make([]byte, InvitationTokenLength)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}
```

---

## 2. リポジトリインターフェース

### 2.1 GroupRepository

```go
// internal/domain/group/group_repository.go

package group

import (
    "context"
    "github.com/google/uuid"
)

type GroupRepository interface {
    // CRUD
    Create(ctx context.Context, group *Group) error
    FindByID(ctx context.Context, id uuid.UUID) (*Group, error)
    Update(ctx context.Context, group *Group) error
    Delete(ctx context.Context, id uuid.UUID) error

    // Queries
    FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*Group, error)
    FindByMemberID(ctx context.Context, userID uuid.UUID) ([]*Group, error)
    ExistsByName(ctx context.Context, name string, ownerID uuid.UUID) (bool, error)
}
```

### 2.2 MembershipRepository

```go
// internal/domain/group/membership_repository.go

package group

import (
    "context"
    "github.com/google/uuid"
)

type MembershipWithUser struct {
    Membership
    UserName  string
    UserEmail string
}

type MembershipRepository interface {
    // CRUD
    Create(ctx context.Context, membership *Membership) error
    FindByID(ctx context.Context, id uuid.UUID) (*Membership, error)
    Update(ctx context.Context, membership *Membership) error
    Delete(ctx context.Context, id uuid.UUID) error

    // Queries
    FindByGroupID(ctx context.Context, groupID uuid.UUID) ([]*Membership, error)
    FindByGroupIDWithUsers(ctx context.Context, groupID uuid.UUID) ([]*MembershipWithUser, error)
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]*Membership, error)
    FindByGroupAndUser(ctx context.Context, groupID, userID uuid.UUID) (*Membership, error)
    Exists(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
    CountByGroupID(ctx context.Context, groupID uuid.UUID) (int, error)
    DeleteByGroupID(ctx context.Context, groupID uuid.UUID) error
}
```

### 2.3 InvitationRepository

```go
// internal/domain/group/invitation_repository.go

package group

import (
    "context"
    "github.com/google/uuid"
)

type InvitationRepository interface {
    // CRUD
    Create(ctx context.Context, invitation *Invitation) error
    FindByID(ctx context.Context, id uuid.UUID) (*Invitation, error)
    Update(ctx context.Context, invitation *Invitation) error
    Delete(ctx context.Context, id uuid.UUID) error

    // Queries
    FindByToken(ctx context.Context, token string) (*Invitation, error)
    FindPendingByGroupID(ctx context.Context, groupID uuid.UUID) ([]*Invitation, error)
    FindPendingByEmail(ctx context.Context, email string) ([]*Invitation, error)
    FindPendingByGroupAndEmail(ctx context.Context, groupID uuid.UUID, email string) (*Invitation, error)
    DeleteByGroupID(ctx context.Context, groupID uuid.UUID) error

    // Maintenance
    ExpireOld(ctx context.Context) (int64, error)
}
```

---

## 3. ユースケース

### 3.1 グループ作成

```go
// internal/usecase/group/create_group.go

package group

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/group"
    "gc-storage/internal/domain/storage"
    "gc-storage/pkg/apperror"
)

type CreateGroupInput struct {
    Name        string
    Description string
    OwnerID     uuid.UUID
}

type CreateGroupOutput struct {
    Group      *group.Group
    Membership *group.Membership
    RootFolder *storage.Folder
}

type CreateGroupUseCase struct {
    groupRepo      group.GroupRepository
    membershipRepo group.MembershipRepository
    folderRepo     storage.FolderRepository
    txManager      TransactionManager
}

func NewCreateGroupUseCase(
    groupRepo group.GroupRepository,
    membershipRepo group.MembershipRepository,
    folderRepo storage.FolderRepository,
    txManager TransactionManager,
) *CreateGroupUseCase {
    return &CreateGroupUseCase{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        folderRepo:     folderRepo,
        txManager:      txManager,
    }
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, input CreateGroupInput) (*CreateGroupOutput, error) {
    // 1. Validate name
    if len(input.Name) == 0 || len(input.Name) > 100 {
        return nil, apperror.NewValidation("group name must be 1-100 characters", nil)
    }
    if len(input.Description) > 500 {
        return nil, apperror.NewValidation("description must not exceed 500 characters", nil)
    }

    var output *CreateGroupOutput

    err := uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        now := time.Now()

        // 2. Create group
        grp := &group.Group{
            ID:          uuid.New(),
            Name:        input.Name,
            Description: input.Description,
            OwnerID:     input.OwnerID,
            Status:      group.GroupStatusActive,
            CreatedAt:   now,
            UpdatedAt:   now,
        }
        if err := uc.groupRepo.Create(ctx, grp); err != nil {
            return err
        }

        // 3. Create owner membership
        membership := &group.Membership{
            ID:       uuid.New(),
            GroupID:  grp.ID,
            UserID:   input.OwnerID,
            Role:     group.GroupRoleOwner,
            JoinedAt: now,
        }
        if err := uc.membershipRepo.Create(ctx, membership); err != nil {
            return err
        }

        // 4. Create group root folder
        rootFolder := &storage.Folder{
            ID:        uuid.New(),
            Name:      storage.MustNewFolderName(grp.Name),
            ParentID:  nil,
            OwnerID:   grp.ID,
            OwnerType: storage.OwnerTypeGroup,
            Path:      "/group/" + grp.ID.String(),
            Depth:     0,
            Status:    storage.FolderStatusActive,
            CreatedAt: now,
            UpdatedAt: now,
        }
        if err := uc.folderRepo.Create(ctx, rootFolder); err != nil {
            return err
        }

        output = &CreateGroupOutput{
            Group:      grp,
            Membership: membership,
            RootFolder: rootFolder,
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return output, nil
}
```

### 3.2 メンバー招待

```go
// internal/usecase/group/invite_member.go

package group

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/group"
    "gc-storage/internal/domain/user"
    "gc-storage/internal/infrastructure/email"
    "gc-storage/pkg/apperror"
)

type InviteMemberInput struct {
    GroupID   uuid.UUID
    Email     string
    Role      group.GroupRole
    InviterID uuid.UUID
}

type InviteMemberOutput struct {
    Invitation *group.Invitation
}

type InviteMemberUseCase struct {
    groupRepo      group.GroupRepository
    membershipRepo group.MembershipRepository
    invitationRepo group.InvitationRepository
    userRepo       user.UserRepository
    emailService   email.Service
    baseURL        string
}

func NewInviteMemberUseCase(
    groupRepo group.GroupRepository,
    membershipRepo group.MembershipRepository,
    invitationRepo group.InvitationRepository,
    userRepo user.UserRepository,
    emailService email.Service,
    baseURL string,
) *InviteMemberUseCase {
    return &InviteMemberUseCase{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        invitationRepo: invitationRepo,
        userRepo:       userRepo,
        emailService:   emailService,
        baseURL:        baseURL,
    }
}

func (uc *InviteMemberUseCase) Execute(ctx context.Context, input InviteMemberInput) (*InviteMemberOutput, error) {
    // 1. Validate role (cannot invite as owner)
    if input.Role == group.GroupRoleOwner {
        return nil, apperror.NewBadRequest("cannot invite with owner role", nil)
    }

    // 2. Get and validate group
    grp, err := uc.groupRepo.FindByID(ctx, input.GroupID)
    if err != nil {
        return nil, apperror.NewNotFound("group not found", err)
    }
    if !grp.IsActive() {
        return nil, apperror.NewBadRequest("group is not active", nil)
    }

    // 3. Verify inviter's permission
    inviterMembership, err := uc.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.InviterID)
    if err != nil {
        return nil, apperror.NewForbidden("not a member of this group", err)
    }
    if !inviterMembership.Role.CanInviteMembers() {
        return nil, apperror.NewForbidden("insufficient permission to invite members", nil)
    }

    // 4. Check if user with email is already a member
    existingUser, _ := uc.userRepo.FindByEmail(ctx, input.Email)
    if existingUser != nil {
        exists, _ := uc.membershipRepo.Exists(ctx, input.GroupID, existingUser.ID)
        if exists {
            return nil, apperror.NewConflict("user is already a member", nil)
        }
    }

    // 5. Check for existing pending invitation
    existingInvite, _ := uc.invitationRepo.FindPendingByGroupAndEmail(ctx, input.GroupID, input.Email)
    if existingInvite != nil {
        return nil, apperror.NewConflict("invitation already exists for this email", nil)
    }

    // 6. Generate invitation token
    token, err := group.GenerateInvitationToken()
    if err != nil {
        return nil, apperror.NewInternal("failed to generate invitation token", err)
    }

    // 7. Create invitation
    now := time.Now()
    invitation := &group.Invitation{
        ID:        uuid.New(),
        GroupID:   input.GroupID,
        Email:     input.Email,
        Token:     token,
        Role:      input.Role,
        InvitedBy: input.InviterID,
        ExpiresAt: now.Add(group.InvitationExpiry),
        Status:    group.InvitationStatusPending,
        CreatedAt: now,
    }
    if err := uc.invitationRepo.Create(ctx, invitation); err != nil {
        return nil, err
    }

    // 8. Send invitation email (async)
    inviteURL := uc.baseURL + "/invite/" + token
    go uc.emailService.SendInvitation(context.Background(), email.InvitationData{
        To:        input.Email,
        GroupName: grp.Name,
        InviteURL: inviteURL,
        ExpiresAt: invitation.ExpiresAt,
    })

    return &InviteMemberOutput{Invitation: invitation}, nil
}
```

### 3.3 招待承諾

```go
// internal/usecase/group/accept_invitation.go

package group

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/group"
    "gc-storage/internal/domain/user"
    "gc-storage/pkg/apperror"
)

type AcceptInvitationInput struct {
    Token  string
    UserID uuid.UUID
}

type AcceptInvitationOutput struct {
    Membership *group.Membership
    Group      *group.Group
}

type AcceptInvitationUseCase struct {
    groupRepo      group.GroupRepository
    membershipRepo group.MembershipRepository
    invitationRepo group.InvitationRepository
    userRepo       user.UserRepository
    txManager      TransactionManager
}

func NewAcceptInvitationUseCase(
    groupRepo group.GroupRepository,
    membershipRepo group.MembershipRepository,
    invitationRepo group.InvitationRepository,
    userRepo user.UserRepository,
    txManager TransactionManager,
) *AcceptInvitationUseCase {
    return &AcceptInvitationUseCase{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        invitationRepo: invitationRepo,
        userRepo:       userRepo,
        txManager:      txManager,
    }
}

func (uc *AcceptInvitationUseCase) Execute(ctx context.Context, input AcceptInvitationInput) (*AcceptInvitationOutput, error) {
    var output *AcceptInvitationOutput

    err := uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. Find invitation by token
        invitation, err := uc.invitationRepo.FindByToken(ctx, input.Token)
        if err != nil {
            return apperror.NewNotFound("invitation not found", err)
        }

        // 2. Validate invitation
        if !invitation.CanAccept() {
            if invitation.IsExpired() {
                return apperror.NewBadRequest("invitation has expired", nil)
            }
            return apperror.NewBadRequest("invitation is no longer valid", nil)
        }

        // 3. Get user
        usr, err := uc.userRepo.FindByID(ctx, input.UserID)
        if err != nil {
            return apperror.NewNotFound("user not found", err)
        }

        // 4. Verify email matches
        if usr.Email != invitation.Email {
            return apperror.NewForbidden("email does not match invitation", nil)
        }

        // 5. Get group
        grp, err := uc.groupRepo.FindByID(ctx, invitation.GroupID)
        if err != nil {
            return apperror.NewNotFound("group not found", err)
        }
        if !grp.IsActive() {
            return apperror.NewBadRequest("group is no longer active", nil)
        }

        // 6. Check if already a member
        exists, _ := uc.membershipRepo.Exists(ctx, invitation.GroupID, input.UserID)
        if exists {
            return apperror.NewConflict("already a member of this group", nil)
        }

        // 7. Create membership
        now := time.Now()
        membership := &group.Membership{
            ID:       uuid.New(),
            GroupID:  invitation.GroupID,
            UserID:   input.UserID,
            Role:     invitation.Role,
            JoinedAt: now,
        }
        if err := uc.membershipRepo.Create(ctx, membership); err != nil {
            return err
        }

        // 8. Update invitation status
        invitation.Status = group.InvitationStatusAccepted
        if err := uc.invitationRepo.Update(ctx, invitation); err != nil {
            return err
        }

        output = &AcceptInvitationOutput{
            Membership: membership,
            Group:      grp,
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return output, nil
}
```

### 3.4 メンバー削除

```go
// internal/usecase/group/remove_member.go

package group

import (
    "context"
    "github.com/google/uuid"
    "gc-storage/internal/domain/group"
    "gc-storage/pkg/apperror"
)

type RemoveMemberInput struct {
    GroupID   uuid.UUID
    TargetID  uuid.UUID  // User to remove
    ActorID   uuid.UUID  // User performing the action
}

type RemoveMemberUseCase struct {
    groupRepo      group.GroupRepository
    membershipRepo group.MembershipRepository
    txManager      TransactionManager
}

func NewRemoveMemberUseCase(
    groupRepo group.GroupRepository,
    membershipRepo group.MembershipRepository,
    txManager TransactionManager,
) *RemoveMemberUseCase {
    return &RemoveMemberUseCase{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        txManager:      txManager,
    }
}

func (uc *RemoveMemberUseCase) Execute(ctx context.Context, input RemoveMemberInput) error {
    return uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. Get group
        grp, err := uc.groupRepo.FindByID(ctx, input.GroupID)
        if err != nil {
            return apperror.NewNotFound("group not found", err)
        }
        if !grp.IsActive() {
            return apperror.NewBadRequest("group is not active", nil)
        }

        // 2. Get actor's membership
        actorMembership, err := uc.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.ActorID)
        if err != nil {
            return apperror.NewForbidden("not a member of this group", err)
        }

        // 3. Get target's membership
        targetMembership, err := uc.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.TargetID)
        if err != nil {
            return apperror.NewNotFound("target user is not a member", err)
        }

        // 4. Verify permission
        if !actorMembership.Role.CanRemove(targetMembership.Role) {
            return apperror.NewForbidden("insufficient permission to remove this member", nil)
        }

        // 5. Cannot remove owner
        if targetMembership.IsOwner() {
            return apperror.NewBadRequest("cannot remove group owner", nil)
        }

        // 6. Delete membership
        if err := uc.membershipRepo.Delete(ctx, targetMembership.ID); err != nil {
            return err
        }

        return nil
    })
}
```

### 3.5 グループ脱退

```go
// internal/usecase/group/leave_group.go

package group

import (
    "context"
    "github.com/google/uuid"
    "gc-storage/internal/domain/group"
    "gc-storage/pkg/apperror"
)

type LeaveGroupInput struct {
    GroupID uuid.UUID
    UserID  uuid.UUID
}

type LeaveGroupUseCase struct {
    groupRepo      group.GroupRepository
    membershipRepo group.MembershipRepository
}

func NewLeaveGroupUseCase(
    groupRepo group.GroupRepository,
    membershipRepo group.MembershipRepository,
) *LeaveGroupUseCase {
    return &LeaveGroupUseCase{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
    }
}

func (uc *LeaveGroupUseCase) Execute(ctx context.Context, input LeaveGroupInput) error {
    // 1. Get group
    grp, err := uc.groupRepo.FindByID(ctx, input.GroupID)
    if err != nil {
        return apperror.NewNotFound("group not found", err)
    }
    if !grp.IsActive() {
        return apperror.NewBadRequest("group is not active", nil)
    }

    // 2. Get membership
    membership, err := uc.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.UserID)
    if err != nil {
        return apperror.NewNotFound("not a member of this group", err)
    }

    // 3. Check if can leave
    if !membership.CanLeave() {
        return apperror.NewBadRequest("owner cannot leave the group, transfer ownership first", nil)
    }

    // 4. Delete membership
    return uc.membershipRepo.Delete(ctx, membership.ID)
}
```

### 3.6 所有権譲渡

```go
// internal/usecase/group/transfer_ownership.go

package group

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/group"
    "gc-storage/pkg/apperror"
)

type TransferOwnershipInput struct {
    GroupID       uuid.UUID
    CurrentOwner  uuid.UUID
    NewOwner      uuid.UUID
}

type TransferOwnershipOutput struct {
    Group *group.Group
}

type TransferOwnershipUseCase struct {
    groupRepo      group.GroupRepository
    membershipRepo group.MembershipRepository
    txManager      TransactionManager
}

func NewTransferOwnershipUseCase(
    groupRepo group.GroupRepository,
    membershipRepo group.MembershipRepository,
    txManager TransactionManager,
) *TransferOwnershipUseCase {
    return &TransferOwnershipUseCase{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        txManager:      txManager,
    }
}

func (uc *TransferOwnershipUseCase) Execute(ctx context.Context, input TransferOwnershipInput) (*TransferOwnershipOutput, error) {
    var output *TransferOwnershipOutput

    err := uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. Get group
        grp, err := uc.groupRepo.FindByID(ctx, input.GroupID)
        if err != nil {
            return apperror.NewNotFound("group not found", err)
        }
        if !grp.IsActive() {
            return apperror.NewBadRequest("group is not active", nil)
        }

        // 2. Verify current owner
        if grp.OwnerID != input.CurrentOwner {
            return apperror.NewForbidden("not the current owner", nil)
        }

        // 3. Cannot transfer to self
        if input.CurrentOwner == input.NewOwner {
            return apperror.NewBadRequest("cannot transfer ownership to yourself", nil)
        }

        // 4. Verify new owner is a member
        newOwnerMembership, err := uc.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.NewOwner)
        if err != nil {
            return apperror.NewBadRequest("new owner must be a group member", err)
        }

        // 5. Get current owner's membership
        currentOwnerMembership, err := uc.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.CurrentOwner)
        if err != nil {
            return err
        }

        now := time.Now()

        // 6. Update new owner's role to owner
        newOwnerMembership.Role = group.GroupRoleOwner
        if err := uc.membershipRepo.Update(ctx, newOwnerMembership); err != nil {
            return err
        }

        // 7. Downgrade current owner to admin
        currentOwnerMembership.Role = group.GroupRoleAdmin
        if err := uc.membershipRepo.Update(ctx, currentOwnerMembership); err != nil {
            return err
        }

        // 8. Update group's owner_id
        grp.OwnerID = input.NewOwner
        grp.UpdatedAt = now
        if err := uc.groupRepo.Update(ctx, grp); err != nil {
            return err
        }

        output = &TransferOwnershipOutput{Group: grp}
        return nil
    })

    if err != nil {
        return nil, err
    }

    return output, nil
}
```

### 3.7 ロール変更

```go
// internal/usecase/group/change_role.go

package group

import (
    "context"
    "github.com/google/uuid"
    "gc-storage/internal/domain/group"
    "gc-storage/pkg/apperror"
)

type ChangeRoleInput struct {
    GroupID  uuid.UUID
    TargetID uuid.UUID
    NewRole  group.GroupRole
    ActorID  uuid.UUID
}

type ChangeRoleOutput struct {
    Membership *group.Membership
}

type ChangeRoleUseCase struct {
    groupRepo      group.GroupRepository
    membershipRepo group.MembershipRepository
}

func NewChangeRoleUseCase(
    groupRepo group.GroupRepository,
    membershipRepo group.MembershipRepository,
) *ChangeRoleUseCase {
    return &ChangeRoleUseCase{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
    }
}

func (uc *ChangeRoleUseCase) Execute(ctx context.Context, input ChangeRoleInput) (*ChangeRoleOutput, error) {
    // 1. Get group
    grp, err := uc.groupRepo.FindByID(ctx, input.GroupID)
    if err != nil {
        return nil, apperror.NewNotFound("group not found", err)
    }
    if !grp.IsActive() {
        return nil, apperror.NewBadRequest("group is not active", nil)
    }

    // 2. Cannot assign owner role (use transfer ownership)
    if input.NewRole == group.GroupRoleOwner {
        return nil, apperror.NewBadRequest("use ownership transfer to assign owner role", nil)
    }

    // 3. Get actor's membership
    actorMembership, err := uc.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.ActorID)
    if err != nil {
        return nil, apperror.NewForbidden("not a member of this group", err)
    }

    // 4. Get target's membership
    targetMembership, err := uc.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.TargetID)
    if err != nil {
        return nil, apperror.NewNotFound("target user is not a member", err)
    }

    // 5. Verify permission
    if !actorMembership.Role.CanChangeRole(targetMembership.Role, input.NewRole) {
        return nil, apperror.NewForbidden("insufficient permission to change role", nil)
    }

    // 6. Cannot change own role
    if input.ActorID == input.TargetID {
        return nil, apperror.NewBadRequest("cannot change your own role", nil)
    }

    // 7. Update role
    targetMembership.Role = input.NewRole
    if err := uc.membershipRepo.Update(ctx, targetMembership); err != nil {
        return nil, err
    }

    return &ChangeRoleOutput{Membership: targetMembership}, nil
}
```

### 3.8 グループ削除

```go
// internal/usecase/group/delete_group.go

package group

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/group"
    "gc-storage/internal/domain/storage"
    "gc-storage/pkg/apperror"
)

type DeleteGroupInput struct {
    GroupID uuid.UUID
    ActorID uuid.UUID
}

type DeleteGroupUseCase struct {
    groupRepo      group.GroupRepository
    membershipRepo group.MembershipRepository
    invitationRepo group.InvitationRepository
    folderRepo     storage.FolderRepository
    txManager      TransactionManager
}

func NewDeleteGroupUseCase(
    groupRepo group.GroupRepository,
    membershipRepo group.MembershipRepository,
    invitationRepo group.InvitationRepository,
    folderRepo storage.FolderRepository,
    txManager TransactionManager,
) *DeleteGroupUseCase {
    return &DeleteGroupUseCase{
        groupRepo:      groupRepo,
        membershipRepo: membershipRepo,
        invitationRepo: invitationRepo,
        folderRepo:     folderRepo,
        txManager:      txManager,
    }
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, input DeleteGroupInput) error {
    return uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 1. Get group
        grp, err := uc.groupRepo.FindByID(ctx, input.GroupID)
        if err != nil {
            return apperror.NewNotFound("group not found", err)
        }
        if !grp.IsActive() {
            return apperror.NewBadRequest("group is already deleted", nil)
        }

        // 2. Verify actor is owner
        if grp.OwnerID != input.ActorID {
            return apperror.NewForbidden("only the owner can delete the group", nil)
        }

        now := time.Now()

        // 3. Delete all invitations
        if err := uc.invitationRepo.DeleteByGroupID(ctx, input.GroupID); err != nil {
            return err
        }

        // 4. Delete all memberships
        if err := uc.membershipRepo.DeleteByGroupID(ctx, input.GroupID); err != nil {
            return err
        }

        // 5. Trash group's root folder (this will cascade to all contents)
        rootFolder, err := uc.folderRepo.FindRootByOwner(ctx, storage.OwnerTypeGroup, input.GroupID)
        if err == nil && rootFolder != nil {
            rootFolder.Status = storage.FolderStatusTrashed
            rootFolder.TrashedAt = &now
            if err := uc.folderRepo.Update(ctx, rootFolder); err != nil {
                return err
            }
        }

        // 6. Soft delete group
        grp.Status = group.GroupStatusDeleted
        grp.UpdatedAt = now
        if err := uc.groupRepo.Update(ctx, grp); err != nil {
            return err
        }

        return nil
    })
}
```

---

## 4. ハンドラー

```go
// internal/interface/handler/group_handler.go

package handler

import (
    "net/http"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "gc-storage/internal/interface/dto"
    "gc-storage/internal/interface/middleware"
    "gc-storage/internal/usecase/group"
)

type GroupHandler struct {
    createGroup       *group.CreateGroupUseCase
    getGroup          *group.GetGroupUseCase
    updateGroup       *group.UpdateGroupUseCase
    deleteGroup       *group.DeleteGroupUseCase
    listMyGroups      *group.ListMyGroupsUseCase
    inviteMember      *group.InviteMemberUseCase
    acceptInvitation  *group.AcceptInvitationUseCase
    declineInvitation *group.DeclineInvitationUseCase
    cancelInvitation  *group.CancelInvitationUseCase
    listInvitations   *group.ListInvitationsUseCase
    listMembers       *group.ListMembersUseCase
    removeMember      *group.RemoveMemberUseCase
    leaveGroup        *group.LeaveGroupUseCase
    changeRole        *group.ChangeRoleUseCase
    transferOwnership *group.TransferOwnershipUseCase
}

// POST /api/v1/groups
func (h *GroupHandler) Create(c echo.Context) error {
    var req dto.CreateGroupRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.createGroup.Execute(c.Request().Context(), group.CreateGroupInput{
        Name:        req.Name,
        Description: req.Description,
        OwnerID:     claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, dto.GroupResponse{
        ID:          output.Group.ID,
        Name:        output.Group.Name,
        Description: output.Group.Description,
        OwnerID:     output.Group.OwnerID,
        Role:        string(output.Membership.Role),
        CreatedAt:   output.Group.CreatedAt,
    })
}

// GET /api/v1/groups/:id
func (h *GroupHandler) Get(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.getGroup.Execute(c.Request().Context(), group.GetGroupInput{
        GroupID: groupID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.GroupDetailResponse{
        ID:          output.Group.ID,
        Name:        output.Group.Name,
        Description: output.Group.Description,
        OwnerID:     output.Group.OwnerID,
        Role:        string(output.Membership.Role),
        MemberCount: output.MemberCount,
        CreatedAt:   output.Group.CreatedAt,
    })
}

// GET /api/v1/groups
func (h *GroupHandler) ListMyGroups(c echo.Context) error {
    claims := middleware.GetClaims(c)
    output, err := h.listMyGroups.Execute(c.Request().Context(), group.ListMyGroupsInput{
        UserID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.GroupListResponse{
        Groups: dto.ToGroupResponses(output.Groups, output.Memberships),
    })
}

// DELETE /api/v1/groups/:id
func (h *GroupHandler) Delete(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    claims := middleware.GetClaims(c)
    err = h.deleteGroup.Execute(c.Request().Context(), group.DeleteGroupInput{
        GroupID: groupID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// POST /api/v1/groups/:id/invitations
func (h *GroupHandler) InviteMember(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    var req dto.InviteMemberRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.inviteMember.Execute(c.Request().Context(), group.InviteMemberInput{
        GroupID:   groupID,
        Email:     req.Email,
        Role:      group.GroupRole(req.Role),
        InviterID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, dto.InvitationResponse{
        ID:        output.Invitation.ID,
        Email:     output.Invitation.Email,
        Role:      string(output.Invitation.Role),
        ExpiresAt: output.Invitation.ExpiresAt,
        Status:    string(output.Invitation.Status),
    })
}

// POST /api/v1/invitations/:token/accept
func (h *GroupHandler) AcceptInvitation(c echo.Context) error {
    token := c.Param("token")

    claims := middleware.GetClaims(c)
    output, err := h.acceptInvitation.Execute(c.Request().Context(), group.AcceptInvitationInput{
        Token:  token,
        UserID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.AcceptInvitationResponse{
        GroupID:   output.Group.ID,
        GroupName: output.Group.Name,
        Role:      string(output.Membership.Role),
    })
}

// POST /api/v1/invitations/:token/decline
func (h *GroupHandler) DeclineInvitation(c echo.Context) error {
    token := c.Param("token")

    claims := middleware.GetClaims(c)
    err := h.declineInvitation.Execute(c.Request().Context(), group.DeclineInvitationInput{
        Token:  token,
        UserID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// GET /api/v1/groups/:id/members
func (h *GroupHandler) ListMembers(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.listMembers.Execute(c.Request().Context(), group.ListMembersInput{
        GroupID: groupID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.MemberListResponse{
        Members: dto.ToMemberResponses(output.Members),
    })
}

// DELETE /api/v1/groups/:id/members/:userId
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
    err = h.removeMember.Execute(c.Request().Context(), group.RemoveMemberInput{
        GroupID:  groupID,
        TargetID: targetID,
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// POST /api/v1/groups/:id/leave
func (h *GroupHandler) Leave(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    claims := middleware.GetClaims(c)
    err = h.leaveGroup.Execute(c.Request().Context(), group.LeaveGroupInput{
        GroupID: groupID,
        UserID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// PATCH /api/v1/groups/:id/members/:userId/role
func (h *GroupHandler) ChangeRole(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }
    targetID, err := uuid.Parse(c.Param("userId"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
    }

    var req dto.ChangeRoleRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.changeRole.Execute(c.Request().Context(), group.ChangeRoleInput{
        GroupID:  groupID,
        TargetID: targetID,
        NewRole:  group.GroupRole(req.Role),
        ActorID:  claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.MemberResponse{
        UserID:   output.Membership.UserID,
        Role:     string(output.Membership.Role),
        JoinedAt: output.Membership.JoinedAt,
    })
}

// POST /api/v1/groups/:id/transfer
func (h *GroupHandler) TransferOwnership(c echo.Context) error {
    groupID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
    }

    var req dto.TransferOwnershipRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.transferOwnership.Execute(c.Request().Context(), group.TransferOwnershipInput{
        GroupID:      groupID,
        CurrentOwner: claims.UserID,
        NewOwner:     req.NewOwnerID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.GroupResponse{
        ID:      output.Group.ID,
        Name:    output.Group.Name,
        OwnerID: output.Group.OwnerID,
    })
}
```

---

## 5. DTO定義

```go
// internal/interface/dto/group.go

package dto

import (
    "time"
    "github.com/google/uuid"
)

// Group DTOs
type CreateGroupRequest struct {
    Name        string `json:"name" validate:"required,min=1,max=100"`
    Description string `json:"description" validate:"max=500"`
}

type UpdateGroupRequest struct {
    Name        *string `json:"name" validate:"omitempty,min=1,max=100"`
    Description *string `json:"description" validate:"omitempty,max=500"`
}

type GroupResponse struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    OwnerID     uuid.UUID `json:"owner_id"`
    Role        string    `json:"role"`
    CreatedAt   time.Time `json:"created_at"`
}

type GroupDetailResponse struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    OwnerID     uuid.UUID `json:"owner_id"`
    Role        string    `json:"role"`
    MemberCount int       `json:"member_count"`
    CreatedAt   time.Time `json:"created_at"`
}

type GroupListResponse struct {
    Groups []GroupResponse `json:"groups"`
}

// Invitation DTOs
type InviteMemberRequest struct {
    Email string `json:"email" validate:"required,email"`
    Role  string `json:"role" validate:"required,oneof=member admin"`
}

type InvitationResponse struct {
    ID        uuid.UUID `json:"id"`
    Email     string    `json:"email"`
    Role      string    `json:"role"`
    ExpiresAt time.Time `json:"expires_at"`
    Status    string    `json:"status"`
}

type InvitationListResponse struct {
    Invitations []InvitationResponse `json:"invitations"`
}

type AcceptInvitationResponse struct {
    GroupID   uuid.UUID `json:"group_id"`
    GroupName string    `json:"group_name"`
    Role      string    `json:"role"`
}

// Member DTOs
type MemberResponse struct {
    UserID   uuid.UUID `json:"user_id"`
    UserName string    `json:"user_name"`
    Email    string    `json:"email"`
    Role     string    `json:"role"`
    JoinedAt time.Time `json:"joined_at"`
}

type MemberListResponse struct {
    Members []MemberResponse `json:"members"`
}

type ChangeRoleRequest struct {
    Role string `json:"role" validate:"required,oneof=member admin"`
}

type TransferOwnershipRequest struct {
    NewOwnerID uuid.UUID `json:"new_owner_id" validate:"required"`
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
// internal/job/invitation_expiry.go

package job

import (
    "context"
    "log/slog"
    "gc-storage/internal/domain/group"
)

type InvitationExpiryJob struct {
    invitationRepo group.InvitationRepository
    logger         *slog.Logger
}

func NewInvitationExpiryJob(
    invitationRepo group.InvitationRepository,
    logger *slog.Logger,
) *InvitationExpiryJob {
    return &InvitationExpiryJob{
        invitationRepo: invitationRepo,
        logger:         logger,
    }
}

// Run executes every hour
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
- [ ] グループ名・説明を更新できる（admin/owner）
- [ ] グループを削除できる（ownerのみ）
- [ ] 削除時にメンバーシップ・招待・ルートフォルダも削除される
- [ ] 所属グループ一覧を取得できる

### メンバー招待
- [ ] admin/ownerがメンバーを招待できる
- [ ] 招待メールが送信される
- [ ] 招待リンクから承諾できる
- [ ] 招待を辞退できる
- [ ] 招待を取消できる（admin/owner）
- [ ] 同じメールへの重複招待は拒否される
- [ ] 既存メンバーへの招待は拒否される
- [ ] ownerロールでの招待は拒否される
- [ ] 7日後に招待が期限切れになる

### メンバー管理
- [ ] メンバー一覧を取得できる
- [ ] admin/ownerがメンバーを削除できる
- [ ] メンバーが自発的に脱退できる
- [ ] ownerは脱退できない（所有権譲渡が必要）
- [ ] ownerがメンバーのロールを変更できる
- [ ] adminはmember/adminロールを変更できる
- [ ] 自分のロールは変更できない

### 所有権譲渡
- [ ] ownerが他のメンバーに所有権を譲渡できる
- [ ] 譲渡後、旧ownerはadminになる
- [ ] 非メンバーへの譲渡は拒否される

### 権限検証
- [ ] member: グループ情報閲覧、共有リソースアクセス
- [ ] admin: 招待、メンバー削除、グループ設定変更
- [ ] owner: 全権限、グループ削除、所有権譲渡

---

## 関連ドキュメント

- [グループドメイン](../03-domains/group.md)
- [セキュリティ設計](../02-architecture/SECURITY.md)
- [Email仕様](./infra-email.md)
- [Storage Core仕様](./storage-core.md)
