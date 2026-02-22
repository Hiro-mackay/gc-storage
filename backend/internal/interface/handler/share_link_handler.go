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
	updateShareLinkCmd *sharingcmd.UpdateShareLinkCommand

	// Queries
	accessShareLinkQuery     *sharingqry.AccessShareLinkQuery
	listShareLinksQuery      *sharingqry.ListShareLinksQuery
	getShareLinkHistoryQuery *sharingqry.GetShareLinkHistoryQuery
	getDownloadViaShareQuery *sharingqry.GetDownloadViaShareQuery

	// Config
	baseURL string
}

// NewShareLinkHandler は新しいShareLinkHandlerを作成します
func NewShareLinkHandler(
	createShareLinkCmd *sharingcmd.CreateShareLinkCommand,
	revokeShareLinkCmd *sharingcmd.RevokeShareLinkCommand,
	updateShareLinkCmd *sharingcmd.UpdateShareLinkCommand,
	accessShareLinkQuery *sharingqry.AccessShareLinkQuery,
	listShareLinksQuery *sharingqry.ListShareLinksQuery,
	getShareLinkHistoryQuery *sharingqry.GetShareLinkHistoryQuery,
	getDownloadViaShareQuery *sharingqry.GetDownloadViaShareQuery,
	baseURL string,
) *ShareLinkHandler {
	return &ShareLinkHandler{
		createShareLinkCmd:       createShareLinkCmd,
		revokeShareLinkCmd:       revokeShareLinkCmd,
		updateShareLinkCmd:       updateShareLinkCmd,
		accessShareLinkQuery:     accessShareLinkQuery,
		listShareLinksQuery:      listShareLinksQuery,
		getShareLinkHistoryQuery: getShareLinkHistoryQuery,
		getDownloadViaShareQuery: getDownloadViaShareQuery,
		baseURL:                  baseURL,
	}
}

// CreateFileShareLink はファイルの共有リンクを作成します
// @Summary ファイル共有リンク作成
// @Description 指定したファイルの共有リンクを作成します
// @Tags ShareLinks
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Param body body request.CreateShareLinkRequest true "共有リンク作成情報"
// @Success 201 {object} handler.SwaggerShareLinkResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/share [post]
func (h *ShareLinkHandler) CreateFileShareLink(c echo.Context) error {
	return h.createShareLink(c, "file")
}

// CreateFolderShareLink はフォルダの共有リンクを作成します
// @Summary フォルダ共有リンク作成
// @Description 指定したフォルダの共有リンクを作成します
// @Tags ShareLinks
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Param body body request.CreateShareLinkRequest true "共有リンク作成情報"
// @Success 201 {object} handler.SwaggerShareLinkResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /folders/{id}/share [post]
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
// @Summary ファイル共有リンク一覧取得
// @Description 指定したファイルの共有リンク一覧を取得します
// @Tags ShareLinks
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Success 200 {object} handler.SwaggerShareLinkListResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/share [get]
func (h *ShareLinkHandler) ListFileShareLinks(c echo.Context) error {
	return h.listShareLinks(c, "file")
}

// ListFolderShareLinks はフォルダの共有リンク一覧を取得します
// @Summary フォルダ共有リンク一覧取得
// @Description 指定したフォルダの共有リンク一覧を取得します
// @Tags ShareLinks
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Success 200 {object} handler.SwaggerShareLinkListResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /folders/{id}/share [get]
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
// @Summary 共有リンク無効化
// @Description 指定した共有リンクを無効化します
// @Tags ShareLinks
// @Security SessionCookie
// @Param id path string true "共有リンクID"
// @Success 204 "No Content"
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /share-links/{id} [delete]
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

// UpdateShareLink は共有リンクを更新します
// @Summary 共有リンク更新
// @Description 指定した共有リンクを更新します
// @Tags ShareLinks
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "共有リンクID"
// @Param body body request.UpdateShareLinkRequest true "共有リンク更新情報"
// @Success 200 {object} handler.SwaggerShareLinkResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /share-links/{id} [patch]
func (h *ShareLinkHandler) UpdateShareLink(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	shareLinkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid share link ID", nil)
	}

	var req request.UpdateShareLinkRequest
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

	output, err := h.updateShareLinkCmd.Execute(c.Request().Context(), sharingcmd.UpdateShareLinkInput{
		ShareLinkID:    shareLinkID,
		UpdatedBy:      claims.UserID,
		Password:       req.Password,
		ExpiresAt:      expiresAt,
		MaxAccessCount: req.MaxAccessCount,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToShareLinkResponse(output.ShareLink, h.baseURL))
}

// GetShareLinkHistory は共有リンクのアクセス履歴を取得します
// @Summary 共有リンクアクセス履歴取得
// @Description 指定した共有リンクのアクセス履歴を取得します
// @Tags ShareLinks
// @Produce json
// @Security SessionCookie
// @Param id path string true "共有リンクID"
// @Param limit query int false "取得件数" default(20)
// @Param offset query int false "オフセット" default(0)
// @Success 200 {object} handler.SwaggerShareLinkAccessListResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /share-links/{id}/history [get]
func (h *ShareLinkHandler) GetShareLinkHistory(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	shareLinkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid share link ID", nil)
	}

	var limit, offset int
	if err := echo.QueryParamsBinder(c).Int("limit", &limit).Int("offset", &offset).BindError(); err != nil {
		return apperror.NewValidationError("invalid query parameters", nil)
	}

	output, err := h.getShareLinkHistoryQuery.Execute(c.Request().Context(), sharingqry.GetShareLinkHistoryInput{
		ShareLinkID: shareLinkID,
		UserID:      claims.UserID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToShareLinkAccessListResponse(output.Accesses, output.Total))
}
