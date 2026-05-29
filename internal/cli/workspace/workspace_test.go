package workspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWorkspaceCmd(t *testing.T) {
	cmd := NewWorkspaceCmd()

	assert.Equal(t, "workspace", cmd.Use)
	assert.Contains(t, cmd.Aliases, "workspaces")
	assert.Contains(t, cmd.Aliases, "ws")

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}
	assert.True(t, subcommands["list"])
	assert.True(t, subcommands["get"])
	assert.True(t, subcommands["create"])
	assert.True(t, subcommands["update"])
	assert.True(t, subcommands["delete"])
}

func TestWorkspaceListCmd(t *testing.T) {
	cmd := newListCmd()

	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Aliases, "ls")
}

func TestWorkspaceCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	assert.Equal(t, "create", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("domain"))
	assert.NotNil(t, cmd.Flags().Lookup("auto-group"))
	assert.NotNil(t, cmd.Flags().Lookup("policy-id"))
}

func TestWorkspaceUpdateCmd(t *testing.T) {
	cmd := newUpdateCmd()

	assert.Equal(t, "update <workspace-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("policy-id"))
	assert.NotNil(t, cmd.Flags().Lookup("rules-ids"))
}

func TestWorkspaceDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	assert.Equal(t, "delete <workspace-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("yes"))
}

func TestWorkspaceGetCmd(t *testing.T) {
	cmd := newGetCmd()

	assert.Equal(t, "get <workspace-id>", cmd.Use)
}
