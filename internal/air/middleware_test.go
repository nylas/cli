package air

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestCompressionMiddleware_WithGzipSupport tests gzip compression when client accepts it.
func TestCompressionMiddleware_WithGzipSupport(t *testing.T) {
	t.Parallel()

	handler := CompressionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello, World! This is a test response that should be compressed."))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check Content-Encoding header
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got %s", w.Header().Get("Content-Encoding"))
	}

	// Decompress and verify content
	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer func() {
		_ = reader.Close() // Error is non-actionable in test cleanup
	}()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read decompressed data: %v", err)
	}

	expected := "Hello, World! This is a test response that should be compressed."
	if string(decompressed) != expected {
		t.Errorf("expected decompressed content %q, got %q", expected, string(decompressed))
	}
}

// TestCompressionMiddleware_WithoutGzipSupport tests no compression when client doesn't accept gzip.
func TestCompressionMiddleware_WithoutGzipSupport(t *testing.T) {
	t.Parallel()

	handler := CompressionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello, World!"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Accept-Encoding header
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should not have Content-Encoding
	if w.Header().Get("Content-Encoding") != "" {
		t.Errorf("expected no Content-Encoding header, got %s", w.Header().Get("Content-Encoding"))
	}

	// Content should be uncompressed
	if w.Body.String() != "Hello, World!" {
		t.Errorf("expected uncompressed content, got %q", w.Body.String())
	}
}

// TestCompressionMiddleware_SkipsPrecompressedFiles tests that already compressed files are not re-compressed.
func TestCompressionMiddleware_SkipsPrecompressedFiles(t *testing.T) {
	t.Parallel()

	tests := []string{
		"/file.gz",
		"/image.jpg",
		"/image.png",
		"/font.woff",
		"/font.woff2",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			handler := CompressionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("content"))
			}))

			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should not have Content-Encoding
			if w.Header().Get("Content-Encoding") != "" {
				t.Errorf("expected no compression for %s, got Content-Encoding: %s", path, w.Header().Get("Content-Encoding"))
			}
		})
	}
}

// TestCacheMiddleware_StaticAssets tests cache headers for static assets.
// For development, CSS/JS use no-cache; only images/fonts get cached.
func TestCacheMiddleware_StaticAssets(t *testing.T) {
	t.Parallel()

	handler := CacheMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// CSS files use no-cache for instant dev updates
	t.Run("css_no_cache", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/css/main.css", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		cacheControl := w.Header().Get("Cache-Control")
		if cacheControl != "no-cache, must-revalidate" {
			t.Errorf("expected no-cache for CSS (dev mode), got %s", cacheControl)
		}
	})

	// Images get cached
	t.Run("images_cached", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/img/logo.png", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		cacheControl := w.Header().Get("Cache-Control")
		if cacheControl != "public, max-age=86400" {
			t.Errorf("expected 1-day cache for images, got %s", cacheControl)
		}
	})
}

// TestCacheMiddleware_APIResponses tests no-cache headers for API responses.
func TestCacheMiddleware_APIResponses(t *testing.T) {
	t.Parallel()

	handler := CacheMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/emails", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-cache, no-store, must-revalidate" {
		t.Errorf("expected no-cache for API responses, got %s", cacheControl)
	}

	pragma := w.Header().Get("Pragma")
	if pragma != "no-cache" {
		t.Errorf("expected Pragma: no-cache, got %s", pragma)
	}

	expires := w.Header().Get("Expires")
	if expires != "0" {
		t.Errorf("expected Expires: 0, got %s", expires)
	}
}

// TestCacheMiddleware_HTMLPages tests cache headers for HTML pages.
// For development, HTML uses no-cache for instant updates.
func TestCacheMiddleware_HTMLPages(t *testing.T) {
	t.Parallel()

	handler := CacheMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-cache, must-revalidate" {
		t.Errorf("expected no-cache for HTML (dev mode), got %s", cacheControl)
	}
}

// TestPerformanceMonitoringMiddleware_AddsServerTimingHeader tests that Server-Timing header is added.
func TestPerformanceMonitoringMiddleware_AddsServerTimingHeader(t *testing.T) {
	t.Parallel()

	handler := PerformanceMonitoringMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	serverTiming := w.Header().Get("Server-Timing")
	if serverTiming == "" {
		t.Error("expected Server-Timing header to be set")
	}

	if !strings.HasPrefix(serverTiming, "total;dur=") {
		t.Errorf("expected Server-Timing format 'total;dur=X', got %s", serverTiming)
	}
}

// TestPerformanceMonitoringMiddleware_ImplicitWrite tests that headers are written even without explicit WriteHeader.
func TestPerformanceMonitoringMiddleware_ImplicitWrite(t *testing.T) {
	t.Parallel()

	handler := PerformanceMonitoringMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write without calling WriteHeader (implicit 200)
		_, _ = w.Write([]byte("test"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	serverTiming := w.Header().Get("Server-Timing")
	if serverTiming == "" {
		t.Error("expected Server-Timing header even with implicit write")
	}
}

// TestSecurityHeadersMiddleware_SetsAllHeaders tests that all security headers are set.
func TestSecurityHeadersMiddleware_SetsAllHeaders(t *testing.T) {
	t.Parallel()

	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"X-Frame-Options", "SAMEORIGIN"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		value := w.Header().Get(tt.header)
		if value != tt.expected {
			t.Errorf("expected %s: %s, got %s", tt.header, tt.expected, value)
		}
	}

	// Check CSP is set
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header to be set")
	}
}

