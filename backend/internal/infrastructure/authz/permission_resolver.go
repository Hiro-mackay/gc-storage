package authz

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/authz"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
)

// PermissionResolverImpl は権限解決サービスの実装です
// 権限解決の優先順位:
// 1. オーナーシップ（所有者は全ての権限を持つ）
// 2. 直接付与された権限
// 3. グループ経由の権限
// 4. 親リソースからの継承（フォルダ階層）
type PermissionResolverImpl struct {
	permissionGrantRepo authz.PermissionGrantRepository
	relationshipRepo    authz.RelationshipRepository
	membershipRepo      repository.MembershipRepository
}

// NewPermissionResolver は新しいPermissionResolverを作成します
func NewPermissionResolver(
	permissionGrantRepo authz.PermissionGrantRepository,
	relationshipRepo authz.RelationshipRepository,
	membershipRepo repository.MembershipRepository,
) *PermissionResolverImpl {
	return &PermissionResolverImpl{
		permissionGrantRepo: permissionGrantRepo,
		relationshipRepo:    relationshipRepo,
		membershipRepo:      membershipRepo,
	}
}

// HasPermission はユーザーがリソースに対して指定された権限を持つかを判定します
func (r *PermissionResolverImpl) HasPermission(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID, permission authz.Permission) (bool, error) {
	// 1. オーナーチェック
	isOwner, err := r.IsOwner(ctx, userID, resourceType, resourceID)
	if err != nil {
		return false, err
	}
	if isOwner {
		return true, nil
	}

	// 2. 権限セットを収集して判定
	permissionSet, err := r.CollectPermissions(ctx, userID, resourceType, resourceID)
	if err != nil {
		return false, err
	}

	return permissionSet.Has(permission), nil
}

// CollectPermissions はユーザーがリソースに対して持つ権限を全て取得します
func (r *PermissionResolverImpl) CollectPermissions(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) (*authz.PermissionSet, error) {
	permissionSet := authz.EmptyPermissionSet()

	// 1. オーナーチェック
	isOwner, err := r.IsOwner(ctx, userID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	if isOwner {
		// オーナーは全ての権限を持つ
		permissionSet.AddFromRole(authz.RoleOwner)
		return permissionSet, nil
	}

	// 2. 直接付与された権限を収集
	directGrants, err := r.permissionGrantRepo.FindByResourceAndGrantee(ctx, resourceType, resourceID, authz.GranteeTypeUser, userID)
	if err != nil {
		return nil, err
	}
	for _, grant := range directGrants {
		permissionSet.AddFromRole(grant.Role)
	}

	// 3. グループ経由の権限を収集
	groupGrants, err := r.collectGroupPermissions(ctx, userID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	for _, grant := range groupGrants {
		permissionSet.AddFromRole(grant.Role)
	}

	// 4. 親リソースからの継承（再帰的）
	parentPermissions, err := r.collectParentPermissions(ctx, userID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	permissionSet = permissionSet.Union(parentPermissions)

	return permissionSet, nil
}

// GetEffectiveRole はユーザーがリソースに対して持つ最も高いロールを取得します
func (r *PermissionResolverImpl) GetEffectiveRole(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) (authz.Role, error) {
	// 1. オーナーチェック
	isOwner, err := r.IsOwner(ctx, userID, resourceType, resourceID)
	if err != nil {
		return "", err
	}
	if isOwner {
		return authz.RoleOwner, nil
	}

	highestRole := authz.Role("")

	// 2. 直接付与されたロールを確認
	directGrants, err := r.permissionGrantRepo.FindByResourceAndGrantee(ctx, resourceType, resourceID, authz.GranteeTypeUser, userID)
	if err != nil {
		return "", err
	}
	for _, grant := range directGrants {
		if grant.Role.Level() > highestRole.Level() {
			highestRole = grant.Role
		}
	}

	// 3. グループ経由のロールを確認
	groupGrants, err := r.collectGroupPermissions(ctx, userID, resourceType, resourceID)
	if err != nil {
		return "", err
	}
	for _, grant := range groupGrants {
		if grant.Role.Level() > highestRole.Level() {
			highestRole = grant.Role
		}
	}

	// 4. 親リソースからの継承
	parentRole, err := r.getParentEffectiveRole(ctx, userID, resourceType, resourceID)
	if err != nil {
		return "", err
	}
	if parentRole.Level() > highestRole.Level() {
		highestRole = parentRole
	}

	return highestRole, nil
}

// IsOwner はユーザーがリソースのオーナーかを判定します
func (r *PermissionResolverImpl) IsOwner(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) (bool, error) {
	tuple := authz.NewTuple(
		authz.SubjectTypeUser,
		userID,
		authz.RelationOwner,
		authz.ObjectType(resourceType),
		resourceID,
	)
	return r.relationshipRepo.Exists(ctx, tuple)
}

// CanGrantRole はユーザーがリソースに対して指定されたロールを付与可能かを判定します
func (r *PermissionResolverImpl) CanGrantRole(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID, targetRole authz.Role) (bool, error) {
	effectiveRole, err := r.GetEffectiveRole(ctx, userID, resourceType, resourceID)
	if err != nil {
		return false, err
	}
	return effectiveRole.CanGrant(targetRole), nil
}

// collectGroupPermissions はユーザーが所属するグループ経由で付与された権限を収集します
func (r *PermissionResolverImpl) collectGroupPermissions(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) ([]*authz.PermissionGrant, error) {
	// ユーザーが所属するグループを取得
	memberships, err := r.membershipRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var grants []*authz.PermissionGrant
	for _, membership := range memberships {
		groupGrants, err := r.permissionGrantRepo.FindByResourceAndGrantee(ctx, resourceType, resourceID, authz.GranteeTypeGroup, membership.GroupID)
		if err != nil {
			return nil, err
		}
		grants = append(grants, groupGrants...)
	}

	return grants, nil
}

// collectParentPermissions は親リソースからの継承権限を収集します
func (r *PermissionResolverImpl) collectParentPermissions(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) (*authz.PermissionSet, error) {
	permissionSet := authz.EmptyPermissionSet()

	// 親リソースを取得
	parent, err := r.relationshipRepo.FindParent(ctx, authz.ObjectType(resourceType), resourceID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return permissionSet, nil
	}

	// 親リソースの権限を再帰的に収集
	return r.CollectPermissions(ctx, userID, parent.Type, parent.ID)
}

// getParentEffectiveRole は親リソースからの有効ロールを取得します
func (r *PermissionResolverImpl) getParentEffectiveRole(ctx context.Context, userID uuid.UUID, resourceType authz.ResourceType, resourceID uuid.UUID) (authz.Role, error) {
	// 親リソースを取得
	parent, err := r.relationshipRepo.FindParent(ctx, authz.ObjectType(resourceType), resourceID)
	if err != nil {
		return "", err
	}
	if parent == nil {
		return "", nil
	}

	// 親リソースの有効ロールを再帰的に取得
	return r.GetEffectiveRole(ctx, userID, parent.Type, parent.ID)
}

// インターフェースの実装を保証
var _ authz.PermissionResolver = (*PermissionResolverImpl)(nil)
