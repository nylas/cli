package notetaker

import (
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCommand_Structure(t *testing.T) {
	cmd := newUpdateCmd()

	assert.Equal(t, "update <notetaker-id> [grant-id]", cmd.Use)
	for _, flag := range []string{"join-time", "bot-name", "video-recording", "audio-recording", "transcription"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "missing --%s flag", flag)
	}
}

// With no flags set there is nothing to PATCH; the command must fail locally
// with a helpful message rather than send an empty update.
func TestUpdateCommand_RequiresAtLeastOneField(t *testing.T) {
	cmd := newUpdateCmd()
	_, _, err := testutil.ExecuteCommand(cmd, "nt-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
}

func TestNotetakerCmd_RegistersUpdate(t *testing.T) {
	cmd := NewNotetakerCmd()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["update"], "notetaker command must register the update subcommand")
}
