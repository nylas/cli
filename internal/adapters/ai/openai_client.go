package ai

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

// OpenAIClient implements LLMProvider for OpenAI.
type OpenAIClient struct {
	*BaseClient
}

// NewOpenAIClient creates a new OpenAI client.
func NewOpenAIClient(config *domain.OpenAIConfig) *OpenAIClient {
	if config == nil {
		config = &domain.OpenAIConfig{
			Model: "gpt-4-turbo",
		}
	}

	apiKey := GetAPIKeyFromEnv(config.APIKey, "OPENAI_API_KEY")

	return &OpenAIClient{
		BaseClient: NewBaseClient(
			apiKey,
			config.Model,
			"https://api.openai.com/v1",
			0, // Use default timeout
		),
	}
}

// Name returns the provider name.
func (c *OpenAIClient) Name() string {
	return "openai"
}

// IsAvailable checks if OpenAI API key is configured.
func (c *OpenAIClient) IsAvailable(ctx context.Context) bool {
	return c.IsConfigured()
}

// Chat sends a chat completion request.
func (c *OpenAIClient) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	return c.ChatWithTools(ctx, req, nil)
}

// ChatWithTools sends a chat request with function calling, delegating to
// the shared OpenAI-compatible pipeline in BaseClient.
func (c *OpenAIClient) ChatWithTools(ctx context.Context, req *domain.ChatRequest, tools []domain.Tool) (*domain.ChatResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("openai API key not configured")
	}
	return c.OpenAICompatibleChat(ctx, "openai", req, tools)
}

// StreamChat streams chat responses.
func (c *OpenAIClient) StreamChat(ctx context.Context, req *domain.ChatRequest, callback func(chunk string) error) error {
	return FallbackStreamChat(ctx, req, c.Chat, callback)
}
