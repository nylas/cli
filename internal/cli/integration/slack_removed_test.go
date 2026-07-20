//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestCLI_SlackRemoved(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("slack", "channels", "list")
	if err == nil {
		t.Fatal("expected slack command to fail")
	}

	output := strings.ToLower(stdout + stderr)
	if !strings.Contains(output, `unknown command "slack"`) {
		t.Fatalf("expected unknown command error, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestCLI_SlackAliasRemoved(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("sl", "channels", "list")
	if err == nil {
		t.Fatal("expected slack alias to fail")
	}

	output := strings.ToLower(stdout + stderr)
	if !strings.Contains(output, `unknown command "sl"`) {
		t.Fatalf("expected unknown command error, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestCLI_HelpOmitsSlack(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("--help")
	if err != nil {
		t.Fatalf("--help failed: %v\nstderr: %s", err, stderr)
	}

	requireCommandList(t, stdout)

	if strings.Contains(stdout, "\n  slack") {
		t.Fatalf("expected root help to omit slack command, got: %s", stdout)
	}
}
