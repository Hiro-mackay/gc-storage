# Authorization Permission 詳細設計

## 概要

Authorization Permissionは、リソースに対するアクセス権限の付与、取り消し、継承解決を担当するモジュールです。
PBAC（Policy-Based）とReBAC（Relationship-Based）のハイブリッドモデルを実装します。

**設計原則:**
- **PBAC**: 「このPermissionを持っているか？」で最終判定
- **ReBAC**: 「どの関係性を通じてPermissionを得ているか？」を解決
- **Role**: Permission の集合（割り当ての便宜のため）
- 関係性の連鎖を辿ってPermissionを収集し、最終的にPermissionで判定

**スコープ:**
- Permission/Role 定義
- Relationship Tuple 管理
- Permission 解決アルゴリズム
- 権限付与・取り消し
- 認可ミドルウェア

**参照ドキュメント:**
- [権限ドメイン](../03-domains/permission.md)
- [セキュリティ設計](../02-architecture/SECURITY.md)

---

## 1. 値オブジェクト定義

### 1.1 Permission

```go
// internal/domain/authz/permission.go

package authz

type Permission string

// File permissions
const (
    PermFileRead            Permission = "file:read"
    PermFileWrite           Permission = "file:write"
    PermFileDelete          Permission = "file:delete"
    PermFileRestore         Permission = "file:restore"
    PermFilePermanentDelete Permission = "file:permanent_delete"
    PermFileMove            Permission = "file:move"
    PermFileRename          Permission = "file:rename"
    PermFileShare           Permission = "file:share"
)

// Folder permissions
const (
    PermFolderRead   Permission = "folder:read"
    PermFolderCreate Permission = "folder:create"
    PermFolderDelete Permission = "folder:delete"
    PermFolderMove   Permission = "folder:move"
    PermFolderRename Permission = "folder:rename"
    PermFolderShare  Permission = "folder:share"
)

// Permission management
const (
    PermPermissionRead   Permission = "permission:read"
    PermPermissionGrant  Permission = "permission:grant"
    PermPermissionRevoke Permission = "permission:revoke"
)

// Group permissions
const (
    PermGroupRead         Permission = "group:read"
    PermGroupUpdate       Permission = "group:update"
    PermGroupDelete       Permission = "group:delete"
    PermGroupMemberRead   Permission = "group:member:read"
    PermGroupMemberAdd    Permission = "group:member:add"
    PermGroupMemberRemove Permission = "group:member:remove"
    PermGroupMemberRole   Permission = "group:member:role"
)

func (p Permission) String() string {
    return string(p)
}

func (p Permission) IsValid() bool {
    validPerms := map[Permission]bool{
        PermFileRead: true, PermFileWrite: true, PermFileDelete: true,
        PermFileRestore: true, PermFilePermanentDelete: true,
        PermFileMove: true, PermFileRename: true, PermFileShare: true,
        PermFolderRead: true, PermFolderCreate: true, PermFolderDelete: true,
        PermFolderMove: true, PermFolderRename: true, PermFolderShare: true,
        PermPermissionRead: true, PermPermissionGrant: true, PermPermissionRevoke: true,
        PermGroupRead: true, PermGroupUpdate: true, PermGroupDelete: true,
        PermGroupMemberRead: true, PermGroupMemberAdd: true,
        PermGroupMemberRemove: true, PermGroupMemberRole: true,
    }
    return validPerms[p]
}
```

### 1.2 PermissionSet

```go
// internal/domain/authz/permission_set.go

package authz

type PermissionSet struct {
    perms map[Permission]bool
}

func NewPermissionSet() *PermissionSet {
    return &PermissionSet{perms: make(map[Permission]bool)}
}

func (ps *PermissionSet) Add(p Permission) {
    ps.perms[p] = true
}

func (ps *PermissionSet) AddAll(perms []Permission) {
    for _, p := range perms {
        ps.perms[p] = true
    }
}

func (ps *PermissionSet) Has(p Permission) bool {
    return ps.perms[p]
}

func (ps *PermissionSet) HasAny(perms []Permission) bool {
    for _, p := range perms {
        if ps.perms[p] {
            return true
        }
    }
    return false
}

func (ps *PermissionSet) HasAll(perms []Permission) bool {
    for _, p := range perms {
        if !ps.perms[p] {
            return false
        }
    }
    return true
}

func (ps *PermissionSet) ToSlice() []Permission {
    result := make([]Permission, 0, len(ps.perms))
    for p := range ps.perms {
        result = append(result, p)
    }
    return result
}
```

### 1.3 Role

```go
// internal/domain/authz/role.go

package authz

type Role string

const (
    RoleViewer  Role = "viewer"
    RoleEditor  Role = "editor"
    RoleManager Role = "manager"
    RoleOwner   Role = "owner"
)

// Permissions returns the permissions included in this role
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

// Includes returns true if this role includes the other role
func (r Role) Includes(other Role) bool {
    hierarchy := map[Role]int{
        RoleViewer:  1,
        RoleEditor:  2,
        RoleManager: 3,
        RoleOwner:   4,
    }
    return hierarchy[r] >= hierarchy[other]
}

func (r Role) IsValid() bool {
    return r == RoleViewer || r == RoleEditor || r == RoleManager || r == RoleOwner
}

func (r Role) String() string {
    return string(r)
}
```

