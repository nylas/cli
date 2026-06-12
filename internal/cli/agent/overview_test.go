package agent

import (
	"testing"

	"github.com/nylas/cli/internal/agentgraph"
	"github.com/stretchr/testify/assert"
)

// Graph-assembly behavior is tested in internal/agentgraph; this file covers
// the command wiring and the policy-line rendering.

func TestAgentOverviewCmd(t *testing.T) {
	cmd := newOverviewCmd()

	assert.Equal(t, "overview", cmd.Use)
	assert.Contains(t, cmd.Aliases, "tree")
	assert.Contains(t, cmd.Short, "overview")
}

// A workspace without a (live) policy is a normal state, not an error: the
// account runs at the billing plan's maximum limits, and the output must say
// so instead of implying a policy is required.
func TestPrintOverviewWorkspace_PolicyFallbackMessaging(t *testing.T) {
	base := agentgraph.Account{WorkspaceID: "ws-1", WorkspaceName: "Support"}

	noPolicy := base
	out := captureStdout(t, func() { printOverviewWorkspace(noPolicy) })
	assert.Contains(t, out, "no policy attached — plan maximums apply")

	missing := base
	missing.Policy = &agentgraph.Policy{ID: "policy-gone", Missing: true}
	out = captureStdout(t, func() { printOverviewWorkspace(missing) })
	assert.Contains(t, out, "Policy policy-gone no longer exists — plan maximums apply")

	attached := base
	attached.Policy = &agentgraph.Policy{ID: "policy-1", Name: "Strict"}
	out = captureStdout(t, func() { printOverviewWorkspace(attached) })
	assert.Contains(t, out, "Policy: Strict")
	assert.NotContains(t, out, "plan maximums")
}
