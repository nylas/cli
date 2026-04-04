//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

// =============================================================================
// ADMIN CALLBACK URI TESTS
// =============================================================================

func TestCLI_AdminCallbackURIsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("admin", "callback-uris", "--help")

	if err != nil {
		t.Fatalf("admin callback-uris --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected callback URI subcommands in help, got: %s", stdout)
	}

	if !strings.Contains(stdout, "delete") || !strings.Contains(stdout, "update") {
		t.Errorf("Expected delete and update subcommands in help, got: %s", stdout)
	}

	t.Logf("admin callback-uris --help output:\n%s", stdout)
}

func TestCLI_AdminCallbackURIsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "callback-uris", "list")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin callback-uris list failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No callback URIs found") {
		t.Errorf("Expected callback URIs list output, got: %s", stdout)
	}

	t.Logf("admin callback-uris list output:\n%s", stdout)
}

func TestCLI_AdminCallbackURIsListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "callback-uris", "list", "--json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin callback-uris list --json failed: %v\nstderr: %s", err, stderr)
	}

	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "[") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("admin callback-uris list --json output:\n%s", stdout)
}

func TestCLI_AdminCallbackURIs_CRUD(t *testing.T) {
	skipIfMissingCreds(t)

	// Create
	createStdout, createStderr, createErr := runCLIWithRateLimit(t, "admin", "callback-uris", "create",
		"--url", "http://localhost:19876/test-callback-crud",
		"--platform", "web")
	skipIfProviderNotSupported(t, createStderr)

	if createErr != nil {
		t.Fatalf("admin callback-uris create failed: %v\nstderr: %s", createErr, createStderr)
	}

	if !strings.Contains(createStdout, "Created callback URI") {
		t.Fatalf("Expected success message, got: %s", createStdout)
	}

	// Extract the created URI ID from JSON list
	listStdout, listStderr, listErr := runCLIWithRateLimit(t, "admin", "callback-uris", "list", "--json")
	skipIfProviderNotSupported(t, listStderr)

	if listErr != nil {
		t.Fatalf("admin callback-uris list --json failed: %v\nstderr: %s", listErr, listStderr)
	}

	var uris []struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Platform string `json:"platform"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(listStdout)), &uris); err != nil {
		t.Fatalf("Failed to parse callback URIs JSON: %v\noutput: %s", err, listStdout)
	}

	// Find our test URI
	var testURIID string
	for _, uri := range uris {
		if uri.URL == "http://localhost:19876/test-callback-crud" {
			testURIID = uri.ID
			break
		}
	}

	if testURIID == "" {
		t.Fatal("Could not find created test callback URI in list")
	}

	// Cleanup: delete the test URI regardless of subsequent test results
	t.Cleanup(func() {
		_, _, _ = runCLI("admin", "callback-uris", "delete", testURIID, "--yes")
	})

	// Show
	showStdout, showStderr, showErr := runCLIWithRateLimit(t, "admin", "callback-uris", "show", testURIID)
	skipIfProviderNotSupported(t, showStderr)

	if showErr != nil {
		t.Fatalf("admin callback-uris show failed: %v\nstderr: %s", showErr, showStderr)
	}

	if !strings.Contains(showStdout, testURIID) {
		t.Errorf("Expected URI ID in show output, got: %s", showStdout)
	}

	if !strings.Contains(showStdout, "http://localhost:19876/test-callback-crud") {
		t.Errorf("Expected URL in show output, got: %s", showStdout)
	}

	// Update
	updateStdout, updateStderr, updateErr := runCLIWithRateLimit(t, "admin", "callback-uris", "update", testURIID,
		"--url", "http://localhost:19876/test-callback-updated")
	skipIfProviderNotSupported(t, updateStderr)

	if updateErr != nil {
		t.Fatalf("admin callback-uris update failed: %v\nstderr: %s", updateErr, updateStderr)
	}

	if !strings.Contains(updateStdout, "Updated callback URI") {
		t.Errorf("Expected update success message, got: %s", updateStdout)
	}

	t.Logf("CRUD test passed: created %s, verified show+update, cleanup scheduled", testURIID)
}

func TestCLI_AdminCallbackURIsAlias(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Test "cb" alias works
	stdout, stderr, err := runCLI("admin", "cb", "--help")

	if err != nil {
		t.Fatalf("admin cb --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "list") {
		t.Errorf("Expected subcommands via alias, got: %s", stdout)
	}
}
