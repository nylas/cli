package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestNewCallbackServer(t *testing.T) {
	port := 8080
	server := NewCallbackServer(port)

	if server == nil {
		t.Error("NewCallbackServer() returned nil")
		return
	}
	if server.port != port {
		t.Errorf("port = %d, want %d", server.port, port)
	}
	if server.codeChan == nil {
		t.Error("codeChan is nil")
	}
	if server.errChan == nil {
		t.Error("errChan is nil")
	}
}

func TestCallbackServer_GetRedirectURI(t *testing.T) {
	tests := []struct {
		name string
		port int
		want string
	}{
		{
			name: "default port",
			port: 8080,
			want: "http://localhost:8080/callback",
		},
		{
			name: "custom port",
			port: 9000,
			want: "http://localhost:9000/callback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewCallbackServer(tt.port)
			got := server.GetRedirectURI()
			if got != tt.want {
				t.Errorf("GetRedirectURI() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCallbackServer_handleCallback_Success(t *testing.T) {
	server := NewCallbackServer(8080)

	// Create request with auth code
	req := httptest.NewRequest(http.MethodGet, "/callback?code=test-code-123", nil)
	w := httptest.NewRecorder()

	// Handle callback
	server.handleCallback(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Check that code was sent to channel
	select {
	case code := <-server.codeChan:
		if code != "test-code-123" {
			t.Errorf("code = %q, want %q", code, "test-code-123")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Code not sent to channel")
	}

	// Check HTML response
	body := w.Body.String()
	if !contains(body, "Authentication Successful") {
		t.Error("Response should contain success message")
	}
}

func TestCallbackServer_handleCallback_ErrorInQuery(t *testing.T) {
	server := NewCallbackServer(8080)

	// Create request with error
	req := httptest.NewRequest(http.MethodGet, "/callback?error=access_denied", nil)
	w := httptest.NewRecorder()

	// Handle callback
	server.handleCallback(w, req)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Check that error was sent to channel
	select {
	case err := <-server.errChan:
		if err == nil {
			t.Error("Expected error in channel")
		}
		if !contains(err.Error(), "access_denied") {
			t.Errorf("Error message = %q, should contain 'access_denied'", err.Error())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Error not sent to channel")
	}
}

func TestCallbackServer_handleCallback_MissingCode(t *testing.T) {
	server := NewCallbackServer(8080)

	// Create request without code or error
	req := httptest.NewRequest(http.MethodGet, "/callback", nil)
	w := httptest.NewRecorder()

	// Handle callback
	server.handleCallback(w, req)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Check that error was sent to channel
	select {
	case err := <-server.errChan:
		if err == nil {
			t.Error("Expected error in channel")
		}
		if !contains(err.Error(), "no authorization code received") {
			t.Errorf("Error message = %q, should contain 'no authorization code received'", err.Error())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Error not sent to channel")
	}
}

func TestCallbackServer_WaitForCallback_Success(t *testing.T) {
	server := NewCallbackServer(8080)

	// Send code to channel in background
	go func() {
		time.Sleep(10 * time.Millisecond)
		server.codeChan <- "test-code"
	}()

	ctx := context.Background()
	code, err := server.WaitForCallback(ctx)

	if err != nil {
		t.Errorf("WaitForCallback() error = %v, want nil", err)
	}
	if code != "test-code" {
		t.Errorf("code = %q, want %q", code, "test-code")
	}
}

func TestCallbackServer_WaitForCallback_Error(t *testing.T) {
	server := NewCallbackServer(8080)

	// Send error to channel in background
	testErr := domain.ErrAuthFailed
	go func() {
		time.Sleep(10 * time.Millisecond)
		server.errChan <- testErr
	}()

	ctx := context.Background()
	code, err := server.WaitForCallback(ctx)

	if err == nil {
		t.Error("WaitForCallback() error = nil, want error")
	}
	if err != testErr {
		t.Errorf("error = %v, want %v", err, testErr)
	}
	if code != "" {
		t.Errorf("code = %q, want empty string", code)
	}
}

func TestCallbackServer_WaitForCallback_Timeout(t *testing.T) {
	server := NewCallbackServer(8080)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	code, err := server.WaitForCallback(ctx)

	if err != domain.ErrAuthTimeout {
		t.Errorf("error = %v, want %v", err, domain.ErrAuthTimeout)
	}
	if code != "" {
		t.Errorf("code = %q, want empty string", code)
	}
}

func TestCallbackServer_Stop(t *testing.T) {
	server := NewCallbackServer(0) // Use port 0 for dynamic allocation

	// Test stopping before starting
	if err := server.Stop(); err != nil {
		t.Errorf("Stop() before Start() error = %v, want nil", err)
	}

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestCallbackServer_handleCallback_OnlyOnce(t *testing.T) {
	server := NewCallbackServer(8080)

	// First callback - should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/callback?code=first", nil)
	w1 := httptest.NewRecorder()
	server.handleCallback(w1, req1)

	// Second callback - should not overwrite
	req2 := httptest.NewRequest(http.MethodGet, "/callback?code=second", nil)
	w2 := httptest.NewRecorder()
	server.handleCallback(w2, req2)

	// Only first code should be in channel
	select {
	case code := <-server.codeChan:
		if code != "first" {
			t.Errorf("code = %q, want %q (first callback only)", code, "first")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Code not sent to channel")
	}

	// Channel should be empty now
	select {
	case code := <-server.codeChan:
		t.Errorf("Unexpected second code in channel: %q", code)
	default:
		// Expected - channel is empty
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