### 1.4 RelationType

```go
// internal/domain/authz/relation.go

package authz

type RelationType string

const (
    RelationOwner   RelationType = "owner"
    RelationMember  RelationType = "member"
    RelationParent  RelationType = "parent"
    RelationViewer  RelationType = "viewer"
    RelationEditor  RelationType = "editor"
    RelationManager RelationType = "manager"
)

// ToRole converts relation to role if applicable
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

func (r RelationType) String() string {
    return string(r)
}
```

---

## 2. エンティティ定義

### 2.1 PermissionGrant

```go
// internal/domain/authz/permission_grant.go

package authz

import (
    "time"
    "github.com/google/uuid"
)

type ResourceType string

const (
    ResourceTypeFile   ResourceType = "file"
    ResourceTypeFolder ResourceType = "folder"
    ResourceTypeGroup  ResourceType = "group"
)

type GranteeType string

const (
    GranteeTypeUser  GranteeType = "user"
    GranteeTypeGroup GranteeType = "group"
)

type PermissionGrant struct {
    ID           uuid.UUID
    ResourceType ResourceType
    ResourceID   uuid.UUID
    GranteeType  GranteeType
    GranteeID    uuid.UUID
    Role         *Role       // nil if direct permission
    Permission   *Permission // nil if role-based
    GrantedBy    uuid.UUID
    GrantedAt    time.Time
}

// GetPermissions returns all permissions from this grant
func (g *PermissionGrant) GetPermissions() []Permission {
    if g.Role != nil {
        return g.Role.Permissions()
    }
    if g.Permission != nil {
        return []Permission{*g.Permission}
    }
    return nil
}
```

### 2.2 Relationship (Zanzibar-style Tuple)

```go
// internal/domain/authz/relationship.go

package authz

import (
    "time"
    "github.com/google/uuid"
)

type Relationship struct {
    ID          uuid.UUID
    SubjectType string       // "user", "group", "folder"
    SubjectID   uuid.UUID
    Relation    RelationType
    ObjectType  string       // "file", "folder", "group"
    ObjectID    uuid.UUID
    CreatedAt   time.Time
}

// Tuple represents a relationship tuple for queries
type Tuple struct {
    SubjectType string
    SubjectID   uuid.UUID
    Relation    RelationType
    ObjectType  string
    ObjectID    uuid.UUID
}

// Resource represents a resource reference
type Resource struct {
    Type string
    ID   uuid.UUID
}
```

---

## 3. リポジトリインターフェース

### 3.1 PermissionGrantRepository

```go
// internal/domain/authz/permission_grant_repository.go

package authz

import (
    "context"
    "github.com/google/uuid"
)

type PermissionGrantRepository interface {
    // CRUD
    Create(ctx context.Context, grant *PermissionGrant) error
    FindByID(ctx context.Context, id uuid.UUID) (*PermissionGrant, error)
    Delete(ctx context.Context, id uuid.UUID) error

    // Queries
    FindByResource(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID) ([]*PermissionGrant, error)
    FindByResourceAndGrantee(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID, granteeType GranteeType, granteeID uuid.UUID) ([]*PermissionGrant, error)
    FindByResourceGranteeAndRole(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID, granteeType GranteeType, granteeID uuid.UUID, role Role) (*PermissionGrant, error)
    FindByGrantee(ctx context.Context, granteeType GranteeType, granteeID uuid.UUID) ([]*PermissionGrant, error)

    // Batch delete
    DeleteByResource(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID) error
    DeleteByResourceAndGrantee(ctx context.Context, resourceType ResourceType, resourceID uuid.UUID, granteeType GranteeType, granteeID uuid.UUID) error
}
```

### 3.2 RelationshipRepository

```go
// internal/domain/authz/relationship_repository.go

package authz

import (
    "context"
    "github.com/google/uuid"
)

type RelationshipRepository interface {
    // CRUD
    Create(ctx context.Context, rel *Relationship) error
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteByTuple(ctx context.Context, tuple Tuple) error

    // Existence check
    Exists(ctx context.Context, tuple Tuple) (bool, error)

    // Find subjects with relation to object (e.g., find users who are members of group)
    FindSubjects(ctx context.Context, objectType string, objectID uuid.UUID, relation RelationType) ([]Resource, error)

    // Find objects that subject has relation with (e.g., find groups user is member of)
    FindObjects(ctx context.Context, subjectType string, subjectID uuid.UUID, relation RelationType, objectType string) ([]uuid.UUID, error)

    // Find all relationships for an object
    FindByObject(ctx context.Context, objectType string, objectID uuid.UUID) ([]*Relationship, error)

    // Find all relationships from a subject
    FindBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) ([]*Relationship, error)

    // Find parent relationship (for hierarchy)
    FindParent(ctx context.Context, objectType string, objectID uuid.UUID) (*Resource, error)

    // Batch delete
    DeleteByObject(ctx context.Context, objectType string, objectID uuid.UUID) error
    DeleteBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) error
}
```

