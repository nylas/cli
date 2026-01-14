// Package browser provides browser opening functionality.
package browser

import (
	"os/exec"
	"runtime"
)

// DefaultBrowser opens URLs in the system default browser.
type DefaultBrowser struct{}

// NewDefaultBrowser creates a new DefaultBrowser.
func NewDefaultBrowser() *DefaultBrowser {
	return &DefaultBrowser{}
}

// Open opens a URL in the default browser.
// On Linux, it ensures the browser is started in its own process group
// so that Ctrl+C doesn't kill the browser when stopping the CLI.
func (b *DefaultBrowser) Open(url string) error {
	return openURL(url)
}

// createCommand creates the appropriate command to open a URL based on the OS.
func createCommand(url string) *exec.Cmd {
	switch runtime.GOOS {
	case "linux":
		// Use xdg-open on Linux
		return exec.Command("xdg-open", url)
	case "darwin":
		// Use open on macOS
		return exec.Command("open", url)
	case "windows":
		// Use start on Windows
		return exec.Command("cmd", "/c", "start", url)
	default:
		// Fallback to xdg-open
		return exec.Command("xdg-open", url)
	}
}

// MockBrowser is a mock implementation for testing.
type MockBrowser struct {
	OpenCalled bool
	LastURL    string
	OpenFunc   func(url string) error
}

// NewMockBrowser creates a new MockBrowser.
func NewMockBrowser() *MockBrowser {
	return &MockBrowser{}
}

// Open records the call and optionally calls the custom function.
func (m *MockBrowser) Open(url string) error {
	m.OpenCalled = true
	m.LastURL = url
	if m.OpenFunc != nil {
		return m.OpenFunc(url)
	}
	return nil
}
