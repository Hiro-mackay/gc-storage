package handler

import (
	"strconv"

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

// TrashHandler はゴミ箱関連のHTTPハンドラーです
type TrashHandler struct {
	trashFileCommand             *storagecmd.TrashFileCommand
	restoreFileCommand           *storagecmd.RestoreFileCommand
	permanentlyDeleteFileCommand *storagecmd.PermanentlyDeleteFileCommand
	emptyTrashCommand            *storagecmd.EmptyTrashCommand
	listTrashQuery               *storageqry.ListTrashQuery
}

// NewTrashHandler は新しいTrashHandlerを作成します
func NewTrashHandler(
	trashFileCommand *storagecmd.TrashFileCommand,
	restoreFileCommand *storagecmd.RestoreFileCommand,
	permanentlyDeleteFileCommand *storagecmd.PermanentlyDeleteFileCommand,
	emptyTrashCommand *storagecmd.EmptyTrashCommand,
	listTrashQuery *storageqry.ListTrashQuery,
) *TrashHandler {
	return &TrashHandler{
		trashFileCommand:             trashFileCommand,
		restoreFileCommand:           restoreFileCommand,
		permanentlyDeleteFileCommand: permanentlyDeleteFileCommand,
		emptyTrashCommand:            emptyTrashCommand,
		listTrashQuery:               listTrashQuery,
	}
}

// TrashFile はファイルをゴミ箱に移動します
// @Summary ファイルをゴミ箱に移動
// @Description 指定されたファイルをゴミ箱に移動します
// @Tags Files
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Success 200 {object} handler.SwaggerTrashFileResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/trash [post]
func (h *TrashHandler) TrashFile(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid file ID", nil)
	}

	output, err := h.trashFileCommand.Execute(c.Request().Context(), storagecmd.TrashFileInput{
		FileID: fileID,
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.TrashFileResponse{
		ArchivedFileID: output.ArchivedFileID.String(),
		ExpiresAt:      output.ExpiresAt,
	})
}

// ListTrash はゴミ箱一覧を取得します
// @Summary ゴミ箱一覧取得
// @Description ゴミ箱に入っているファイルの一覧を取得します
// @Tags Trash
// @Produce json
// @Security SessionCookie
// @Param limit query int false "取得件数"
// @Param cursor query string false "カーソル"
// @Success 200 {object} handler.SwaggerTrashListResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /trash [get]
func (h *TrashHandler) ListTrash(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	var cursor *uuid.UUID
	if cursorStr := c.QueryParam("cursor"); cursorStr != "" {
		id, err := uuid.Parse(cursorStr)
		if err != nil {
			return apperror.NewValidationError("invalid cursor", nil)
		}
		cursor = &id
	}

	output, err := h.listTrashQuery.Execute(c.Request().Context(), storageqry.ListTrashInput{
		OwnerID: claims.UserID,
		UserID:  claims.UserID,
		Limit:   limit,
		Cursor:  cursor,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToTrashListResponse(output))
}

// RestoreFile はファイルをゴミ箱から復元します
// @Summary ファイル復元
// @Description ゴミ箱からファイルを復元します
// @Tags Trash
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "アーカイブファイルID"
// @Param body body request.RestoreFileRequest false "復元先フォルダ情報"
// @Success 200 {object} handler.SwaggerRestoreFileResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /trash/files/{id}/restore [post]
func (h *TrashHandler) RestoreFile(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	archivedFileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid archived file ID", nil)
	}

	var req request.RestoreFileRequest
	if err := c.Bind(&req); err != nil {
		req = request.RestoreFileRequest{}
	}

	var restoreFolderID *uuid.UUID
	if req.RestoreFolderID != nil {
		id, err := uuid.Parse(*req.RestoreFolderID)
		if err != nil {
			return apperror.NewValidationError("invalid restore folder ID", nil)
		}
		restoreFolderID = &id
	}

	output, err := h.restoreFileCommand.Execute(c.Request().Context(), storagecmd.RestoreFileInput{
		ArchivedFileID:  archivedFileID,
		RestoreFolderID: restoreFolderID,
		UserID:          claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.RestoreFileResponse{
		FileID:   output.FileID.String(),
		FolderID: output.FolderID.String(),
		Name:     output.Name,
	})
}

// PermanentlyDeleteFile はファイルを完全に削除します
// @Summary ファイル完全削除
// @Description ゴミ箱のファイルを完全に削除します
// @Tags Trash
// @Produce json
// @Security SessionCookie
// @Param id path string true "アーカイブファイルID"
// @Success 204
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /trash/files/{id} [delete]
func (h *TrashHandler) PermanentlyDeleteFile(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	archivedFileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid archived file ID", nil)
	}

	err = h.permanentlyDeleteFileCommand.Execute(c.Request().Context(), storagecmd.PermanentlyDeleteFileInput{
		ArchivedFileID: archivedFileID,
		UserID:         claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}

// EmptyTrash はゴミ箱を空にします
// @Summary ゴミ箱を空にする
// @Description ゴミ箱の全ファイルを完全に削除します
// @Tags Trash
// @Produce json
// @Security SessionCookie
// @Success 202 {object} handler.SwaggerEmptyTrashResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 403 {object} handler.SwaggerErrorResponse
// @Router /trash [delete]
func (h *TrashHandler) EmptyTrash(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	output, err := h.emptyTrashCommand.Execute(c.Request().Context(), storagecmd.EmptyTrashInput{
		OwnerID: claims.UserID,
		UserID:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.Accepted(c, response.EmptyTrashResponse{
		Message:      "Trash emptying started",
		DeletedCount: output.DeletedCount,
	})
}
