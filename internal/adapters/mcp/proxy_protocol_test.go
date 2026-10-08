package mcp

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// headerRecorder answers like the stateless hosted server: it hands out a
// session id the proxy must ignore, and negotiates a protocol version.
type headerRecorder struct {
	mu      sync.Mutex
	headers []http.Header
}

func (h *headerRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.headers = append(h.headers, r.Header.Clone())
	h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Mcp-Session-Id", "should-be-ignored")
	_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","instructions":"x"}}`))
}

func (h *headerRecorder) all() []http.Header {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]http.Header(nil), h.headers...)
}

func newProtocolProxy(t *testing.T) (*Proxy, *headerRecorder) {
	t.Helper()
	rec := &headerRecorder{}
	server := httptest.NewServer(rec)
	t.Cleanup(server.Close)
	proxy := NewProxy("k", "us")
	proxy.endpoint = server.URL
	return proxy, rec
}

func TestProxy_SendsDocumentedProtocolHeaders(t *testing.T) {
	proxy, rec := newProtocolProxy(t)

	for _, raw := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_messages","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"prompts/get","params":{"name":"summarise"}}`,
	} {
		req := parseRPC(raw)
		_, err := proxy.forward(t.Context(), req.raw, req.parsed)
		require.NoError(t, err)
	}

	got := rec.all()
	require.Len(t, got, 4)

	assert.Equal(t, "initialize", got[0].Get("Mcp-Method"))
	assert.Empty(t, got[0].Get("Mcp-Protocol-Version"), "nothing is negotiated before initialize answers")

	assert.Equal(t, "tools/list", got[1].Get("Mcp-Method"))
	assert.Empty(t, got[1].Get("Mcp-Name"))
	assert.Equal(t, "2025-06-18", got[1].Get("Mcp-Protocol-Version"), "the negotiated version follows")

	assert.Equal(t, "tools/call", got[2].Get("Mcp-Method"))
	assert.Equal(t, "list_messages", got[2].Get("Mcp-Name"), "the server refuses a Mcp-Name that disagrees with the body")

	assert.Equal(t, "prompts/get", got[3].Get("Mcp-Method"))
	assert.Equal(t, "summarise", got[3].Get("Mcp-Name"))

	for i, h := range got {
		assert.Empty(t, h.Get("Mcp-Session-Id"), "request %d: the server is stateless", i)
	}
}

func TestProxy_UnsafeNamesStayInTheBody(t *testing.T) {
	proxy, rec := newProtocolProxy(t)

	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"bad name\r\nX-Injected: 1","arguments":{}}}`)
	_, err := proxy.forward(t.Context(), req.raw, req.parsed)
	require.NoError(t, err)

	got := rec.all()
	require.Len(t, got, 1)
	assert.Empty(t, got[0].Get("Mcp-Method"), "no method header without its name")
	assert.Empty(t, got[0].Get("Mcp-Name"))
	assert.Empty(t, got[0].Get("X-Injected"))
}

func TestProxy_UnparsedRequestGetsNoRoutingHeaders(t *testing.T) {
	proxy, rec := newProtocolProxy(t)

	_, err := proxy.forward(t.Context(), []byte(`[{"jsonrpc":"2.0","id":1,"method":"ping"}]`), nil)
	require.NoError(t, err)

	assert.Empty(t, rec.all()[0].Get("Mcp-Method"))
}

func TestRememberProtocolVersion_IgnoresMalformedValues(t *testing.T) {
	proxy := NewProxy("k", "us")
	proxy.rememberProtocolVersion([]byte(`{"result":{"protocolVersion":"2025-06-18\r\nX: y"}}`))
	assert.Empty(t, proxy.protocolVersion)

	proxy.rememberProtocolVersion([]byte(`{"result":{"protocolVersion":"2026-07-28"}}`))
	assert.Equal(t, "2026-07-28", proxy.protocolVersion)
}
