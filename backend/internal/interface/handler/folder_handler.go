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

// FolderHandler はフォルダ関連のHTTPハンドラーです
type FolderHandler struct {
	// Commands
	createFolderCommand *storagecmd.CreateFolderCommand
	renameFolderCommand *storagecmd.RenameFolderCommand
	moveFolderCommand   *storagecmd.MoveFolderCommand
	deleteFolderCommand *storagecmd.DeleteFolderCommand

	// Queries
	getFolderQuery          *storageqry.GetFolderQuery
	listFolderContentsQuery *storageqry.ListFolderContentsQuery
	getAncestorsQuery       *storageqry.GetAncestorsQuery
}

// NewFolderHandler は新しいFolderHandlerを作成します
func NewFolderHandler(
	createFolderCommand *storagecmd.CreateFolderCommand,
	renameFolderCommand *storagecmd.RenameFolderCommand,
	moveFolderCommand *storagecmd.MoveFolderCommand,
	deleteFolderCommand *storagecmd.DeleteFolderCommand,
	getFolderQuery *storageqry.GetFolderQuery,
	listFolderContentsQuery *storageqry.ListFolderContentsQuery,
	getAncestorsQuery *storageqry.GetAncestorsQuery,
) *FolderHandler {
	return &FolderHandler{
		createFolderCommand:     createFolderCommand,
		renameFolderCommand:     renameFolderCommand,
		moveFolderCommand:       moveFolderCommand,
		deleteFolderCommand:     deleteFolderCommand,
		getFolderQuery:          getFolderQuery,
		listFolderContentsQuery: listFolderContentsQuery,
		getAncestorsQuery:       getAncestorsQuery,
	}
}

// CreateFolder はフォルダを作成します
// @Summary フォルダ作成
// @Description 新しいフォルダを作成します
// @Tags Folders
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param body body request.CreateFolderRequest true "フォルダ情報"
// @Success 201 {object} handler.SwaggerFolderResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /folders [post]
func (h *FolderHandler) CreateFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	var req request.CreateFolderRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return apperror.NewValidationError("invalid parent ID", nil)
		}
		parentID = &id
	}

	output, err := h.createFolderCommand.Execute(c.Request().Context(), storagecmd.CreateFolderInput{
		Name:     req.Name,
		ParentID: parentID,
		OwnerID:  claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.Created(c, response.ToFolderResponse(output.Folder))
}

// GetFolder はフォルダを取得します
// @Summary フォルダ取得
// @Description 指定したフォルダの情報を取得します
// @Tags Folders
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Success 200 {object} handler.SwaggerFolderResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /folders/{id} [get]
func (h *FolderHandler) GetFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	output, err := h.getFolderQuery.Execute(c.Request().Context(), storageqry.GetFolderInput{
		FolderID: folderID,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToFolderResponse(output.Folder))
}

// ListFolderContents はフォルダ内容一覧を取得します
// @Summary フォルダ内容一覧取得
// @Description 指定したフォルダ内のフォルダとファイルの一覧を取得します
// @Tags Folders
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Success 200 {object} handler.SwaggerFolderContentsResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /folders/{id}/contents [get]
func (h *FolderHandler) ListFolderContents(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	var folderID *uuid.UUID
	idParam := c.Param("id")
	if idParam != "" && idParam != "root" {
		id, err := uuid.Parse(idParam)
		if err != nil {
			return apperror.NewValidationError("invalid folder ID", nil)
		}
		folderID = &id
	}

	output, err := h.listFolderContentsQuery.Execute(c.Request().Context(), storageqry.ListFolderContentsInput{
		FolderID: folderID,
		OwnerID:  claims.UserID,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	resp := response.FolderContentsResponse{
		Folders: response.ToFolderListResponse(output.Folders),
		Files:   response.ToFileListResponse(output.Files),
	}
	if output.Folder != nil {
		folder := response.ToFolderResponse(output.Folder)
		resp.Folder = &folder
	}

	return presenter.OK(c, resp)
}

// GetAncestors はフォルダの祖先一覧を取得します（パンくずリスト用）
// @Summary フォルダ祖先一覧取得
// @Description 指定したフォルダの祖先一覧を取得します（パンくずリスト用）
// @Tags Folders
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Success 200 {object} handler.SwaggerBreadcrumbResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /folders/{id}/ancestors [get]
func (h *FolderHandler) GetAncestors(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	output, err := h.getAncestorsQuery.Execute(c.Request().Context(), storageqry.GetAncestorsInput{
		FolderID: folderID,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToBreadcrumbResponse(output.Ancestors))
}

// RenameFolder はフォルダ名を変更します
// @Summary フォルダ名変更
// @Description 指定したフォルダの名前を変更します
// @Tags Folders
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Param body body request.RenameFolderRequest true "新しいフォルダ名"
// @Success 200 {object} handler.SwaggerFolderResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /folders/{id}/rename [patch]
func (h *FolderHandler) RenameFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	var req request.RenameFolderRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.renameFolderCommand.Execute(c.Request().Context(), storagecmd.RenameFolderInput{
		FolderID: folderID,
		NewName:  req.Name,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToFolderResponse(output.Folder))
}

// MoveFolder はフォルダを移動します
// @Summary フォルダ移動
// @Description 指定したフォルダを別のフォルダに移動します
// @Tags Folders
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Param body body request.MoveFolderRequest true "移動先フォルダ情報"
// @Success 200 {object} handler.SwaggerFolderResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /folders/{id}/move [patch]
func (h *FolderHandler) MoveFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	var req request.MoveFolderRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}

	var newParentID *uuid.UUID
	if req.NewParentID != nil {
		id, err := uuid.Parse(*req.NewParentID)
		if err != nil {
			return apperror.NewValidationError("invalid new parent ID", nil)
		}
		newParentID = &id
	}

	output, err := h.moveFolderCommand.Execute(c.Request().Context(), storagecmd.MoveFolderInput{
		FolderID:    folderID,
		NewParentID: newParentID,
		UserID:      claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToFolderResponse(output.Folder))
}

// DeleteFolder はフォルダを削除します
// @Summary フォルダ削除
// @Description 指定したフォルダを削除します
// @Tags Folders
// @Produce json
// @Security SessionCookie
// @Param id path string true "フォルダID"
// @Success 204 "No Content"
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Failure 404 {object} handler.SwaggerErrorResponse
// @Router /folders/{id} [delete]
func (h *FolderHandler) DeleteFolder(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apperror.NewValidationError("invalid folder ID", nil)
	}

	_, err = h.deleteFolderCommand.Execute(c.Request().Context(), storagecmd.DeleteFolderInput{
		FolderID: folderID,
		UserID:   claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.NoContent(c)
}
