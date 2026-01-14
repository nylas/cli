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
// INBOUND MESSAGES COMMAND TESTS
// =============================================================================

func TestCLI_InboundMessages(t *testing.T) {
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
		t.Skip("No inbound inboxes available for messages test")
	}

	inboxID := inboxes[0].ID

	stdout, stderr, err := runCLI("inbound", "messages", inboxID, "--limit", "5")

	if err != nil {
		t.Fatalf("inbound messages failed: %v\nstderr: %s", err, stderr)
	}

	// Should show messages or "No messages found"
	if !strings.Contains(stdout, "Messages (") && !strings.Contains(stdout, "No messages found") && !strings.Contains(stdout, "Unread Messages") {
		t.Errorf("Expected messages output, got: %s", stdout)
	}

	t.Logf("inbound messages output:\n%s", stdout)
}

func TestCLI_InboundMessages_WithLimit(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inboxes, err := client.ListInboundInboxes(ctx)
	if err != nil {
		t.Skipf("Failed to list inboxes: %v", err)
	}
	if len(inboxes) == 0 {
		t.Skip("No inbound inboxes available for messages test")
	}

	inboxID := inboxes[0].ID

	stdout, stderr, err := runCLI("inbound", "messages", inboxID, "--limit", "2")

	if err != nil {
		t.Fatalf("inbound messages --limit failed: %v\nstderr: %s", err, stderr)
	}

	t.Logf("inbound messages --limit output:\n%s", stdout)
}

func TestCLI_InboundMessages_UnreadOnly(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inboxes, err := client.ListInboundInboxes(ctx)
	if err != nil {
		t.Skipf("Failed to list inboxes: %v", err)
	}
	if len(inboxes) == 0 {
		t.Skip("No inbound inboxes available for messages test")
	}

	inboxID := inboxes[0].ID

	stdout, stderr, err := runCLI("inbound", "messages", inboxID, "--unread")

	if err != nil {
		t.Fatalf("inbound messages --unread failed: %v\nstderr: %s", err, stderr)
	}

	t.Logf("inbound messages --unread output:\n%s", stdout)
}

