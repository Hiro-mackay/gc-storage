package handler

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	sharingqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/sharing/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// GetShareLinkInfo は共有リンク情報を取得します（認証不要）
// @Summary 共有リンク情報取得
// @Description トークンを使用して共有リンクの情報を取得します（認証不要）
// @Tags ShareLinks
// @Produce json
// @Param token path string true "共有リンクトークン"
// @Success 200 {object} handler.SwaggerShareLinkInfoResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /share/{token} [get]
func (h *ShareLinkHandler) GetShareLinkInfo(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return apperror.NewValidationError("invalid share link token", nil)
	}

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
// @Summary 共有リンクアクセス
// @Description 共有リンクにアクセスしてリソース情報を取得します（認証不要）
// @Tags ShareLinks
// @Accept json
// @Produce json
// @Param token path string true "共有リンクトークン"
// @Param action query string false "アクション (view, download)" default(download)
// @Param body body request.AccessShareLinkRequest false "パスワード（パスワード保護されている場合）"
// @Success 200 {object} handler.SwaggerShareLinkAccessResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /share/{token}/access [post]
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

	return presenter.OK(c, response.ToShareLinkAccessResponse(output))
}

// GetDownloadViaShare は共有リンク経由でダウンロードURLを取得します
// @Summary 共有リンク経由ダウンロード
// @Description 共有リンクのトークンを使用してファイルのダウンロードURLを取得します（認証不要）
// @Tags ShareLinks
// @Produce json
// @Param token path string true "共有リンクトークン"
// @Param fileId query string false "ファイルID（フォルダ共有の場合必須）"
// @Param X-Share-Password header string false "パスワード（パスワード保護されている場合）"
// @Success 200 {object} handler.SwaggerShareDownloadResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Failure 410 {object} handler.SwaggerErrorResponse
// @Router /share/{token}/download [get]
func (h *ShareLinkHandler) GetDownloadViaShare(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return apperror.NewValidationError("invalid share link token", nil)
	}

	password := c.Request().Header.Get("X-Share-Password")

	fileIDStr := c.QueryParam("fileId")
	var fileID *uuid.UUID
	if fileIDStr != "" {
		parsed, err := uuid.Parse(fileIDStr)
		if err != nil {
			return apperror.NewValidationError("invalid file_id", nil)
		}
		fileID = &parsed
	}

	var userID *uuid.UUID
	claims := middleware.GetAccessClaims(c)
	if claims != nil {
		userID = &claims.UserID
	}

	output, err := h.getDownloadViaShareQuery.Execute(c.Request().Context(), sharingqry.GetDownloadViaShareInput{
		Token:     token,
		Password:  password,
		FileID:    fileID,
		UserID:    userID,
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToShareDownloadResponse(output))
}
