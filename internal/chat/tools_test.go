package chat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolCalls(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedCalls  int
		expectedText   string
		validateResult func(t *testing.T, calls []ToolCall, text string)
	}{
		{
			name:          "no tool calls",
			output:        "Just a regular response\nWith multiple lines",
			expectedCalls: 0,
			expectedText:  "Just a regular response\nWith multiple lines",
		},
		{
			name: "single tool call",
			output: `Let me search for emails.
TOOL_CALL: {"name":"search_emails","args":{"query":"budget","limit":5}}
I'll search for budget-related emails.`,
			expectedCalls: 1,
			expectedText:  "Let me search for emails.\nI'll search for budget-related emails.",
			validateResult: func(t *testing.T, calls []ToolCall, text string) {
				require.Len(t, calls, 1)
				assert.Equal(t, "search_emails", calls[0].Name)
				assert.Equal(t, "budget", calls[0].Args["query"])
				assert.Equal(t, float64(5), calls[0].Args["limit"])
			},
		},
		{
			name: "multiple tool calls",
			output: `I'll list your emails first.
TOOL_CALL: {"name":"list_emails","args":{"limit":10}}
Then I'll check your calendar.
TOOL_CALL: {"name":"list_events","args":{"limit":5}}
All done!`,
			expectedCalls: 2,
			expectedText:  "I'll list your emails first.\nThen I'll check your calendar.\nAll done!",
			validateResult: func(t *testing.T, calls []ToolCall, text string) {
				require.Len(t, calls, 2)
				assert.Equal(t, "list_emails", calls[0].Name)
				assert.Equal(t, "list_events", calls[1].Name)
			},
		},
		{
			name:          "malformed JSON - kept as text",
			output:        "TOOL_CALL: {invalid json}\nValid text here",
			expectedCalls: 0,
			expectedText:  "TOOL_CALL: {invalid json}\nValid text here",
		},
		{
			name:          "tool call with leading whitespace",
			output:        "   TOOL_CALL: {\"name\":\"list_folders\",\"args\":{}}\nNext line",
			expectedCalls: 1,
			expectedText:  "Next line",
			validateResult: func(t *testing.T, calls []ToolCall, text string) {
				require.Len(t, calls, 1)
				assert.Equal(t, "list_folders", calls[0].Name)
			},
		},
		{
			name:          "empty output",
			output:        "",
			expectedCalls: 0,
			expectedText:  "",
		},
		{
			name:          "only tool calls, no text",
			output:        "TOOL_CALL: {\"name\":\"list_contacts\",\"args\":{\"limit\":20}}",
			expectedCalls: 1,
			expectedText:  "",
			validateResult: func(t *testing.T, calls []ToolCall, text string) {
				require.Len(t, calls, 1)
				assert.Equal(t, "list_contacts", calls[0].Name)
			},
		},
		{
			name: "tool call with complex args",
			output: `Sending email now.
TOOL_CALL: {"name":"send_email","args":{"to":"test@example.com","subject":"Test","body":"Hello\nWorld"}}
Email sent!`,
			expectedCalls: 1,
			expectedText:  "Sending email now.\nEmail sent!",
			validateResult: func(t *testing.T, calls []ToolCall, text string) {
				require.Len(t, calls, 1)
				assert.Equal(t, "send_email", calls[0].Name)
				assert.Equal(t, "test@example.com", calls[0].Args["to"])
				assert.Equal(t, "Test", calls[0].Args["subject"])
				assert.Equal(t, "Hello\nWorld", calls[0].Args["body"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, text := ParseToolCalls(tt.output)
			assert.Len(t, calls, tt.expectedCalls, "unexpected number of tool calls")
			assert.Equal(t, tt.expectedText, text, "unexpected remaining text")

			if tt.validateResult != nil {
				tt.validateResult(t, calls, text)
			}
		})
	}
}

func TestFormatToolResult(t *testing.T) {
	tests := []struct {
		name           string
		result         ToolResult
		expectedPrefix string
		validateJSON   func(t *testing.T, output string)
	}{
		{
			name: "success with data",
			result: ToolResult{
				Name: "list_emails",
				Data: map[string]any{
					"emails": []string{"email1", "email2"},
					"count":  2,
				},
			},
			expectedPrefix: "TOOL_RESULT: ",
			validateJSON: func(t *testing.T, output string) {
				assert.Contains(t, output, `"name":"list_emails"`)
				assert.Contains(t, output, `"data"`)
				assert.NotContains(t, output, `"error"`)
			},
		},
		{
			name: "error result",
			result: ToolResult{
				Name:  "send_email",
				Error: "failed to send: network timeout",
			},
			expectedPrefix: "TOOL_RESULT: ",
			validateJSON: func(t *testing.T, output string) {
				assert.Contains(t, output, `"name":"send_email"`)
				assert.Contains(t, output, `"error":"failed to send: network timeout"`)
			},
		},
		{
			name: "both data and error",
			result: ToolResult{
				Name:  "read_email",
				Data:  map[string]string{"partial": "data"},
				Error: "incomplete read",
			},
			expectedPrefix: "TOOL_RESULT: ",
			validateJSON: func(t *testing.T, output string) {
				assert.Contains(t, output, `"name":"read_email"`)
				assert.Contains(t, output, `"data"`)
				assert.Contains(t, output, `"error"`)
			},
		},
		{
			name: "empty data",
			result: ToolResult{
				Name: "list_folders",
			},
			expectedPrefix: "TOOL_RESULT: ",
			validateJSON: func(t *testing.T, output string) {
				assert.Contains(t, output, `"name":"list_folders"`)
			},
		},
		{
			name: "data with special characters",
			result: ToolResult{
				Name: "search_emails",
				Data: map[string]string{
					"subject": "Test \"quoted\" & <special>",
				},
			},
			expectedPrefix: "TOOL_RESULT: ",
			validateJSON: func(t *testing.T, output string) {
				assert.Contains(t, output, `"name":"search_emails"`)
				assert.Contains(t, output, `\u003cspecial\u003e`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatToolResult(tt.result)
			assert.True(t, strings.HasPrefix(output, tt.expectedPrefix), "output should start with TOOL_RESULT:")

			if tt.validateJSON != nil {
				tt.validateJSON(t, output)
			}
		})
	}
}

func TestAvailableTools(t *testing.T) {
	tools := AvailableTools()

	t.Run("returns correct tool count", func(t *testing.T) {
		// Expecting: list_emails, read_email, search_emails, send_email,
		// list_events, create_event, list_contacts, list_folders
		assert.Equal(t, 8, len(tools), "expected 8 tools")
	})

	t.Run("contains expected tool names", func(t *testing.T) {
		expectedTools := []string{
			"list_emails", "read_email", "search_emails", "send_email",
			"list_events", "create_event", "list_contacts", "list_folders",
		}

		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name] = true
		}

		for _, expected := range expectedTools {
			assert.True(t, toolNames[expected], "tool %s should be available", expected)
		}
	})

	t.Run("all tools have descriptions", func(t *testing.T) {
		for _, tool := range tools {
			assert.NotEmpty(t, tool.Description, "tool %s should have a description", tool.Name)
		}
	})

	t.Run("required parameters are marked", func(t *testing.T) {
		// Find send_email tool
		var sendEmail *Tool
		for i, tool := range tools {
			if tool.Name == "send_email" {
				sendEmail = &tools[i]
				break
			}
		}
		require.NotNil(t, sendEmail, "send_email tool should exist")

		// Check required parameters
		requiredParams := []string{"to", "subject", "body"}
		for _, param := range sendEmail.Parameters {
			if contains(requiredParams, param.Name) {
				assert.True(t, param.Required, "parameter %s should be required", param.Name)
			}
		}
	})

	t.Run("all parameters have types", func(t *testing.T) {
		for _, tool := range tools {
			for _, param := range tool.Parameters {
				assert.NotEmpty(t, param.Type, "parameter %s in tool %s should have a type", param.Name, tool.Name)
			}
		}
	})
}

