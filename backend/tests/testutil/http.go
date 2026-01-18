package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HTTPRequest represents a test HTTP request
type HTTPRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	Cookies     []*http.Cookie
	AccessToken string
}

// HTTPResponse wraps the HTTP response for testing
type HTTPResponse struct {
	*httptest.ResponseRecorder
	t *testing.T
}

// DoRequest performs an HTTP request against the test server
func DoRequest(t *testing.T, e *echo.Echo, req HTTPRequest) *HTTPResponse {
	t.Helper()

	var body io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		require.NoError(t, err)
		body = bytes.NewReader(jsonBody)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, body)
	httpReq.Header.Set("Content-Type", "application/json")

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set access token
	if req.AccessToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.AccessToken)
	}

	// Set cookies
	for _, cookie := range req.Cookies {
		httpReq.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httpReq)

	return &HTTPResponse{ResponseRecorder: rec, t: t}
}

// AssertStatus asserts the response status code
func (r *HTTPResponse) AssertStatus(expected int) *HTTPResponse {
	assert.Equal(r.t, expected, r.Code, "unexpected status code, body: %s", r.Body.String())
	return r
}

// AssertJSON asserts the response body matches expected JSON
func (r *HTTPResponse) AssertJSON(expected map[string]interface{}) *HTTPResponse {
	var actual map[string]interface{}
	err := json.Unmarshal(r.Body.Bytes(), &actual)
	require.NoError(r.t, err)
	assert.Equal(r.t, expected, actual)
	return r
}

// AssertJSONPath asserts a specific path in the JSON response
func (r *HTTPResponse) AssertJSONPath(path string, expected interface{}) *HTTPResponse {
	var actual map[string]interface{}
	err := json.Unmarshal(r.Body.Bytes(), &actual)
	require.NoError(r.t, err)

	value := getJSONPath(actual, path)
	assert.Equal(r.t, expected, value, "JSON path %s mismatch", path)
	return r
}

// AssertJSONPathExists asserts a path exists in the JSON response
func (r *HTTPResponse) AssertJSONPathExists(path string) *HTTPResponse {
	var actual map[string]interface{}
	err := json.Unmarshal(r.Body.Bytes(), &actual)
	require.NoError(r.t, err)

	value := getJSONPath(actual, path)
	assert.NotNil(r.t, value, "JSON path %s does not exist", path)
	return r
}

// AssertJSONError asserts the response contains an error with expected code
func (r *HTTPResponse) AssertJSONError(code string, message string) *HTTPResponse {
	var actual map[string]interface{}
	err := json.Unmarshal(r.Body.Bytes(), &actual)
	require.NoError(r.t, err)

	errorObj, ok := actual["error"].(map[string]interface{})
	require.True(r.t, ok, "response does not contain error object")

	assert.Equal(r.t, code, errorObj["code"], "error code mismatch")
	if message != "" {
		assert.Equal(r.t, message, errorObj["message"], "error message mismatch")
	}
	return r
}

// GetJSON parses the response body as JSON
func (r *HTTPResponse) GetJSON() map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal(r.Body.Bytes(), &result)
	require.NoError(r.t, err)
	return result
}

// GetJSONData returns the "data" field from the response
func (r *HTTPResponse) GetJSONData() map[string]interface{} {
	json := r.GetJSON()
	data, ok := json["data"].(map[string]interface{})
	if !ok {
		return nil
	}
	return data
}

// GetCookie returns a cookie by name
func (r *HTTPResponse) GetCookie(name string) *http.Cookie {
	cookies := r.Result().Cookies()
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// getJSONPath gets a value from nested JSON using dot notation (e.g., "data.user.id")
func getJSONPath(data map[string]interface{}, path string) interface{} {
	keys := splitPath(path)
	current := interface{}(data)

	for _, key := range keys {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[key]
		default:
			return nil
		}
	}

	return current
}

// splitPath splits a dot-notation path into keys
func splitPath(path string) []string {
	var keys []string
	var current string

	for _, c := range path {
		if c == '.' {
			if current != "" {
				keys = append(keys, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}

	if current != "" {
		keys = append(keys, current)
	}

	return keys
}
