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
	return createCommandForOS(runtime.GOOS, url)
}

func createCommandForOS(goos, url string) *exec.Cmd {
	switch goos {
	case "linux":
		// Use xdg-open on Linux
		return exec.Command("xdg-open", url)
	case "darwin":
		// Use open on macOS
		return exec.Command("open", url)
	case "windows":
		// Invoke the URL handler directly so shell metacharacters remain inert.
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
	default:
		// Fallback to xdg-open
		return exec.Command("xdg-open", url)
	}
}
