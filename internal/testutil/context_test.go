package testutil_test

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/testutil"
)

func TestTestContext(t *testing.T) {
	ctx := testutil.TestContext(t)
	if ctx == nil {
		t.Fatal("Expected non-nil context")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("Expected context to have deadline")
	}

	// Should be approximately 30 seconds from now
	actualDuration := time.Until(deadline)
	if actualDuration < 29*time.Second || actualDuration > 31*time.Second {
		t.Errorf("Expected deadline ~30s from now, got %v", actualDuration)
	}
}

func TestLongTestContext(t *testing.T) {
	ctx := testutil.LongTestContext(t)
	if ctx == nil {
		t.Fatal("Expected non-nil context")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("Expected context to have deadline")
	}

	// Should be approximately 120 seconds from now
	actualDuration := time.Until(deadline)
	if actualDuration < 119*time.Second || actualDuration > 121*time.Second {
		t.Errorf("Expected deadline ~120s from now, got %v", actualDuration)
	}
}

func TestQuickTestContext(t *testing.T) {
	ctx := testutil.QuickTestContext(t)
	if ctx == nil {
		t.Fatal("Expected non-nil context")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("Expected context to have deadline")
	}

	// Should be approximately 5 seconds from now
	actualDuration := time.Until(deadline)
	if actualDuration < 4*time.Second || actualDuration > 6*time.Second {
		t.Errorf("Expected deadline ~5s from now, got %v", actualDuration)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx := testutil.TestContext(t)

	// Context should not be cancelled initially
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be cancelled initially")
	default:
		// Expected
	}

	// Context should be cancelled after test cleanup
	// (This is implicitly tested by t.Cleanup when test finishes)
}

func TestPointerHelpers(t *testing.T) {
	// Test StringPtr
	str := "test"
	strPtr := testutil.StringPtr(str)
	if strPtr == nil {
		t.Fatal("StringPtr returned nil")
		return
	}
	if *strPtr != str {
		t.Errorf("Expected %q, got %q", str, *strPtr)
	}

	// Test BoolPtr
	boolVal := true
	boolPtr := testutil.BoolPtr(boolVal)
	if boolPtr == nil {
		t.Fatal("BoolPtr returned nil")
		return
	}
	if *boolPtr != boolVal {
		t.Errorf("Expected %v, got %v", boolVal, *boolPtr)
	}

	// Test IntPtr
	intVal := 42
	intPtr := testutil.IntPtr(intVal)
	if intPtr == nil {
		t.Fatal("IntPtr returned nil")
		return
	}
	if *intPtr != intVal {
		t.Errorf("Expected %d, got %d", intVal, *intPtr)
	}
}

func TestRequireEnv(t *testing.T) {
	// Test with existing env var
	t.Setenv("TEST_VAR", "test_value")
	value := testutil.RequireEnv(t, "TEST_VAR")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %q", value)
	}

	// Note: Testing the skip behavior is complex with the standard testing package
	// The skip functionality is tested implicitly when RequireEnv is used in real tests
}
