// Package webhookserver provides a local webhook receiver server implementation.
package webhookserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nylas/cli/internal/ports"
)

// Server implements the WebhookServer interface.
type Server struct {
	config    ports.WebhookServerConfig
	server    *http.Server
	listener  net.Listener
	tunnel    ports.Tunnel
	events    chan *ports.WebhookEvent
	handlers  []ports.WebhookEventHandler
	stats     ports.WebhookServerStats
	mu        sync.RWMutex
	startedAt time.Time
}

// NewServer creates a new webhook server.
func NewServer(config ports.WebhookServerConfig) *Server {
	if config.Port == 0 {
		config.Port = 3000
	}
	if config.Path == "" {
		config.Path = "/webhook"
	}

	return &Server{
		config: config,
		events: make(chan *ports.WebhookEvent, 100),
		stats: ports.WebhookServerStats{
			LocalURL: fmt.Sprintf("http://localhost:%d%s", config.Port, config.Path),
		},
	}
}

// SetTunnel sets the tunnel to use for exposing the server.
func (s *Server) SetTunnel(tunnel ports.Tunnel) {
	s.tunnel = tunnel
}

// Start starts the webhook server and optional tunnel.
func (s *Server) Start(ctx context.Context) error {
	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc(s.config.Path, s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRoot)

	s.server = &http.Server{
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start listener
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to start listener on port %d: %w", s.config.Port, err)
	}

	s.startedAt = time.Now()
	s.mu.Lock()
	s.stats.StartedAt = s.startedAt
	s.stats.LocalURL = fmt.Sprintf("http://localhost:%d%s", s.config.Port, s.config.Path)
	s.mu.Unlock()

	// Start HTTP server in goroutine
	go func() {
		_ = s.server.Serve(s.listener) // Error handled on shutdown, ErrServerClosed expected
	}()

	// Start tunnel if configured
	if s.tunnel != nil {
		localURL := fmt.Sprintf("http://localhost:%d", s.config.Port)
		publicURL, err := s.tunnel.Start(ctx)
		if err != nil {
			_ = s.Stop() // Ignore stop error - we're returning tunnel start error
			return fmt.Errorf("failed to start tunnel: %w", err)
		}

		s.mu.Lock()
		s.stats.PublicURL = publicURL + s.config.Path
		s.stats.TunnelProvider = s.config.TunnelProvider
		s.stats.TunnelStatus = string(s.tunnel.Status())
		s.mu.Unlock()

		_ = localURL // used by tunnel
	}

	return nil
}

// Stop stops the webhook server and tunnel.
func (s *Server) Stop() error {
	var errs []error

	// Stop tunnel first
	if s.tunnel != nil {
		if err := s.tunnel.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("tunnel stop: %w", err))
		}
	}

	// Stop HTTP server
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("server shutdown: %w", err))
		}
	}

	// Close events channel
	close(s.events)

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// GetLocalURL returns the local server URL.
func (s *Server) GetLocalURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats.LocalURL
}

// GetPublicURL returns the public URL (from tunnel, if any).
func (s *Server) GetPublicURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.stats.PublicURL != "" {
		return s.stats.PublicURL
	}
	return s.stats.LocalURL
}

// GetStats returns server statistics.
func (s *Server) GetStats() ports.WebhookServerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.stats
	if s.tunnel != nil {
		stats.TunnelStatus = string(s.tunnel.Status())
	}
	return stats
}

// OnEvent registers a handler for webhook events.
func (s *Server) OnEvent(handler ports.WebhookEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}

// Events returns a channel for receiving webhook events.
func (s *Server) Events() <-chan *ports.WebhookEvent {
	return s.events
}

// handleWebhook handles incoming webhook requests.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// Handle Nylas webhook verification challenge
		if r.Method == http.MethodGet {
			challenge := r.URL.Query().Get("challenge")
			if challenge != "" {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(challenge)) // Ignore write error - response already sent
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Parse webhook event
	event := &ports.WebhookEvent{
		Timestamp:  time.Now(),
		ReceivedAt: time.Now(),
		Headers:    make(map[string]string),
		RawBody:    body,
	}

	// Copy relevant headers
	for k, v := range r.Header {
		if len(v) > 0 {
			event.Headers[k] = v[0]
		}
	}

	// Get signature header
	event.Signature = r.Header.Get("X-Nylas-Signature")

	// Verify signature if secret is configured
	if s.config.WebhookSecret != "" && event.Signature != "" {
		event.Verified = s.verifySignature(body, event.Signature)
	}

	// Parse JSON body
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		event.Body = payload

		// Extract common fields from CloudEvents format
		if id, ok := payload["id"].(string); ok {
			event.ID = id
		}
		if eventType, ok := payload["type"].(string); ok {
			event.Type = eventType
		}
		if source, ok := payload["source"].(string); ok {
			event.Source = source
		}

		// Extract grant_id from data.object if present
		if data, ok := payload["data"].(map[string]any); ok {
			if obj, ok := data["object"].(map[string]any); ok {
				if grantID, ok := obj["grant_id"].(string); ok {
					event.GrantID = grantID
				}
			}
		}
	}

	// Update stats
	s.mu.Lock()
	s.stats.EventsReceived++
	s.stats.LastEventAt = time.Now()
	s.mu.Unlock()

	// Send to channel (non-blocking)
	select {
	case s.events <- event:
	default:
		// Channel full, drop oldest
	}

	// Call handlers
	s.mu.RLock()
	handlers := s.handlers
	s.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}

	// Respond with 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"received"}`)) // Ignore write error - response already sent
}

// handleHealth handles health check requests.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats := s.GetStats()

	response := map[string]any{
		"status":          "healthy",
		"started_at":      stats.StartedAt,
		"events_received": stats.EventsReceived,
		"local_url":       stats.LocalURL,
		"public_url":      stats.PublicURL,
	}

	if s.tunnel != nil {
		response["tunnel_status"] = stats.TunnelStatus
		response["tunnel_provider"] = stats.TunnelProvider
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response) // Ignore encode error - best effort response
}

// handleRoot handles root requests with server info.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	stats := s.GetStats()

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Nylas Webhook Server</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
               max-width: 600px; margin: 50px auto; padding: 20px; }
        h1 { color: #0066cc; }
        .status { background: #e8f5e9; padding: 15px; border-radius: 8px; margin: 20px 0; }
        .url { background: #f5f5f5; padding: 10px; border-radius: 4px; font-family: monospace; word-break: break-all; }
        .stat { margin: 10px 0; }
        .label { font-weight: bold; color: #666; }
    </style>
</head>
<body>
    <h1>Nylas Webhook Server</h1>
    <div class="status">
        <div class="stat"><span class="label">Status:</span> Running</div>
        <div class="stat"><span class="label">Events Received:</span> %d</div>
        <div class="stat"><span class="label">Started:</span> %s</div>
    </div>
    <h3>Webhook Endpoint</h3>
    <div class="url">%s</div>
    <p>Send POST requests to this URL to receive webhook events.</p>
    <h3>Health Check</h3>
    <div class="url">%s/health</div>
</body>
</html>`,
		stats.EventsReceived,
		stats.StartedAt.Format(time.RFC3339),
		stats.PublicURL,
		s.GetPublicURL(),
	)

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(html)) // Ignore write error - best effort response
}

// verifySignature verifies the webhook signature using HMAC-SHA256.
func (s *Server) verifySignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(s.config.WebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
