package common

import (
	"context"
	"testing"
	"time"
)

func TestCreateContext(t *testing.T) {
	ctx, cancel := CreateContext()
	defer cancel()

	if ctx == nil {
		t.Fatal("CreateContext() returned nil context")
	}

	// Check that context has a deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("CreateContext() context has no deadline")
	}

	// Check that deadline is approximately 90 seconds from now (TimeoutAPI)
	expectedDeadline := time.Now().Add(90 * time.Second)
	diff := expectedDeadline.Sub(deadline)
	if diff < -1*time.Second || diff > 1*time.Second {
		t.Errorf("CreateContext() deadline is %v, expected around 90s from now", deadline)
	}
}

func TestCreateContextWithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "5 second timeout",
			timeout: 5 * time.Second,
		},
		{
			name:    "1 minute timeout",
			timeout: 1 * time.Minute,
		},
		{
			name:    "100ms timeout",
			timeout: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := CreateContextWithTimeout(tt.timeout)
			defer cancel()

			if ctx == nil {
				t.Fatal("CreateContextWithTimeout() returned nil context")
			}

			deadline, ok := ctx.Deadline()
			if !ok {
				t.Error("CreateContextWithTimeout() context has no deadline")
			}

			expectedDeadline := time.Now().Add(tt.timeout)
			diff := expectedDeadline.Sub(deadline)
			// Allow 100ms tolerance
			if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
				t.Errorf("CreateContextWithTimeout(%v) deadline diff is %v, expected < 100ms", tt.timeout, diff)
			}
		})
	}
}

func TestCreateContext_Cancellation(t *testing.T) {
	ctx, cancel := CreateContext()

	// Context should not be done initially
	select {
	case <-ctx.Done():
		t.Error("Context was done before cancellation")
	default:
		// Expected: context is not done
	}

	// Cancel the context
	cancel()

	// Context should be done after cancellation
	select {
	case <-ctx.Done():
		// Expected: context is done
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", ctx.Err())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Context was not done after cancellation")
	}
}
