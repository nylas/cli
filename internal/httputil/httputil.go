// Package httputil provides common HTTP utilities shared across servers.
package httputil

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/version"
)

// MaxRequestBodySize is the maximum allowed request body size (1MB).
// This prevents memory exhaustion attacks via large payloads.
const MaxRequestBodySize = 1 << 20

// DefaultClientTimeout is the standard timeout for outbound HTTP clients. It
// tracks domain.TimeoutAPI (120s) so the client-level cap and the per-request
// API timeout share one source of truth.
const DefaultClientTimeout = domain.TimeoutAPI

// userAgentTransport sets the standard CLI User-Agent on every outbound
// request that doesn't already specify one, so all clients built via
// NewClient identify consistently.
type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") != "" {
		return t.base.RoundTrip(req)
	}
	// RoundTrippers must not mutate the caller's request (net/http contract) —
	// a retried or shared request would race. Clone before setting the header.
	clone := req.Clone(req.Context())
	clone.Header.Set("User-Agent", version.UserAgent())
	return t.base.RoundTrip(clone)
}

// NewClient returns an outbound HTTP client with the given timeout and the
// standard CLI User-Agent. A zero (or negative) timeout uses DefaultClientTimeout.
//
// Most callers should use the shared DefaultClient instead of minting their
// own; NewClient exists for the rare case that needs a non-default timeout.
func NewClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = DefaultClientTimeout
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: userAgentTransport{base: http.DefaultTransport},
	}
}

// DefaultClient is the shared outbound HTTP client (120s timeout + standard
// User-Agent). http.Client is safe for concurrent use, so every caller that
// wants the default policy should reuse this single instance rather than
// constructing its own. Do not mutate it; build a dedicated client for
// special behavior (e.g. disabled redirects).
var DefaultClient = NewClient(DefaultClientTimeout)

// NewServer returns an *http.Server hardened with the standard CLI defaults:
// a 10s header-read timeout, 120s idle timeout, and a 1MB max header size.
//
// writeTimeout is per-caller because streaming endpoints (e.g. SSE) need a
// long or disabled (0) write deadline; pass 0 for no write timeout.
func NewServer(addr string, handler http.Handler, writeTimeout time.Duration) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: domain.HTTPReadHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       domain.HTTPIdleTimeout,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}
}

// LimitedBody wraps a request body with a size limit.
// Returns a ReadCloser that will return an error if the body exceeds maxBytes.
func LimitedBody(w http.ResponseWriter, r *http.Request, maxBytes int64) io.ReadCloser {
	return http.MaxBytesReader(w, r.Body, maxBytes)
}

// WriteJSON writes a JSON response with the given status code.
// It sets the Content-Type header to application/json.
//
// Encoder errors are logged rather than swallowed: once WriteHeader has
// fired we can't change the status code, but we *can* leave a server-side
// breadcrumb so that "client received truncated JSON" is debuggable
// instead of invisible. Common failure modes are client disconnect
// mid-write and unmarshallable values (e.g. NaN, channels) — both of
// which a developer would want to know about.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("WriteJSON encode failed", "status", status, "err", err)
	}
}

// DecodeJSON decodes JSON from the request body into the target.
// It uses LimitedBody to prevent oversized payloads.
func DecodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	return json.NewDecoder(LimitedBody(w, r, MaxRequestBodySize)).Decode(target)
}
