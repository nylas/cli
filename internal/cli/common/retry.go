package common

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"time"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxRetries  int           // Maximum number of retries (default: 3)
	BaseDelay   time.Duration // Initial delay (default: 1s)
	MaxDelay    time.Duration // Maximum delay cap (default: 30s)
	Multiplier  float64       // Delay multiplier (default: 2.0)
	JitterRatio float64       // Jitter ratio 0-1 (default: 0.1)
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:  3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0.1,
	}
}

// RetryableError wraps an error that should be retried.
type RetryableError struct {
	Err        error
	StatusCode int
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error should be retried.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for RetryableError
	var retryErr *RetryableError
	if errors.As(err, &retryErr) {
		return true
	}

	// Check for context errors (not retryable)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network errors are generally retryable
	return true
}

// IsRetryableStatusCode checks if an HTTP status code should be retried.
func IsRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// RetryFunc is a function that can be retried.
type RetryFunc func() error

// WithRetry executes a function with retry logic.
func WithRetry(ctx context.Context, config RetryConfig, fn RetryFunc) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !IsRetryable(err) {
			Debug("not retrying non-retryable error", "error", err)
			return err
		}

		// Don't sleep after the last attempt
		if attempt >= config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := calculateDelay(config, attempt)

		Debug("retrying after error",
			"attempt", attempt+1,
			"max_retries", config.MaxRetries,
			"delay", delay,
			"error", err,
		)

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// calculateDelay calculates the delay for a retry attempt with jitter.
func calculateDelay(config RetryConfig, attempt int) time.Duration {
	// Exponential backoff: baseDelay * multiplier^attempt
	delay := float64(config.BaseDelay)
	for i := 0; i < attempt; i++ {
		delay *= config.Multiplier
	}

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter using crypto/rand for unpredictable timing
	if config.JitterRatio > 0 {
		// Generate random value 0-1000000 and convert to float 0.0-1.0
		n, err := rand.Int(rand.Reader, big.NewInt(1000001))
		if err == nil {
			randFloat := float64(n.Int64()) / 1000000.0              // 0.0 to 1.0
			jitter := delay * config.JitterRatio * (randFloat*2 - 1) // -jitter to +jitter
			delay += jitter
		}
	}

	return time.Duration(delay)
}

// NoRetryConfig returns a config that disables retries.
func NoRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 0,
	}
}
