package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_ListWebhooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":            "webhook-1",
					"description":   "Test Webhook 1",
					"trigger_types": []string{"message.created", "message.updated"},
					"webhook_url":   "https://example.com/webhook1",
					"status":        "active",
					"created_at":    int64(1609459200),
					"updated_at":    int64(1609459200),
				},
				{
					"id":            "webhook-2",
					"description":   "Test Webhook 2",
					"trigger_types": []string{"calendar.created"},
					"webhook_url":   "https://example.com/webhook2",
					"status":        "inactive",
					"created_at":    int64(1609459200),
					"updated_at":    int64(1609459200),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	webhooks, err := client.ListWebhooks(ctx)

	require.NoError(t, err)
	assert.Len(t, webhooks, 2)
	assert.Equal(t, "webhook-1", webhooks[0].ID)
	assert.Equal(t, "Test Webhook 1", webhooks[0].Description)
	assert.Equal(t, "https://example.com/webhook1", webhooks[0].WebhookURL)
	assert.Equal(t, "active", webhooks[0].Status)
}

func TestHTTPClient_GetWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks/webhook-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":                           "webhook-123",
				"description":                  "Production Webhook",
				"trigger_types":                []string{"message.created", "thread.replied"},
				"webhook_url":                  "https://api.example.com/nylas-webhook",
				"webhook_secret":               "secret-key-123",
				"status":                       "active",
				"notification_email_addresses": []string{"admin@example.com"},
				"status_updated_at":            int64(1609459200),
				"created_at":                   int64(1609459200),
				"updated_at":                   int64(1609459200),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	webhook, err := client.GetWebhook(ctx, "webhook-123")

	require.NoError(t, err)
	assert.Equal(t, "webhook-123", webhook.ID)
	assert.Equal(t, "Production Webhook", webhook.Description)
	assert.Equal(t, "https://api.example.com/nylas-webhook", webhook.WebhookURL)
	assert.Equal(t, "secret-key-123", webhook.WebhookSecret)
	assert.Equal(t, "active", webhook.Status)
	assert.Len(t, webhook.TriggerTypes, 2)
	assert.Contains(t, webhook.TriggerTypes, "message.created")
}

func TestHTTPClient_GetWebhook_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	webhook, err := client.GetWebhook(ctx, "")

	require.Error(t, err)
	assert.Nil(t, webhook)
	assert.Contains(t, err.Error(), "webhook ID")
}

func TestHTTPClient_CreateWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Webhook", body["description"])
		assert.Equal(t, "https://example.com/webhook", body["webhook_url"])

		response := map[string]any{
			"data": map[string]any{
				"id":            "webhook-new",
				"description":   "New Webhook",
				"trigger_types": body["trigger_types"],
				"webhook_url":   "https://example.com/webhook",
				"status":        "active",
				"created_at":    int64(1609459200),
				"updated_at":    int64(1609459200),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.CreateWebhookRequest{
		Description:  "New Webhook",
		TriggerTypes: []string{"message.created"},
		WebhookURL:   "https://example.com/webhook",
	}

	webhook, err := client.CreateWebhook(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "webhook-new", webhook.ID)
	assert.Equal(t, "New Webhook", webhook.Description)
	assert.Equal(t, "https://example.com/webhook", webhook.WebhookURL)
}

func TestHTTPClient_UpdateWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks/webhook-456", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Webhook", body["description"])

		response := map[string]any{
			"data": map[string]any{
				"id":            "webhook-456",
				"description":   "Updated Webhook",
				"trigger_types": []string{"calendar.created", "calendar.updated"},
				"webhook_url":   "https://example.com/updated-webhook",
				"status":        "active",
				"created_at":    int64(1609459200),
				"updated_at":    int64(1609459300),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.UpdateWebhookRequest{
		Description:  "Updated Webhook",
		TriggerTypes: []string{"calendar.created", "calendar.updated"},
		WebhookURL:   "https://example.com/updated-webhook",
	}

	webhook, err := client.UpdateWebhook(ctx, "webhook-456", req)

	require.NoError(t, err)
	assert.Equal(t, "webhook-456", webhook.ID)
	assert.Equal(t, "Updated Webhook", webhook.Description)
	assert.Equal(t, "https://example.com/updated-webhook", webhook.WebhookURL)
	assert.Len(t, webhook.TriggerTypes, 2)
}

func TestHTTPClient_UpdateWebhook_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	req := &domain.UpdateWebhookRequest{
		Description: "Test",
	}

	webhook, err := client.UpdateWebhook(ctx, "", req)

	require.Error(t, err)
	assert.Nil(t, webhook)
	assert.Contains(t, err.Error(), "webhook ID")
}

func TestHTTPClient_DeleteWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks/webhook-789", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteWebhook(ctx, "webhook-789")

	require.NoError(t, err)
}

func TestHTTPClient_DeleteWebhook_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	err := client.DeleteWebhook(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook ID")
}

func TestHTTPClient_RotateWebhookSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks/rotate-secret/webhook-789", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":             "webhook-789",
				"webhook_secret": "rotated-secret-123",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	rotated, err := client.RotateWebhookSecret(ctx, "webhook-789")

	require.NoError(t, err)
	require.NotNil(t, rotated)
	assert.Equal(t, "webhook-789", rotated.ID)
	assert.Equal(t, "rotated-secret-123", rotated.WebhookSecret)
}

func TestHTTPClient_RotateWebhookSecret_RootLevelSecretFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks/rotate-secret/webhook-789", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		response := map[string]any{
			"webhook_secret": "root-secret-123",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	rotated, err := client.RotateWebhookSecret(ctx, "webhook-789")

	require.NoError(t, err)
	require.NotNil(t, rotated)
	assert.Equal(t, "webhook-789", rotated.ID)
	assert.Equal(t, "root-secret-123", rotated.WebhookSecret)
}

func TestHTTPClient_RotateWebhookSecret_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	rotated, err := client.RotateWebhookSecret(ctx, "")

	require.Error(t, err)
	assert.Nil(t, rotated)
	assert.Contains(t, err.Error(), "webhook ID")
}

func TestHTTPClient_RotateWebhookSecret_MissingSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks/rotate-secret/webhook-789", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id": "webhook-789",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	rotated, err := client.RotateWebhookSecret(ctx, "webhook-789")

	require.Error(t, err)
	assert.Nil(t, rotated)
	assert.Contains(t, err.Error(), "missing webhook_secret")
}

func TestHTTPClient_SendWebhookTestEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks/send-test-event", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "https://test.example.com/webhook", body["webhook_url"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.SendWebhookTestEvent(ctx, "https://test.example.com/webhook")

	require.NoError(t, err)
}

func TestHTTPClient_SendWebhookTestEvent_EmptyURL(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	err := client.SendWebhookTestEvent(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook URL")
}

func TestHTTPClient_GetWebhookMockPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/webhooks/mock-payload", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "message.created", body["trigger_type"])

		response := map[string]any{
			"trigger_type": "message.created",
			"data": map[string]any{
				"id":      "msg-123",
				"subject": "Test Message",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	payload, err := client.GetWebhookMockPayload(ctx, "message.created")

	require.NoError(t, err)
	assert.NotNil(t, payload)
	assert.Equal(t, "message.created", payload["trigger_type"])
}

func TestHTTPClient_GetWebhookMockPayload_EmptyTriggerType(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	payload, err := client.GetWebhookMockPayload(ctx, "")

	require.Error(t, err)
	assert.Nil(t, payload)
	assert.Contains(t, err.Error(), "trigger type")
}
