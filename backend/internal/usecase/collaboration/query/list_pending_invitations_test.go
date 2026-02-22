package query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type listPendingInvitationsTestDeps struct {
	invitationRepo *mocks.MockInvitationRepository
	userRepo       *mocks.MockUserRepository
	groupRepo      *mocks.MockGroupRepository
}

func newListPendingInvitationsTestDeps(t *testing.T) *listPendingInvitationsTestDeps {
	t.Helper()
	return &listPendingInvitationsTestDeps{
		invitationRepo: mocks.NewMockInvitationRepository(t),
		userRepo:       mocks.NewMockUserRepository(t),
		groupRepo:      mocks.NewMockGroupRepository(t),
	}
}

func (d *listPendingInvitationsTestDeps) newQuery() *query.ListPendingInvitationsQuery {
	return query.NewListPendingInvitationsQuery(
		d.invitationRepo,
		d.userRepo,
		d.groupRepo,
	)
}

func newTestUser(email string) *entity.User {
	vo, _ := valueobject.NewEmail(email)
	return &entity.User{
		ID:    uuid.New(),
		Email: vo,
		Name:  "Test User",
	}
}

func TestListPendingInvitationsQuery_Execute_UserHasPendingInvitations_ReturnsWithGroup(t *testing.T) {
	ctx := context.Background()
	deps := newListPendingInvitationsTestDeps(t)

	userID := uuid.New()
	user := newTestUser("user@example.com")
	user.ID = userID

	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	emailVO, _ := valueobject.NewEmail("user@example.com")
	invitations := []*entity.Invitation{
		entity.ReconstructInvitation(
			uuid.New(), groupID, emailVO, "token1", valueobject.GroupRoleViewer,
			uuid.New(), time.Now().Add(24*time.Hour), valueobject.InvitationStatusPending, time.Now(),
		),
	}

	deps.userRepo.On("FindByID", ctx, userID).Return(user, nil)
	deps.invitationRepo.On("FindPendingByEmail", ctx, user.Email).Return(invitations, nil)
	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListPendingInvitationsInput{
		UserID: userID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Invitations, 1)
	assert.Equal(t, groupID, output.Invitations[0].Group.ID)
}

func TestListPendingInvitationsQuery_Execute_UserNotFound_NotFoundError(t *testing.T) {
	ctx := context.Background()
	deps := newListPendingInvitationsTestDeps(t)

	userID := uuid.New()

	deps.userRepo.On("FindByID", ctx, userID).Return(nil, errors.New("not found"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListPendingInvitationsInput{
		UserID: userID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestListPendingInvitationsQuery_Execute_NoPendingInvitations_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	deps := newListPendingInvitationsTestDeps(t)

	userID := uuid.New()
	user := newTestUser("nomail@example.com")
	user.ID = userID

	deps.userRepo.On("FindByID", ctx, userID).Return(user, nil)
	deps.invitationRepo.On("FindPendingByEmail", ctx, user.Email).Return([]*entity.Invitation{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListPendingInvitationsInput{
		UserID: userID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output.Invitations)
}

func TestListPendingInvitationsQuery_Execute_GroupNotFound_SkipsInvitation(t *testing.T) {
	ctx := context.Background()
	deps := newListPendingInvitationsTestDeps(t)

	userID := uuid.New()
	user := newTestUser("user@example.com")
	user.ID = userID

	groupID := uuid.New()
	emailVO, _ := valueobject.NewEmail("user@example.com")
	invitations := []*entity.Invitation{
		entity.ReconstructInvitation(
			uuid.New(), groupID, emailVO, "token1", valueobject.GroupRoleViewer,
			uuid.New(), time.Now().Add(24*time.Hour), valueobject.InvitationStatusPending, time.Now(),
		),
	}

	deps.userRepo.On("FindByID", ctx, userID).Return(user, nil)
	deps.invitationRepo.On("FindPendingByEmail", ctx, user.Email).Return(invitations, nil)
	deps.groupRepo.On("FindByID", ctx, groupID).Return(nil, errors.New("group not found"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListPendingInvitationsInput{
		UserID: userID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	// Invitation is skipped because group is not found
	assert.Empty(t, output.Invitations)
}
