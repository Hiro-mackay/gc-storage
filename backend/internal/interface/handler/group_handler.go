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
// @Summary グループ作成
// @Description 新しいグループを作成します
// @Tags Groups
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param body body request.CreateGroupRequest true "グループ情報"
// @Success 201 {object} handler.SwaggerGroupWithMembershipResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /groups [post]
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
// @Summary 所属グループ一覧取得
// @Description 認証ユーザーが所属するグループの一覧を取得します
// @Tags Groups
// @Produce json
// @Security SessionCookie
// @Success 200 {object} handler.SwaggerGroupListResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /groups [get]
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
// @Summary グループ取得
// @Description 指定されたグループの詳細情報を取得します
// @Tags Groups
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Success 200 {object} handler.SwaggerGroupWithMembershipResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /groups/{id} [get]
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
// @Summary グループ削除
// @Description 指定されたグループを削除します
// @Tags Groups
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /groups/{id} [delete]
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
// @Summary メンバー招待
// @Description グループにメンバーを招待します
// @Tags Invitations
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Param body body request.InviteMemberRequest true "招待情報"
// @Success 201 {object} handler.SwaggerInvitationResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 409 {object} handler.SwaggerErrorResponse
// @Router /groups/{id}/invitations [post]
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
// @Summary 招待承諾
// @Description 招待トークンを使用して招待を承諾します
// @Tags Invitations
// @Produce json
// @Security SessionCookie
// @Param token path string true "招待トークン"
// @Success 200 {object} handler.SwaggerGroupWithMembershipResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /invitations/{token}/accept [post]
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
// @Summary メンバー一覧取得
// @Description グループのメンバー一覧を取得します
// @Tags Groups
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Success 200 {object} handler.SwaggerMemberListResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Router /groups/{id}/members [get]
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
// @Summary メンバー削除
// @Description グループからメンバーを削除します
// @Tags Groups
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Param userId path string true "ユーザーID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /groups/{id}/members/{userId} [delete]
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
// @Summary グループ退出
// @Description 認証ユーザーがグループから退出します
// @Tags Groups
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Router /groups/{id}/leave [post]
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
// @Summary ロール変更
// @Description グループメンバーのロールを変更します
// @Tags Groups
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Param userId path string true "ユーザーID" format(uuid)
// @Param body body request.ChangeRoleRequest true "ロール情報"
// @Success 200 {object} handler.SwaggerMembershipResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /groups/{id}/members/{userId}/role [patch]
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
// @Summary 所有権譲渡
// @Description グループの所有権を他のメンバーに譲渡します
// @Tags Groups
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Param body body request.TransferOwnershipRequest true "譲渡先情報"
// @Success 200 {object} handler.SwaggerGroupResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /groups/{id}/transfer [post]
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
// @Summary グループ更新
// @Description グループの名前や説明を更新します
// @Tags Groups
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Param body body request.UpdateGroupRequest true "更新情報"
// @Success 200 {object} handler.SwaggerGroupResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /groups/{id} [patch]
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
// @Summary 招待辞退
// @Description 招待トークンを使用して招待を辞退します
// @Tags Invitations
// @Produce json
// @Security SessionCookie
// @Param token path string true "招待トークン"
// @Success 204 "No Content"
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /invitations/{token}/decline [post]
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
// @Summary 招待キャンセル
// @Description グループへの招待をキャンセルします
// @Tags Invitations
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Param invitationId path string true "招待ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /groups/{id}/invitations/{invitationId} [delete]
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
// @Summary 招待一覧取得
// @Description グループの招待一覧を取得します
// @Tags Invitations
// @Produce json
// @Security SessionCookie
// @Param id path string true "グループID" format(uuid)
// @Success 200 {object} handler.SwaggerInvitationListResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Router /groups/{id}/invitations [get]
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
// @Summary 保留中招待一覧取得
// @Description 認証ユーザー宛の保留中招待一覧を取得します
// @Tags Invitations
// @Produce json
// @Security SessionCookie
// @Success 200 {object} handler.SwaggerPendingInvitationListResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /invitations/pending [get]
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
