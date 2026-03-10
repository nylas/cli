package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProviderCmd(t *testing.T) {
	cmd := NewProviderCmd()
	assert.Equal(t, "provider", cmd.Use)
	assert.True(t, cmd.HasSubCommands())

	// Check subcommands exist
	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "setup")
	assert.Contains(t, names, "status")
}

func TestNewSetupCmd(t *testing.T) {
	cmd := newSetupCmd()
	assert.Equal(t, "setup", cmd.Use)
	assert.True(t, cmd.HasSubCommands())

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "google")
}

func TestNewGoogleSetupCmd(t *testing.T) {
	cmd := newGoogleSetupCmd()
	assert.Equal(t, "google", cmd.Use)

	// Check flags exist
	flags := []string{"region", "project-id", "email", "calendar", "contacts", "pubsub", "yes", "fresh"}
	for _, flag := range flags {
		f := cmd.Flags().Lookup(flag)
		require.NotNil(t, f, "flag %s should exist", flag)
	}
}
