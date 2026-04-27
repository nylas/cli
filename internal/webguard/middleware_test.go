package webguard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsLoopbackHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		host string
		want bool
	}{
		{"localhost with port", "localhost:7365", true},
		{"localhost without port", "localhost", true},
		{"LOCALHOST mixed case", "LocalHost:7365", true},
		{"127.0.0.1 with port", "127.0.0.1:7365", true},
		{"127.1.2.3 (loopback range)", "127.1.2.3:80", true},
		{"::1 IPv6 loopback", "[::1]:7365", true},
		{"public IP", "8.8.8.8:80", false},
		{"public hostname", "evil.example:7365", false},
		{"empty", "", false},
		{"whitespace", "   ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsLoopbackHost(tc.host); got != tc.want {
				t.Fatalf("IsLoopbackHost(%q) = %v, want %v", tc.host, got, tc.want)
			}
		})
	}
}

func TestHostValidationMiddleware(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		host string
		want int
	}{
		{"loopback host allowed", "localhost:7367", http.StatusOK},
		{"non-loopback rejected", "evil.example:7367", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := HostValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tc.host
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestOriginProtectionMiddleware(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		method  string
		host    string
		origin  string
		referer string
		want    int
	}{
		{"GET passes without origin", http.MethodGet, "localhost:7367", "", "", http.StatusOK},
		{"POST same-origin passes", http.MethodPost, "localhost:7367", "http://localhost:7367", "", http.StatusOK},
		{"POST cross-origin rejected", http.MethodPost, "localhost:7367", "https://evil.example", "", http.StatusForbidden},
		{"POST without origin/referer rejected", http.MethodPost, "localhost:7367", "", "", http.StatusForbidden},
		{"DELETE cross-origin rejected", http.MethodDelete, "localhost:7367", "https://evil.example", "", http.StatusForbidden},
		// Browsers strip Origin in some cross-document scenarios (file://,
		// strict Referrer-Policy on the source page). The middleware must
		// fall back to the Referer header — these cases verify that path
		// is exercised, not just the Origin path.
		{"POST referer-only same-origin passes", http.MethodPost, "localhost:7367", "", "http://localhost:7367/dashboard", http.StatusOK},
		{"POST referer-only cross-origin rejected", http.MethodPost, "localhost:7367", "", "https://evil.example/page", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := OriginProtectionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(tc.method, "/", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestIsLoopbackBrowserOrigin(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		rawOrigin   string
		requestHost string
		want        bool
	}{
		{"matching loopback origin", "http://localhost:7367", "localhost:7367", true},
		{"127.0.0.1 matches localhost no — different host string", "http://127.0.0.1:7367", "localhost:7367", false},
		{"https origin rejected", "https://localhost:7367", "localhost:7367", false},
		{"non-loopback origin rejected", "http://evil.example", "localhost:7367", false},
		{"non-loopback request host rejected", "http://localhost:7367", "evil.example", false},
		{"empty origin rejected", "", "localhost:7367", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsLoopbackBrowserOrigin(tc.rawOrigin, tc.requestHost); got != tc.want {
				t.Fatalf("IsLoopbackBrowserOrigin(%q,%q) = %v, want %v", tc.rawOrigin, tc.requestHost, got, tc.want)
			}
		})
	}
}
