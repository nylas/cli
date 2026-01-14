//go:build integration

package integration

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/cli/common"
)

// =============================================================================
// WEBHOOK SERVER TESTS
// =============================================================================

func TestCLI_WebhookServerHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "server", "--help")

	if err != nil {
		t.Fatalf("webhook server --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show server options
	if !strings.Contains(stdout, "--port") {
		t.Errorf("Expected --port flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--tunnel") {
		t.Errorf("Expected --tunnel flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "cloudflared") {
		t.Errorf("Expected cloudflared mentioned in help, got: %s", stdout)
	}

	t.Logf("webhook server --help output:\n%s", stdout)
}

func TestCLI_WebhookServerSubcommandExists(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Verify server is listed in webhook subcommands
	stdout, stderr, err := runCLI("webhook", "--help")

	if err != nil {
		t.Fatalf("webhook --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "server") {
		t.Errorf("Expected 'server' subcommand in webhook help, got: %s", stdout)
	}

	t.Logf("webhook --help output:\n%s", stdout)
}

func TestCLI_WebhookLifecycle(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	// Start webhook server with cloudflared tunnel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the webhook server in the background
	serverCmd := exec.CommandContext(ctx, testBinary, "webhook", "server", "--tunnel", "cloudflared", "--port", "3099")
	serverCmd.Env = os.Environ()

	stdout, err := serverCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start webhook server: %v", err)
	}

	// Ensure cleanup
	defer func() {
		cancel()
		_ = serverCmd.Wait()
	}()

	// Wait for tunnel URL to appear in output
	var webhookURL string
	scanner := bufio.NewScanner(stdout)
	timeout := time.After(60 * time.Second)

	for webhookURL == "" {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for cloudflared tunnel URL")
		default:
			if scanner.Scan() {
				line := scanner.Text()
				t.Logf("Server output: %s", line)
				// Look for Public URL line
				if strings.Contains(line, "Public URL:") {
					parts := strings.Split(line, "Public URL:")
					if len(parts) > 1 {
						webhookURL = strings.TrimSpace(parts[1])
						t.Logf("Found webhook URL: %s", webhookURL)
					}
				}
			}
		}
	}

	if webhookURL == "" {
		t.Fatal("Failed to get webhook URL from server output")
	}

	// Give the tunnel a moment to stabilize
	time.Sleep(5 * time.Second)

	webhookDesc := fmt.Sprintf("CLI Test Webhook %d", time.Now().Unix())
	var webhookID string

	// Create webhook with retry (cloudflare tunnels may need time to become reachable)
	t.Run("create", func(t *testing.T) {
		var stdout, stderr string

		// Retry config: 5s base delay, 2x multiplier = 5s, 10s delays (matches original)
		retryConfig := common.RetryConfig{
			MaxRetries:  2, // 3 total attempts
			BaseDelay:   5 * time.Second,
			MaxDelay:    30 * time.Second,
			Multiplier:  2.0,
			JitterRatio: 0,
		}

		ctx := context.Background()
		err := common.WithRetry(ctx, retryConfig, func() error {
			var cmdErr error
			stdout, stderr, cmdErr = runCLI("webhook", "create",
				"--url", webhookURL,
				"--triggers", "message.created",
				"--description", webhookDesc)

			if cmdErr != nil {
				t.Logf("Attempt failed: %v, retrying...", cmdErr)
				return cmdErr
			}
			if !strings.Contains(stdout, "created") {
				t.Logf("Output missing 'created', retrying...")
				return errors.New("webhook not yet created")
			}
			return nil
		})

		if err != nil {
			t.Fatalf("webhook create failed after retries: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "created") {
			t.Errorf("Expected 'created' in output, got: %s", stdout)
		}

		// Extract webhook ID from output
		if idx := strings.Index(stdout, "ID:"); idx != -1 {
			webhookID = strings.TrimSpace(stdout[idx+3:])
			if newline := strings.Index(webhookID, "\n"); newline != -1 {
				webhookID = webhookID[:newline]
			}
		}

		t.Logf("webhook create output: %s", stdout)
		t.Logf("Webhook ID: %s", webhookID)
	})

	if webhookID == "" {
		t.Fatal("Failed to get webhook ID from create output")
	}

	// Wait for webhook to be created
	time.Sleep(2 * time.Second)

	// Show webhook
	t.Run("show", func(t *testing.T) {
		stdout, stderr, err := runCLI("webhook", "show", webhookID)
		if err != nil {
			t.Fatalf("webhook show failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, webhookID) {
			t.Errorf("Expected webhook ID in output, got: %s", stdout)
		}

		t.Logf("webhook show output:\n%s", stdout)
	})

	// Update webhook
	t.Run("update", func(t *testing.T) {
		newDesc := "Updated " + webhookDesc
		stdout, stderr, err := runCLI("webhook", "update", webhookID,
			"--description", newDesc)
		if err != nil {
			t.Fatalf("webhook update failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "updated") {
			t.Errorf("Expected 'updated' in output, got: %s", stdout)
		}

		t.Logf("webhook update output:\n%s", stdout)
	})

	// Delete webhook
	t.Run("delete", func(t *testing.T) {
		stdout, stderr, err := runCLI("webhook", "delete", webhookID, "--force")
		if err != nil {
			t.Fatalf("webhook delete failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "deleted") {
			t.Errorf("Expected 'deleted' in output, got: %s", stdout)
		}

		t.Logf("webhook delete output: %s", stdout)
	})
}