---

## 4. Permission Resolver

```go
// internal/domain/authz/permission_resolver.go

package authz

import (
    "context"
    "github.com/google/uuid"
)

type PermissionResolver interface {
    // HasPermission checks if user has specific permission on resource
    HasPermission(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID, permission Permission) (bool, error)

    // CollectPermissions gathers all permissions user has on resource
    CollectPermissions(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID) (*PermissionSet, error)

    // GetEffectiveRole returns the highest role user has on resource
    GetEffectiveRole(ctx context.Context, userID uuid.UUID, resourceType ResourceType, resourceID uuid.UUID) (Role, error)
}
```

### 4.1 Permission Resolver 実装

```go
// internal/infrastructure/authz/permission_resolver.go

package authz

import (
    "context"
    "github.com/google/uuid"
    "gc-storage/internal/domain/authz"
)

type PermissionResolverImpl struct {
    grantRepo        authz.PermissionGrantRepository
    relationshipRepo authz.RelationshipRepository
}

func NewPermissionResolver(
    grantRepo authz.PermissionGrantRepository,
    relationshipRepo authz.RelationshipRepository,
) *PermissionResolverImpl {
    return &PermissionResolverImpl{
        grantRepo:        grantRepo,
        relationshipRepo: relationshipRepo,
    }
}

func (r *PermissionResolverImpl) HasPermission(
    ctx context.Context,
    userID uuid.UUID,
    resourceType authz.ResourceType,
    resourceID uuid.UUID,
    permission authz.Permission,
) (bool, error) {
    permissions, err := r.CollectPermissions(ctx, userID, resourceType, resourceID)
    if err != nil {
        return false, err
    }
    return permissions.Has(permission), nil
}

func (r *PermissionResolverImpl) CollectPermissions(
    ctx context.Context,
    userID uuid.UUID,
    resourceType authz.ResourceType,
    resourceID uuid.UUID,
) (*authz.PermissionSet, error) {
    permissions := authz.NewPermissionSet()

    // Step 1: Check if user is owner (user --owner--> resource)
    isOwner, err := r.relationshipRepo.Exists(ctx, authz.Tuple{
        SubjectType: "user",
        SubjectID:   userID,
        Relation:    authz.RelationOwner,
        ObjectType:  string(resourceType),
        ObjectID:    resourceID,
    })
    if err != nil {
        return nil, err
    }
    if isOwner {
        permissions.AddAll(authz.RoleOwner.Permissions())
        return permissions, nil // Owner has all permissions
    }

    // Step 2: Direct grants (user --{role}--> resource)
    directGrants, err := r.grantRepo.FindByResourceAndGrantee(
        ctx, resourceType, resourceID, authz.GranteeTypeUser, userID,
    )
    if err != nil {
        return nil, err
    }
    for _, grant := range directGrants {
        permissions.AddAll(grant.GetPermissions())
    }

    // Step 3: Group-based grants (user --member--> group --{role}--> resource)
    groupIDs, err := r.relationshipRepo.FindObjects(
        ctx, "user", userID, authz.RelationMember, "group",
    )
    if err != nil {
        return nil, err
    }
    for _, groupID := range groupIDs {
        groupGrants, err := r.grantRepo.FindByResourceAndGrantee(
            ctx, resourceType, resourceID, authz.GranteeTypeGroup, groupID,
        )
        if err != nil {
            continue
        }
        for _, grant := range groupGrants {
            permissions.AddAll(grant.GetPermissions())
        }
    }

    // Step 4: Hierarchy-based grants (resource <--parent-- ancestor)
    if resourceType == authz.ResourceTypeFile || resourceType == authz.ResourceTypeFolder {
        ancestors, err := r.getAncestors(ctx, string(resourceType), resourceID)
        if err != nil {
            return nil, err
        }
        for _, ancestor := range ancestors {
            // Direct grants on ancestor
            ancestorGrants, err := r.grantRepo.FindByResourceAndGrantee(
                ctx, authz.ResourceType(ancestor.Type), ancestor.ID, authz.GranteeTypeUser, userID,
            )
            if err != nil {
                continue
            }
            for _, grant := range ancestorGrants {
                permissions.AddAll(grant.GetPermissions())
            }

            // Group grants on ancestor
            for _, groupID := range groupIDs {
                groupGrants, err := r.grantRepo.FindByResourceAndGrantee(
                    ctx, authz.ResourceType(ancestor.Type), ancestor.ID, authz.GranteeTypeGroup, groupID,
                )
                if err != nil {
                    continue
                }
                for _, grant := range groupGrants {
                    permissions.AddAll(grant.GetPermissions())
                }
            }
        }
    }

    return permissions, nil
}

func (r *PermissionResolverImpl) GetEffectiveRole(
    ctx context.Context,
    userID uuid.UUID,
    resourceType authz.ResourceType,
    resourceID uuid.UUID,
) (authz.Role, error) {
    // Check ownership first
    isOwner, err := r.relationshipRepo.Exists(ctx, authz.Tuple{
        SubjectType: "user",
        SubjectID:   userID,
        Relation:    authz.RelationOwner,
        ObjectType:  string(resourceType),
        ObjectID:    resourceID,
    })
    if err != nil {
        return "", err
    }
    if isOwner {
        return authz.RoleOwner, nil
    }

    // Collect all applicable roles
    highestRole := authz.Role("")
    roleHierarchy := map[authz.Role]int{
        authz.RoleViewer:  1,
        authz.RoleEditor:  2,
        authz.RoleManager: 3,
    }

    // Check direct grants
    grants, err := r.grantRepo.FindByResourceAndGrantee(
        ctx, resourceType, resourceID, authz.GranteeTypeUser, userID,
    )
    if err != nil {
        return "", err
    }
    for _, grant := range grants {
        if grant.Role != nil && roleHierarchy[*grant.Role] > roleHierarchy[highestRole] {
            highestRole = *grant.Role
        }
    }

    // Check group grants
    groupIDs, _ := r.relationshipRepo.FindObjects(ctx, "user", userID, authz.RelationMember, "group")
    for _, groupID := range groupIDs {
        groupGrants, _ := r.grantRepo.FindByResourceAndGrantee(
            ctx, resourceType, resourceID, authz.GranteeTypeGroup, groupID,
        )
        for _, grant := range groupGrants {
            if grant.Role != nil && roleHierarchy[*grant.Role] > roleHierarchy[highestRole] {
                highestRole = *grant.Role
            }
        }
    }

    // Check ancestor hierarchy
    if resourceType == authz.ResourceTypeFile || resourceType == authz.ResourceTypeFolder {
        ancestors, _ := r.getAncestors(ctx, string(resourceType), resourceID)
        for _, ancestor := range ancestors {
            ancestorGrants, _ := r.grantRepo.FindByResourceAndGrantee(
                ctx, authz.ResourceType(ancestor.Type), ancestor.ID, authz.GranteeTypeUser, userID,
            )
            for _, grant := range ancestorGrants {
                if grant.Role != nil && roleHierarchy[*grant.Role] > roleHierarchy[highestRole] {
                    highestRole = *grant.Role
                }
            }
            for _, groupID := range groupIDs {
                groupGrants, _ := r.grantRepo.FindByResourceAndGrantee(
                    ctx, authz.ResourceType(ancestor.Type), ancestor.ID, authz.GranteeTypeGroup, groupID,
                )
                for _, grant := range groupGrants {
                    if grant.Role != nil && roleHierarchy[*grant.Role] > roleHierarchy[highestRole] {
                        highestRole = *grant.Role
                    }
                }
            }
        }
    }

    return highestRole, nil
}

func (r *PermissionResolverImpl) getAncestors(
    ctx context.Context,
    resourceType string,
    resourceID uuid.UUID,
) ([]authz.Resource, error) {
    var ancestors []authz.Resource

    currentType := resourceType
    currentID := resourceID

    for {
        parent, err := r.relationshipRepo.FindParent(ctx, currentType, currentID)
        if err != nil || parent == nil {
            break
        }
        ancestors = append(ancestors, *parent)
        currentType = parent.Type
        currentID = parent.ID
    }

    return ancestors, nil
}
```

