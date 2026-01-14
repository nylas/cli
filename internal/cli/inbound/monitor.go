package inbound

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

func newMonitorCmd() *cobra.Command {
	var (
		port          int
		tunnelType    string
		webhookSecret string
		jsonOutput    bool
		quiet         bool
	)

	cmd := &cobra.Command{
		Use:   "monitor [inbox-id]",
		Short: "Monitor an inbound inbox for new messages in real-time",
		Long: `Monitor an inbound inbox for new messages via webhooks.

This starts a local webhook server to receive real-time notifications
when new emails arrive at your inbound inbox. The server can optionally
be exposed via a tunnel for receiving webhooks from the internet.

Examples:
  # Start monitoring with default settings
  nylas inbound monitor abc123

  # Monitor with cloudflared tunnel (for public access)
  nylas inbound monitor abc123 --tunnel cloudflared

  # Monitor on custom port
  nylas inbound monitor abc123 --port 8080

  # Output events as JSON
  nylas inbound monitor abc123 --tunnel cloudflared --json

  # Use environment variable for inbox ID
  export NYLAS_INBOUND_GRANT_ID=abc123
  nylas inbound monitor --tunnel cloudflared

Press Ctrl+C to stop monitoring.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitor(args, port, tunnelType, webhookSecret, jsonOutput, quiet)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to listen on")
	cmd.Flags().StringVarP(&tunnelType, "tunnel", "t", "", "Tunnel provider (cloudflared)")
	cmd.Flags().StringVarP(&webhookSecret, "secret", "s", "", "Webhook secret for signature verification")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output events as JSON")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress startup messages, only show events")

	return cmd
}

func runMonitor(args []string, port int, tunnelType, webhookSecret string, jsonOutput, quiet bool) error {
	inboxID, err := getInboxID(args)
	if err != nil {
		printError("%v", err)
		return err
	}

	// Get inbox details
	client, err := getClient()
	if err != nil {
		printError("%v", err)
		return err
	}

	ctx, cancel := common.CreateContext()
	inbox, err := client.GetInboundInbox(ctx, inboxID)
	cancel()
	if err != nil {
		printError("Failed to get inbox: %v", err)
		return err
	}

	// Create server config
	config := ports.WebhookServerConfig{
		Port:           port,
		Path:           "/webhook",
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
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Print startup banner
	if !quiet {
		printMonitorBanner(inbox.Email)
	}

	// Start spinner while starting tunnel
	var spinner *common.Spinner
	if tunnelType != "" && !quiet {
		spinner = common.NewSpinner("Starting tunnel...")
		spinner.Start()
	}

	// Start the server
	if err := server.Start(serverCtx); err != nil {
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
		printMonitorInfo(stats, tunnelType, inbox.Email)
	}

	// Event display loop - filter for inbound events
	go func() {
		for event := range server.Events() {
			// Filter for message.created events from inbox source
			if event.Type == "message.created" && event.Source == "inbox" {
				// Only show events for our inbox
				if event.GrantID == "" || event.GrantID == inboxID {
					if jsonOutput {
						printEventJSON(event)
					} else {
						printInboundEvent(event, quiet)
					}
				}
			} else if strings.HasPrefix(event.Type, "message.") {
				// Show all message events if no source filter
				if jsonOutput {
					printEventJSON(event)
				} else {
					printInboundEvent(event, quiet)
				}
			}
		}
	}()

	// Wait for interrupt
	<-sigChan

	if !quiet {
		fmt.Println("\n\nStopping monitor...")
	}

	// Stop the server
	if err := server.Stop(); err != nil {
		return common.WrapError(err)
	}

	if !quiet {
		finalStats := server.GetStats()
		fmt.Printf("Monitor stopped. Total events received: %d\n", finalStats.EventsReceived)
	}

	return nil
}

func printMonitorBanner(email string) {
	fmt.Println()
	_, _ = common.Cyan.Println("╔══════════════════════════════════════════════════════════════╗")
	_, _ = common.Cyan.Print("║")
	fmt.Print("            ")
	_, _ = common.BoldWhite.Print("Nylas Inbound Monitor")
	fmt.Print("                        ")
	_, _ = common.Cyan.Println("║")
	_, _ = common.Cyan.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Monitoring: %s\n", common.Cyan.Sprint(email))
	fmt.Println()
}

func printMonitorInfo(stats ports.WebhookServerStats, tunnelType, _ string) {
	_, _ = common.Green.Println("Monitor started successfully!")
	fmt.Println()

	_, _ = common.BoldWhite.Print("  Local URL:    ")
	fmt.Println(stats.LocalURL)

	if stats.PublicURL != "" {
		_, _ = common.BoldWhite.Print("  Public URL:   ")
		_, _ = common.Green.Println(stats.PublicURL)
		fmt.Println()
		_, _ = common.BoldWhite.Print("  Tunnel:       ")
		fmt.Printf("%s (%s)\n", tunnelType, stats.TunnelStatus)
	}

	fmt.Println()
	_, _ = common.Yellow.Println("To receive events, register this webhook URL with Nylas:")
	webhookURL := stats.LocalURL
	if stats.PublicURL != "" {
		webhookURL = stats.PublicURL
	}
	fmt.Printf("  nylas webhooks create --url %s --triggers message.created\n", webhookURL)
	fmt.Println()
	_, _ = common.Dim.Println("Press Ctrl+C to stop")
	fmt.Println()
	_, _ = common.Cyan.Println("─────────────────────────────────────────────────────────────────")
	_, _ = common.BoldWhite.Println("Incoming Messages:")
	fmt.Println()
}

func printEventJSON(event *ports.WebhookEvent) {
	data, _ := json.Marshal(event)
	fmt.Println(string(data))
}

func printInboundEvent(event *ports.WebhookEvent, quiet bool) {
	timestamp := event.ReceivedAt.Format("15:04:05")

	// Determine verification status
	verifyIcon := ""
	if event.Signature != "" {
		if event.Verified {
			verifyIcon = common.Green.Sprint(" [verified]")
		} else {
			verifyIcon = common.Red.Sprint(" [unverified]")
		}
	}

	// Event type coloring
	var typeStr string
	switch event.Type {
	case "message.created":
		typeStr = common.Green.Sprint("NEW MESSAGE")
	case "message.updated":
		typeStr = common.Cyan.Sprint("UPDATED")
	case "message.opened":
		typeStr = common.Yellow.Sprint("OPENED")
	default:
		typeStr = common.Dim.Sprint(event.Type)
	}

	fmt.Printf("%s %s%s\n",
		common.Dim.Sprintf("[%s]", timestamp),
		typeStr,
		verifyIcon,
	)

	if !quiet {
		// Print message details
		if event.Body != nil {
			if data, ok := event.Body["data"].(map[string]any); ok {
				if obj, ok := data["object"].(map[string]any); ok {
					// Print subject
					if subject, ok := obj["subject"].(string); ok {
						fmt.Printf("  %s %s\n", common.Dim.Sprint("Subject:"), common.Truncate(subject, 60))
					}
					// Print from
					if from, ok := obj["from"].([]any); ok && len(from) > 0 {
						if fromObj, ok := from[0].(map[string]any); ok {
							email := ""
							name := ""
							if e, ok := fromObj["email"].(string); ok {
								email = e
							}
							if n, ok := fromObj["name"].(string); ok {
								name = n
							}
							if name != "" {
								fmt.Printf("  %s %s <%s>\n", common.Dim.Sprint("From:"), name, email)
							} else {
								fmt.Printf("  %s %s\n", common.Dim.Sprint("From:"), email)
							}
						}
					}
					// Print snippet
					if snippet, ok := obj["snippet"].(string); ok {
						fmt.Printf("  %s %s\n", common.Dim.Sprint("Preview:"), common.Truncate(snippet, 60))
					}
					// Print message ID
					if id, ok := obj["id"].(string); ok {
						_, _ = common.Dim.Printf("  ID: %s\n", id)
					}
				}
			}
		}
		fmt.Println()
	}
}
