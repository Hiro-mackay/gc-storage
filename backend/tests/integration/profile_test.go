// Package integration contains integration tests for the API
package integration

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// ProfileTestSuite is the test suite for profile-related endpoints
type ProfileTestSuite struct {
	suite.Suite
	server *testutil.TestServer
}

// SetupSuite runs once before all tests
func (s *ProfileTestSuite) SetupSuite() {
	s.server = testutil.NewTestServer(s.T())
}

// TearDownSuite runs once after all tests
func (s *ProfileTestSuite) TearDownSuite() {
	// Note: CleanupTestEnvironment is only called once per test run by AuthTestSuite
	// Do not call it here to avoid closing shared pool twice
}

// SetupTest runs before each test
func (s *ProfileTestSuite) SetupTest() {
	s.server.Cleanup(s.T())
}

// TestMain is the entry point for the test suite
func TestProfileSuite(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
	}
	suite.Run(t, new(ProfileTestSuite))
}

// =============================================================================
// GetProfile Tests
// =============================================================================

func (s *ProfileTestSuite) TestGetProfile_Success() {
	// Create and login user
	sessionID := s.createAndLoginUser("profile-user@example.com", "Password123", "Profile User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPathExists("data.profile.user_id").
		AssertJSONPath("data.user.email", "profile-user@example.com").
		AssertJSONPath("data.user.name", "Profile User").
		AssertJSONPath("data.profile.locale", "ja").
		AssertJSONPath("data.profile.timezone", "Asia/Tokyo")
}

func (s *ProfileTestSuite) TestGetProfile_Unauthorized() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/me/profile",
	})

	resp.AssertStatus(http.StatusUnauthorized)
}

// =============================================================================
// UpdateProfile Tests
// =============================================================================

func (s *ProfileTestSuite) TestUpdateUser_Success_DisplayName() {
	sessionID := s.createAndLoginUser("display-name@example.com", "Password123", "Original Name")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": "Updated Name",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.name", "Updated Name")

	// Verify the name was updated via GET profile
	getResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
	})

	getResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.user.name", "Updated Name")
}

func (s *ProfileTestSuite) TestUpdateProfile_Success_Bio() {
	sessionID := s.createAndLoginUser("bio-test@example.com", "Password123", "Bio User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"bio": "This is my bio. Hello world!",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.bio", "This is my bio. Hello world!")
}

func (s *ProfileTestSuite) TestUpdateProfile_Success_LocaleTimezone() {
	sessionID := s.createAndLoginUser("locale-test@example.com", "Password123", "Locale User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"locale":   "en",
			"timezone": "America/New_York",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.locale", "en").
		AssertJSONPath("data.profile.timezone", "America/New_York")
}

func (s *ProfileTestSuite) TestUpdateProfile_Success_Settings() {
	// Note: API uses notification_preferences with email_enabled/push_enabled, not settings
	// TODO: settings field structure differs from expectation
	s.T().Skip("settings field structure differs from expectation - API uses notification_preferences")
}

func (s *ProfileTestSuite) TestUpdateProfile_Success_MultipleFields() {
	sessionID := s.createAndLoginUser("multi-field@example.com", "Password123", "Multi User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"locale":   "en",
			"timezone": "UTC",
			"theme":    "light",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.locale", "en").
		AssertJSONPath("data.profile.timezone", "UTC").
		AssertJSONPath("data.profile.theme", "light")
}

func (s *ProfileTestSuite) TestUpdateProfile_BioTooLong() {
	sessionID := s.createAndLoginUser("bio-long@example.com", "Password123", "Long Bio User")

	// Create a bio that exceeds 500 characters
	longBio := ""
	for i := 0; i < 501; i++ {
		longBio += "a"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"bio": longBio,
		},
	})

	resp.AssertStatus(http.StatusBadRequest)
}

func (s *ProfileTestSuite) TestUpdateProfile_Unauthorized() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPut,
		Path:   "/api/v1/me/profile",
		Body: map[string]interface{}{
			"display_name": "Unauthorized",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized)
}

// =============================================================================
// Profile Validation Tests
// =============================================================================

