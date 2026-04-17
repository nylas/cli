//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestCLI_InboundRemoved(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("inbound", "list")
	if err == nil {
		t.Fatal("expected inbound command to fail")
	}

	output := strings.ToLower(stdout + stderr)
	if !strings.Contains(output, `unknown command "inbound"`) {
		t.Fatalf("expected unknown command error, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestCLI_InboxAliasRemoved(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("inbox", "list")
	if err == nil {
		t.Fatal("expected inbox alias to fail")
	}

	output := strings.ToLower(stdout + stderr)
	if !strings.Contains(output, `unknown command "inbox"`) {
		t.Fatalf("expected unknown command error, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestCLI_HelpOmitsInbound(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("--help")
	if err != nil {
		t.Fatalf("--help failed: %v\nstderr: %s", err, stderr)
	}

	if strings.Contains(stdout, "\ninbound") || strings.Contains(stdout, "\n  inbound") {
		t.Fatalf("expected root help to omit inbound command, got: %s", stdout)
	}
}

func TestCLI_AuthLoginRejectsInboxProvider(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("auth", "login", "--provider", "inbox")
	if err == nil {
		t.Fatal("expected auth login with provider inbox to fail")
	}

	output := strings.ToLower(stdout + stderr)
	if !strings.Contains(output, "invalid provider: inbox") {
		t.Fatalf("expected invalid provider error, got stdout=%q stderr=%q", stdout, stderr)
	}
	if !strings.Contains(output, "use 'google' or 'microsoft'") {
		t.Fatalf("expected provider guidance, got stdout=%q stderr=%q", stdout, stderr)
	}
}
