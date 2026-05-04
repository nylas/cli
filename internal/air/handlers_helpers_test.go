package air

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_withTimeout(t *testing.T) {
	t.Run("creates context with default timeout", func(t *testing.T) {
		server := &Server{}
		req := httptest.NewRequest("GET", "/", nil)

		ctx, cancel := server.withTimeout(req)
		defer cancel()

		assert.NotNil(t, ctx)
		deadline, ok := ctx.Deadline()
		require.True(t, ok, "context should have a deadline")

		// Verify deadline is approximately 30 seconds from now
		expectedDeadline := time.Now().Add(defaultTimeout)
		timeDiff := expectedDeadline.Sub(deadline)
		assert.Less(t, timeDiff, 1*time.Second, "deadline should be ~30 seconds from now")
	})

	t.Run("context inherits from request context", func(t *testing.T) {
		server := &Server{}
		type contextKey string
		key := contextKey("testKey")
		parentCtx := context.WithValue(context.Background(), key, "value")
		req := httptest.NewRequest("GET", "/", nil).WithContext(parentCtx)

		ctx, cancel := server.withTimeout(req)
		defer cancel()

		// Verify parent context value is accessible
		assert.Equal(t, "value", ctx.Value(key))
	})

	t.Run("cancel function can be called", func(t *testing.T) {
		server := &Server{}
		req := httptest.NewRequest("GET", "/", nil)

		ctx, cancel := server.withTimeout(req)

		// Cancel should not panic
		cancel()

		// Context should be canceled
		select {
		case <-ctx.Done():
			assert.Equal(t, context.Canceled, ctx.Err())
		case <-time.After(100 * time.Millisecond):
			t.Error("context should be canceled immediately after cancel()")
		}
	})
}

func TestServer_requireConfig(t *testing.T) {
	t.Run("returns true when client is configured", func(t *testing.T) {
		server := &Server{
			nylasClient: nylas.NewMockClient(),
		}
		w := httptest.NewRecorder()

		result := server.requireConfig(w)

		assert.True(t, result)
		assert.Equal(t, 200, w.Code) // No error response written
	})

	t.Run("returns false and writes error when client is nil", func(t *testing.T) {
		server := &Server{
			nylasClient: nil,
		}
		w := httptest.NewRecorder()

		result := server.requireConfig(w)

		assert.False(t, result)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "Not configured")
		assert.Contains(t, w.Body.String(), "nylas auth login")
	})
}

