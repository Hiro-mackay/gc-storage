// Package integration contains integration tests for the API
package integration

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/Hiro-mackay/gc-storage/backend/internal/domain/service"
	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// AuthTestSuite is the test suite for auth-related endpoints
type AuthTestSuite struct {
	suite.Suite
	server *testutil.TestServer
}

// SetupSuite runs once before all tests
func (s *AuthTestSuite) SetupSuite() {
	s.server = testutil.NewTestServer(s.T())
}

// TearDownSuite runs once after all tests
func (s *AuthTestSuite) TearDownSuite() {
	// Cleanup is handled by TestMain in main_test.go
}

// SetupTest runs before each test
func (s *AuthTestSuite) SetupTest() {
	s.server.Cleanup(s.T())
}

// TestMain is the entry point for the test suite
func TestAuthSuite(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
	}
	suite.Run(t, new(AuthTestSuite))
}

// =============================================================================
// Registration Tests
// =============================================================================

func (s *AuthTestSuite) TestRegister_Success() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "test@example.com",
			"password": "Password123",
			"name":     "Test User",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPathExists("data.user.id").
		AssertJSONPath("data.user.email", "test@example.com").
		AssertJSONPath("data.user.name", "Test User").
		AssertJSONPath("data.user.status", "pending").
		AssertJSONPath("data.user.email_verified", false)

	// Verify session cookie is set (auto-login)
	cookie := resp.GetCookie("session_id")
	s.NotNil(cookie, "session_id cookie should be set on registration")
	s.True(cookie.HttpOnly, "session_id cookie should be HttpOnly")
}

func (s *AuthTestSuite) TestRegister_InvalidEmail() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "invalid-email",
			"password": "Password123",
			"name":     "Test User",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestRegister_WeakPassword() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "test@example.com",
			"password": "weak",
			"name":     "Test User",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestRegister_DuplicateEmail() {
	// First registration
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "duplicate@example.com",
			"password": "Password123",
			"name":     "Test User",
		},
	}).AssertStatus(http.StatusCreated)

	// Second registration with same email
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "duplicate@example.com",
			"password": "Password456",
			"name":     "Another User",
		},
	})

	resp.AssertStatus(http.StatusConflict).
		AssertJSONError("CONFLICT", "email already exists")
}

func (s *AuthTestSuite) TestRegister_MissingFields() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body:   map[string]string{},
	})

	resp.AssertStatus(http.StatusBadRequest)
}

// =============================================================================
// Password Validation Tests - R-PW001
// Requirements: 8-256 characters, 2+ character types (uppercase, lowercase, numbers)
// =============================================================================

func (s *AuthTestSuite) TestRegister_Password_TooShort() {
	// R-PW001: Password must be at least 8 characters
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "short-pass@example.com",
			"password": "Abc123", // 6 characters - too short
			"name":     "Short Pass User",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestRegister_Password_TooLong() {
	// R-PW001: Password must not exceed 256 characters
	longPassword := ""
	for i := 0; i < 257; i++ {
		longPassword += "a"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "long-pass@example.com",
			"password": longPassword,
			"name":     "Long Pass User",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestRegister_Password_OnlyLowercase() {
	// R-PW001: Password must contain at least 2 character types
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "lowercase@example.com",
			"password": "abcdefgh", // Only lowercase - 1 character type
			"name":     "Lowercase User",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestRegister_Password_OnlyUppercase() {
	// R-PW001: Password must contain at least 2 character types
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "uppercase@example.com",
			"password": "ABCDEFGH", // Only uppercase - 1 character type
			"name":     "Uppercase User",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestRegister_Password_OnlyNumbers() {
	// R-PW001: Password must contain at least 2 character types
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "numbers@example.com",
			"password": "12345678", // Only numbers - 1 character type
			"name":     "Numbers User",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestRegister_Password_LowercaseAndNumbers_Valid() {
	// R-PW001: 2 character types is valid (lowercase + numbers)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "lower-num@example.com",
			"password": "abcd1234", // lowercase + numbers - valid
			"name":     "Lower Num User",
		},
	})

	resp.AssertStatus(http.StatusCreated)
}

func (s *AuthTestSuite) TestRegister_Password_UppercaseAndLowercase_Valid() {
	// R-PW001: 2 character types is valid (uppercase + lowercase)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "upper-lower@example.com",
			"password": "ABCDabcd", // uppercase + lowercase - valid
			"name":     "Upper Lower User",
		},
	})

	resp.AssertStatus(http.StatusCreated)
}

func (s *AuthTestSuite) TestRegister_Password_BoundaryMinLength() {
	// R-PW001: Exactly 8 characters is valid
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "min-length@example.com",
			"password": "Abcd1234", // Exactly 8 characters with 3 types - valid
			"name":     "Min Length User",
		},
	})

	resp.AssertStatus(http.StatusCreated)
}

// =============================================================================
// Personal Folder Auto-Creation Tests - R-U001
// Requirement: Personal Folder is automatically created on registration
// =============================================================================

