package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListWebhooks(ctx context.Context) ([]domain.Webhook, error) {
	return []domain.Webhook{
		{
			ID:           "webhook-1",
			Description:  "Test Webhook",
			TriggerTypes: []string{domain.TriggerMessageCreated},
			WebhookURL:   "https://example.com/webhook",
			Status:       "active",
		},
	}, nil
}

// GetWebhook retrieves a single webhook.
func (m *MockClient) GetWebhook(ctx context.Context, webhookID string) (*domain.Webhook, error) {
	return &domain.Webhook{
		ID:           webhookID,
		Description:  "Test Webhook",
		TriggerTypes: []string{domain.TriggerMessageCreated},
		WebhookURL:   "https://example.com/webhook",
		Status:       "active",
	}, nil
}

// CreateWebhook creates a new webhook.
func (m *MockClient) CreateWebhook(ctx context.Context, req *domain.CreateWebhookRequest) (*domain.Webhook, error) {
	return &domain.Webhook{
		ID:            "new-webhook-id",
		Description:   req.Description,
		TriggerTypes:  req.TriggerTypes,
		WebhookURL:    req.WebhookURL,
		WebhookSecret: "mock-secret-12345",
		Status:        "active",
	}, nil
}

// UpdateWebhook updates an existing webhook.
func (m *MockClient) UpdateWebhook(ctx context.Context, webhookID string, req *domain.UpdateWebhookRequest) (*domain.Webhook, error) {
	webhook := &domain.Webhook{
		ID:     webhookID,
		Status: "active",
	}
	if req.Description != "" {
		webhook.Description = req.Description
	}
	if req.WebhookURL != "" {
		webhook.WebhookURL = req.WebhookURL
	}
	if len(req.TriggerTypes) > 0 {
		webhook.TriggerTypes = req.TriggerTypes
	}
	if req.Status != "" {
		webhook.Status = req.Status
	}
	return webhook, nil
}

// DeleteWebhook deletes a webhook.
func (m *MockClient) DeleteWebhook(ctx context.Context, webhookID string) error {
	return nil
}

// SendWebhookTestEvent sends a test event to a webhook URL.
func (m *MockClient) SendWebhookTestEvent(ctx context.Context, webhookURL string) error {
	return nil
}

// GetWebhookMockPayload returns a mock payload for a trigger type.
func (m *MockClient) GetWebhookMockPayload(ctx context.Context, triggerType string) (map[string]any, error) {
	return map[string]any{
		"specversion": "1.0",
		"type":        triggerType,
		"source":      "/nylas/test",
		"id":          "mock-event-id",
		"data": map[string]any{
			"object": map[string]any{
				"id": "mock-object-id",
			},
		},
	}, nil
}

// ListNotetakers lists all notetakers for a grant.
