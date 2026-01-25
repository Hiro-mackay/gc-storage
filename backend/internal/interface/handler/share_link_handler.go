package handler

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	sharingcmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/command"
	sharingqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ShareLinkHandler は共有リンク関連のHTTPハンドラーです
type ShareLinkHandler struct {
	// Commands
	createShareLinkCmd *sharingcmd.CreateShareLinkCommand
	revokeShareLinkCmd *sharingcmd.RevokeShareLinkCommand

	// Queries
	accessShareLinkQuery *sharingqry.AccessShareLinkQuery
	listShareLinksQuery  *sharingqry.ListShareLinksQuery

	// Config
	baseURL string
}

// NewShareLinkHandler は新しいShareLinkHandlerを作成します
func NewShareLinkHandler(
	createShareLinkCmd *sharingcmd.CreateShareLinkCommand,
	revokeShareLinkCmd *sharingcmd.RevokeShareLinkCommand,
	accessShareLinkQuery *sharingqry.AccessShareLinkQuery,
	listShareLinksQuery *sharingqry.ListShareLinksQuery,
	baseURL string,
) *ShareLinkHandler {
	return &ShareLinkHandler{
		createShareLinkCmd:   createShareLinkCmd,
		revokeShareLinkCmd:   revokeShareLinkCmd,
		accessShareLinkQuery: accessShareLinkQuery,
		listShareLinksQuery:  listShareLinksQuery,
		baseURL:              baseURL,
	}
}

// CreateFileShareLink はファイルの共有リンクを作成します
// POST /api/v1/files/:id/share
func (h *ShareLinkHandler) CreateFileShareLink(c echo.Context) error {
	return h.createShareLink(c, "file")
}

// CreateFolderShareLink はフォルダの共有リンクを作成します
// POST /api/v1/folders/:id/share
func (h *ShareLinkHandler) CreateFolderShareLink(c echo.Context) error {
	return h.createShareLink(c, "folder")
}

func (h *ShareLinkHandler) createShareLink(c echo.Context, resourceType string) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	resourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid resource ID", nil)
	}

	var req request.CreateShareLinkRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return apperror.NewValidationError("invalid expiration date format", nil)
		}
		expiresAt = &t
	}

	var password string
	if req.Password != nil {
		password = *req.Password
	}

	output, err := h.createShareLinkCmd.Execute(c.Request().Context(), sharingcmd.CreateShareLinkInput{
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		CreatedBy:      claims.UserID,
		Permission:     req.Permission,
		Password:       password,
		ExpiresAt:      expiresAt,
		MaxAccessCount: req.MaxAccessCount,
	})
	if err != nil {
		return err
	}

	return presenter.Created(c, response.ToShareLinkResponse(output.ShareLink, h.baseURL))
}

// ListFileShareLinks はファイルの共有リンク一覧を取得します
// GET /api/v1/files/:id/share
func (h *ShareLinkHandler) ListFileShareLinks(c echo.Context) error {
	return h.listShareLinks(c, "file")
}

// ListFolderShareLinks はフォルダの共有リンク一覧を取得します
// GET /api/v1/folders/:id/share
func (h *ShareLinkHandler) ListFolderShareLinks(c echo.Context) error {
	return h.listShareLinks(c, "folder")
}

func (h *ShareLinkHandler) listShareLinks(c echo.Context, resourceType string) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	resourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid resource ID", nil)
	}

	output, err := h.listShareLinksQuery.Execute(c.Request().Context(), sharingqry.ListShareLinksInput{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		UserID:       claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToShareLinkListResponse(output.ShareLinks, h.baseURL))
}

// RevokeShareLink は共有リンクを無効化します
// DELETE /api/v1/share-links/:id
func (h *ShareLinkHandler) RevokeShareLink(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	shareLinkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid share link ID", nil)
	}

	_, err = h.revokeShareLinkCmd.Execute(c.Request().Context(), sharingcmd.RevokeShareLinkInput{
		ShareLinkID: shareLinkID,
		RevokedBy:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}

// GetShareLinkInfo は共有リンク情報を取得します（認証不要）
// GET /api/v1/share/:token
func (h *ShareLinkHandler) GetShareLinkInfo(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return apperror.NewValidationError("invalid share link token", nil)
	}

	// 情報取得のみなのでアクセスカウントは増やさない
	output, err := h.accessShareLinkQuery.Execute(c.Request().Context(), sharingqry.AccessShareLinkInput{
		Token:     token,
		Action:    "view",
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToShareLinkInfoResponse(output.ShareLink))
}

// AccessShareLink は共有リンクにアクセスします
// POST /api/v1/share/:token/access
func (h *ShareLinkHandler) AccessShareLink(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return apperror.NewValidationError("invalid share link token", nil)
	}

	var req request.AccessShareLinkRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}

	action := c.QueryParam("action")
	if action == "" {
		// デフォルトはdownload（view以外でパスワードチェックを行う）
		action = "download"
	}

	var userID *uuid.UUID
	claims := middleware.GetAccessClaims(c)
	if claims != nil {
		userID = &claims.UserID
	}

	output, err := h.accessShareLinkQuery.Execute(c.Request().Context(), sharingqry.AccessShareLinkInput{
		Token:     token,
		Password:  req.Password,
		UserID:    userID,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		Action:    action,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToShareLinkAccessResponse(output.ShareLink))
}
