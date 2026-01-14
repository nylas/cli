package common

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// CreateContext creates a context with the standard API timeout.
// Returns the context and a cancel function that should be deferred.
func CreateContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), domain.TimeoutAPI)
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
