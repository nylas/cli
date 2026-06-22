package common

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// apiTimeoutNanos holds the resolved per-request API timeout (0 = use the
// domain.TimeoutAPI default). It is set once at client construction
// (GetNylasClient) from the install's config/env and read by CreateContext,
// which runs on every command. atomic because spinner/background goroutines
// may also create contexts.
var apiTimeoutNanos atomic.Int64

// SetAPITimeout records the resolved API timeout for subsequent CreateContext
// calls. A non-positive value resets to the default.
func SetAPITimeout(d time.Duration) {
	if d <= 0 {
		apiTimeoutNanos.Store(0)
		return
	}
	apiTimeoutNanos.Store(int64(d))
}

// apiTimeout returns the configured API timeout, or the default if unset.
func apiTimeout() time.Duration {
	if n := apiTimeoutNanos.Load(); n > 0 {
		return time.Duration(n)
	}
	return domain.TimeoutAPI
}

// CreateContext creates a context with the configured API timeout (see
// SetAPITimeout; defaults to domain.TimeoutAPI).
// Returns the context and a cancel function that should be deferred.
func CreateContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), apiTimeout())
}

// CreateContextWithTimeout creates a context with a custom timeout.
func CreateContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// CreateLongContext creates a context with the OAuth timeout (5 minutes).
// Use for operations requiring user interaction (OAuth flows, browser auth).
func CreateLongContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), domain.TimeoutOAuth)
}
