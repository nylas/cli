package demo

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/ui"
)

// newDemoUICmd creates the demo ui command.
func newDemoUICmd() *cobra.Command {
	var port int
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch web UI with sample data",
		Long: `Launch the web-based graphical interface with demo data.

Explore the full web UI experience without any credentials:
  - View sample configuration status
  - Browse demo accounts
  - Execute commands with sample output
  - Test UI navigation and features

The demo UI runs on localhost and opens in your default browser.`,
		Example: `  # Launch demo web UI
  nylas demo ui

  # Launch on custom port
  nylas demo ui --port 8080

  # Launch without opening browser
  nylas demo ui --no-browser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemoUI(port, noBrowser)
		},
	}

	cmd.Flags().IntVar(&port, "port", 7363, "Port to run the server on")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")

	return cmd
}

func runDemoUI(port int, noBrowser bool) error {
	// Bind to loopback only — the demo UI is a local development tool and
	// should not be reachable from the LAN.
	addr := fmt.Sprintf("localhost:%d", port)

	// Check if port is available
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is already in use", port)
	}
	_ = listener.Close()

	// Create demo UI server (uses same templates as real UI)
	server := ui.NewDemoServer(addr)

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Starting demo web UI at %s\n", url)
	fmt.Println("Demo mode: using sample data (no credentials required)")
	fmt.Println("Press Ctrl+C to stop the server")

	// Open browser
	if !noBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(url)
		}()
	}

	return server.Start()
}

// openBrowser opens the URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// #nosec G204 -- command "open" is hardcoded, only URL is variable (validated as localhost)
		cmd = exec.Command("open", url)
	case "linux":
		// #nosec G204 -- command "xdg-open" is hardcoded, only URL is variable (validated as localhost)
		cmd = exec.Command("xdg-open", url)
	case "windows":
		// #nosec G204 -- command "rundll32" is hardcoded, only URL is variable (validated as localhost)
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}