---

## 5. ユースケース

### 5.1 ロール付与

```go
// internal/usecase/authz/grant_role.go

package authz

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/authz"
    "gc-storage/pkg/apperror"
)

type GrantRoleInput struct {
    ResourceType authz.ResourceType
    ResourceID   uuid.UUID
    GranteeType  authz.GranteeType
    GranteeID    uuid.UUID
    Role         authz.Role
    GrantedBy    uuid.UUID
}

type GrantRoleOutput struct {
    Grant *authz.PermissionGrant
}

type GrantRoleUseCase struct {
    grantRepo        authz.PermissionGrantRepository
    relationshipRepo authz.RelationshipRepository
    resolver         authz.PermissionResolver
    txManager        TransactionManager
}

func NewGrantRoleUseCase(
    grantRepo authz.PermissionGrantRepository,
    relationshipRepo authz.RelationshipRepository,
    resolver authz.PermissionResolver,
    txManager TransactionManager,
) *GrantRoleUseCase {
    return &GrantRoleUseCase{
        grantRepo:        grantRepo,
        relationshipRepo: relationshipRepo,
        resolver:         resolver,
        txManager:        txManager,
    }
}

func (uc *GrantRoleUseCase) Execute(ctx context.Context, input GrantRoleInput) (*GrantRoleOutput, error) {
    // 1. Validate role
    if !input.Role.IsValid() {
        return nil, apperror.NewValidation("invalid role", nil)
    }

    // 2. Cannot grant owner role directly
    if input.Role == authz.RoleOwner {
        return nil, apperror.NewBadRequest("owner role cannot be directly granted, use ownership transfer", nil)
    }

    // 3. Verify granter has permission:grant permission
    hasGrant, err := uc.resolver.HasPermission(
        ctx, input.GrantedBy, input.ResourceType, input.ResourceID, authz.PermPermissionGrant,
    )
    if err != nil {
        return nil, err
    }
    if !hasGrant {
        return nil, apperror.NewForbidden("insufficient permission to grant roles", nil)
    }

    // 4. Verify granter's role is higher than the role being granted
    granterRole, err := uc.resolver.GetEffectiveRole(
        ctx, input.GrantedBy, input.ResourceType, input.ResourceID,
    )
    if err != nil {
        return nil, err
    }
    if !granterRole.Includes(input.Role) {
        return nil, apperror.NewForbidden("cannot grant a role higher than your own", nil)
    }

    var grant *authz.PermissionGrant

    err = uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 5. Check for existing grant
        existing, _ := uc.grantRepo.FindByResourceGranteeAndRole(
            ctx, input.ResourceType, input.ResourceID,
            input.GranteeType, input.GranteeID, input.Role,
        )
        if existing != nil {
            return apperror.NewConflict("role already granted", nil)
        }

        // 6. Create permission grant
        now := time.Now()
        grant = &authz.PermissionGrant{
            ID:           uuid.New(),
            ResourceType: input.ResourceType,
            ResourceID:   input.ResourceID,
            GranteeType:  input.GranteeType,
            GranteeID:    input.GranteeID,
            Role:         &input.Role,
            GrantedBy:    input.GrantedBy,
            GrantedAt:    now,
        }
        if err := uc.grantRepo.Create(ctx, grant); err != nil {
            return err
        }

        // 7. Create relationship tuple for the role
        relationship := &authz.Relationship{
            ID:          uuid.New(),
            SubjectType: string(input.GranteeType),
            SubjectID:   input.GranteeID,
            Relation:    authz.RelationType(input.Role),
            ObjectType:  string(input.ResourceType),
            ObjectID:    input.ResourceID,
            CreatedAt:   now,
        }
        if err := uc.relationshipRepo.Create(ctx, relationship); err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    return &GrantRoleOutput{Grant: grant}, nil
}
```

