//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

type integrationCommandSpec struct {
	Name           string                   `json:"name"`
	Path           string                   `json:"path"`
	Flags          []integrationFlagSpec    `json:"flags"`
	InheritedFlags []integrationFlagSpec    `json:"inherited_flags"`
	Subcommands    []integrationCommandSpec `json:"subcommands"`
}

type integrationFlagSpec struct {
	Name string `json:"name"`
}

func TestCLI_CommandsJSONRoot(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("commands", "--json")
	if err != nil {
		t.Fatalf("commands --json failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var spec integrationCommandSpec
	if err := json.Unmarshal([]byte(stdout), &spec); err != nil {
		t.Fatalf("failed to parse commands JSON: %v\noutput: %s", err, stdout)
	}

	if spec.Name != "nylas" {
		t.Fatalf("root name = %q, want nylas", spec.Name)
	}

	required := map[string]bool{
		"commands": false,
		"email":    false,
		"calendar": false,
		"mcp":      false,
	}
	for _, sub := range spec.Subcommands {
		if _, ok := required[sub.Name]; ok {
			required[sub.Name] = true
		}
	}

	for name, found := range required {
		if !found {
			t.Fatalf("expected subcommand %q in commands output, got %+v", name, spec.Subcommands)
		}
	}
}

func TestCLI_CommandsJSONSubtree(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("commands", "email", "send", "--json")
	if err != nil {
		t.Fatalf("commands email send --json failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var spec integrationCommandSpec
	if err := json.Unmarshal([]byte(stdout), &spec); err != nil {
		t.Fatalf("failed to parse commands subtree JSON: %v\noutput: %s", err, stdout)
	}

	if spec.Path != "nylas email send" {
		t.Fatalf("path = %q, want %q", spec.Path, "nylas email send")
	}

	hasToFlag := false
	for _, flag := range spec.Flags {
		if flag.Name == "to" {
			hasToFlag = true
			break
		}
	}
	if !hasToFlag {
		t.Fatalf("expected local --to flag in %+v", spec.Flags)
	}

	hasJSONFlag := false
	for _, flag := range spec.Flags {
		if flag.Name == "json" {
			hasJSONFlag = true
			break
		}
	}
	for _, flag := range spec.InheritedFlags {
		if flag.Name == "json" {
			hasJSONFlag = true
			break
		}
	}
	if !hasJSONFlag {
		t.Fatalf("expected --json flag in local or inherited flags: local=%+v inherited=%+v", spec.Flags, spec.InheritedFlags)
	}

	if strings.TrimSpace(stdout) == "" {
		t.Fatal("expected JSON output")
	}
}
