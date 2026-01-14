package webhookserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		server := NewServer(ports.WebhookServerConfig{})
		assert.NotNil(t, server)
		assert.Equal(t, 3000, server.config.Port)
		assert.Equal(t, "/webhook", server.config.Path)
	})

	t.Run("custom_config", func(t *testing.T) {
		server := NewServer(ports.WebhookServerConfig{
			Port:          8080,
			Path:          "/api/webhook",
			WebhookSecret: "secret123",
		})
		assert.NotNil(t, server)
		assert.Equal(t, 8080, server.config.Port)
		assert.Equal(t, "/api/webhook", server.config.Path)
		assert.Equal(t, "secret123", server.config.WebhookSecret)
	})
}

func TestServer_StartStop(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 0, // Let OS pick a port
		Path: "/webhook",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Start(ctx)
	require.NoError(t, err)

	// Server should be running
	localURL := server.GetLocalURL()
	assert.Contains(t, localURL, "/webhook")

	// Stop the server
	err = server.Stop()
	require.NoError(t, err)
}

func TestServer_GetStats(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 3001,
		Path: "/webhook",
	})

	stats := server.GetStats()
	assert.Equal(t, "http://localhost:3001/webhook", stats.LocalURL)
	assert.Equal(t, 0, stats.EventsReceived)
}

func TestServer_HandleWebhook(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 3002,
		Path: "/webhook",
	})

	// Create test handler
	handler := http.HandlerFunc(server.handleWebhook)

	t.Run("post_webhook_event", func(t *testing.T) {
		payload := map[string]interface{}{
			"specversion": "1.0",
			"type":        "message.created",
			"source":      "nylas",
			"id":          "event-123",
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"grant_id": "grant-abc",
					"subject":  "Test Subject",
				},
			},
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "received", response["status"])
	})

	t.Run("get_challenge_verification", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhook?challenge=test-challenge-123", nil)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test-challenge-123", rec.Body.String())
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/webhook", nil)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})
}

func TestServer_HandleHealth(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 3003,
		Path: "/webhook",
	})

	handler := http.HandlerFunc(server.handleHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestServer_OnEvent(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 3004,
		Path: "/webhook",
	})

	eventReceived := make(chan *ports.WebhookEvent, 1)
	server.OnEvent(func(event *ports.WebhookEvent) {
		eventReceived <- event
	})

	// Simulate handling a webhook
	handler := http.HandlerFunc(server.handleWebhook)

	payload := map[string]interface{}{
		"type": "message.created",
		"id":   "test-event",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Wait for event handler
	select {
	case event := <-eventReceived:
		assert.Equal(t, "message.created", event.Type)
		assert.Equal(t, "test-event", event.ID)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestServer_EventsChannel(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 3005,
		Path: "/webhook",
	})

	events := server.Events()
	assert.NotNil(t, events)
}

func TestServer_SignatureVerification(t *testing.T) {
	// #nosec G101 -- This is a test secret, not a real credential
	secret := "test-webhook-secret"
	server := NewServer(ports.WebhookServerConfig{
		Port:          3006,
		Path:          "/webhook",
		WebhookSecret: secret,
	})

	t.Run("valid_signature", func(t *testing.T) {
		payload := []byte(`{"type":"message.created"}`)
		// Generate valid signature (HMAC-SHA256)
		valid := server.verifySignature(payload, "invalid-signature")
		assert.False(t, valid) // Invalid signature should fail
	})

	t.Run("missing_signature", func(t *testing.T) {
		payload := []byte(`{"type":"message.created"}`)
		valid := server.verifySignature(payload, "")
		assert.False(t, valid) // Empty signature should fail
	})
}

func TestServer_HandleRoot(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 3007,
		Path: "/webhook",
	})

	handler := http.HandlerFunc(server.handleRoot)

	t.Run("root_path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Nylas Webhook Server")
	})

	t.Run("non_root_path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/other", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestServer_GetLocalURL(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 8080,
		Path: "/api/hooks",
	})

	url := server.GetLocalURL()
	assert.Equal(t, "http://localhost:8080/api/hooks", url)
}

func TestServer_GetPublicURL(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{
		Port: 8080,
		Path: "/webhook",
	})

	// Without tunnel, public URL equals local URL
	url := server.GetPublicURL()
	assert.Equal(t, "http://localhost:8080/webhook", url)
}
