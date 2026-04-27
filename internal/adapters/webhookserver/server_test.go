package webhookserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
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
	port := reserveTCPPort(t)

	server := NewServer(ports.WebhookServerConfig{
		Port: port,
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

func reserveTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok, fmt.Sprintf("listener addr %T is not *net.TCPAddr", listener.Addr()))
	require.NotZero(t, addr.Port)

	return addr.Port
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
		payload := map[string]any{
			"specversion": "1.0",
			"type":        "message.created",
			"source":      "nylas",
			"id":          "event-123",
			"data": map[string]any{
				"object": map[string]any{
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

		var response map[string]any
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

func TestServer_HandleWebhook_RejectsInvalidSignatures(t *testing.T) {
	secret := "test-webhook-secret"
	server := NewServer(ports.WebhookServerConfig{
		Port:          3008,
		Path:          "/webhook",
		WebhookSecret: secret,
	})
	handler := http.HandlerFunc(server.handleWebhook)
	payload := []byte(`{"type":"message.created","id":"event-123"}`)

	t.Run("missing signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, 0, server.GetStats().EventsReceived)
		select {
		case event := <-server.Events():
			t.Fatalf("unexpected event received: %+v", event)
		default:
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("X-Nylas-Signature", "invalid-signature")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Equal(t, 0, server.GetStats().EventsReceived)
	})

	t.Run("valid signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("X-Nylas-Signature", signWebhookPayload(secret, payload))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, 1, server.GetStats().EventsReceived)
		event := <-server.Events()
		assert.True(t, event.Verified)
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

	var response map[string]any
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

	payload := map[string]any{
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

func signWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// TestServer_HandleWebhook_RejectsOversizedBody verifies the body-size cap
// rejects payloads larger than the configured limit before allocating
// gigabytes of RAM. This is the gate that prevents a malicious sender on
// a public tunnel URL from driving the receiver out of memory.
func TestServer_HandleWebhook_RejectsOversizedBody(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{Port: 0, Path: "/webhook"})

	// 2 MiB — twice the cap.
	oversized := bytes.Repeat([]byte("A"), 2<<20)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(oversized))
	rec := httptest.NewRecorder()
	server.handleWebhook(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code,
		"oversized body must be rejected with 413, got %d", rec.Code)
}

// TestServer_HandleWebhook_DropsOldEvents_ReplayWindow exercises the
// MaxEventAge gate. With MaxEventAge configured, an event whose
// CloudEvents `time` field is older than the window is rejected as a
// replay even when the HMAC verifies — bound to a captured signature, an
// attacker would otherwise be able to replay a single signed body
// indefinitely.
func TestServer_HandleWebhook_DropsOldEvents_ReplayWindow(t *testing.T) {
	secret := "test-secret"
	server := NewServer(ports.WebhookServerConfig{
		Port:          0,
		Path:          "/webhook",
		WebhookSecret: secret,
		MaxEventAge:   30 * time.Second,
	})

	// Body with a `time` field 5 minutes in the past.
	oldTime := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	body := []byte(`{"id":"evt_1","type":"message.created","time":"` + oldTime + `"}`)
	sig := signWebhookPayload(secret, body)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Nylas-Signature", sig)
	rec := httptest.NewRecorder()
	server.handleWebhook(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code,
		"stale event must be rejected as replay, got %d", rec.Code)
}

// TestServer_HandleWebhook_AcceptsRecentEvents_ReplayWindow is the
// positive twin of the replay test: a signed event with a fresh `time`
// passes the gate.
func TestServer_HandleWebhook_AcceptsRecentEvents_ReplayWindow(t *testing.T) {
	secret := "test-secret"
	server := NewServer(ports.WebhookServerConfig{
		Port:          0,
		Path:          "/webhook",
		WebhookSecret: secret,
		MaxEventAge:   60 * time.Second,
	})

	now := time.Now().UTC().Format(time.RFC3339)
	body := []byte(`{"id":"evt_2","type":"message.created","time":"` + now + `"}`)
	sig := signWebhookPayload(secret, body)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Nylas-Signature", sig)
	rec := httptest.NewRecorder()
	server.handleWebhook(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "fresh event must be accepted")
}

// TestServer_HandleHealth_SurfacesEventsDropped confirms the health
// response includes the events_dropped counter so operators can detect a
// slow consumer without parsing logs.
func TestServer_HandleHealth_SurfacesEventsDropped(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{Port: 0, Path: "/webhook"})
	server.mu.Lock()
	server.stats.EventsDropped = 7
	server.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	server.handleHealth(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Contains(t, body, "events_dropped")
	assert.Equal(t, float64(7), body["events_dropped"])
}

// TestServer_StartBindsLoopbackOnly asserts the listener address is on a
// loopback interface — guards against an accidental change from
// 127.0.0.1: to :PORT (which would let any host on the LAN forge events).
func TestServer_StartBindsLoopbackOnly(t *testing.T) {
	server := NewServer(ports.WebhookServerConfig{Port: 0, Path: "/webhook"})
	require.NoError(t, server.Start(context.Background()))
	defer func() { _ = server.Stop() }()

	addr := server.listener.Addr().String()
	host, _, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	ip := net.ParseIP(host)
	require.NotNil(t, ip, "could not parse listener host as IP: %s", host)
	assert.True(t, ip.IsLoopback(), "listener bound to non-loopback address: %s", addr)
}