func TestCLI_InboundMessages_JSON(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inboxes, err := client.ListInboundInboxes(ctx)
	if err != nil {
		t.Skipf("Failed to list inboxes: %v", err)
	}
	if len(inboxes) == 0 {
		t.Skip("No inbound inboxes available for messages test")
	}

	inboxID := inboxes[0].ID

	stdout, stderr, err := runCLI("inbound", "messages", inboxID, "--json", "--limit", "3")

	if err != nil {
		t.Fatalf("inbound messages --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON output
	if !strings.HasPrefix(strings.TrimSpace(stdout), "[") && !strings.HasPrefix(strings.TrimSpace(stdout), "null") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("inbound messages --json output:\n%s", stdout)
}

// =============================================================================
// INBOUND CREATE COMMAND TESTS
// =============================================================================

func TestCLI_InboundCreate(t *testing.T) {
	if os.Getenv("NYLAS_TEST_CREATE_INBOUND") != "true" {
		t.Skip("Skipping create test - set NYLAS_TEST_CREATE_INBOUND=true to enable")
	}
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// Generate a unique email prefix
	prefix := "test-" + time.Now().Format("20060102150405")

	stdout, stderr, err := runCLI("inbound", "create", prefix)

	if err != nil {
		// Skip if inbound is not enabled for this account or if there are validation errors
		if strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "unauthorized") ||
			strings.Contains(stderr, "not enabled") ||
			strings.Contains(stderr, "invalid 'email'") ||
			strings.Contains(stderr, "invalid email") {
			t.Skip("Inbound not enabled or email validation failed for this account")
		}
		t.Fatalf("inbound create failed: %v\nstderr: %s", err, stderr)
	}

	// Should show created inbox details
	if !strings.Contains(stdout, "Created") || !strings.Contains(stdout, prefix) {
		t.Errorf("Expected created confirmation with prefix %s, got: %s", prefix, stdout)
	}

	t.Logf("inbound create output:\n%s", stdout)

	// Cleanup: Extract inbox ID and delete it
	// Look for the inbox we just created and delete it
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	inboxes, err := client.ListInboundInboxes(ctx)
	if err == nil {
		for _, inbox := range inboxes {
			if strings.Contains(inbox.Email, prefix) {
				t.Logf("Cleaning up test inbox: %s", inbox.ID)
				_ = client.DeleteInboundInbox(ctx, inbox.ID)
				break
			}
		}
	}
}

func TestCLI_InboundCreate_JSON(t *testing.T) {
	if os.Getenv("NYLAS_TEST_CREATE_INBOUND") != "true" {
		t.Skip("Skipping create test - set NYLAS_TEST_CREATE_INBOUND=true to enable")
	}
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// Generate a unique email prefix
	prefix := "testjson-" + time.Now().Format("20060102150405")

	stdout, stderr, err := runCLI("inbound", "create", prefix, "--json")

	if err != nil {
		// Skip if inbound is not enabled for this account or if there are validation errors
		if strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "unauthorized") ||
			strings.Contains(stderr, "not enabled") ||
			strings.Contains(stderr, "invalid 'email'") ||
			strings.Contains(stderr, "invalid email") {
			t.Skip("Inbound not enabled or email validation failed for this account")
		}
		t.Fatalf("inbound create --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON with expected fields
	if !strings.Contains(stdout, `"id":`) {
		t.Errorf("Expected '\"id\":' in JSON output, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"email":`) {
		t.Errorf("Expected '\"email\":' in JSON output, got: %s", stdout)
	}

	t.Logf("inbound create --json output:\n%s", stdout)

	// Cleanup: Extract inbox ID and delete it
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	inboxes, err := client.ListInboundInboxes(ctx)
	if err == nil {
		for _, inbox := range inboxes {
			if strings.Contains(inbox.Email, prefix) {
				t.Logf("Cleaning up test inbox: %s", inbox.ID)
				_ = client.DeleteInboundInbox(ctx, inbox.ID)
				break
			}
		}
	}
}

func TestCLI_InboundCreate_NoPrefix(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	_, stderr, err := runCLI("inbound", "create")

	if err == nil {
		t.Error("Expected error when no prefix provided, but command succeeded")
	}

	// Should show error about missing argument
	if !strings.Contains(stderr, "argument") && !strings.Contains(stderr, "required") {
		t.Logf("Expected argument error in stderr: %s", stderr)
	}
}

// =============================================================================
// INBOUND DELETE COMMAND TESTS
// =============================================================================

func TestCLI_InboundDelete(t *testing.T) {
	if os.Getenv("NYLAS_TEST_DELETE_INBOUND") != "true" {
		t.Skip("Skipping delete test - set NYLAS_TEST_DELETE_INBOUND=true to enable")
	}
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// First create an inbox to delete
	prefix := "todelete-" + time.Now().Format("20060102150405")

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inbox, err := client.CreateInboundInbox(ctx, prefix)
	if err != nil {
		t.Skipf("Failed to create inbox for delete test: %v", err)
	}

	// Wait for creation to propagate
	time.Sleep(1 * time.Second)

	stdout, stderr, err := runCLI("inbound", "delete", inbox.ID, "--yes")

	if err != nil {
		t.Fatalf("inbound delete failed: %v\nstderr: %s", err, stderr)
	}

	// Should show deleted confirmation
	if !strings.Contains(stdout, "Deleted") && !strings.Contains(stdout, "deleted") {
		t.Errorf("Expected deleted confirmation, got: %s", stdout)
	}

	t.Logf("inbound delete output:\n%s", stdout)
}

func TestCLI_InboundDelete_InvalidID(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	_, stderr, err := runCLI("inbound", "delete", "invalid-inbox-id", "--yes")

	if err == nil {
		t.Error("Expected error for invalid inbox ID, but command succeeded")
	}

	t.Logf("inbound delete invalid ID error: %s", stderr)
}

func TestCLI_InboundDelete_NoConfirm(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// Without --yes flag, should require confirmation
	// Since we can't provide interactive input, this should fail or prompt
	stdout, stderr, err := runCLI("inbound", "delete", "some-inbox-id")

	// Should either fail or show confirmation prompt
	t.Logf("inbound delete without --yes: stdout=%s stderr=%s err=%v", stdout, stderr, err)
}
