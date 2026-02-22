package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

// MockEmailSender is a mock of service.EmailSender
type MockEmailSender struct {
	mock.Mock
}

func NewMockEmailSender(t *testing.T) *MockEmailSender {
	m := &MockEmailSender{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockEmailSender) SendEmailVerification(ctx context.Context, to, userName, verifyURL string) error {
	args := m.Called(ctx, to, userName, verifyURL)
	return args.Error(0)
}

func (m *MockEmailSender) SendPasswordReset(ctx context.Context, to, userName, resetURL string) error {
	args := m.Called(ctx, to, userName, resetURL)
	return args.Error(0)
}

func (m *MockEmailSender) SendGroupInvitation(ctx context.Context, to, userName, inviterName, groupName, inviteURL string) error {
	args := m.Called(ctx, to, userName, inviterName, groupName, inviteURL)
	return args.Error(0)
}
