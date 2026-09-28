package oauth

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

func TestLoopbackIPCallbackServer_AdvertisesTheAddressItBinds(t *testing.T) {
	// The Nylas authorization server registers http://127.0.0.1/callback for
	// the CLI's static client, and never treats localhost and 127.0.0.1 as the
	// same host. Advertising the literal it listens on keeps the two in step.
	server := NewLoopbackIPCallbackServer(0)
	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = server.Stop() }()

	want := "http://127.0.0.1:" + strconv.Itoa(server.port) + "/callback"
	if got := server.GetRedirectURI(); got != want {
		t.Fatalf("GetRedirectURI() = %q, want %q", got, want)
	}
	if len(server.listeners) != 1 {
		t.Fatalf("listeners = %d, want 1 (IPv4 loopback only)", len(server.listeners))
	}
	addr, ok := server.listeners[0].Addr().(*net.TCPAddr)
	if !ok || !addr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("listener bound to %v, want 127.0.0.1", server.listeners[0].Addr())
	}
}

func TestCallbackServer_StartAcceptsIPv6LoopbackForAdvertisedLocalhost(t *testing.T) {
	probe, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback is not available: %v", err)
	}
	_ = probe.Close()

	server := NewCallbackServer(0)
	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = server.Stop() }()
	server.setExpectedState("test-state")

	client := &http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	resp, err := client.Get("http://[::1]:" + strconv.Itoa(server.port) + "/callback?code=test-code&state=test-state")
	if err != nil {
		t.Fatalf("IPv6 loopback callback request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("IPv6 loopback callback status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestCallbackServer_handleCallback_Success(t *testing.T) {
	server := NewCallbackServer(8080)
	server.setExpectedState("test-state-123")

	// Create request with auth code
	req := httptest.NewRequest(http.MethodGet, "/callback?code=test-code-123&state=test-state-123", nil)
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
	// An error redirect that answers this login (it carries the state) ends
	// the wait with the server's error code.
	server := NewCallbackServer(8080)
	server.SetExpectedState("test-state")

	req := httptest.NewRequest(http.MethodGet, "/callback?error=access_denied&state=test-state", nil)
	w := httptest.NewRecorder()
	server.handleCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	select {
	case err := <-server.errChan:
		if !errors.Is(err, domain.ErrAuthFailed) || !contains(err.Error(), "access_denied") {
			t.Errorf("error = %v, want ErrAuthFailed with access_denied", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Error not sent to channel")
	}
}
func TestCallbackServer_handleCallback_MissingCode(t *testing.T) {
	server := NewCallbackServer(8080)
	server.SetExpectedState("test-state")

	req := httptest.NewRequest(http.MethodGet, "/callback?state=test-state", nil)
	w := httptest.NewRecorder()
	server.handleCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	select {
	case err := <-server.errChan:
		if !contains(err.Error(), "no authorization code received") {
			t.Errorf("Error message = %q, should contain 'no authorization code received'", err.Error())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Error not sent to channel")
	}
}

func TestCallbackServer_handleCallback_StrayRequestsDoNotEndTheLogin(t *testing.T) {
	// The port is fixed and reachable by any page in the user's browser. A
	// request without this login's state is refused, but must not use up the
	// one callback the real redirect needs.
	for _, query := range []string{
		"",
		"?error=access_denied",
		"?error=%1b[31mred",
		"?code=stolen",
		"?code=stolen&state=wrong-state",
		"?error=access_denied&state=wrong-state",
	} {
		t.Run(query, func(t *testing.T) {
			server := NewCallbackServer(8080)
			server.SetExpectedState("test-state")

			w := httptest.NewRecorder()
			server.handleCallback(w, httptest.NewRequest(http.MethodGet, "/callback"+query, nil))
			if w.Code != http.StatusBadRequest {
				t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
			}
			select {
			case err := <-server.errChan:
				t.Fatalf("a stray request ended the login: %v", err)
			case code := <-server.codeChan:
				t.Fatalf("a stray request yielded a code: %q", code)
			default:
			}
		})
	}
}

func TestCallbackServer_handleCallback_ErrorCodeIsAllowListed(t *testing.T) {
	// The error code reaches the terminal. Escape sequences in it would be
	// interpreted, so anything that is not an RFC 6749 error code is dropped.
	server := NewCallbackServer(8080)
	server.SetExpectedState("test-state")

	w := httptest.NewRecorder()
	server.handleCallback(w, httptest.NewRequest(http.MethodGet, "/callback?state=test-state&error=%1b%5b31mowned", nil))

	select {
	case err := <-server.errChan:
		if strings.ContainsRune(err.Error(), 0x1b) || contains(err.Error(), "owned") {
			t.Errorf("error = %q, want the raw value dropped", err.Error())
		}
		if strings.ContainsRune(w.Body.String(), 0x1b) {
			t.Errorf("response body reflects the raw value: %q", w.Body.String())
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Error not sent to channel")
	}
}
func TestCallbackServer_handleCallback_InvalidState(t *testing.T) {
	server := NewCallbackServer(8080)
	server.setExpectedState("expected-state")

	req := httptest.NewRequest(http.MethodGet, "/callback?code=test-code-123&state=wrong-state", nil)
	w := httptest.NewRecorder()
	server.handleCallback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	select {
	case code := <-server.codeChan:
		t.Errorf("unexpected code received: %q", code)
	case err := <-server.errChan:
		t.Errorf("a wrong state must be refused without ending the login: %v", err)
	default:
	}
}
func TestCallbackServer_WaitForCallback_InvalidState(t *testing.T) {
	// A wrong-state request is refused, and the real redirect that follows
	// still completes the login.
	server := NewCallbackServer(8080)
	server.SetExpectedState("expected-state")

	w := httptest.NewRecorder()
	server.handleCallback(w, httptest.NewRequest(http.MethodGet, "/callback?code=stolen&state=wrong-state", nil))
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}

	w = httptest.NewRecorder()
	server.handleCallback(w, httptest.NewRequest(http.MethodGet, "/callback?code=real-code&state=expected-state", nil))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	code, err := server.WaitForCallback(ctx, "expected-state")
	if err != nil {
		t.Fatalf("WaitForCallback() error = %v", err)
	}
	if code != "real-code" {
		t.Errorf("code = %q, want real-code", code)
	}
}

func TestCallbackServer_RedirectBeforeWaitIsAccepted(t *testing.T) {
	// A browser can redirect before the goroutine calling WaitForCallback
	// runs. With the state set up front, that redirect is still the login.
	server := NewCallbackServer(8080)
	server.SetExpectedState("expected-state")

	w := httptest.NewRecorder()
	server.handleCallback(w, httptest.NewRequest(http.MethodGet, "/callback?code=fast&state=expected-state", nil))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	code, err := server.WaitForCallback(ctx, "expected-state")
	if err != nil || code != "fast" {
		t.Fatalf("WaitForCallback() = %q, %v; want fast, nil", code, err)
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
	code, err := server.WaitForCallback(ctx, "expected-state")

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
	code, err := server.WaitForCallback(ctx, "expected-state")

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

	code, err := server.WaitForCallback(ctx, "expected-state")

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
	server.setExpectedState("test-state")
	req1 := httptest.NewRequest(http.MethodGet, "/callback?code=first&state=test-state", nil)
	w1 := httptest.NewRecorder()
	server.handleCallback(w1, req1)

	// Second callback - should not overwrite
	req2 := httptest.NewRequest(http.MethodGet, "/callback?code=second&state=test-state", nil)
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
