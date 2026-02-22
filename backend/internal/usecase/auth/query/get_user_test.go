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
	"github.com/Hiro-mackay/gc-storage/backend/internal/usecase/auth/query"
	"github.com/Hiro-mackay/gc-storage/backend/pkg/apperror"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil/mocks"
)

func TestGetUserQuery_Execute_ValidUserID_ReturnsUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	email, _ := valueobject.NewEmail("test@example.com")

	user := &entity.User{
		ID:            userID,
		Email:         email,
		Name:          "Test User",
		Status:        entity.UserStatusActive,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	userRepo := mocks.NewMockUserRepository(t)
	userRepo.On("FindByID", ctx, userID).Return(user, nil)

	q := query.NewGetUserQuery(userRepo)
	output, err := q.Execute(ctx, query.GetUserInput{UserID: userID})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, userID, output.User.ID)
	assert.Equal(t, "Test User", output.User.Name)
}

func TestGetUserQuery_Execute_UserNotFound_ReturnsNotFoundError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	userRepo := mocks.NewMockUserRepository(t)
	userRepo.On("FindByID", ctx, userID).Return(nil, errors.New("not found"))

	q := query.NewGetUserQuery(userRepo)
	output, err := q.Execute(ctx, query.GetUserInput{UserID: userID})

	require.Error(t, err)
	assert.Nil(t, output)
	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.CodeNotFound, appErr.Code)
}
