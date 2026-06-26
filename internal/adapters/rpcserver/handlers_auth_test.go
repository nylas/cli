package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeAuthClient struct {
	ports.AuthClient

	buildAuthURL      func(domain.Provider, string, string, string) string
	getGrant          func(context.Context, string) (*domain.Grant, error)
	revokeGrant       func(context.Context, string) error
	createCustomGrant func(context.Context, string, map[string]any) (*domain.Grant, error)
	exchangeCode      func(context.Context, string, string, string) (*domain.Grant, error)

	buildAuthURLCalls      int
	getGrantCalls          int
	revokeGrantCalls       int
	createCustomGrantCalls int
	exchangeCodeCalls      int
}

func (f *fakeAuthClient) ExchangeCode(ctx context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error) {
	f.exchangeCodeCalls++
	if f.exchangeCode == nil {
		return nil, errors.New("unexpected ExchangeCode")
	}
	return f.exchangeCode(ctx, code, redirectURI, codeVerifier)
}

func (f *fakeAuthClient) BuildAuthURL(provider domain.Provider, redirectURI, state, codeChallenge string) string {
	f.buildAuthURLCalls++
	if f.buildAuthURL == nil {
		return ""
	}
	return f.buildAuthURL(provider, redirectURI, state, codeChallenge)
}

func (f *fakeAuthClient) GetGrant(ctx context.Context, grantID string) (*domain.Grant, error) {
	f.getGrantCalls++
	if f.getGrant == nil {
		return nil, errors.New("unexpected GetGrant")
	}
	return f.getGrant(ctx, grantID)
}

func (f *fakeAuthClient) RevokeGrant(ctx context.Context, grantID string) error {
	f.revokeGrantCalls++
	if f.revokeGrant == nil {
		return errors.New("unexpected RevokeGrant")
	}
	return f.revokeGrant(ctx, grantID)
}

func (f *fakeAuthClient) CreateCustomGrant(ctx context.Context, provider string, settings map[string]any) (*domain.Grant, error) {
	f.createCustomGrantCalls++
	if f.createCustomGrant == nil {
		return nil, errors.New("unexpected CreateCustomGrant")
	}
	return f.createCustomGrant(ctx, provider, settings)
}

