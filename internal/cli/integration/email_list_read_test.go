//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// EMAIL LIST COMMAND TESTS
// =============================================================================

func TestCLI_EmailList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "list", testGrantID, "--limit", "5")

	if err != nil {
		t.Fatalf("email list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show message count or "No messages found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No messages found") {
		t.Errorf("Expected message list output, got: %s", stdout)
	}

	t.Logf("email list output:\n%s", stdout)
}

func TestCLI_EmailList_WithID(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "list", testGrantID, "--limit", "3", "--id")

	if err != nil {
		t.Fatalf("email list --id failed: %v\nstderr: %s", err, stderr)
	}

	// Should show "ID:" lines when --id flag is used
	if strings.Contains(stdout, "Found") && !strings.Contains(stdout, "ID:") {
		t.Errorf("Expected message IDs in output with --id flag, got: %s", stdout)
	}

	t.Logf("email list --id output:\n%s", stdout)
}

func TestCLI_EmailList_Filters(t *testing.T) {
	skipIfMissingCreds(t)

	tests := []struct {
		name string
		args []string
	}{
		{"unread", []string{"email", "list", testGrantID, "--unread", "--limit", "3"}},
		{"starred", []string{"email", "list", testGrantID, "--starred", "--limit", "3"}},
		{"limit", []string{"email", "list", testGrantID, "--limit", "1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)
			if err != nil {
				t.Fatalf("email list %s failed: %v\nstderr: %s", tt.name, err, stderr)
			}
			t.Logf("email list %s output:\n%s", tt.name, stdout)
		})
	}
}

// =============================================================================
// EMAIL READ COMMAND TESTS
// =============================================================================

func TestCLI_EmailRead(t *testing.T) {
	skipIfMissingCreds(t)

	// First get a message ID
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for read test")
	}

	messageID := messages[0].ID

	stdout, stderr, err := runCLI("email", "read", messageID, testGrantID)

	if err != nil {
		t.Fatalf("email read failed: %v\nstderr: %s", err, stderr)
	}

	// Should show message details
	if !strings.Contains(stdout, "Subject:") {
		t.Errorf("Expected 'Subject:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "From:") {
		t.Errorf("Expected 'From:' in output, got: %s", stdout)
	}

	t.Logf("email read output:\n%s", stdout)
}

func TestCLI_EmailShow(t *testing.T) {
	skipIfMissingCreds(t)

	// Test the 'show' alias for 'read' command
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for show test")
	}

	messageID := messages[0].ID

	// Use 'show' alias instead of 'read'
	stdout, stderr, err := runCLI("email", "show", messageID, testGrantID)

	if err != nil {
		t.Fatalf("email show (alias) failed: %v\nstderr: %s", err, stderr)
	}

	// Should show message details (same output as 'read')
	if !strings.Contains(stdout, "Subject:") {
		t.Errorf("Expected 'Subject:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "From:") {
		t.Errorf("Expected 'From:' in output, got: %s", stdout)
	}

	t.Logf("email show (alias) output:\n%s", stdout)
}

func TestCLI_EmailRead_JSON(t *testing.T) {
	skipIfMissingCreds(t)

	// First get a message ID
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for read test")
	}

	messageID := messages[0].ID

	stdout, stderr, err := runCLI("email", "read", messageID, testGrantID, "--json")

	if err != nil {
		t.Fatalf("email read --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON with expected fields
	if !strings.Contains(stdout, `"id":`) {
		t.Errorf("Expected '\"id\":' in JSON output, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"subject":`) {
		t.Errorf("Expected '\"subject\":' in JSON output, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"from":`) {
		t.Errorf("Expected '\"from\":' in JSON output, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"body":`) {
		t.Errorf("Expected '\"body\":' in JSON output, got: %s", stdout)
	}

	// Should NOT contain formatted headers (means it's JSON not formatted)
	if strings.Contains(stdout, "Subject:") && strings.Contains(stdout, "────") {
		t.Errorf("JSON output should not contain formatted headers")
	}

	t.Logf("email read --json output:\n%s", stdout)
}

func TestCLI_EmailRead_Raw(t *testing.T) {
	skipIfMissingCreds(t)

	// First get a message ID
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for read test")
	}

	messageID := messages[0].ID

	stdout, stderr, err := runCLI("email", "read", messageID, testGrantID, "--raw")

	if err != nil {
		t.Fatalf("email read --raw failed: %v\nstderr: %s", err, stderr)
	}

	// Should show message headers
	if !strings.Contains(stdout, "Subject:") {
		t.Errorf("Expected 'Subject:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "ID:") {
		t.Errorf("Expected 'ID:' in raw output (shows message ID), got: %s", stdout)
	}

	// Raw output typically contains HTML tags if the message is HTML
	// OR it's plain text - either way it should have body content
	t.Logf("email read --raw output:\n%s", stdout)
}

// =============================================================================
// EMAIL SEARCH COMMAND TESTS
// =============================================================================

func TestCLI_EmailSearch(t *testing.T) {
	skipIfMissingCreds(t)

	// Search for a common subject
	stdout, stderr, err := runCLI("email", "search", "test", testGrantID, "--limit", "5")

	if err != nil {
		t.Fatalf("email search failed: %v\nstderr: %s", err, stderr)
	}

	// Should show results or "No messages found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No messages found") {
		t.Errorf("Expected search results output, got: %s", stdout)
	}

	t.Logf("email search output:\n%s", stdout)
}

func TestCLI_EmailSearch_WithFilters(t *testing.T) {
	skipIfMissingCreds(t)

	// Search with date filter
	stdout, stderr, err := runCLI("email", "search", "email", testGrantID,
		"--limit", "3",
		"--after", "2024-01-01")

	if err != nil {
		t.Fatalf("email search with filters failed: %v\nstderr: %s", err, stderr)
	}

	t.Logf("email search with filters output:\n%s", stdout)
}

// =============================================================================
// EMAIL MARK COMMAND TESTS
// =============================================================================

func TestCLI_EmailMark(t *testing.T) {
	skipIfMissingCreds(t)

	// Get a message to test marking
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for mark test")
	}

	messageID := messages[0].ID

	tests := []struct {
		name     string
		action   string
		expected string
	}{
		{"starred", "starred", "starred"},
		{"unstarred", "unstarred", "removed"},
		{"unread", "unread", "unread"},
		{"read", "read", "read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI("email", "mark", tt.action, messageID, testGrantID)
			if err != nil {
				t.Fatalf("email mark %s failed: %v\nstderr: %s", tt.action, err, stderr)
			}

			if !strings.Contains(strings.ToLower(stdout), tt.expected) {
				t.Errorf("Expected '%s' in output, got: %s", tt.expected, stdout)
			}

			t.Logf("email mark %s output: %s", tt.action, stdout)

			// Small delay between operations
			time.Sleep(500 * time.Millisecond)
		})
	}
}

// =============================================================================
// EMAIL SEND COMMAND TESTS
// =============================================================================
