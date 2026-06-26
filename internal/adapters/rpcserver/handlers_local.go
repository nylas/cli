package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type agentAccountGetParams struct {
	GrantID string `json:"grant_id"`
}

type agentAccountListResult struct {
	Accounts []domain.AgentAccount `json:"accounts"`
}

type grantListResult struct {
	Grants []domain.GrantInfo `json:"grants"`
}

type configLoader interface {
	Load() (*domain.Config, error)
}

type configReadResult struct {
	Region              string                     `json:"region"`
	DefaultGrant        string                     `json:"default_grant"`
	CallbackPort        int                        `json:"callback_port"`
	TUITheme            string                     `json:"tui_theme"`
	API                 *configReadAPI             `json:"api,omitempty"`
	WorkingHours        *domain.WorkingHoursConfig `json:"working_hours"`
	AIConfigured        bool                       `json:"ai_configured"`
	GPGConfigured       bool                       `json:"gpg_configured"`
	DashboardConfigured bool                       `json:"dashboard_configured"`
}

type configReadAPI struct {
	BaseURL string `json:"base_url"`
	Timeout string `json:"timeout"`
}

func RegisterAgentHandlers(d *Dispatcher, client ports.AgentClient) {
	d.Register("agentAccount.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct{}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		accounts, err := client.ListAgentAccounts(ctx)
		if err != nil {
			return nil, fmt.Errorf("agentAccount.list: %w", err)
		}
		return agentAccountListResult{Accounts: accounts}, nil
	})

	d.Register("agentAccount.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p agentAccountGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.GrantID == "" {
			return nil, NewRPCError(InvalidParams, "grant_id required", nil)
		}

		account, err := client.GetAgentAccount(ctx, p.GrantID)
		if err != nil {
			return nil, fmt.Errorf("agentAccount.get: %w", err)
		}
		return account, nil
	})
}

func RegisterGrantHandlers(d *Dispatcher, store ports.GrantStore) {
	d.Register("grant.list", func(_ context.Context, params json.RawMessage) (any, error) {
		var p struct{}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grants, err := store.ListGrants()
		if err != nil {
			return nil, fmt.Errorf("grant.list: %w", err)
		}
		return grantListResult{Grants: grants}, nil
	})
}

func RegisterConfigHandlers(d *Dispatcher, loader configLoader) {
	d.Register("config.read", func(_ context.Context, params json.RawMessage) (any, error) {
		var p struct{}
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		cfg, err := loader.Load()
		if err != nil {
			return nil, fmt.Errorf("config.read: %w", err)
		}

		result := configReadResult{
			Region:              cfg.Region,
			DefaultGrant:        cfg.DefaultGrant,
			CallbackPort:        cfg.CallbackPort,
			TUITheme:            cfg.TUITheme,
			WorkingHours:        cfg.WorkingHours,
			AIConfigured:        cfg.AI != nil,
			GPGConfigured:       cfg.GPG != nil,
			DashboardConfigured: cfg.Dashboard != nil,
		}
		if cfg.API != nil {
			result.API = &configReadAPI{
				BaseURL: cfg.API.BaseURL,
				Timeout: cfg.API.Timeout,
			}
		}

		return result, nil
	})
}
