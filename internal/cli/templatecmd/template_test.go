package templatecmd

import "testing"

func TestNewTemplateCmd(t *testing.T) {
	cmd := NewTemplateCmd()

	if cmd.Use != "template" {
		t.Fatalf("cmd.Use = %q, want template", cmd.Use)
	}

	expected := []string{"list", "show", "create", "update", "delete", "render", "render-html"}
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

func TestTemplateCreateCommandFlags(t *testing.T) {
	cmd := newCreateCmd()

	for _, flag := range []string{"scope", "grant-id", "name", "subject", "body", "body-file", "engine"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing flag %q", flag)
		}
	}
}

func TestTemplateRenderCommands(t *testing.T) {
	render := newRenderCmd()
	if render.Flags().Lookup("data") == nil || render.Flags().Lookup("data-file") == nil {
		t.Fatal("render command must support --data and --data-file")
	}

	renderHTML := newRenderHTMLCmd()
	if renderHTML.Flags().Lookup("body-file") == nil || renderHTML.Flags().Lookup("engine") == nil {
		t.Fatal("render-html command must support --body-file and --engine")
	}
}
