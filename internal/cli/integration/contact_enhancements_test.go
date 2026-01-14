//go:build integration

package integration

import (
	"strings"
	"testing"
)

// =============================================================================
// CONTACT ENHANCEMENTS TESTS
// =============================================================================

func TestCLI_ContactsSearch(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("contacts", "search", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("contacts search failed: %v\nstderr: %s", err, stderr)
	}

	// Should show search results or "Found 0 contacts"
	if !strings.Contains(stdout, "Found") {
		t.Errorf("Expected search results output, got: %s", stdout)
	}

	t.Logf("contacts search output:\n%s", stdout)
}

func TestCLI_ContactsSearchWithCompany(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("contacts", "search", testGrantID, "--company", "test", "--limit", "10")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("contacts search --company failed: %v\nstderr: %s", err, stderr)
	}

	// Should show filtered search results
	if !strings.Contains(stdout, "Found") {
		t.Errorf("Expected search results output, got: %s", stdout)
	}

	t.Logf("contacts search --company output:\n%s", stdout)
}

func TestCLI_ContactsSearchHasEmail(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("contacts", "search", testGrantID, "--has-email", "--limit", "10")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("contacts search --has-email failed: %v\nstderr: %s", err, stderr)
	}

	// Should show only contacts with emails
	if !strings.Contains(stdout, "Found") {
		t.Errorf("Expected search results output, got: %s", stdout)
	}

	t.Logf("contacts search --has-email output:\n%s", stdout)
}

func TestCLI_ContactsSearchJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("contacts", "search", testGrantID, "--limit", "5", "--json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("contacts search --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should be valid JSON (starts with [ or null)
	stdout = strings.TrimSpace(stdout)
	if !strings.HasPrefix(stdout, "[") && stdout != "null" {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("contacts search --json output:\n%s", stdout)
}

func TestCLI_ContactsPhotoInfo(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "photo", "info")

	if err != nil {
		t.Fatalf("contacts photo info failed: %v\nstderr: %s", err, stderr)
	}

	// Should show profile picture information
	if !strings.Contains(stdout, "Profile Picture") {
		t.Errorf("Expected profile picture information, got: %s", stdout)
	}

	if !strings.Contains(stdout, "Retrieval") && !strings.Contains(stdout, "Base64") {
		t.Errorf("Expected information about Base64 encoding, got: %s", stdout)
	}

	t.Logf("contacts photo info output:\n%s", stdout)
}

func TestCLI_ContactsSyncInfo(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "sync")

	if err != nil {
		t.Fatalf("contacts sync failed: %v\nstderr: %s", err, stderr)
	}

	// Should show synchronization information
	if !strings.Contains(stdout, "Synchronization") && !strings.Contains(stdout, "sync") {
		t.Errorf("Expected synchronization information, got: %s", stdout)
	}

	if !strings.Contains(stdout, "v3") {
		t.Errorf("Expected information about Nylas v3, got: %s", stdout)
	}

	t.Logf("contacts sync output:\n%s", stdout)
}

func TestCLI_ContactsSearchHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "search", "--help")

	if err != nil {
		t.Fatalf("contacts search --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show search options
	if !strings.Contains(stdout, "--company") || !strings.Contains(stdout, "--email") {
		t.Errorf("Expected --company and --email flags in help, got: %s", stdout)
	}

	if !strings.Contains(stdout, "--has-email") {
		t.Errorf("Expected --has-email flag in help, got: %s", stdout)
	}

	t.Logf("contacts search --help output:\n%s", stdout)
}

func TestCLI_ContactsPhotoHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "photo", "--help")

	if err != nil {
		t.Fatalf("contacts photo --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show photo subcommands
	if !strings.Contains(stdout, "download") || !strings.Contains(stdout, "info") {
		t.Errorf("Expected download and info subcommands in help, got: %s", stdout)
	}

	t.Logf("contacts photo --help output:\n%s", stdout)
}

func TestCLI_ContactsPhotoDownloadHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("contacts", "photo", "download", "--help")

	if err != nil {
		t.Fatalf("contacts photo download --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show download options
	if !strings.Contains(stdout, "--output") || !strings.Contains(stdout, "-o") {
		t.Errorf("Expected --output flag in help, got: %s", stdout)
	}

	if !strings.Contains(stdout, "Base64") {
		t.Errorf("Expected mention of Base64 in help, got: %s", stdout)
	}

	t.Logf("contacts photo download --help output:\n%s", stdout)
}
