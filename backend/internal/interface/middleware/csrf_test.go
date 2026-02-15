package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func setupCSRFTest(method, path string, sessionID, csrfCookie, csrfHeader string) (*echo.Echo, *http.Request, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()

	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	}
	if csrfCookie != "" {
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: csrfCookie})
	}
	if csrfHeader != "" {
		req.Header.Set(CSRFHeaderName, csrfHeader)
	}

	return e, req, rec
}

func TestCSRF_GET_SkipsValidation(t *testing.T) {
	e, req, rec := setupCSRFTest(http.MethodGet, "/test", "session-123", "", "")

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Errorf("GET request should skip CSRF validation, got error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCSRF_HEAD_SkipsValidation(t *testing.T) {
	e, req, rec := setupCSRFTest(http.MethodHead, "/test", "session-123", "", "")

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Errorf("HEAD request should skip CSRF validation, got error: %v", err)
	}
}

func TestCSRF_OPTIONS_SkipsValidation(t *testing.T) {
	e, req, rec := setupCSRFTest(http.MethodOptions, "/test", "session-123", "", "")

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Errorf("OPTIONS request should skip CSRF validation, got error: %v", err)
	}
}

func TestCSRF_POST_NoSessionCookie_SkipsValidation(t *testing.T) {
	e, req, rec := setupCSRFTest(http.MethodPost, "/test", "", "", "")

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Errorf("POST without session should skip CSRF validation, got error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCSRF_POST_WithSession_NoCSRFCookie_ReturnsForbidden(t *testing.T) {
	e, req, rec := setupCSRFTest(http.MethodPost, "/test", "session-123", "", "")

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	err := handler(c)
	if err == nil {
		t.Fatal("expected error for missing CSRF cookie")
	}

	he, ok := err.(*echo.HTTPError)
	if ok {
		if he.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", he.Code)
		}
	}
}

func TestCSRF_POST_WithSession_NoCSRFHeader_ReturnsForbidden(t *testing.T) {
	e, req, rec := setupCSRFTest(http.MethodPost, "/test", "session-123", "valid-token", "")

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	err := handler(c)
	if err == nil {
		t.Fatal("expected error for missing CSRF header")
	}
}

func TestCSRF_POST_WithSession_TokenMismatch_ReturnsForbidden(t *testing.T) {
	e, req, rec := setupCSRFTest(http.MethodPost, "/test", "session-123", "token-a", "token-b")

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	err := handler(c)
	if err == nil {
		t.Fatal("expected error for CSRF token mismatch")
	}
}

func TestCSRF_POST_WithSession_ValidToken_Passes(t *testing.T) {
	token := "matching-csrf-token"
	e, req, rec := setupCSRFTest(http.MethodPost, "/test", "session-123", token, token)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Errorf("valid CSRF token should pass, got error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCSRF_PUT_WithSession_ValidToken_Passes(t *testing.T) {
	token := "matching-csrf-token"
	e, req, rec := setupCSRFTest(http.MethodPut, "/test", "session-123", token, token)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Errorf("PUT with valid CSRF token should pass, got error: %v", err)
	}
}

func TestCSRF_DELETE_WithSession_ValidToken_Passes(t *testing.T) {
	token := "matching-csrf-token"
	e, req, rec := setupCSRFTest(http.MethodDelete, "/test", "session-123", token, token)

	handler := CSRF()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Errorf("DELETE with valid CSRF token should pass, got error: %v", err)
	}
}

func TestGenerateCSRFToken_ReturnsHexString(t *testing.T) {
	token, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("GenerateCSRFToken returned error: %v", err)
	}

	// 32 bytes = 64 hex characters
	if len(token) != 64 {
		t.Errorf("expected 64 character hex string, got %d characters", len(token))
	}

	// Verify it's valid hex
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token contains non-hex character: %c", c)
			break
		}
	}
}

func TestGenerateCSRFToken_IsUnique(t *testing.T) {
	token1, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("first GenerateCSRFToken returned error: %v", err)
	}

	token2, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("second GenerateCSRFToken returned error: %v", err)
	}

	if token1 == token2 {
		t.Error("two generated tokens should be different")
	}
}