func (s *AuthTestSuite) TestRegister_CreatesPersonalFolder() {
	// R-U001: User registration should create a Personal Folder
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "folder-test@example.com",
			"password": "Password123",
			"name":     "Folder Test User",
		},
	})
	resp.AssertStatus(http.StatusCreated)

	// First check if personal_folder_id is set
	var personalFolderID *string
	err := s.server.Pool.QueryRow(
		context.Background(),
		`SELECT personal_folder_id::text FROM users WHERE email = $1`,
		"folder-test@example.com",
	).Scan(&personalFolderID)
	s.Require().NoError(err)
	s.Require().NotNil(personalFolderID, "Personal Folder ID should be set")

	// Verify folder name
	var folderName string
	err = s.server.Pool.QueryRow(
		context.Background(),
		`SELECT name FROM folders WHERE id = $1::uuid`,
		*personalFolderID,
	).Scan(&folderName)
	s.Require().NoError(err)
	s.Equal("Folder Test User's folder", folderName)

	// Verify folder_paths table has self-reference
	var pathsCount int
	err = s.server.Pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM folder_paths WHERE ancestor_id = $1::uuid AND descendant_id = $1::uuid`,
		*personalFolderID,
	).Scan(&pathsCount)
	s.Require().NoError(err)
	s.Equal(1, pathsCount, "folder_paths table should have self-reference")
}

func (s *AuthTestSuite) TestOAuthLogin_NewUser_CreatesPersonalFolder() {
	// R-U001: OAuth new user should also have Personal Folder created

	// Setup mock OAuth client to return a unique new user
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "oauth-folder-test-user-123",
		Email:          "oauth-folder-test@example.com",
		Name:           "OAuth Folder User",
		AvatarURL:      "https://example.com/avatar.png",
	})

	// Perform OAuth login (endpoint is /api/v1/auth/oauth/:provider)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "mock-auth-code",
		},
	})
	resp.AssertStatus(http.StatusOK)

	// First check if personal_folder_id is set
	var personalFolderID *string
	err := s.server.Pool.QueryRow(
		context.Background(),
		`SELECT personal_folder_id::text FROM users WHERE email = $1`,
		"oauth-folder-test@example.com",
	).Scan(&personalFolderID)
	s.Require().NoError(err)
	s.Require().NotNil(personalFolderID, "Personal Folder ID should be set for OAuth user")

	// Verify folder name
	var folderName string
	err = s.server.Pool.QueryRow(
		context.Background(),
		`SELECT name FROM folders WHERE id = $1::uuid`,
		*personalFolderID,
	).Scan(&folderName)
	s.Require().NoError(err)
	s.Equal("OAuth Folder User's folder", folderName)

	// Verify folder_paths table has self-reference
	var pathsCount int
	err = s.server.Pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM folder_paths WHERE ancestor_id = $1::uuid AND descendant_id = $1::uuid`,
		*personalFolderID,
	).Scan(&pathsCount)
	s.Require().NoError(err)
	s.Equal(1, pathsCount, "folder_paths table should have self-reference for OAuth user")
}

// =============================================================================
// Session Limit Tests - R-SS002
// Requirement: Maximum 10 sessions per user
// =============================================================================

func (s *AuthTestSuite) TestLogin_SessionLimit_MaxTenSessions() {
	// R-SS002: User can have maximum 10 active sessions
	s.registerAndActivateUser("session-limit@example.com", "Password123", "Session Limit User")

	// Create 10 sessions
	var sessionIDs []string
	for i := 0; i < 10; i++ {
		resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/auth/login",
			Body: map[string]string{
				"email":    "session-limit@example.com",
				"password": "Password123",
			},
		})
		resp.AssertStatus(http.StatusOK)
		cookie := resp.GetCookie("session_id")
		s.Require().NotNil(cookie, "session_id cookie should be set")
		sessionIDs = append(sessionIDs, cookie.Value)

		// Clear rate limiter between logins (preserve sessions)
		testutil.ClearRateLimits(s.T(), s.server.Redis)
	}

	// All 10 sessions should work
	for _, sessionID := range sessionIDs {
		resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
			Method:    http.MethodGet,
			Path:      "/api/v1/me",
			SessionID: sessionID,
		})
		resp.AssertStatus(http.StatusOK)
	}
}

func (s *AuthTestSuite) TestLogin_SessionLimit_EleventhSessionRevokesOldest() {
	// R-SS002: When 11th session is created, oldest session should be revoked
	s.registerAndActivateUser("session-revoke@example.com", "Password123", "Session Revoke User")

	// Create 10 sessions
	var firstSessionID string
	for i := 0; i < 10; i++ {
		resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/auth/login",
			Body: map[string]string{
				"email":    "session-revoke@example.com",
				"password": "Password123",
			},
		})
		resp.AssertStatus(http.StatusOK)
		if i == 0 {
			cookie := resp.GetCookie("session_id")
			s.Require().NotNil(cookie, "first session_id cookie should exist")
			firstSessionID = cookie.Value
		}
		// Clear rate limiter between logins (preserve sessions)
		testutil.ClearRateLimits(s.T(), s.server.Redis)
	}

	// Create 11th session (should delete oldest)
	testutil.ClearRateLimits(s.T(), s.server.Redis)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "session-revoke@example.com",
			"password": "Password123",
		},
	})
	resp.AssertStatus(http.StatusOK)

	// First session should no longer work
	// (session was deleted when 11th was created)
	meResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: firstSessionID,
	})
	meResp.AssertStatus(http.StatusUnauthorized)
}

// =============================================================================
// Login Tests
// =============================================================================

func (s *AuthTestSuite) TestLogin_Success() {
	// Register user first
	s.registerAndActivateUser("login@example.com", "Password123", "Login User")

	// Login
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "login@example.com",
			"password": "Password123",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPathExists("data.user.id").
		AssertJSONPath("data.user.email", "login@example.com").
		AssertJSONPath("data.user.name", "Login User")

	// Verify session_id cookie is set
	cookie := resp.GetCookie("session_id")
	s.NotNil(cookie)
	s.True(cookie.HttpOnly)
}

func (s *AuthTestSuite) TestLogin_InvalidCredentials() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "nonexistent@example.com",
			"password": "Password123",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "invalid credentials")
}

