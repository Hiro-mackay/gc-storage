package handler

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	collabcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/command"
	collabqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GroupHandler はグループ関連のHTTPハンドラーです
type GroupHandler struct {
	// Commands
	createGroupCmd           *collabcmd.CreateGroupCommand
	updateGroupCmd           *collabcmd.UpdateGroupCommand
	deleteGroupCmd           *collabcmd.DeleteGroupCommand
	inviteMemberCmd          *collabcmd.InviteMemberCommand
	acceptInvitationCmd      *collabcmd.AcceptInvitationCommand
	declineInvitationCmd     *collabcmd.DeclineInvitationCommand
	cancelInvitationCmd      *collabcmd.CancelInvitationCommand
	removeMemberCmd          *collabcmd.RemoveMemberCommand
	leaveGroupCmd            *collabcmd.LeaveGroupCommand
	changeRoleCmd            *collabcmd.ChangeRoleCommand
	transferOwnershipCmd     *collabcmd.TransferOwnershipCommand

	// Queries
	getGroupQuery               *collabqry.GetGroupQuery
	listMyGroupsQuery           *collabqry.ListMyGroupsQuery
	listMembersQuery            *collabqry.ListMembersQuery
	listInvitationsQuery        *collabqry.ListInvitationsQuery
	listPendingInvitationsQuery *collabqry.ListPendingInvitationsQuery
}

// NewGroupHandler は新しいGroupHandlerを作成します
func NewGroupHandler(
	createGroupCmd *collabcmd.CreateGroupCommand,
	updateGroupCmd *collabcmd.UpdateGroupCommand,
	deleteGroupCmd *collabcmd.DeleteGroupCommand,
	inviteMemberCmd *collabcmd.InviteMemberCommand,
	acceptInvitationCmd *collabcmd.AcceptInvitationCommand,
	declineInvitationCmd *collabcmd.DeclineInvitationCommand,
	cancelInvitationCmd *collabcmd.CancelInvitationCommand,
	removeMemberCmd *collabcmd.RemoveMemberCommand,
	leaveGroupCmd *collabcmd.LeaveGroupCommand,
	changeRoleCmd *collabcmd.ChangeRoleCommand,
	transferOwnershipCmd *collabcmd.TransferOwnershipCommand,
	getGroupQuery *collabqry.GetGroupQuery,
	listMyGroupsQuery *collabqry.ListMyGroupsQuery,
	listMembersQuery *collabqry.ListMembersQuery,
	listInvitationsQuery *collabqry.ListInvitationsQuery,
	listPendingInvitationsQuery *collabqry.ListPendingInvitationsQuery,
) *GroupHandler {
	return &GroupHandler{
		createGroupCmd:              createGroupCmd,
		updateGroupCmd:              updateGroupCmd,
		deleteGroupCmd:              deleteGroupCmd,
		inviteMemberCmd:             inviteMemberCmd,
		acceptInvitationCmd:         acceptInvitationCmd,
		declineInvitationCmd:        declineInvitationCmd,
		cancelInvitationCmd:         cancelInvitationCmd,
		removeMemberCmd:             removeMemberCmd,
		leaveGroupCmd:               leaveGroupCmd,
		changeRoleCmd:               changeRoleCmd,
		transferOwnershipCmd:        transferOwnershipCmd,
		getGroupQuery:               getGroupQuery,
		listMyGroupsQuery:           listMyGroupsQuery,
		listMembersQuery:            listMembersQuery,
		listInvitationsQuery:        listInvitationsQuery,
		listPendingInvitationsQuery: listPendingInvitationsQuery,
	}
}

