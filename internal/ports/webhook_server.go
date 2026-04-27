// Package ports defines interfaces for webhook server and tunnel management.
package ports

import (
	"context"
	"time"
)

// WebhookEvent represents a received webhook event.
type WebhookEvent struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Timestamp  time.Time         `json:"timestamp"`
	Source     string            `json:"source"`
	GrantID    string            `json:"grant_id,omitempty"`
	Headers    map[string]string `json:"headers"`
	Body       map[string]any    `json:"body"`
	RawBody    []byte            `json:"-"`
	Signature  string            `json:"signature,omitempty"`
	Verified   bool              `json:"verified"`
	ReceivedAt time.Time         `json:"received_at"`
}

// WebhookServerConfig holds configuration for the webhook server.
type WebhookServerConfig struct {
	Port           int
	Path           string // Webhook endpoint path (default: /webhook)
	WebhookSecret  string // For signature verification
	TunnelProvider string // cloudflared, ngrok, or empty for no tunnel
	// MaxEventAge bounds how stale an event's `time` field (CloudEvents) can
	// be before the server rejects it as a replay. Zero disables the check.
	// Only applied when WebhookSecret is set; without HMAC the timestamp
	// would be attacker-controlled anyway.
	MaxEventAge time.Duration
}

// WebhookServerStats holds server statistics.
type WebhookServerStats struct {
	StartedAt      time.Time
	EventsReceived int
	// EventsDropped counts events the server discarded because the events
	// channel was full. Operators can use this to detect a slow consumer.
	EventsDropped  int
	LastEventAt    time.Time
	PublicURL      string
	LocalURL       string
	TunnelProvider string
	TunnelStatus   string
}

// WebhookEventHandler is called when a webhook event is received.
type WebhookEventHandler func(event *WebhookEvent)

// WebhookServer defines the interface for a local webhook receiver server.
type WebhookServer interface {
	// Start starts the webhook server.
	Start(ctx context.Context) error

	// Stop stops the webhook server.
	Stop() error

	// GetLocalURL returns the local server URL.
	GetLocalURL() string

	// GetPublicURL returns the public URL (from tunnel, if any).
	GetPublicURL() string

	// GetStats returns server statistics.
	GetStats() WebhookServerStats

	// OnEvent registers a handler for webhook events.
	OnEvent(handler WebhookEventHandler)

	// Events returns a channel for receiving webhook events.
	Events() <-chan *WebhookEvent
}

// TunnelConfig holds configuration for a tunnel.
type TunnelConfig struct {
	Provider string // cloudflared or ngrok
	LocalURL string // Local URL to tunnel to
}

// TunnelStatus represents the current tunnel status.
type TunnelStatus string

const (
	TunnelStatusStarting     TunnelStatus = "starting"
	TunnelStatusConnected    TunnelStatus = "connected"
	TunnelStatusReconnecting TunnelStatus = "reconnecting"
	TunnelStatusDisconnected TunnelStatus = "disconnected"
	TunnelStatusError        TunnelStatus = "error"
)

// Tunnel defines the interface for managing a tunnel to expose local server.
type Tunnel interface {
	// Start starts the tunnel and returns the public URL.
	Start(ctx context.Context) (publicURL string, err error)

	// Stop stops the tunnel.
	Stop() error

	// GetPublicURL returns the current public URL.
	GetPublicURL() string

	// Status returns the current tunnel status.
	Status() TunnelStatus

	// StatusMessage returns a human-readable status message.
	StatusMessage() string
}
