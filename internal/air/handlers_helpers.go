package air

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout is the default API request timeout.
const defaultTimeout = 30 * time.Second

// withTimeout creates a context with the default timeout.
// Returns the context and a cancel function that must be deferred.
func (s *Server) withTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), defaultTimeout)
}

// requireConfig checks if the Nylas client is configured.
// Returns true if configured, false if not (error response already written).
// Callers should return immediately when this returns false.
func (s *Server) requireConfig(w http.ResponseWriter) bool {
	if s.nylasClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Not configured. Run 'nylas auth login' first.",
		})
		return false
	}
	return true
}

// parseJSONBody decodes a JSON request body into the provided destination.
// Returns true if successful, false if not (error response already written).
// Callers should return immediately when this returns false.
//
// The raw decoder error is logged via slog rather than echoed to the
// client. encoding/json's UnmarshalTypeError carries Go struct field
// paths and value fragments from the request body — fingerprintable
// detail that does not belong in a browser toast. This mirrors the
// writeUpstreamError discipline used at the upstream-error sites.
func parseJSONBody[T any](w http.ResponseWriter, r *http.Request, dest *T) bool {
	if err := json.NewDecoder(limitedBody(w, r)).Decode(dest); err != nil {
		slog.Warn("invalid JSON request body",
			"err", err,
			"path", r.URL.Path,
			"method", r.Method,
		)
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return false
	}
	return true
}

// writeBadParamError writes a generic 400 to the client and logs the
// raw parsing error via slog. Callers pass the parameter key (e.g.
// "start_time") which is safe to surface; the parser's err carries
// the raw query value, which is NOT — `parseInt64Param` formats it
// via %q and reflecting that back is gratuitous attacker-input echo.
func writeBadParamError(w http.ResponseWriter, key string, perr error) {
	slog.Warn("invalid query parameter", "key", key, "err", perr)
	writeError(w, http.StatusBadRequest, "invalid "+key)
}

// handleDemoMode returns the demo response if in demo mode.
// Returns true if demo mode is active (response already written), false otherwise.
// Callers should return immediately when this returns true.
func (s *Server) handleDemoMode(w http.ResponseWriter, data any) bool {
	if s.demoMode {
		writeJSON(w, http.StatusOK, data)
		return true
	}
	return false
}

// requireMethod checks if the request method matches the expected method.
// Returns true if method matches, false if not (error response already written).
// Callers should return immediately when this returns false.
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}
	return true
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeUpstreamError logs the raw upstream error via slog and writes a
// generic JSON envelope to the client. Use when an upstream call (Nylas
// API, cache, etc.) failed and the user-facing message must NOT include
// raw error details — Nylas error strings can include grant IDs,
// endpoint paths, or response-body fragments that don't belong in a
// browser toast. The log line carries the raw err for debugging.
//
// `msg` is the user-facing string written as-is (no err.Error()
// concatenation). `attrs` are extra slog key/value pairs appended after
// "err". Callers in handlers_email_rsvp.go model the same pattern
// inline; this helper makes the convention easy to apply elsewhere.
func writeUpstreamError(w http.ResponseWriter, status int, msg string, err error, attrs ...any) {
	slog.Error(msg, append([]any{"err", err}, attrs...)...)
	writeError(w, status, msg)
}

// redactEmail returns a log-safe rendering of an email address: the
// local part is replaced with "***" so the domain remains debuggable
// without writing the full address into log files. Empty input stays
// empty so missing-account branches don't gain a misleading "***".
//
// Example: "alice@example.com" → "***@example.com".
func redactEmail(email string) string {
	if email == "" {
		return ""
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "***"
	}
	return "***" + email[at:]
}

// withAuthGrant combines demo mode check, config check, and grant ID retrieval.
// Returns the grant ID if all checks pass, or empty string if any check fails
// (appropriate error response already written).
//
// Usage:
//
//	grantID := s.withAuthGrant(w, demoResponse)
//	if grantID == "" {
//	    return
//	}
func (s *Server) withAuthGrant(w http.ResponseWriter, demoResponse any) string {
	if demoResponse != nil && s.handleDemoMode(w, demoResponse) {
		return ""
	}
	if !s.requireConfig(w) {
		return ""
	}
	grantID, ok := s.requireDefaultGrant(w)
	if !ok {
		return ""
	}
	return grantID
}
