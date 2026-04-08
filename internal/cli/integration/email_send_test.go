//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// =============================================================================
// EMAIL LIST COMMAND TESTS
// =============================================================================

func TestCLI_EmailSend(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	skipIfMissingCreds(t)

	email := getSendTargetEmail(t)

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
	if !strings.Contains(stdout, "--template-id") {
		t.Errorf("Expected '--template-id' in send help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--render-only") {
		t.Errorf("Expected '--render-only' in send help, got: %s", stdout)
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

	email := getSendTargetEmail(t)

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

func TestCLI_EmailSend_RenderOnlyHostedTemplate(t *testing.T) {
	skipIfMissingCreds(t)

	createStdout, createStderr, createErr := runCLIWithRateLimit(t,
		"template", "create",
		"--name", "Send Preview Integration Template",
		"--subject", "Hosted Hello {{user.name}}",
		"--body", "<p>Hello {{user.name}}</p>",
		"--engine", "mustache",
		"--json",
	)
	if createErr != nil {
		t.Fatalf("template create failed: %v\nstderr: %s", createErr, createStderr)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(createStdout)), &created); err != nil {
		t.Fatalf("failed to parse template create output: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("template create did not return an id: %s", createStdout)
	}

	t.Cleanup(func() {
		if created.ID != "" {
			_, _, _ = runCLI("template", "delete", created.ID, "--yes")
		}
	})

	isolatedHome := t.TempDir()
	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, map[string]string{
		"NYLAS_GRANT_ID":        "",
		"XDG_CONFIG_HOME":       isolatedHome,
		"HOME":                  isolatedHome,
		"NYLAS_DISABLE_KEYRING": "true",
	},
		"email", "send",
		"--template-id", created.ID,
		"--template-data", `{"user":{"name":"Integration"}}`,
		"--render-only",
	)
	if err != nil {
		t.Fatalf("email send --render-only failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Hosted Hello Integration") {
		t.Fatalf("expected rendered subject in preview, got: %s", stdout)
	}
	if !strings.Contains(stdout, "<p>Hello Integration</p>") {
		t.Fatalf("expected rendered body in preview, got: %s", stdout)
	}
}

func TestCLI_EmailSend_HostedTemplate(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping send test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	skipIfMissingCreds(t)

	email := getSendTargetEmail(t)

	createStdout, createStderr, createErr := runCLIWithRateLimit(t,
		"template", "create",
		"--name", "Send Integration Template",
		"--subject", "Hosted Send {{user.name}}",
		"--body", "<p>Hello {{user.name}}, this message was sent from a hosted template.</p>",
		"--engine", "mustache",
		"--json",
	)
	if createErr != nil {
		t.Fatalf("template create failed: %v\nstderr: %s", createErr, createStderr)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(createStdout)), &created); err != nil {
		t.Fatalf("failed to parse template create output: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("template create did not return an id: %s", createStdout)
	}

	t.Cleanup(func() {
		if created.ID != "" {
			_, _, _ = runCLI("template", "delete", created.ID, "--yes")
		}
	})

	stdout, stderr, err := runCLIWithRateLimit(t,
		"email", "send",
		"--to", email,
		"--template-id", created.ID,
		"--template-data", `{"user":{"name":"Integration"}}`,
		"--yes",
		testGrantID,
	)
	if err != nil {
		t.Fatalf("email send with hosted template failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "sent") && !strings.Contains(stdout, "Message") && !strings.Contains(stdout, "✓") {
		t.Fatalf("expected send confirmation in output, got: %s", stdout)
	}
}

func TestCLI_EmailSend_HostedTemplate_SelfRoundTrip(t *testing.T) {
	if os.Getenv("NYLAS_TEST_SEND_EMAIL") != "true" {
		t.Skip("Skipping self-send round-trip test - set NYLAS_TEST_SEND_EMAIL=true to enable")
	}
	skipIfMissingCreds(t)

	selfEmail := getGrantEmail(t)
	token := fmt.Sprintf("self-check-%d", time.Now().UnixNano())
	renderedAt := time.Now().Format(time.RFC3339)

	createStdout, createStderr, createErr := runCLIWithRateLimit(t,
		"template", "create",
		"--name", "Hosted Self Round Trip "+token,
		"--subject", "[CLI Self Check] {{token}} for {{name}}",
		"--body", "<p>Hello {{name}},</p><p>This is a hosted-template self-check.</p><p>Token: {{token}}</p><p>Rendered at: {{ts}}</p>",
		"--engine", "mustache",
		"--json",
	)
	if createErr != nil {
		t.Fatalf("template create failed: %v\nstderr: %s", createErr, createStderr)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(createStdout)), &created); err != nil {
		t.Fatalf("failed to parse template create output: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("template create did not return an id: %s", createStdout)
	}

	t.Cleanup(func() {
		if created.ID != "" {
			_, _, _ = runCLI("template", "delete", created.ID, "--yes")
		}
	})

	sendStdout, sendStderr, sendErr := runCLIWithRateLimit(t,
		"email", "send",
		"--to", selfEmail,
		"--template-id", created.ID,
		"--template-data", fmt.Sprintf(`{"name":"Qasim","token":"%s","ts":"%s"}`, token, renderedAt),
		"--yes",
		testGrantID,
	)
	if sendErr != nil {
		t.Fatalf("email send with hosted template failed: %v\nstderr: %s", sendErr, sendStderr)
	}

	messageID := extractMessageID(sendStdout)
	if messageID == "" {
		t.Fatalf("failed to extract message ID from send output: %s", sendStdout)
	}

	t.Cleanup(func() {
		_, _, _ = runCLI("email", "delete", messageID, "--yes", testGrantID)
	})

	var delivered struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
		Snippet string `json:"snippet"`
	}

	var lastStdout string
	var lastStderr string
	var found bool

	for attempt := 1; attempt <= 10; attempt++ {
		readStdout, readStderr, readErr := runCLIWithRateLimit(t, "email", "read", messageID, testGrantID, "--json")
		lastStdout = readStdout
		lastStderr = readStderr

		if readErr == nil {
			if err := json.Unmarshal([]byte(strings.TrimSpace(readStdout)), &delivered); err != nil {
				t.Fatalf("failed to parse email read output: %v\noutput: %s", err, readStdout)
			}

			if strings.Contains(delivered.Subject, token) &&
				strings.Contains(delivered.Body, token) &&
				strings.Contains(delivered.Body, "hosted-template self-check") {
				found = true
				break
			}
		}

		time.Sleep(3 * time.Second)
	}

	if !found {
		t.Fatalf("self-send round trip did not verify rendered content\nlast stdout: %s\nlast stderr: %s", lastStdout, lastStderr)
	}

	if delivered.ID != messageID {
		t.Fatalf("read message ID = %q, want %q", delivered.ID, messageID)
	}
}

func TestCLI_EmailSend_RenderOnlyHostedTemplateGrantScopedFlags(t *testing.T) {
	skipIfMissingCreds(t)
	grantIdentifier := getGrantEmail(t)
	envOverrides := newSeededGrantStoreEnv(t, domain.GrantInfo{ID: testGrantID, Email: grantIdentifier})

	createStdout, createStderr, createErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "create",
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--name", "Grant Hosted Send Preview Template",
		"--subject", "Grant scoped preview",
		"--body", "<p>Hello {{user.name}}</p>",
		"--engine", "mustache",
		"--json",
	)
	if createErr != nil {
		t.Fatalf("grant-scoped template create failed: %v\nstderr: %s", createErr, createStderr)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(createStdout)), &created); err != nil {
		t.Fatalf("failed to parse template create output: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("template create did not return an id: %s", createStdout)
	}

	t.Cleanup(func() {
		if created.ID != "" {
			_, _, _ = runCLIWithOverrides(2*time.Minute, envOverrides,
				"template", "delete", created.ID,
				"--scope", "grant",
				"--grant-id", grantIdentifier,
				"--yes",
			)
		}
	})

	tempDir := t.TempDir()
	dataPath := filepath.Join(tempDir, "send-template-data.json")
	if err := os.WriteFile(dataPath, []byte(`{"user":{"name":"Grant Preview"}}`), 0o600); err != nil {
		t.Fatalf("failed to write template data file: %v", err)
	}

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"email", "send",
		"--template-id", created.ID,
		"--template-scope", "grant",
		"--template-grant-id", grantIdentifier,
		"--template-data-file", dataPath,
		"--template-strict=false",
		"--render-only",
	)
	if err != nil {
		t.Fatalf("grant-scoped email send --render-only failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Grant scoped preview") {
		t.Fatalf("expected rendered subject in preview, got: %s", stdout)
	}
	if !strings.Contains(stdout, "<p>Hello Grant Preview</p>") {
		t.Fatalf("expected rendered body in preview, got: %s", stdout)
	}
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
				// Microsoft Graph doesn't support combining search query with unread/starred filters
				if strings.Contains(stderr, "not supported for Microsoft") {
					t.Skipf("filter combination not supported for Microsoft grants")
				}
				t.Fatalf("email search %s failed: %v\nstderr: %s", tt.name, err, stderr)
			}
			t.Logf("email search %s output:\n%s", tt.name, stdout)
		})
	}
}

// =============================================================================
// THREAD SEARCH COMMAND TESTS (Phase 3)
// =============================================================================
