package chat

import (
	"context"
	"strings"
)

const (
	// compactionThreshold is the number of user+assistant messages before compaction.
	compactionThreshold = 30

	// compactionWindow is the number of recent messages to keep after compaction.
	compactionWindow = 15
)

// ContextBuilder constructs prompts with conversation context and manages compaction.
type ContextBuilder struct {
	agent   *Agent
	memory  *MemoryStore
	grantID string
}

// NewContextBuilder creates a new ContextBuilder.
func NewContextBuilder(agent *Agent, memory *MemoryStore, grantID string) *ContextBuilder {
	return &ContextBuilder{
		agent:   agent,
		memory:  memory,
		grantID: grantID,
	}
}

// BuildPrompt constructs the full prompt for the agent including:
// 1. System prompt (identity + tools + instructions)
// 2. Conversation summary (if compacted)
// 3. Recent messages (within context window)
// 4. Latest user message
func (c *ContextBuilder) BuildPrompt(conv *Conversation, newMessage string) string {
	var sb strings.Builder

	// System prompt
	sb.WriteString(BuildSystemPrompt(c.grantID, c.agent.Type))
	sb.WriteString("\n---\n\n")

	// Include conversation summary if available
	if conv.Summary != "" {
		sb.WriteString("## Previous Conversation Summary\n\n")
		sb.WriteString(conv.Summary)
		sb.WriteString("\n\n---\n\n")
	}

	// Include recent messages
	sb.WriteString("## Conversation\n\n")
	for _, msg := range conv.Messages {
		switch msg.Role {
		case "user":
			sb.WriteString("User: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "assistant":
			sb.WriteString("Assistant: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "tool_call":
			sb.WriteString(toolCallPrefix + " ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "tool_result":
			sb.WriteString(toolResultPrefix + " ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}

	// Add new user message
	sb.WriteString("User: ")
	sb.WriteString(newMessage)
	sb.WriteString("\n\nAssistant: ")

	return sb.String()
}

// NeedsCompaction checks if the conversation should be compacted.
// Returns true when there are more than compactionThreshold user+assistant messages.
func (c *ContextBuilder) NeedsCompaction(conv *Conversation) bool {
	count := 0
	for _, msg := range conv.Messages {
		if msg.Role == "user" || msg.Role == "assistant" {
			count++
		}
	}
	return count > compactionThreshold
}

// Compact summarizes older messages and trims the conversation.
// It keeps the most recent compactionWindow messages and summarizes the rest.
func (c *ContextBuilder) Compact(ctx context.Context, conv *Conversation) error {
	if !c.NeedsCompaction(conv) {
		return nil
	}

	// Find the split point: keep last compactionWindow user+assistant messages
	splitIdx := c.findSplitIndex(conv)
	if splitIdx <= 0 {
		return nil
	}

	// Build the older messages into text for summarization
	var older strings.Builder
	for _, msg := range conv.Messages[:splitIdx] {
		if msg.Role == "user" || msg.Role == "assistant" {
			older.WriteString(msg.Role + ": " + msg.Content + "\n")
		}
	}

	// Ask the agent to summarize
	prompt := "Summarize this conversation so far in 3-4 sentences. " +
		"Preserve key facts, names, email IDs, dates, and any commitments made.\n\n" +
		older.String()

	summary, err := c.agent.Run(ctx, prompt)
	if err != nil {
		return err
	}

	// Merge with existing summary if present
	if conv.Summary != "" {
		summary = conv.Summary + "\n\n" + summary
	}

	// Update memory store with summary and trim messages
	return c.memory.UpdateSummary(conv.ID, summary, splitIdx)
}

// findSplitIndex finds the index to split at, keeping the last compactionWindow
// user+assistant messages intact.
func (c *ContextBuilder) findSplitIndex(conv *Conversation) int {
	count := 0
	for _, msg := range conv.Messages {
		if msg.Role == "user" || msg.Role == "assistant" {
			count++
		}
	}

	keep := compactionWindow
	if count <= keep {
		return 0
	}

	// Walk backwards to find where to split
	seen := 0
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		if conv.Messages[i].Role == "user" || conv.Messages[i].Role == "assistant" {
			seen++
		}
		if seen >= keep {
			return i
		}
	}

	return 0
}
