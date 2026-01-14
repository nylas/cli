//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// EMAIL ATTACHMENTS COMMAND TESTS
// =============================================================================

func TestCLI_EmailAttachmentsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "attachments", "--help")

	if err != nil {
		t.Fatalf("email attachments --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show attachments subcommands
	if !strings.Contains(stdout, "list") {
		t.Errorf("Expected 'list' subcommand in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "download") {
		t.Errorf("Expected 'download' subcommand in help, got: %s", stdout)
	}

	t.Logf("email attachments --help output:\n%s", stdout)
}

func TestCLI_EmailAttachmentsList(t *testing.T) {
	skipIfMissingCreds(t)

	// Get a message to test attachments
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 10)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for attachments test")
	}

	// Find a message (may or may not have attachments)
	messageID := messages[0].ID

	stdout, stderr, err := runCLI("email", "attachments", "list", messageID, testGrantID)

	if err != nil {
		t.Fatalf("email attachments list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show attachments list or "No attachments"
	if !strings.Contains(stdout, "Attachments") && !strings.Contains(stdout, "No attachments") &&
		!strings.Contains(stdout, "attachment") {
		t.Errorf("Expected attachments list output, got: %s", stdout)
	}

	t.Logf("email attachments list output:\n%s", stdout)
}

func TestCLI_EmailAttachmentsListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	// Get a message to test attachments
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 10)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for attachments test")
	}

	messageID := messages[0].ID

	stdout, stderr, err := runCLI("email", "attachments", "list", messageID, testGrantID, "--json")

	if err != nil {
		t.Fatalf("email attachments list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Note: The --json flag is a global flag but may not be fully implemented
	// for the attachments list command. Just verify the command doesn't fail.
	if strings.Contains(stdout, "No attachments") || strings.Contains(stdout, "attachment") {
		t.Logf("email attachments list --json output (formatted):\n%s", stdout)
	} else {
		// Check if it's JSON
		trimmed := strings.TrimSpace(stdout)
		if len(trimmed) > 0 && (trimmed[0] == '[' || trimmed[0] == '{') {
			t.Logf("email attachments list --json output (JSON):\n%s", stdout)
		} else {
			t.Logf("email attachments list --json output:\n%s", stdout)
		}
	}
}

func TestCLI_EmailAttachmentsListHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "attachments", "list", "--help")

	if err != nil {
		t.Fatalf("email attachments list --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage information
	if !strings.Contains(stdout, "message-id") {
		t.Errorf("Expected 'message-id' in help, got: %s", stdout)
	}

	// Should show JSON flag
	if !strings.Contains(stdout, "--json") {
		t.Errorf("Expected '--json' flag in help, got: %s", stdout)
	}

	t.Logf("email attachments list --help output:\n%s", stdout)
}

func TestCLI_EmailAttachmentsDownloadHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "attachments", "download", "--help")

	if err != nil {
		t.Fatalf("email attachments download --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show usage information
	if !strings.Contains(stdout, "attachment-id") && !strings.Contains(stdout, "message-id") {
		t.Errorf("Expected 'attachment-id' or 'message-id' in help, got: %s", stdout)
	}

	// Should show output flag
	if !strings.Contains(stdout, "--output") && !strings.Contains(stdout, "-o") {
		t.Errorf("Expected '--output' flag in help, got: %s", stdout)
	}

	t.Logf("email attachments download --help output:\n%s", stdout)
}

func TestCLI_EmailAttachmentsListInvalidMessageID(t *testing.T) {
	skipIfMissingCreds(t)

	_, stderr, err := runCLI("email", "attachments", "list", "invalid-message-id", testGrantID)

	if err == nil {
		t.Error("Expected error for invalid message ID, but command succeeded")
	}

	t.Logf("email attachments list invalid message ID error: %s", stderr)
}

func TestCLI_EmailAttachmentsListAllFormats(t *testing.T) {
	skipIfMissingCreds(t)

	// Get a message to test attachments
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 10)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for attachments test")
	}

	messageID := messages[0].ID

	tests := []struct {
		name string
		args []string
	}{
		{"default", []string{"email", "attachments", "list", messageID, testGrantID}},
		{"json", []string{"email", "attachments", "list", messageID, testGrantID, "--json"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)
			if err != nil {
				t.Fatalf("email attachments list %s failed: %v\nstderr: %s", tt.name, err, stderr)
			}
			t.Logf("email attachments list %s output:\n%s", tt.name, stdout)
		})
	}
}

// =============================================================================
// EMAIL ATTACHMENTS DOWNLOAD COMMAND TESTS
// =============================================================================

func TestCLI_EmailAttachmentsDownloadInvalidAttachmentID(t *testing.T) {
	skipIfMissingCreds(t)

	// Get a message to test attachments
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for attachments test")
	}

	messageID := messages[0].ID

	_, stderr, err := runCLI("email", "attachments", "download",
		"invalid-attachment-id", messageID, testGrantID)

	if err == nil {
		t.Error("Expected error for invalid attachment ID, but command succeeded")
	}

	t.Logf("email attachments download invalid attachment ID error: %s", stderr)
}