func TestFormatToolsForPrompt(t *testing.T) {
	tools := []Tool{
		{
			Name:        "test_tool",
			Description: "A test tool for testing",
			Parameters: []ToolParameter{
				{Name: "arg1", Type: "string", Description: "First argument", Required: true},
				{Name: "arg2", Type: "number", Description: "Second argument", Required: false},
			},
		},
		{
			Name:        "simple_tool",
			Description: "A tool with no parameters",
			Parameters:  []ToolParameter{},
		},
	}

	output := FormatToolsForPrompt(tools)

	t.Run("contains header", func(t *testing.T) {
		assert.Contains(t, output, "Available tools:")
	})

	t.Run("contains all tool names", func(t *testing.T) {
		assert.Contains(t, output, "test_tool")
		assert.Contains(t, output, "simple_tool")
	})

	t.Run("contains tool descriptions", func(t *testing.T) {
		assert.Contains(t, output, "A test tool for testing")
		assert.Contains(t, output, "A tool with no parameters")
	})

	t.Run("contains parameter details", func(t *testing.T) {
		assert.Contains(t, output, "arg1")
		assert.Contains(t, output, "string")
		assert.Contains(t, output, "First argument")
		assert.Contains(t, output, "(required)")
	})

	t.Run("shows optional parameters", func(t *testing.T) {
		assert.Contains(t, output, "arg2")
		assert.Contains(t, output, "number")
		assert.Contains(t, output, "Second argument")
		// Should NOT contain "(required)" for arg2
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "arg2") {
				assert.NotContains(t, line, "(required)", "arg2 should not be marked as required")
			}
		}
	})

	t.Run("tool with no parameters omits Parameters section", func(t *testing.T) {
		// Find the simple_tool in the output
		lines := strings.Split(output, "\n")
		foundSimpleTool := false
		hasParameters := false

		for i, line := range lines {
			if strings.Contains(line, "simple_tool") {
				foundSimpleTool = true
				// Check next few lines for "Parameters:"
				for j := i + 1; j < len(lines) && j < i+5; j++ {
					if strings.Contains(lines[j], "- **") {
						// Hit next tool
						break
					}
					if strings.Contains(lines[j], "Parameters:") {
						hasParameters = true
						break
					}
				}
				break
			}
		}

		assert.True(t, foundSimpleTool, "should find simple_tool in output")
		assert.False(t, hasParameters, "simple_tool should not have Parameters section")
	})
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
