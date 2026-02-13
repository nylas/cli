package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAgent(t *testing.T) {
	agents := []Agent{
		{Type: AgentClaude, Path: "/usr/bin/claude", Version: "1.0.0"},
		{Type: AgentOllama, Path: "/usr/local/bin/ollama", Model: "mistral"},
		{Type: AgentCodex, Path: "/opt/codex", Version: "2.1.0"},
	}

	tests := []struct {
		name      string
		agentType AgentType
		wantFound bool
		validate  func(t *testing.T, agent *Agent)
	}{
		{
			name:      "finds claude agent",
			agentType: AgentClaude,
			wantFound: true,
			validate: func(t *testing.T, agent *Agent) {
				assert.Equal(t, AgentClaude, agent.Type)
				assert.Equal(t, "/usr/bin/claude", agent.Path)
				assert.Equal(t, "1.0.0", agent.Version)
			},
		},
		{
			name:      "finds ollama agent",
			agentType: AgentOllama,
			wantFound: true,
			validate: func(t *testing.T, agent *Agent) {
				assert.Equal(t, AgentOllama, agent.Type)
				assert.Equal(t, "/usr/local/bin/ollama", agent.Path)
				assert.Equal(t, "mistral", agent.Model)
			},
		},
		{
			name:      "finds codex agent",
			agentType: AgentCodex,
			wantFound: true,
			validate: func(t *testing.T, agent *Agent) {
				assert.Equal(t, AgentCodex, agent.Type)
				assert.Equal(t, "/opt/codex", agent.Path)
				assert.Equal(t, "2.1.0", agent.Version)
			},
		},
		{
			name:      "returns nil for non-existent agent",
			agentType: "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := FindAgent(agents, tt.agentType)

			if tt.wantFound {
				require.NotNil(t, agent, "agent should be found")
				if tt.validate != nil {
					tt.validate(t, agent)
				}
			} else {
				assert.Nil(t, agent, "agent should not be found")
			}
		})
	}
}

func TestFindAgent_EmptyList(t *testing.T) {
	agents := []Agent{}
	agent := FindAgent(agents, AgentClaude)
	assert.Nil(t, agent, "should return nil for empty agent list")
}

func TestFindAgent_FirstMatch(t *testing.T) {
	// Test that it returns the first matching agent
	agents := []Agent{
		{Type: AgentClaude, Path: "/usr/bin/claude", Version: "1.0"},
		{Type: AgentClaude, Path: "/usr/local/bin/claude", Version: "2.0"},
		{Type: AgentOllama, Path: "/usr/bin/ollama"},
	}

	agent := FindAgent(agents, AgentClaude)
	require.NotNil(t, agent)
	assert.Equal(t, "/usr/bin/claude", agent.Path, "should return first matching agent")
	assert.Equal(t, "1.0", agent.Version)
}

func TestAgent_String(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		expected string
	}{
		{
			name: "claude with version",
			agent: Agent{
				Type:    AgentClaude,
				Path:    "/usr/bin/claude",
				Version: "1.0.0",
			},
			expected: "claude 1.0.0",
		},
		{
			name: "ollama with model and version",
			agent: Agent{
				Type:    AgentOllama,
				Path:    "/usr/bin/ollama",
				Model:   "mistral",
				Version: "0.1.0",
			},
			expected: "ollama (mistral) 0.1.0",
		},
		{
			name: "ollama with model only",
			agent: Agent{
				Type:  AgentOllama,
				Path:  "/usr/bin/ollama",
				Model: "llama2",
			},
			expected: "ollama (llama2)",
		},
		{
			name: "codex without version or model",
			agent: Agent{
				Type: AgentCodex,
				Path: "/opt/codex",
			},
			expected: "codex",
		},
		{
			name: "agent with version but no model",
			agent: Agent{
				Type:    AgentClaude,
				Path:    "/usr/bin/claude",
				Version: "2.5.1",
			},
			expected: "claude 2.5.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_String_AllAgentTypes(t *testing.T) {
	agentTypes := []AgentType{AgentClaude, AgentCodex, AgentOllama}

	for _, agentType := range agentTypes {
		t.Run(string(agentType), func(t *testing.T) {
			agent := Agent{Type: agentType, Path: "/usr/bin/test"}
			str := agent.String()
			assert.Contains(t, str, string(agentType), "string representation should contain agent type")
		})
	}
}

func TestAgentType_Constants(t *testing.T) {
	t.Run("agent type values", func(t *testing.T) {
		assert.Equal(t, AgentType("claude"), AgentClaude)
		assert.Equal(t, AgentType("codex"), AgentCodex)
		assert.Equal(t, AgentType("ollama"), AgentOllama)
	})

	t.Run("agent types are unique", func(t *testing.T) {
		types := map[AgentType]bool{
			AgentClaude: true,
			AgentCodex:  true,
			AgentOllama: true,
		}
		assert.Equal(t, 3, len(types), "all agent types should be unique")
	})
}

func TestDetectAgents_Structure(t *testing.T) {
	// This test validates the structure without requiring actual binaries
	// We can't test the actual detection without mocking exec.LookPath
	t.Run("returns slice of agents", func(t *testing.T) {
		agents := DetectAgents()
		assert.NotNil(t, agents, "should return non-nil slice")
		// May be empty if no agents are installed
	})

	t.Run("detected agents have required fields", func(t *testing.T) {
		agents := DetectAgents()
		for _, agent := range agents {
			assert.NotEmpty(t, agent.Type, "agent should have a type")
			assert.NotEmpty(t, agent.Path, "agent should have a path")
			// Version and Model are optional
		}
	})

	t.Run("ollama agents have default model", func(t *testing.T) {
		agents := DetectAgents()
		for _, agent := range agents {
			if agent.Type == AgentOllama {
				assert.Equal(t, "mistral", agent.Model, "ollama should have default model 'mistral'")
			}
		}
	})
}

func TestAgent_Fields(t *testing.T) {
	t.Run("agent with all fields", func(t *testing.T) {
		agent := Agent{
			Type:    AgentClaude,
			Path:    "/usr/bin/claude",
			Model:   "claude-3",
			Version: "3.0.0",
		}

		assert.Equal(t, AgentClaude, agent.Type)
		assert.Equal(t, "/usr/bin/claude", agent.Path)
		assert.Equal(t, "claude-3", agent.Model)
		assert.Equal(t, "3.0.0", agent.Version)
	})

	t.Run("agent with minimal fields", func(t *testing.T) {
		agent := Agent{
			Type: AgentCodex,
			Path: "/opt/codex",
		}

		assert.Equal(t, AgentCodex, agent.Type)
		assert.Equal(t, "/opt/codex", agent.Path)
		assert.Empty(t, agent.Model)
		assert.Empty(t, agent.Version)
	})
}

func TestFindAgent_PointerSafety(t *testing.T) {
	agents := []Agent{
		{Type: AgentClaude, Path: "/usr/bin/claude"},
	}

	agent1 := FindAgent(agents, AgentClaude)
	agent2 := FindAgent(agents, AgentClaude)

	// Both should point to the same agent in the slice
	require.NotNil(t, agent1)
	require.NotNil(t, agent2)

	// Modifying via pointer should affect the original
	agent1.Version = "modified"
	assert.Equal(t, "modified", agents[0].Version, "pointer should reference original agent")
}
