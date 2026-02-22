package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

// MockTransactionManager is a mock of repository.TransactionManager
type MockTransactionManager struct {
	mock.Mock
}

func NewMockTransactionManager(t *testing.T) *MockTransactionManager {
	m := &MockTransactionManager{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

// WithTransaction executes the function directly without a real transaction
func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Simply execute the function to simulate a successful transaction
	return fn(ctx)
}
