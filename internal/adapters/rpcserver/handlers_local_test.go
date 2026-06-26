package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeLocalAgentClient struct {
	ports.AgentClient

	listAgentAccounts func(context.Context) ([]domain.AgentAccount, error)
	getAgentAccount   func(context.Context, string) (*domain.AgentAccount, error)
}

func (f *fakeLocalAgentClient) ListAgentAccounts(ctx context.Context) ([]domain.AgentAccount, error) {
	if f.listAgentAccounts == nil {
		return nil, errors.New("unexpected ListAgentAccounts")
	}
	return f.listAgentAccounts(ctx)
}

func (f *fakeLocalAgentClient) GetAgentAccount(ctx context.Context, grantID string) (*domain.AgentAccount, error) {
	if f.getAgentAccount == nil {
		return nil, errors.New("unexpected GetAgentAccount")
	}
	return f.getAgentAccount(ctx, grantID)
}

type fakeLocalGrantStore struct {
	ports.GrantStore

	listGrants func() ([]domain.GrantInfo, error)
}

func (f *fakeLocalGrantStore) ListGrants() ([]domain.GrantInfo, error) {
	if f.listGrants == nil {
		return nil, errors.New("unexpected ListGrants")
	}
	return f.listGrants()
}

type fakeLocalConfigLoader struct {
	load func() (*domain.Config, error)
}

func (f *fakeLocalConfigLoader) Load() (*domain.Config, error) {
	if f.load == nil {
		return nil, errors.New("unexpected Load")
	}
	return f.load()
}

func TestRegisterAgentHandlers_Local(t *testing.T) {
	tests := []struct {
		name   string
		method string
		params string
		client *fakeLocalAgentClient
		assert func(*testing.T, rpcTestResponse)
	}{
		{
			name:   "agentAccount.list returns accounts",
			method: "agentAccount.list",
			params: `{}`,
			client: &fakeLocalAgentClient{
				listAgentAccounts: func(ctx context.Context) ([]domain.AgentAccount, error) {
					return []domain.AgentAccount{
						{ID: "grant-1", Provider: domain.ProviderNylas, Email: "agent@example.com", Name: "Agent One"},
					}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result struct {
					Accounts []domain.AgentAccount `json:"accounts"`
				}
				unmarshalResult(t, resp, &result)
				if len(result.Accounts) != 1 || result.Accounts[0].ID != "grant-1" || result.Accounts[0].Email != "agent@example.com" {
					t.Fatalf("accounts = %#v, want grant-1 agent@example.com", result.Accounts)
				}
			},
		},
		{
			name:   "agentAccount.get with grant_id returns account",
			method: "agentAccount.get",
			params: `{"grant_id":"grant-1"}`,
			client: &fakeLocalAgentClient{
				getAgentAccount: func(ctx context.Context, grantID string) (*domain.AgentAccount, error) {
					if grantID != "grant-1" {
						t.Fatalf("grantID = %q, want grant-1", grantID)
					}
					return &domain.AgentAccount{ID: "grant-1", Provider: domain.ProviderNylas, Email: "agent@example.com"}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var account domain.AgentAccount
				unmarshalResult(t, resp, &account)
				if account.ID != "grant-1" || account.Provider != domain.ProviderNylas {
					t.Fatalf("account = %#v, want grant-1 nylas", account)
				}
			},
		},
		{
			name:   "agentAccount.get missing grant_id returns invalid params",
			method: "agentAccount.get",
			params: `{}`,
			client: &fakeLocalAgentClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterAgentHandlers(d, tt.client)

			resp := dispatchLocalRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func TestRegisterGrantHandlers_Local(t *testing.T) {
	d := NewDispatcher()
	RegisterGrantHandlers(d, &fakeLocalGrantStore{
		listGrants: func() ([]domain.GrantInfo, error) {
			return []domain.GrantInfo{
				{ID: "grant-1", Email: "user@example.com", Provider: domain.ProviderGoogle},
			}, nil
		},
	})

	resp := dispatchLocalRequest(t, d, "grant.list", `{}`)
	requireNoRPCError(t, resp)

	var result struct {
		Grants []domain.GrantInfo `json:"grants"`
	}
	unmarshalResult(t, resp, &result)
	if len(result.Grants) != 1 || result.Grants[0].ID != "grant-1" || result.Grants[0].Provider != domain.ProviderGoogle {
		t.Fatalf("grants = %#v, want grant-1 google", result.Grants)
	}
}

func TestRegisterConfigHandlers_Local(t *testing.T) {
	d := NewDispatcher()
	RegisterConfigHandlers(d, &fakeLocalConfigLoader{
		load: func() (*domain.Config, error) {
			return &domain.Config{
				Region:       "eu",
				DefaultGrant: "grant-1",
				CallbackPort: 9008,
				Grants:       []domain.GrantInfo{{ID: "hidden-grant", Email: "hidden@example.com", Provider: domain.ProviderGoogle}},
				API:          &domain.APIConfig{BaseURL: "https://api.example.test", Timeout: "30s"},
				TUITheme:     "catppuccin",
				WorkingHours: &domain.WorkingHoursConfig{
					Default: &domain.DaySchedule{Enabled: true, Start: "09:00", End: "17:00"},
				},
				AI:        &domain.AIConfig{DefaultProvider: "openai"},
				GPG:       &domain.GPGConfig{DefaultKey: "key-id"},
				Dashboard: &domain.DashboardConfig{AccountBaseURL: "https://dashboard.example.test"},
			}, nil
		},
	})

	resp := dispatchLocalRequest(t, d, "config.read", `{}`)
	requireNoRPCError(t, resp)

	var result map[string]json.RawMessage
	unmarshalResult(t, resp, &result)
	requireJSONBool(t, result, "ai_configured", true)
	requireJSONBool(t, result, "gpg_configured", true)
	requireJSONBool(t, result, "dashboard_configured", true)

	for _, key := range []string{"ai", "gpg", "dashboard", "grants"} {
		if _, ok := result[key]; ok {
			t.Fatalf("result contains %q key: %s", key, resp.Result)
		}
	}

	var whitelisted struct {
		Region       string `json:"region"`
		DefaultGrant string `json:"default_grant"`
		CallbackPort int    `json:"callback_port"`
		TUITheme     string `json:"tui_theme"`
		API          struct {
			BaseURL string `json:"base_url"`
			Timeout string `json:"timeout"`
		} `json:"api"`
		WorkingHours *domain.WorkingHoursConfig `json:"working_hours"`
	}
	unmarshalResult(t, resp, &whitelisted)
	if whitelisted.Region != "eu" || whitelisted.DefaultGrant != "grant-1" || whitelisted.CallbackPort != 9008 || whitelisted.TUITheme != "catppuccin" {
		t.Fatalf("config result = %+v, want whitelisted scalar fields", whitelisted)
	}
	if whitelisted.API.BaseURL != "https://api.example.test" || whitelisted.API.Timeout != "30s" {
		t.Fatalf("api = %+v, want base_url and timeout", whitelisted.API)
	}
	if whitelisted.WorkingHours == nil {
		t.Fatal("working_hours = nil, want configured working hours")
	}
}

func dispatchLocalRequest(t *testing.T, d *Dispatcher, method, params string) rpcTestResponse {
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

func requireJSONBool(t *testing.T, fields map[string]json.RawMessage, key string, want bool) {
	t.Helper()

	raw, ok := fields[key]
	if !ok {
		t.Fatalf("missing %q key", key)
	}
	var got bool
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal %q: %v", key, err)
	}
	if got != want {
		t.Fatalf("%s = %t, want %t", key, got, want)
	}
}
