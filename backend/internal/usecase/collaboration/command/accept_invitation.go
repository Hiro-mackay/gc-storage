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
	invitationRepo repository.InvitationRepository
	groupRepo      repository.GroupRepository
	membershipRepo repository.MembershipRepository
	userRepo       repository.UserRepository
	txManager      repository.TransactionManager
}

// NewAcceptInvitationCommand は新しいAcceptInvitationCommandを作成します
func NewAcceptInvitationCommand(
	invitationRepo repository.InvitationRepository,
	groupRepo repository.GroupRepository,
	membershipRepo repository.MembershipRepository,
	userRepo repository.UserRepository,
	txManager repository.TransactionManager,
) *AcceptInvitationCommand {
	return &AcceptInvitationCommand{
		invitationRepo: invitationRepo,
		groupRepo:      groupRepo,
		membershipRepo: membershipRepo,
		userRepo:       userRepo,
		txManager:      txManager,
	}
}

// Execute は招待承諾を実行します
func (c *AcceptInvitationCommand) Execute(ctx context.Context, input AcceptInvitationInput) (*AcceptInvitationOutput, error) {
	// 1. 招待の取得
	invitation, err := c.invitationRepo.FindByToken(ctx, input.Token)
	if err != nil {
		return nil, err
	}

	// 2. 招待の有効性チェック
	if err := invitation.CanRespond(); err != nil {
		return nil, apperror.NewValidationError(err.Error(), nil)
	}

	// 3. ユーザーのメールアドレスが招待と一致するか確認
	user, err := c.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if !invitation.IsForEmail(user.Email) {
		return nil, apperror.NewForbiddenError("this invitation is not for you")
	}

	// 4. グループの存在確認
	group, err := c.groupRepo.FindByID(ctx, invitation.GroupID)
	if err != nil {
		return nil, err
	}

	// 5. 既にメンバーかどうかチェック
	membershipExists, err := c.membershipRepo.Exists(ctx, invitation.GroupID, input.UserID)
	if err != nil {
		return nil, err
	}
	if membershipExists {
		return nil, apperror.NewConflictError("you are already a member of this group")
	}

	// 6. 招待を承諾してメンバーシップを作成
	membership := entity.NewMembership(invitation.GroupID, input.UserID, invitation.Role)

	err = c.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// 招待を承諾済みに更新
		if err := invitation.Accept(); err != nil {
			return err
		}
		if err := c.invitationRepo.Update(ctx, invitation); err != nil {
			return err
		}

		// メンバーシップを作成
		if err := c.membershipRepo.Create(ctx, membership); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &AcceptInvitationOutput{
		Membership: membership,
		Group:      group,
	}, nil
}
