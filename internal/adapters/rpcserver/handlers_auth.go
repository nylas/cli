package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type authGrantGetParams struct {
	GrantID string `json:"grant_id,omitempty"`
}

type authGrantCreateCustomParams struct {
	Provider string         `json:"provider"`
	Settings map[string]any `json:"settings,omitempty"`
}

type authURLParams struct {
	Provider      string `json:"provider"`
	RedirectURI   string `json:"redirect_uri"`
	State         string `json:"state,omitempty"`
	CodeChallenge string `json:"code_challenge,omitempty"`
}

type authGrantRevokeResult struct {
	Revoked bool `json:"revoked"`
}

type authURLResult struct {
	URL string `json:"url"`
}

type authGrantExchangeParams struct {
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	CodeVerifier string `json:"code_verifier,omitempty"`
}

func RegisterAuthHandlers(d *Dispatcher, client ports.AuthClient, defaultGrant string) {
	d.Register("auth.grant.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p authGrantGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		grant, err := client.GetGrant(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("auth.grant.get: %w", err)
		}
		return grant, nil
	})

	d.Register("auth.grant.revoke", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p authGrantGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.RevokeGrant(ctx, grantID); err != nil {
			return nil, fmt.Errorf("auth.grant.revoke: %w", err)
		}
		return authGrantRevokeResult{Revoked: true}, nil
	})

	d.Register("auth.grant.createCustom", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p authGrantCreateCustomParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.Provider == "" {
			return nil, NewRPCError(InvalidParams, "provider required", nil)
		}

		grant, err := client.CreateCustomGrant(ctx, p.Provider, p.Settings)
		if err != nil {
			return nil, fmt.Errorf("auth.grant.createCustom: %w", err)
		}
		return grant, nil
	})

	d.Register("auth.url", func(_ context.Context, params json.RawMessage) (any, error) {
		var p authURLParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.Provider == "" {
			return nil, NewRPCError(InvalidParams, "provider required", nil)
		}
		if p.RedirectURI == "" {
			return nil, NewRPCError(InvalidParams, "redirect_uri required", nil)
		}

		url := client.BuildAuthURL(domain.Provider(p.Provider), p.RedirectURI, p.State, p.CodeChallenge)
		return authURLResult{URL: url}, nil
	})

	d.Register("auth.grant.exchange", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p authGrantExchangeParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.Code == "" {
			return nil, NewRPCError(InvalidParams, "code required", nil)
		}
		if p.RedirectURI == "" {
			return nil, NewRPCError(InvalidParams, "redirect_uri required", nil)
		}

		grant, err := client.ExchangeCode(ctx, p.Code, p.RedirectURI, p.CodeVerifier)
		if err != nil {
			return nil, fmt.Errorf("auth.grant.exchange: %w", err)
		}
		return grant, nil
	})
}