func (s *AuthTestSuite) TestLogin_WrongPassword() {
	// Register user first
	s.registerAndActivateUser("wrongpass@example.com", "Password123", "Test User")

	// Login with wrong password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "wrongpass@example.com",
			"password": "WrongPassword",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "invalid credentials")
}

func (s *AuthTestSuite) TestLogin_PendingUser() {
	// Register but don't activate (FS-LOGIN-001: pending users can login)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "pending@example.com",
			"password": "Password123",
			"name":     "Pending User",
		},
	}).AssertStatus(http.StatusCreated)

	// Login should succeed for pending user
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "pending@example.com",
			"password": "Password123",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.user.status", "pending").
		AssertJSONPath("data.user.email_verified", false)
	cookie := resp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")
}

// =============================================================================
// Logout Tests
// =============================================================================

func (s *AuthTestSuite) TestLogout_Success() {
	// Register and login
	s.registerAndActivateUser("logout@example.com", "Password123", "Logout User")
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "logout@example.com",
			"password": "Password123",
		},
	})
	loginResp.AssertStatus(http.StatusOK)

	cookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")

	// Logout
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/logout",
		SessionID: cookie.Value,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "logged out successfully")
}

func (s *AuthTestSuite) TestLogout_SessionInvalidated() {
	// Register and login
	s.registerAndActivateUser("invalidate@example.com", "Password123", "Invalidate User")
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "invalidate@example.com",
			"password": "Password123",
		},
	})
	loginResp.AssertStatus(http.StatusOK)

	cookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")
	sessionID := cookie.Value

	// Logout
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/logout",
		SessionID: sessionID,
	}).AssertStatus(http.StatusOK)

	// Try to access protected endpoint with invalidated session
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "invalid session")
}

func (s *AuthTestSuite) TestLogout_NoSession() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/logout",
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "session not found")
}

// =============================================================================
// Protected Endpoint Tests
// =============================================================================

func (s *AuthTestSuite) TestProtectedEndpoint_WithValidSession() {
	// Register and login
	s.registerAndActivateUser("protected@example.com", "Password123", "Protected User")
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "protected@example.com",
			"password": "Password123",
		},
	})
	loginResp.AssertStatus(http.StatusOK)

	cookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")

	// Access protected endpoint
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: cookie.Value,
	})

	resp.AssertStatus(http.StatusOK)
}

func (s *AuthTestSuite) TestProtectedEndpoint_NoSession() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/me",
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "session not found")
}

func (s *AuthTestSuite) TestProtectedEndpoint_InvalidSession() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: "invalid-session-id",
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "invalid session")
}

// =============================================================================
// Email Verification Tests
// =============================================================================

func (s *AuthTestSuite) TestVerifyEmail_Success() {
	// Register user
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "verify@example.com",
			"password": "Password123",
			"name":     "Verify User",
		},
	}).AssertStatus(http.StatusCreated)

	// Get token from database
	token := s.getVerificationToken("verify@example.com")
	s.NotEmpty(token)

	// Verify email
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/verify?token=" + token,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "email verified successfully")

	// Verify user is now active
	var status string
	var emailVerifiedAt interface{}
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT status, email_verified_at FROM users WHERE email = $1",
		"verify@example.com",
	).Scan(&status, &emailVerifiedAt)
	s.Require().NoError(err)
	s.Equal("active", status)
	s.NotNil(emailVerifiedAt)
}

func (s *AuthTestSuite) TestVerifyEmail_InvalidToken() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/verify?token=invalid-token",
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "invalid or expired token")
}

func (s *AuthTestSuite) TestVerifyEmail_MissingToken() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/verify",
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "token is required")
}

func (s *AuthTestSuite) TestVerifyEmail_ExpiredToken() {
	// Register user
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "expired@example.com",
			"password": "Password123",
			"name":     "Expired User",
		},
	}).AssertStatus(http.StatusCreated)

	// Get token and expire it
	token := s.getVerificationToken("expired@example.com")
	_, err := s.server.Pool.Exec(
		context.Background(),
		"UPDATE email_verification_tokens SET expires_at = NOW() - INTERVAL '1 hour' WHERE token = $1",
		token,
	)
	s.Require().NoError(err)

	// Try to verify with expired token
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/verify?token=" + token,
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "token has expired")
}

func (s *AuthTestSuite) TestVerifyEmail_AlreadyVerified() {
	// Register and verify user
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "alreadyverified@example.com",
			"password": "Password123",
			"name":     "Already Verified",
		},
	}).AssertStatus(http.StatusCreated)

	token := s.getVerificationToken("alreadyverified@example.com")

	// First verification
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/verify?token=" + token,
	}).AssertStatus(http.StatusOK)

	// Create another token for the same user
	_, err := s.server.Pool.Exec(
		context.Background(),
		`INSERT INTO email_verification_tokens (user_id, token, expires_at)
		 SELECT id, 'second-token', NOW() + INTERVAL '24 hours' FROM users WHERE email = $1`,
		"alreadyverified@example.com",
	)
	s.Require().NoError(err)

	// Try to verify again with new token
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/verify?token=second-token",
	})

	// Should succeed but with "already verified" message
	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "email already verified")
}

// =============================================================================
// Resend Email Verification Tests
// =============================================================================

func (s *AuthTestSuite) TestResendEmailVerification_Success() {
	// Register user
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "resend@example.com",
			"password": "Password123",
			"name":     "Resend User",
		},
	}).AssertStatus(http.StatusCreated)

	// Get original token
	originalToken := s.getVerificationToken("resend@example.com")

	// Resend verification email
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/resend",
		Body: map[string]string{
			"email": "resend@example.com",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "If your email address is registered, a verification email has been sent.")

	// Verify new token was created (original should be deleted)
	newToken := s.getVerificationToken("resend@example.com")
	s.NotEmpty(newToken)
	s.NotEqual(originalToken, newToken)
}