func TestRegisterAuthHandlers_GrantGet(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		params       string
		defaultGrant string
		client       *fakeAuthClient
		assert       func(*testing.T, *fakeAuthClient, rpcTestResponse)
	}{
		{
			name:         "success",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeAuthClient{
				getGrant: func(ctx context.Context, grantID string) (*domain.Grant, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want default-grant", grantID)
					}
					return &domain.Grant{ID: "default-grant", Provider: domain.ProviderGoogle, Email: "user@example.com"}, nil
				},
			},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.getGrantCalls != 1 {
					t.Fatalf("GetGrant calls = %d, want 1", client.getGrantCalls)
				}

				var grant domain.Grant
				unmarshalResult(t, resp, &grant)
				if grant.ID != "default-grant" || grant.Provider != domain.ProviderGoogle || grant.Email != "user@example.com" {
					t.Fatalf("grant = %#v, want default-grant google user@example.com", grant)
				}
			},
		},
		{
			name:   "missing grant_id when no default",
			params: `{}`,
			client: &fakeAuthClient{},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.getGrantCalls != 0 {
					t.Fatalf("GetGrant calls = %d, want 0", client.getGrantCalls)
				}
			},
		},
		{
			name:         "client error",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "default-grant",
			client: &fakeAuthClient{
				getGrant: func(ctx context.Context, grantID string) (*domain.Grant, error) {
					if grantID != "grant-1" {
						t.Fatalf("grantID = %q, want grant-1", grantID)
					}
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
				if client.getGrantCalls != 1 {
					t.Fatalf("GetGrant calls = %d, want 1", client.getGrantCalls)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterAuthHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchAuthRequest(t, d, "auth.grant.get", tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}

func TestRegisterAuthHandlers_GrantRevoke(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		params       string
		defaultGrant string
		client       *fakeAuthClient
		assert       func(*testing.T, *fakeAuthClient, rpcTestResponse)
	}{
		{
			name:         "success",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeAuthClient{
				revokeGrant: func(ctx context.Context, grantID string) error {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want default-grant", grantID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.revokeGrantCalls != 1 {
					t.Fatalf("RevokeGrant calls = %d, want 1", client.revokeGrantCalls)
				}

				var result authGrantRevokeResult
				unmarshalResult(t, resp, &result)
				if !result.Revoked {
					t.Fatal("revoked = false, want true")
				}
			},
		},
		{
			name:   "missing grant",
			params: `{}`,
			client: &fakeAuthClient{},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.revokeGrantCalls != 0 {
					t.Fatalf("RevokeGrant calls = %d, want 0", client.revokeGrantCalls)
				}
			},
		},
		{
			name:         "client error",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "default-grant",
			client: &fakeAuthClient{
				revokeGrant: func(ctx context.Context, grantID string) error {
					if grantID != "grant-1" {
						t.Fatalf("grantID = %q, want grant-1", grantID)
					}
					return clientErr
				},
			},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
				if client.revokeGrantCalls != 1 {
					t.Fatalf("RevokeGrant calls = %d, want 1", client.revokeGrantCalls)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterAuthHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchAuthRequest(t, d, "auth.grant.revoke", tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}

func TestRegisterAuthHandlers_GrantCreateCustom(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name   string
		params string
		client *fakeAuthClient
		assert func(*testing.T, *fakeAuthClient, rpcTestResponse)
	}{
		{
			name:   "success",
			params: `{"provider":"imap","settings":{"username":"user@example.com","host":"imap.example.com"}}`,
			client: &fakeAuthClient{
				createCustomGrant: func(ctx context.Context, provider string, settings map[string]any) (*domain.Grant, error) {
					if provider != "imap" {
						t.Fatalf("provider = %q, want imap", provider)
					}
					if settings["username"] != "user@example.com" || settings["host"] != "imap.example.com" {
						t.Fatalf("settings = %#v, want username and host", settings)
					}
					return &domain.Grant{ID: "grant-1", Provider: domain.ProviderIMAP, Email: "user@example.com"}, nil
				},
			},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.createCustomGrantCalls != 1 {
					t.Fatalf("CreateCustomGrant calls = %d, want 1", client.createCustomGrantCalls)
				}

				var grant domain.Grant
				unmarshalResult(t, resp, &grant)
				if grant.ID != "grant-1" || grant.Provider != domain.ProviderIMAP {
					t.Fatalf("grant = %#v, want grant-1 imap", grant)
				}
			},
		},
		{
			name:   "missing provider",
			params: `{"settings":{"username":"user@example.com"}}`,
			client: &fakeAuthClient{},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.createCustomGrantCalls != 0 {
					t.Fatalf("CreateCustomGrant calls = %d, want 0", client.createCustomGrantCalls)
				}
			},
		},
		{
			name:   "client error",
			params: `{"provider":"imap"}`,
			client: &fakeAuthClient{
				createCustomGrant: func(ctx context.Context, provider string, settings map[string]any) (*domain.Grant, error) {
					if provider != "imap" {
						t.Fatalf("provider = %q, want imap", provider)
					}
					if settings != nil {
						t.Fatalf("settings = %#v, want nil", settings)
					}
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
				if client.createCustomGrantCalls != 1 {
					t.Fatalf("CreateCustomGrant calls = %d, want 1", client.createCustomGrantCalls)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterAuthHandlers(d, tt.client, "")

			resp := dispatchAuthRequest(t, d, "auth.grant.createCustom", tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}

func TestRegisterAuthHandlers_URL(t *testing.T) {
	tests := []struct {
		name   string
		params string
		client *fakeAuthClient
		assert func(*testing.T, *fakeAuthClient, rpcTestResponse)
	}{
		{
			name:   "success",
			params: `{"provider":"google","redirect_uri":"http://localhost:8080/callback","state":"state-1","code_challenge":"challenge-1"}`,
			client: &fakeAuthClient{
				buildAuthURL: func(provider domain.Provider, redirectURI, state, codeChallenge string) string {
					if provider != domain.ProviderGoogle {
						t.Fatalf("provider = %q, want google", provider)
					}
					if redirectURI != "http://localhost:8080/callback" {
						t.Fatalf("redirectURI = %q, want callback URI", redirectURI)
					}
					if state != "state-1" || codeChallenge != "challenge-1" {
						t.Fatalf("state/codeChallenge = %q/%q, want state-1/challenge-1", state, codeChallenge)
					}
					return "https://api.example.test/oauth?provider=google"
				},
			},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.buildAuthURLCalls != 1 {
					t.Fatalf("BuildAuthURL calls = %d, want 1", client.buildAuthURLCalls)
				}
				if client.getGrantCalls != 0 || client.revokeGrantCalls != 0 || client.createCustomGrantCalls != 0 {
					t.Fatalf("API calls = get:%d revoke:%d create:%d, want none", client.getGrantCalls, client.revokeGrantCalls, client.createCustomGrantCalls)
				}

				var result authURLResult
				unmarshalResult(t, resp, &result)
				if result.URL == "" {
					t.Fatal("url = empty, want non-empty")
				}
			},
		},
		{
			name:   "missing provider",
			params: `{"redirect_uri":"http://localhost:8080/callback"}`,
			client: &fakeAuthClient{},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.buildAuthURLCalls != 0 {
					t.Fatalf("BuildAuthURL calls = %d, want 0", client.buildAuthURLCalls)
				}
			},
		},
		{
			name:   "missing redirect_uri",
			params: `{"provider":"google"}`,
			client: &fakeAuthClient{},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.buildAuthURLCalls != 0 {
					t.Fatalf("BuildAuthURL calls = %d, want 0", client.buildAuthURLCalls)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterAuthHandlers(d, tt.client, "")

			resp := dispatchAuthRequest(t, d, "auth.url", tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}

func TestRegisterAuthHandlers_GrantExchange(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name   string
		params string
		client *fakeAuthClient
		assert func(*testing.T, *fakeAuthClient, rpcTestResponse)
	}{
		{
			name:   "success",
			params: `{"code":"auth-code-1","redirect_uri":"http://localhost:8080/callback","code_verifier":"verifier-1"}`,
			client: &fakeAuthClient{
				exchangeCode: func(_ context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error) {
					if code != "auth-code-1" || redirectURI != "http://localhost:8080/callback" || codeVerifier != "verifier-1" {
						t.Fatalf("args = %q/%q/%q, want auth-code-1/callback/verifier-1", code, redirectURI, codeVerifier)
					}
					return &domain.Grant{ID: "grant-123"}, nil
				},
			},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.exchangeCodeCalls != 1 {
					t.Fatalf("ExchangeCode calls = %d, want 1", client.exchangeCodeCalls)
				}
				var grant domain.Grant
				unmarshalResult(t, resp, &grant)
				if grant.ID != "grant-123" {
					t.Fatalf("grant ID = %q, want grant-123", grant.ID)
				}
			},
		},
		{
			name:   "missing code",
			params: `{"redirect_uri":"http://localhost:8080/callback"}`,
			client: &fakeAuthClient{},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.exchangeCodeCalls != 0 {
					t.Fatalf("ExchangeCode calls = %d, want 0", client.exchangeCodeCalls)
				}
			},
		},
		{
			name:   "missing redirect_uri",
			params: `{"code":"auth-code-1"}`,
			client: &fakeAuthClient{},
			assert: func(t *testing.T, client *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "client error",
			params: `{"code":"auth-code-1","redirect_uri":"http://localhost:8080/callback"}`,
			client: &fakeAuthClient{
				exchangeCode: func(context.Context, string, string, string) (*domain.Grant, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, _ *fakeAuthClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterAuthHandlers(d, tt.client, "")

			resp := dispatchAuthRequest(t, d, "auth.grant.exchange", tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}

func dispatchAuthRequest(t *testing.T, d *Dispatcher, method, params string) rpcTestResponse {
	t.Helper()

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`)
	got := d.Dispatch(context.Background(), raw)
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}

	var resp rpcTestResponse
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	return resp
}
