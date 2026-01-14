//go:build windows

package browser

// openURL opens a URL in the default browser on Windows.
// Windows doesn't need special process group handling.
func openURL(url string) error {
	cmd := createCommand(url)
	return cmd.Start()
}