func (s *AuthTestSuite) TestResendEmailVerification_NonExistentEmail() {
	// Security: should return same response for non-existent email
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/resend",
		Body: map[string]string{
			"email": "nonexistent@example.com",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "If your email address is registered, a verification email has been sent.")
}

func (s *AuthTestSuite) TestResendEmailVerification_AlreadyVerified() {
	// Register and verify user
	s.registerAndActivateUser("verified-resend@example.com", "Password123", "Verified User")

	// Resend verification email for already verified user
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/resend",
		Body: map[string]string{
			"email": "verified-resend@example.com",
		},
	})

	// Security: should return same response for already verified email
	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "If your email address is registered, a verification email has been sent.")
}

func (s *AuthTestSuite) TestResendEmailVerification_InvalidEmail() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/resend",
		Body: map[string]string{
			"email": "invalid-email",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

// =============================================================================
// Email Verification Flow Integration Tests
// =============================================================================

func (s *AuthTestSuite) TestFullEmailVerificationFlow() {
	// 1. Register
	registerResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "fullflow@example.com",
			"password": "Password123",
			"name":     "Full Flow User",
		},
	})
	registerResp.AssertStatus(http.StatusCreated)

	// 2. Login should succeed even for pending user (FS-LOGIN-001)
	pendingLoginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "fullflow@example.com",
			"password": "Password123",
		},
	})
	pendingLoginResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.user.status", "pending").
		AssertJSONPath("data.user.email_verified", false)

	// 3. Get verification token
	token := s.getVerificationToken("fullflow@example.com")

	// 4. Verify email
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/email/verify?token=" + token,
	}).AssertStatus(http.StatusOK)

	// 5. Login (should succeed)
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "fullflow@example.com",
			"password": "Password123",
		},
	})
	loginResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.user.email_verified", true).
		AssertJSONPath("data.user.status", "active")
	cookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")
}

// =============================================================================
// Helper Methods
// =============================================================================

// registerAndActivateUser registers a user and activates them (sets status to 'active')
func (s *AuthTestSuite) registerAndActivateUser(email, password, name string) {
	// Register
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    email,
			"password": password,
			"name":     name,
		},
	}).AssertStatus(http.StatusCreated)

	// Activate user in database (simulating email verification)
	_, err := s.server.Pool.Exec(
		context.Background(),
		"UPDATE users SET status = 'active', email_verified_at = NOW() WHERE email = $1",
		email,
	)
	s.Require().NoError(err)
}

// getVerificationToken gets the verification token for a user from the database
func (s *AuthTestSuite) getVerificationToken(email string) string {
	var token string
	err := s.server.Pool.QueryRow(
		context.Background(),
		`SELECT t.token FROM email_verification_tokens t
		 JOIN users u ON t.user_id = u.id
		 WHERE u.email = $1
		 ORDER BY t.created_at DESC LIMIT 1`,
		email,
	).Scan(&token)
	if err != nil {
		return ""
	}
	return token
}

// getPasswordResetToken gets the password reset token for a user from the database
func (s *AuthTestSuite) getPasswordResetToken(email string) string {
	var token string
	err := s.server.Pool.QueryRow(
		context.Background(),
		`SELECT t.token FROM password_reset_tokens t
		 JOIN users u ON t.user_id = u.id
		 WHERE u.email = $1
		 ORDER BY t.created_at DESC LIMIT 1`,
		email,
	).Scan(&token)
	if err != nil {
		return ""
	}
	return token
}

// =============================================================================
// Forgot Password Tests
// =============================================================================

func (s *AuthTestSuite) TestForgotPassword_Success() {
	// Register and activate user
	s.registerAndActivateUser("forgot@example.com", "Password123", "Forgot User")

	// Request password reset
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "forgot@example.com",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "If your email address is registered, a password reset email has been sent.")

	// Verify token was created
	token := s.getPasswordResetToken("forgot@example.com")
	s.NotEmpty(token)
}

func (s *AuthTestSuite) TestForgotPassword_NonExistentEmail() {
	// Security: should return same response for non-existent email
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "nonexistent@example.com",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "If your email address is registered, a password reset email has been sent.")
}

func (s *AuthTestSuite) TestForgotPassword_InvalidEmail() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "invalid-email",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestForgotPassword_PendingUser() {
	// Register but don't activate
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "pending-forgot@example.com",
			"password": "Password123",
			"name":     "Pending User",
		},
	}).AssertStatus(http.StatusCreated)

	// Request password reset (should work for pending users too)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "pending-forgot@example.com",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "If your email address is registered, a password reset email has been sent.")
}

// =============================================================================
// Reset Password Tests
// =============================================================================

func (s *AuthTestSuite) TestResetPassword_Success() {
	// Register and activate user
	s.registerAndActivateUser("reset@example.com", "Password123", "Reset User")

	// Request password reset
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "reset@example.com",
		},
	}).AssertStatus(http.StatusOK)

	// Get token
	token := s.getPasswordResetToken("reset@example.com")
	s.NotEmpty(token)

	// Reset password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    token,
			"password": "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "password reset successfully")

	// Verify can login with new password
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "reset@example.com",
			"password": "NewPassword456",
		},
	})
	loginResp.AssertStatus(http.StatusOK)
	cookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")

	// Verify cannot login with old password
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "reset@example.com",
			"password": "Password123",
		},
	}).AssertStatus(http.StatusUnauthorized)
}

func (s *AuthTestSuite) TestResetPassword_InvalidToken() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    "invalid-token",
			"password": "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "invalid or expired token")
}

