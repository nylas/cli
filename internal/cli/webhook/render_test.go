package webhook

import (
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplayWebhookDetails_TypedRendering(t *testing.T) {
	webhook := &domain.Webhook{
		ID:                         "webhook-1234567890",
		Description:                "Test webhook",
		WebhookURL:                 "https://example.com/webhook",
		WebhookSecret:              "secret-12345678",
		Status:                     "active",
		TriggerTypes:               []string{"message.created", "message.updated"},
		NotificationEmailAddresses: []string{"alerts@example.com"},
		CreatedAt:                  time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:                  time.Date(2024, time.January, 3, 4, 5, 6, 0, time.UTC),
		StatusUpdatedAt:            time.Date(2024, time.January, 4, 5, 6, 7, 0, time.UTC),
	}

	output := captureStdout(t, func() {
		err := displayWebhookDetails(webhook)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Webhook: webhook-1234567890")
	assert.Contains(t, output, "Description:  Test webhook")
	assert.Contains(t, output, "URL:          https://example.com/webhook")
	assert.Contains(t, output, "message.created")
	assert.Contains(t, output, "alerts@example.com")
	assert.Contains(t, output, "2024-01-02T03:04:05Z")
	assert.NotContains(t, output, "secret-12345678")
}

func TestOutputTable_TypedRendering(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	webhooks := []domain.Webhook{
		{
			ID:           "webhook-1234567890",
			Description:  "Test webhook",
			WebhookURL:   "https://example.com/webhook",
			Status:       "active",
			TriggerTypes: []string{"message.created", "message.updated"},
		},
	}

	output := captureStdout(t, func() {
		err := outputTable(webhooks, false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Description")
	assert.Contains(t, output, "https://example.com/webhook")
	assert.Contains(t, output, "message.created")
	assert.Contains(t, output, "Total: 1 webhooks")
}

func TestOutputTable_Branches(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	webhooks := []domain.Webhook{
		{
			ID:           "webhook-1234567890",
			WebhookURL:   "https://example.com/webhook",
			Status:       "unknown",
			TriggerTypes: nil,
		},
	}

	output := captureStdout(t, func() {
		err := outputTable(webhooks, true)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "webhook-1234567890")
	assert.Contains(t, output, "○ unknown")
}

func TestOutputHelpers(t *testing.T) {
	webhooks := []domain.Webhook{
		{
			ID:           "webhook-123",
			Description:  "Test webhook",
			WebhookURL:   "https://example.com/webhook",
			Status:       "active",
			TriggerTypes: []string{"message.created", "message.updated"},
		},
	}

	t.Run("outputJSON", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := outputJSON(webhooks)
			require.NoError(t, err)
		})

		var decoded []domain.Webhook
		err := json.Unmarshal([]byte(output), &decoded)
		require.NoError(t, err)
		require.Len(t, decoded, 1)
		assert.Equal(t, "webhook-123", decoded[0].ID)
	})

	t.Run("outputYAML", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := outputYAML(webhooks)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "id: webhook-123")
		assert.Contains(t, output, "status: active")
	})

	t.Run("outputCSV", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := outputCSV(webhooks)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "ID,Description,URL,Status,Triggers")
		assert.Contains(t, output, "webhook-123,Test webhook,https://example.com/webhook,active,message.created;message.updated")
	})
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	require.NoError(t, w.Close())
	output, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(output)
}
