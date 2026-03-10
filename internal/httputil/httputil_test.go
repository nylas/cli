package httputil_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// WriteJSON Tests
// =============================================================================

func TestWriteJSON_SetsContentTypeHeader(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"key": "value"})

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestWriteJSON_StatusCode(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			httputil.WriteJSON(w, tt.status, map[string]string{})
			assert.Equal(t, tt.status, w.Code)
		})
	}
}

func TestWriteJSON_EncodesBody(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		contains []string
	}{
		{
			name:     "simple map",
			data:     map[string]string{"hello": "world"},
			contains: []string{`"hello"`, `"world"`},
		},
		{
			name:     "struct",
			data:     struct{ Name string }{Name: "Alice"},
			contains: []string{`"Name"`, `"Alice"`},
		},
		{
			name:     "array",
			data:     []int{1, 2, 3},
			contains: []string{"1", "2", "3"},
		},
		{
			name:     "nil",
			data:     nil,
			contains: []string{"null"},
		},
		{
			name:     "empty object",
			data:     map[string]string{},
			contains: []string{"{}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			httputil.WriteJSON(w, http.StatusOK, tt.data)

			body := w.Body.String()
			for _, substr := range tt.contains {
				assert.Contains(t, body, substr)
			}
		})
	}
}

func TestWriteJSON_ValidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]any{
		"id":    42,
		"name":  "test",
		"tags":  []string{"a", "b"},
		"valid": true,
	}

	httputil.WriteJSON(w, http.StatusOK, data)

	var decoded map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &decoded)
	require.NoError(t, err)
	assert.Equal(t, float64(42), decoded["id"])
	assert.Equal(t, "test", decoded["name"])
	assert.Equal(t, true, decoded["valid"])
}

// =============================================================================
// LimitedBody Tests
// =============================================================================

func TestLimitedBody_ReturnsReadCloser(t *testing.T) {
	body := strings.NewReader("hello world")
	req := &http.Request{Body: io.NopCloser(body)}
	w := httptest.NewRecorder()

	rc := httputil.LimitedBody(w, req, 100)
	assert.NotNil(t, rc)

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestLimitedBody_RejectsOversizedBody(t *testing.T) {
	// Create body larger than limit
	large := bytes.Repeat([]byte("x"), 200)
	req := &http.Request{Body: io.NopCloser(bytes.NewReader(large))}
	w := httptest.NewRecorder()

	rc := httputil.LimitedBody(w, req, 100)

	data, err := io.ReadAll(rc)
	// MaxBytesReader returns an error when the limit is exceeded.
	// The behavior: data up to limit may be returned, plus an error.
	assert.Error(t, err, "expected error for oversized body; got %d bytes", len(data))
}

func TestLimitedBody_AllowsBodyAtExactLimit(t *testing.T) {
	content := strings.Repeat("x", 100)
	req := &http.Request{Body: io.NopCloser(strings.NewReader(content))}
	w := httptest.NewRecorder()

	rc := httputil.LimitedBody(w, req, 100)

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestLimitedBody_EmptyBody(t *testing.T) {
	req := &http.Request{Body: io.NopCloser(strings.NewReader(""))}
	w := httptest.NewRecorder()

	rc := httputil.LimitedBody(w, req, 100)

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Empty(t, data)
}

// =============================================================================
// DecodeJSON Tests
// =============================================================================

func TestDecodeJSON_ValidPayload(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		target any
		check  func(t *testing.T, target any)
	}{
		{
			name: "simple struct",
			body: `{"name":"Alice","age":30}`,
			target: &struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{},
			check: func(t *testing.T, target any) {
				v, ok := target.(*struct {
					Name string `json:"name"`
					Age  int    `json:"age"`
				})
				require.True(t, ok)
				assert.Equal(t, "Alice", v.Name)
				assert.Equal(t, 30, v.Age)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			err := httputil.DecodeJSON(w, req, tt.target)
			require.NoError(t, err)
			tt.check(t, tt.target)
		})
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{invalid json"))
	w := httptest.NewRecorder()

	var target map[string]string
	err := httputil.DecodeJSON(w, req, &target)
	assert.Error(t, err)
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	var target map[string]string
	err := httputil.DecodeJSON(w, req, &target)
	assert.Error(t, err) // EOF
}

func TestDecodeJSON_OversizedBody(t *testing.T) {
	// Body larger than MaxRequestBodySize (1MB)
	large := bytes.Repeat([]byte("x"), httputil.MaxRequestBodySize+1)
	// Wrap in JSON string to make it valid-ish JSON until truncated
	payload := `{"data":"` + string(large) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	w := httptest.NewRecorder()

	var target map[string]string
	err := httputil.DecodeJSON(w, req, &target)
	assert.Error(t, err)
}

func TestDecodeJSON_MaxRequestBodySizeConstant(t *testing.T) {
	// MaxRequestBodySize must equal 1MB (1 << 20)
	assert.Equal(t, int64(1<<20), int64(httputil.MaxRequestBodySize))
}

func TestDecodeJSON_MapTarget(t *testing.T) {
	body := `{"foo":"bar","num":42}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()

	var target map[string]any
	err := httputil.DecodeJSON(w, req, &target)
	require.NoError(t, err)
	assert.Equal(t, "bar", target["foo"])
	assert.Equal(t, float64(42), target["num"])
}
