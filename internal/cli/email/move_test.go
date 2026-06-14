package email

import (
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMoveCommand_Structure(t *testing.T) {
	cmd := newMoveCmd()

	assert.Equal(t, "move <message-id> [grant-id]", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("folder"))
	assert.NotNil(t, cmd.Flags().Lookup("archive"))
}

// The guard checks run before any client call, so these exercise real intent
// without network access: --folder and --archive are mutually exclusive, and
// at least one destination is required.
func TestMoveCommand_RejectsFolderAndArchiveTogether(t *testing.T) {
	cmd := newMoveCmd()
	_, _, err := testutil.ExecuteCommand(cmd, "msg-1", "--folder", "F1", "--archive")

	require := assert.New(t)
	require.Error(err)
	require.Contains(err.Error(), "both")
}

func TestMoveCommand_RequiresDestination(t *testing.T) {
	cmd := newMoveCmd()
	_, _, err := testutil.ExecuteCommand(cmd, "msg-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination")
}

func TestEmailCmd_RegistersMove(t *testing.T) {
	cmd := NewEmailCmd()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["move"], "email command must register the move subcommand")
}
