package di

import (
	profilecmd "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/profile/command"
	profileqry "github.com/Hiro-mackay/gc-storage/backend/internal/usecase/profile/query"
)

// ProfileUseCases はProfile関連のUseCaseを保持します
type ProfileUseCases struct {
	// Queries
	GetProfile *profileqry.GetProfileQuery

	// Commands
	UpdateProfile *profilecmd.UpdateProfileCommand
	UpdateUser    *profilecmd.UpdateUserCommand
}

// NewProfileUseCases は新しいProfileUseCasesを作成します
func NewProfileUseCases(c *Container) *ProfileUseCases {
	return &ProfileUseCases{
		// Queries
		GetProfile: profileqry.NewGetProfileQuery(
			c.UserProfileRepo,
			c.UserRepo,
		),

		// Commands
		UpdateProfile: profilecmd.NewUpdateProfileCommand(
			c.UserProfileRepo,
			c.UserRepo,
		),
		UpdateUser: profilecmd.NewUpdateUserCommand(
			c.UserRepo,
		),
	}
}
