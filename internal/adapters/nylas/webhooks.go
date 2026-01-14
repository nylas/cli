package nylas

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

// webhookResponse represents a webhook from the API.
type webhookResponse struct {
	ID                         string   `json:"id"`
	Description                string   `json:"description"`
	TriggerTypes               []string `json:"trigger_types"`
	WebhookURL                 string   `json:"webhook_url"`
	WebhookSecret              string   `json:"webhook_secret"`
	Status                     string   `json:"status"`
	NotificationEmailAddresses []string `json:"notification_email_addresses"`
	StatusUpdatedAt            int64    `json:"status_updated_at"`
	CreatedAt                  int64    `json:"created_at"`
	UpdatedAt                  int64    `json:"updated_at"`
}

// ListWebhooks retrieves all webhooks.
func (c *HTTPClient) ListWebhooks(ctx context.Context) ([]domain.Webhook, error) {
	queryURL := fmt.Sprintf("%s/v3/webhooks", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []webhookResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return util.Map(result.Data, convertWebhook), nil
}

// GetWebhook retrieves a single webhook by ID.
func (c *HTTPClient) GetWebhook(ctx context.Context, webhookID string) (*domain.Webhook, error) {
	if err := validateRequired("webhook ID", webhookID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/webhooks/%s", c.baseURL, webhookID)

	var result struct {
		Data webhookResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: webhook not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}

	webhook := convertWebhook(result.Data)
	return &webhook, nil
}

// CreateWebhook creates a new webhook.
func (c *HTTPClient) CreateWebhook(ctx context.Context, req *domain.CreateWebhookRequest) (*domain.Webhook, error) {
	queryURL := fmt.Sprintf("%s/v3/webhooks", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data webhookResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	webhook := convertWebhook(result.Data)
	return &webhook, nil
}

// UpdateWebhook updates an existing webhook.
func (c *HTTPClient) UpdateWebhook(ctx context.Context, webhookID string, req *domain.UpdateWebhookRequest) (*domain.Webhook, error) {
	if err := validateRequired("webhook ID", webhookID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/webhooks/%s", c.baseURL, webhookID)

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data webhookResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	webhook := convertWebhook(result.Data)
	return &webhook, nil
}

// DeleteWebhook deletes a webhook.
func (c *HTTPClient) DeleteWebhook(ctx context.Context, webhookID string) error {
	if err := validateRequired("webhook ID", webhookID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/webhooks/%s", c.baseURL, webhookID)
	return c.doDelete(ctx, queryURL)
}

// SendWebhookTestEvent sends a test event to a webhook URL.
func (c *HTTPClient) SendWebhookTestEvent(ctx context.Context, webhookURL string) error {
	if err := validateRequired("webhook URL", webhookURL); err != nil {
		return err
	}

	queryURL := fmt.Sprintf("%s/v3/webhooks/send-test-event", c.baseURL)

	payload := map[string]string{"webhook_url": webhookURL}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload, http.StatusOK, http.StatusNoContent)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// GetWebhookMockPayload gets a mock payload for a trigger type.
func (c *HTTPClient) GetWebhookMockPayload(ctx context.Context, triggerType string) (map[string]any, error) {
	if err := validateRequired("trigger type", triggerType); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/webhooks/mock-payload", c.baseURL)

	payload := map[string]string{"trigger_type": triggerType}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// convertWebhook converts an API webhook response to domain model.
func convertWebhook(w webhookResponse) domain.Webhook {
	return domain.Webhook{
		ID:                         w.ID,
		Description:                w.Description,
		TriggerTypes:               w.TriggerTypes,
		WebhookURL:                 w.WebhookURL,
		WebhookSecret:              w.WebhookSecret,
		Status:                     w.Status,
		NotificationEmailAddresses: w.NotificationEmailAddresses,
		StatusUpdatedAt:            unixToTime(w.StatusUpdatedAt),
		CreatedAt:                  unixToTime(w.CreatedAt),
		UpdatedAt:                  unixToTime(w.UpdatedAt),
	}
}

// unixToTime converts a Unix timestamp to time.Time, returning zero time if timestamp is 0.
func unixToTime(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}
