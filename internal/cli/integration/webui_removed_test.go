//go:build integration

package integration

import (
	"strings"
	"testing"
)

// removedWebUICommands are the localhost web surfaces removed from the CLI.
// nylas demo ui went with the ui package it rendered.
var removedWebUICommands = [][]string{
	{"air"},
	{"ui"},
	{"chat"},
	{"demo", "ui"},
}

func TestCLI_WebUIsRemoved(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	for _, args := range removedWebUICommands {
		name := strings.Join(args, " ")
		t.Run(name, func(t *testing.T) {
			stdout, stderr, err := runCLI(args...)
			if err == nil {
				t.Fatalf("expected %q to fail", name)
			}

			output := strings.ToLower(stdout + stderr)
			want := `unknown command "` + args[len(args)-1] + `"`
			if !strings.Contains(output, want) {
				t.Fatalf("expected %s, got stdout=%q stderr=%q", want, stdout, stderr)
			}
		})
	}
}

func TestCLI_HelpOmitsWebUIs(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("--help")
	if err != nil {
		t.Fatalf("--help failed: %v\nstderr: %s", err, stderr)
	}

	requireCommandList(t, stdout)

	for _, name := range []string{"air", "ui", "chat"} {
		if strings.Contains(stdout, "\n  "+name+" ") {
			t.Fatalf("expected root help to omit %s command, got: %s", name, stdout)
		}
	}
}

func TestCLI_DemoHelpOmitsUI(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}

	stdout, stderr, err := runCLI("demo", "--help")
	if err != nil {
		t.Fatalf("demo --help failed: %v\nstderr: %s", err, stderr)
	}

	requireCommandList(t, stdout)

	if strings.Contains(stdout, "\n  ui ") {
		t.Fatalf("expected demo help to omit ui command, got: %s", stdout)
	}
}
