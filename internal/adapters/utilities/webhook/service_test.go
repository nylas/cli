package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestNewService(t *testing.T) {
	service := NewService()

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.webhooks == nil {
		t.Error("expected webhooks slice to be initialized")
	}
	if service.nextID != 1 {
		t.Errorf("expected nextID to be 1, got %d", service.nextID)
	}
}

func TestService_StartAndStopServer(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	config := &domain.WebhookServerConfig{
		Host: "localhost",
		Port: 18080,
	}

	// Start server
	err := service.StartServer(ctx, config)
	if err != nil {
		t.Fatalf("StartServer() error = %v", err)
	}

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Verify server is running
	if service.server == nil {
		t.Error("expected server to be running")
	}

	// Try to start second server (should fail)
	err = service.StartServer(ctx, config)
	if err == nil {
		t.Error("expected error when starting server twice")
	}
	if err.Error() != "server already running" {
		t.Errorf("expected 'server already running' error, got: %v", err)
	}

	// Stop server
	err = service.StopServer(ctx)
	if err != nil {
		t.Errorf("StopServer() error = %v", err)
	}

	// Give server time to stop
	time.Sleep(200 * time.Millisecond)

	// Verify server is stopped
	if service.server != nil {
		t.Error("expected server to be nil after stopping")
	}
}

func TestService_StopServer_NotRunning(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	err := service.StopServer(ctx)
	if err == nil {
		t.Error("expected error when stopping non-running server")
	}
	if err.Error() != "no server running" {
		t.Errorf("expected 'no server running' error, got: %v", err)
	}
}

func TestService_GetReceivedWebhooks(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Initially should be empty
	webhooks, err := service.GetReceivedWebhooks(ctx)
	if err != nil {
		t.Fatalf("GetReceivedWebhooks() error = %v", err)
	}
	if len(webhooks) != 0 {
		t.Errorf("expected 0 webhooks, got %d", len(webhooks))
	}

	// Add a webhook
	service.webhooks = append(service.webhooks, domain.WebhookPayload{
		ID:        "wh_1",
		Timestamp: time.Now(),
		Method:    "POST",
		URL:       "/webhook",
		Body:      []byte("test"),
	})

	// Should return the webhook
	webhooks, err = service.GetReceivedWebhooks(ctx)
	if err != nil {
		t.Fatalf("GetReceivedWebhooks() error = %v", err)
	}
	if len(webhooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(webhooks))
	}
	if webhooks[0].ID != "wh_1" {
		t.Errorf("expected webhook ID 'wh_1', got %q", webhooks[0].ID)
	}
}

func TestService_ValidateSignature(t *testing.T) {
	service := NewService()

	tests := []struct {
		name      string
		payload   []byte
		secret    string
		signature string
		want      bool
	}{
		{
			name:      "valid signature",
			payload:   []byte("test payload"),
			secret:    "test-secret",
			signature: calculateHMAC([]byte("test payload"), "test-secret"),
			want:      true,
		},
		{
			name:      "invalid signature",
			payload:   []byte("test payload"),
			secret:    "test-secret",
			signature: "invalid-signature",
			want:      false,
		},
		{
			name:      "empty secret (skip validation)",
			payload:   []byte("test payload"),
			secret:    "",
			signature: "any-signature",
			want:      true,
		},
		{
			name:      "wrong secret",
			payload:   []byte("test payload"),
			secret:    "wrong-secret",
			signature: calculateHMAC([]byte("test payload"), "test-secret"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.ValidateSignature(tt.payload, tt.signature, tt.secret)
			if got != tt.want {
				t.Errorf("ValidateSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_SaveAndLoadWebhook(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	tmpFile := t.TempDir() + "/webhook.json"

	payload := &domain.WebhookPayload{
		ID:        "wh_test",
		Timestamp: time.Now(),
		Method:    "POST",
		URL:       "/test",
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      []byte(`{"test": "data"}`),
		Signature: "test-signature",
		Verified:  true,
	}

	// Save webhook
	err := service.SaveWebhook(ctx, payload, tmpFile)
	if err != nil {
		t.Fatalf("SaveWebhook() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("webhook file was not created")
	}

	// Verify file permissions (owner-only)
	info, _ := os.Stat(tmpFile)
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected file permissions 0600, got %o", info.Mode().Perm())
	}

	// Load webhook
	loaded, err := service.LoadWebhook(ctx, tmpFile)
	if err != nil {
		t.Fatalf("LoadWebhook() error = %v", err)
	}

	// Verify loaded webhook matches original
	if loaded.ID != payload.ID {
		t.Errorf("expected ID %q, got %q", payload.ID, loaded.ID)
	}
	if loaded.Method != payload.Method {
		t.Errorf("expected Method %q, got %q", payload.Method, loaded.Method)
	}
	if loaded.URL != payload.URL {
		t.Errorf("expected URL %q, got %q", payload.URL, loaded.URL)
	}
	if string(loaded.Body) != string(payload.Body) {
		t.Errorf("expected Body %q, got %q", payload.Body, loaded.Body)
	}
}

func TestService_LoadWebhook_FileNotFound(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	_, err := service.LoadWebhook(ctx, "/nonexistent/webhook.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestService_LoadWebhook_InvalidJSON(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	tmpFile := t.TempDir() + "/invalid.json"

	// Write invalid JSON
	if err := os.WriteFile(tmpFile, []byte("{invalid json"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := service.LoadWebhook(ctx, tmpFile)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestService_HandleWebhook(t *testing.T) {
	service := NewService()
	service.config = &domain.WebhookServerConfig{
		ValidateSignature: false,
	}

	payload := []byte(`{"event": "test"}`)
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "test-value")

	w := httptest.NewRecorder()

	service.handleWebhook(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify webhook was captured
	service.mu.RLock()
	count := len(service.webhooks)
	service.mu.RUnlock()

	if count != 1 {
		t.Errorf("expected 1 webhook captured, got %d", count)
	}
}

func TestService_HandleHealth(t *testing.T) {
	service := NewService()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	service.handleHealth(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", resp["status"])
	}
}

func TestService_ReplayWebhook(t *testing.T) {
	// Create target server
	targetCalled := false
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	service := NewService()
	ctx := context.Background()

	// Add a webhook to replay
	service.webhooks = append(service.webhooks, domain.WebhookPayload{
		ID:      "wh_replay",
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"test": "data"}`),
	})

	// Replay webhook
	err := service.ReplayWebhook(ctx, "wh_replay", targetServer.URL)
	if err != nil {
		t.Fatalf("ReplayWebhook() error = %v", err)
	}

	if !targetCalled {
		t.Error("target server was not called")
	}
}

func TestService_ReplayWebhook_NotFound(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	err := service.ReplayWebhook(ctx, "nonexistent", "http://localhost")
	if err == nil {
		t.Error("expected error for nonexistent webhook")
	}
	if err.Error() != "webhook not found: nonexistent" {
		t.Errorf("expected 'webhook not found' error, got: %v", err)
	}
}

// Helper function to calculate HMAC
func calculateHMAC(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