### 5.2 権限取り消し

```go
// internal/usecase/authz/revoke_grant.go

package authz

import (
    "context"
    "github.com/google/uuid"
    "gc-storage/internal/domain/authz"
    "gc-storage/pkg/apperror"
)

type RevokeGrantInput struct {
    GrantID  uuid.UUID
    ActorID  uuid.UUID
}

type RevokeGrantUseCase struct {
    grantRepo        authz.PermissionGrantRepository
    relationshipRepo authz.RelationshipRepository
    resolver         authz.PermissionResolver
    txManager        TransactionManager
}

func NewRevokeGrantUseCase(
    grantRepo authz.PermissionGrantRepository,
    relationshipRepo authz.RelationshipRepository,
    resolver authz.PermissionResolver,
    txManager TransactionManager,
) *RevokeGrantUseCase {
    return &RevokeGrantUseCase{
        grantRepo:        grantRepo,
        relationshipRepo: relationshipRepo,
        resolver:         resolver,
        txManager:        txManager,
    }
}

func (uc *RevokeGrantUseCase) Execute(ctx context.Context, input RevokeGrantInput) error {
    // 1. Get the grant
    grant, err := uc.grantRepo.FindByID(ctx, input.GrantID)
    if err != nil {
        return apperror.NewNotFound("grant not found", err)
    }

    // 2. Verify actor has permission:revoke permission
    hasRevoke, err := uc.resolver.HasPermission(
        ctx, input.ActorID, grant.ResourceType, grant.ResourceID, authz.PermPermissionRevoke,
    )
    if err != nil {
        return err
    }
    if !hasRevoke {
        return apperror.NewForbidden("insufficient permission to revoke grants", nil)
    }

    // 3. Cannot revoke owner's grant (ownership must be transferred)
    if grant.Role != nil && *grant.Role == authz.RoleOwner {
        return apperror.NewBadRequest("cannot revoke owner role, use ownership transfer", nil)
    }

    return uc.txManager.WithTransaction(ctx, func(ctx context.Context) error {
        // 4. Delete the grant
        if err := uc.grantRepo.Delete(ctx, input.GrantID); err != nil {
            return err
        }

        // 5. Delete the relationship tuple if role-based
        if grant.Role != nil {
            if err := uc.relationshipRepo.DeleteByTuple(ctx, authz.Tuple{
                SubjectType: string(grant.GranteeType),
                SubjectID:   grant.GranteeID,
                Relation:    authz.RelationType(*grant.Role),
                ObjectType:  string(grant.ResourceType),
                ObjectID:    grant.ResourceID,
            }); err != nil {
                return err
            }
        }

        return nil
    })
}
```

### 5.3 所有権設定

```go
// internal/usecase/authz/set_owner.go

package authz

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/authz"
)

type SetOwnerInput struct {
    ResourceType authz.ResourceType
    ResourceID   uuid.UUID
    OwnerID      uuid.UUID
}

type SetOwnerUseCase struct {
    relationshipRepo authz.RelationshipRepository
}

func NewSetOwnerUseCase(
    relationshipRepo authz.RelationshipRepository,
) *SetOwnerUseCase {
    return &SetOwnerUseCase{
        relationshipRepo: relationshipRepo,
    }
}

func (uc *SetOwnerUseCase) Execute(ctx context.Context, input SetOwnerInput) error {
    // Create owner relationship
    relationship := &authz.Relationship{
        ID:          uuid.New(),
        SubjectType: "user",
        SubjectID:   input.OwnerID,
        Relation:    authz.RelationOwner,
        ObjectType:  string(input.ResourceType),
        ObjectID:    input.ResourceID,
        CreatedAt:   time.Now(),
    }

    return uc.relationshipRepo.Create(ctx, relationship)
}
```

