package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/collaboration/command"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

type createGroupTestDeps struct {
	groupRepo      *mocks.MockGroupRepository
	membershipRepo *mocks.MockMembershipRepository
	txManager      *mocks.MockTransactionManager
}

func newCreateGroupTestDeps(t *testing.T) *createGroupTestDeps {
	t.Helper()
	return &createGroupTestDeps{
		groupRepo:      mocks.NewMockGroupRepository(t),
		membershipRepo: mocks.NewMockMembershipRepository(t),
		txManager:      mocks.NewMockTransactionManager(t),
	}
}

func (d *createGroupTestDeps) newCommand() *command.CreateGroupCommand {
	return command.NewCreateGroupCommand(d.groupRepo, d.membershipRepo, d.txManager)
}

func TestCreateGroupCommand_Execute_ValidInput_CreatesGroupAndOwnerMembership(t *testing.T) {
	ctx := context.Background()
	deps := newCreateGroupTestDeps(t)
	ownerID := uuid.New()

	deps.groupRepo.On("Create", ctx, mock.MatchedBy(func(g *entity.Group) bool {
		return g.OwnerID == ownerID
	})).Return(nil)
	deps.membershipRepo.On("Create", ctx, mock.MatchedBy(func(m *entity.Membership) bool {
		return m.UserID == ownerID && m.Role == valueobject.GroupRoleOwner
	})).Return(nil)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.CreateGroupInput{
		Name:        "My Group",
		Description: "A test group",
		OwnerID:     ownerID,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.NotNil(t, output.Group)
	assert.NotNil(t, output.Membership)
	assert.Equal(t, ownerID, output.Group.OwnerID)
	assert.Equal(t, ownerID, output.Membership.UserID)
	assert.Equal(t, valueobject.GroupRoleOwner, output.Membership.Role)
}

func TestCreateGroupCommand_Execute_InvalidGroupName_ValidationError(t *testing.T) {
	ctx := context.Background()
	deps := newCreateGroupTestDeps(t)

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.CreateGroupInput{
		Name:    "",
		OwnerID: uuid.New(),
	})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeValidationError, appErr.Code)
}

func TestCreateGroupCommand_Execute_GroupRepoFails_ReturnsError(t *testing.T) {
	ctx := context.Background()
	deps := newCreateGroupTestDeps(t)
	ownerID := uuid.New()

	deps.groupRepo.On("Create", ctx, mock.AnythingOfType("*entity.Group")).Return(errors.New("db error"))

	cmd := deps.newCommand()
	output, err := cmd.Execute(ctx, command.CreateGroupInput{
		Name:    "My Group",
		OwnerID: ownerID,
	})

	require.Error(t, err)
	assert.Nil(t, output)
}
