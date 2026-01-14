//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

// =============================================================================
// AUTH ENHANCEMENTS TESTS
// =============================================================================

func TestCLI_AuthProvidersHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "providers", "--help")

	if err != nil {
		t.Fatalf("auth providers --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "List all available authentication providers") {
		t.Errorf("Expected providers help text, got: %s", stdout)
	}

	t.Logf("auth providers --help output:\n%s", stdout)
}

func TestCLI_AuthProvidersList(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	if testAPIKey == "" {
		t.Skip("NYLAS_API_KEY not set")
	}

	stdout, stderr, err := runCLI("auth", "providers")

	if err != nil {
		t.Fatalf("auth providers failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Available Authentication Providers") {
		t.Errorf("Expected providers list header, got: %s", stdout)
	}

	t.Logf("auth providers output:\n%s", stdout)
}

func TestCLI_AuthProvidersListJSON(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	if testAPIKey == "" {
		t.Skip("NYLAS_API_KEY not set")
	}

	stdout, stderr, err := runCLI("auth", "providers", "--json")

	if err != nil {
		t.Fatalf("auth providers --json failed: %v\nstderr: %s", err, stderr)
	}

	var connectors []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &connectors); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\noutput: %s", err, stdout)
	}

	t.Logf("auth providers --json output: %d connectors", len(connectors))
}

// Detect command tests

func TestCLI_AuthDetectHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "detect", "--help")

	if err != nil {
		t.Fatalf("auth detect --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Detect the authentication provider based on an email address") {
		t.Errorf("Expected detect help text, got: %s", stdout)
	}

	t.Logf("auth detect --help output:\n%s", stdout)
}

func TestCLI_AuthDetectGmail(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "detect", "user@gmail.com")

	if err != nil {
		t.Fatalf("auth detect user@gmail.com failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "gmail.com") || !strings.Contains(stdout, "google") {
		t.Errorf("Expected Gmail detection, got: %s", stdout)
	}

	t.Logf("auth detect user@gmail.com output:\n%s", stdout)
}

func TestCLI_AuthDetectOutlook(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "detect", "user@outlook.com")

	if err != nil {
		t.Fatalf("auth detect user@outlook.com failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "outlook.com") || !strings.Contains(stdout, "microsoft") {
		t.Errorf("Expected Outlook detection, got: %s", stdout)
	}

	t.Logf("auth detect user@outlook.com output:\n%s", stdout)
}

func TestCLI_AuthDetectICloud(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "detect", "user@icloud.com")

	if err != nil {
		t.Fatalf("auth detect user@icloud.com failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "icloud.com") || !strings.Contains(stdout, "icloud") {
		t.Errorf("Expected iCloud detection, got: %s", stdout)
	}

	t.Logf("auth detect user@icloud.com output:\n%s", stdout)
}

func TestCLI_AuthDetectYahoo(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "detect", "user@yahoo.com")

	if err != nil {
		t.Fatalf("auth detect user@yahoo.com failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "yahoo.com") || !strings.Contains(stdout, "yahoo") {
		t.Errorf("Expected Yahoo detection, got: %s", stdout)
	}

	t.Logf("auth detect user@yahoo.com output:\n%s", stdout)
}

func TestCLI_AuthDetectCustomDomain(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "detect", "user@company.com")

	if err != nil {
		t.Fatalf("auth detect user@company.com failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "company.com") || !strings.Contains(stdout, "imap") {
		t.Errorf("Expected IMAP detection, got: %s", stdout)
	}

	t.Logf("auth detect user@company.com output:\n%s", stdout)
}

func TestCLI_AuthDetectJSON(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "detect", "user@gmail.com", "--json")

	if err != nil {
		t.Fatalf("auth detect --json failed: %v\nstderr: %s", err, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\noutput: %s", err, stdout)
	}

	if result["provider"] != "google" {
		t.Errorf("Expected provider=google, got: %v", result["provider"])
	}

	t.Logf("auth detect --json output:\n%s", stdout)
}

func TestCLI_AuthDetectInvalidEmail(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	_, stderr, err := runCLI("auth", "detect", "notanemail")

	if err == nil {
		t.Fatal("Expected error for invalid email, got nil")
	}

	if !strings.Contains(stderr, "invalid email") {
		t.Errorf("Expected 'invalid email' error, got: %s", stderr)
	}

	t.Logf("auth detect notanemail error output:\n%s", stderr)
}

// Scopes command tests

func TestCLI_AuthScopesHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("auth", "scopes", "--help")

	if err != nil {
		t.Fatalf("auth scopes --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Display the OAuth scopes") {
		t.Errorf("Expected scopes help text, got: %s", stdout)
	}

	t.Logf("auth scopes --help output:\n%s", stdout)
}

func TestCLI_AuthScopes(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	if testAPIKey == "" {
		t.Skip("NYLAS_API_KEY not set")
	}

	if testGrantID == "" {
		t.Skip("NYLAS_GRANT_ID not set")
	}

	stdout, stderr, err := runCLI("auth", "scopes", testGrantID)

	if err != nil {
		t.Fatalf("auth scopes failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "Grant ID") {
		t.Errorf("Expected grant info in output, got: %s", stdout)
	}

	t.Logf("auth scopes output:\n%s", stdout)
}

func TestCLI_AuthScopesJSON(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	if testAPIKey == "" {
		t.Skip("NYLAS_API_KEY not set")
	}

	if testGrantID == "" {
		t.Skip("NYLAS_GRANT_ID not set")
	}

	stdout, stderr, err := runCLI("auth", "scopes", testGrantID, "--json")

	if err != nil {
		t.Fatalf("auth scopes --json failed: %v\nstderr: %s", err, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\noutput: %s", err, stdout)
	}

	if result["grant_id"] == nil {
		t.Error("Expected grant_id in JSON output")
	}

	if result["scopes"] == nil {
		t.Error("Expected scopes in JSON output")
	}

	t.Logf("auth scopes --json output:\n%s", stdout)
}
