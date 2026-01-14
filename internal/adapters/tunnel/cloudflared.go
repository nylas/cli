// Package tunnel provides tunnel implementations for exposing local servers.
package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nylas/cli/internal/ports"
)

// CloudflaredTunnel implements the Tunnel interface using cloudflared.
type CloudflaredTunnel struct {
	localURL      string
	publicURL     string
	status        ports.TunnelStatus
	statusMessage string
	cmd           *exec.Cmd
	cancel        context.CancelFunc
	mu            sync.RWMutex
	urlChan       chan string
	errChan       chan error
}

// NewCloudflaredTunnel creates a new cloudflared tunnel.
func NewCloudflaredTunnel(localURL string) *CloudflaredTunnel {
	return &CloudflaredTunnel{
		localURL: localURL,
		status:   ports.TunnelStatusDisconnected,
		urlChan:  make(chan string, 1),
		errChan:  make(chan error, 1),
	}
}

// Start starts the cloudflared tunnel and returns the public URL.
func (t *CloudflaredTunnel) Start(ctx context.Context) (string, error) {
	// Check if cloudflared is installed
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return "", fmt.Errorf("cloudflared not found in PATH. Install it with: brew install cloudflared")
	}

	t.mu.Lock()
	t.status = ports.TunnelStatusStarting
	t.statusMessage = "Starting cloudflared tunnel..."
	t.mu.Unlock()

	// Create a cancellable context
	tunnelCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	// Validate localURL to prevent command injection
	if err := validateLocalURL(t.localURL); err != nil {
		return "", err
	}

	// Start cloudflared tunnel
	// #nosec G204 -- localURL is validated above (localhost only, valid port, http/https scheme)
	t.cmd = exec.CommandContext(tunnelCtx, "cloudflared", "tunnel", "--url", t.localURL)

	// Get stderr pipe (cloudflared outputs to stderr)
	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	if err := t.cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start cloudflared: %w", err)
	}

	// Parse output to extract URL
	go func() {
		scanner := bufio.NewScanner(stderr)
		urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
		connectedRegex := regexp.MustCompile(`Registered tunnel connection|connection=`)

		for scanner.Scan() {
			line := scanner.Text()

			// Look for the public URL
			if match := urlRegex.FindString(line); match != "" {
				t.mu.Lock()
				if t.publicURL == "" {
					t.publicURL = match
					t.mu.Unlock()
					// Send URL through channel (non-blocking)
					select {
					case t.urlChan <- match:
					default:
					}
				} else {
					t.mu.Unlock()
				}
			}

			// Check for connection status
			if connectedRegex.MatchString(line) {
				t.mu.Lock()
				t.status = ports.TunnelStatusConnected
				t.statusMessage = "Tunnel connected"
				t.mu.Unlock()
			}

			// Check for reconnection
			if strings.Contains(line, "Retrying") || strings.Contains(line, "reconnect") {
				t.mu.Lock()
				t.status = ports.TunnelStatusReconnecting
				t.statusMessage = "Reconnecting..."
				t.mu.Unlock()
			}
		}
	}()

	// Wait for URL with timeout
	select {
	case url := <-t.urlChan:
		t.mu.Lock()
		t.status = ports.TunnelStatusConnected
		t.statusMessage = fmt.Sprintf("Connected: %s", url)
		t.mu.Unlock()
		return url, nil
	case err := <-t.errChan:
		return "", err
	case <-time.After(30 * time.Second):
		_ = t.Stop() // Ignore error - we're returning a timeout error anyway
		return "", fmt.Errorf("timeout waiting for cloudflared tunnel URL")
	case <-ctx.Done():
		_ = t.Stop() // Ignore error - we're returning context error anyway
		return "", ctx.Err()
	}
}

// Stop stops the cloudflared tunnel.
func (t *CloudflaredTunnel) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancel != nil {
		t.cancel()
	}

	if t.cmd != nil && t.cmd.Process != nil {
		// Give it a moment to cleanup gracefully
		done := make(chan error, 1)
		go func() {
			done <- t.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited
		case <-time.After(2 * time.Second):
			// Force kill - ignore error as process may already be dead
			_ = t.cmd.Process.Kill()
		}
	}

	t.status = ports.TunnelStatusDisconnected
	t.statusMessage = "Tunnel stopped"
	t.publicURL = ""

	return nil
}

// GetPublicURL returns the current public URL.
func (t *CloudflaredTunnel) GetPublicURL() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.publicURL
}

// Status returns the current tunnel status.
func (t *CloudflaredTunnel) Status() ports.TunnelStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// StatusMessage returns a human-readable status message.
func (t *CloudflaredTunnel) StatusMessage() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.statusMessage
}

// IsCloudflaredInstalled checks if cloudflared is available in PATH.
func IsCloudflaredInstalled() bool {
	_, err := exec.LookPath("cloudflared")
	return err == nil
}

// validateLocalURL validates that the URL is a safe localhost URL with valid port.
func validateLocalURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate scheme
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid scheme: must be http or https")
	}

	// Validate host is localhost only
	hostname := parsed.Hostname()
	if hostname != "localhost" && hostname != "127.0.0.1" {
		return fmt.Errorf("invalid host: must be localhost or 127.0.0.1")
	}

	// Validate port is present and valid
	portStr := parsed.Port()
	if portStr == "" {
		return fmt.Errorf("port is required")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: must be a number")
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: must be between 1 and 65535")
	}

	// Validate no unexpected path components that could be shell injection
	if parsed.User != nil {
		return fmt.Errorf("URL must not contain user credentials")
	}

	return nil
}
