package common

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCreateContext(t *testing.T) {
	SetAPITimeout(0) // default
	defer SetAPITimeout(0)

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

	// Default deadline is TimeoutAPI from now.
	expectedDeadline := time.Now().Add(domain.TimeoutAPI)
	diff := expectedDeadline.Sub(deadline)
	if diff < -1*time.Second || diff > 1*time.Second {
		t.Errorf("CreateContext() deadline is %v, expected around %v from now", deadline, domain.TimeoutAPI)
	}
}

func TestCreateContext_HonorsConfiguredTimeout(t *testing.T) {
	SetAPITimeout(45 * time.Second)
	defer SetAPITimeout(0)

	ctx, cancel := CreateContext()
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("CreateContext() context has no deadline")
	}
	diff := time.Until(deadline) - 45*time.Second
	if diff < -1*time.Second || diff > 1*time.Second {
		t.Errorf("CreateContext() deadline ~%v from now, expected ~45s", time.Until(deadline))
	}

	// Resetting to default restores TimeoutAPI.
	SetAPITimeout(0)
	ctx2, cancel2 := CreateContext()
	defer cancel2()
	d2, _ := ctx2.Deadline()
	if diff := time.Until(d2) - domain.TimeoutAPI; diff < -1*time.Second || diff > 1*time.Second {
		t.Errorf("after reset, deadline ~%v, expected ~%v", time.Until(d2), domain.TimeoutAPI)
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
