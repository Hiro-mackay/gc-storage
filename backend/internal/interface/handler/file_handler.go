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

// FileHandler はファイル操作関連のHTTPハンドラーです
type FileHandler struct {
	renameFileCommand     *storagecmd.RenameFileCommand
	moveFileCommand       *storagecmd.MoveFileCommand
	getDownloadURLQuery   *storageqry.GetDownloadURLQuery
	listFileVersionsQuery *storageqry.ListFileVersionsQuery
}

// NewFileHandler は新しいFileHandlerを作成します
func NewFileHandler(
	renameFileCommand *storagecmd.RenameFileCommand,
	moveFileCommand *storagecmd.MoveFileCommand,
	getDownloadURLQuery *storageqry.GetDownloadURLQuery,
	listFileVersionsQuery *storageqry.ListFileVersionsQuery,
) *FileHandler {
	return &FileHandler{
		renameFileCommand:     renameFileCommand,
		moveFileCommand:       moveFileCommand,
		getDownloadURLQuery:   getDownloadURLQuery,
		listFileVersionsQuery: listFileVersionsQuery,
	}
}

// GetDownloadURL はダウンロードURLを取得します
// @Summary ダウンロードURL取得
// @Description ファイルのダウンロード用署名付きURLを取得します
// @Tags Files
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Param version query int false "バージョン番号"
// @Success 200 {object} handler.SwaggerDownloadURLResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/download [get]
func (h *FileHandler) GetDownloadURL(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid file ID", nil)
	}

	var versionNumber *int
	if versionParam := c.QueryParam("version"); versionParam != "" {
		v, err := strconv.Atoi(versionParam)
		if err != nil {
			return apperror.NewValidationError("version must be a positive integer", nil)
		}
		if v < 1 {
			return apperror.NewValidationError("version number must be positive", nil)
		}
		versionNumber = &v
	}

	output, err := h.getDownloadURLQuery.Execute(c.Request().Context(), storageqry.GetDownloadURLInput{
		FileID:        fileID,
		VersionNumber: versionNumber,
		UserID:        claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToDownloadURLResponse(output))
}

// ListFileVersions はファイルバージョン一覧を取得します
// @Summary ファイルバージョン一覧取得
// @Description 指定されたファイルのバージョン一覧を取得します
// @Tags Files
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Success 200 {object} handler.SwaggerFileVersionsResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/versions [get]
func (h *FileHandler) ListFileVersions(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid file ID", nil)
	}

	output, err := h.listFileVersionsQuery.Execute(c.Request().Context(), storageqry.ListFileVersionsInput{
		FileID: fileID,
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToFileVersionsResponse(output))
}

// RenameFile はファイル名を変更します
// @Summary ファイル名変更
// @Description 指定されたファイルの名前を変更します
// @Tags Files
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Param body body request.RenameFileRequest true "新しいファイル名"
// @Success 200 {object} handler.SwaggerRenameFileResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/rename [patch]
func (h *FileHandler) RenameFile(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid file ID", nil)
	}

	var req request.RenameFileRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.renameFileCommand.Execute(c.Request().Context(), storagecmd.RenameFileInput{
		FileID:  fileID,
		NewName: req.Name,
		UserID:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.RenameFileResponse{
		FileID: output.FileID.String(),
		Name:   output.Name,
	})
}

// MoveFile はファイルを移動します
// @Summary ファイル移動
// @Description 指定されたファイルを別のフォルダに移動します
// @Tags Files
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "ファイルID"
// @Param body body request.MoveFileRequest true "移動先フォルダ情報"
// @Success 200 {object} handler.SwaggerMoveFileResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /files/{id}/move [patch]
func (h *FileHandler) MoveFile(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid file ID", nil)
	}

	var req request.MoveFileRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}

	if req.NewFolderID == nil {
		return apperror.NewValidationError("new folder ID is required", nil)
	}
	newFolderID, err := uuid.Parse(*req.NewFolderID)
	if err != nil {
		return apperror.NewValidationError("invalid new folder ID", nil)
	}

	output, err := h.moveFileCommand.Execute(c.Request().Context(), storagecmd.MoveFileInput{
		FileID:      fileID,
		NewFolderID: newFolderID,
		UserID:      claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.MoveFileResponse{
		FileID:   output.FileID.String(),
		FolderID: output.FolderID.String(),
	})
}
