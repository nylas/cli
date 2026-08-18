//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/cli/common"
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

func TestCLI_EmailSubscriptionsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLIWithRateLimit(t, "email", "subscriptions", "list", "--limit", "20", "--since", "30d", "--all-folders", "--json")
	if err != nil {
		t.Fatalf("email subscriptions list failed: %v\nstderr: %s", err, stderr)
	}

	var subscriptions []map[string]any
	if err := json.Unmarshal([]byte(stdout), &subscriptions); err != nil {
		t.Fatalf("email subscriptions list returned invalid JSON: %v\nstdout: %s", err, stdout)
	}
	for _, subscription := range subscriptions {
		email, _ := subscription["email"].(string)
		listID, _ := subscription["list_id"].(string)
		if email == "" && listID == "" {
			t.Fatalf("subscription must have an email or list ID: %#v", subscription)
		}
		if messages, ok := subscription["messages"].(float64); !ok || messages < 1 {
			t.Fatalf("subscription must have a positive message count: %#v", subscription)
		}
		switch subscription["method"] {
		case "Web", "Email", "Unsupported":
		default:
			t.Fatalf("subscription has invalid method: %#v", subscription)
		}
		lastSeen, ok := subscription["last_seen"].(string)
		if !ok {
			t.Fatalf("subscription must have last_seen: %#v", subscription)
		}
		if _, err := time.Parse(time.RFC3339, lastSeen); err != nil {
			t.Fatalf("subscription last_seen is not RFC3339: %q", lastSeen)
		}
		for key := range subscription {
			if strings.Contains(strings.ToLower(key), "unsubscribe") || strings.Contains(strings.ToLower(key), "url") {
				t.Fatalf("subscription output exposes an action target in field %q", key)
			}
		}
	}
}

