package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMCPServer records what reached it and answers each request with the
// next scripted response.
type fakeMCPServer struct {
	mu       sync.Mutex
	requests []recordedRequest
	script   []func(w http.ResponseWriter)
}

type recordedRequest struct {
	authorization string
	grantHeader   string
	body          map[string]any
}

func (f *fakeMCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)

	f.mu.Lock()
	f.requests = append(f.requests, recordedRequest{
		authorization: r.Header.Get("Authorization"),
		grantHeader:   r.Header.Get("X-Nylas-Grant-Id"),
		body:          body,
	})
	var respond func(w http.ResponseWriter)
	if len(f.script) > 0 {
		respond, f.script = f.script[0], f.script[1:]
	}
	f.mu.Unlock()

	if respond == nil {
		respond = respondOK
	}
	respond(w)
}

func (f *fakeMCPServer) recorded() []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedRequest(nil), f.requests...)
}

func respondOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`))
}

func respondStatus(status int, challenge string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		if challenge != "" {
			w.Header().Set("WWW-Authenticate", challenge)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"refused"}`))
	}
}

// fakeCredentials is a scripted ports.MCPCredentialSource.
type fakeCredentials struct {
	mu          sync.Mutex
	credentials []*domain.MCPCredential // returned in turn by Credential
	renewed     *domain.MCPCredential
	credErr     error
	renewErr    error
	calls       int
	renewCalls  []*domain.MCPCredential
}

func (f *fakeCredentials) Credential(context.Context) (*domain.MCPCredential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.credErr != nil {
		return nil, f.credErr
	}
	cred := f.credentials[min(f.calls, len(f.credentials)-1)]
	f.calls++
	return cred, nil
}

func (f *fakeCredentials) Renew(_ context.Context, rejected *domain.MCPCredential) (*domain.MCPCredential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.renewCalls = append(f.renewCalls, rejected)
	if f.renewErr != nil {
		return nil, f.renewErr
	}
	return f.renewed, nil
}

func newOAuthTestProxy(t *testing.T, script ...func(w http.ResponseWriter)) (*fakeMCPServer, *httptest.Server) {
	t.Helper()
	fake := &fakeMCPServer{script: script}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	return fake, server
}

func oauthCred(server *httptest.Server, token string, grantIDs ...string) *domain.MCPCredential {
	cred := &domain.MCPCredential{Token: token, Endpoint: server.URL, GrantScoped: true}
	for _, id := range grantIDs {
		cred.Grants = append(cred.Grants, domain.OAuthTokenGrant{ID: id, ApplicationID: "app-1"})
	}
	return cred
}

