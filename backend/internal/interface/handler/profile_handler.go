package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/request"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/dto/response"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/middleware"
	"github.com/Hiro-mackay/gc-storage/backend/internal/interface/presenter"
	profilecmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/profile/command"
	profileqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/profile/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
)

// ProfileHandler はプロファイル関連のHTTPハンドラーです
type ProfileHandler struct {
	// Queries
	getProfileQuery *profileqry.GetProfileQuery

	// Commands
	updateProfileCommand *profilecmd.UpdateProfileCommand
	updateUserCommand    *profilecmd.UpdateUserCommand
}

// NewProfileHandler は新しいProfileHandlerを作成します
func NewProfileHandler(
	getProfileQuery *profileqry.GetProfileQuery,
	updateProfileCommand *profilecmd.UpdateProfileCommand,
	updateUserCommand *profilecmd.UpdateUserCommand,
) *ProfileHandler {
	return &ProfileHandler{
		getProfileQuery:      getProfileQuery,
		updateProfileCommand: updateProfileCommand,
		updateUserCommand:    updateUserCommand,
	}
}

// GetProfile は現在のユーザーのプロファイルを取得します
// @Summary プロファイル取得
// @Description 現在のユーザーのプロファイル情報を取得します
// @Tags Profile
// @Produce json
// @Security SessionCookie
// @Success 200 {object} handler.SwaggerProfileResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /me/profile [get]
func (h *ProfileHandler) GetProfile(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	output, err := h.getProfileQuery.Execute(c.Request().Context(), profileqry.GetProfileInput{
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToGetProfileResponse(output.User, output.Profile))
}

// UpdateProfile は現在のユーザーのプロファイルを更新します
// @Summary プロファイル更新
// @Description 現在のユーザーのプロファイル情報を更新します
// @Tags Profile
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param body body request.UpdateProfileRequest true "プロファイル更新情報"
// @Success 200 {object} handler.SwaggerUpdateProfileResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /me/profile [put]
func (h *ProfileHandler) UpdateProfile(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	var req request.UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	input := profilecmd.UpdateProfileInput{
		UserID:    claims.UserID,
		AvatarURL: req.AvatarURL,
		Bio:       req.Bio,
		Locale:    req.Locale,
		Timezone:  req.Timezone,
		Theme:     req.Theme,
	}

	if req.NotificationPreferences != nil {
		notifPrefs := &entity.NotificationPreferences{}
		if req.NotificationPreferences.EmailEnabled != nil {
			notifPrefs.EmailEnabled = *req.NotificationPreferences.EmailEnabled
		}
		if req.NotificationPreferences.PushEnabled != nil {
			notifPrefs.PushEnabled = *req.NotificationPreferences.PushEnabled
		}
		input.NotificationPreferences = notifPrefs
	}

	output, err := h.updateProfileCommand.Execute(c.Request().Context(), input)
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToUpdateProfileResponse(output.Profile))
}

// UpdateMe は現在のユーザーの基本情報を更新します
// @Summary ユーザー情報更新
// @Description 現在のユーザーの名前などの基本情報を更新します
// @Tags Profile
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param body body request.UpdateUserRequest true "ユーザー情報更新"
// @Success 200 {object} handler.SwaggerUserResponse
// @Failure 400 {object} handler.SwaggerErrorResponse
// @Failure 401 {object} handler.SwaggerErrorResponse
// @Router /me [put]
func (h *ProfileHandler) UpdateMe(c echo.Context) error {
	claims := middleware.GetAccessClaims(c)
	if claims == nil {
		return apperror.NewUnauthorizedError("invalid token")
	}

	var req request.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return apperror.NewValidationError("invalid request body", nil)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	output, err := h.updateUserCommand.Execute(c.Request().Context(), profilecmd.UpdateUserInput{
		UserID: claims.UserID,
		Name:   req.Name,
	})
	if err != nil {
		return err
	}

	return presenter.OK(c, response.ToUserResponse(output.User))
}
