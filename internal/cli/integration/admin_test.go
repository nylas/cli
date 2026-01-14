//go:build integration

package integration

import (
	"strings"
	"testing"
)

// =============================================================================
// ADMIN COMMAND TESTS
// =============================================================================

func TestCLI_AdminHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("admin", "--help")

	if err != nil {
		t.Fatalf("admin --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show admin subcommands
	if !strings.Contains(stdout, "applications") || !strings.Contains(stdout, "connectors") {
		t.Errorf("Expected admin subcommands in help, got: %s", stdout)
	}

	t.Logf("admin --help output:\n%s", stdout)
}

// Applications Tests

func TestCLI_AdminApplicationsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("admin", "applications", "--help")

	if err != nil {
		t.Fatalf("admin applications --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show application subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected application subcommands in help, got: %s", stdout)
	}

	t.Logf("admin applications --help output:\n%s", stdout)
}

func TestCLI_AdminApplicationsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "applications", "list")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin applications list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show applications list or "No applications found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No applications found") {
		t.Errorf("Expected applications list output, got: %s", stdout)
	}

	t.Logf("admin applications list output:\n%s", stdout)
}

func TestCLI_AdminApplicationsListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "applications", "list", "--json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin applications list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should output JSON (array)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "[") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("admin applications list --json output:\n%s", stdout)
}

// Connectors Tests

func TestCLI_AdminConnectorsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("admin", "connectors", "--help")

	if err != nil {
		t.Fatalf("admin connectors --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show connector subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected connector subcommands in help, got: %s", stdout)
	}

	t.Logf("admin connectors --help output:\n%s", stdout)
}

func TestCLI_AdminConnectorsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "connectors", "list")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin connectors list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show connectors list or "No connectors found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No connectors found") {
		t.Errorf("Expected connectors list output, got: %s", stdout)
	}

	t.Logf("admin connectors list output:\n%s", stdout)
}

func TestCLI_AdminConnectorsListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "connectors", "list", "--json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin connectors list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should output JSON (array)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "[") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("admin connectors list --json output:\n%s", stdout)
}

// TODO: Credentials Tests - Uncomment when credentials.go is implemented
/*
func TestCLI_AdminCredentialsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("admin", "credentials", "--help")

	if err != nil {
		t.Fatalf("admin credentials --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show credential subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected credential subcommands in help, got: %s", stdout)
	}

	t.Logf("admin credentials --help output:\n%s", stdout)
}

func TestCLI_AdminCredentialsList(t *testing.T) {
	skipIfMissingCreds(t)

	// First, get a list of connectors to find a connector provider
	connStdout, connStderr, connErr := runCLI("admin", "connectors", "list", "--json")
	skipIfProviderNotSupported(t, connStderr)

	if connErr != nil {
		t.Skipf("Cannot list connectors: %v", connErr)
	}

	// Parse connectors JSON to get a connector provider
	var connectors []map[string]interface{}
	if err := json.Unmarshal([]byte(connStdout), &connectors); err != nil || len(connectors) == 0 {
		t.Skip("No connectors found to test credentials list")
	}

	// Use provider field as connector identifier (v3 API uses provider, not id)
	connectorProvider, ok := connectors[0]["provider"].(string)
	if !ok || connectorProvider == "" {
		t.Skip("No valid connector provider found")
	}

	stdout, stderr, err := runCLI("admin", "credentials", "list", "--connector-id", connectorProvider)
	skipIfProviderNotSupported(t, stderr)

	// Skip if credentials endpoint isn't available (404 - endpoint may not exist for all providers)
	if err != nil && strings.Contains(stderr, "status 404") {
		t.Skip("Credentials endpoint not available for this connector")
	}

	if err != nil {
		t.Fatalf("admin credentials list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show credentials list or "No credentials found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No credentials found") {
		t.Errorf("Expected credentials list output, got: %s", stdout)
	}

	t.Logf("admin credentials list output:\n%s", stdout)
}

func TestCLI_AdminCredentialsListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	// First, get a list of connectors to find a connector provider
	connStdout, connStderr, connErr := runCLI("admin", "connectors", "list", "--json")
	skipIfProviderNotSupported(t, connStderr)

	if connErr != nil {
		t.Skipf("Cannot list connectors: %v", connErr)
	}

	// Parse connectors JSON to get a connector provider
	var connectors []map[string]interface{}
	if err := json.Unmarshal([]byte(connStdout), &connectors); err != nil || len(connectors) == 0 {
		t.Skip("No connectors found to test credentials list")
	}

	// Use provider field as connector identifier (v3 API uses provider, not id)
	connectorProvider, ok := connectors[0]["provider"].(string)
	if !ok || connectorProvider == "" {
		t.Skip("No valid connector provider found")
	}

	stdout, stderr, err := runCLI("admin", "credentials", "list", "--connector-id", connectorProvider, "--json")
	skipIfProviderNotSupported(t, stderr)

	// Skip if credentials endpoint isn't available (404 - endpoint may not exist for all providers)
	if err != nil && strings.Contains(stderr, "status 404") {
		t.Skip("Credentials endpoint not available for this connector")
	}

	if err != nil {
		t.Fatalf("admin credentials list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should output JSON (array)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "[") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("admin credentials list --json output:\n%s", stdout)
}
*/

// Grants Tests

func TestCLI_AdminGrantsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("admin", "grants", "--help")

	if err != nil {
		t.Fatalf("admin grants --help failed: %v\nstderr: %s", err, stderr)
	}

	// Should show grant subcommands
	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "stats") {
		t.Errorf("Expected grant subcommands in help, got: %s", stdout)
	}

	t.Logf("admin grants --help output:\n%s", stdout)
}

func TestCLI_AdminGrantsList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "grants", "list")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin grants list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show grants list or "No grants found"
	if !strings.Contains(stdout, "Found") && !strings.Contains(stdout, "No grants found") {
		t.Errorf("Expected grants list output, got: %s", stdout)
	}

	t.Logf("admin grants list output:\n%s", stdout)
}

func TestCLI_AdminGrantsListJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "grants", "list", "--json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin grants list --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should output JSON (array)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "[") {
		t.Errorf("Expected JSON array output, got: %s", stdout)
	}

	t.Logf("admin grants list --json output:\n%s", stdout)
}

func TestCLI_AdminGrantsStats(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "grants", "stats")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin grants stats failed: %v\nstderr: %s", err, stderr)
	}

	// Should show grant statistics
	if !strings.Contains(stdout, "Grant Statistics") && !strings.Contains(stdout, "Total Grants") {
		t.Errorf("Expected grant statistics output, got: %s", stdout)
	}

	t.Logf("admin grants stats output:\n%s", stdout)
}

func TestCLI_AdminGrantsStatsJSON(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("admin", "grants", "stats", "--json")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("admin grants stats --json failed: %v\nstderr: %s", err, stderr)
	}

	// Should output JSON (object)
	trimmed := strings.TrimSpace(stdout)
	if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "{") {
		t.Errorf("Expected JSON object output, got: %s", stdout)
	}

	t.Logf("admin grants stats --json output:\n%s", stdout)
}