const toolCall = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{}}}`

func TestOAuthProxy_AsksForACredentialBeforeEveryRequest(t *testing.T) {
	// A fifteen-minute token must not be cached for the life of the process.
	fake, server := newOAuthTestProxy(t)
	creds := &fakeCredentials{credentials: []*domain.MCPCredential{
		oauthCred(server, "token-1"),
		oauthCred(server, "token-2"),
	}}
	proxy := NewOAuthProxy(creds)

	for range 2 {
		_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)
		require.NoError(t, err)
	}

	got := fake.recorded()
	require.Len(t, got, 2)
	assert.Equal(t, "Bearer token-1", got[0].authorization)
	assert.Equal(t, "Bearer token-2", got[1].authorization)
}

func TestOAuthProxy_SendsToTheCredentialsEndpoint(t *testing.T) {
	fake, server := newOAuthTestProxy(t)
	proxy := NewOAuthProxy(&fakeCredentials{credentials: []*domain.MCPCredential{oauthCred(server, "t")}})
	proxy.endpoint = "http://127.0.0.1:1" // a regional default the credential must override

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)
	require.NoError(t, err)
	assert.Len(t, fake.recorded(), 1)
}

func TestOAuthProxy_GrantHintOnlyForGrantsTheTokenNames(t *testing.T) {
	tests := []struct {
		name       string
		tokenGrant []string
		wantHint   string
	}{
		{"listed", []string{"grant-1", "grant-2"}, "grant-1"},
		{"not listed", []string{"grant-2"}, ""},
		{"no grants", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake, server := newOAuthTestProxy(t)
			proxy := NewOAuthProxy(&fakeCredentials{credentials: []*domain.MCPCredential{oauthCred(server, "t", tt.tokenGrant...)}})
			proxy.SetDefaultGrant("grant-1")

			req := parseRPC(toolCall)
			_, err := proxy.forward(t.Context(), req.raw, req.parsed)
			require.NoError(t, err)

			got := fake.recorded()
			require.Len(t, got, 1)
			assert.Equal(t, tt.wantHint, got[0].grantHeader)

			args := got[0].body["params"].(map[string]any)["arguments"].(map[string]any)
			if tt.wantHint == "" {
				assert.NotContains(t, args, "grant_id", "an unlisted grant must not be injected either")
			} else {
				assert.Equal(t, tt.wantHint, args["grant_id"])
			}
		})
	}
}

func TestOAuthProxy_RenewsOnceOn401ChallengeAndRetries(t *testing.T) {
	fake, server := newOAuthTestProxy(t,
		respondStatus(http.StatusUnauthorized, `Bearer error="invalid_token"`),
		respondOK,
	)
	stale := oauthCred(server, "stale")
	creds := &fakeCredentials{credentials: []*domain.MCPCredential{stale}, renewed: oauthCred(server, "fresh")}
	proxy := NewOAuthProxy(creds)

	resp, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)
	require.NoError(t, err)
	assert.Contains(t, string(resp), `"ok":true`)

	got := fake.recorded()
	require.Len(t, got, 2)
	assert.Equal(t, "Bearer stale", got[0].authorization)
	assert.Equal(t, "Bearer fresh", got[1].authorization)
	require.Len(t, creds.renewCalls, 1)
	assert.Same(t, stale, creds.renewCalls[0], "renew is told which token was refused")
}

func TestOAuthProxy_SecondRefusalTellsTheUserToLogIn(t *testing.T) {
	challenge := `Bearer error="invalid_token"`
	fake, server := newOAuthTestProxy(t,
		respondStatus(http.StatusUnauthorized, challenge),
		respondStatus(http.StatusUnauthorized, challenge),
		respondOK,
	)
	creds := &fakeCredentials{credentials: []*domain.MCPCredential{oauthCred(server, "a")}, renewed: oauthCred(server, "b")}
	proxy := NewOAuthProxy(creds)

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), OAuthLoginCommand)
	assert.Len(t, creds.renewCalls, 1, "renew once, never loop")
	assert.Len(t, fake.recorded(), 2)
}

func TestOAuthProxy_401WithoutChallengeIsNotRenewed(t *testing.T) {
	fake, server := newOAuthTestProxy(t, respondStatus(http.StatusUnauthorized, ""))
	creds := &fakeCredentials{credentials: []*domain.MCPCredential{oauthCred(server, "a")}, renewed: oauthCred(server, "b")}
	proxy := NewOAuthProxy(creds)

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), OAuthLoginCommand)
	assert.Empty(t, creds.renewCalls)
	assert.Len(t, fake.recorded(), 1)
}

func TestOAuthProxy_FailedRenewalTellsTheUserToLogIn(t *testing.T) {
	_, server := newOAuthTestProxy(t, respondStatus(http.StatusUnauthorized, `Bearer error="invalid_token"`))
	creds := &fakeCredentials{
		credentials: []*domain.MCPCredential{oauthCred(server, "a")},
		renewErr:    &domain.OAuthError{Code: "invalid_grant", StatusCode: http.StatusBadRequest},
	}
	proxy := NewOAuthProxy(creds)

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), OAuthLoginCommand)
	var oauthErr *domain.OAuthError
	assert.True(t, errors.As(err, &oauthErr), "the cause is kept for anyone who inspects it")
}

func TestOAuthProxy_InsufficientScopeNamesTheScope(t *testing.T) {
	fake, server := newOAuthTestProxy(t,
		respondStatus(http.StatusForbidden, `Bearer error="insufficient_scope", scope="email.send"`),
	)
	creds := &fakeCredentials{credentials: []*domain.MCPCredential{oauthCred(server, "a")}, renewed: oauthCred(server, "b")}
	proxy := NewOAuthProxy(creds)

	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"send_message","arguments":{}}}`)
	_, err := proxy.forward(t.Context(), req.raw, req.parsed)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "email.send")
	assert.Contains(t, err.Error(), OAuthLoginCommand)
	assert.Empty(t, creds.renewCalls, "a refresh cannot add a scope that was never granted")
	assert.Len(t, fake.recorded(), 1)
}

func TestOAuthProxy_NoSessionFailsBeforeAnyRequest(t *testing.T) {
	fake, _ := newOAuthTestProxy(t)
	proxy := NewOAuthProxy(&fakeCredentials{credErr: domain.ErrOAuthNotLoggedIn})

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)

	require.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)
	assert.Contains(t, err.Error(), OAuthLoginCommand)
	assert.Empty(t, fake.recorded(), "nothing may be sent without a credential")
}

func TestOAuthProxy_ErrorsNeverCarryTheToken(t *testing.T) {
	_, server := newOAuthTestProxy(t,
		respondStatus(http.StatusUnauthorized, `Bearer error="invalid_token"`),
		respondStatus(http.StatusUnauthorized, `Bearer error="invalid_token"`),
	)
	secret := "eyJ-very-secret-access-token"
	proxy := NewOAuthProxy(&fakeCredentials{
		credentials: []*domain.MCPCredential{oauthCred(server, secret)},
		renewed:     oauthCred(server, secret+"-2"),
	})

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), secret)
}

func TestAPIKeyProxy_401ChallengeIsNotRenewed(t *testing.T) {
	// An API key has nothing to refresh; the answer is reported as it was.
	fake, server := newOAuthTestProxy(t, respondStatus(http.StatusUnauthorized, `Bearer error="invalid_token"`))
	proxy := NewProxy("api-key", "us")
	proxy.endpoint = server.URL

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.NotContains(t, err.Error(), OAuthLoginCommand)
	assert.Len(t, fake.recorded(), 1)
	assert.Equal(t, "Bearer api-key", fake.recorded()[0].authorization)
}

func TestAPIKeyProxy_GrantHintIsUnrestricted(t *testing.T) {
	fake, server := newOAuthTestProxy(t)
	proxy := NewProxy("api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("grant-9")

	_, err := proxy.forward(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), nil)
	require.NoError(t, err)

	assert.Equal(t, "grant-9", fake.recorded()[0].grantHeader)
}
