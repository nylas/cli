//go:build !integration

package common

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryableError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *RetryableError
		expected string
	}{
		{
			name: "basic error",
			err: &RetryableError{
				Err:        errors.New("test error"),
				StatusCode: 429,
			},
			expected: "test error",
		},
		{
			name: "no status code",
			err: &RetryableError{
				Err: errors.New("network failure"),
			},
			expected: "network failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestRetryableError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	retryErr := &RetryableError{
		Err:        originalErr,
		StatusCode: 503,
	}

	assert.Equal(t, originalErr, retryErr.Unwrap())
	assert.True(t, errors.Is(retryErr, originalErr))
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.BaseDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.Equal(t, 0.1, config.JitterRatio)
}

func TestNoRetryConfig(t *testing.T) {
	config := NoRetryConfig()

	assert.Equal(t, 0, config.MaxRetries)
}

func TestIsRetryable_NilError(t *testing.T) {
	assert.False(t, IsRetryable(nil))
}

func TestIsRetryable_RetryableError(t *testing.T) {
	retryErr := &RetryableError{
		Err:        errors.New("rate limited"),
		StatusCode: 429,
	}

	assert.True(t, IsRetryable(retryErr))
}

func TestIsRetryable_ContextErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "wrapped context canceled",
			err:      errors.Join(errors.New("wrapper"), context.Canceled),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryable(tt.err))
		})
	}
}

func TestIsRetryable_GenericErrors(t *testing.T) {
	// Generic errors are retryable by default (assuming network issues)
	err := errors.New("something went wrong")
	assert.True(t, IsRetryable(err))
}

func TestIsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"429 Too Many Requests", http.StatusTooManyRequests, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"502 Bad Gateway", http.StatusBadGateway, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout", http.StatusGatewayTimeout, true},
		{"200 OK", http.StatusOK, false},
		{"201 Created", http.StatusCreated, false},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"401 Unauthorized", http.StatusUnauthorized, false},
		{"403 Forbidden", http.StatusForbidden, false},
		{"404 Not Found", http.StatusNotFound, false},
		{"422 Unprocessable Entity", http.StatusUnprocessableEntity, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryableStatusCode(tt.statusCode))
		})
	}
}

func TestWithRetry_ImmediateSuccess(t *testing.T) {
	config := DefaultRetryConfig()
	calls := 0

	err := WithRetry(context.Background(), config, func() error {
		calls++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestWithRetry_SuccessAfterRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:  3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	err := WithRetry(context.Background(), config, func() error {
		calls++
		if calls < 3 {
			return &RetryableError{Err: errors.New("temporary failure")}
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestWithRetry_MaxRetriesExceeded(t *testing.T) {
	config := RetryConfig{
		MaxRetries:  2,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	expectedErr := errors.New("persistent failure")

	err := WithRetry(context.Background(), config, func() error {
		calls++
		return &RetryableError{Err: expectedErr}
	})

	assert.Error(t, err)
	assert.Equal(t, 3, calls) // Initial + 2 retries
	assert.True(t, errors.Is(err, expectedErr))
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	config := RetryConfig{
		MaxRetries:  3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	expectedErr := context.Canceled // Not retryable

	err := WithRetry(context.Background(), config, func() error {
		calls++
		return expectedErr
	})

	assert.Error(t, err)
	assert.Equal(t, 1, calls) // Should not retry
	assert.True(t, errors.Is(err, expectedErr))
}

func TestWithRetry_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := RetryConfig{
		MaxRetries:  5,
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	err := WithRetry(ctx, config, func() error {
		calls++
		if calls == 2 {
			cancel() // Cancel after second call
		}
		return &RetryableError{Err: errors.New("retry me")}
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.LessOrEqual(t, calls, 3) // Should stop soon after cancel
}

func TestWithRetry_ContextAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := DefaultRetryConfig()
	calls := 0

	err := WithRetry(ctx, config, func() error {
		calls++
		return nil
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 0, calls) // Should not call function at all
}

func TestWithRetry_NoRetries(t *testing.T) {
	config := NoRetryConfig()
	calls := 0

	err := WithRetry(context.Background(), config, func() error {
		calls++
		return &RetryableError{Err: errors.New("fail")}
	})

	assert.Error(t, err)
	assert.Equal(t, 1, calls) // Only initial call, no retries
}

func TestCalculateDelay_ExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0, // No jitter for predictable testing
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},  // 100ms
		{1, 200 * time.Millisecond},  // 100ms * 2
		{2, 400 * time.Millisecond},  // 100ms * 2^2
		{3, 800 * time.Millisecond},  // 100ms * 2^3
		{4, 1600 * time.Millisecond}, // 100ms * 2^4
	}

	for _, tt := range tests {
		t.Run("attempt "+string(rune('0'+tt.attempt)), func(t *testing.T) {
			delay := calculateDelay(config, tt.attempt)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestCalculateDelay_MaxDelayCap(t *testing.T) {
	config := RetryConfig{
		BaseDelay:   1 * time.Second,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	// Attempt 10 would be 1024 seconds without cap
	delay := calculateDelay(config, 10)

	assert.Equal(t, 5*time.Second, delay)
}

func TestCalculateDelay_WithJitter(t *testing.T) {
	config := RetryConfig{
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0.1, // 10% jitter
	}

	// Run multiple times to verify jitter produces varying results
	delays := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		delay := calculateDelay(config, 0)
		delays[delay] = true

		// Delay should be within 10% of base (900ms - 1100ms)
		assert.GreaterOrEqual(t, delay, 900*time.Millisecond)
		assert.LessOrEqual(t, delay, 1100*time.Millisecond)
	}

	// With jitter, we should get some variation (though not guaranteed)
	// This is a probabilistic test, but 10 iterations should produce some variation
}

func TestCalculateDelay_ZeroJitter(t *testing.T) {
	config := RetryConfig{
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0, // No jitter
	}

	// Should always return exact value
	for i := 0; i < 5; i++ {
		delay := calculateDelay(config, 0)
		assert.Equal(t, 1*time.Second, delay)
	}
}

func TestWithRetry_DelayBetweenRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:  2,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	start := time.Now()
	calls := 0

	err := WithRetry(context.Background(), config, func() error {
		calls++
		return &RetryableError{Err: errors.New("fail")}
	})

	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, 3, calls)
	// Should have delayed: 10ms + 20ms = 30ms minimum
	assert.GreaterOrEqual(t, elapsed, 25*time.Millisecond)
}

func TestRetryConfig_CustomConfiguration(t *testing.T) {
	config := RetryConfig{
		MaxRetries:  5,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    1 * time.Minute,
		Multiplier:  1.5,
		JitterRatio: 0.2,
	}

	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, config.BaseDelay)
	assert.Equal(t, 1*time.Minute, config.MaxDelay)
	assert.Equal(t, 1.5, config.Multiplier)
	assert.Equal(t, 0.2, config.JitterRatio)
}

func TestWithRetry_ContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	config := RetryConfig{
		MaxRetries:  10,
		BaseDelay:   50 * time.Millisecond, // Long delay to trigger timeout
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	err := WithRetry(ctx, config, func() error {
		calls++
		return &RetryableError{Err: errors.New("fail")}
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}
