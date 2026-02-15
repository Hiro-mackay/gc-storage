package handler

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	authzcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/authz/command"
	authzqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/authz/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// PermissionHandler は権限関連のHTTPハンドラーです
type PermissionHandler struct {
	// Commands
	grantRoleCmd   *authzcmd.GrantRoleCommand
	revokeGrantCmd *authzcmd.RevokeGrantCommand

	// Queries
	listGrantsQuery      *authzqry.ListGrantsQuery
	checkPermissionQuery *authzqry.CheckPermissionQuery
}

// NewPermissionHandler は新しいPermissionHandlerを作成します
func NewPermissionHandler(
	grantRoleCmd *authzcmd.GrantRoleCommand,
	revokeGrantCmd *authzcmd.RevokeGrantCommand,
	listGrantsQuery *authzqry.ListGrantsQuery,
	checkPermissionQuery *authzqry.CheckPermissionQuery,
) *PermissionHandler {
	return &PermissionHandler{
		grantRoleCmd:         grantRoleCmd,
		revokeGrantCmd:       revokeGrantCmd,
		listGrantsQuery:      listGrantsQuery,
		checkPermissionQuery: checkPermissionQuery,
	}
}

// ListFileGrants はファイルの権限一覧を取得します
// @Summary ファイル権限一覧取得
// @Description 指定したファイルに付与されている権限の一覧を取得します
// @Tags Permissions
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Success 200 {object} handler.SwaggerPermissionGrantListResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/permissions [get]
func (h *PermissionHandler) ListFileGrants(c echo.Context) error {
	return h.listGrants(c, "file")
}

// ListFolderGrants はフォルダの権限一覧を取得します
// @Summary フォルダ権限一覧取得
// @Description 指定したフォルダに付与されている権限の一覧を取得します
// @Tags Permissions
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Success 200 {object} handler.SwaggerPermissionGrantListResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /folders/{id}/permissions [get]
func (h *PermissionHandler) ListFolderGrants(c echo.Context) error {
	return h.listGrants(c, "folder")
}

func (h *PermissionHandler) listGrants(c echo.Context, resourceType string) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	resourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid resource ID", nil)
	}

	output, err := h.listGrantsQuery.Execute(c.Request().Context(), authzqry.ListGrantsInput{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		UserID:       claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToPermissionGrantListResponse(output.Grants))
}

// GrantFileRole はファイルに権限を付与します
// @Summary ファイル権限付与
// @Description 指定したファイルにユーザーまたはグループの権限を付与します
// @Tags Permissions
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Param body body request.GrantRoleRequest true "権限付与情報"
// @Success 201 {object} handler.SwaggerPermissionGrantResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/permissions [post]
func (h *PermissionHandler) GrantFileRole(c echo.Context) error {
	return h.grantRole(c, "file")
}

// GrantFolderRole はフォルダに権限を付与します
// @Summary フォルダ権限付与
// @Description 指定したフォルダにユーザーまたはグループの権限を付与します
// @Tags Permissions
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Param body body request.GrantRoleRequest true "権限付与情報"
// @Success 201 {object} handler.SwaggerPermissionGrantResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /folders/{id}/permissions [post]
func (h *PermissionHandler) GrantFolderRole(c echo.Context) error {
	return h.grantRole(c, "folder")
}

func (h *PermissionHandler) grantRole(c echo.Context, resourceType string) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	resourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid resource ID", nil)
	}

	var req request.GrantRoleRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	granteeID, err := uuid.Parse(req.GranteeID)
	if err != nil {
		return apperror.NewValidationError("invalid grantee ID", nil)
	}

	output, err := h.grantRoleCmd.Execute(c.Request().Context(), authzcmd.GrantRoleInput{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		GranteeType:  req.GranteeType,
		GranteeID:    granteeID,
		Role:         req.Role,
		GrantedBy:    claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.Created(c, response.ToPermissionGrantResponse(output.Grant))
}

// RevokeGrant は権限を取り消します
// @Summary 権限取り消し
// @Description 指定した権限付与を取り消します
// @Tags Permissions
// @Security SessionCookie
// @Param id path string true "権限付与ID"
// @Success 204 "No Content"
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /permissions/{id} [delete]
func (h *PermissionHandler) RevokeGrant(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	grantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid grant ID", nil)
	}

	_, err = h.revokeGrantCmd.Execute(c.Request().Context(), authzcmd.RevokeGrantInput{
		GrantID:   grantID,
		RevokedBy: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}

// CheckPermission は権限を確認します
// @Summary 権限チェック
// @Description 指定したリソースに対する権限を確認します
// @Tags Permissions
// @Produce json
// @Security SessionCookie
// @Param resourceType path string true "リソースタイプ (files, folders)"
// @Param id path string true "リソースID"
// @Param permission query string true "確認する権限"
// @Success 200 {object} handler.SwaggerCheckPermissionResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /{resourceType}/{id}/permissions/check [get]
func (h *PermissionHandler) CheckPermission(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	resourceType := c.Param("resourceType")
	if resourceType != "files" && resourceType != "folders" {
		return apperror.NewValidationError("invalid resource type", nil)
	}
	// Convert plural to singular
	if resourceType == "files" {
		resourceType = "file"
	} else {
		resourceType = "folder"
	}

	resourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid resource ID", nil)
	}

	permission := c.QueryParam("permission")
	if permission == "" {
		return apperror.NewValidationError("permission is required", nil)
	}

	output, err := h.checkPermissionQuery.Execute(c.Request().Context(), authzqry.CheckPermissionInput{
		UserID:       claims.UserID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Permission:   permission,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.CheckPermissionResponse{
		HasPermission: output.HasPermission,
		EffectiveRole: output.EffectiveRole,
	})
}
