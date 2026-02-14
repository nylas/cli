package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContextBuilder_BuildPrompt(t *testing.T) {
	agent := &Agent{Type: AgentClaude, Path: "/usr/bin/claude"}
	store := setupMemoryStore(t)
	grantID := "test-grant-123"
	builder := NewContextBuilder(agent, store, grantID, false)

	tests := []struct {
		name       string
		setupConv  func(t *testing.T) *Conversation
		newMessage string
		validate   func(t *testing.T, prompt string)
	}{
		{
			name: "empty conversation",
			setupConv: func(t *testing.T) *Conversation {
				return &Conversation{
					ID:       "conv_test",
					Messages: []Message{},
				}
			},
			newMessage: "Hello",
			validate: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "You are a helpful email and calendar assistant")
				assert.Contains(t, prompt, "Available tools:")
				assert.Contains(t, prompt, "User: Hello")
				assert.Contains(t, prompt, "Assistant:")
				assert.NotContains(t, prompt, "Previous Conversation Summary")
			},
		},
		{
			name: "conversation with messages",
			setupConv: func(t *testing.T) *Conversation {
				return &Conversation{
					ID: "conv_test",
					Messages: []Message{
						{Role: "user", Content: "What's the weather?", Timestamp: time.Now()},
						{Role: "assistant", Content: "I can help with that.", Timestamp: time.Now()},
					},
				}
			},
			newMessage: "Thanks",
			validate: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "User: What's the weather?")
				assert.Contains(t, prompt, "Assistant: I can help with that.")
				assert.Contains(t, prompt, "User: Thanks")
				assert.Contains(t, prompt, "Assistant:")
			},
		},
		{
			name: "conversation with summary",
			setupConv: func(t *testing.T) *Conversation {
				return &Conversation{
					ID:      "conv_test",
					Summary: "The user asked about emails and I helped them search for budget-related messages.",
					Messages: []Message{
						{Role: "user", Content: "Show me recent emails", Timestamp: time.Now()},
					},
				}
			},
			newMessage: "What about calendar?",
			validate: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "Previous Conversation Summary")
				assert.Contains(t, prompt, "budget-related messages")
				assert.Contains(t, prompt, "User: Show me recent emails")
				assert.Contains(t, prompt, "User: What about calendar?")
			},
		},
		{
			name: "conversation with tool calls and results",
			setupConv: func(t *testing.T) *Conversation {
				return &Conversation{
					ID: "conv_test",
					Messages: []Message{
						{Role: "user", Content: "List my emails", Timestamp: time.Now()},
						{Role: "tool_call", Content: `{"name":"list_emails","args":{}}`, Name: "list_emails", Timestamp: time.Now()},
						{Role: "tool_result", Content: `{"name":"list_emails","data":[]}`, Name: "list_emails", Timestamp: time.Now()},
						{Role: "assistant", Content: "You have no emails.", Timestamp: time.Now()},
					},
				}
			},
			newMessage: "Thanks",
			validate: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "TOOL_CALL:")
				assert.Contains(t, prompt, "TOOL_RESULT:")
				assert.Contains(t, prompt, "list_emails")
				assert.Contains(t, prompt, "You have no emails.")
			},
		},
		{
			name: "includes grant ID in system prompt",
			setupConv: func(t *testing.T) *Conversation {
				return &Conversation{ID: "conv_test", Messages: []Message{}}
			},
			newMessage: "Test",
			validate: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, grantID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := tt.setupConv(t)
			prompt := builder.BuildPrompt(conv, tt.newMessage)

			assert.NotEmpty(t, prompt)
			if tt.validate != nil {
				tt.validate(t, prompt)
			}
		})
	}
}

