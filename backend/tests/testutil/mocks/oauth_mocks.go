package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/entity"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/valueobject"
)

// MockOAuthAccountRepository is a mock of repository.OAuthAccountRepository
type MockOAuthAccountRepository struct {
	mock.Mock
}

func NewMockOAuthAccountRepository(t *testing.T) *MockOAuthAccountRepository {
	m := &MockOAuthAccountRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockOAuthAccountRepository) Create(ctx context.Context, account *entity.OAuthAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockOAuthAccountRepository) Update(ctx context.Context, account *entity.OAuthAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockOAuthAccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.OAuthAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.OAuthAccount), args.Error(1)
}

func (m *MockOAuthAccountRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.OAuthAccount, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.OAuthAccount), args.Error(1)
}

func (m *MockOAuthAccountRepository) FindByProviderAndUserID(ctx context.Context, provider valueobject.OAuthProvider, providerUserID string) (*entity.OAuthAccount, error) {
	args := m.Called(ctx, provider, providerUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.OAuthAccount), args.Error(1)
}

func (m *MockOAuthAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockOAuthClient is a mock of service.OAuthClient
type MockOAuthClient struct {
	mock.Mock
}

func NewMockOAuthClient(t *testing.T) *MockOAuthClient {
	m := &MockOAuthClient{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockOAuthClient) ExchangeCode(ctx context.Context, code string) (*service.OAuthTokens, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.OAuthTokens), args.Error(1)
}

func (m *MockOAuthClient) GetUserInfo(ctx context.Context, accessToken string) (*service.OAuthUserInfo, error) {
	args := m.Called(ctx, accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.OAuthUserInfo), args.Error(1)
}

func (m *MockOAuthClient) Provider() valueobject.OAuthProvider {
	args := m.Called()
	return args.Get(0).(valueobject.OAuthProvider)
}

// MockOAuthClientFactory is a mock of service.OAuthClientFactory
type MockOAuthClientFactory struct {
	mock.Mock
}

func NewMockOAuthClientFactory(t *testing.T) *MockOAuthClientFactory {
	m := &MockOAuthClientFactory{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockOAuthClientFactory) GetClient(provider valueobject.OAuthProvider) (service.OAuthClient, error) {
	args := m.Called(provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(service.OAuthClient), args.Error(1)
}
