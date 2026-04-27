package ai

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

// GroqClient implements LLMProvider for Groq.
type GroqClient struct {
	*BaseClient
}

// NewGroqClient creates a new Groq client.
func NewGroqClient(config *domain.GroqConfig) *GroqClient {
	if config == nil {
		config = &domain.GroqConfig{
			Model: "mixtral-8x7b-32768",
		}
	}

	apiKey := GetAPIKeyFromEnv(config.APIKey, "GROQ_API_KEY")

	return &GroqClient{
		BaseClient: NewBaseClient(
			apiKey,
			config.Model,
			"https://api.groq.com/openai/v1",
			0, // Use default timeout
		),
	}
}

// Name returns the provider name.
func (c *GroqClient) Name() string {
	return "groq"
}

// IsAvailable checks if Groq API key is configured.
func (c *GroqClient) IsAvailable(ctx context.Context) bool {
	return c.IsConfigured()
}

// Chat sends a chat completion request.
func (c *GroqClient) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	return c.ChatWithTools(ctx, req, nil)
}

// ChatWithTools sends a chat request with function calling. Groq exposes the
// OpenAI /v1/chat/completions surface, so this delegates to the shared
// pipeline.
func (c *GroqClient) ChatWithTools(ctx context.Context, req *domain.ChatRequest, tools []domain.Tool) (*domain.ChatResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("groq API key not configured")
	}
	return c.OpenAICompatibleChat(ctx, "groq", req, tools)
}

// StreamChat streams chat responses.
func (c *GroqClient) StreamChat(ctx context.Context, req *domain.ChatRequest, callback func(chunk string) error) error {
	return FallbackStreamChat(ctx, req, c.Chat, callback)
}
