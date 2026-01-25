package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// InviteMemberInput は招待作成の入力を定義します
type InviteMemberInput struct {
	GroupID   uuid.UUID
	Email     string
	Role      string
	InvitedBy uuid.UUID
}

// InviteMemberOutput は招待作成の出力を定義します
type InviteMemberOutput struct {
	Invitation *entity.Invitation
}

// InviteMemberCommand は招待作成コマンドです
type InviteMemberCommand struct {
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
	invitationRepo repository.InvitationRepository
	userRepo       repository.UserRepository
}

// NewInviteMemberCommand は新しいInviteMemberCommandを作成します
func NewInviteMemberCommand(
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
	invitationRepo repository.InvitationRepository,
	userRepo repository.UserRepository,
) *InviteMemberCommand {
	return &InviteMemberCommand{
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
	}
}

// Execute は招待作成を実行します
func (c *InviteMemberCommand) Execute(ctx context.Context, input InviteMemberInput) (*InviteMemberOutput, error) {
	// 1. メールアドレスのバリデーション
	email, err := valueobject.NewEmail(input.Email)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 2. ロールのバリデーション
	role, err := valueobject.NewGroupRole(input.Role)
	if err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}
	// Ownerは招待時に指定不可
	if role.IsOwner() {
		return nil, apperror.NewValidationError("cannot invite as owner", nil)
	}

	// 3. グループの存在確認
	_, err = c.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	// 4. 招待者の権限チェック
	inviterMembership, err := c.membershipRepo.FindByGroupAndUser(ctx, input.GroupID, input.InvitedBy)
	if err != nil {
		return nil, apperror.NewForbiddenError("you are not a member of this group")
	}
	if !inviterMembership.CanInvite() {
		return nil, apperror.NewForbiddenError("you do not have permission to invite members")
	}
	// 自分より高いロールは付与不可
	if !inviterMembership.CanChangeRoleTo(role) {
		return nil, apperror.NewForbiddenError("you cannot assign a role higher than your own")
	}

	// 5. 既にメンバーかどうかチェック（メールアドレスで既存ユーザーを検索）
	existingUser, err := c.userRepo.FindByEmail(ctx, email)
	if err == nil && existingUser != nil {
		membershipExists, err := c.membershipRepo.Exists(ctx, input.GroupID, existingUser.ID)
		if err != nil {
			return nil, err
		}
		if membershipExists {
			return nil, apperror.NewConflictError("user is already a member of this group")
		}
	}

	// 6. 既存の保留中招待チェック
	existingInvitation, err := c.invitationRepo.FindPendingByGroupAndEmail(ctx, input.GroupID, email)
	if err == nil && existingInvitation != nil {
		return nil, apperror.NewConflictError("invitation already exists for this email")
	}

	// 7. 招待エンティティの作成
	invitation, err := entity.NewInvitation(input.GroupID, email, role, input.InvitedBy)
	if err != nil {
		return nil, apperror.NewInternalError(err)
	}

	// 8. 招待を保存
	if err := c.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, err
	}

	return &InviteMemberOutput{Invitation: invitation}, nil
}
