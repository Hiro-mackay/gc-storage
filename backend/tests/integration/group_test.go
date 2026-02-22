// Package integration contains integration tests for the Group API
package integration

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/Hiro-mackay/gc-storage/backend/tests/testutil"
)

// GroupTestSuite is the test suite for group-related endpoints
type GroupTestSuite struct {
	suite.Suite
	server *testutil.TestServer
}

// SetupSuite runs once before all tests
func (s *GroupTestSuite) SetupSuite() {
	s.server = testutil.NewTestServer(s.T())
}

// TearDownSuite runs once after all tests
func (s *GroupTestSuite) TearDownSuite() {
	// Cleanup is handled by TestMain in main_test.go
}

// SetupTest runs before each test
func (s *GroupTestSuite) SetupTest() {
	s.server.Cleanup(s.T())
}

// TestMain is the entry point for the test suite
func TestGroupSuite(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
	}
	suite.Run(t, new(GroupTestSuite))
}

// =============================================================================
// Helper Functions
// =============================================================================

func (s *GroupTestSuite) registerAndActivateUser(email, password, name string) {
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

func (s *GroupTestSuite) loginAndGetToken(email, password string) string {
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

func (s *GroupTestSuite) createUser(email, password, name string) string {
	s.registerAndActivateUser(email, password, name)
	return s.loginAndGetToken(email, password)
}

// createGroup creates a group and returns the group data (nested inside data.group)
func (s *GroupTestSuite) createGroup(token, name, description string) (string, map[string]interface{}) {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups",
		SessionID: token,
		Body: map[string]string{
			"name":        name,
			"description": description,
		},
	})
	resp.AssertStatus(http.StatusCreated)
	data := resp.GetJSONData()
	group := data["group"].(map[string]interface{})
	return group["id"].(string), group
}

// =============================================================================
// Group Creation Tests
// =============================================================================

func (s *GroupTestSuite) TestCreateGroup_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups",
		SessionID: token,
		Body: map[string]string{
			"name":        "Test Group",
			"description": "A test group",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPathExists("data.group.id").
		AssertJSONPath("data.group.name", "Test Group").
		AssertJSONPath("data.group.description", "A test group").
		AssertJSONPath("data.myRole", "owner")
}

func (s *GroupTestSuite) TestCreateGroup_Unauthorized() {
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method: http.MethodPost,
		Path:   "/api/v1/groups",
		Body: map[string]string{
			"name":        "Test Group",
			"description": "A test group",
		},
	})

	resp.AssertStatus(http.StatusUnauthorized)
}

func (s *GroupTestSuite) TestCreateGroup_InvalidName_TooShort() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups",
		SessionID: token,
		Body: map[string]string{
			"name":        "",
			"description": "A test group",
		},
	})

	resp.AssertStatus(http.StatusBadRequest)
}

// =============================================================================
// Group Name Validation Tests - R-GN001, R-GN002, R-GN003
// =============================================================================

func (s *GroupTestSuite) TestCreateGroup_R_GN001_NameTooLong() {
	// R-GN001: Group name must be 1-100 characters
	token := s.createUser("owner@example.com", "Password123", "Owner User")

	// Create a name with 101 characters
	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups",
		SessionID: token,
		Body: map[string]string{
			"name":        longName,
			"description": "A test group",
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *GroupTestSuite) TestCreateGroup_R_GN001_NameBoundary100() {
	// R-GN001: Exactly 100 characters should be valid
	token := s.createUser("owner@example.com", "Password123", "Owner User")

	// Create a name with exactly 100 characters
	name100 := ""
	for i := 0; i < 100; i++ {
		name100 += "a"
	}

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups",
		SessionID: token,
		Body: map[string]string{
			"name":        name100,
			"description": "A test group",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPath("data.group.name", name100)
}

func (s *GroupTestSuite) TestCreateGroup_R_GN002_NameTrimmed() {
	// R-GN002: Leading and trailing whitespace should be trimmed
	token := s.createUser("owner@example.com", "Password123", "Owner User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups",
		SessionID: token,
		Body: map[string]string{
			"name":        "  Test Group  ",
			"description": "A test group",
		},
	})

	resp.AssertStatus(http.StatusCreated).
		AssertJSONPath("data.group.name", "Test Group")
}

func (s *GroupTestSuite) TestCreateGroup_R_GN003_WhitespaceOnlyForbidden() {
	// R-GN003: Whitespace-only names are not allowed
	token := s.createUser("owner@example.com", "Password123", "Owner User")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups",
		SessionID: token,
		Body: map[string]string{
			"name":        "   ",
			"description": "A test group",
		},
	})

	resp.AssertStatus(http.StatusBadRequest)
}

