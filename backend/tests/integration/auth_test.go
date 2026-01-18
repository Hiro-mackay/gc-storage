// Package integration contains integration tests for the API
package integration

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

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
	testutil.CleanupTestEnvironment()
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
		AssertJSONPathExists("data.user_id").
		AssertJSONPath("data.message", "Registration successful. Please check your email to verify your account.")
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
		AssertJSONPathExists("data.access_token").
		AssertJSONPathExists("data.expires_in").
		AssertJSONPathExists("data.user.id").
		AssertJSONPath("data.user.email", "login@example.com").
		AssertJSONPath("data.user.name", "Login User")

	// Verify refresh token cookie is set
	cookie := resp.GetCookie("refresh_token")
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
	// Register but don't activate
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    "pending@example.com",
			"password": "Password123",
			"name":     "Pending User",
		},
	}).AssertStatus(http.StatusCreated)

	// Try to login (should fail because user is pending)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "pending@example.com",
			"password": "Password123",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "please verify your email first")
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

	data := loginResp.GetJSONData()
	accessToken := data["access_token"].(string)

	// Logout
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		AccessToken: accessToken,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.message", "logged out successfully")
}

func (s *AuthTestSuite) TestLogout_TokenBlacklisted() {
	// Register and login
	s.registerAndActivateUser("blacklist@example.com", "Password123", "Blacklist User")
	loginResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "blacklist@example.com",
			"password": "Password123",
		},
	})
	loginResp.AssertStatus(http.StatusOK)

	data := loginResp.GetJSONData()
	accessToken := data["access_token"].(string)

	// Logout
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		AccessToken: accessToken,
	}).AssertStatus(http.StatusOK)

	// Try to access protected endpoint with blacklisted token
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/me",
		AccessToken: accessToken,
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "token has been revoked")
}

func (s *AuthTestSuite) TestLogout_NoToken() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/logout",
	})

	resp.AssertStatus(http.StatusUnauthorized)
}

// =============================================================================
// Protected Endpoint Tests
// =============================================================================

func (s *AuthTestSuite) TestProtectedEndpoint_WithValidToken() {
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

	data := loginResp.GetJSONData()
	accessToken := data["access_token"].(string)

	// Access protected endpoint
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/me",
		AccessToken: accessToken,
	})

	// /me returns null data because it's not implemented yet
	resp.AssertStatus(http.StatusOK)
}

func (s *AuthTestSuite) TestProtectedEndpoint_NoToken() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/me",
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "authorization header required")
}

func (s *AuthTestSuite) TestProtectedEndpoint_InvalidToken() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/me",
		AccessToken: "invalid-token",
	})

	resp.AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "invalid or expired token")
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

	// 2. Try to login (should fail - pending)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body: map[string]string{
			"email":    "fullflow@example.com",
			"password": "Password123",
		},
	}).AssertStatus(http.StatusUnauthorized).
		AssertJSONError("UNAUTHORIZED", "please verify your email first")

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
		AssertJSONPathExists("data.access_token").
		AssertJSONPath("data.user.email_verified", true).
		AssertJSONPath("data.user.status", "active")
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
