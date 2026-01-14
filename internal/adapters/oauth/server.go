// Package oauth provides OAuth callback server implementation.
package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// CallbackServer implements the OAuth callback server.
type CallbackServer struct {
	port     int
	server   *http.Server
	listener net.Listener
	codeChan chan string
	errChan  chan error
	once     sync.Once
}

// NewCallbackServer creates a new callback server.
func NewCallbackServer(port int) *CallbackServer {
	return &CallbackServer{
		port:     port,
		codeChan: make(chan string, 1),
		errChan:  make(chan error, 1),
	}
}

// Start starts the callback server.
func (s *CallbackServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}

	go func() {
		if err := s.server.Serve(s.listener); err != http.ErrServerClosed {
			s.errChan <- err
		}
	}()

	return nil
}

// Stop stops the callback server.
func (s *CallbackServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// WaitForCallback waits for the OAuth callback and returns the auth code.
func (s *CallbackServer) WaitForCallback(ctx context.Context) (string, error) {
	select {
	case code := <-s.codeChan:
		return code, nil
	case err := <-s.errChan:
		return "", err
	case <-ctx.Done():
		return "", domain.ErrAuthTimeout
	}
}

// GetRedirectURI returns the redirect URI for OAuth.
func (s *CallbackServer) GetRedirectURI() string {
	return fmt.Sprintf("http://localhost:%d/callback", s.port)
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		errMsg := r.URL.Query().Get("error")
		if errMsg == "" {
			errMsg = "no authorization code received"
		}
		s.once.Do(func() {
			s.errChan <- fmt.Errorf("%w: %s", domain.ErrAuthFailed, errMsg)
		})
		http.Error(w, "Authentication failed: "+errMsg, http.StatusBadRequest)
		return
	}

	s.once.Do(func() {
		s.codeChan <- code
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Authentication Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
               display: flex; justify-content: center; align-items: center; height: 100vh;
               margin: 0; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); }
        .container { text-align: center; background: white; padding: 3rem; border-radius: 1rem;
                     box-shadow: 0 10px 40px rgba(0,0,0,0.2); }
        h1 { color: #22c55e; margin-bottom: 1rem; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>✓ Authentication Successful</h1>
        <p>You can close this window and return to the terminal.</p>
    </div>
</body>
</html>`)
}