func (s *AuthTestSuite) TestResetPassword_ExpiredToken() {
	// Register and activate user
	s.registerAndActivateUser("expired-reset@example.com", "Password123", "Expired Reset User")

	// Request password reset
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "expired-reset@example.com",
		},
	}).AssertStatus(http.StatusOK)

	// Get token and expire it
	token := s.getPasswordResetToken("expired-reset@example.com")
	_, err := s.server.Pool.Exec(
		context.Background(),
		"UPDATE password_reset_tokens SET expires_at = NOW() - INTERVAL '1 hour' WHERE token = $1",
		token,
	)
	s.Require().NoError(err)

	// Try to reset with expired token
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    token,
			"password": "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "token has expired")
}

func (s *AuthTestSuite) TestResetPassword_MissingToken() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"password": "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestResetPassword_WeakPassword() {
	// Register and activate user
	s.registerAndActivateUser("weak-reset@example.com", "Password123", "Weak Reset User")

	// Request password reset
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "weak-reset@example.com",
		},
	}).AssertStatus(http.StatusOK)

	// Get token
	token := s.getPasswordResetToken("weak-reset@example.com")

	// Try to reset with weak password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    token,
			"password": "weak",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestResetPassword_TokenAlreadyUsed() {
	// Register and activate user
	s.registerAndActivateUser("used-token@example.com", "Password123", "Used Token User")

	// Request password reset
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "used-token@example.com",
		},
	}).AssertStatus(http.StatusOK)

	// Get token
	token := s.getPasswordResetToken("used-token@example.com")

	// First reset (should succeed)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    token,
			"password": "NewPassword456",
		},
	}).AssertStatus(http.StatusOK)

	// Second reset with same token (should fail)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    token,
			"password": "AnotherPassword789",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "token has already been used")
}

// =============================================================================
// Password Reset Flow Integration Tests
// =============================================================================

func (s *AuthTestSuite) TestFullPasswordResetFlow() {
	// 1. Register and activate user
	s.registerAndActivateUser("fullreset@example.com", "OriginalPass123", "Full Reset User")

	// 2. Verify can login with original password
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "fullreset@example.com",
			"password": "OriginalPass123",
		},
	}).AssertStatus(http.StatusOK)

	// Clear rate limiter to avoid rate limit issues in this flow test
	testutil.FlushRedis(s.T(), s.server.Redis)

	// 3. Request password reset
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "fullreset@example.com",
		},
	}).AssertStatus(http.StatusOK)

	// 4. Get reset token
	token := s.getPasswordResetToken("fullreset@example.com")
	s.NotEmpty(token)

	// 5. Reset password
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    token,
			"password": "NewSecurePass456",
		},
	}).AssertStatus(http.StatusOK)

	// Clear rate limiter before login attempts
	testutil.FlushRedis(s.T(), s.server.Redis)

	// 6. Verify cannot login with old password
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "fullreset@example.com",
			"password": "OriginalPass123",
		},
	}).AssertStatus(http.StatusUnauthorized)

	// 7. Verify can login with new password
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "fullreset@example.com",
			"password": "NewSecurePass456",
		},
	})
	loginResp.AssertStatus(http.StatusOK)
	cookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")

	// Clear rate limiter before final check
	testutil.FlushRedis(s.T(), s.server.Redis)

	// 8. Verify token cannot be reused
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    token,
			"password": "AnotherPassword789",
		},
	}).AssertStatus(http.StatusBadRequest)
}

// =============================================================================
// Change Password Tests
// =============================================================================

func (s *AuthTestSuite) TestChangePassword_Success() {
	// Register, activate and login
	s.registerAndActivateUser("change@example.com", "OldPassword123", "Change User")
	sessionID := s.loginAndGetSessionID("change@example.com", "OldPassword123")

	// Change password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/change",
		SessionID: sessionID,
		Body: map[string]string{
			"current_password": "OldPassword123",
			"new_password":     "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "password changed successfully")
}

func (s *AuthTestSuite) TestChangePassword_WrongCurrentPassword() {
	// Register, activate and login
	s.registerAndActivateUser("wrongcurrent@example.com", "CorrectPassword123", "Wrong Current User")
	sessionID := s.loginAndGetSessionID("wrongcurrent@example.com", "CorrectPassword123")

	// Try to change password with wrong current password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/change",
		SessionID: sessionID,
		Body: map[string]string{
			"current_password": "WrongPassword123",
			"new_password":     "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "current password is incorrect")
}

func (s *AuthTestSuite) TestChangePassword_WeakNewPassword() {
	// Register, activate and login
	s.registerAndActivateUser("weaknew@example.com", "StrongPassword123", "Weak New User")
	sessionID := s.loginAndGetSessionID("weaknew@example.com", "StrongPassword123")

	// Try to change password with weak new password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/change",
		SessionID: sessionID,
		Body: map[string]string{
			"current_password": "StrongPassword123",
			"new_password":     "weak",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestChangePassword_MissingCurrentPassword() {
	// Register, activate and login
	s.registerAndActivateUser("missingcurrent@example.com", "Password123", "Missing Current User")
	sessionID := s.loginAndGetSessionID("missingcurrent@example.com", "Password123")

	// Try to change password without current password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/change",
		SessionID: sessionID,
		Body: map[string]string{
			"new_password": "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestChangePassword_MissingNewPassword() {
	// Register, activate and login
	s.registerAndActivateUser("missingnew@example.com", "Password123", "Missing New User")
	sessionID := s.loginAndGetSessionID("missingnew@example.com", "Password123")

	// Try to change password without new password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/change",
		SessionID: sessionID,
		Body: map[string]string{
			"current_password": "Password123",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestChangePassword_Unauthorized() {
	// Try to change password without session
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/change",
		Body: map[string]string{
			"current_password": "OldPassword123",
			"new_password":     "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized)
}

func (s *AuthTestSuite) TestFullChangePasswordFlow() {
	// 1. Register and activate user
	s.registerAndActivateUser("fullchange@example.com", "OriginalPass123", "Full Change User")

	// 2. Login and get session
	sessionID := s.loginAndGetSessionID("fullchange@example.com", "OriginalPass123")

	// 3. Change password
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/change",
		SessionID: sessionID,
		Body: map[string]string{
			"current_password": "OriginalPass123",
			"new_password":     "NewSecurePass456",
		},
	}).AssertStatus(http.StatusOK)

	// Clear rate limiter
	testutil.FlushRedis(s.T(), s.server.Redis)

	// 4. Verify cannot login with old password
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "fullchange@example.com",
			"password": "OriginalPass123",
		},
	}).AssertStatus(http.StatusUnauthorized)

	// 5. Verify can login with new password
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "fullchange@example.com",
			"password": "NewSecurePass456",
		},
	})
	loginResp.AssertStatus(http.StatusOK)
	newCookie := loginResp.GetCookie("session_id")
	s.NotNil(newCookie, "session_id cookie should be set")
}

