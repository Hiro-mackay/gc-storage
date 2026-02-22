package handler

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	storagecmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/command"
	storageqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/storage/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// UploadHandler はファイルアップロード関連のHTTPハンドラーです
type UploadHandler struct {
	initiateUploadCommand *storagecmd.InitiateUploadCommand
	completeUploadCommand *storagecmd.CompleteUploadCommand
	abortUploadCommand    *storagecmd.AbortUploadCommand
	getUploadStatusQuery  *storageqry.GetUploadStatusQuery
}

// NewUploadHandler は新しいUploadHandlerを作成します
func NewUploadHandler(
	initiateUploadCommand *storagecmd.InitiateUploadCommand,
	completeUploadCommand *storagecmd.CompleteUploadCommand,
	abortUploadCommand *storagecmd.AbortUploadCommand,
	getUploadStatusQuery *storageqry.GetUploadStatusQuery,
) *UploadHandler {
	return &UploadHandler{
		initiateUploadCommand: initiateUploadCommand,
		completeUploadCommand: completeUploadCommand,
		abortUploadCommand:    abortUploadCommand,
		getUploadStatusQuery:  getUploadStatusQuery,
	}
}

// InitiateUpload はアップロードを開始します
// @Summary アップロード開始
// @Description ファイルアップロードセッションを開始し、署名付きURLを返します
// @Tags Files
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param body body request.InitiateUploadRequest true "アップロード情報"
// @Success 201 {object} handler.SwaggerInitiateUploadResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /files/upload [post]
func (h *UploadHandler) InitiateUpload(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	var req request.InitiateUploadRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	if req.FolderID == nil {
		return apperror.NewValidationError("folder ID is required", nil)
	}
	folderID, err := uuid.Parse(*req.FolderID)
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	output, err := h.initiateUploadCommand.Execute(c.Request().Context(), storagecmd.InitiateUploadInput{
		FolderID: folderID,
		FileName: req.FileName,
		MimeType: req.MimeType,
		Size:     req.Size,
		OwnerID:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.Created(c, response.ToInitiateUploadResponse(output))
}

// CompleteUpload はアップロードを完了します（MinIO Webhook用）
// @Summary アップロード完了
// @Description MinIO Webhookからの通知を受けてアップロードを完了します
// @Tags Files
// @Accept json
// @Produce json
// @Param body body request.CompleteUploadRequest true "アップロード完了情報"
// @Success 200 {object} handler.SwaggerCompleteUploadResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Router /files/upload/complete [post]
func (h *UploadHandler) CompleteUpload(c echo.Context) error {
	var req request.CompleteUploadRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.completeUploadCommand.Execute(c.Request().Context(), storagecmd.CompleteUploadInput{
		StorageKey:     req.StorageKey,
		MinioVersionID: req.MinioVersionID,
		Size:           req.Size,
		ETag:           req.ETag,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.CompleteUploadResponse{
		FileID:    output.FileID.String(),
		SessionID: output.SessionID.String(),
		Completed: output.Completed,
	})
}

// GetUploadStatus はアップロード状態を取得します
// @Summary アップロード状態取得
// @Description 指定されたセッションIDのアップロード状態を取得します
// @Tags Files
// @Produce json
// @Security SessionCookie
// @Param sessionId path string true "セッションID"
// @Success 200 {object} handler.SwaggerUploadStatusResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /files/upload/{sessionId} [get]
func (h *UploadHandler) GetUploadStatus(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		return apperror.NewValidationError("invalid session ID", nil)
	}

	output, err := h.getUploadStatusQuery.Execute(c.Request().Context(), storageqry.GetUploadStatusInput{
		SessionID: sessionID,
		UserID:    claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToUploadStatusResponse(output))
}

// AbortUpload はアップロードを中断します
// @Summary アップロード中断
// @Description 進行中のアップロードセッションを中断します
// @Tags Files
// @Produce json
// @Security SessionCookie
// @Param sessionId path string true "セッションID"
// @Success 200 {object} handler.SwaggerAbortUploadResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /files/upload/{sessionId} [delete]
func (h *UploadHandler) AbortUpload(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		return apperror.NewValidationError("invalid session ID", nil)
	}

	output, err := h.abortUploadCommand.Execute(c.Request().Context(), storagecmd.AbortUploadInput{
		SessionID: sessionID,
		UserID:    claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.AbortUploadResponse{
		SessionID: output.SessionID.String(),
		Aborted:   output.Aborted,
	})
}
