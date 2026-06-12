package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Graph-assembly behavior is tested in internal/agentgraph; this file covers
// only the command wiring.

func TestAgentOverviewCmd(t *testing.T) {
	cmd := newOverviewCmd()

	assert.Equal(t, "overview", cmd.Use)
	assert.Contains(t, cmd.Aliases, "tree")
	assert.Contains(t, cmd.Short, "overview")
}
