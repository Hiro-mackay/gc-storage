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

type listInvitationsTestDeps struct {
	invitationRepo *mocks.MockInvitationRepository
	membershipRepo *mocks.MockMembershipRepository
	groupRepo      *mocks.MockGroupRepository
}

func newListInvitationsTestDeps(t *testing.T) *listInvitationsTestDeps {
	t.Helper()
	return &listInvitationsTestDeps{
		invitationRepo: mocks.NewMockInvitationRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
		groupRepo:      mocks.NewMockGroupRepository(t),
	}
}

func (d *listInvitationsTestDeps) newQuery() *query.ListInvitationsQuery {
	return query.NewListInvitationsQuery(
		d.invitationRepo,
		d.membershipRepo,
		d.groupRepo,
	)
}

func newTestGroup(ownerID uuid.UUID) *entity.Group {
	name, _ := valueobject.NewGroupName("Test Group")
	return entity.ReconstructGroup(uuid.New(), name, "", ownerID, time.Now(), time.Now())
}

func newTestMembership(groupID, userID uuid.UUID, role valueobject.GroupRole) *entity.Membership {
	return entity.ReconstructMembership(uuid.New(), groupID, userID, role, time.Now())
}

func TestListInvitationsQuery_Execute_OwnerRequestsInvitations_Success(t *testing.T) {
	ctx := context.Background()
	deps := newListInvitationsTestDeps(t)

	ownerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(ownerID)
	group.ID = groupID

	ownerMembership := newTestMembership(groupID, ownerID, valueobject.GroupRoleOwner)

	emailVO, _ := valueobject.NewEmail("invited@example.com")
	invitations := []*entity.Invitation{
		entity.ReconstructInvitation(
			uuid.New(), groupID, emailVO, "token1", valueobject.GroupRoleViewer,
			ownerID, time.Now().Add(24*time.Hour), valueobject.InvitationStatusPending, time.Now(),
		),
	}

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, ownerID).Return(ownerMembership, nil)
	deps.invitationRepo.On("FindPendingByGroupID", ctx, groupID).Return(invitations, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListInvitationsInput{
		GroupID:   groupID,
		RequestBy: ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.Invitations, 1)
}

func TestListInvitationsQuery_Execute_ContributorRequestsInvitations_Success(t *testing.T) {
	ctx := context.Background()
	deps := newListInvitationsTestDeps(t)

	contributorID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	contributorMembership := newTestMembership(groupID, contributorID, valueobject.GroupRoleContributor)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, contributorID).Return(contributorMembership, nil)
	deps.invitationRepo.On("FindPendingByGroupID", ctx, groupID).Return([]*entity.Invitation{}, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListInvitationsInput{
		GroupID:   groupID,
		RequestBy: contributorID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output.Invitations)
}

func TestListInvitationsQuery_Execute_ViewerRequests_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newListInvitationsTestDeps(t)

	viewerID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	viewerMembership := newTestMembership(groupID, viewerID, valueobject.GroupRoleViewer)

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, viewerID).Return(viewerMembership, nil)

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListInvitationsInput{
		GroupID:   groupID,
		RequestBy: viewerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}

func TestListInvitationsQuery_Execute_NonMemberRequests_ForbiddenError(t *testing.T) {
	ctx := context.Background()
	deps := newListInvitationsTestDeps(t)

	nonMemberID := uuid.New()
	groupID := uuid.New()
	group := newTestGroup(uuid.New())
	group.ID = groupID

	deps.groupRepo.On("FindByID", ctx, groupID).Return(group, nil)
	deps.membershipRepo.On("FindByGroupAndUser", ctx, groupID, nonMemberID).Return(nil, errors.New("not found"))

	q := deps.newQuery()
	output, err := q.Execute(ctx, query.ListInvitationsInput{
		GroupID:   groupID,
		RequestBy: nonMemberID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeForbidden, appErr.Code)
}
