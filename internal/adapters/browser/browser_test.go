package browser

import (
	"slices"
	"testing"
)

func TestNewDefaultBrowser(t *testing.T) {
	browser := NewDefaultBrowser()
	if browser == nil {
		t.Error("NewDefaultBrowser() returned nil")
	}
}

func TestDefaultBrowser_Open(t *testing.T) {
	// Skip to avoid opening actual browser during automated test runs
	// This test is for manual verification only
	t.Skip("skipping browser open test - use manual testing to verify browser functionality")

	browser := NewDefaultBrowser()

	// We can't actually test opening a URL without a browser
	// Just test that it doesn't panic and returns a result
	err := browser.Open("https://example.com")
	// Error is OK (no browser available), we just want to ensure it doesn't panic
	_ = err
}

func TestCreateCommand(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"simple URL", "https://example.com"},
		{"URL with path", "https://example.com/page"},
		{"URL with query", "https://example.com?param=value"},
		{"local URL", "http://localhost:3000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createCommand(tt.url)

			if cmd == nil {
				t.Fatal("createCommand() returned nil")
				return
			}

			// Verify the command was created (we can't test execution)
			if cmd.Path == "" {
				t.Error("Command path is empty")
			}

			// Verify the URL is in the args
			if !slices.Contains(cmd.Args, tt.url) {
				t.Errorf("Command args %v do not contain URL %q", cmd.Args, tt.url)
			}
		})
	}
}

func TestCreateCommand_ReturnsNonNil(t *testing.T) {
	// Test that createCommand always returns a non-nil command
	cmd := createCommand("https://example.com")
	if cmd == nil {
		t.Fatal("createCommand() should never return nil")
	}
}
