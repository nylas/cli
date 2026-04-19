//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestIntegration_NewAgentSandboxEnv_ClearsAgentGrantOverride(t *testing.T) {
	t.Setenv("NYLAS_AGENT_GRANT_ID", "outer-agent-grant")

	env := newAgentSandboxEnv(t)
	entries := cliTestEnv(env)

	value := ""
	for _, entry := range entries {
		if strings.HasPrefix(entry, "NYLAS_AGENT_GRANT_ID=") {
			value = strings.TrimPrefix(entry, "NYLAS_AGENT_GRANT_ID=")
			break
		}
	}

	if value != "" {
		t.Fatalf("NYLAS_AGENT_GRANT_ID = %q, want empty override", value)
	}
}
