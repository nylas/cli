package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// BaseClient provides common HTTP client functionality for AI providers.
type BaseClient struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string
}

// NewBaseClient creates a new base client with common configuration.
func NewBaseClient(apiKey, model, baseURL string, timeout time.Duration) *BaseClient {
	if timeout == 0 {
		timeout = domain.TimeoutAI
	}

	return &BaseClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// IsConfigured returns true if the API key is set.
func (b *BaseClient) IsConfigured() bool {
	return b.apiKey != ""
}

// GetModel returns the configured model or falls back to the provided default.
func (b *BaseClient) GetModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return b.model
}

// DoJSONRequest performs an HTTP request with JSON body and returns the response.
func (b *BaseClient) DoJSONRequest(ctx context.Context, method, endpoint string, body any, headers map[string]string) (*http.Response, error) {
	// Marshal request body
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create HTTP request
	url := b.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")

	// Set additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// ReadJSONResponse reads and unmarshals a JSON response.
func (b *BaseClient) ReadJSONResponse(resp *http.Response, v any) error {
	defer func() { _ = resp.Body.Close() }()

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Decode response
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// DoJSONRequestAndDecode performs a request and decodes the response in one call.
func (b *BaseClient) DoJSONRequestAndDecode(ctx context.Context, method, endpoint string, body any, headers map[string]string, result any) error {
	resp, err := b.DoJSONRequest(ctx, method, endpoint, body, headers)
	if err != nil {
		return err
	}

	return b.ReadJSONResponse(resp, result)
}

// ExpandEnvVar expands environment variables in the format ${VAR_NAME}.
// This is a utility function used by all AI clients.
func ExpandEnvVar(value string) string {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envVar := value[2 : len(value)-1]
		return os.Getenv(envVar)
	}
	return value
}

// GetAPIKeyFromEnv tries to get API key from config, then falls back to env var.
func GetAPIKeyFromEnv(configKey, envVarName string) string {
	apiKey := ExpandEnvVar(configKey)
	if apiKey == "" {
		apiKey = os.Getenv(envVarName)
	}
	return apiKey
}

// ConvertMessagesToMaps converts domain.ChatMessage slice to OpenAI-compatible format.
// Used by OpenAI, Groq, and Ollama clients which share the same message format.
func ConvertMessagesToMaps(messages []domain.ChatMessage) []map[string]string {
	result := make([]map[string]string, len(messages))
	for i, msg := range messages {
		result[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	return result
}

// ConvertToolsOpenAIFormat converts tools to OpenAI-compatible format.
// Used by OpenAI, Groq, and Ollama clients which share the same tools format.
func ConvertToolsOpenAIFormat(tools []domain.Tool) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, tool := range tools {
		result[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		}
	}
	return result
}

// FallbackStreamChat provides a simple streaming implementation that calls Chat and returns the result.
// Used by clients that don't have native streaming support.
func FallbackStreamChat(ctx context.Context, req *domain.ChatRequest, chatFunc func(context.Context, *domain.ChatRequest) (*domain.ChatResponse, error), callback func(chunk string) error) error {
	resp, err := chatFunc(ctx, req)
	if err != nil {
		return err
	}
	return callback(resp.Content)
}

// openAICompatibleResponse is the shared shape of /v1/chat/completions
// responses across providers that speak the OpenAI API surface (OpenAI,
// Groq, Together, Anyscale, etc.). Kept private to this package.
type openAICompatibleResponse struct {
	Choices []struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAICompatibleChat performs a chat request against any provider that
// implements the OpenAI /v1/chat/completions surface. provider is used to
// label the response and shape error messages.
//
// Callers (OpenAIClient, GroqClient, …) should validate IsConfigured before
// calling this method.
func (b *BaseClient) OpenAICompatibleChat(ctx context.Context, provider string, req *domain.ChatRequest, tools []domain.Tool) (*domain.ChatResponse, error) {
	body := map[string]any{
		"model":    b.GetModel(req.Model),
		"messages": ConvertMessagesToMaps(req.Messages),
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if len(tools) > 0 {
		body["tools"] = ConvertToolsOpenAIFormat(tools)
		body["tool_choice"] = "auto"
	}

	headers := map[string]string{
		"Authorization": "Bearer " + b.apiKey,
	}

	var raw openAICompatibleResponse
	if err := b.DoJSONRequestAndDecode(ctx, "POST", "/chat/completions", body, headers, &raw); err != nil {
		return nil, err
	}
	if len(raw.Choices) == 0 {
		return nil, fmt.Errorf("no response from %s", provider)
	}

	resp := &domain.ChatResponse{
		Content:  raw.Choices[0].Message.Content,
		Model:    raw.Model,
		Provider: provider,
		Usage: domain.TokenUsage{
			PromptTokens:     raw.Usage.PromptTokens,
			CompletionTokens: raw.Usage.CompletionTokens,
			TotalTokens:      raw.Usage.TotalTokens,
		},
	}
	for _, tc := range raw.Choices[0].Message.ToolCalls {
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			// The model emitted a tool-call with malformed JSON arguments.
			// Silently dropping it would leave the caller wondering why
			// `len(ToolCalls)` is short — return the parse error so the
			// scheduler can decide whether to retry or surface it.
			return nil, fmt.Errorf("model tool-call %q has invalid JSON arguments: %w",
				tc.Function.Name, err)
		}
		resp.ToolCalls = append(resp.ToolCalls, domain.ToolCall{
			ID:        tc.ID,
			Function:  tc.Function.Name,
			Arguments: args,
		})
	}
	return resp, nil
}