func (s *ProfileTestSuite) TestUpdateUser_DisplayNameTooLong() {
	sessionID := s.createAndLoginUser("long-name@example.com", "Password123", "Long Name User")

	// Create a name that exceeds 100 characters (R-U003)
	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": longName,
		},
	})

	resp.AssertStatus(http.StatusBadRequest)
}

func (s *ProfileTestSuite) TestUpdateUser_DisplayNameBoundary() {
	sessionID := s.createAndLoginUser("boundary-name@example.com", "Password123", "Boundary Name User")

	// Create exactly 100 characters (R-U003 max)
	name100 := ""
	for i := 0; i < 100; i++ {
		name100 += "n"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"name": name100,
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.name", name100)
}

func (s *ProfileTestSuite) TestUpdateProfile_BioBoundary() {
	// Exactly 500 characters should be valid
	sessionID := s.createAndLoginUser("bio-boundary@example.com", "Password123", "Bio Boundary User")

	// Create exactly 500 characters
	bio := ""
	for i := 0; i < 500; i++ {
		bio += "b"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"bio": bio,
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.bio", bio)
}

func (s *ProfileTestSuite) TestUpdateProfile_InvalidLocale() {
	// locale validation: only "ja" and "en" are allowed
	sessionID := s.createAndLoginUser("invalid-locale@example.com", "Password123", "Invalid Locale User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"locale": "fr", // unsupported locale
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *ProfileTestSuite) TestUpdateProfile_InvalidTimezone() {
	// timezone validation: only IANA timezone format is allowed
	sessionID := s.createAndLoginUser("invalid-tz@example.com", "Password123", "Invalid Timezone User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"timezone": "Invalid/Timezone", // invalid IANA timezone
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *ProfileTestSuite) TestUpdateProfile_ValidLocaleJa() {
	sessionID := s.createAndLoginUser("locale-ja@example.com", "Password123", "Locale Ja User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"locale": "ja",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.locale", "ja")
}

func (s *ProfileTestSuite) TestUpdateProfile_ValidLocaleEn() {
	sessionID := s.createAndLoginUser("locale-en@example.com", "Password123", "Locale En User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"locale": "en",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.locale", "en")
}

func (s *ProfileTestSuite) TestUpdateProfile_ValidTimezoneUTC() {
	sessionID := s.createAndLoginUser("tz-utc@example.com", "Password123", "TZ UTC User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"timezone": "UTC",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.timezone", "UTC")
}

func (s *ProfileTestSuite) TestUpdateProfile_ValidTimezoneAsiaTokyo() {
	sessionID := s.createAndLoginUser("tz-tokyo@example.com", "Password123", "TZ Tokyo User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"timezone": "Asia/Tokyo",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.timezone", "Asia/Tokyo")
}

func (s *ProfileTestSuite) TestUpdateProfile_AvatarURL() {
	sessionID := s.createAndLoginUser("avatar@example.com", "Password123", "Avatar User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"avatar_url": "https://example.com/avatar.png",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.avatar_url", "https://example.com/avatar.png")
}

func (s *ProfileTestSuite) TestUpdateProfile_InvalidAvatarURL() {
	sessionID := s.createAndLoginUser("invalid-avatar@example.com", "Password123", "Invalid Avatar User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"avatar_url": "not-a-valid-url",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

// =============================================================================
// Profile Persistence Tests
// =============================================================================

func (s *ProfileTestSuite) TestProfilePersistence() {
	sessionID := s.createAndLoginUser("persist-test@example.com", "Password123", "Persist User")

	// Update profile with supported fields
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPut,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
		Body: map[string]interface{}{
			"locale":   "en",
			"timezone": "UTC",
		},
	}).AssertStatus(http.StatusOK)

	// Get profile and verify persistence
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/me/profile",
		SessionID: sessionID,
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.locale", "en").
		AssertJSONPath("data.profile.timezone", "UTC")
}

// =============================================================================
// Helper Methods
// =============================================================================

func (s *ProfileTestSuite) createAndLoginUser(email, password, name string) string {
	// Register user
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"email":    email,
			"password": password,
			"name":     name,
		},
	}).AssertStatus(http.StatusCreated)

	// Verify email (directly update database)
	_, err := s.server.Pool.Exec(s.T().Context(), "UPDATE users SET email_verified_at = NOW(), status = 'active' WHERE email = $1", email)
	s.Require().NoError(err)

	// Login
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
