package workflow

import "testing"

func TestNewWorkflowCmd(t *testing.T) {
	cmd := NewWorkflowCmd()

	if cmd.Use != "workflow" {
		t.Fatalf("cmd.Use = %q, want workflow", cmd.Use)
	}

	expected := []string{"list", "show", "create", "update", "delete"}
	subcommands := make(map[string]bool, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}

	for _, name := range expected {
		if !subcommands[name] {
			t.Fatalf("missing subcommand %q", name)
		}
	}
}

func TestWorkflowCreateCommandFlags(t *testing.T) {
	cmd := newCreateCmd()

	for _, flag := range []string{"scope", "grant-id", "file", "name", "template-id", "trigger-event", "delay", "enabled", "disabled"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing flag %q", flag)
		}
	}
}

func TestWorkflowUpdateCommandFlags(t *testing.T) {
	cmd := newUpdateCmd()

	for _, flag := range []string{"file", "set-delay", "from-name", "from-email"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing flag %q", flag)
		}
	}
}
