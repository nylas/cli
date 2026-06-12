package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nylas/cli/internal/domain"
)

// OllamaClient implements LLMProvider for Ollama.
type OllamaClient struct {
	*BaseClient
}

// NewOllamaClient creates a new Ollama client.
// Returns nil if config is nil - callers should validate config before calling.
func NewOllamaClient(config *domain.OllamaConfig) *OllamaClient {
	if config == nil {
		return nil
	}

	return &OllamaClient{
		BaseClient: NewBaseClient(
			"", // No API key for local Ollama
			config.Model,
			config.Host,
			0, // Use default timeout
		),
	}
}

// Name returns the provider name.
func (c *OllamaClient) Name() string {
	return "ollama"
}

// IsAvailable checks if Ollama is accessible.
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}

// Chat sends a chat completion request.
func (c *OllamaClient) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	return c.ChatWithTools(ctx, req, nil)
}

// ChatWithTools sends a chat request with function calling.
// Note: Ollama's tool support may vary by model.
func (c *OllamaClient) ChatWithTools(ctx context.Context, req *domain.ChatRequest, tools []domain.Tool) (*domain.ChatResponse, error) {
	// Prepare Ollama request
	ollamaReq := map[string]any{
		"model":    c.GetModel(req.Model),
		"messages": ConvertMessagesToMaps(req.Messages),
		"stream":   false,
	}

	if req.MaxTokens > 0 {
		ollamaReq["options"] = map[string]any{
			"num_predict": req.MaxTokens,
		}
	}

	if req.Temperature > 0 {
		if options, ok := ollamaReq["options"].(map[string]any); ok {
			options["temperature"] = req.Temperature
		} else {
			ollamaReq["options"] = map[string]any{
				"temperature": req.Temperature,
			}
		}
	}

	// Tools support (if model supports it)
	if len(tools) > 0 {
		ollamaReq["tools"] = ConvertToolsOpenAIFormat(tools)
	}

	// Send request using base client
	var ollamaResp struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		Model string `json:"model"`
		Done  bool   `json:"done"`
	}

	if err := c.DoJSONRequestAndDecode(ctx, "POST", "/api/chat", ollamaReq, nil, &ollamaResp); err != nil {
		return nil, err
	}

	response := &domain.ChatResponse{
		Content:  ollamaResp.Message.Content,
		Model:    ollamaResp.Model,
		Provider: "ollama",
		Usage: domain.TokenUsage{
			// Ollama doesn't always provide token counts
			TotalTokens: 0,
		},
	}

	// Convert tool calls if present
	for _, tc := range ollamaResp.Message.ToolCalls {
		response.ToolCalls = append(response.ToolCalls, domain.ToolCall{
			Function:  tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return response, nil
}

// StreamChat streams chat responses.
func (c *OllamaClient) StreamChat(ctx context.Context, req *domain.ChatRequest, callback func(chunk string) error) error {
	// Prepare Ollama request
	ollamaReq := map[string]any{
		"model":    c.GetModel(req.Model),
		"messages": ConvertMessagesToMaps(req.Messages),
		"stream":   true,
	}

	if req.Temperature > 0 {
		ollamaReq["options"] = map[string]any{
			"temperature": req.Temperature,
		}
	}

	// Send streaming request using base client
	resp, err := c.DoJSONRequest(ctx, "POST", "/api/chat", ollamaReq, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError(resp)
	}

	// Stream response
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode chunk: %w", err)
		}

		if chunk.Message.Content != "" {
			if err := callback(chunk.Message.Content); err != nil {
				return err
			}
		}

		if chunk.Done {
			break
		}
	}

	return nil
}
