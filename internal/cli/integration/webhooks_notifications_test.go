//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/webhookserver"
)

func TestCLI_WebhookRotateSecretHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "rotate-secret", "--help")

	if err != nil {
		t.Fatalf("webhook rotate-secret --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "--yes") {
		t.Errorf("Expected --yes flag in help, got: %s", stdout)
	}

	t.Logf("webhook rotate-secret --help output:\n%s", stdout)
}

func TestCLI_WebhookVerifyHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "verify", "--help")

	if err != nil {
		t.Fatalf("webhook verify --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "--payload") || !strings.Contains(stdout, "--payload-file") {
		t.Errorf("Expected payload flags in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "--signature") || !strings.Contains(stdout, "--secret") {
		t.Errorf("Expected signature and secret flags in help, got: %s", stdout)
	}

	t.Logf("webhook verify --help output:\n%s", stdout)
}

func TestCLI_WebhookVerifyLocal(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	payload := `{"type":"message.created"}`
	secret := "integration-secret"
	signature := webhookserver.ComputeSignature([]byte(payload), secret)

	stdout, stderr, err := runCLI(
		"webhook", "verify",
		"--payload", payload,
		"--signature", "sha256="+signature,
		"--secret", secret,
	)

	if err != nil {
		t.Fatalf("webhook verify failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Signature is valid") {
		t.Errorf("Expected signature validation output, got: %s", stdout)
	}
}

func TestCLI_WebhookPubSubHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("webhook", "pubsub", "--help")

	if err != nil {
		t.Fatalf("webhook pubsub --help failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "list") || !strings.Contains(stdout, "create") {
		t.Errorf("Expected list and create subcommands in help, got: %s", stdout)
	}
	if !strings.Contains(stdout, "show") || !strings.Contains(stdout, "update") || !strings.Contains(stdout, "delete") {
		t.Errorf("Expected show, update, and delete subcommands in help, got: %s", stdout)
	}

	t.Logf("webhook pubsub --help output:\n%s", stdout)
}