func (s *GroupTestSuite) TestCreateGroup_OwnerIsMember() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")

	// Create group
	groupID, _ := s.createGroup(token, "Test Group", "Description")

	// Get members
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/groups/" + groupID + "/members",
		SessionID: token,
	})

	resp.AssertStatus(http.StatusOK)
	members := resp.GetJSONDataArray()
	s.Len(members, 1)

	member := members[0].(map[string]interface{})
	s.Equal("owner", member["role"])
}

// =============================================================================
// Group Update Tests
// =============================================================================

func (s *GroupTestSuite) TestUpdateGroup_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	groupID, _ := s.createGroup(token, "Original Name", "Original Description")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/groups/" + groupID,
		SessionID: token,
		Body: map[string]string{
			"name":        "Updated Name",
			"description": "Updated Description",
		},
	})

	// UpdateGroup returns GroupResponse (flat, not nested)
	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.name", "Updated Name").
		AssertJSONPath("data.description", "Updated Description")
}

func (s *GroupTestSuite) TestUpdateGroup_NonMemberForbidden() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	otherToken := s.createUser("other@example.com", "Password123", "Other User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/groups/" + groupID,
		SessionID: otherToken,
		Body: map[string]string{
			"name": "Hacked Name",
		},
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Group Delete Tests
// =============================================================================

func (s *GroupTestSuite) TestDeleteGroup_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")
	groupID, _ := s.createGroup(token, "Test Group", "Description")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodDelete,
		Path:      "/api/v1/groups/" + groupID,
		SessionID: token,
	})

	resp.AssertStatus(http.StatusNoContent)

	// Verify group is deleted
	getResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/groups/" + groupID,
		SessionID: token,
	})
	getResp.AssertStatus(http.StatusNotFound)
}

func (s *GroupTestSuite) TestDeleteGroup_NonOwnerForbidden() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Invite and accept member
	s.inviteAndAcceptMember(ownerToken, memberToken, groupID, "member@example.com", "contributor")

	// Member tries to delete
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodDelete,
		Path:      "/api/v1/groups/" + groupID,
		SessionID: memberToken,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Member Invitation Tests
// =============================================================================

func (s *GroupTestSuite) inviteAndAcceptMember(ownerToken, memberToken, groupID, email, role string) {
	// Owner invites member
	inviteResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: ownerToken,
		Body: map[string]string{
			"email": email,
			"role":  role,
		},
	})
	inviteResp.AssertStatus(http.StatusCreated)

	// Get invitation token from database
	var token string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT token FROM invitations WHERE email = $1 AND group_id = $2::uuid",
		email, groupID,
	).Scan(&token)
	s.Require().NoError(err)

	// Member accepts invitation
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/invitations/" + token + "/accept",
		SessionID: memberToken,
	}).AssertStatus(http.StatusOK)
}

func (s *GroupTestSuite) TestInviteMember_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: ownerToken,
		Body: map[string]string{
			"email": "member@example.com",
			"role":  "contributor",
		},
	})

	// InviteMember returns InvitationResponse (flat)
	resp.AssertStatus(http.StatusCreated).
		AssertJSONPathExists("data.id")
	// Note: InvitationResponse doesn't expose token in response for security
}

func (s *GroupTestSuite) TestInviteMember_AlreadyMember() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// First invitation and accept
	s.inviteAndAcceptMember(ownerToken, memberToken, groupID, "member@example.com", "contributor")

	// Try to invite again
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: ownerToken,
		Body: map[string]string{
			"email": "member@example.com",
			"role":  "contributor",
		},
	})

	resp.AssertStatus(http.StatusConflict)
}

func (s *GroupTestSuite) TestAcceptInvitation_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Owner invites member
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: ownerToken,
		Body: map[string]string{
			"email": "member@example.com",
			"role":  "contributor",
		},
	}).AssertStatus(http.StatusCreated)

	// Get invitation token
	var token string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT token FROM invitations WHERE email = $1",
		"member@example.com",
	).Scan(&token)
	s.Require().NoError(err)

	// Accept invitation
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/invitations/" + token + "/accept",
		SessionID: memberToken,
	})

	resp.AssertStatus(http.StatusOK)

	// Verify member was added
	membersResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/groups/" + groupID + "/members",
		SessionID: ownerToken,
	})
	membersResp.AssertStatus(http.StatusOK)
	members := membersResp.GetJSONDataArray()
	s.Len(members, 2)
}

func (s *GroupTestSuite) TestDeclineInvitation_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Owner invites member
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: ownerToken,
		Body: map[string]string{
			"email": "member@example.com",
			"role":  "contributor",
		},
	}).AssertStatus(http.StatusCreated)

	// Get invitation token
	var token string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT token FROM invitations WHERE email = $1",
		"member@example.com",
	).Scan(&token)
	s.Require().NoError(err)

	// Decline invitation
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/invitations/" + token + "/decline",
		SessionID: memberToken,
	})

	resp.AssertStatus(http.StatusNoContent)
}