### 5.4 親子関係設定

```go
// internal/usecase/authz/set_parent.go

package authz

import (
    "context"
    "time"
    "github.com/google/uuid"
    "gc-storage/internal/domain/authz"
)

type SetParentInput struct {
    ChildType  string
    ChildID    uuid.UUID
    ParentType string
    ParentID   uuid.UUID
}

type SetParentUseCase struct {
    relationshipRepo authz.RelationshipRepository
}

func NewSetParentUseCase(
    relationshipRepo authz.RelationshipRepository,
) *SetParentUseCase {
    return &SetParentUseCase{
        relationshipRepo: relationshipRepo,
    }
}

func (uc *SetParentUseCase) Execute(ctx context.Context, input SetParentInput) error {
    // First delete any existing parent relationship
    _ = uc.relationshipRepo.DeleteByTuple(ctx, authz.Tuple{
        SubjectType: input.ParentType,
        SubjectID:   input.ParentID,
        Relation:    authz.RelationParent,
        ObjectType:  input.ChildType,
        ObjectID:    input.ChildID,
    })

    // Create new parent relationship
    relationship := &authz.Relationship{
        ID:          uuid.New(),
        SubjectType: input.ParentType,
        SubjectID:   input.ParentID,
        Relation:    authz.RelationParent,
        ObjectType:  input.ChildType,
        ObjectID:    input.ChildID,
        CreatedAt:   time.Now(),
    }

    return uc.relationshipRepo.Create(ctx, relationship)
}
```

### 5.5 権限一覧取得

```go
// internal/usecase/authz/list_grants.go

package authz

import (
    "context"
    "github.com/google/uuid"
    "gc-storage/internal/domain/authz"
    "gc-storage/pkg/apperror"
)

type ListGrantsInput struct {
    ResourceType authz.ResourceType
    ResourceID   uuid.UUID
    ActorID      uuid.UUID
}

type GrantWithUser struct {
    Grant     *authz.PermissionGrant
    UserName  string
    UserEmail string
}

type ListGrantsOutput struct {
    Grants []*GrantWithUser
}

type ListGrantsUseCase struct {
    grantRepo authz.PermissionGrantRepository
    resolver  authz.PermissionResolver
}

func NewListGrantsUseCase(
    grantRepo authz.PermissionGrantRepository,
    resolver authz.PermissionResolver,
) *ListGrantsUseCase {
    return &ListGrantsUseCase{
        grantRepo: grantRepo,
        resolver:  resolver,
    }
}

func (uc *ListGrantsUseCase) Execute(ctx context.Context, input ListGrantsInput) (*ListGrantsOutput, error) {
    // 1. Verify actor has permission:read permission
    hasRead, err := uc.resolver.HasPermission(
        ctx, input.ActorID, input.ResourceType, input.ResourceID, authz.PermPermissionRead,
    )
    if err != nil {
        return nil, err
    }
    if !hasRead {
        return nil, apperror.NewForbidden("insufficient permission to view grants", nil)
    }

    // 2. Get all grants for the resource
    grants, err := uc.grantRepo.FindByResource(ctx, input.ResourceType, input.ResourceID)
    if err != nil {
        return nil, err
    }

    // 3. TODO: Enrich with user/group names
    result := make([]*GrantWithUser, len(grants))
    for i, grant := range grants {
        result[i] = &GrantWithUser{
            Grant: grant,
            // UserName and UserEmail would be fetched from user service
        }
    }

    return &ListGrantsOutput{Grants: result}, nil
}
```

---

## 6. ミドルウェア

### 6.1 Permission Middleware

