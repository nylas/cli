//go:build !windows

package browser

import "syscall"

// openURL opens a URL in the default browser with proper process isolation.
// On Unix systems, the browser is started in its own process group to prevent
// SIGINT (Ctrl+C) from propagating when the user stops the CLI.
func openURL(url string) error {
	cmd := createCommand(url)

	// Start the browser in its own process group.
	// This prevents SIGINT (Ctrl+C) from propagating to the browser
	// when the user stops the CLI.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd.Start()
}
