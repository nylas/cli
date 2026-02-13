package chat

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name      string
		grantID   string
		agentType AgentType
		want      []string // Expected substrings in the output
	}{
		{
			name:      "claude agent with grant ID",
			grantID:   "grant-123",
			agentType: AgentClaude,
			want: []string{
				"You are a helpful email and calendar assistant",
				"Grant ID: grant-123",
				"## Tool Usage",
				"TOOL_CALL:",
				"## Conversation Context",
				"## Response Format",
				"Use markdown formatting",
			},
		},
		{
			name:      "codex agent with grant ID",
			grantID:   "grant-456",
			agentType: AgentCodex,
			want: []string{
				"Grant ID: grant-456",
				"TOOL_CALL:",
			},
		},
		{
			name:      "ollama agent with grant ID",
			grantID:   "grant-789",
			agentType: AgentOllama,
			want: []string{
				"Grant ID: grant-789",
			},
		},
		{
			name:      "empty grant ID",
			grantID:   "",
			agentType: AgentClaude,
			want: []string{
				"Grant ID: ",
				"You are a helpful email and calendar assistant",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSystemPrompt(tt.grantID, tt.agentType)

			// Verify the output is non-empty
			if got == "" {
				t.Error("BuildSystemPrompt returned empty string")
			}

			// Check that all expected substrings are present
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("BuildSystemPrompt output missing expected substring:\nwant: %q\ngot: %s", want, got)
				}
			}
		})
	}
}

func TestBuildSystemPrompt_Structure(t *testing.T) {
	grantID := "test-grant"
	agentType := AgentClaude
	prompt := BuildSystemPrompt(grantID, agentType)

	// Verify key sections are present in order
	sections := []string{
		"You are a helpful email and calendar assistant",
		"Grant ID:",
		"## Tool Usage",
		"## Conversation Context",
		"## Response Format",
	}

	lastIndex := -1
	for _, section := range sections {
		index := strings.Index(prompt, section)
		if index == -1 {
			t.Errorf("Missing section in prompt: %q", section)
			continue
		}
		if index <= lastIndex {
			t.Errorf("Section %q appears before previous section (out of order)", section)
		}
		lastIndex = index
	}
}
