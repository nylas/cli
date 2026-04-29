// Package webhookserver provides a local webhook receiver server implementation.
package webhookserver

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nylas/cli/internal/ports"
)

const (
	// maxWebhookBodyBytes caps the request body size accepted by the webhook
	// receiver. Nylas events are well under 100 KB; the cap exists to prevent
	// a malicious sender (the public tunnel URL is reachable by anyone) from
	// asking the server to allocate gigabytes of RAM before HMAC verifies.
	maxWebhookBodyBytes = 1 << 20 // 1 MiB

	// maxConcurrentHandlers bounds the goroutines that fan-out registered
	// handlers. Without a bound, a flood of events combined with slow handlers
	// would let an attacker drive unbounded goroutine creation.
	maxConcurrentHandlers = 32
)

// LocalBaseURL returns the loopback URL used by the webhook receiver and
// local tunnels. It intentionally uses IPv4 loopback because the server binds
// to 127.0.0.1, not localhost's platform-dependent IPv4/IPv6 resolution.
func LocalBaseURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func localEndpointURL(port int, path string) string {
	return LocalBaseURL(port) + path
}

// rootTemplate renders the webhook server landing page. html/template HTML-
// escapes every value, preventing PublicURL (and any future user-derived
// fields) from breaking out of the document.
var rootTemplate = template.Must(template.New("root").Parse(`<!DOCTYPE html>
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
        <div class="stat"><span class="label">Events Received:</span> {{.EventsReceived}}</div>
        <div class="stat"><span class="label">Started:</span> {{.StartedAt}}</div>
    </div>
    <h3>Webhook Endpoint</h3>
    <div class="url">{{.PublicURL}}</div>
    <p>Send POST requests to this URL to receive webhook events.</p>
    <h3>Health Check</h3>
    <div class="url">{{.HealthURL}}</div>
</body>
</html>`))

// Server implements the WebhookServer interface.
type Server struct {
	config         ports.WebhookServerConfig
	server         *http.Server
	listener       net.Listener
	tunnel         ports.Tunnel
	events         chan *ports.WebhookEvent
	handlers       []ports.WebhookEventHandler
	handlerSlots   chan struct{}
	seenSignatures map[string]time.Time
	stats          ports.WebhookServerStats
	mu             sync.RWMutex
	startedAt      time.Time
	closeOnce      sync.Once
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
		config:         config,
		events:         make(chan *ports.WebhookEvent, 100),
		handlerSlots:   make(chan struct{}, maxConcurrentHandlers),
		seenSignatures: make(map[string]time.Time),
		stats: ports.WebhookServerStats{
			LocalURL: localEndpointURL(config.Port, config.Path),
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

	// Start listener bound to loopback only. Tunnels (cloudflared, ngrok)
	// connect to 127.0.0.1 — there is no use case for accepting webhooks
	// directly from the LAN, and binding to 0.0.0.0 would let any host on
	// the local network forge webhook events.
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to start listener on port %d: %w", s.config.Port, err)
	}

	s.startedAt = time.Now()
	s.mu.Lock()
	s.stats.StartedAt = s.startedAt
	s.stats.LocalURL = localEndpointURL(s.config.Port, s.config.Path)
	s.mu.Unlock()

	// Start HTTP server in goroutine
	go func() {
		_ = s.server.Serve(s.listener) // Error handled on shutdown, ErrServerClosed expected
	}()

	// Start tunnel if configured
	if s.tunnel != nil {
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
	}

	return nil
}

