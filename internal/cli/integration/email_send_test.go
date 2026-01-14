//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"
)

// =============================================================================
// EMAIL LIST COMMAND TESTS
// =============================================================================

func TestCLI_EmailSend(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	skipIfMissingCreds(t)

	email := testEmail
	if email == "" {
		email = "test@example.com"
	}

	stdout, stderr, err := runCLI("email", "send",
		"--to", email,
		"--subject", "CLI Integration Test",
		"--body", "This is a test email from the CLI integration tests.",
		"--yes",
		testGrantID)

	if err != nil {
		t.Fatalf("email send failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "sent") && !strings.Contains(stdout, "Message") && !strings.Contains(stdout, "✓") {
		t.Errorf("Expected send confirmation in output, got: %s", stdout)
	}

	t.Logf("email send output:\n%s", stdout)
}

func TestCLI_EmailHelp(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "--help")

	if err != nil {
		t.Fatalf("email --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show email subcommands
	if !strings.Contains(stdout, "list") {
		t.Errorf("Expected 'list' in email help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "read") {
		t.Errorf("Expected 'read' in email help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "send") {
		t.Errorf("Expected 'send' in email help, got: %s", stdout)
	}

	t.Logf("email help output:\n%s", stdout)
}

func TestCLI_EmailRead_InvalidID(t *testing.T) {
	skipIfMissingCreds(t)

	_, stderr, err := runCLI("email", "read", "invalid-message-id", testGrantID)

	if err == nil {
		t.Error("Expected error for invalid message ID, but command succeeded")
	}

	t.Logf("email read invalid ID error: %s", stderr)
}

func TestCLI_EmailList_InvalidGrantID(t *testing.T) {
	skipIfMissingCreds(t)

	_, stderr, err := runCLI("email", "list", "invalid-grant-id", "--limit", "1")

	if err == nil {
		t.Error("Expected error for invalid grant ID, but command succeeded")
	}

	t.Logf("email list invalid grant error: %s", stderr)
}

// =============================================================================
// EMAIL LIST ALL COMMAND TESTS
// =============================================================================

func TestCLI_EmailList_All(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "list", "all", "--limit", "5")

	if err != nil {
		// Skip if auth fails for "all" command (requires different auth setup)
		if strings.Contains(stderr, "Bearer token invalid") || strings.Contains(stderr, "unauthorized") {
			t.Skip("email list all requires different auth setup")
		}
		t.Fatalf("email list all failed: %v\nstderr: %s", err, stderr)
	}

	// Should show message count or "No messages found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No messages found") {
		t.Errorf("Expected message list output, got: %s", stdout)
	}

	t.Logf("email list all output:\n%s", stdout)
}

func TestCLI_EmailList_AllWithID(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "list", "all", "--limit", "3", "--id")

	if err != nil {
		// Skip if auth fails for "all" command (requires different auth setup)
		if strings.Contains(stderr, "Bearer token invalid") || strings.Contains(stderr, "unauthorized") {
			t.Skip("email list all requires different auth setup")
		}
		t.Fatalf("email list all --id failed: %v\nstderr: %s", err, stderr)
	}

	// Should show "ID:" lines when --id flag is used (if messages exist)
	if strings.Contains(stdout, "Found") && !strings.Contains(stdout, "ID:") {
		t.Errorf("Expected message IDs in output with --id flag, got: %s", stdout)
	}

	t.Logf("email list all --id output:\n%s", stdout)
}

func TestCLI_EmailList_AllHelp(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "list", "all", "--help")

	if err != nil {
		t.Fatalf("email list all --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show help for the all subcommand
	if !strings.Contains(stdout, "all") {
		t.Errorf("Expected 'all' in help output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--limit") {
		t.Errorf("Expected '--limit' flag in help output, got: %s", stdout)
	}

	t.Logf("email list all help output:\n%s", stdout)
}

// =============================================================================
// SCHEDULED SEND TESTS
// =============================================================================

func TestCLI_EmailSendHelp_Schedule(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "send", "--help")

	if err != nil {
		t.Fatalf("email send --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show schedule options in help
	if !strings.Contains(stdout, "--schedule") {
		t.Errorf("Expected '--schedule' in send help, got: %s", stdout)
	}

	t.Logf("email send help output:\n%s", stdout)
}

func TestCLI_EmailSend_ScheduleFlag(t *testing.T) {
	// Test that schedule flag is recognized (without actually sending)
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Just verify the flag is accepted by checking help
	stdout, stderr, err := runCLI("email", "send", "--help")

	if err != nil {
		t.Fatalf("email send --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show schedule flag with duration examples
	if !strings.Contains(stdout, "2h") && !strings.Contains(stdout, "tomorrow") {
		t.Errorf("Expected schedule duration examples in help, got: %s", stdout)
	}

	t.Logf("email send help shows schedule options")
}

func TestCLI_EmailSend_Scheduled(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	skipIfMissingCreds(t)

	email := testEmail
	if email == "" {
		email = "test@example.com"
	}

	// Schedule for 1 hour from now using duration format
	stdout, stderr, err := runCLI("email", "send",
		"--to", email,
		"--subject", "Scheduled Email Test",
		"--body", "This is a scheduled test email from CLI integration tests.",
		"--schedule", "1h",
		testGrantID)

	if err != nil {
		t.Fatalf("email send scheduled failed: %v\nstderr: %s", err, stderr)
	}

	// Should show scheduled confirmation
	if !strings.Contains(stdout, "scheduled") && !strings.Contains(stdout, "Scheduled") && !strings.Contains(stdout, "Message") {
		t.Errorf("Expected scheduled confirmation in output, got: %s", stdout)
	}

	t.Logf("email send scheduled output:\n%s", stdout)
}

// =============================================================================
// ADVANCED SEARCH COMMAND TESTS (Phase 3)
// =============================================================================

func TestCLI_EmailSearchHelp_AdvancedFlags(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("email", "search", "--help")

	if err != nil {
		t.Fatalf("email search --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show advanced search flags
	if !strings.Contains(stdout, "--unread") {
		t.Errorf("Expected '--unread' flag in search help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--starred") {
		t.Errorf("Expected '--starred' flag in search help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--in") {
		t.Errorf("Expected '--in' flag in search help, got: %s", stdout)
	}

	t.Logf("email search help output:\n%s", stdout)
}

func TestCLI_EmailSearch_AdvancedFilters(t *testing.T) {
	skipIfMissingCreds(t)

	tests := []struct {
		name string
		args []string
	}{
		{"unread", []string{"email", "search", "test", testGrantID, "--unread", "--limit", "3"}},
		{"starred", []string{"email", "search", "test", testGrantID, "--starred", "--limit", "3"}},
		{"folder", []string{"email", "search", "test", testGrantID, "--in", "INBOX", "--limit", "3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)
			if err != nil {
				t.Fatalf("email search %s failed: %v\nstderr: %s", tt.name, err, stderr)
			}
			t.Logf("email search %s output:\n%s", tt.name, stdout)
		})
	}
}

// =============================================================================
// THREAD SEARCH COMMAND TESTS (Phase 3)
// =============================================================================