// =============================================================================
// Invitation Role Restriction Tests - R-I005, R-I007
// =============================================================================

func (s *GroupTestSuite) TestInviteMember_R_I005_OwnerRoleForbidden() {
	// R-I005: Cannot invite with owner role
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: ownerToken,
		Body: map[string]string{
			"email": "member@example.com",
			"role":  "owner", // Cannot invite with owner role
		},
	})

	resp.AssertStatus(http.StatusBadRequest).
		AssertJSONError("VALIDATION_ERROR", "")
}

func (s *GroupTestSuite) TestInviteMember_R_I007_ContributorCannotInviteWithHigherRole() {
	// R-I007: Inviter can only grant roles <= their own role
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	contributorToken := s.createUser("contributor@example.com", "Password123", "Contributor User")
	s.createUser("invitee@example.com", "Password123", "Invitee User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Add contributor to the group
	s.inviteAndAcceptMember(ownerToken, contributorToken, groupID, "contributor@example.com", "contributor")

	// Contributor tries to invite with higher role (contributor cannot invite as contributor since invitation can only be viewer)
	// Actually, based on the spec, contributor can only invite with viewer role
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: contributorToken,
		Body: map[string]string{
			"email": "invitee@example.com",
			"role":  "contributor", // Contributor trying to grant contributor role (same level, might be forbidden)
		},
	})

	// Contributor cannot invite with roles higher than viewer
	resp.AssertStatus(http.StatusForbidden)
}

func (s *GroupTestSuite) TestInviteMember_R_I007_ViewerCannotInvite() {
	// R-I007: Viewer should not be able to invite at all
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	viewerToken := s.createUser("viewer@example.com", "Password123", "Viewer User")
	s.createUser("invitee@example.com", "Password123", "Invitee User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Add viewer to the group
	s.inviteAndAcceptMember(ownerToken, viewerToken, groupID, "viewer@example.com", "viewer")

	// Viewer tries to invite someone
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: viewerToken,
		Body: map[string]string{
			"email": "invitee@example.com",
			"role":  "viewer",
		},
	})

	resp.AssertStatus(http.StatusForbidden)
}

func (s *GroupTestSuite) TestInviteMember_DefaultRoleIsViewer() {
	// R-I006: Default invitation role is viewer
	// TODO: Default role feature is not implemented - role field is required
	s.T().Skip("Default role feature is not implemented - role field is required")
}

// =============================================================================
// Member Remove Tests
// =============================================================================

func (s *GroupTestSuite) TestRemoveMember_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Invite and accept member
	s.inviteAndAcceptMember(ownerToken, memberToken, groupID, "member@example.com", "contributor")

	// Get member's user ID
	var userID string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT id FROM users WHERE email = $1",
		"member@example.com",
	).Scan(&userID)
	s.Require().NoError(err)

	// Remove member
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodDelete,
		Path:      "/api/v1/groups/" + groupID + "/members/" + userID,
		SessionID: ownerToken,
	})

	resp.AssertStatus(http.StatusNoContent)
}

func (s *GroupTestSuite) TestRemoveMember_CannotRemoveOwner() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Get owner's user ID
	var userID string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT id FROM users WHERE email = $1",
		"owner@example.com",
	).Scan(&userID)
	s.Require().NoError(err)

	// Try to remove owner
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodDelete,
		Path:      "/api/v1/groups/" + groupID + "/members/" + userID,
		SessionID: ownerToken,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Leave Group Tests
// =============================================================================

func (s *GroupTestSuite) TestLeaveGroup_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Invite and accept member
	s.inviteAndAcceptMember(ownerToken, memberToken, groupID, "member@example.com", "contributor")

	// Member leaves
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/leave",
		SessionID: memberToken,
	})

	resp.AssertStatus(http.StatusNoContent)
}

func (s *GroupTestSuite) TestLeaveGroup_OwnerCannotLeave() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Owner tries to leave
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/leave",
		SessionID: ownerToken,
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Role Change Tests
// =============================================================================

func (s *GroupTestSuite) TestChangeRole_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Invite and accept member as viewer
	s.inviteAndAcceptMember(ownerToken, memberToken, groupID, "member@example.com", "viewer")

	// Get member's user ID
	var userID string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT id FROM users WHERE email = $1",
		"member@example.com",
	).Scan(&userID)
	s.Require().NoError(err)

	// Change role to contributor
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/groups/" + groupID + "/members/" + userID + "/role",
		SessionID: ownerToken,
		Body: map[string]string{
			"role": "contributor",
		},
	})

	// ChangeRole returns MembershipResponse (flat)
	resp.AssertStatus(http.StatusOK).
		AssertJSONPath("data.role", "contributor")
}

