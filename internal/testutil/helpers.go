// Package testutil provides common test utilities and helpers to reduce duplication
// across test files.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempConfig creates a temporary config file with the given content for testing.
// The file is automatically cleaned up when the test completes.
//
// Example:
//
//	configPath := testutil.TempConfig(t, `region: "us"\ncallback_port: 8080`)
//	store := config.NewFileStore(configPath)
func TempConfig(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	return configPath
}

// TempDir creates a temporary directory for testing.
// The directory is automatically cleaned up when the test completes.
//
// Note: This is just a wrapper around t.TempDir() for consistency.
func TempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// TempFile creates a temporary file with the given content for testing.
// The file is automatically cleaned up when the test completes.
//
// Example:
//
//	filePath := testutil.TempFile(t, "test.txt", "file contents")
func TempFile(t *testing.T, name, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, name)

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return filePath
}

// AssertNoError fails the test if err is not nil.
//
// Example:
//
//	testutil.AssertNoError(t, err, "failed to create user")
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError fails the test if err is nil.
//
// Example:
//
//	testutil.AssertError(t, err, "expected error for invalid input")
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected error, got nil", msg)
	}
}

// AssertEqual fails the test if got != want.
//
// Example:
//
//	testutil.AssertEqual(t, user.Name, "John", "user name")
func AssertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()

	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertNotEqual fails the test if got == want.
//
// Example:
//
//	testutil.AssertNotEqual(t, user.ID, "", "user ID should not be empty")
func AssertNotEqual[T comparable](t *testing.T, got, unwanted T, msg string) {
	t.Helper()

	if got == unwanted {
		t.Errorf("%s: got %v, should not equal %v", msg, got, unwanted)
	}
}

// AssertContains fails the test if haystack does not contain needle.
//
// Example:
//
//	testutil.AssertContains(t, output, "Success", "output should contain success message")
func AssertContains(t *testing.T, haystack, needle, msg string) {
	t.Helper()

	if !contains(haystack, needle) {
		t.Errorf("%s: %q does not contain %q", msg, haystack, needle)
	}
}

// AssertNotContains fails the test if haystack contains needle.
//
// Example:
//
//	testutil.AssertNotContains(t, output, "Error", "output should not contain error")
func AssertNotContains(t *testing.T, haystack, needle, msg string) {
	t.Helper()

	if contains(haystack, needle) {
		t.Errorf("%s: %q should not contain %q", msg, haystack, needle)
	}
}

// AssertNil fails the test if value is not nil.
//
// Example:
//
//	testutil.AssertNil(t, err, "error should be nil")
func AssertNil(t *testing.T, value any, msg string) {
	t.Helper()

	if value != nil {
		t.Errorf("%s: expected nil, got %v", msg, value)
	}
}

// AssertNotNil fails the test if value is nil.
//
// Example:
//
//	testutil.AssertNotNil(t, user, "user should not be nil")
func AssertNotNil(t *testing.T, value any, msg string) {
	t.Helper()

	if value == nil {
		t.Errorf("%s: expected non-nil value, got nil", msg)
	}
}

// AssertTrue fails the test if condition is false.
//
// Example:
//
//	testutil.AssertTrue(t, user.IsActive, "user should be active")
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()

	if !condition {
		t.Errorf("%s: expected true, got false", msg)
	}
}

// AssertFalse fails the test if condition is true.
//
// Example:
//
//	testutil.AssertFalse(t, user.IsDeleted, "user should not be deleted")
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()

	if condition {
		t.Errorf("%s: expected false, got true", msg)
	}
}

// AssertLen fails the test if the length of slice is not equal to expected.
//
// Example:
//
//	testutil.AssertLen(t, users, 5, "should have 5 users")
func AssertLen[T any](t *testing.T, slice []T, expected int, msg string) {
	t.Helper()

	if len(slice) != expected {
		t.Errorf("%s: got length %d, want %d", msg, len(slice), expected)
	}
}

// AssertEmpty fails the test if slice is not empty.
//
// Example:
//
//	testutil.AssertEmpty(t, errors, "should have no errors")
func AssertEmpty[T any](t *testing.T, slice []T, msg string) {
	t.Helper()

	if len(slice) != 0 {
		t.Errorf("%s: expected empty slice, got length %d", msg, len(slice))
	}
}

// AssertNotEmpty fails the test if slice is empty.
//
// Example:
//
//	testutil.AssertNotEmpty(t, results, "should have results")
func AssertNotEmpty[T any](t *testing.T, slice []T, msg string) {
	t.Helper()

	if len(slice) == 0 {
		t.Errorf("%s: expected non-empty slice, got empty slice", msg)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
