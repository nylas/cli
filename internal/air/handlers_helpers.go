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

// parseJSONBody decodes the request body into dest. Returns false on
// error after writing a generic 400; the raw decoder error stays in slog
// (it can quote request bytes — PII or attacker input).
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

// writeBadParamError writes a generic 400 ("invalid <key>") and logs the
// raw parser error via slog (it formats the raw query value with %q).
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

// writeUpstreamError writes msg to the client and logs the raw err via
// slog. Use whenever an upstream error string could leak grant IDs,
// endpoint paths, or response fragments. attrs are extra slog kv pairs.
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
