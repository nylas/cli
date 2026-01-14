package oauth

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// MockServer is a mock implementation of OAuthServer for testing.
type MockServer struct {
	Port                  int
	AuthCode              string
	StartCalled           bool
	StopCalled            bool
	WaitForCallbackCalled bool
	TimeoutAfter          time.Duration
}

// NewMockServer creates a new MockServer.
func NewMockServer(authCode string) *MockServer {
	return &MockServer{
		Port:     8080,
		AuthCode: authCode,
	}
}

// Start starts the mock server.
func (m *MockServer) Start() error {
	m.StartCalled = true
	return nil
}

// Stop stops the mock server.
func (m *MockServer) Stop() error {
	m.StopCalled = true
	return nil
}

// WaitForCallback waits for the OAuth callback.
func (m *MockServer) WaitForCallback(ctx context.Context) (string, error) {
	m.WaitForCallbackCalled = true

	if m.TimeoutAfter > 0 {
		select {
		case <-time.After(m.TimeoutAfter):
			return "", domain.ErrAuthTimeout
		case <-ctx.Done():
			return "", domain.ErrAuthTimeout
		}
	}

	if m.AuthCode == "" {
		return "", domain.ErrAuthFailed
	}
	return m.AuthCode, nil
}

// GetRedirectURI returns the redirect URI.
func (m *MockServer) GetRedirectURI() string {
	return "http://localhost:8080/callback"
}
