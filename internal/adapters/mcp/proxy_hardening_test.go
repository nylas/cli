package mcp

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthProxy_RetryAfterRenewalIsNormalizedToo(t *testing.T) {
	// The first attempt's normalization used to write into the caller's
	// nested arguments, so the retry found nothing to change and sent the
	// original bytes: an integer start the upstream schema rejects.
	fake, server := newOAuthTestProxy(t,
		respondStatus(http.StatusUnauthorized, `Bearer error="invalid_token"`),
		respondOK,
	)
	creds := &fakeCredentials{
		credentials: []*domain.MCPCredential{oauthCred(server, "stale")},
		renewed:     oauthCred(server, "fresh"),
	}
	proxy := NewOAuthProxy(creds)

	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1700000000}}}}`)
	_, err := proxy.forward(t.Context(), req.raw, req.parsed)
	require.NoError(t, err)

	got := fake.recorded()
	require.Len(t, got, 2)
	for i, sent := range got {
		params := sent.body["params"].(map[string]any)["arguments"].(map[string]any)["get_all_query_parameters"].(map[string]any)
		assert.Equal(t, "1700000000", params["start"], "attempt %d", i+1)
	}
	callerParams := req.parsed.Params.Arguments["get_all_query_parameters"].(map[string]any)
	assert.Equal(t, float64(1700000000), callerParams["start"], "the caller's request is never modified")
}

func TestProxy_RewrittenNotificationKeepsNoID(t *testing.T) {
	// Adding "id": null while injecting a grant would turn a notification
	// into a request, and the server's reply would then be relayed.
	fake, server := newOAuthTestProxy(t)
	proxy := NewProxy("api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("grant-1")

	in := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"list_messages","arguments":{}}}` + "\n"
	var out bytes.Buffer
	require.NoError(t, proxy.serve(t.Context(), strings.NewReader(in), &out))

	got := fake.recorded()
	require.Len(t, got, 1)
	assert.NotContains(t, got[0].body, "id")
	assert.Equal(t, "grant-1", got[0].body["params"].(map[string]any)["arguments"].(map[string]any)["grant_id"])
	assert.Empty(t, out.String(), "the server's reply to a notification is not relayed")
}

func TestProxy_NotificationsAreNeverAnswered(t *testing.T) {
	refuse := respondStatus(http.StatusInternalServerError, "")
	_, server := newOAuthTestProxy(t, refuse, refuse)
	proxy := NewProxy("api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetGrantStore(&mockGrantStore{grants: []domain.GrantInfo{{ID: "grant-1", Email: "user@example.com"}}})

	in := strings.Join([]string{
		// A batch of notifications whose forwarding fails.
		`[{"jsonrpc":"2.0","method":"notifications/a"},{"jsonrpc":"2.0","method":"notifications/b"}]`,
		// params as an array does not parse as a request.
		`{"jsonrpc":"2.0","method":"notifications/c","params":[1]}`,
		// An id-less call the proxy would otherwise answer locally.
		`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_grant","arguments":{}}}`,
	}, "\n") + "\n"
	var out bytes.Buffer

	require.NoError(t, proxy.serve(t.Context(), strings.NewReader(in), &out))

	assert.Empty(t, out.String())
}

func TestAPIKeyProxy_InsufficientScopeDoesNotSuggestAnOAuthLogin(t *testing.T) {
	_, server := newOAuthTestProxy(t,
		respondStatus(http.StatusForbidden, `Bearer error="insufficient_scope", scope="email.send"`),
	)
	proxy := NewProxy("api-key", "us")
	proxy.endpoint = server.URL

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.NotContains(t, err.Error(), OAuthLoginCommand)
}

func TestProxy_ServerErrorTextCannotRewriteTheTerminal(t *testing.T) {
	_, server := newOAuthTestProxy(t, func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad\x1b]0;pwned\x07\r\nforged log line"))
	})
	proxy := NewProxy("api-key", "us")
	proxy.endpoint = server.URL

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "\x1b")
	assert.NotContains(t, err.Error(), "\n")
	assert.Contains(t, err.Error(), "forged log line")
}
