package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/repository"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// DeclineInvitationInput は招待辞退の入力を定義します
type DeclineInvitationInput struct {
	Token  string
	UserID uuid.UUID
}

// DeclineInvitationCommand は招待辞退コマンドです
type DeclineInvitationCommand struct {
	invitationRepo repository.InvitationRepository
	userRepo       repository.UserRepository
}

// NewDeclineInvitationCommand は新しいDeclineInvitationCommandを作成します
func NewDeclineInvitationCommand(
	invitationRepo repository.InvitationRepository,
	userRepo repository.UserRepository,
) *DeclineInvitationCommand {
	return &DeclineInvitationCommand{
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
	}
}

// Execute は招待辞退を実行します
func (c *DeclineInvitationCommand) Execute(ctx context.Context, input DeclineInvitationInput) error {
	// 1. 招待の取得
	invitation, err := c.invitationRepo.FindByToken(ctx, input.Token)
	if err != nil {
		return apperror.NewNotFoundError("invitation")
	}

	// 2. ユーザーの取得
	user, err := c.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return apperror.NewNotFoundError("user")
	}

	// 3. メールアドレスの一致確認
	if !invitation.IsForEmail(user.Email) {
		return apperror.NewForbiddenError("invitation is not for this user")
	}

	// 4. 招待の辞退
	if err := invitation.Decline(); err != nil {
		if err == entity.ErrInvitationExpired {
			return apperror.NewValidationError("invitation has expired", nil)
		}
		if err == entity.ErrInvitationNotPending {
			return apperror.NewValidationError("invitation is not pending", nil)
		}
		return err
	}

	// 5. 更新を保存
	return c.invitationRepo.Update(ctx, invitation)
}
