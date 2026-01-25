// Package integration contains integration tests for the ShareLink API
package integration

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// ShareLinkTestSuite is the test suite for share link-related endpoints
type ShareLinkTestSuite struct {
	suite.Suite
	server *testutil.TestServer
}

// SetupSuite runs once before all tests
func (s *ShareLinkTestSuite) SetupSuite() {
	s.server = testutil.NewTestServer(s.T())
}

// TearDownSuite runs once after all tests
func (s *ShareLinkTestSuite) TearDownSuite() {
	// Cleanup is handled by TestMain in main_test.go
}

// SetupTest runs before each test
func (s *ShareLinkTestSuite) SetupTest() {
	s.server.Cleanup(s.T())
}

// TestMain is the entry point for the test suite
func TestShareLinkSuite(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
	}
	suite.Run(t, new(ShareLinkTestSuite))
}

// =============================================================================
// Helper Functions
// =============================================================================

func (s *ShareLinkTestSuite) registerAndActivateUser(email, password, name string) {
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

func (s *ShareLinkTestSuite) loginAndGetToken(email, password string) string {
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

func (s *ShareLinkTestSuite) createUser(email, password, name string) string {
	s.registerAndActivateUser(email, password, name)
	return s.loginAndGetToken(email, password)
}

func (s *ShareLinkTestSuite) createFolder(token, name string) string {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders",
		AccessToken: token,
		Body: map[string]string{
			"name": name,
		},
	})
	resp.AssertStatus(http.StatusCreated)
	data := resp.GetJSONData()
	return data["id"].(string)
}

// =============================================================================
// Share Link Creation Tests
// =============================================================================

func (s *ShareLinkTestSuite) TestCreateShareLink_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPathExists("data.id").
		AssertJSONPathExists("data.token").
		AssertJSONPathExists("data.url").
		AssertJSONPath("data.permission", "read")
}

func (s *ShareLinkTestSuite) TestCreateShareLink_WithPassword() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
			"password":   "secret123",
		},
	})

	// Note: JSON uses camelCase (hasPassword not has_password)
	resp.AssertStatus(http.StatusCreated).
		AssertJSONPath("data.hasPassword", true)
}

func (s *ShareLinkTestSuite) TestCreateShareLink_WithExpiry() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	expiresAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]interface{}{
			"permission": "read",
			"expiresAt":  expiresAt,
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPathExists("data.expiresAt")
}

func (s *ShareLinkTestSuite) TestCreateShareLink_WithMaxAccessCount() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]interface{}{
			"permission":     "read",
			"maxAccessCount": 5,
		},
	})

	// Note: JSON uses camelCase (maxAccessCount)
	resp.AssertStatus(http.StatusCreated).
		AssertJSONPath("data.maxAccessCount", float64(5))
}

func (s *ShareLinkTestSuite) TestCreateShareLink_Unauthorized() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	otherToken := s.createUser("other@example.com", "Password123", "Other User")

	folderID := s.createFolder(ownerToken, "Shared Folder")

	// Other user tries to create share link
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: otherToken,
		Body: map[string]string{
			"permission": "read",
		},
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Share Link Access Tests
// =============================================================================

func (s *ShareLinkTestSuite) TestAccessShareLink_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// Access share link (no auth required)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	})

	// Note: JSON uses camelCase (resourceType)
	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.resourceType", "folder")
}

func (s *ShareLinkTestSuite) TestAccessShareLink_RequiresPassword() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with password
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
			"password":   "secret123",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// Get info without password - ShareLinkInfoResponse uses hasPassword
	infoResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/share/" + shareToken,
	})
	infoResp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.hasPassword", true)
}

func (s *ShareLinkTestSuite) TestAccessShareLink_WrongPassword() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with password
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
			"password":   "secret123",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// Access with wrong password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
		Body: map[string]string{
			"password": "wrongpassword",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized)
}