```go
// internal/interface/middleware/permission.go

package middleware

import (
    "fmt"
    "net/http"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "gc-storage/internal/domain/authz"
    "gc-storage/pkg/apperror"
)

type PermissionMiddleware struct {
    resolver authz.PermissionResolver
}

func NewPermissionMiddleware(resolver authz.PermissionResolver) *PermissionMiddleware {
    return &PermissionMiddleware{resolver: resolver}
}

// RequirePermission creates middleware that requires a specific permission
func (m *PermissionMiddleware) RequirePermission(
    resourceType authz.ResourceType,
    permission authz.Permission,
    resourceIDParam string,
) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            claims := GetClaims(c)
            if claims == nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
            }

            resourceID, err := uuid.Parse(c.Param(resourceIDParam))
            if err != nil {
                return echo.NewHTTPError(http.StatusBadRequest, "invalid resource id")
            }

            hasPermission, err := m.resolver.HasPermission(
                c.Request().Context(),
                claims.UserID,
                resourceType,
                resourceID,
                permission,
            )
            if err != nil {
                return apperror.NewInternal("permission check failed", err)
            }

            if !hasPermission {
                return echo.NewHTTPError(
                    http.StatusForbidden,
                    fmt.Sprintf("permission %s required", permission),
                )
            }

            return next(c)
        }
    }
}

// RequireAnyPermission creates middleware that requires any of the specified permissions
func (m *PermissionMiddleware) RequireAnyPermission(
    resourceType authz.ResourceType,
    permissions []authz.Permission,
    resourceIDParam string,
) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            claims := GetClaims(c)
            if claims == nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
            }

            resourceID, err := uuid.Parse(c.Param(resourceIDParam))
            if err != nil {
                return echo.NewHTTPError(http.StatusBadRequest, "invalid resource id")
            }

            permSet, err := m.resolver.CollectPermissions(
                c.Request().Context(),
                claims.UserID,
                resourceType,
                resourceID,
            )
            if err != nil {
                return apperror.NewInternal("permission check failed", err)
            }

            if !permSet.HasAny(permissions) {
                return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
            }

            return next(c)
        }
    }
}

// RequireOwner creates middleware that requires ownership
func (m *PermissionMiddleware) RequireOwner(
    resourceType authz.ResourceType,
    resourceIDParam string,
) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            claims := GetClaims(c)
            if claims == nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
            }

            resourceID, err := uuid.Parse(c.Param(resourceIDParam))
            if err != nil {
                return echo.NewHTTPError(http.StatusBadRequest, "invalid resource id")
            }

            role, err := m.resolver.GetEffectiveRole(
                c.Request().Context(),
                claims.UserID,
                resourceType,
                resourceID,
            )
            if err != nil {
                return apperror.NewInternal("permission check failed", err)
            }

            if role != authz.RoleOwner {
                return echo.NewHTTPError(http.StatusForbidden, "owner permission required")
            }

            return next(c)
        }
    }
}
```

---

## 7. ハンドラー

```go
// internal/interface/handler/permission_handler.go

package handler

import (
    "net/http"
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "gc-storage/internal/domain/authz"
    "gc-storage/internal/interface/dto"
    "gc-storage/internal/interface/middleware"
    usecase "gc-storage/internal/usecase/authz"
)

type PermissionHandler struct {
    grantRole   *usecase.GrantRoleUseCase
    revokeGrant *usecase.RevokeGrantUseCase
    listGrants  *usecase.ListGrantsUseCase
}

// POST /api/v1/files/:id/permissions or /api/v1/folders/:id/permissions
func (h *PermissionHandler) GrantRole(c echo.Context) error {
    resourceType := authz.ResourceType(c.Param("type"))
    resourceID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid resource id")
    }

    var req dto.GrantRoleRequest
    if err := c.Bind(&req); err != nil {
        return err
    }
    if err := c.Validate(&req); err != nil {
        return err
    }

    claims := middleware.GetClaims(c)
    output, err := h.grantRole.Execute(c.Request().Context(), usecase.GrantRoleInput{
        ResourceType: resourceType,
        ResourceID:   resourceID,
        GranteeType:  authz.GranteeType(req.GranteeType),
        GranteeID:    req.GranteeID,
        Role:         authz.Role(req.Role),
        GrantedBy:    claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, dto.PermissionGrantResponse{
        ID:           output.Grant.ID,
        GranteeType:  string(output.Grant.GranteeType),
        GranteeID:    output.Grant.GranteeID,
        Role:         string(*output.Grant.Role),
        GrantedAt:    output.Grant.GrantedAt,
    })
}

// DELETE /api/v1/permissions/:id
func (h *PermissionHandler) RevokeGrant(c echo.Context) error {
    grantID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid grant id")
    }

    claims := middleware.GetClaims(c)
    err = h.revokeGrant.Execute(c.Request().Context(), usecase.RevokeGrantInput{
        GrantID: grantID,
        ActorID: claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.NoContent(http.StatusNoContent)
}

// GET /api/v1/files/:id/permissions or /api/v1/folders/:id/permissions
func (h *PermissionHandler) ListGrants(c echo.Context) error {
    resourceType := authz.ResourceType(c.Param("type"))
    resourceID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid resource id")
    }

    claims := middleware.GetClaims(c)
    output, err := h.listGrants.Execute(c.Request().Context(), usecase.ListGrantsInput{
        ResourceType: resourceType,
        ResourceID:   resourceID,
        ActorID:      claims.UserID,
    })
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, dto.PermissionListResponse{
        Grants: dto.ToPermissionGrantResponses(output.Grants),
    })
}
```

---

## 8. DTO定義

```go
// internal/interface/dto/permission.go

package dto

import (
    "time"
    "github.com/google/uuid"
)

type GrantRoleRequest struct {
    GranteeType string    `json:"grantee_type" validate:"required,oneof=user group"`
    GranteeID   uuid.UUID `json:"grantee_id" validate:"required"`
    Role        string    `json:"role" validate:"required,oneof=viewer editor manager"`
}

type PermissionGrantResponse struct {
    ID          uuid.UUID `json:"id"`
    GranteeType string    `json:"grantee_type"`
    GranteeID   uuid.UUID `json:"grantee_id"`
    GranteeName string    `json:"grantee_name,omitempty"`
    Role        string    `json:"role,omitempty"`
    Permission  string    `json:"permission,omitempty"`
    GrantedAt   time.Time `json:"granted_at"`
}

type PermissionListResponse struct {
    Grants []PermissionGrantResponse `json:"grants"`
}
```

