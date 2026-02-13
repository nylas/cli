package chat

import (
	"encoding/json"
	"strings"
)

// Tool represents a tool available to the AI agent.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters"`
}

// ToolParameter describes a parameter for a tool.
type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ToolCall represents a parsed tool call from agent output.
type ToolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ToolResult represents the result of executing a tool call.
type ToolResult struct {
	Name  string `json:"name"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// toolCallPrefix is the marker agents use to invoke tools.
const toolCallPrefix = "TOOL_CALL:"

// toolResultPrefix is the marker for returning results.
const toolResultPrefix = "TOOL_RESULT:"

// AvailableTools returns the tools exposed to AI agents.
func AvailableTools() []Tool {
	return []Tool{
		{
			Name:        "list_emails",
			Description: "List recent emails from the user's inbox",
			Parameters: []ToolParameter{
				{Name: "limit", Type: "number", Description: "Max emails to return (default 10)", Required: false},
				{Name: "subject", Type: "string", Description: "Filter by subject keyword", Required: false},
				{Name: "from", Type: "string", Description: "Filter by sender email", Required: false},
				{Name: "unread", Type: "boolean", Description: "Only show unread emails", Required: false},
			},
		},
		{
			Name:        "read_email",
			Description: "Read a specific email by ID to see full body content",
			Parameters: []ToolParameter{
				{Name: "id", Type: "string", Description: "The email/message ID", Required: true},
			},
		},
		{
			Name:        "search_emails",
			Description: "Search emails by query string",
			Parameters: []ToolParameter{
				{Name: "query", Type: "string", Description: "Search query (e.g. 'from:sarah budget')", Required: true},
				{Name: "limit", Type: "number", Description: "Max results (default 10)", Required: false},
			},
		},
		{
			Name:        "send_email",
			Description: "Send a new email",
			Parameters: []ToolParameter{
				{Name: "to", Type: "string", Description: "Recipient email address", Required: true},
				{Name: "subject", Type: "string", Description: "Email subject line", Required: true},
				{Name: "body", Type: "string", Description: "Email body (plain text)", Required: true},
			},
		},
		{
			Name:        "list_events",
			Description: "List upcoming calendar events",
			Parameters: []ToolParameter{
				{Name: "limit", Type: "number", Description: "Max events to return (default 10)", Required: false},
				{Name: "calendar_id", Type: "string", Description: "Calendar ID (default: primary)", Required: false},
			},
		},
		{
			Name:        "create_event",
			Description: "Create a new calendar event",
			Parameters: []ToolParameter{
				{Name: "title", Type: "string", Description: "Event title", Required: true},
				{Name: "start_time", Type: "string", Description: "Start time (RFC3339, e.g. 2026-02-12T14:00:00Z)", Required: true},
				{Name: "end_time", Type: "string", Description: "End time (RFC3339)", Required: true},
				{Name: "calendar_id", Type: "string", Description: "Calendar ID (default: primary)", Required: false},
				{Name: "description", Type: "string", Description: "Event description", Required: false},
			},
		},
		{
			Name:        "list_contacts",
			Description: "List contacts from the address book",
			Parameters: []ToolParameter{
				{Name: "limit", Type: "number", Description: "Max contacts to return (default 10)", Required: false},
				{Name: "query", Type: "string", Description: "Search by name or email", Required: false},
			},
		},
		{
			Name:        "list_folders",
			Description: "List email folders/labels",
			Parameters:  []ToolParameter{},
		},
	}
}

// ParseToolCalls extracts TOOL_CALL: lines from agent output.
// Returns parsed tool calls and the remaining text (non-tool-call content).
func ParseToolCalls(output string) ([]ToolCall, string) {
	var calls []ToolCall
	var textParts []string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, toolCallPrefix) {
			jsonStr := strings.TrimSpace(strings.TrimPrefix(trimmed, toolCallPrefix))
			var call ToolCall
			if err := json.Unmarshal([]byte(jsonStr), &call); err == nil {
				calls = append(calls, call)
				continue
			}
		}
		textParts = append(textParts, line)
	}

	return calls, strings.TrimSpace(strings.Join(textParts, "\n"))
}

// FormatToolResult formats a tool result for injection back into the prompt.
func FormatToolResult(result ToolResult) string {
	data, err := json.Marshal(result)
	if err != nil {
		return toolResultPrefix + " " + `{"error":"failed to marshal result"}`
	}
	return toolResultPrefix + " " + string(data)
}

// FormatToolsForPrompt generates the tool description section for the system prompt.
func FormatToolsForPrompt(tools []Tool) string {
	var sb strings.Builder
	sb.WriteString("Available tools:\n\n")

	for _, t := range tools {
		sb.WriteString("- **" + t.Name + "**: " + t.Description + "\n")
		if len(t.Parameters) > 0 {
			sb.WriteString("  Parameters:\n")
			for _, p := range t.Parameters {
				req := ""
				if p.Required {
					req = " (required)"
				}
				sb.WriteString("    - " + p.Name + " (" + p.Type + "): " + p.Description + req + "\n")
			}
		}
	}

	return sb.String()
}
