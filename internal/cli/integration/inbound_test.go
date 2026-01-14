//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// INBOUND LIST COMMAND TESTS
// =============================================================================

func TestCLI_InboundList(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("inbound", "list")

	if err != nil {
		// Skip if inbound is not enabled for this account
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "unauthorized") {
			t.Skip("Inbound not enabled for this account")
		}
		t.Fatalf("inbound list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show inboxes or "No inboxes found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No inbound inboxes found") && !strings.Contains(stdout, "nylas.email") {
		t.Errorf("Expected inbox list output, got: %s", stdout)
	}

	t.Logf("inbound list output:\n%s", stdout)
}

func TestCLI_InboundList_JSON(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("inbound", "list", "--json")

	if err != nil {
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "unauthorized") {
			t.Skip("Inbound not enabled for this account")
		}
		t.Fatalf("inbound list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON output
	if !strings.HasPrefix(strings.TrimSpace(stdout), "[") && !strings.HasPrefix(strings.TrimSpace(stdout), "null") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("inbound list --json output:\n%s", stdout)
}

func TestCLI_InboundList_InboxAlias(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// Test the 'inbox' alias for 'inbound' command
	stdout, stderr, err := runCLI("inbox", "list")

	if err != nil {
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "unauthorized") {
			t.Skip("Inbound not enabled for this account")
		}
		t.Fatalf("inbox list (alias) failed: %v\nstderr: %s", err, stderr)
	}

	// Should show same output as 'inbound list'
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No inbound inboxes found") && !strings.Contains(stdout, "nylas.email") {
		t.Errorf("Expected inbox list output, got: %s", stdout)
	}

	t.Logf("inbox list (alias) output:\n%s", stdout)
}

// =============================================================================
// INBOUND SHOW COMMAND TESTS
// =============================================================================

func TestCLI_InboundShow(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// First get an inbox ID
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inboxes, err := client.ListInboundInboxes(ctx)
	if err != nil {
		t.Skipf("Failed to list inboxes: %v", err)
	}
	if len(inboxes) == 0 {
		t.Skip("No inbound inboxes available for show test")
	}

	inboxID := inboxes[0].ID

	stdout, stderr, err := runCLI("inbound", "show", inboxID)

	if err != nil {
		t.Fatalf("inbound show failed: %v\nstderr: %s", err, stderr)
	}

	// Should show inbox details
	if !strings.Contains(stdout, "ID:") && !strings.Contains(stdout, "Email:") {
		t.Errorf("Expected inbox details in output, got: %s", stdout)
	}

	t.Logf("inbound show output:\n%s", stdout)
}

func TestCLI_InboundShow_JSON(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// First get an inbox ID
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inboxes, err := client.ListInboundInboxes(ctx)
	if err != nil {
		t.Skipf("Failed to list inboxes: %v", err)
	}
	if len(inboxes) == 0 {
		t.Skip("No inbound inboxes available for show test")
	}

	inboxID := inboxes[0].ID

	stdout, stderr, err := runCLI("inbound", "show", inboxID, "--json")

	if err != nil {
		t.Fatalf("inbound show --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON with expected fields
	if !strings.Contains(stdout, `"id":`) {
		t.Errorf("Expected '\"id\":' in JSON output, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"email":`) {
		t.Errorf("Expected '\"email\":' in JSON output, got: %s", stdout)
	}

	t.Logf("inbound show --json output:\n%s", stdout)
}

func TestCLI_InboundShow_InvalidID(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	_, stderr, err := runCLI("inbound", "show", "invalid-inbox-id")

	if err == nil {
		t.Error("Expected error for invalid inbox ID, but command succeeded")
	}

	t.Logf("inbound show invalid ID error: %s", stderr)
}

// =============================================================================
// INBOUND HELP TESTS
// =============================================================================

func TestCLI_InboundHelp(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("inbound", "--help")

	if err != nil {
		t.Fatalf("inbound --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show inbound subcommands
	expectedCommands := []string{"list", "show", "create", "delete", "messages", "monitor"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("Expected '%s' in inbound help, got: %s", cmd, stdout)
		}
	}

	t.Logf("inbound help output:\n%s", stdout)
}

func TestCLI_InboundListHelp(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("inbound", "list", "--help")

	if err != nil {
		t.Fatalf("inbound list --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "--json") {
		t.Errorf("Expected '--json' flag in help, got: %s", stdout)
	}

	t.Logf("inbound list help output:\n%s", stdout)
}

func TestCLI_InboundMessagesHelp(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("inbound", "messages", "--help")

	if err != nil {
		t.Fatalf("inbound messages --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "--limit") {
		t.Errorf("Expected '--limit' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--unread") {
		t.Errorf("Expected '--unread' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--json") {
		t.Errorf("Expected '--json' flag in help, got: %s", stdout)
	}

	t.Logf("inbound messages help output:\n%s", stdout)
}

func TestCLI_InboundMonitorHelp(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("inbound", "monitor", "--help")

	if err != nil {
		t.Fatalf("inbound monitor --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "--port") {
		t.Errorf("Expected '--port' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--tunnel") {
		t.Errorf("Expected '--tunnel' flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "cloudflared") {
		t.Errorf("Expected 'cloudflared' in help, got: %s", stdout)
	}

	t.Logf("inbound monitor help output:\n%s", stdout)
}

// =============================================================================
// ENVIRONMENT VARIABLE TESTS
// =============================================================================

func TestCLI_InboundWithEnvVar(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// First get an inbox ID
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inboxes, err := client.ListInboundInboxes(ctx)
	if err != nil {
		t.Skipf("Failed to list inboxes: %v", err)
	}
	if len(inboxes) == 0 {
		t.Skip("No inbound inboxes available")
	}

	inboxID := inboxes[0].ID

	// Run with NYLAS_INBOUND_GRANT_ID set
	stdout, stderr, err := runCLIWithEnv(
		map[string]string{"NYLAS_INBOUND_GRANT_ID": inboxID},
		"inbound", "messages", "--limit", "2",
	)

	if err != nil {
		t.Fatalf("inbound messages with env var failed: %v\nstderr: %s", err, stderr)
	}

	t.Logf("inbound messages with env var output:\n%s", stdout)
}

// runCLIWithEnv executes a CLI command with additional environment variables
func runCLIWithEnv(env map[string]string, args ...string) (string, string, error) {
	return runCLIWithEnvImpl(env, args...)
}

func runCLIWithEnvImpl(env map[string]string, args ...string) (string, string, error) {
	// This is a simplified implementation that sets env vars before running
	// For full implementation, we'd need to modify runCLI to accept env vars
	for k, v := range env {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}
	return runCLI(args...)
}