---

## 9. APIエンドポイント

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | /api/v1/files/:id/permissions | PermissionHandler.ListGrants | ファイル権限一覧 |
| POST | /api/v1/files/:id/permissions | PermissionHandler.GrantRole | ファイル権限付与 |
| DELETE | /api/v1/permissions/:id | PermissionHandler.RevokeGrant | 権限取り消し |
| GET | /api/v1/folders/:id/permissions | PermissionHandler.ListGrants | フォルダ権限一覧 |
| POST | /api/v1/folders/:id/permissions | PermissionHandler.GrantRole | フォルダ権限付与 |

---

## 10. ルーティング設定例

```go
// internal/interface/router/router.go

func SetupRoutes(e *echo.Echo, h *Handlers, m *Middlewares) {
    api := e.Group("/api/v1")
    api.Use(m.Auth.RequireAuth())

    // Files with permission checks
    files := api.Group("/files")
    files.GET("/:id", h.File.Get,
        m.Permission.RequirePermission(authz.ResourceTypeFile, authz.PermFileRead, "id"))
    files.PUT("/:id", h.File.Update,
        m.Permission.RequirePermission(authz.ResourceTypeFile, authz.PermFileWrite, "id"))
    files.PATCH("/:id/rename", h.File.Rename,
        m.Permission.RequirePermission(authz.ResourceTypeFile, authz.PermFileRename, "id"))
    files.DELETE("/:id", h.File.Trash,
        m.Permission.RequirePermission(authz.ResourceTypeFile, authz.PermFileDelete, "id"))
    files.DELETE("/:id/permanent", h.File.PermanentDelete,
        m.Permission.RequirePermission(authz.ResourceTypeFile, authz.PermFilePermanentDelete, "id"))
    files.POST("/:id/share", h.File.CreateShareLink,
        m.Permission.RequirePermission(authz.ResourceTypeFile, authz.PermFileShare, "id"))

    // File permissions
    files.GET("/:id/permissions", h.Permission.ListGrants,
        m.Permission.RequirePermission(authz.ResourceTypeFile, authz.PermPermissionRead, "id"))
    files.POST("/:id/permissions", h.Permission.GrantRole,
        m.Permission.RequirePermission(authz.ResourceTypeFile, authz.PermPermissionGrant, "id"))

    // Folders with permission checks
    folders := api.Group("/folders")
    folders.GET("/:id", h.Folder.Get,
        m.Permission.RequirePermission(authz.ResourceTypeFolder, authz.PermFolderRead, "id"))
    folders.POST("/:id/folders", h.Folder.Create,
        m.Permission.RequirePermission(authz.ResourceTypeFolder, authz.PermFolderCreate, "id"))
    folders.DELETE("/:id", h.Folder.Trash,
        m.Permission.RequirePermission(authz.ResourceTypeFolder, authz.PermFolderDelete, "id"))

    // Folder permissions
    folders.GET("/:id/permissions", h.Permission.ListGrants,
        m.Permission.RequirePermission(authz.ResourceTypeFolder, authz.PermPermissionRead, "id"))
    folders.POST("/:id/permissions", h.Permission.GrantRole,
        m.Permission.RequirePermission(authz.ResourceTypeFolder, authz.PermPermissionGrant, "id"))

    // Permission management
    permissions := api.Group("/permissions")
    permissions.DELETE("/:id", h.Permission.RevokeGrant)
}
```

---

## 11. 受け入れ基準

### 権限付与
- [ ] viewer/editor/managerロールを付与できる
- [ ] ownerロールは直接付与できない（所有権譲渡を使用）
- [ ] 自分より高いロールは付与できない
- [ ] permission:grant権限がないと付与できない
- [ ] 同じロールの重複付与は拒否される

### 権限取り消し
- [ ] 付与した権限を取り消せる
- [ ] permission:revoke権限がないと取り消せない
- [ ] ownerロールは取り消せない

### 権限継承
- [ ] 親フォルダの権限が子フォルダ・ファイルに継承される
- [ ] グループ経由の権限が解決される
- [ ] 複数経路からの権限が合算される

### 権限解決
- [ ] ownerは全権限を持つ
- [ ] 直接付与 > グループ経由 > 階層経由の順で解決
- [ ] 最も高いロールが有効になる

### ミドルウェア
- [ ] RequirePermissionで特定権限をチェックできる
- [ ] RequireAnyPermissionで複数権限のいずれかをチェックできる
- [ ] RequireOwnerで所有者のみ許可できる

---

## 関連ドキュメント

- [権限ドメイン](../03-domains/permission.md)
- [セキュリティ設計](../02-architecture/SECURITY.md)
- [Storage Core仕様](./storage-core.md)
- [Collab Group仕様](./collab-group.md)