// TestMethodOverrideMiddleware_OverridesPOSTMethod tests method override functionality.
func TestMethodOverrideMiddleware_OverridesPOSTMethod(t *testing.T) {
	t.Parallel()

	handler := MethodOverrideMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler sees the overridden method
		if r.Method != http.MethodDelete {
			t.Errorf("expected method DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-HTTP-Method-Override", "DELETE")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

// TestMethodOverrideMiddleware_OnlyOverridesPOST tests that only POST can be overridden.
func TestMethodOverrideMiddleware_OnlyOverridesPOST(t *testing.T) {
	t.Parallel()

	handler := MethodOverrideMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected method GET (not overridden), got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-HTTP-Method-Override", "DELETE")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

// TestCORSMiddleware_SetsHeaders tests CORS headers.
func TestCORSMiddleware_ReflectsSameOrigin(t *testing.T) {
	t.Parallel()

	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "localhost:7365"
	req.Header.Set("Origin", "http://localhost:7365")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"Access-Control-Allow-Origin", "http://localhost:7365"},
		{"Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS"},
		{"Access-Control-Max-Age", "86400"},
		{"Vary", "Origin"},
	}

	for _, tt := range tests {
		value := w.Header().Get(tt.header)
		if value != tt.expected {
			t.Errorf("expected %s: %s, got %s", tt.header, tt.expected, value)
		}
	}
}

// TestCORSMiddleware_HandlesPreflightRequests tests OPTIONS preflight.
func TestCORSMiddleware_RejectsCrossOriginPreflight(t *testing.T) {
	t.Parallel()

	handlerCalled := false
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Host = "localhost:7365"
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for cross-origin OPTIONS, got %d", w.Code)
	}

	if handlerCalled {
		t.Error("expected handler not to be called for OPTIONS preflight")
	}
}

func TestHostValidationMiddleware_RejectsNonLoopbackHost(t *testing.T) {
	t.Parallel()

	handlerCalled := false
	handler := HostValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "evil.example:7365"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
	if handlerCalled {
		t.Fatal("expected non-loopback host to be rejected before handler execution")
	}
}

func TestOriginProtectionMiddleware_AllowsSameOriginMutation(t *testing.T) {
	t.Parallel()

	handlerCalled := false
	handler := OriginProtectionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Host = "localhost:7365"
	req.Header.Set("Origin", "http://localhost:7365")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !handlerCalled {
		t.Error("expected same-origin POST to reach the handler")
	}
}

func TestOriginProtectionMiddleware_BlocksCrossOriginMutation(t *testing.T) {
	t.Parallel()

	handlerCalled := false
	handler := OriginProtectionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Host = "localhost:7365"
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
	if handlerCalled {
		t.Error("expected cross-origin POST to be rejected before handler execution")
	}
}

// TestMiddlewareChain_OrderMatters tests that middleware chain executes in correct order.
func TestMiddlewareChain_OrderMatters(t *testing.T) {
	t.Parallel()

	var executionOrder []string

	handler := HostValidationMiddleware(
		CORSMiddleware(
			OriginProtectionMiddleware(
				SecurityHeadersMiddleware(
					CompressionMiddleware(
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							executionOrder = append(executionOrder, "handler")
							w.WriteHeader(http.StatusOK)
						}))))))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "localhost:7365"
	req.Header.Set("Origin", "http://localhost:7365")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify CORS headers (outermost middleware)
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:7365" {
		t.Error("expected CORS headers to be set")
	}

	// Verify Security headers
	if w.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Error("expected security headers to be set")
	}

	// Verify compression
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected compression to be applied")
	}

	// Verify handler was called
	if len(executionOrder) == 0 || executionOrder[0] != "handler" {
		t.Error("expected handler to be executed")
	}
}

// TestFormatDuration tests duration formatting.
func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{100 * time.Millisecond, "100"},
		{1 * time.Second, "1000"},
		{1500 * time.Millisecond, "1500"},
		{50 * time.Microsecond, "0.05"},
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

// TestTimingResponseWriter_MultipleWrites tests that Server-Timing is only set once.
func TestTimingResponseWriter_MultipleWrites(t *testing.T) {
	t.Parallel()

	handler := PerformanceMonitoringMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Multiple writes
		_, _ = w.Write([]byte("first"))
		_, _ = w.Write([]byte("second"))
		_, _ = w.Write([]byte("third"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should have exactly one Server-Timing header
	serverTimings := w.Header().Values("Server-Timing")
	if len(serverTimings) != 1 {
		t.Errorf("expected exactly one Server-Timing header, got %d", len(serverTimings))
	}
}

// TestGzipResponseWriter_StatusCode tests that status code is captured correctly.
func TestGzipResponseWriter_StatusCode(t *testing.T) {
	t.Parallel()

	handler := CompressionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