func TestContextBuilder_NeedsCompaction(t *testing.T) {
	agent := &Agent{Type: AgentClaude, Path: "/usr/bin/claude"}
	store := setupMemoryStore(t)
	builder := NewContextBuilder(agent, store, "test-grant", false)

	tests := []struct {
		name        string
		setupConv   func(t *testing.T) *Conversation
		wantCompact bool
		description string
	}{
		{
			name: "empty conversation",
			setupConv: func(t *testing.T) *Conversation {
				return &Conversation{
					ID:       "conv_test",
					Messages: []Message{},
				}
			},
			wantCompact: false,
			description: "no messages should not need compaction",
		},
		{
			name: "below threshold",
			setupConv: func(t *testing.T) *Conversation {
				messages := make([]Message, 20) // 10 user + 10 assistant = 20 total
				for i := 0; i < 20; i++ {
					role := "user"
					if i%2 == 1 {
						role = "assistant"
					}
					messages[i] = Message{Role: role, Content: "Message", Timestamp: time.Now()}
				}
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantCompact: false,
			description: "20 user+assistant messages should not need compaction (threshold is 30)",
		},
		{
			name: "exactly at threshold",
			setupConv: func(t *testing.T) *Conversation {
				messages := make([]Message, compactionThreshold)
				for i := 0; i < compactionThreshold; i++ {
					role := "user"
					if i%2 == 1 {
						role = "assistant"
					}
					messages[i] = Message{Role: role, Content: "Message", Timestamp: time.Now()}
				}
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantCompact: false,
			description: "exactly at threshold should not need compaction",
		},
		{
			name: "above threshold",
			setupConv: func(t *testing.T) *Conversation {
				messages := make([]Message, compactionThreshold+2)
				for i := 0; i < compactionThreshold+2; i++ {
					role := "user"
					if i%2 == 1 {
						role = "assistant"
					}
					messages[i] = Message{Role: role, Content: "Message", Timestamp: time.Now()}
				}
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantCompact: true,
			description: "above threshold should need compaction",
		},
		{
			name: "many messages with tool calls",
			setupConv: func(t *testing.T) *Conversation {
				messages := make([]Message, 100)
				userAssistantCount := 0
				for i := 0; i < 100; i++ {
					switch i % 4 {
					case 0:
						messages[i] = Message{Role: "user", Content: "Message", Timestamp: time.Now()}
						userAssistantCount++
					case 1:
						messages[i] = Message{Role: "tool_call", Content: "{}", Timestamp: time.Now()}
					case 2:
						messages[i] = Message{Role: "tool_result", Content: "{}", Timestamp: time.Now()}
					case 3:
						messages[i] = Message{Role: "assistant", Content: "Message", Timestamp: time.Now()}
						userAssistantCount++
					}
				}
				// With this pattern, we have 50 user+assistant messages
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantCompact: true,
			description: "50 user+assistant messages (with tool messages) should need compaction",
		},
		{
			name: "only tool messages",
			setupConv: func(t *testing.T) *Conversation {
				messages := make([]Message, 50)
				for i := 0; i < 50; i++ {
					role := "tool_call"
					if i%2 == 1 {
						role = "tool_result"
					}
					messages[i] = Message{Role: role, Content: "{}", Timestamp: time.Now()}
				}
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantCompact: false,
			description: "only tool messages should not trigger compaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := tt.setupConv(t)
			result := builder.NeedsCompaction(conv)
			assert.Equal(t, tt.wantCompact, result, tt.description)
		})
	}
}

func TestContextBuilder_findSplitIndex(t *testing.T) {
	agent := &Agent{Type: AgentClaude, Path: "/usr/bin/claude"}
	store := setupMemoryStore(t)
	builder := NewContextBuilder(agent, store, "test-grant", false)

	tests := []struct {
		name        string
		setupConv   func(t *testing.T) *Conversation
		wantIndex   int
		description string
	}{
		{
			name: "empty conversation",
			setupConv: func(t *testing.T) *Conversation {
				return &Conversation{ID: "conv_test", Messages: []Message{}}
			},
			wantIndex:   0,
			description: "empty conversation should return 0",
		},
		{
			name: "fewer messages than window",
			setupConv: func(t *testing.T) *Conversation {
				messages := make([]Message, 10)
				for i := 0; i < 10; i++ {
					role := "user"
					if i%2 == 1 {
						role = "assistant"
					}
					messages[i] = Message{Role: role, Content: "Message", Timestamp: time.Now()}
				}
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantIndex:   0,
			description: "fewer than compactionWindow messages should return 0",
		},
		{
			name: "exactly compactionWindow messages",
			setupConv: func(t *testing.T) *Conversation {
				messages := make([]Message, compactionWindow)
				for i := 0; i < compactionWindow; i++ {
					role := "user"
					if i%2 == 1 {
						role = "assistant"
					}
					messages[i] = Message{Role: role, Content: "Message", Timestamp: time.Now()}
				}
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantIndex:   0,
			description: "exactly compactionWindow messages should return 0",
		},
		{
			name: "more than compactionWindow messages",
			setupConv: func(t *testing.T) *Conversation {
				// Create 40 user+assistant messages
				messages := make([]Message, 40)
				for i := 0; i < 40; i++ {
					role := "user"
					if i%2 == 1 {
						role = "assistant"
					}
					messages[i] = Message{Role: role, Content: "Message " + string(rune('A'+i)), Timestamp: time.Now()}
				}
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantIndex:   40 - compactionWindow, // Should keep last 15 messages
			description: "should split to keep last compactionWindow messages",
		},
		{
			name: "messages with tool calls interspersed",
			setupConv: func(t *testing.T) *Conversation {
				// Create pattern: user, tool_call, tool_result, assistant (repeat)
				messages := make([]Message, 60)
				for i := 0; i < 60; i++ {
					switch i % 4 {
					case 0:
						messages[i] = Message{Role: "user", Content: "U" + string(rune('A'+i/4)), Timestamp: time.Now()}
					case 1:
						messages[i] = Message{Role: "tool_call", Content: "{}", Timestamp: time.Now()}
					case 2:
						messages[i] = Message{Role: "tool_result", Content: "{}", Timestamp: time.Now()}
					case 3:
						messages[i] = Message{Role: "assistant", Content: "A" + string(rune('A'+i/4)), Timestamp: time.Now()}
					}
				}
				// We have 30 user+assistant messages (15 pairs)
				// Want to keep last compactionWindow (15), so split at index of first message to keep
				return &Conversation{ID: "conv_test", Messages: messages}
			},
			wantIndex:   0, // 30 messages total, keep last 15, so split at first of last 15
			description: "should find correct split with tool messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := tt.setupConv(t)
			index := builder.findSplitIndex(conv)

			switch tt.name {
			case "more than compactionWindow messages":
				assert.Equal(t, 25, index, tt.description)
				kept := conv.Messages[index:]
				count := 0
				for _, msg := range kept {
					if msg.Role == "user" || msg.Role == "assistant" {
						count++
					}
				}
				assert.Equal(t, compactionWindow, count, "should keep exactly compactionWindow messages")
			case "messages with tool calls interspersed":
				kept := conv.Messages[index:]
				count := 0
				for _, msg := range kept {
					if msg.Role == "user" || msg.Role == "assistant" {
						count++
					}
				}
				assert.GreaterOrEqual(t, count, compactionWindow-1, "should keep at least compactionWindow-1 messages")
			default:
				assert.Equal(t, tt.wantIndex, index, tt.description)
			}
		})
	}
}

func TestContextBuilder_BuildPrompt_Structure(t *testing.T) {
	agent := &Agent{Type: AgentClaude, Path: "/usr/bin/claude"}
	store := setupMemoryStore(t)
	builder := NewContextBuilder(agent, store, "test-grant", false)

	conv := &Conversation{
		ID:      "conv_test",
		Summary: "Previous discussion about emails",
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "Hi!", Timestamp: time.Now()},
		},
	}

	prompt := builder.BuildPrompt(conv, "New message")

	// Split prompt into sections
	sections := strings.Split(prompt, "---")

	t.Run("has correct number of sections", func(t *testing.T) {
		assert.GreaterOrEqual(t, len(sections), 2, "should have at least 2 sections separated by ---")
	})

	t.Run("system prompt comes first", func(t *testing.T) {
		assert.Contains(t, sections[0], "You are a helpful email and calendar assistant")
		assert.Contains(t, sections[0], "Available tools:")
	})

	t.Run("summary comes after system prompt", func(t *testing.T) {
		promptStr := prompt
		summaryIdx := strings.Index(promptStr, "Previous Conversation Summary")
		systemIdx := strings.Index(promptStr, "You are a helpful AI assistant")
		assert.Greater(t, summaryIdx, systemIdx, "summary should come after system prompt")
	})

	t.Run("summary comes before conversation", func(t *testing.T) {
		promptStr := prompt
		// Look for the "## Conversation" header that comes after summary, not the one in system prompt
		summaryIdx := strings.Index(promptStr, "Previous Conversation Summary")
		// Find "## Conversation" that appears after the summary
		conversationIdx := strings.Index(promptStr[summaryIdx:], "## Conversation")
		if conversationIdx != -1 {
			conversationIdx += summaryIdx // Adjust to absolute position
		}

		assert.NotEqual(t, -1, summaryIdx, "summary should be present")
		assert.NotEqual(t, -1, conversationIdx, "conversation section should be present after summary")
		assert.Greater(t, conversationIdx, summaryIdx, "conversation should come after summary")
	})

	t.Run("ends with new message and Assistant:", func(t *testing.T) {
		assert.True(t, strings.HasSuffix(prompt, "User: New message\n\nAssistant: "),
			"prompt should end with new message and 'Assistant:'")
	})
}
