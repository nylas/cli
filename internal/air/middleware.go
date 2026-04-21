package air

import (
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// gzipResponseWriter wraps http.ResponseWriter to support gzip compression.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before writing.
func (w *gzipResponseWriter) WriteHeader(status int) {
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
}

// Write compresses the response body.
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// CompressionMiddleware adds gzip compression to responses.
// This significantly reduces bandwidth and improves load times.
func CompressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Don't compress already compressed formats
		if strings.HasSuffix(r.URL.Path, ".gz") ||
			strings.HasSuffix(r.URL.Path, ".jpg") ||
			strings.HasSuffix(r.URL.Path, ".jpeg") ||
			strings.HasSuffix(r.URL.Path, ".png") ||
			strings.HasSuffix(r.URL.Path, ".gif") ||
			strings.HasSuffix(r.URL.Path, ".woff") ||
			strings.HasSuffix(r.URL.Path, ".woff2") {
			next.ServeHTTP(w, r)
			return
		}

		// Create gzip writer
		gz := gzip.NewWriter(w)
		defer func() {
			_ = gz.Close() // Error is non-actionable in deferred context
		}()

		// Wrap response writer
		gzw := &gzipResponseWriter{
			Writer:         gz,
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Set headers
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // Length will change after compression

		next.ServeHTTP(gzw, r)
	})
}

// CacheMiddleware adds appropriate cache headers for static assets.
// This reduces server load and improves perceived performance.
func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// For local development tools, minimal caching is better
		// Only cache static images/fonts that rarely change
		if strings.HasSuffix(path, ".woff") ||
			strings.HasSuffix(path, ".woff2") ||
			strings.HasSuffix(path, ".png") ||
			strings.HasSuffix(path, ".jpg") ||
			strings.HasSuffix(path, ".jpeg") ||
			strings.HasSuffix(path, ".gif") ||
			strings.HasSuffix(path, ".svg") ||
			strings.HasSuffix(path, ".ico") {
			// Images and fonts - cache for 1 day
			w.Header().Set("Cache-Control", "public, max-age=86400")
		} else if strings.HasPrefix(path, "/api/") {
			// API responses - no cache (always fresh)
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		} else {
			// Everything else (JS, CSS, HTML) - minimal cache for instant updates
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		}

		next.ServeHTTP(w, r)
	})
}

// PerformanceMonitoringMiddleware tracks request timing and adds performance headers.
// This helps identify slow endpoints and enables browser performance monitoring.
func PerformanceMonitoringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create custom response writer to capture status code and add timing after response
		srw := &timingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			start:          start,
		}

		// Process request
		next.ServeHTTP(srw, r)
	})
}

// timingResponseWriter wraps http.ResponseWriter to add performance timing.
type timingResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	start         time.Time
	headerWritten bool
}

// WriteHeader captures the status code and adds Server-Timing header.
func (w *timingResponseWriter) WriteHeader(code int) {
	if !w.headerWritten {
		w.statusCode = code

		// Add Server-Timing header before writing
		duration := time.Since(w.start)
		w.ResponseWriter.Header().Set("Server-Timing",
			"total;dur="+formatDuration(duration))

		w.headerWritten = true
		w.ResponseWriter.WriteHeader(code)
	}
}

// Write ensures headers are written before body.
func (w *timingResponseWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// formatDuration formats duration in milliseconds with 2 decimal places.
func formatDuration(d time.Duration) string {
	ms := float64(d.Nanoseconds()) / 1e6
	// Use strconv for accurate formatting
	formatted := strconv.FormatFloat(ms, 'f', 2, 64)
	// Remove trailing zeros and decimal point if not needed
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")
	return formatted
}

// SecurityHeadersMiddleware adds security headers to all responses.
// This improves security posture and prevents common attacks.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (relaxed for local development)
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' data: https://fonts.gstatic.com; "+
				"connect-src 'self' https://api.us.nylas.com https://api.eu.nylas.com;")

		next.ServeHTTP(w, r)
	})
}

// MethodOverrideMiddleware allows using X-HTTP-Method-Override header.
// This enables REST methods in environments that only support GET/POST.
func MethodOverrideMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			override := r.Header.Get("X-HTTP-Method-Override")
			if override != "" {
				r.Method = strings.ToUpper(override)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// HostValidationMiddleware rejects requests that do not target an explicit
// loopback Air host.
func HostValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAllowedAirHost(r.Host) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware reflects CORS headers only for the Air origin. Cross-origin
// sites should not be able to read responses from the local Air API.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && isAllowedBrowserOrigin(origin, r.Host) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-HTTP-Method-Override")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			if origin == "" || !isAllowedBrowserOrigin(origin, r.Host) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// OriginProtectionMiddleware rejects state-changing requests that do not
// originate from the active Air origin.
func OriginProtectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresSameOriginProtection(r.Method) || requestMatchesAirOrigin(r) {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

func requiresSameOriginProtection(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func requestMatchesAirOrigin(r *http.Request) bool {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return isAllowedBrowserOrigin(origin, r.Host)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return isAllowedBrowserOrigin(referer, r.Host)
	}
	return false
}

func isAllowedBrowserOrigin(rawOrigin, requestHost string) bool {
	if !isAllowedAirHost(requestHost) {
		return false
	}

	parsed, err := url.Parse(rawOrigin)
	if err != nil {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" {
		return false
	}
	if !isAllowedAirHost(parsed.Host) {
		return false
	}

	return strings.EqualFold(parsed.Host, requestHost)
}

func isAllowedAirHost(requestHost string) bool {
	host := requestHost
	if parsedHost, _, err := net.SplitHostPort(requestHost); err == nil {
		host = parsedHost
	}

	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