// =============================================================================
// Additional Helper Methods
// =============================================================================

// loginAndGetSessionID logs in a user and returns the session ID from cookie
func (s *AuthTestSuite) loginAndGetSessionID(email, password string) string {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    email,
			"password": password,
		},
	})
	resp.AssertStatus(http.StatusOK)

	cookie := resp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")
	return cookie.Value
}

// =============================================================================
// OAuth Login Tests
// =============================================================================

func (s *AuthTestSuite) TestOAuthLogin_Google_NewUser_Success() {
	// Configure mock to return specific user info
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-new-user-123",
		Email:          "oauth-new@example.com",
		Name:           "OAuth New User",
		AvatarURL:      "https://example.com/avatar.png",
	})

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", true).
		AssertJSONPath("data.user.email", "oauth-new@example.com").
		AssertJSONPath("data.user.name", "OAuth New User").
		AssertJSONPath("data.user.status", "active").
		AssertJSONPath("data.user.email_verified", true)

	// Verify session_id cookie is set
	cookie := resp.GetCookie("session_id")
	s.NotNil(cookie)
	s.True(cookie.HttpOnly)
}

func (s *AuthTestSuite) TestOAuthLogin_Google_ExistingOAuthUser_Success() {
	// First OAuth login to create user
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-existing-user-456",
		Email:          "oauth-existing@example.com",
		Name:           "OAuth Existing User",
		AvatarURL:      "https://example.com/avatar.png",
	})

	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	}).AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", true)

	// Clear rate limiter
	testutil.FlushRedis(s.T(), s.server.Redis)

	// Second OAuth login with same user
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", false).
		AssertJSONPath("data.user.email", "oauth-existing@example.com")

	// Verify session_id cookie is set
	cookie := resp.GetCookie("session_id")
	s.NotNil(cookie)
}

func (s *AuthTestSuite) TestOAuthLogin_Google_ExistingEmailUser_LinkAccount() {
	// First, create a user via traditional registration
	s.registerAndActivateUser("link-oauth@example.com", "Password123", "Link OAuth User")

	// Configure mock with same email
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-link-user-789",
		Email:          "link-oauth@example.com",
		Name:           "Link OAuth User",
		AvatarURL:      "https://example.com/avatar.png",
	})

	// OAuth login should link the account
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", false).
		AssertJSONPath("data.user.email", "link-oauth@example.com")

	// Verify session_id cookie is set
	cookie := resp.GetCookie("session_id")
	s.NotNil(cookie)
}

func (s *AuthTestSuite) TestOAuthLogin_GitHub_Success() {
	// Configure GitHub mock
	s.server.MockGitHubClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "github-user-999",
		Email:          "oauth-github@example.com",
		Name:           "GitHub OAuth User",
		AvatarURL:      "https://example.com/github-avatar.png",
	})

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/github",
		Body: map[string]string{
			"code": "valid-github-auth-code",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", true).
		AssertJSONPath("data.user.email", "oauth-github@example.com").
		AssertJSONPath("data.user.name", "GitHub OAuth User")

	// Verify session_id cookie is set
	cookie := resp.GetCookie("session_id")
	s.NotNil(cookie)
}

func (s *AuthTestSuite) TestOAuthLogin_InvalidProvider() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/invalid-provider",
		Body: map[string]string{
			"code": "some-code",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "unsupported oauth provider")
}

func (s *AuthTestSuite) TestOAuthLogin_MissingCode() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body:   map[string]string{},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestOAuthLogin_InvalidCode() {
	// Mock returns error for "invalid-code"
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "invalid-code",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "invalid authorization code")
}

func (s *AuthTestSuite) TestOAuthLogin_PendingUserActivation() {
	// Register a pending user (not activated)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "pending-oauth@example.com",
			"password": "Password123",
			"name":     "Pending OAuth User",
		},
	}).AssertStatus(http.StatusCreated)

	// Configure mock with same email
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-pending-user-111",
		Email:          "pending-oauth@example.com",
		Name:           "Pending OAuth User",
		AvatarURL:      "https://example.com/avatar.png",
	})

	// OAuth login should activate the pending user
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", false).
		AssertJSONPath("data.user.status", "active").
		AssertJSONPath("data.user.email_verified", true)
}

