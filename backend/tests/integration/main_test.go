// Package integration contains integration tests for the API
package integration

import (
	"os"
	"testing"

	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// TestMain is the entry point for all integration tests in this package
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup after all tests
	testutil.CleanupTestEnvironment()

	os.Exit(code)
}
