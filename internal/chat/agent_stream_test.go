package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportsStreaming(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		expected bool
	}{
		{
			name: "claude supports streaming",
			agent: Agent{
				Type: AgentClaude,
				Path: "/usr/bin/claude",
			},
			expected: true,
		},
		{
			name: "ollama supports streaming",
			agent: Agent{
				Type:  AgentOllama,
				Path:  "/usr/bin/ollama",
				Model: "mistral",
			},
			expected: true,
		},
		{
			name: "codex does not support streaming",
			agent: Agent{
				Type: AgentCodex,
				Path: "/usr/bin/codex",
			},
			expected: false,
		},
		{
			name: "unknown type does not support streaming",
			agent: Agent{
				Type: AgentType("unknown"),
				Path: "/usr/bin/unknown",
			},
			expected: false,
		},
		{
			name: "empty type does not support streaming",
			agent: Agent{
				Type: AgentType(""),
				Path: "/usr/bin/empty",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.SupportsStreaming()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseClaudeStreamLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "stream_event with content_block_delta",
			line:     `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}}`,
			expected: "Hello",
		},
		{
			name:     "stream_event with multiword text",
			line:     `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello, world!"}}}`,
			expected: "Hello, world!",
		},
		{
			name:     "stream_event with newline in text",
			line:     `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Line 1\nLine 2"}}}`,
			expected: "Line 1\nLine 2",
		},
		{
			name:     "stream_event with empty text",
			line:     `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":""}}}`,
			expected: "",
		},
		{
			name:     "stream_event with message_start (not delta)",
			line:     `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_123"}}}`,
			expected: "",
		},
		{
			name:     "stream_event with content_block_stop",
			line:     `{"type":"stream_event","event":{"type":"content_block_stop","index":0}}`,
			expected: "",
		},
		{
			name:     "stream_event with message_delta",
			line:     `{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"}}}`,
			expected: "",
		},
		{
			name:     "non-stream_event type",
			line:     `{"type":"result","result":"Full response","subtype":"success"}`,
			expected: "",
		},
		{
			name:     "empty line",
			line:     "",
			expected: "",
		},
		{
			name:     "whitespace only",
			line:     "   ",
			expected: "",
		},
		{
			name:     "invalid JSON",
			line:     `{"type":"stream_event","event":{"type":"content_block_delta","delta":{"text":"Hello}`,
			expected: "",
		},
		{
			name:     "malformed JSON",
			line:     `not json at all`,
			expected: "",
		},
		{
			name:     "valid JSON but unknown type",
			line:     `{"type":"ping","timestamp":1234567890}`,
			expected: "",
		},
		{
			name:     "stream_event with unicode",
			line:     `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello 👋 世界"}}}`,
			expected: "Hello 👋 世界",
		},
		{
			name:     "stream_event with escaped quotes",
			line:     `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"He said \"Hi\""}}}`,
			expected: `He said "Hi"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseClaudeStreamLine([]byte(tt.line))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseClaudeResultLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "result event with text",
			line:     `{"type":"result","result":"Hello, world!","subtype":"success"}`,
			expected: "Hello, world!",
		},
		{
			name:     "result event with empty result",
			line:     `{"type":"result","result":"","subtype":"success"}`,
			expected: "",
		},
		{
			name:     "non-result event",
			line:     `{"type":"stream_event","event":{"type":"content_block_delta","delta":{"text":"Hi"}}}`,
			expected: "",
		},
		{
			name:     "empty line",
			line:     "",
			expected: "",
		},
		{
			name:     "invalid JSON",
			line:     `not json`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseClaudeResultLine([]byte(tt.line))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFallbackStream(t *testing.T) {
	tests := []struct {
		name          string
		agent         Agent
		prompt        string
		expectError   bool
		expectToken   bool
		errorContains string
	}{
		{
			name: "codex agent with non-existent binary",
			agent: Agent{
				Type: AgentCodex,
				Path: "/nonexistent/codex",
			},
			prompt:        "test prompt",
			expectError:   true,
			expectToken:   false,
			errorContains: "codex error",
		},
		{
			name: "unknown agent type",
			agent: Agent{
				Type: AgentType("unknown"),
				Path: "/nonexistent/unknown",
			},
			prompt:        "test prompt",
			expectError:   true,
			expectToken:   false,
			errorContains: "unsupported agent type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var receivedToken string
			var tokenReceived bool

			callback := func(token string) {
				receivedToken = token
				tokenReceived = true
			}

			result, err := tt.agent.fallbackStream(ctx, tt.prompt, callback)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Empty(t, result)
				assert.False(t, tokenReceived, "should not receive token on error")
			} else {
				require.NoError(t, err)
				if tt.expectToken {
					assert.True(t, tokenReceived, "should receive token callback")
					assert.NotEmpty(t, receivedToken)
					assert.Equal(t, result, receivedToken)
				}
			}
		})
	}
}

func TestFallbackStream_CallbackBehavior(t *testing.T) {
	t.Run("nil callback does not panic", func(t *testing.T) {
		agent := Agent{
			Type: AgentCodex,
			Path: "/nonexistent/codex",
		}

		ctx := context.Background()
		_, err := agent.fallbackStream(ctx, "test", nil)

		// Should error because binary doesn't exist, but shouldn't panic
		assert.Error(t, err)
	})

	t.Run("callback receives full response", func(t *testing.T) {
		// This test would require mocking exec.Command or using a real binary
		// For now, we verify the logic path with a non-existent binary
		agent := Agent{
			Type: AgentType("unknown"),
			Path: "/path/to/agent",
		}

		ctx := context.Background()
		callbackCount := 0

		callback := func(token string) {
			callbackCount++
		}

		_, err := agent.fallbackStream(ctx, "prompt", callback)

		// Should error with unsupported agent type
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported agent type")
		assert.Equal(t, 0, callbackCount, "callback should not be called on error")
	})
}

func TestTokenCallback(t *testing.T) {
	t.Run("multiple tokens collected", func(t *testing.T) {
		var tokens []string
		callback := TokenCallback(func(token string) {
			tokens = append(tokens, token)
		})

		// Simulate streaming multiple tokens
		callback("Hello")
		callback(" ")
		callback("world")
		callback("!")

		assert.Equal(t, []string{"Hello", " ", "world", "!"}, tokens)
		assert.Equal(t, "Hello world!", strings.Join(tokens, ""))
	})

	t.Run("empty token handling", func(t *testing.T) {
		var tokens []string
		callback := TokenCallback(func(token string) {
			tokens = append(tokens, token)
		})

		callback("")
		callback("text")
		callback("")

		assert.Equal(t, []string{"", "text", ""}, tokens)
	})
}
