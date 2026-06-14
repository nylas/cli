package notetaker

import (
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/stretchr/testify/assert"
)

func TestLeaveCommand(t *testing.T) {
	cmd := newLeaveCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "leave <notetaker-id> [grant-id]", cmd.Use)
	})

	t.Run("describes the leave-vs-delete distinction", func(t *testing.T) {
		// The whole reason this command exists is that "leave" keeps the
		// recording while "delete" discards it — the help must say so.
		assert.Contains(t, cmd.Long, "delete")
		assert.Contains(t, cmd.Short, "leave")
	})
}

func TestNotetakerCmd_RegistersLeave(t *testing.T) {
	cmd := NewNotetakerCmd()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["leave"], "notetaker command must register the leave subcommand")
}

func TestNotetakerLeaveHelp(t *testing.T) {
	root := testutil.NewTestRoot(NewNotetakerCmd())
	stdout, _, err := testutil.ExecuteCommand(root, "notetaker", "leave", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "leave")
}
