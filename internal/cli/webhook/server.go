package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/nylas/cli/internal/adapters/tunnel"
	"github.com/nylas/cli/internal/adapters/webhookserver"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
)

const defaultSignedWebhookMaxEventAge = 5 * time.Minute

func newServerCmd() *cobra.Command {
	var (
		port          int
		path          string
		tunnelType    string
		webhookSecret string
		allowUnsigned bool
		noTunnel      bool
		jsonOutput    bool
		quiet         bool
	)

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start a local webhook receiver server",
		Long: `Start a local HTTP server to receive and display webhook events.

The server can optionally expose itself via a tunnel (cloudflared) for
receiving webhooks from the internet when developing locally.

When --tunnel is set, --secret is required so the server can verify the
HMAC signature on each incoming event. Pass --allow-unsigned to opt out
explicitly (events from anyone who can reach the public tunnel URL will
be processed).

If neither --tunnel nor --no-tunnel is set, the command runs an
interactive preflight that detects cloudflared and offers to enable it.
Pass --no-tunnel to skip the preflight and run loopback-only (useful
when driving the server from local tooling such as curl).

Examples:
  # Start server with interactive tunnel preflight
  nylas webhooks server

  # Start loopback-only server (no prompt; localhost only)
  nylas webhooks server --no-tunnel

  # Start server with cloudflared tunnel + signature verification
  nylas webhooks server --tunnel cloudflared --secret your-webhook-secret

  # Start server with tunnel and explicitly accept unsigned events
  nylas webhooks server --tunnel cloudflared --allow-unsigned

Press Ctrl+C to stop the server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(port, path, tunnelType, webhookSecret, allowUnsigned, noTunnel, jsonOutput, quiet)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to listen on")
	cmd.Flags().StringVar(&path, "path", "/webhook", "Webhook endpoint path")
	cmd.Flags().StringVarP(&tunnelType, "tunnel", "t", "", "Tunnel provider (cloudflared)")
	cmd.Flags().StringVarP(&webhookSecret, "secret", "s", "", "Webhook secret for signature verification")
	cmd.Flags().BoolVar(&allowUnsigned, "allow-unsigned", false, "Allow unsigned webhook events when --tunnel is set (insecure)")
	cmd.Flags().BoolVar(&noTunnel, "no-tunnel", false, "Skip the tunnel preflight prompt and run loopback-only")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output events as JSON")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress startup messages, only show events")

	return cmd
}

func runServer(port int, path, tunnelType, webhookSecret string, allowUnsigned, noTunnel, jsonOutput, quiet bool) error {
	// --tunnel and --no-tunnel are mutually exclusive: the user can't both
	// request a tunnel and opt out of one in the same invocation.
	if tunnelType != "" && noTunnel {
		return common.NewUserError(
			"--tunnel and --no-tunnel cannot be combined",
			"Choose one: --tunnel <provider> to expose publicly, or --no-tunnel to run loopback-only.",
		)
	}

	// Interactive preflight when neither --tunnel nor --no-tunnel was set.
	// May modify tunnelType/webhookSecret/allowUnsigned, or signal that the
	// user wants to exit (e.g. cloudflared not installed and they declined
	// loopback-only).
	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	resolvedTunnel, resolvedSecret, resolvedAllowUnsigned, exit, err := preflightTunnelChoice(
		newStdinPrompter(),
		interactive,
		tunnelType, noTunnel, quiet, jsonOutput, webhookSecret, allowUnsigned,
	)
	if err != nil {
		return err
	}
	if exit {
		return nil
	}
	tunnelType = resolvedTunnel
	webhookSecret = resolvedSecret
	allowUnsigned = resolvedAllowUnsigned

	// When exposing the server via a tunnel, refuse to start without a
	// secret unless the user explicitly opted into accepting unsigned
	// events. Otherwise anyone who can reach the public tunnel URL can
	// inject forged webhook events.
	if tunnelType != "" && webhookSecret == "" && !allowUnsigned {
		return common.NewUserError(
			"--secret is required when --tunnel is set",
			"Pass --secret <value> to verify the HMAC signature on each event, "+
				"or pass --allow-unsigned to accept unverified events (insecure).",
		)
	}

	config := newWebhookServerConfig(port, path, tunnelType, webhookSecret)

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

	// Event display loop. Recover from any panic in the formatters so a
	// malformed event body cannot take down the CLI; exit cleanly when the
	// events channel closes (server.Stop) or the parent context cancels.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "warn: event display recovered from panic: %v\n", r)
			}
		}()
		events := server.Events()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if jsonOutput {
					printEventJSON(event)
				} else {
					printEventFormatted(event, quiet)
				}
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

func newWebhookServerConfig(port int, path, tunnelType, webhookSecret string) ports.WebhookServerConfig {
	config := ports.WebhookServerConfig{
		Port:           port,
		Path:           path,
		WebhookSecret:  webhookSecret,
		TunnelProvider: tunnelType,
	}
	if webhookSecret != "" {
		config.MaxEventAge = defaultSignedWebhookMaxEventAge
	}
	return config
}

// Test seams: package vars so unit tests can override host-state probes
// without a real cloudflared binary or brew. Production calls go through
// the real adapters by default.
var (
	cloudflaredInstalled = tunnel.IsCloudflaredInstalled
	cloudflaredViaBrew   = canInstallCloudflaredViaBrew
	installCloudflaredFn = installCloudflaredViaBrew
)

// preflightTunnelChoice walks the user through enabling a cloudflared tunnel
// when no explicit --tunnel/--no-tunnel choice was made on an interactive
// terminal. The returned (tunnelType, secret, allowUnsigned) reflect the
// resolved values; exit=true means the caller should return nil without
// starting the server (the user declined to continue or cancelled a prompt).
//
// The preflight is intentionally skipped in scripted modes (--quiet, --json,
// interactive=false, --no-tunnel set, --tunnel already set) so that
// automation keeps the previous behaviour. `interactive` is passed in by the
// caller — runServer resolves it from term.IsTerminal(os.Stdin.Fd()), tests
// pass true — so the function itself stays testable without a real TTY.
//
// Any prompt error (including io.EOF from Ctrl-D) aborts the preflight by
// returning exit=true. We deliberately do NOT default to the safer-looking
// answer on error: cancellation must never silently flip into
// --allow-unsigned or auto-run brew, both of which weaken security or change
// system state without explicit consent.
func preflightTunnelChoice(
	prompter preflightPrompter,
	interactive bool,
	tunnelType string,
	noTunnel, quiet, jsonOutput bool,
	secret string,
	allowUnsigned bool,
) (string, string, bool, bool, error) {
	if tunnelType != "" || noTunnel || quiet || jsonOutput {
		return tunnelType, secret, allowUnsigned, false, nil
	}
	if !interactive {
		return tunnelType, secret, allowUnsigned, false, nil
	}

	if !cloudflaredInstalled() {
		_, _ = common.Yellow.Println("⚠ cloudflared is not installed.")
		fmt.Println("  Nylas delivers webhooks to public URLs, so a loopback-only")
		fmt.Println("  server cannot receive events from Nylas.")
		fmt.Println()

		installed := false
		if cloudflaredViaBrew() {
			confirmInstall, err := prompter.Confirm("Install cloudflared via brew now?", true)
			if err != nil {
				// Cancelled before deciding — abort cleanly. Notably, do NOT
				// fall back to running brew on EOF: invoking a state-changing
				// system command requires explicit consent.
				return tunnelType, secret, allowUnsigned, true, nil
			}
			if confirmInstall {
				if err := installCloudflaredFn(); err != nil {
					_, _ = common.Red.Printf("  brew install cloudflared failed: %v\n", err)
					fmt.Println()
				} else if cloudflaredInstalled() {
					_, _ = common.Green.Println("✓ cloudflared installed.")
					fmt.Println()
					installed = true
				}
			}
		}

		if !installed {
			// User declined the install, install failed, or this host has no
			// brew. Fall back to manual instructions and offer loopback-only.
			fmt.Println("  Install cloudflared manually:")
			if runtime.GOOS == "darwin" {
				fmt.Println("    brew install cloudflared")
			}
			fmt.Println("    https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/")
			fmt.Println()
			cont, err := prompter.Confirm("Continue with a loopback-only server (useful for local curl tests)?", false)
			if err != nil || !cont {
				return tunnelType, secret, allowUnsigned, true, nil
			}
			return tunnelType, secret, allowUnsigned, false, nil
		}
		// Cloudflared was just installed — fall through to the tunnel-enable prompt.
	}

	enableTunnel, err := prompter.Confirm("Enable cloudflared tunnel so Nylas can reach this server?", true)
	if err != nil {
		// Cancellation. Returning exit=true rather than silently starting a
		// tunnel — the user typed Ctrl-D, not "yes".
		return tunnelType, secret, allowUnsigned, true, nil
	}
	if !enableTunnel {
		return tunnelType, secret, allowUnsigned, false, nil
	}

	// User opted into the tunnel. Make sure we have a signing posture before
	// we expose a public URL. The secret is read with terminal echo disabled
	// so it never lands in shell history or scrollback. An empty input is
	// the user's explicit answer that they want unsigned mode; an EOF/error
	// is cancellation and must NOT flip them into --allow-unsigned.
	if secret == "" && !allowUnsigned {
		entered, err := prompter.Password("Webhook secret for HMAC verification (leave empty to allow unsigned events): ")
		if err != nil {
			return tunnelType, secret, allowUnsigned, true, nil
		}
		if entered == "" {
			// Confirm the insecure choice with a separate explicit prompt so
			// it can't be reached by hammering Enter through the secret prompt.
			confirmUnsigned, err := prompter.Confirm("No secret entered. Accept unsigned events on the public tunnel? (insecure)", false)
			if err != nil || !confirmUnsigned {
				return tunnelType, secret, allowUnsigned, true, nil
			}
			allowUnsigned = true
			_, _ = common.Yellow.Println("  ⚠ Continuing without signature verification (--allow-unsigned).")
		} else {
			secret = entered
		}
	}
	return "cloudflared", secret, allowUnsigned, false, nil
}

// canInstallCloudflaredViaBrew reports whether the host environment can
// auto-install cloudflared via Homebrew. Restricted to macOS where brew is
// the de-facto package manager and the cloudflared formula is well-supported.
func canInstallCloudflaredViaBrew() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("brew")
	return err == nil
}

// installCloudflaredViaBrew runs `brew install cloudflared`, streaming brew's
// stdout/stderr/stdin to the user's terminal so progress and any prompts (e.g.
// for sudo) flow through naturally. Arguments are static literals — no user
// input crosses the exec boundary.
func installCloudflaredViaBrew() error {
	// #nosec G204 -- command "brew" and args ("install", "cloudflared") are
	// compile-time string literals, so there is no user-controlled input that
	// could influence which binary runs or which package is installed. The
	// caller has already gated this behind explicit user consent via the
	// preflight prompt.
	cmd := exec.Command("brew", "install", "cloudflared")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
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
	if stats.PublicURL != "" {
		_, _ = common.Yellow.Println("Register this URL with Nylas:")
		fmt.Printf("  nylas webhooks create --url %s --triggers message.created\n", stats.PublicURL)
	} else {
		_, _ = common.Yellow.Println("⚠ Loopback-only server")
		fmt.Println("  Nylas cannot deliver webhooks to localhost. To expose this server")
		fmt.Println("  publicly, re-run with:")
		fmt.Println("    nylas webhooks server --tunnel cloudflared --secret <your-secret>")
	}
	fmt.Println()
	_, _ = common.Dim.Println("Press Ctrl+C to stop")
	fmt.Println()
	_, _ = common.Cyan.Println("─────────────────────────────────────────────────────────────────")
	_, _ = common.Bold.Println("Incoming Webhooks:")
	fmt.Println()
}

func printEventJSON(event *ports.WebhookEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: failed to marshal event: %v\n", err)
		return
	}
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
