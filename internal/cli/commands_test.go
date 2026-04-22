package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newCommandFixtureRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "nylas",
		Short: "Test root",
	}
	root.PersistentFlags().Bool("json", false, "Output as JSON")
	root.PersistentFlags().String("format", "", "Output format")
	root.PersistentFlags().BoolP("quiet", "q", false, "Quiet output")

	sample := &cobra.Command{
		Use:     "sample",
		Short:   "Sample command",
		Aliases: []string{"sm"},
	}
	sample.Flags().String("token", "", "Sample token")
	_ = sample.MarkFlagRequired("token")

	leaf := &cobra.Command{
		Use:   "leaf",
		Short: "Leaf command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	leaf.Flags().Bool("dry-run", false, "Preview the operation")

	sample.AddCommand(leaf)
	root.AddCommand(sample)
	root.AddCommand(newCommandsCmd())

	return root
}

func TestCommandsCmd_JSONRoot(t *testing.T) {
	root := newCommandFixtureRoot()

	stdout, stderr, err := executeCommand(root, "commands", "--json")
	if err != nil {
		t.Fatalf("commands --json failed: %v\nstderr: %s", err, stderr)
	}

	var spec commandSpec
	if err := json.Unmarshal([]byte(stdout), &spec); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, stdout)
	}

	if spec.Name != "nylas" {
		t.Fatalf("root name = %q, want nylas", spec.Name)
	}

	foundCommands := false
	for _, sub := range spec.Subcommands {
		if sub.Name == "commands" {
			foundCommands = true
			break
		}
	}
	if !foundCommands {
		t.Fatalf("expected commands command in root schema, got %+v", spec.Subcommands)
	}
}

func TestCommandsCmd_JSONSubtreeIncludesFlags(t *testing.T) {
	root := newCommandFixtureRoot()

	stdout, stderr, err := executeCommand(root, "commands", "sample", "--json")
	if err != nil {
		t.Fatalf("commands sample --json failed: %v\nstderr: %s", err, stderr)
	}

	var spec commandSpec
	if err := json.Unmarshal([]byte(stdout), &spec); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, stdout)
	}

	if spec.Path != "nylas sample" {
		t.Fatalf("path = %q, want %q", spec.Path, "nylas sample")
	}

	hasRequiredToken := false
	for _, flag := range spec.Flags {
		if flag.Name == "token" && flag.Required {
			hasRequiredToken = true
			break
		}
	}
	if !hasRequiredToken {
		t.Fatalf("expected required token flag in %+v", spec.Flags)
	}

	hasInheritedJSON := false
	for _, flag := range spec.InheritedFlags {
		if flag.Name == "json" {
			hasInheritedJSON = true
			break
		}
	}
	if !hasInheritedJSON {
		t.Fatalf("expected inherited json flag in %+v", spec.InheritedFlags)
	}

	if len(spec.Subcommands) != 1 || spec.Subcommands[0].Name != "leaf" {
		t.Fatalf("unexpected subcommands: %+v", spec.Subcommands)
	}
}

func TestCommandsCmd_DefaultListOutput(t *testing.T) {
	root := newCommandFixtureRoot()

	stdout, stderr, err := executeCommand(root, "commands")
	if err != nil {
		t.Fatalf("commands failed: %v\nstderr: %s", err, stderr)
	}

	if strings.Contains(stdout, "nylas\n") {
		t.Fatalf("default list output should not include the root row, got: %s", stdout)
	}
	if !strings.Contains(stdout, "nylas sample") {
		t.Fatalf("expected sample command in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "nylas sample leaf") {
		t.Fatalf("expected nested leaf command in output, got: %s", stdout)
	}
}