func (s *AuthTestSuite) TestFullOAuthFlow() {
	// 1. New user OAuth login
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-fullflow-user",
		Email:          "oauth-fullflow@example.com",
		Name:           "OAuth Full Flow User",
		AvatarURL:      "https://example.com/avatar.png",
	})

	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})
	loginResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", true)

	cookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")
	sessionID := cookie.Value

	// 2. Access protected endpoint
	meResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionID,
	})
	meResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.email", "oauth-fullflow@example.com")

	// 3. Logout
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/logout",
		SessionID: sessionID,
	}).AssertStatus(http.StatusOK)

	// 4. Session should be invalidated
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionID,
	}).AssertStatus(http.StatusUnauthorized)

	// 5. Re-login with OAuth
	testutil.FlushRedis(s.T(), s.server.Redis)
	reLoginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})
	reLoginResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", false)
}

// =============================================================================
// Set Password Tests (for OAuth-only users)
// =============================================================================

func (s *AuthTestSuite) TestSetPassword_Success() {
	// Create OAuth-only user and get session
	sessionID := s.createOAuthUserAndGetSessionID("setpass@example.com", "Set Password User")

	// Set password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/set",
		SessionID: sessionID,
		Body: map[string]string{
			"password": "NewPassword123",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "password set successfully")

	// Clear rate limiter
	testutil.FlushRedis(s.T(), s.server.Redis)

	// Verify can now login with password
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "setpass@example.com",
			"password": "NewPassword123",
		},
	})
	loginResp.AssertStatus(http.StatusOK)
	cookie := loginResp.GetCookie("session_id")
	s.NotNil(cookie, "session_id cookie should be set")
}

func (s *AuthTestSuite) TestSetPassword_AlreadyHasPassword() {
	// Create user with password
	s.registerAndActivateUser("haspass@example.com", "ExistingPass123", "Has Password User")
	sessionID := s.loginAndGetSessionID("haspass@example.com", "ExistingPass123")

	// Try to set password (should fail - already has password)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/set",
		SessionID: sessionID,
		Body: map[string]string{
			"password": "NewPassword456",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "password already set, use change password instead")
}

func (s *AuthTestSuite) TestSetPassword_WeakPassword() {
	// Create OAuth-only user and get session
	sessionID := s.createOAuthUserAndGetSessionID("weaksetpass@example.com", "Weak Set Password User")

	// Try to set weak password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/set",
		SessionID: sessionID,
		Body: map[string]string{
			"password": "weak",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestSetPassword_MissingPassword() {
	// Create OAuth-only user and get session
	sessionID := s.createOAuthUserAndGetSessionID("missingsetpass@example.com", "Missing Set Password User")

	// Try without password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/set",
		SessionID: sessionID,
		Body:      map[string]string{},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *AuthTestSuite) TestSetPassword_Unauthorized() {
	// Try to set password without session
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/set",
		Body: map[string]string{
			"password": "NewPassword123",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized)
}

func (s *AuthTestSuite) TestFullOAuthToPasswordFlow() {
	// 1. Create OAuth-only user
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-fullset-user",
		Email:          "oauth-to-password@example.com",
		Name:           "OAuth To Password User",
		AvatarURL:      "https://example.com/avatar.png",
	})

	oauthResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})
	oauthResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", true)

	// 2. Verify cannot login with password (no password set)
	testutil.FlushRedis(s.T(), s.server.Redis)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "oauth-to-password@example.com",
			"password": "AnyPassword123",
		},
	}).AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "invalid credentials")

	// 3. Re-authenticate via OAuth to get a fresh session (FlushRedis cleared the previous one)
	testutil.FlushRedis(s.T(), s.server.Redis)
	oauthResp2 := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})
	oauthResp2.AssertStatus(http.StatusOK)
	cookie := oauthResp2.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")
	sessionID := cookie.Value

	// 4. Set password
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/set",
		SessionID: sessionID,
		Body: map[string]string{
			"password": "MyNewPassword123",
		},
	}).AssertStatus(http.StatusOK)

	// 4. Verify can now login with password
	testutil.FlushRedis(s.T(), s.server.Redis)
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "oauth-to-password@example.com",
			"password": "MyNewPassword123",
		},
	})
	loginResp.AssertStatus(http.StatusOK)
	loginCookie := loginResp.GetCookie("session_id")
	s.NotNil(loginCookie, "session_id cookie should be set")

	// 5. Verify can still login with OAuth
	testutil.FlushRedis(s.T(), s.server.Redis)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	}).AssertStatus(http.StatusOK).
		AssertJSONPath("data.is_new_user", false)
}

// =============================================================================
// Additional Helper Methods for Set Password
// =============================================================================

// createOAuthUserAndGetSessionID creates an OAuth-only user and returns session ID from cookie
func (s *AuthTestSuite) createOAuthUserAndGetSessionID(email, name string) string {
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-" + email,
		Email:          email,
		Name:           name,
		AvatarURL:      "https://example.com/avatar.png",
	})

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})
	resp.AssertStatus(http.StatusOK)

	cookie := resp.GetCookie("session_id")
	s.Require().NotNil(cookie, "session_id cookie should be set")
	return cookie.Value
}

// =============================================================================
// Session Invalidation Tests (Security)
// =============================================================================

func (s *AuthTestSuite) TestChangePassword_InvalidatesOtherSessions() {
	// Register and activate user
	s.registerAndActivateUser("change-inv@example.com", "Password123", "Change Invalidate User")

	// Login twice to create two sessions
	sessionA := s.loginAndGetSessionID("change-inv@example.com", "Password123")
	testutil.ClearRateLimits(s.T(), s.server.Redis)
	sessionB := s.loginAndGetSessionID("change-inv@example.com", "Password123")

	// Change password using sessionA
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/change",
		SessionID: sessionA,
		Body: map[string]string{
			"current_password": "Password123",
			"new_password":     "NewPassword456",
		},
	}).AssertStatus(http.StatusOK)

	// sessionA should still work (current session is preserved)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionA,
	}).AssertStatus(http.StatusOK)

	// sessionB should be invalidated
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionB,
	}).AssertStatus(http.StatusUnauthorized)
}

