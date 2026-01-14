package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// Service implements ports.WebhookService.
// Provides local webhook server for testing without ngrok.
type Service struct {
	mu         sync.RWMutex
	server     *http.Server
	httpClient *http.Client // Reused for ReplayWebhook calls
	webhooks   []domain.WebhookPayload
	config     *domain.WebhookServerConfig
	nextID     int
}

// NewService creates a new webhook service.
func NewService() *Service {
	return &Service{
		webhooks: make([]domain.WebhookPayload, 0),
		nextID:   1,
		// Single HTTP client for all replay requests (connection pooling)
		httpClient: &http.Client{
			Timeout: domain.TimeoutAPI,
		},
	}
}

// StartServer starts a local webhook server.
func (s *Service) StartServer(ctx context.Context, config *domain.WebhookServerConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		return fmt.Errorf("server already running")
	}

	s.config = config

	// Create HTTP handler
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)

	// Create server with timeouts to prevent slowloris attacks
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second, // Prevent slowloris attacks
		ReadTimeout:       30 * time.Second, // Timeout for reading entire request
		WriteTimeout:      30 * time.Second, // Timeout for writing response
		IdleTimeout:       60 * time.Second, // Timeout for keep-alive connections
	}

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Webhook server error: %v\n", err)
		}
	}()

	return nil
}

// StopServer stops the running webhook server.
func (s *Service) StopServer(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return fmt.Errorf("no server running")
	}

	err := s.server.Shutdown(ctx)
	s.server = nil
	return err
}

// GetReceivedWebhooks returns all captured webhook payloads.
func (s *Service) GetReceivedWebhooks(ctx context.Context) ([]domain.WebhookPayload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return copy to avoid race conditions
	webhooks := make([]domain.WebhookPayload, len(s.webhooks))
	copy(webhooks, s.webhooks)

	return webhooks, nil
}

// ValidateSignature validates a webhook signature using HMAC-SHA256.
func (s *Service) ValidateSignature(payload []byte, signature string, secret string) bool {
	if secret == "" {
		return true // No secret configured, skip validation
	}

	// Calculate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expected := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// ReplayWebhook replays a captured webhook to a target URL.
func (s *Service) ReplayWebhook(ctx context.Context, webhookID string, targetURL string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find webhook by ID
	var webhook *domain.WebhookPayload
	for i := range s.webhooks {
		if s.webhooks[i].ID == webhookID {
			webhook = &s.webhooks[i]
			break
		}
	}

	if webhook == nil {
		return fmt.Errorf("webhook not found: %s", webhookID)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, webhook.Method, targetURL, bytes.NewReader(webhook.Body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Copy headers
	for k, v := range webhook.Headers {
		req.Header.Set(k, v)
	}

	// Send request using reusable HTTP client
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("replay failed with status %d", resp.StatusCode)
	}

	return nil
}

// SaveWebhook saves a webhook payload to file for later replay.
func (s *Service) SaveWebhook(ctx context.Context, payload *domain.WebhookPayload, filepath string) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal webhook: %w", err)
	}

	// Use restrictive permissions (owner-only) for webhook payloads
	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// LoadWebhook loads a webhook payload from file.
func (s *Service) LoadWebhook(ctx context.Context, filepath string) (*domain.WebhookPayload, error) {
	// #nosec G304 -- filepath comes from validated CLI argument, user controls their own file system
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var payload domain.WebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal webhook: %w", err)
	}

	return &payload, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// handleWebhook handles incoming webhook requests.
func (s *Service) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Extract headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Get signature if present
	signature := r.Header.Get("X-Nylas-Signature")

	// Validate signature if configured
	verified := true
	if s.config.ValidateSignature && s.config.Secret != "" {
		verified = s.ValidateSignature(body, signature, s.config.Secret)
	}

	// Create webhook payload
	s.mu.Lock()
	payload := domain.WebhookPayload{
		ID:        fmt.Sprintf("wh_%d", s.nextID),
		Timestamp: time.Now(),
		Method:    r.Method,
		URL:       r.URL.String(),
		Headers:   headers,
		Body:      body,
		Signature: signature,
		Verified:  verified,
	}
	s.nextID++
	s.webhooks = append(s.webhooks, payload)
	s.mu.Unlock()

	// Save to file if configured
	if s.config.SaveToFile && s.config.FilePath != "" {
		_ = s.SaveWebhook(context.Background(), &payload, s.config.FilePath)
	}

	// Print webhook for debugging
	fmt.Printf("📥 Webhook received: %s %s (verified: %v)\n", payload.Method, payload.URL, verified)
	if len(body) > 0 && len(body) < 1000 {
		fmt.Printf("   Body: %s\n", string(body))
	}

	// Respond
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"id":     payload.ID,
	})
}

// handleHealth handles health check requests.
func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "healthy",
		"webhooks": len(s.webhooks),
	})
}
