//go:build integration

package integration

import (
	"encoding/json"
	"testing"
)

type integrationMCPStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func TestCLI_MCPStatusJSON(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("mcp", "status", "--json")
	if err != nil {
		t.Fatalf("mcp status --json failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var statuses []integrationMCPStatus
	if err := json.Unmarshal([]byte(stdout), &statuses); err != nil {
		t.Fatalf("failed to parse mcp status JSON: %v\noutput: %s", err, stdout)
	}
	if len(statuses) == 0 {
		t.Fatalf("expected at least one assistant status, got empty output: %s", stdout)
	}

	for _, status := range statuses {
		if status.ID == "" {
			t.Fatalf("assistant status missing id: %+v", status)
		}
		if status.Name == "" {
			t.Fatalf("assistant status missing name: %+v", status)
		}
		if status.Status == "" {
			t.Fatalf("assistant status missing status: %+v", status)
		}
	}
}
