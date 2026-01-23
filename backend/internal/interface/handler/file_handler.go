package handler

import (
	"fmt"

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

// FileHandler はファイル関連のHTTPハンドラーです
type FileHandler struct {
	// Commands
	initiateUploadCommand *storagecmd.InitiateUploadCommand
	completeUploadCommand *storagecmd.CompleteUploadCommand
	renameFileCommand     *storagecmd.RenameFileCommand
	moveFileCommand       *storagecmd.MoveFileCommand
	trashFileCommand      *storagecmd.TrashFileCommand
	restoreFileCommand    *storagecmd.RestoreFileCommand

	// Queries
	getDownloadURLQuery   *storageqry.GetDownloadURLQuery
	getUploadStatusQuery  *storageqry.GetUploadStatusQuery
	listFileVersionsQuery *storageqry.ListFileVersionsQuery
	listTrashQuery        *storageqry.ListTrashQuery
}

// NewFileHandler は新しいFileHandlerを作成します
func NewFileHandler(
	initiateUploadCommand *storagecmd.InitiateUploadCommand,
	completeUploadCommand *storagecmd.CompleteUploadCommand,
	renameFileCommand *storagecmd.RenameFileCommand,
	moveFileCommand *storagecmd.MoveFileCommand,
	trashFileCommand *storagecmd.TrashFileCommand,
	restoreFileCommand *storagecmd.RestoreFileCommand,
	getDownloadURLQuery *storageqry.GetDownloadURLQuery,
	getUploadStatusQuery *storageqry.GetUploadStatusQuery,
	listFileVersionsQuery *storageqry.ListFileVersionsQuery,
	listTrashQuery *storageqry.ListTrashQuery,
) *FileHandler {
	return &FileHandler{
		initiateUploadCommand: initiateUploadCommand,
		completeUploadCommand: completeUploadCommand,
		renameFileCommand:     renameFileCommand,
		moveFileCommand:       moveFileCommand,
		trashFileCommand:      trashFileCommand,
		restoreFileCommand:    restoreFileCommand,
		getDownloadURLQuery:   getDownloadURLQuery,
		getUploadStatusQuery:  getUploadStatusQuery,
		listFileVersionsQuery: listFileVersionsQuery,
		listTrashQuery:        listTrashQuery,
	}
}

// InitiateUpload はアップロードを開始します
// POST /api/v1/files/upload
func (h *FileHandler) InitiateUpload(c echo.Context) error {
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

	// FolderID is required - files must belong to a folder
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
// POST /api/v1/files/upload/complete
func (h *FileHandler) CompleteUpload(c echo.Context) error {
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

	return presenter.OK(c, map[string]interface{}{
		"fileId":    output.FileID.String(),
		"sessionId": output.SessionID.String(),
		"completed": output.Completed,
	})
}

// GetUploadStatus はアップロード状態を取得します
// GET /api/v1/files/upload/:sessionId
func (h *FileHandler) GetUploadStatus(c echo.Context) error {
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

// GetDownloadURL はダウンロードURLを取得します
// GET /api/v1/files/:id/download
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
		var v int
		if _, err := fmt.Sscan(versionParam, &v); err == nil {
			versionNumber = &v
		}
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
// GET /api/v1/files/:id/versions
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
// PATCH /api/v1/files/:id/rename
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

	return presenter.OK(c, map[string]interface{}{
		"fileId": output.FileID.String(),
		"name":   output.Name,
	})
}

// MoveFile はファイルを移動します
// PATCH /api/v1/files/:id/move
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

	// NewFolderID is required - files must belong to a folder
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

	return presenter.OK(c, map[string]interface{}{
		"fileId":   output.FileID.String(),
		"folderId": output.FolderID.String(),
	})
}

// TrashFile はファイルをゴミ箱に移動します
// POST /api/v1/files/:id/trash
func (h *FileHandler) TrashFile(c echo.Context) error {
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

	return presenter.OK(c, map[string]interface{}{
		"archivedFileId": output.ArchivedFileID.String(),
	})
}

// ListTrash はゴミ箱一覧を取得します
// GET /api/v1/trash
func (h *FileHandler) ListTrash(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	output, err := h.listTrashQuery.Execute(c.Request().Context(), storageqry.ListTrashInput{
		OwnerID: claims.UserID,
		UserID:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToTrashListResponse(output))
}

// RestoreFile はファイルをゴミ箱から復元します
// POST /api/v1/trash/:id/restore
func (h *FileHandler) RestoreFile(c echo.Context) error {
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
		// Empty body is acceptable
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

	return presenter.OK(c, map[string]interface{}{
		"fileId":   output.FileID.String(),
		"folderId": output.FolderID.String(),
	})
}
