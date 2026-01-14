package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nylas/cli/internal/adapters/tunnel"
	"github.com/nylas/cli/internal/adapters/webhookserver"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	var (
		port          int
		path          string
		tunnelType    string
		webhookSecret string
		jsonOutput    bool
		quiet         bool
	)

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start a local webhook receiver server",
		Long: `Start a local HTTP server to receive and display webhook events.

The server can optionally expose itself via a tunnel (cloudflared) for
receiving webhooks from the internet when developing locally.

Examples:
  # Start server on default port 3000
  nylas webhooks server

  # Start server with cloudflared tunnel
  nylas webhooks server --tunnel cloudflared

  # Start server on custom port with tunnel
  nylas webhooks server --port 8080 --tunnel cloudflared

  # Start server with webhook signature verification
  nylas webhooks server --tunnel cloudflared --secret your-webhook-secret

Press Ctrl+C to stop the server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(port, path, tunnelType, webhookSecret, jsonOutput, quiet)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to listen on")
	cmd.Flags().StringVar(&path, "path", "/webhook", "Webhook endpoint path")
	cmd.Flags().StringVarP(&tunnelType, "tunnel", "t", "", "Tunnel provider (cloudflared)")
	cmd.Flags().StringVarP(&webhookSecret, "secret", "s", "", "Webhook secret for signature verification")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output events as JSON")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress startup messages, only show events")

	return cmd
}

func runServer(port int, path, tunnelType, webhookSecret string, jsonOutput, quiet bool) error {
	// Create server config
	config := ports.WebhookServerConfig{
		Port:           port,
		Path:           path,
		WebhookSecret:  webhookSecret,
		TunnelProvider: tunnelType,
	}

	// Create webhook server
	server := webhookserver.NewServer(config)

	// Set up tunnel if requested
	if tunnelType != "" {
		switch strings.ToLower(tunnelType) {
		case "cloudflared", "cloudflare", "cf":
			if !tunnel.IsCloudflaredInstalled() {
				return common.NewUserError(
					"cloudflared is not installed",
					"Install it with: brew install cloudflared (macOS) or see https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/",
				)
			}
			localURL := fmt.Sprintf("http://localhost:%d", port)
			t := tunnel.NewCloudflaredTunnel(localURL)
			server.SetTunnel(t)
		default:
			return common.NewUserError(
				fmt.Sprintf("unsupported tunnel provider: %s", tunnelType),
				"Supported providers: cloudflared",
			)
		}
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Print startup message
	if !quiet {
		printStartupBanner()
	}

	// Start spinner while starting tunnel
	var spinner *common.Spinner
	if tunnelType != "" && !quiet {
		spinner = common.NewSpinner("Starting tunnel...")
		spinner.Start()
	}

	// Start the server
	if err := server.Start(ctx); err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		return common.WrapError(err)
	}

	if spinner != nil {
		spinner.Stop()
	}

	// Print server info
	stats := server.GetStats()
	if !quiet {
		printServerInfo(stats, tunnelType)
	}

	// Event display loop
	go func() {
		for event := range server.Events() {
			if jsonOutput {
				printEventJSON(event)
			} else {
				printEventFormatted(event, quiet)
			}
		}
	}()

	// Wait for interrupt
	<-sigChan

	if !quiet {
		fmt.Println("\n\nShutting down server...")
	}

	// Stop the server
	if err := server.Stop(); err != nil {
		return common.WrapError(err)
	}

	if !quiet {
		finalStats := server.GetStats()
		fmt.Printf("Server stopped. Total events received: %d\n", finalStats.EventsReceived)
	}

	return nil
}

func printStartupBanner() {
	fmt.Println()
	_, _ = common.Cyan.Println("╔══════════════════════════════════════════════════════════════╗")
	_, _ = common.Cyan.Print("║")
	fmt.Print("              ")
	_, _ = common.Bold.Print("Nylas Webhook Server")
	fmt.Print("                         ")
	_, _ = common.Cyan.Println("║")
	_, _ = common.Cyan.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func printServerInfo(stats ports.WebhookServerStats, tunnelType string) {
	_, _ = common.Green.Println("✓ Server started successfully")
	fmt.Println()

	_, _ = common.Bold.Print("  Local URL:    ")
	fmt.Println(stats.LocalURL)

	if stats.PublicURL != "" {
		_, _ = common.Bold.Print("  Public URL:   ")
		_, _ = common.Green.Println(stats.PublicURL)
		fmt.Println()
		_, _ = common.Bold.Print("  Tunnel:       ")
		fmt.Printf("%s (%s)\n", tunnelType, stats.TunnelStatus)
	}

	fmt.Println()
	_, _ = common.Yellow.Println("Register this URL with Nylas:")
	webhookURL := stats.LocalURL
	if stats.PublicURL != "" {
		webhookURL = stats.PublicURL
	}
	fmt.Printf("  nylas webhooks create --url %s --triggers message.created\n", webhookURL)
	fmt.Println()
	_, _ = common.Dim.Println("Press Ctrl+C to stop")
	fmt.Println()
	_, _ = common.Cyan.Println("─────────────────────────────────────────────────────────────────")
	_, _ = common.Bold.Println("Incoming Webhooks:")
	fmt.Println()
}

func printEventJSON(event *ports.WebhookEvent) {
	data, _ := json.Marshal(event)
	fmt.Println(string(data))
}

func printEventFormatted(event *ports.WebhookEvent, quiet bool) {
	timestamp := event.ReceivedAt.Format("15:04:05")

	// Determine verification status
	verifyIcon := ""
	if event.Signature != "" {
		if event.Verified {
			verifyIcon = common.Green.Sprint(" ✓")
		} else {
			verifyIcon = common.Red.Sprint(" ✗")
		}
	}

	// Event type coloring
	var typeColorFn func(a ...any) string
	switch {
	case strings.Contains(event.Type, "created"):
		typeColorFn = common.Green.Sprint
	case strings.Contains(event.Type, "deleted"):
		typeColorFn = common.Red.Sprint
	case strings.Contains(event.Type, "updated"):
		typeColorFn = common.Blue.Sprint
	default:
		typeColorFn = common.Yellow.Sprint
	}

	fmt.Printf("%s %s%s\n",
		common.Dim.Sprintf("[%s]", timestamp),
		typeColorFn(event.Type),
		verifyIcon,
	)

	if !quiet {
		// Print additional details
		if event.ID != "" {
			fmt.Printf("  %s %s\n", common.Dim.Sprint("ID:"), event.ID)
		}
		if event.GrantID != "" {
			fmt.Printf("  %s %s\n", common.Dim.Sprint("Grant:"), event.GrantID)
		}

		// Print a summary of the payload
		if event.Body != nil {
			if data, ok := event.Body["data"].(map[string]any); ok {
				if obj, ok := data["object"].(map[string]any); ok {
					// Print key fields based on event type
					if subject, ok := obj["subject"].(string); ok {
						fmt.Printf("  %s %s\n", common.Dim.Sprint("Subject:"), common.Truncate(subject, 60))
					}
					if title, ok := obj["title"].(string); ok {
						fmt.Printf("  %s %s\n", common.Dim.Sprint("Title:"), common.Truncate(title, 60))
					}
					if email, ok := obj["email"].(string); ok {
						fmt.Printf("  %s %s\n", common.Dim.Sprint("Email:"), email)
					}
				}
			}
		}
		fmt.Println()
	}
}