func (s *AuthTestSuite) TestResetPassword_InvalidatesAllSessions() {
	// Register and activate user
	s.registerAndActivateUser("reset-inv@example.com", "Password123", "Reset Invalidate User")

	// Login to create a session
	sessionA := s.loginAndGetSessionID("reset-inv@example.com", "Password123")

	// Verify session works
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionA,
	}).AssertStatus(http.StatusOK)

	// Request password reset
	testutil.ClearRateLimits(s.T(), s.server.Redis)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/forgot",
		Body: map[string]string{
			"email": "reset-inv@example.com",
		},
	}).AssertStatus(http.StatusOK)

	// Get reset token and reset password
	token := s.getPasswordResetToken("reset-inv@example.com")
	s.Require().NotEmpty(token)

	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/password/reset",
		Body: map[string]string{
			"token":    token,
			"password": "NewPassword789",
		},
	}).AssertStatus(http.StatusOK)

	// sessionA should be invalidated (all sessions deleted after password reset)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionA,
	}).AssertStatus(http.StatusUnauthorized)
}

func (s *AuthTestSuite) TestSetPassword_InvalidatesOtherSessions() {
	// Create OAuth-only user and login twice
	sessionA := s.createOAuthUserAndGetSessionID("setpass-inv@example.com", "SetPass Invalidate User")
	testutil.ClearRateLimits(s.T(), s.server.Redis)

	// Create second session via OAuth
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-setpass-inv@example.com",
		Email:          "setpass-inv@example.com",
		Name:           "SetPass Invalidate User",
		AvatarURL:      "https://example.com/avatar.png",
	})
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})
	resp.AssertStatus(http.StatusOK)
	cookieB := resp.GetCookie("session_id")
	s.Require().NotNil(cookieB, "session_id cookie should be set")
	sessionB := cookieB.Value

	// Set password using sessionA
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/password/set",
		SessionID: sessionA,
		Body: map[string]string{
			"password": "NewPassword123",
		},
	}).AssertStatus(http.StatusOK)

	// sessionA should still work (current session is preserved)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionA,
	}).AssertStatus(http.StatusOK)

	// sessionB should be invalidated
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me",
		SessionID: sessionB,
	}).AssertStatus(http.StatusUnauthorized)
}

// =============================================================================
// CSRF Cookie Tests (Security)
// =============================================================================

func (s *AuthTestSuite) TestLogin_SetsCsrfCookie() {
	s.registerAndActivateUser("csrf-login@example.com", "Password123", "CSRF Login User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "csrf-login@example.com",
			"password": "Password123",
		},
	})
	resp.AssertStatus(http.StatusOK)

	// Verify csrf_token cookie exists
	csrfCookie := resp.GetCookie("csrf_token")
	s.Require().NotNil(csrfCookie, "csrf_token cookie should be set on login")
	s.NotEmpty(csrfCookie.Value, "csrf_token cookie should have a value")

	// Verify cookie attributes
	s.False(csrfCookie.HttpOnly, "csrf_token should NOT be HttpOnly (JS needs to read it)")
	s.True(csrfCookie.Secure, "csrf_token should be Secure")
	s.Equal(http.SameSiteLaxMode, csrfCookie.SameSite, "csrf_token should have SameSite=Lax")
}

func (s *AuthTestSuite) TestOAuthLogin_SetsCsrfCookie() {
	s.server.MockGoogleClient.SetUserInfo(&service.OAuthUserInfo{
		ProviderUserID: "google-csrf-oauth-user",
		Email:          "csrf-oauth@example.com",
		Name:           "CSRF OAuth User",
		AvatarURL:      "https://example.com/avatar.png",
	})

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/oauth/google",
		Body: map[string]string{
			"code": "valid-google-auth-code",
		},
	})
	resp.AssertStatus(http.StatusOK)

	// Verify csrf_token cookie exists
	csrfCookie := resp.GetCookie("csrf_token")
	s.Require().NotNil(csrfCookie, "csrf_token cookie should be set on OAuth login")
	s.NotEmpty(csrfCookie.Value, "csrf_token cookie should have a value")
}

func (s *AuthTestSuite) TestLogout_ClearsCsrfCookie() {
	// Login first
	s.registerAndActivateUser("csrf-logout@example.com", "Password123", "CSRF Logout User")
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "csrf-logout@example.com",
			"password": "Password123",
		},
	})
	loginResp.AssertStatus(http.StatusOK)
	sessionCookie := loginResp.GetCookie("session_id")
	s.Require().NotNil(sessionCookie, "session_id cookie should be set")

	// Verify CSRF cookie was set on login
	csrfLoginCookie := loginResp.GetCookie("csrf_token")
	s.Require().NotNil(csrfLoginCookie, "csrf_token cookie should be set on login")

	// Logout
	logoutResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/auth/logout",
		SessionID: sessionCookie.Value,
	})
	logoutResp.AssertStatus(http.StatusOK)

	// Verify csrf_token cookie is cleared (MaxAge <= 0)
	csrfLogoutCookie := logoutResp.GetCookie("csrf_token")
	s.Require().NotNil(csrfLogoutCookie, "csrf_token cookie should be present in logout response")
	s.True(csrfLogoutCookie.MaxAge < 0, "csrf_token MaxAge should be negative (cookie deleted)")
}
