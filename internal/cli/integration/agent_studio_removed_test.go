//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestCLI_AgentStudioRemoved(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("agent", "studio")
	if err == nil {
		t.Fatal("expected agent studio command to fail")
	}

	output := strings.ToLower(stdout + stderr)
	if !strings.Contains(output, `unknown command "studio"`) {
		t.Fatalf("expected unknown command error, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestCLI_AgentsStudioRemoved(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("agents", "studio")
	if err == nil {
		t.Fatal("expected agents studio command to fail")
	}

	output := strings.ToLower(stdout + stderr)
	if !strings.Contains(output, `unknown command "studio"`) {
		t.Fatalf("expected unknown command error, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestCLI_AgentHelpOmitsStudio(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("agent", "--help")
	if err != nil {
		t.Fatalf("agent --help failed: %v\nstderr: %s", err, stderr)
	}

	output := strings.ToLower(stdout)
	if strings.Contains(output, "\n  studio ") || strings.Contains(output, "agent studio") {
		t.Fatalf("expected agent help to omit studio command, got: %s", stdout)
	}
}
