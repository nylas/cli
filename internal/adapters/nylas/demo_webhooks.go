package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListWebhooks(ctx context.Context) ([]domain.Webhook, error) {
	return []domain.Webhook{
		{
			ID:           "webhook-001",
			Description:  "Message notifications",
			TriggerTypes: []string{domain.TriggerMessageCreated, domain.TriggerMessageUpdated},
			WebhookURL:   "https://api.myapp.com/webhooks/nylas",
			Status:       "active",
			CreatedAt:    time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:           "webhook-002",
			Description:  "Calendar sync",
			TriggerTypes: []string{domain.TriggerEventCreated, domain.TriggerEventUpdated},
			WebhookURL:   "https://api.myapp.com/calendar/sync",
			Status:       "active",
			CreatedAt:    time.Now().Add(-14 * 24 * time.Hour),
		},
		{
			ID:           "webhook-003",
			Description:  "Contact updates (paused)",
			TriggerTypes: []string{domain.TriggerContactCreated},
			WebhookURL:   "https://api.myapp.com/contacts",
			Status:       "inactive",
			CreatedAt:    time.Now().Add(-7 * 24 * time.Hour),
		},
	}, nil
}

// GetWebhook returns a demo webhook.
func (d *DemoClient) GetWebhook(ctx context.Context, webhookID string) (*domain.Webhook, error) {
	webhooks, _ := d.ListWebhooks(ctx)
	for _, webhook := range webhooks {
		if webhook.ID == webhookID {
			return &webhook, nil
		}
	}
	return &webhooks[0], nil
}

// CreateWebhook simulates creating a webhook.
func (d *DemoClient) CreateWebhook(ctx context.Context, req *domain.CreateWebhookRequest) (*domain.Webhook, error) {
	return &domain.Webhook{
		ID:            "new-webhook",
		Description:   req.Description,
		TriggerTypes:  req.TriggerTypes,
		WebhookURL:    req.WebhookURL,
		WebhookSecret: "wh_secret_demo_12345",
		Status:        "active",
	}, nil
}

// UpdateWebhook simulates updating a webhook.
func (d *DemoClient) UpdateWebhook(ctx context.Context, webhookID string, req *domain.UpdateWebhookRequest) (*domain.Webhook, error) {
	return &domain.Webhook{ID: webhookID, Description: req.Description, Status: req.Status}, nil
}

// DeleteWebhook simulates deleting a webhook.
func (d *DemoClient) DeleteWebhook(ctx context.Context, webhookID string) error {
	return nil
}

// SendWebhookTestEvent simulates sending a test event.
func (d *DemoClient) SendWebhookTestEvent(ctx context.Context, webhookURL string) error {
	return nil
}

// GetWebhookMockPayload returns a mock payload for a trigger type.
func (d *DemoClient) GetWebhookMockPayload(ctx context.Context, triggerType string) (map[string]any, error) {
	return map[string]any{
		"specversion": "1.0",
		"type":        triggerType,
		"source":      "/nylas/demo",
		"id":          "demo-event-id",
		"data":        map[string]any{"object": map[string]any{"id": "demo-object-id"}},
	}, nil
}

// ListScheduledMessages returns demo scheduled messages.