func TestParseJSONBody(t *testing.T) {
	type testRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	t.Run("parses valid JSON body", func(t *testing.T) {
		body := strings.NewReader(`{"name":"John","email":"john@example.com"}`)
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		var dest testRequest
		result := parseJSONBody(w, req, &dest)

		assert.True(t, result)
		assert.Equal(t, "John", dest.Name)
		assert.Equal(t, "john@example.com", dest.Email)
	})

	t.Run("returns false on invalid JSON", func(t *testing.T) {
		body := strings.NewReader(`{invalid json}`)
		req := httptest.NewRequest("POST", "/", body)
		w := httptest.NewRecorder()

		var dest testRequest
		result := parseJSONBody(w, req, &dest)

		assert.False(t, result)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid request body")
	})

	t.Run("returns false on empty body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		w := httptest.NewRecorder()

		var dest testRequest
		result := parseJSONBody(w, req, &dest)

		assert.False(t, result)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("handles type mismatch returns false", func(t *testing.T) {
		body := strings.NewReader(`{"name":123,"email":true}`)
		req := httptest.NewRequest("POST", "/", body)
		w := httptest.NewRecorder()

		var dest testRequest
		result := parseJSONBody(w, req, &dest)

		// JSON decoder fails on type mismatch (number/bool to string)
		assert.False(t, result)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestServer_handleDemoMode(t *testing.T) {
	t.Run("returns false when demo mode is disabled", func(t *testing.T) {
		server := &Server{
			demoMode: false,
		}
		w := httptest.NewRecorder()
		data := map[string]string{"test": "data"}

		result := server.handleDemoMode(w, data)

		assert.False(t, result)
		assert.Equal(t, 200, w.Code) // No response written
		assert.Empty(t, w.Body.String())
	})

	t.Run("returns true and writes response when demo mode is enabled", func(t *testing.T) {
		server := &Server{
			demoMode: true,
		}
		w := httptest.NewRecorder()
		data := map[string]string{"test": "data"}

		result := server.handleDemoMode(w, data)

		assert.True(t, result)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "test")
		assert.Contains(t, w.Body.String(), "data")
	})

	t.Run("handles complex data structures", func(t *testing.T) {
		server := &Server{
			demoMode: true,
		}
		w := httptest.NewRecorder()
		data := map[string]any{
			"users":  []string{"alice", "bob"},
			"count":  2,
			"active": true,
		}

		result := server.handleDemoMode(w, data)

		assert.True(t, result)
		body := w.Body.String()
		assert.Contains(t, body, "alice")
		assert.Contains(t, body, "bob")
		assert.Contains(t, body, "2")
		assert.Contains(t, body, "true")
	})
}

func TestRequireMethod(t *testing.T) {
	t.Run("returns true when method matches", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		result := requireMethod(w, req, "GET")

		assert.True(t, result)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("returns false when method does not match", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		w := httptest.NewRecorder()

		result := requireMethod(w, req, "GET")

		assert.False(t, result)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		assert.Contains(t, w.Body.String(), "Method not allowed")
	})

	t.Run("handles different HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				req := httptest.NewRequest(method, "/", nil)
				w := httptest.NewRecorder()

				result := requireMethod(w, req, method)

				assert.True(t, result)
			})
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Run("writes JSON error response", func(t *testing.T) {
		w := httptest.NewRecorder()

		writeError(w, http.StatusBadRequest, "Invalid input")

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "error")
		assert.Contains(t, w.Body.String(), "Invalid input")
	})

	t.Run("handles different status codes", func(t *testing.T) {
		tests := []struct {
			status  int
			message string
		}{
			{http.StatusBadRequest, "Bad request"},
			{http.StatusNotFound, "Not found"},
			{http.StatusInternalServerError, "Internal error"},
			{http.StatusUnauthorized, "Unauthorized"},
		}

		for _, tt := range tests {
			t.Run(tt.message, func(t *testing.T) {
				w := httptest.NewRecorder()

				writeError(w, tt.status, tt.message)

				assert.Equal(t, tt.status, w.Code)
				assert.Contains(t, w.Body.String(), tt.message)
			})
		}
	})

	t.Run("escapes special characters in error message", func(t *testing.T) {
		w := httptest.NewRecorder()

		writeError(w, http.StatusBadRequest, `Error with "quotes" and <tags>`)

		body := w.Body.String()
		assert.Contains(t, body, "Error with")
		assert.Contains(t, body, "quotes")
	})
}

// TestParseJSONBody_DoesNotLeakDecoderError pins the privacy contract on
// the JSON-body intake helper. Today it returns
//
//	"Invalid request body: " + err.Error()
//
// to the client. encoding/json's decode errors fingerprint the request
// — they include the Go struct field path, the wire type, and (for
// UnmarshalTypeError) value fragments from the body. None of that
// belongs in a browser toast or a customer-visible error envelope.
//
// The same writeUpstreamError discipline applied at six handler sites in
// the air-i003 review-pass should apply here. Until parseJSONBody is
// migrated, this test fails on the substring assertions and serves as
// the bug receipt.
//
// EXPECTED FAILURE today: response body contains "cannot unmarshal" and
// the field path. After the fix the body is a generic message and the
// raw decoder error lives in slog only.
func TestParseJSONBody_DoesNotLeakDecoderError(t *testing.T) {
	t.Parallel()

	body := strings.NewReader(`{"unread":"not-a-bool"}`)
	req := httptest.NewRequest(http.MethodPost, "/x", body)
	w := httptest.NewRecorder()

	var dest struct {
		Unread *bool `json:"unread"`
	}
	ok := parseJSONBody(w, req, &dest)

	assert.False(t, ok, "parseJSONBody should reject malformed body")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	got := w.Body.String()
	// Substrings drawn from encoding/json's UnmarshalTypeError format.
	// Any of them in the response body proves the decoder error leaked.
	forbidden := []string{
		"cannot unmarshal",
		"Go struct field",
		".unread",
		"of type bool",
	}
	for _, s := range forbidden {
		assert.NotContains(t, got, s,
			"response body must not echo decoder detail %q; got %s", s, got)
	}
}

// TestRedactEmail pins the local-part-only redaction. Six-line privacy
// helper, exercised today only indirectly through slog calls — a
// regression that returned the full address would only show up in logs
// (which tests don't observe) until a customer report surfaced it.
func TestRedactEmail(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"alice@example.com", "***@example.com"},
		{"a@b.example", "***@b.example"},
		// LastIndex semantics: only the final @ is treated as the boundary.
		{"user@inner@outer.example", "***@outer.example"},
		// Degenerate inputs: no local part, no domain, no @ at all.
		{"@example.com", "***"},
		{"user@", "***"},
		{"no-at-sign", "***"},
		{"@", "***"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, redactEmail(tc.in))
		})
	}
}