func TestCLI_EmailSubscriptionsList_Table(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLIWithRateLimit(t, "email", "subscriptions", "list", "--limit", "20", "--since", "30d", "--all-folders")
	if err != nil {
		t.Fatalf("email subscriptions table failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "SENDER") && !strings.Contains(stdout, "No subscriptions found") {
		t.Fatalf("email subscriptions table returned unexpected output: %s", stdout)
	}
}

func TestCLI_EmailSubscriptionsList_RejectsInvalidInput(t *testing.T) {
	skipIfMissingCreds(t)

	for _, args := range [][]string{
		{"email", "subscriptions", "list", "--limit", "0"},
		{"email", "subscriptions", "list", "--since", "0d"},
		{"email", "subscriptions", "list", testGrantID},
	} {
		if _, _, err := runCLI(args...); err == nil {
			t.Fatalf("email subscriptions list unexpectedly accepted args: %v", args)
		}
	}
}

func TestCLI_EmailSubscriptionsUnsubscribe_DryRun(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLIWithRateLimit(t, "email", "subscriptions", "list", "--limit", "200", "--since", "90d", "--all-folders", "--json")
	if err != nil {
		t.Fatalf("email subscriptions list for dry run failed: %v\nstderr: %s", err, stderr)
	}
	var subscriptions []struct {
		Email  string `json:"email"`
		ListID string `json:"list_id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal([]byte(stdout), &subscriptions); err != nil {
		t.Fatalf("email subscriptions list returned invalid JSON: %v", err)
	}

	selector := ""
	for _, subscription := range subscriptions {
		if subscription.Method == "Unsupported" {
			continue
		}
		selector = subscription.Email
		if selector == "" {
			selector = subscription.ListID
		}
		if selector != "" {
			break
		}
	}
	if selector == "" {
		t.Skip("no actionable subscriptions found in the integration account")
	}

	stdout, stderr, err = runCLIWithRateLimit(t, "email", "subscriptions", "unsubscribe", selector, "--limit", "200", "--since", "90d", "--all-folders", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("email subscriptions unsubscribe dry run failed: %v\nstderr: %s", err, stderr)
	}
	var results []map[string]any
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		t.Fatalf("email subscriptions unsubscribe dry run returned invalid JSON: %v\nstdout: %s", err, stdout)
	}
	if len(results) == 0 {
		t.Fatal("email subscriptions unsubscribe dry run returned no results")
	}
	foundPlan := false
	for _, result := range results {
		if status, _ := result["status"].(string); strings.Contains(status, "Would") || strings.Contains(status, "would") {
			foundPlan = true
		}
		for key := range result {
			if strings.Contains(strings.ToLower(key), "unsubscribe") || strings.Contains(strings.ToLower(key), "url") {
				t.Fatalf("unsubscribe dry-run output exposes an action target in field %q", key)
			}
		}
	}
	if !foundPlan {
		t.Fatalf("unsubscribe dry run did not describe a planned action: %#v", results)
	}

	stdout, stderr, err = runCLIWithRateLimit(t, "email", "subscriptions", "cleanup", selector, "--limit", "200", "--since", "90d", "--all-folders", "--permanent", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("email subscriptions cleanup dry run failed: %v\nstderr: %s", err, stderr)
	}
	results = nil
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		t.Fatalf("email subscriptions cleanup dry run returned invalid JSON: %v\nstdout: %s", err, stdout)
	}
	if len(results) == 0 {
		t.Fatal("email subscriptions cleanup dry run returned no results")
	}
	if status, _ := results[0]["status"].(string); !strings.Contains(status, "Would permanently delete") {
		t.Fatalf("cleanup dry run did not describe permanent deletion: %#v", results)
	}
}

func TestCLI_EmailSubscriptionsUnsubscribe_RejectsInvalidInput(t *testing.T) {
	skipIfMissingCreds(t)

	for _, args := range [][]string{
		{"email", "subscriptions", "unsubscribe"},
		{"email", "subscriptions", "unsubscribe", "bad selector", "--dry-run"},
		{"email", "subscriptions", "unsubscribe", "news@example.com", "--limit", "0", "--dry-run"},
		{"email", "subscriptions", "unsubscribe", "news@example.com", "--permanent", "--dry-run"},
		{"email", "subscriptions", "unsubscribe", "news@example.com", "--json"},
		{"email", "subscriptions", "cleanup"},
		{"email", "subscriptions", "cleanup", "bad selector", "--dry-run"},
		{"email", "subscriptions", "cleanup", "news@example.com", "--limit", "0", "--dry-run"},
		{"email", "subscriptions", "cleanup", "news@example.com", "--json"},
	} {
		if _, _, err := runCLI(args...); err == nil {
			t.Fatalf("email subscriptions unsubscribe unexpectedly accepted args: %v", args)
		}
	}
}

// =============================================================================
// EMAIL READ COMMAND TESTS
// =============================================================================

func TestCLI_EmailRead(t *testing.T) {
	skipIfMissingCreds(t)

	messageID := getRecentMessageID(t)

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
	messageID := getRecentMessageID(t)

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

	messageID := getRecentMessageID(t)

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

	messageID := getRecentMessageID(t)

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

func getRecentMessageID(t *testing.T) string {
	t.Helper()

	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messageID, err := lookupRecentMessageID(ctx, func() {
		acquireRateLimit(t)
	}, func(ctx context.Context) (string, error) {
		messages, err := client.GetMessages(ctx, testGrantID, 1)
		if err != nil {
			return "", err
		}
		if len(messages) == 0 {
			return "", fmt.Errorf("no messages available for read test")
		}

		return messages[0].ID, nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "no messages available") {
			t.Skip("No messages available for read test")
		}
		t.Fatalf("Failed to get messages: %v", err)
	}

	return messageID
}

func lookupRecentMessageID(ctx context.Context, beforeAttempt func(), fetch func(context.Context) (string, error)) (string, error) {
	var messageID string
	err := common.WithRetry(ctx, common.RetryConfig{
		MaxRetries:  3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0,
	}, func() error {
		if beforeAttempt != nil {
			beforeAttempt()
		}
		id, err := fetch(ctx)
		if err != nil {
			return err
		}
		messageID = id
		return nil
	})
	return messageID, err
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

func TestCLI_EmailRead_MIME(t *testing.T) {
	skipIfMissingCreds(t)
	acquireRateLimit(t)

	// Get a message ID first
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := client.GetMessages(ctx, testGrantID, 1)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) == 0 {
		t.Skip("No messages available for MIME test")
	}

	messageID := messages[0].ID

	stdout, stderr, err := runCLI("email", "read", messageID, testGrantID, "--mime")

	if err != nil {
		t.Fatalf("email read --mime failed: %v\nstderr: %s", err, stderr)
	}

	// Should show MIME header or error message if MIME not available
	if !strings.Contains(stdout, "RAW RFC822/MIME FORMAT") &&
		!strings.Contains(stdout, "No raw MIME data available") {
		t.Errorf("Expected MIME output or error message, got: %s", stdout)
	}

	t.Logf("email read --mime output:\n%s", stdout)
}

// =============================================================================
// EMAIL SEND COMMAND TESTS
// =============================================================================