func (s *GroupTestSuite) TestChangeRole_NonOwnerForbidden() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	member1Token := s.createUser("member1@example.com", "Password123", "Member 1")
	s.createUser("member2@example.com", "Password123", "Member 2")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Invite both members
	s.inviteAndAcceptMember(ownerToken, member1Token, groupID, "member1@example.com", "contributor")

	// Get member1's user ID
	var userID string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT id FROM users WHERE email = $1",
		"member1@example.com",
	).Scan(&userID)
	s.Require().NoError(err)

	// Member1 tries to change own role (should fail - not owner)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPatch,
		Path:      "/api/v1/groups/" + groupID + "/members/" + userID + "/role",
		SessionID: member1Token,
		Body: map[string]string{
			"role": "viewer",
		},
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// Transfer Ownership Tests
// =============================================================================

func (s *GroupTestSuite) TestTransferOwnership_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Invite and accept member
	s.inviteAndAcceptMember(ownerToken, memberToken, groupID, "member@example.com", "contributor")

	// Get new owner's user ID
	var newOwnerID string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT id FROM users WHERE email = $1",
		"member@example.com",
	).Scan(&newOwnerID)
	s.Require().NoError(err)

	// Transfer ownership
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/transfer",
		SessionID: ownerToken,
		Body: map[string]string{
			"newOwnerId": newOwnerID,
		},
	})

	resp.AssertStatus(http.StatusOK)

	// Verify new owner (TransferOwnership returns GroupResponse, ownerId in camelCase)
	groupResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/groups/" + groupID,
		SessionID: memberToken,
	})
	groupResp.AssertStatus(http.StatusOK)
	// GetGroup returns GroupWithMembershipAndCountResponse (nested)
	groupData := groupResp.GetJSONData()
	group := groupData["group"].(map[string]interface{})
	s.Equal(newOwnerID, group["ownerId"])

	// Old owner should now be able to leave (as contributor)
	leaveResp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/leave",
		SessionID: ownerToken,
	})
	leaveResp.AssertStatus(http.StatusNoContent)
}

func (s *GroupTestSuite) TestTransferOwnership_NonOwnerForbidden() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	s.createUser("other@example.com", "Password123", "Other User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Invite member
	s.inviteAndAcceptMember(ownerToken, memberToken, groupID, "member@example.com", "contributor")

	// Get other's user ID
	var otherID string
	err := s.server.Pool.QueryRow(
		context.Background(),
		"SELECT id FROM users WHERE email = $1",
		"other@example.com",
	).Scan(&otherID)
	s.Require().NoError(err)

	// Member tries to transfer ownership (should fail)
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/transfer",
		SessionID: memberToken,
		Body: map[string]string{
			"newOwnerId": otherID,
		},
	})

	resp.AssertStatus(http.StatusForbidden)
}

// =============================================================================
// List Tests
// =============================================================================

func (s *GroupTestSuite) TestListMyGroups_Success() {
	token := s.createUser("owner@example.com", "Password123", "Owner User")

	// Create multiple groups
	s.createGroup(token, "Group 1", "Description 1")
	s.createGroup(token, "Group 2", "Description 2")

	// List groups
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/groups",
		SessionID: token,
	})

	resp.AssertStatus(http.StatusOK)
	// ListMyGroups returns []GroupWithMembershipResponse (array)
	groups := resp.GetJSONDataArray()
	s.Len(groups, 2)
}

func (s *GroupTestSuite) TestListPendingInvitations_Success() {
	ownerToken := s.createUser("owner@example.com", "Password123", "Owner User")
	memberToken := s.createUser("member@example.com", "Password123", "Member User")
	groupID, _ := s.createGroup(ownerToken, "Test Group", "Description")

	// Send invitation
	testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodPost,
		Path:      "/api/v1/groups/" + groupID + "/invitations",
		SessionID: ownerToken,
		Body: map[string]string{
			"email": "member@example.com",
			"role":  "contributor",
		},
	}).AssertStatus(http.StatusCreated)

	// Member checks pending invitations
	resp := testutil.DoRequest(s.T(), s.server.Echo, testutil.HTTPRequest{
		Method:    http.MethodGet,
		Path:      "/api/v1/invitations/pending",
		SessionID: memberToken,
	})

	resp.AssertStatus(http.StatusOK)
	// ListPendingInvitations returns []PendingInvitationResponse (array)
	invitations := resp.GetJSONDataArray()
	s.Len(invitations, 1)
}
