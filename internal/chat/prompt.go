package chat

import "strings"

// BuildSystemPrompt constructs the system prompt for the AI agent.
// It includes identity, available tools, and the text-based tool protocol.
func BuildSystemPrompt(grantID string, agentType AgentType, hasSlack bool) string {
	var sb strings.Builder

	if hasSlack {
		sb.WriteString("You are a helpful email, calendar, and Slack assistant powered by the Nylas API.\n")
		sb.WriteString("You help users manage their emails, calendar events, contacts, and Slack messages.\n\n")
	} else {
		sb.WriteString("You are a helpful email and calendar assistant powered by the Nylas API.\n")
		sb.WriteString("You help users manage their emails, calendar events, and contacts.\n\n")
	}

	sb.WriteString("Grant ID: " + grantID + "\n\n")

	// Tool protocol instructions
	sb.WriteString("## Tool Usage\n\n")
	if hasSlack {
		sb.WriteString("When you need to access the user's email, calendar, contacts, or Slack, use the tools below.\n")
	} else {
		sb.WriteString("When you need to access the user's email, calendar, or contacts, use the tools below.\n")
	}
	sb.WriteString("To call a tool, output EXACTLY this format on its own line:\n\n")
	sb.WriteString("TOOL_CALL: {\"name\": \"tool_name\", \"args\": {\"param\": \"value\"}}\n\n")
	sb.WriteString("IMPORTANT RULES:\n")
	sb.WriteString("1. When you output a TOOL_CALL, output ONLY the TOOL_CALL line and nothing else.\n")
	sb.WriteString("   Do NOT include any other text before or after the TOOL_CALL line.\n")
	sb.WriteString("2. After you receive tool results (TOOL_RESULT), use the data to answer the user.\n")
	sb.WriteString("   Summarize and format the results clearly — do NOT make another tool call\n")
	sb.WriteString("   unless you need different or additional data.\n")
	sb.WriteString("3. Only make one TOOL_CALL per response. Wait for the result before proceeding.\n\n")

	// Tool definitions
	sb.WriteString(FormatToolsForPrompt(AvailableTools(hasSlack)))
	sb.WriteString("\n")

	// Context instructions
	sb.WriteString("## Conversation Context\n\n")
	sb.WriteString("You have access to a conversation history. If a summary of earlier messages\n")
	sb.WriteString("is provided, use it to maintain continuity. Reference previous topics naturally.\n\n")

	// Formatting instructions
	sb.WriteString("## Response Format\n\n")
	sb.WriteString("- Use markdown formatting for readability\n")
	sb.WriteString("- Present email lists as numbered items with sender, subject, and date\n")
	sb.WriteString("- Present calendar events with time, title, and attendees\n")
	if hasSlack {
		sb.WriteString("- Present Slack messages with username, timestamp, and content\n")
	}
	sb.WriteString("- Keep responses concise but informative\n")
	sb.WriteString("- If an error occurs, explain it clearly and suggest alternatives\n")

	return sb.String()
}
