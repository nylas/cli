//go:build integration
// +build integration

package air

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntegration_Middleware_Compression(t *testing.T) {
	// Create test server with middleware
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("test data ", 100))) // 1000 bytes uncompressed
	}))

	handler := CompressionMiddleware(mux)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Test WITH gzip
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Error("expected Content-Encoding: gzip")
	}

	// Test WITHOUT gzip
	req2, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.Header.Get("Content-Encoding") == "gzip" {
		t.Error("expected no gzip compression without Accept-Encoding header")
	}

	t.Log("✓ Compression middleware working correctly")
}

// TestIntegration_Middleware_SecurityHeaders verifies all security headers are present.
func TestIntegration_Middleware_SecurityHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Use the server's handler (which has all middleware applied)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeadersMiddleware(mux)
	handler.ServeHTTP(w, req)

	requiredHeaders := map[string]string{
		"X-Frame-Options":         "SAMEORIGIN",
		"X-Content-Type-Options":  "nosniff",
		"X-XSS-Protection":        "1; mode=block",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Content-Security-Policy": "", // Just verify it exists
	}

	for header, expected := range requiredHeaders {
		actual := w.Header().Get(header)
		if actual == "" {
			t.Errorf("missing security header: %s", header)
		} else if expected != "" && actual != expected {
			t.Errorf("%s: expected %q, got %q", header, expected, actual)
		}
	}

	// Verify CSP includes correct API endpoints
	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "https://api.us.nylas.com") {
		t.Error("CSP should include api.us.nylas.com")
	}
	if !strings.Contains(csp, "https://api.eu.nylas.com") {
		t.Error("CSP should include api.eu.nylas.com")
	}

	t.Log("✓ All security headers present and correct")
}

// TestIntegration_Middleware_CacheHeaders verifies cache headers based on endpoint type.
func TestIntegration_Middleware_CacheHeaders(t *testing.T) {
	tests := []struct {
		path          string
		expectedCache string
	}{
		// CSS files use minimal caching for instant updates during local development
		{"/static/css/main.css", "no-cache, must-revalidate"},
		// API responses are never cached
		{"/api/emails", "no-cache, no-store, must-revalidate"},
		// HTML pages use minimal caching for instant updates
		{"/", "no-cache, must-revalidate"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		w := httptest.NewRecorder()

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := CacheMiddleware(mux)
		handler.ServeHTTP(w, req)

		cacheControl := w.Header().Get("Cache-Control")
		if cacheControl != tt.expectedCache {
			t.Errorf("%s: expected Cache-Control %q, got %q", tt.path, tt.expectedCache, cacheControl)
		} else {
			t.Logf("✓ %s: correct cache headers", tt.path)
		}
	}
}

// TestIntegration_Middleware_CORS verifies CORS headers and preflight.
func TestIntegration_Middleware_CORS(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORSMiddleware(mux)

	// Test regular request
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected Access-Control-Allow-Origin: *")
	}

	// Test OPTIONS preflight
	reqOptions := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	wOptions := httptest.NewRecorder()
	handler.ServeHTTP(wOptions, reqOptions)

	if wOptions.Code != http.StatusNoContent {
		t.Errorf("OPTIONS should return 204, got %d", wOptions.Code)
	}

	if wOptions.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("OPTIONS should include Access-Control-Allow-Methods")
	}

	t.Log("✓ CORS headers working correctly")
}

// TestIntegration_Middleware_PerformanceMonitoring verifies Server-Timing headers.
func TestIntegration_Middleware_PerformanceMonitoring(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond) // Simulate work
		w.WriteHeader(http.StatusOK)
	})

	handler := PerformanceMonitoringMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	serverTiming := w.Header().Get("Server-Timing")
	if serverTiming == "" {
		t.Error("expected Server-Timing header")
	}

	if !strings.HasPrefix(serverTiming, "total;dur=") {
		t.Errorf("expected Server-Timing format 'total;dur=X', got %s", serverTiming)
	}

	t.Logf("✓ Server-Timing header present: %s", serverTiming)
}

// TestIntegration_Middleware_FullStack verifies all middleware working together.
func TestIntegration_Middleware_FullStack(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("integration test data ", 50)))
	})

	// Apply full middleware stack (same order as server.go)
	handler := CORSMiddleware(
		SecurityHeadersMiddleware(
			CompressionMiddleware(
				CacheMiddleware(
					PerformanceMonitoringMiddleware(mux)))))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify all middleware features are active
	checks := map[string]func() bool{
		"CORS":        func() bool { return w.Header().Get("Access-Control-Allow-Origin") == "*" },
		"Security":    func() bool { return w.Header().Get("X-Frame-Options") == "SAMEORIGIN" },
		"Compression": func() bool { return w.Header().Get("Content-Encoding") == "gzip" },
		"Cache":       func() bool { return w.Header().Get("Cache-Control") == "no-cache, no-store, must-revalidate" },
		"Performance": func() bool { return strings.HasPrefix(w.Header().Get("Server-Timing"), "total;dur=") },
	}

	for name, check := range checks {
		if !check() {
			t.Errorf("%s middleware not working in full stack", name)
		} else {
			t.Logf("✓ %s middleware active", name)
		}
	}

	t.Log("✓ Full middleware stack integration successful")
}