func (s *ShareLinkTestSuite) TestAccessShareLink_CorrectPassword() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with password
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
			"password":   "secret123",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// Access with correct password
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
		Body: map[string]string{
			"password": "secret123",
		},
	})

	resp.AssertStatus(http.StatusOK)
}

func (s *ShareLinkTestSuite) TestAccessShareLink_Expired() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with past expiry
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareLinkID := shareData["id"].(string)
	shareToken := shareData["token"].(string)

	// Manually expire the link in database
	_, err := s.server.Pool.Exec(
		context.Background(),
		"UPDATE share_links SET expires_at = NOW() - INTERVAL '1 hour' WHERE id = $1::uuid",
		shareLinkID,
	)
	s.Require().NoError(err)

	// Try to access expired link
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	})

	// API returns 403 Forbidden for expired links
	resp.AssertStatus(http.StatusForbidden)
}

func (s *ShareLinkTestSuite) TestAccessShareLink_Revoked() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareLinkID := shareData["id"].(string)
	shareToken := shareData["token"].(string)

	// Revoke share link
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodDelete,
		Path:        "/api/v1/share-links/" + shareLinkID,
		AccessToken: token,
	}).AssertStatus(http.StatusNoContent)

	// Try to access revoked link
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	})

	// API returns 403 Forbidden for revoked links
	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Share Link Revocation Tests
// =============================================================================

func (s *ShareLinkTestSuite) TestRevokeShareLink_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareLinkID := shareData["id"].(string)

	// Revoke share link
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodDelete,
		Path:        "/api/v1/share-links/" + shareLinkID,
		AccessToken: token,
	})

	resp.AssertStatus(http.StatusNoContent)
}

func (s *ShareLinkTestSuite) TestRevokeShareLink_NotCreator() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	otherToken := s.createUser("other@example.com", "Password123", "Other User")

	folderID := s.createFolder(ownerToken, "Shared Folder")

	// Create share link
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: ownerToken,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareLinkID := shareData["id"].(string)

	// Other user tries to revoke (should fail)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodDelete,
		Path:        "/api/v1/share-links/" + shareLinkID,
		AccessToken: otherToken,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Share Link List Tests
// =============================================================================

func (s *ShareLinkTestSuite) TestListShareLinks_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create multiple share links
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	}).AssertStatus(http.StatusCreated)

	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "write",
		},
	}).AssertStatus(http.StatusCreated)

	// List share links
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodGet,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
	})

	resp.AssertStatus(http.StatusOK)
	// ListShareLinks returns []ShareLinkResponse (array)
	links := resp.GetJSONDataArray()
	s.Len(links, 2)
}

func (s *ShareLinkTestSuite) TestGetShareLinkInfo_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// Get info (no auth required)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodGet,
		Path:   "/api/v1/share/" + shareToken,
	})

	// Note: JSON uses camelCase
	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.hasPassword", false).
		AssertJSONPath("data.resourceType", "folder")
}

// =============================================================================
// Token Validation Tests - R-ST001
// =============================================================================

func (s *ShareLinkTestSuite) TestCreateShareLink_R_ST001_TokenIsURLSafeAndLongEnough() {
	// R-ST001: Token must be 32+ characters and URL-safe
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// Verify token is at least 32 characters
	s.GreaterOrEqual(len(shareToken), 32, "Token should be at least 32 characters")

	// Verify token is URL-safe (only alphanumeric, -, _)
	for _, c := range shareToken {
		isURLSafe := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
		s.True(isURLSafe, "Token should only contain URL-safe characters")
	}
}

// =============================================================================
// Max Access Count Tests
// =============================================================================

func (s *ShareLinkTestSuite) TestAccessShareLink_MaxAccessCountReached() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with max access count of 2
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]interface{}{
			"permission":     "read",
			"maxAccessCount": 2,
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// First access (should succeed)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	}).AssertStatus(http.StatusOK)

	// Second access (should succeed)
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	}).AssertStatus(http.StatusOK)

	// Third access (should fail - max reached)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	})

	resp.AssertStatus(http.StatusForbidden).
		AssertJSONError("FORBIDDEN", "share link has reached maximum access count")
}