// CreateGroup はグループを作成します
// POST /api/v1/groups
func (h *GroupHandler) CreateGroup(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	var req request.CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.createGroupCmd.Execute(c.Request().Context(), collabcmd.CreateGroupInput{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.Created(c, response.ToGroupWithMembershipAndCountResponse(output.Group, output.Membership, 1))
}

// ListMyGroups はユーザーが所属するグループ一覧を取得します
// GET /api/v1/groups
func (h *GroupHandler) ListMyGroups(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	output, err := h.listMyGroupsQuery.Execute(c.Request().Context(), collabqry.ListMyGroupsInput{
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToGroupListResponse(output.Groups))
}

// GetGroup はグループを取得します
// GET /api/v1/groups/:id
func (h *GroupHandler) GetGroup(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	output, err := h.getGroupQuery.Execute(c.Request().Context(), collabqry.GetGroupInput{
		GroupID: groupID,
		UserID:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToGroupWithMembershipAndCountResponse(output.Group, output.Membership, output.MemberCount))
}

// DeleteGroup はグループを削除します
// DELETE /api/v1/groups/:id
func (h *GroupHandler) DeleteGroup(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	_, err = h.deleteGroupCmd.Execute(c.Request().Context(), collabcmd.DeleteGroupInput{
		GroupID:   groupID,
		DeletedBy: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}

// InviteMember はメンバーを招待します
// POST /api/v1/groups/:id/invitations
func (h *GroupHandler) InviteMember(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	var req request.InviteMemberRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.inviteMemberCmd.Execute(c.Request().Context(), collabcmd.InviteMemberInput{
		GroupID:   groupID,
		Email:     req.Email,
		Role:      req.Role,
		InvitedBy: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.Created(c, response.ToInvitationResponse(output.Invitation))
}

// AcceptInvitation は招待を承諾します
// POST /api/v1/invitations/:token/accept
func (h *GroupHandler) AcceptInvitation(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	token := c.Param("token")
	if token == "" {
		return apperror.NewValidationError("invalid invitation token", nil)
	}

	output, err := h.acceptInvitationCmd.Execute(c.Request().Context(), collabcmd.AcceptInvitationInput{
		Token:  token,
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToGroupWithMembershipAndCountResponse(output.Group, output.Membership, 0))
}

// ListMembers はグループメンバー一覧を取得します
// GET /api/v1/groups/:id/members
func (h *GroupHandler) ListMembers(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	output, err := h.listMembersQuery.Execute(c.Request().Context(), collabqry.ListMembersInput{
		GroupID: groupID,
		UserID:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToMemberListResponse(output.Members))
}

// RemoveMember はメンバーを削除します
// DELETE /api/v1/groups/:id/members/:userId
func (h *GroupHandler) RemoveMember(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return apperror.NewValidationError("invalid user ID", nil)
	}

	_, err = h.removeMemberCmd.Execute(c.Request().Context(), collabcmd.RemoveMemberInput{
		GroupID:      groupID,
		TargetUserID: targetUserID,
		RemovedBy:    claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}

// LeaveGroup はグループから退出します
// POST /api/v1/groups/:id/leave
func (h *GroupHandler) LeaveGroup(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	_, err = h.leaveGroupCmd.Execute(c.Request().Context(), collabcmd.LeaveGroupInput{
		GroupID: groupID,
		UserID:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}

// ChangeRole はメンバーのロールを変更します
// PATCH /api/v1/groups/:id/members/:userId/role
func (h *GroupHandler) ChangeRole(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return apperror.NewValidationError("invalid user ID", nil)
	}

	var req request.ChangeRoleRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.changeRoleCmd.Execute(c.Request().Context(), collabcmd.ChangeRoleInput{
		GroupID:      groupID,
		TargetUserID: targetUserID,
		NewRole:      req.Role,
		ChangedBy:    claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToMembershipResponse(output.Membership))
}

// TransferOwnership は所有権を譲渡します
// POST /api/v1/groups/:id/transfer
func (h *GroupHandler) TransferOwnership(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	var req request.TransferOwnershipRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	newOwnerID, err := uuid.Parse(req.NewOwnerID)
	if err != nil {
		return apperror.NewValidationError("invalid new owner ID", nil)
	}

	output, err := h.transferOwnershipCmd.Execute(c.Request().Context(), collabcmd.TransferOwnershipInput{
		GroupID:        groupID,
		NewOwnerID:     newOwnerID,
		CurrentOwnerID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToGroupResponse(output.Group))
}

// UpdateGroup はグループを更新します
// PATCH /api/v1/groups/:id
func (h *GroupHandler) UpdateGroup(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	var req request.UpdateGroupRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.updateGroupCmd.Execute(c.Request().Context(), collabcmd.UpdateGroupInput{
		GroupID:     groupID,
		Name:        req.Name,
		Description: req.Description,
		UpdatedBy:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToGroupResponse(output.Group))
}

// DeclineInvitation は招待を辞退します
// POST /api/v1/invitations/:token/decline
func (h *GroupHandler) DeclineInvitation(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	token := c.Param("token")
	if token == "" {
		return apperror.NewValidationError("invalid invitation token", nil)
	}

	err := h.declineInvitationCmd.Execute(c.Request().Context(), collabcmd.DeclineInvitationInput{
		Token:  token,
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}

// CancelInvitation は招待をキャンセルします
// DELETE /api/v1/groups/:id/invitations/:invitationId
func (h *GroupHandler) CancelInvitation(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		return apperror.NewValidationError("invalid invitation ID", nil)
	}

	err = h.cancelInvitationCmd.Execute(c.Request().Context(), collabcmd.CancelInvitationInput{
		InvitationID: invitationID,
		GroupID:      groupID,
		CancelledBy:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}

// ListInvitations はグループの招待一覧を取得します
// GET /api/v1/groups/:id/invitations
func (h *GroupHandler) ListInvitations(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid group ID", nil)
	}

	output, err := h.listInvitationsQuery.Execute(c.Request().Context(), collabqry.ListInvitationsInput{
		GroupID:   groupID,
		RequestBy: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToInvitationListResponse(output.Invitations))
}

// ListPendingInvitations はユーザー宛の保留中招待一覧を取得します
// GET /api/v1/invitations/pending
func (h *GroupHandler) ListPendingInvitations(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	output, err := h.listPendingInvitationsQuery.Execute(c.Request().Context(), collabqry.ListPendingInvitationsInput{
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToPendingInvitationListResponse(output.Invitations))
}
