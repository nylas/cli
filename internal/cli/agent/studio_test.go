package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentStudioCmd(t *testing.T) {
	cmd := newStudioCmd()

	assert.Equal(t, "studio", cmd.Use)
	assert.Empty(t, cmd.Aliases, "studio must not have aliases")
	assert.Contains(t, cmd.Short, "Agent Studio")

	portFlag := cmd.Flags().Lookup("port")
	if assert.NotNil(t, portFlag, "studio needs a --port flag") {
		assert.Equal(t, "7368", portFlag.DefValue)
	}
	assert.NotNil(t, cmd.Flags().Lookup("no-browser"), "studio needs a --no-browser flag")
}