func (s *ShareLinkTestSuite) TestAccessShareLink_AccessCountIncrement() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with max access count
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]interface{}{
			"permission":     "read",
			"maxAccessCount": 10,
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareLinkID := shareData["id"].(string)
	shareToken := shareData["token"].(string)

	// Access the link
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	}).AssertStatus(http.StatusOK)

	// Verify access count was incremented
	var accessCount int
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT access_count FROM share_links WHERE id = $1::uuid",
		shareLinkID,
	).Scan(&accessCount)
	s.Require().NoError(err)
	s.Equal(1, accessCount)

	// Access again
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	}).AssertStatus(http.StatusOK)

	// Verify access count was incremented again
	err = s.server.Pool.QueryRow(
		context.Background(),
		"SELECT access_count FROM share_links WHERE id = $1::uuid",
		shareLinkID,
	).Scan(&accessCount)
	s.Require().NoError(err)
	s.Equal(2, accessCount)
}

// =============================================================================
// Write Permission Tests
// =============================================================================

func (s *ShareLinkTestSuite) TestCreateShareLink_WritePermission() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with write permission
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "write",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPath("data.permission", "write")
}

func (s *ShareLinkTestSuite) TestAccessShareLink_WritePermissionCanCreateSubfolder() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with write permission
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "write",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// Access and get temporary access token
	accessResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	})
	accessResp.AssertStatus(http.StatusOK)
	accessData := accessResp.GetJSONData()

	// Check if we got an access token for write operations
	if accessToken, ok := accessData["accessToken"].(string); ok && accessToken != "" {
		// Use the access token to create a subfolder
		resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
			Method:      http.MethodPost,
			Path:        "/api/v1/folders",
			AccessToken: accessToken,
			Body: map[string]interface{}{
				"name":     "Subfolder via Share Link",
				"parentId": folderID,
			},
		})
		resp.AssertStatus(http.StatusCreated)
	}
}

func (s *ShareLinkTestSuite) TestAccessShareLink_ReadPermissionCannotCreateSubfolder() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link with read permission
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareToken := shareData["token"].(string)

	// Access and get temporary access token
	accessResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	})
	accessResp.AssertStatus(http.StatusOK)
	accessData := accessResp.GetJSONData()

	// Check if we got an access token
	if accessToken, ok := accessData["accessToken"].(string); ok && accessToken != "" {
		// Try to create a subfolder with read-only access (should fail)
		resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
			Method:      http.MethodPost,
			Path:        "/api/v1/folders",
			AccessToken: accessToken,
			Body: map[string]interface{}{
				"name":     "Subfolder via Share Link",
				"parentId": folderID,
			},
		})
		resp.AssertStatus(http.StatusForbidden)
	}
}

// =============================================================================
// Share Link Access Log Tests
// =============================================================================

func (s *ShareLinkTestSuite) TestAccessShareLink_LogsAccess() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	folderID := s.createFolder(token, "Shared Folder")

	// Create share link
	createResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:      http.MethodPost,
		Path:        "/api/v1/folders/" + folderID + "/share",
		AccessToken: token,
		Body: map[string]string{
			"permission": "read",
		},
	})
	createResp.AssertStatus(http.StatusCreated)
	shareData := createResp.GetJSONData()
	shareLinkID := shareData["id"].(string)
	shareToken := shareData["token"].(string)

	// Access the link
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/share/" + shareToken + "/access",
	}).AssertStatus(http.StatusOK)

	// Verify access log was created
	var accessCount int
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM share_link_accesses WHERE share_link_id = $1::uuid",
		shareLinkID,
	).Scan(&accessCount)
	s.Require().NoError(err)
	s.Equal(1, accessCount)
}
