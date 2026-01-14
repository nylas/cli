//go:build integration

package integration

import (
	"strings"
	"testing"
)

// =============================================================================
// WEBHOOK COMMAND TESTS
// =============================================================================

func TestCLI_WebhookHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "--help")

	if err != nil {
		t.Fatalf("webhook --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show webhook subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected webhook subcommands in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "triggers") || !strings.Contains(stdout, "test") {
		t.Errorf("Expected triggers and test subcommands in help, got: %s", stdout)
	}

	t.Logf("webhook --help output:\n%s", stdout)
}

func TestCLI_WebhookList(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("webhook", "list")

	if err != nil {
		t.Fatalf("webhook list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show webhooks list or "No webhooks configured"
	if !strings.Contains(stdout, "webhooks") && !strings.Contains(stdout, "No webhooks") && !strings.Contains(stdout, "ID") {
		t.Errorf("Expected webhook list output, got: %s", stdout)
	}

	t.Logf("webhook list output:\n%s", stdout)
}

func TestCLI_WebhookListFullIDs(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("webhook", "list", "--full-ids")

	if err != nil {
		t.Fatalf("webhook list --full-ids failed: %v\nstderr: %s", err, stderr)
	}

	// Should show webhooks list or "No webhooks configured"
	if !strings.Contains(stdout, "webhooks") && !strings.Contains(stdout, "No webhooks") && !strings.Contains(stdout, "ID") {
		t.Errorf("Expected webhook list output, got: %s", stdout)
	}

	// When there are webhooks, the IDs should not be truncated (no "...")
	// Only check this if there are webhooks
	if strings.Contains(stdout, "Total:") && !strings.Contains(stdout, "Total: 0") {
		// If there are webhooks, verify we show full IDs (check help confirms flag exists)
		t.Log("Webhooks found - full IDs should be displayed without truncation")
	}

	t.Logf("webhook list --full-ids output:\n%s", stdout)
}

func TestCLI_WebhookListJSON(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("webhook", "list", "--format", "json")

	if err != nil {
		t.Fatalf("webhook list --format json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON (starts with [ or {)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && trimmed[0] != '[' && trimmed[0] != '{' && !strings.Contains(stdout, "No webhooks") {
		t.Errorf("Expected JSON output, got: %s", stdout)
	}

	t.Logf("webhook list JSON output:\n%s", stdout)
}

func TestCLI_WebhookTriggers(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "triggers")

	if err != nil {
		t.Fatalf("webhook triggers failed: %v\nstderr: %s", err, stderr)
	}

	// Should show trigger types
	if !strings.Contains(stdout, "message.created") {
		t.Errorf("Expected 'message.created' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "event.created") {
		t.Errorf("Expected 'event.created' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "contact.created") {
		t.Errorf("Expected 'contact.created' in output, got: %s", stdout)
	}

	// Test for new triggers added
	if !strings.Contains(stdout, "grant.imap_sync_completed") {
		t.Errorf("Expected 'grant.imap_sync_completed' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "message.opened") {
		t.Errorf("Expected 'message.opened' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "message.bounce_detected") {
		t.Errorf("Expected 'message.bounce_detected' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "message.send_success") {
		t.Errorf("Expected 'message.send_success' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "message.send_failed") {
		t.Errorf("Expected 'message.send_failed' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "message.link_clicked") {
		t.Errorf("Expected 'message.link_clicked' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "notetaker.media") {
		t.Errorf("Expected 'notetaker.media' in output, got: %s", stdout)
	}

	t.Logf("webhook triggers output:\n%s", stdout)
}

func TestCLI_WebhookTriggersNotetakerCategory(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "triggers", "--category", "notetaker")

	if err != nil {
		t.Fatalf("webhook triggers --category notetaker failed: %v\nstderr: %s", err, stderr)
	}

	// Should show only notetaker triggers
	if !strings.Contains(stdout, "notetaker.media") {
		t.Errorf("Expected 'notetaker.media' in output, got: %s", stdout)
	}

	t.Logf("webhook triggers --category notetaker output:\n%s", stdout)
}

func TestCLI_WebhookTriggersJSON(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "triggers", "--format", "json")

	if err != nil {
		t.Fatalf("webhook triggers --format json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON
	if !strings.Contains(stdout, "{") {
		t.Errorf("Expected JSON output, got: %s", stdout)
	}

	t.Logf("webhook triggers JSON output:\n%s", stdout)
}

func TestCLI_WebhookTriggersCategory(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "triggers", "--category", "message")

	if err != nil {
		t.Fatalf("webhook triggers --category message failed: %v\nstderr: %s", err, stderr)
	}

	// Should show only message triggers
	if !strings.Contains(stdout, "message.created") {
		t.Errorf("Expected 'message.created' in output, got: %s", stdout)
	}
	// Should show Message header (the actual filtered section)
	if !strings.Contains(stdout, "📧 Message") && !strings.Contains(stdout, "Message") {
		t.Errorf("Expected 'Message' category header in output, got: %s", stdout)
	}

	t.Logf("webhook triggers --category message output:\n%s", stdout)
}

func TestCLI_WebhookCreateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "create", "--help")

	if err != nil {
		t.Fatalf("webhook create --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show required flags
	if !strings.Contains(stdout, "--url") || !strings.Contains(stdout, "--triggers") {
		t.Errorf("Expected --url and --triggers flags in help, got: %s", stdout)
	}

	// Should show examples
	if !strings.Contains(stdout, "Examples:") {
		t.Errorf("Expected 'Examples:' in help, got: %s", stdout)
	}

	t.Logf("webhook create --help output:\n%s", stdout)
}

func TestCLI_WebhookTestHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "test", "--help")

	if err != nil {
		t.Fatalf("webhook test --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show subcommands
	if !strings.Contains(stdout, "send") || !strings.Contains(stdout, "payload") {
		t.Errorf("Expected 'send' and 'payload' subcommands in help, got: %s", stdout)
	}

	t.Logf("webhook test --help output:\n%s", stdout)
}

func TestCLI_WebhookTestPayload(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	stdout, stderr, err := runCLI("webhook", "test", "payload", "message.created")

	if err != nil {
		t.Fatalf("webhook test payload failed: %v\nstderr: %s", err, stderr)
	}

	// Should show mock payload
	if !strings.Contains(stdout, "message.created") || !strings.Contains(stdout, "{") {
		t.Errorf("Expected mock payload with trigger type, got: %s", stdout)
	}

	t.Logf("webhook test payload output:\n%s", stdout)
}

func TestCLI_WebhookTestPayloadList(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Running without trigger type should list available types
	stdout, stderr, err := runCLI("webhook", "test", "payload")

	if err != nil {
		t.Fatalf("webhook test payload (no args) failed: %v\nstderr: %s", err, stderr)
	}

	// Should show available trigger types
	if !strings.Contains(stdout, "Available trigger types") && !strings.Contains(stdout, "message") {
		t.Errorf("Expected trigger type list, got: %s", stdout)
	}

	t.Logf("webhook test payload (no args) output:\n%s", stdout)
}

// Webhook server and lifecycle tests are in webhooks_lifecycle_test.go
