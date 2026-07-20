package chat

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerStart_SecurityHeaders verifies the running chat server actually
// serves the webguard security headers (strict CSP). The middleware is unit
// tested in webguard; this guards the wiring in Start(), which a refactor
// could silently drop without any other test failing.
func TestServerStart_SecurityHeaders(t *testing.T) {
	memory := setupMemoryStore(t)
	agent := Agent{Type: AgentClaude, Version: "1.0"}
	server := NewServer(freeLoopbackAddr(t), &agent, []Agent{agent}, nil, "grant-id", memory)

	// Start blocks on ListenAndServe and the server has no shutdown seam, so
	// it runs until the test binary exits.
	go func() { _ = server.Start() }()

	resp := waitForServer(t, "http://"+server.addr+"/api/health")
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	csp := resp.Header.Get("Content-Security-Policy")
	require.NotEmpty(t, csp, "chat server response is missing the CSP header — SecurityHeadersMiddleware not wired in Start()")
	assert.Contains(t, csp, "script-src 'self';", "CSP must keep strict script-src")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "SAMEORIGIN", resp.Header.Get("X-Frame-Options"))
}

// freeLoopbackAddr reserves a loopback port and returns it as host:port.
func freeLoopbackAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

// waitForServer polls url until the server responds or the deadline passes.
func waitForServer(t *testing.T, url string) *http.Response {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(url) // #nosec G107 -- loopback test URL
		if err == nil {
			return resp
		}
		if time.Now().After(deadline) {
			t.Fatalf("server at %s did not come up: %v", url, err)
		}
		if !strings.Contains(err.Error(), "connection refused") {
			t.Logf("waiting for server: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestServer_ConcurrentSetAgentAndHandlers exercises agent switching racing
// with handlers that read the agent and context builder. s.agent and
// s.context are written under agentMu by SetAgent, so every read must go
// through the ActiveAgent/ActiveContext accessors — run with -race to verify.
func TestServer_ConcurrentSetAgentAndHandlers(t *testing.T) {
	t.Parallel()

	memory := setupMemoryStore(t)
	agents := []Agent{
		{Type: AgentClaude, Version: "1.0"},
		{Type: AgentOllama, Version: "1.0"},
	}
	server := NewServer("127.0.0.1:0", &agents[0], agents, nil, "grant-id", memory)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				server.SetAgent(AgentClaude)
			} else {
				server.SetAgent(AgentOllama)
			}
		}(i)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/conversations", nil)
			w := httptest.NewRecorder()
			server.handleConversations(w, req)
			assert.Equal(t, http.StatusCreated, w.Code)
			require.NotNil(t, server.ActiveContext())
			require.NotNil(t, server.ActiveAgent())
		}()
	}
	wg.Wait()
}
