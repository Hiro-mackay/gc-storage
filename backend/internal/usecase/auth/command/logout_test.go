package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/command"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

func TestLogoutCommand_Execute_ValidSession_DeletesSession(t *testing.T) {
	ctx := context.Background()
	sessionID := "test-session-id"

	sessionRepo := mocks.NewMockSessionRepository(t)
	sessionRepo.On("Delete", ctx, sessionID).Return(nil)

	cmd := command.NewLogoutCommand(sessionRepo)
	err := cmd.Execute(ctx, sessionID)

	require.NoError(t, err)
}

func TestLogoutCommand_Execute_DeleteFails_ReturnsError(t *testing.T) {
	ctx := context.Background()
	sessionID := "test-session-id"

	sessionRepo := mocks.NewMockSessionRepository(t)
	sessionRepo.On("Delete", ctx, sessionID).Return(errors.New("redis error"))

	cmd := command.NewLogoutCommand(sessionRepo)
	err := cmd.Execute(ctx, sessionID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis error")
}

func TestLogoutCommand_ExecuteAll_ValidUserID_DeletesAllSessions(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	sessionRepo := mocks.NewMockSessionRepository(t)
	sessionRepo.On("DeleteByUserID", ctx, userID).Return(nil)

	cmd := command.NewLogoutCommand(sessionRepo)
	err := cmd.ExecuteAll(ctx, userID)

	require.NoError(t, err)
}
