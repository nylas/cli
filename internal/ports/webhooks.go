package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// WebhookClient defines the interface for webhook operations.
type WebhookClient interface {
	// ListWebhooks retrieves all webhooks.
	ListWebhooks(ctx context.Context) ([]domain.Webhook, error)

	// GetWebhook retrieves a specific webhook.
	GetWebhook(ctx context.Context, webhookID string) (*domain.Webhook, error)

	// CreateWebhook creates a new webhook.
	CreateWebhook(ctx context.Context, req *domain.CreateWebhookRequest) (*domain.Webhook, error)

	// UpdateWebhook updates an existing webhook.
	UpdateWebhook(ctx context.Context, webhookID string, req *domain.UpdateWebhookRequest) (*domain.Webhook, error)

	// DeleteWebhook deletes a webhook.
	DeleteWebhook(ctx context.Context, webhookID string) error

	// RotateWebhookSecret rotates and returns the secret for a webhook.
	RotateWebhookSecret(ctx context.Context, webhookID string) (*domain.RotateWebhookSecretResponse, error)

	// SendWebhookTestEvent sends a test event to a webhook URL.
	SendWebhookTestEvent(ctx context.Context, webhookURL string) error

	// GetWebhookMockPayload retrieves a mock payload for a webhook trigger type.
	GetWebhookMockPayload(ctx context.Context, triggerType string) (map[string]any, error)
}