// Stop stops the webhook server and tunnel. Safe to call more than once —
// the channel is closed under sync.Once, and the events channel is only
// closed after http.Server.Shutdown returns (which waits for in-flight
// handlers to complete) so producers cannot race the close.
func (s *Server) Stop() error {
	var errs []error

	s.closeOnce.Do(func() {
		// Stop tunnel first.
		if s.tunnel != nil {
			if err := s.tunnel.Stop(); err != nil {
				errs = append(errs, fmt.Errorf("tunnel stop: %w", err))
			}
		}

		// Stop HTTP server. Shutdown blocks until all in-flight requests have
		// finished, so when it returns no handler is sending to s.events.
		if s.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.server.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("server shutdown: %w", err))
			}
		}

		close(s.events)
	})

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

	// Cap request body size so a malicious sender on a public tunnel can't
	// drive unbounded RAM allocation. MaxBytesReader closes the body and
	// returns an error from ReadAll once the limit is exceeded.
	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// MaxBytesReader returns *http.MaxBytesError once the cap is hit; any
		// other read error (timeout, connection reset) is also surfaced as a
		// 413 to keep the response simple — the client cannot recover either
		// way.
		http.Error(w, "Request body too large or unreadable", http.StatusRequestEntityTooLarge)
		return
	}
	defer func() { _ = r.Body.Close() }()

	signature := r.Header.Get("X-Nylas-Signature")
	if s.config.WebhookSecret != "" {
		if signature == "" {
			http.Error(w, "Missing webhook signature", http.StatusUnauthorized)
			return
		}
		if !s.verifySignature(body, signature) {
			http.Error(w, "Invalid webhook signature", http.StatusForbidden)
			return
		}
	}

	// Parse webhook event
	event := &ports.WebhookEvent{
		Timestamp:  time.Now(),
		ReceivedAt: time.Now(),
		Headers:    make(map[string]string),
		RawBody:    body,
		Signature:  signature,
		Verified:   s.config.WebhookSecret != "",
	}

	// Copy relevant headers
	for k, v := range r.Header {
		if len(v) > 0 {
			event.Headers[k] = v[0]
		}
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

		// Replay protection. The signature has already been verified above.
		// When configured, reject events whose CloudEvents `time` field is
		// older than the allowed skew. Payloads without `time` are covered
		// by the signed-body dedupe below.
		if s.config.WebhookSecret != "" && s.config.MaxEventAge > 0 {
			if rawTime, ok := payload["time"].(string); ok {
				eventTime, terr := time.Parse(time.RFC3339, rawTime)
				if terr != nil {
					http.Error(w, "Invalid event timestamp", http.StatusBadRequest)
					return
				}
				skew := time.Since(eventTime)
				if skew > s.config.MaxEventAge || skew < -s.config.MaxEventAge {
					http.Error(w, "Event timestamp outside allowed skew", http.StatusUnauthorized)
					return
				}
			}
		}
	}

	if s.shouldSuppressSignedReplay(signature, time.Now()) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Duplicate webhook ignored"))
		return
	}

	// Update stats
	s.mu.Lock()
	s.stats.EventsReceived++
	s.stats.LastEventAt = time.Now()
	s.mu.Unlock()

	// Send to channel non-blocking. If the buffer is full we drop the
	// *new* event (not the oldest) — bump a stat so callers can see the
	// loss.
	select {
	case s.events <- event:
	default:
		s.mu.Lock()
		s.stats.EventsDropped++
		s.mu.Unlock()
	}

	// Call handlers. Goroutines are bounded by handlerSlots so a flood of
	// events combined with slow handlers cannot drive unbounded goroutine
	// creation. If the slot pool is saturated, we run synchronously rather
	// than dropping the call — handlers are typically cheap (channel sends).
	s.mu.RLock()
	handlers := s.handlers
	s.mu.RUnlock()

	for _, handler := range handlers {
		select {
		case s.handlerSlots <- struct{}{}:
			go func(h ports.WebhookEventHandler) {
				defer func() {
					<-s.handlerSlots
					// Recover so a buggy handler can't take down the server.
					_ = recover()
				}()
				h(event)
			}(handler)
		default:
			// Slot pool full — invoke inline to apply backpressure.
			func() {
				defer func() { _ = recover() }()
				handler(event)
			}()
		}
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
		"events_dropped":  stats.EventsDropped,
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
	data := struct {
		EventsReceived int
		StartedAt      string
		PublicURL      string
		HealthURL      string
	}{
		EventsReceived: stats.EventsReceived,
		StartedAt:      stats.StartedAt.Format(time.RFC3339),
		PublicURL:      stats.PublicURL,
		HealthURL:      s.GetPublicURL() + "/health",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = rootTemplate.Execute(w, data) // best-effort response
}

// verifySignature verifies the webhook signature using HMAC-SHA256.
func (s *Server) verifySignature(payload []byte, signature string) bool {
	return VerifySignature(payload, signature, s.config.WebhookSecret)
}

func (s *Server) shouldSuppressSignedReplay(signature string, now time.Time) bool {
	if s.config.WebhookSecret == "" || s.config.MaxEventAge <= 0 || signature == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := now.Add(-s.config.MaxEventAge)
	for key, seenAt := range s.seenSignatures {
		if seenAt.Before(cutoff) {
			delete(s.seenSignatures, key)
		}
	}

	if seenAt, ok := s.seenSignatures[signature]; ok && !seenAt.Before(cutoff) {
		return true
	}

	s.seenSignatures[signature] = now
	return false
}
