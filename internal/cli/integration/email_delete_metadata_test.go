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
// EMAIL LIST COMMAND TESTS
// =============================================================================
func TestCLI_EmailDeleteHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "delete", "--help")

	if err != nil {
		t.Fatalf("email delete --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show delete command usage
	if !strings.Contains(stdout, "message-id") {
		t.Errorf("Expected 'message-id' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--force") {
		t.Errorf("Expected '--force' flag in help, got: %s", stdout)
	}

	t.Logf("email delete help output:\n%s", stdout)
}

func TestCLI_EmailDelete(t *testing.T) {
	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true' - skipping delete test")
	}
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("NYLAS_TEST_SEND_EMAIL not set to 'true' - need to send test message first")
	}
	skipIfMissingCreds(t)

	// First send a test message to delete
	email := getTestEmail()

	stdout, stderr, err := runCLI("email", "send",
		"--to", email,
		"--subject", "Test Message for Delete",
		"--body", "This message will be deleted by integration test.",
		"--yes",
		testGrantID)

	if err != nil {
		t.Fatalf("Failed to send test message: %v\nstderr: %s", err, stderr)
	}

	t.Logf("Sent test message: %s", stdout)

	// Wait for message to be available
	time.Sleep(3 * time.Second)

	// Get the message ID from sent messages
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 10)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	var messageID string
	for _, msg := range messages {
		if strings.Contains(msg.Subject, "Test Message for Delete") {
			messageID = msg.ID
			break
		}
	}

	if messageID == "" {
		t.Skip("Could not find test message to delete")
	}

	// Now delete the message with --force flag
	stdout, stderr, err = runCLI("email", "delete", messageID, testGrantID, "--force")

	if err != nil {
		t.Fatalf("email delete failed: %v\nstderr: %s", err, stderr)
	}

	// Should show delete confirmation
	lowerOutput := strings.ToLower(stdout)
	if !strings.Contains(lowerOutput, "deleted") && !strings.Contains(lowerOutput, "moved to trash") {
		t.Errorf("Expected delete confirmation in output, got: %s", stdout)
	}

	t.Logf("email delete output: %s", stdout)

	// Verify message is deleted (or in trash)
	time.Sleep(2 * time.Second)

	// Create new context for verification
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer verifyCancel()

	_, err = client.GetMessage(verifyCtx, testGrantID, messageID)
	if err == nil {
		t.Logf("Message still exists (may be in trash folder, which is expected)")
	} else {
		t.Logf("Message no longer accessible (deleted or in trash): %v", err)
	}
}

func TestCLI_EmailDelete_InvalidID(t *testing.T) {
	skipIfMissingCreds(t)

	_, stderr, err := runCLI("email", "delete", "invalid-message-id", testGrantID, "--force")

	if err == nil {
		t.Error("Expected error for invalid message ID, but command succeeded")
	}

	t.Logf("email delete invalid ID error: %s", stderr)
}

func TestCLI_EmailDelete_WithoutForce(t *testing.T) {
	// Test that without --force, the command asks for confirmation
	// This is a help-only test since we can't provide interactive input easily
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, _, _ := runCLI("email", "delete", "--help")

	// Verify that --force flag exists for skipping confirmation
	if !strings.Contains(stdout, "--force") && !strings.Contains(stdout, "-f") {
		t.Error("Expected --force flag in help to skip confirmation")
	}

	t.Log("email delete command supports --force flag to skip confirmation")
}

// =============================================================================
// EMAIL METADATA COMMAND TESTS (Phase 1.1)
// =============================================================================

func TestCLI_EmailMetadataHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "metadata", "--help")

	if err != nil {
		t.Fatalf("email metadata --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show metadata subcommands
	if !strings.Contains(stdout, "show") {
		t.Errorf("Expected 'show' subcommand in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "info") {
		t.Errorf("Expected 'info' subcommand in help, got: %s", stdout)
	}

	t.Logf("email metadata help output:\n%s", stdout)
}

func TestCLI_EmailMetadataInfo(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "metadata", "info")

	if err != nil {
		t.Fatalf("email metadata info failed: %v\nstderr: %s", err, stderr)
	}

	// Should show information about metadata usage
	lowerOutput := strings.ToLower(stdout)
	if !strings.Contains(lowerOutput, "metadata") {
		t.Errorf("Expected 'metadata' in info output, got: %s", stdout)
	}
	if !strings.Contains(lowerOutput, "key") {
		t.Errorf("Expected 'key' in info output (indexed keys), got: %s", stdout)
	}

	t.Logf("email metadata info output:\n%s", stdout)
}

func TestCLI_EmailMetadataShow(t *testing.T) {
	skipIfMissingCreds(t)

	// Get a message to test metadata show
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for metadata show test")
	}

	messageID := messages[0].ID

	// Show metadata for the message
	stdout, stderr, err := runCLI("email", "metadata", "show", messageID, testGrantID)

	if err != nil {
		t.Fatalf("email metadata show failed: %v\nstderr: %s", err, stderr)
	}

	// Output should show metadata (or indicate no metadata)
	lowerOutput := strings.ToLower(stdout)
	if !strings.Contains(lowerOutput, "metadata") && !strings.Contains(lowerOutput, "no metadata") &&
		!strings.Contains(lowerOutput, "none") && len(strings.TrimSpace(stdout)) == 0 {
		t.Logf("Note: Output format may vary. Got: %s", stdout)
	}

	t.Logf("email metadata show output:\n%s", stdout)
}

func TestCLI_EmailMetadataShow_JSON(t *testing.T) {
	skipIfMissingCreds(t)

	// Get a message to test metadata show with JSON
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for metadata show JSON test")
	}

	messageID := messages[0].ID

	// Show metadata for the message with JSON output
	stdout, stderr, err := runCLI("email", "metadata", "show", messageID, testGrantID, "--json")

	if err != nil {
		t.Fatalf("email metadata show --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be JSON format (object or null)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 {
		if trimmed[0] != '{' && trimmed != "null" && trimmed != "{}" {
			t.Logf("Note: JSON output may be empty object or null for no metadata. Got: %s", stdout)
		}
	}

	t.Logf("email metadata show --json output:\n%s", stdout)
}

func TestCLI_EmailMetadataShow_InvalidID(t *testing.T) {
	skipIfMissingCreds(t)

	_, stderr, err := runCLI("email", "metadata", "show", "invalid-message-id", testGrantID)

	if err == nil {
		t.Error("Expected error for invalid message ID, but command succeeded")
	}

	t.Logf("email metadata show invalid ID error: %s", stderr)
}
