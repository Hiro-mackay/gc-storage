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
	accessToken := s.createAndLoginUser("profile-user@example.com", "Password123", "Profile User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPathExists("data.user_id").
		AssertJSONPath("data.email", "profile-user@example.com").
		AssertJSONPath("data.name", "Profile User").
		AssertJSONPath("data.locale", "ja").
		AssertJSONPath("data.timezone", "Asia/Tokyo").
		AssertJSONPathExists("data.settings")
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

func (s *ProfileTestSuite) TestUpdateProfile_Success_DisplayName() {
	accessToken := s.createAndLoginUser("update-profile@example.com", "Password123", "Test User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPut,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"display_name": "New Display Name",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.display_name", "New Display Name").
		AssertJSONPath("data.message", "profile updated successfully")
}

func (s *ProfileTestSuite) TestUpdateProfile_Success_Bio() {
	accessToken := s.createAndLoginUser("bio-test@example.com", "Password123", "Bio User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPut,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"bio": "This is my bio. Hello world!",
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.bio", "This is my bio. Hello world!")
}

func (s *ProfileTestSuite) TestUpdateProfile_Success_LocaleTimezone() {
	accessToken := s.createAndLoginUser("locale-test@example.com", "Password123", "Locale User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPut,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
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
	accessToken := s.createAndLoginUser("settings-test@example.com", "Password123", "Settings User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPut,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"settings": map[string]interface{}{
				"notifications_enabled": false,
				"email_notifications":   false,
				"theme":                 "dark",
			},
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.settings.notifications_enabled", false).
		AssertJSONPath("data.profile.settings.email_notifications", false).
		AssertJSONPath("data.profile.settings.theme", "dark")
}

func (s *ProfileTestSuite) TestUpdateProfile_Success_MultipleFields() {
	accessToken := s.createAndLoginUser("multi-field@example.com", "Password123", "Multi User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPut,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"display_name": "Display Name",
			"bio":          "My bio",
			"locale":       "en",
			"timezone":     "UTC",
			"settings": map[string]interface{}{
				"theme": "light",
			},
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.profile.display_name", "Display Name").
		AssertJSONPath("data.profile.bio", "My bio").
		AssertJSONPath("data.profile.locale", "en").
		AssertJSONPath("data.profile.timezone", "UTC").
		AssertJSONPath("data.profile.settings.theme", "light")
}

func (s *ProfileTestSuite) TestUpdateProfile_BioTooLong() {
	accessToken := s.createAndLoginUser("bio-long@example.com", "Password123", "Long Bio User")

	// Create a bio that exceeds 500 characters
	longBio := ""
	for i := 0; i < 501; i++ {
		longBio += "a"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPut,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
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
// Profile Persistence Tests
// =============================================================================

func (s *ProfileTestSuite) TestProfilePersistence() {
	accessToken := s.createAndLoginUser("persist-test@example.com", "Password123", "Persist User")

	// Update profile
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPut,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
		Body: map[string]interface{}{
			"display_name": "Persisted Name",
			"bio":          "Persisted Bio",
		},
	}).AssertStatus(http.StatusOK)

	// Get profile and verify persistence
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/me/profile",
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	})

	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.display_name", "Persisted Name").
		AssertJSONPath("data.bio", "Persisted Bio")
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
	data := resp.GetJSONData()
	return data["access_token"].(string)
}
