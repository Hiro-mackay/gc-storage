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
// GET /api/v1/files/:id/permissions
func (h *PermissionHandler) ListFileGrants(c echo.Context) error {
	return h.listGrants(c, "file")
}

// ListFolderGrants はフォルダの権限一覧を取得します
// GET /api/v1/folders/:id/permissions
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
// POST /api/v1/files/:id/permissions
func (h *PermissionHandler) GrantFileRole(c echo.Context) error {
	return h.grantRole(c, "file")
}

// GrantFolderRole はフォルダに権限を付与します
// POST /api/v1/folders/:id/permissions
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
// DELETE /api/v1/permissions/:id
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
// GET /api/v1/:resourceType/:id/permissions/check
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
