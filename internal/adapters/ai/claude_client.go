package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

// ClaudeClient implements LLMProvider for Anthropic Claude.
type ClaudeClient struct {
	*BaseClient
}

// NewClaudeClient creates a new Claude client.
func NewClaudeClient(config *domain.ClaudeConfig) *ClaudeClient {
	if config == nil {
		config = &domain.ClaudeConfig{
			Model: "claude-3-5-sonnet-20241022",
		}
	}

	apiKey := GetAPIKeyFromEnv(config.APIKey, "ANTHROPIC_API_KEY")

	return &ClaudeClient{
		BaseClient: NewBaseClient(
			apiKey,
			config.Model,
			"https://api.anthropic.com/v1",
			0, // Use default timeout
		),
	}
}

// Name returns the provider name.
func (c *ClaudeClient) Name() string {
	return "claude"
}

// IsAvailable checks if Claude API key is configured.
func (c *ClaudeClient) IsAvailable(ctx context.Context) bool {
	return c.IsConfigured()
}

// Chat sends a chat completion request.
func (c *ClaudeClient) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	return c.ChatWithTools(ctx, req, nil)
}

// ChatWithTools sends a chat request with function calling.
func (c *ClaudeClient) ChatWithTools(ctx context.Context, req *domain.ChatRequest, tools []domain.Tool) (*domain.ChatResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("claude API key not configured")
	}

	// Prepare Claude request
	system, messages := c.extractSystemMessage(req.Messages)

	claudeReq := map[string]any{
		"model":      c.GetModel(req.Model),
		"messages":   c.convertMessages(messages),
		"max_tokens": c.getMaxTokens(req.MaxTokens),
	}

	if system != "" {
		claudeReq["system"] = system
	}

	if req.Temperature > 0 {
		claudeReq["temperature"] = req.Temperature
	}

	// Tools support
	if len(tools) > 0 {
		claudeReq["tools"] = c.convertTools(tools)
	}

	// Send request using base client with Claude-specific headers
	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": "2023-06-01",
	}

	var claudeResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text,omitempty"`
			ID    string `json:"id,omitempty"`
			Name  string `json:"name,omitempty"`
			Input any    `json:"input,omitempty"`
		} `json:"content"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := c.DoJSONRequestAndDecode(ctx, "POST", "/messages", claudeReq, headers, &claudeResp); err != nil {
		return nil, err
	}

	response := &domain.ChatResponse{
		Model:    claudeResp.Model,
		Provider: "claude",
		Usage: domain.TokenUsage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls
	for _, content := range claudeResp.Content {
		switch content.Type {
		case "text":
			response.Content += content.Text
		case "tool_use":
			// Convert tool use to map[string]any
			args, ok := content.Input.(map[string]any)
			if !ok {
				// Best-effort conversion: re-serialize and deserialize to get map[string]any.
				// json.Marshal can't fail on a valid any from API response.
				// Unmarshal errors ignored - empty args is acceptable fallback.
				inputBytes, _ := json.Marshal(content.Input)
				_ = json.Unmarshal(inputBytes, &args)
			}

			response.ToolCalls = append(response.ToolCalls, domain.ToolCall{
				ID:        content.ID,
				Function:  content.Name,
				Arguments: args,
			})
		}
	}

	return response, nil
}

// StreamChat streams chat responses.
func (c *ClaudeClient) StreamChat(ctx context.Context, req *domain.ChatRequest, callback func(chunk string) error) error {
	// Claude streaming requires SSE handling, simplified version here
	if !c.IsConfigured() {
		return fmt.Errorf("claude API key not configured")
	}

	system, messages := c.extractSystemMessage(req.Messages)

	claudeReq := map[string]any{
		"model":      c.GetModel(req.Model),
		"messages":   c.convertMessages(messages),
		"max_tokens": c.getMaxTokens(req.MaxTokens),
		"stream":     true,
	}

	if system != "" {
		claudeReq["system"] = system
	}

	if req.Temperature > 0 {
		claudeReq["temperature"] = req.Temperature
	}

	// Send streaming request using base client
	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": "2023-06-01",
	}

	resp, err := c.DoJSONRequest(ctx, "POST", "/messages", claudeReq, headers)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Simple SSE parsing (production would use proper SSE library)
	scanner := &sseScanner{reader: resp.Body}
	for scanner.Scan() {
		event := scanner.Event()
		if event.Type == "content_block_delta" {
			if delta, ok := event.Data["delta"].(map[string]any); ok {
				if text, ok := delta["text"].(string); ok && text != "" {
					if err := callback(text); err != nil {
						return err
					}
				}
			}
		}
	}

	return scanner.Err()
}

// Helper methods

func (c *ClaudeClient) getMaxTokens(requestMaxTokens int) int {
	if requestMaxTokens > 0 {
		return requestMaxTokens
	}
	return 4096 // Default
}

func (c *ClaudeClient) extractSystemMessage(messages []domain.ChatMessage) (string, []domain.ChatMessage) {
	var system string
	var filtered []domain.ChatMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			system = msg.Content
		} else {
			filtered = append(filtered, msg)
		}
	}

	return system, filtered
}

func (c *ClaudeClient) convertMessages(messages []domain.ChatMessage) []map[string]string {
	result := make([]map[string]string, 0, len(messages))
	for _, msg := range messages {
		if msg.Role != "system" { // System already extracted
			result = append(result, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}
	return result
}

func (c *ClaudeClient) convertTools(tools []domain.Tool) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, tool := range tools {
		result[i] = map[string]any{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.Parameters,
		}
	}
	return result
}

// Simple SSE event scanner
type sseEvent struct {
	Type string
	Data map[string]any
}

type sseScanner struct {
	reader io.Reader
	err    error
	event  sseEvent
}

func (s *sseScanner) Scan() bool {
	buf := make([]byte, 4096)
	n, err := s.reader.Read(buf)
	if err != nil {
		if err != io.EOF {
			s.err = err
		}
		return false
	}

	// Simplified SSE parsing
	data := string(buf[:n])
	if strings.Contains(data, "data: {") {
		start := strings.Index(data, "{")
		end := strings.LastIndex(data, "}")
		if start >= 0 && end > start {
			jsonData := data[start : end+1]
			var evt map[string]any
			if err := json.Unmarshal([]byte(jsonData), &evt); err == nil {
				if t, ok := evt["type"].(string); ok {
					s.event = sseEvent{Type: t, Data: evt}
					return true
				}
			}
		}
	}

	return false
}

func (s *sseScanner) Event() sseEvent {
	return s.event
}

func (s *sseScanner) Err() error {
	return s.err
}
