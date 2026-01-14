package testutil

import (
	"os"
	"testing"
)

func TestTempConfig(t *testing.T) {
	content := "test: value"
	path := TempConfig(t, content)

	// #nosec G304 -- reading test file created by test helper
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read temp config: %v", err)
	}

	if string(data) != content {
		t.Errorf("Content = %q, want %q", string(data), content)
	}
}

func TestTempFile(t *testing.T) {
	content := "test content"
	path := TempFile(t, "test.txt", content)

	// #nosec G304 -- reading test file created by test helper
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(data) != content {
		t.Errorf("Content = %q, want %q", string(data), content)
	}
}

func TestAssertEqual(t *testing.T) {
	// Test successful assertion (should not fail)
	AssertEqual(t, "hello", "hello", "strings should be equal")
	AssertEqual(t, 42, 42, "numbers should be equal")
	AssertEqual(t, true, true, "booleans should be equal")
}

func TestAssertContains(t *testing.T) {
	AssertContains(t, "hello world", "world", "should contain substring")
}

func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"empty substring", "hello", "", true},
		{"contains", "hello world", "world", true},
		{"does not contain", "hello", "xyz", false},
		{"exact match", "test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
