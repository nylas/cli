//go:build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// CONTACTS COMMAND TESTS
// =============================================================================

func TestCLI_ContactsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("contacts", "list", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("contacts list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show contacts list or "No contacts found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No contacts found") {
		t.Errorf("Expected contacts list output, got: %s", stdout)
	}

	t.Logf("contacts list output:\n%s", stdout)
}

func TestCLI_ContactsListWithID(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("contacts", "list", testGrantID, "--id")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("contacts list --id failed: %v\nstderr: %s", err, stderr)
	}

	// Should show contacts list with IDs or "No contacts found"
	if strings.Contains(stdout, "Found") {
		// If contacts are found, the ID column should be present
		if !strings.Contains(stdout, "ID") {
			t.Errorf("Expected 'ID' column in output with --id flag, got: %s", stdout)
		}
	}

	t.Logf("contacts list --id output:\n%s", stdout)
}

func TestCLI_ContactsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "--help")

	if err != nil {
		t.Fatalf("contacts --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show contacts subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected contacts subcommands in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "show") || !strings.Contains(stdout, "delete") {
		t.Errorf("Expected show and delete subcommands in help, got: %s", stdout)
	}

	t.Logf("contacts --help output:\n%s", stdout)
}

func TestCLI_ContactsCreateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "create", "--help")

	if err != nil {
		t.Fatalf("contacts create --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show required flags
	if !strings.Contains(stdout, "--first-name") || !strings.Contains(stdout, "--last-name") {
		t.Errorf("Expected --first-name and --last-name flags in help, got: %s", stdout)
	}

	// Should show optional flags
	if !strings.Contains(stdout, "--email") || !strings.Contains(stdout, "--phone") {
		t.Errorf("Expected --email and --phone flags in help, got: %s", stdout)
	}

	t.Logf("contacts create --help output:\n%s", stdout)
}

func TestCLI_ContactsGroupsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("contacts", "groups", "list", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("contacts groups list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show groups list or "No contact groups found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No contact groups found") {
		t.Errorf("Expected groups list output, got: %s", stdout)
	}

	t.Logf("contacts groups list output:\n%s", stdout)
}

func TestCLI_ContactsLifecycle(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	contactFirstName := "CLI"
	contactLastName := fmt.Sprintf("Test%d", time.Now().Unix())
	contactEmail := fmt.Sprintf("test%d@example.com", time.Now().Unix())

	var contactID string

	// Create contact
	t.Run("create", func(t *testing.T) {
		stdout, stderr, err := runCLI("contacts", "create",
			"--first-name", contactFirstName,
			"--last-name", contactLastName,
			"--email", contactEmail,
			testGrantID)

		if err != nil {
			t.Fatalf("contacts create failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Contact created") {
			t.Errorf("Expected 'Contact created' in output, got: %s", stdout)
		}

		// Extract contact ID from output
		if idx := strings.Index(stdout, "ID:"); idx != -1 {
			contactID = strings.TrimSpace(stdout[idx+3:])
			if newline := strings.Index(contactID, "\n"); newline != -1 {
				contactID = contactID[:newline]
			}
		}

		t.Logf("contacts create output: %s", stdout)
		t.Logf("Contact ID: %s", contactID)
	})

	if contactID == "" {
		t.Fatal("Failed to get contact ID from create output")
	}

	// Wait for contact to sync
	time.Sleep(2 * time.Second)

	// Show contact
	t.Run("show", func(t *testing.T) {
		stdout, stderr, err := runCLI("contacts", "show", contactID, testGrantID)
		if err != nil {
			t.Fatalf("contacts show failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, contactFirstName) || !strings.Contains(stdout, contactLastName) {
			t.Errorf("Expected contact name in output, got: %s", stdout)
		}

		t.Logf("contacts show output:\n%s", stdout)
	})

	// Delete contact
	t.Run("delete", func(t *testing.T) {
		stdout, stderr, err := runCLIWithInput("y\n", "contacts", "delete", contactID, testGrantID)
		if err != nil {
			t.Fatalf("contacts delete failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "deleted") {
			t.Errorf("Expected 'deleted' in output, got: %s", stdout)
		}

		t.Logf("contacts delete output: %s", stdout)
	})
}

// =============================================================================
// CONTACT UPDATE COMMAND TESTS
// =============================================================================

func TestCLI_ContactsUpdateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "update", "--help")

	if err != nil {
		t.Fatalf("contacts update --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show update flags
	if !strings.Contains(stdout, "--given-name") {
		t.Errorf("Expected --given-name flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--surname") {
		t.Errorf("Expected --surname flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--company") {
		t.Errorf("Expected --company flag in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--email") {
		t.Errorf("Expected --email flag in help, got: %s", stdout)
	}

	t.Logf("contacts update --help output:\n%s", stdout)
}

// =============================================================================
// CONTACT GROUPS CRUD COMMAND TESTS
// =============================================================================

func TestCLI_ContactsGroupsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "groups", "--help")

	if err != nil {
		t.Fatalf("contacts groups --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show groups subcommands
	if !strings.Contains(stdout, "list") {
		t.Errorf("Expected 'list' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "show") {
		t.Errorf("Expected 'show' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "create") {
		t.Errorf("Expected 'create' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "update") {
		t.Errorf("Expected 'update' in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "delete") {
		t.Errorf("Expected 'delete' in help, got: %s", stdout)
	}

	t.Logf("contacts groups --help output:\n%s", stdout)
}

func TestCLI_ContactsGroupsListHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "groups", "list", "--help")

	if err != nil {
		t.Fatalf("contacts groups list --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "List") || !strings.Contains(stdout, "contact groups") {
		t.Errorf("Expected list description in help, got: %s", stdout)
	}

	t.Logf("contacts groups list --help output:\n%s", stdout)
}

func TestCLI_ContactsGroupsCreateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "groups", "create", "--help")

	if err != nil {
		t.Fatalf("contacts groups create --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "<name>") {
		t.Errorf("Expected '<name>' in help, got: %s", stdout)
	}

	t.Logf("contacts groups create --help output:\n%s", stdout)
}

func TestCLI_ContactsGroupsUpdateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "groups", "update", "--help")

	if err != nil {
		t.Fatalf("contacts groups update --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "--name") {
		t.Errorf("Expected '--name' flag in help, got: %s", stdout)
	}

	t.Logf("contacts groups update --help output:\n%s", stdout)
}

func TestCLI_ContactsGroupsDeleteHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "groups", "delete", "--help")

	if err != nil {
		t.Fatalf("contacts groups delete --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "--force") {
		t.Errorf("Expected '--force' flag in help, got: %s", stdout)
	}

	t.Logf("contacts groups delete --help output:\n%s", stdout)
}

// =============================================================================
// CONTACT SEARCH COMMAND TESTS
// =============================================================================

// Note: General --query flag search is not supported in contacts search command.
// The CLI provides specific search flags: --email, --company, --phone, --source, --group
// TestCLI_ContactsSearch_Email below tests the --email search functionality.

func TestCLI_ContactsSearch_Email(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	// Create a test contact for searching
	contactFirstName := "EmailSearch"
	contactLastName := fmt.Sprintf("Test%d", time.Now().Unix())
	contactEmail := fmt.Sprintf("emailsearch%d@example.com", time.Now().Unix())

	var contactID string

	// Create contact
	stdout, stderr, err := runCLI("contacts", "create",
		"--first-name", contactFirstName,
		"--last-name", contactLastName,
		"--email", contactEmail,
		testGrantID)

	if err != nil {
		t.Fatalf("contacts create failed: %v\nstderr: %s", err, stderr)
	}

	// Extract contact ID for cleanup
	if idx := strings.Index(stdout, "ID:"); idx != -1 {
		contactID = strings.TrimSpace(stdout[idx+3:])
		if newline := strings.Index(contactID, "\n"); newline != -1 {
			contactID = contactID[:newline]
		}
	}

	// Cleanup contact after test
	if contactID != "" {
		t.Cleanup(func() {
			_, _, _ = runCLIWithInput("y\n", "contacts", "delete", contactID, testGrantID)
		})
	}

	// Wait for contact to sync
	time.Sleep(2 * time.Second)

	// Search by email
	t.Run("search by email", func(t *testing.T) {
		stdout, stderr, err := runCLI("contacts", "search", "--email", contactEmail, testGrantID)
		skipIfProviderNotSupported(t, stderr)

		if err != nil {
			t.Fatalf("contacts search --email failed: %v\nstderr: %s", err, stderr)
		}

		// Should find the contact
		if !strings.Contains(stdout, contactEmail) || !strings.Contains(stdout, contactFirstName) {
			t.Errorf("Expected to find contact with email %s, got: %s", contactEmail, stdout)
		}

		t.Logf("contacts search --email output:\n%s", stdout)
	})
}

// =============================================================================
// CONTACT SYNC COMMAND TESTS
// =============================================================================

func TestCLI_ContactsSyncHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "sync", "--help")

	if err != nil {
		t.Fatalf("contacts sync --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show sync command description
	if !strings.Contains(stdout, "sync") || !strings.Contains(stdout, "Sync") {
		t.Errorf("Expected sync description in help, got: %s", stdout)
	}

	t.Logf("contacts sync --help output:\n%s", stdout)
}

func TestCLI_ContactsSync(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("contacts", "sync", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("contacts sync failed: %v\nstderr: %s", err, stderr)
	}

	// Should show sync status information
	// Look for sync-related keywords like "cursor", "state", or sync status
	validOutput := strings.Contains(stdout, "cursor") ||
		strings.Contains(stdout, "state") ||
		strings.Contains(stdout, "Sync") ||
		strings.Contains(stdout, "sync") ||
		strings.Contains(stdout, "Status")

	if !validOutput {
		t.Errorf("Expected sync status information in output, got: %s", stdout)
	}

	t.Logf("contacts sync output:\n%s", stdout)
}
