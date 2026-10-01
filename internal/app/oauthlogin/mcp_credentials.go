package oauthlogin

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

// MCPCredentials serves the stored OAuth session to the MCP proxy
// (ports.MCPCredentialSource).
//
// The MCP host comes from the token's audience, not from the configured
// region: the token is only good at the server it was issued for, and the
// audience is where the authorization server wrote that down. The claims are
// decoded, not verified — they route the request, and the MCP server checks
// the signature.
type MCPCredentials struct {
	service *Service
}

// NewMCPCredentials returns a credential source backed by s.
func NewMCPCredentials(s *Service) *MCPCredentials {
	return &MCPCredentials{service: s}
}

// Credential returns the current access token, refreshed when it is at or
// near expiry, and the MCP server it was issued for.
func (m *MCPCredentials) Credential(ctx context.Context) (*domain.MCPCredential, error) {
	token, err := m.service.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	return credentialFor(token)
}

// Renew refreshes after the server refused rejected with 401.
func (m *MCPCredentials) Renew(ctx context.Context, rejected *domain.MCPCredential) (*domain.MCPCredential, error) {
	previous := ""
	if rejected != nil {
		previous = rejected.Token
	}
	token, err := m.service.RefreshAccessToken(ctx, previous)
	if err != nil {
		return nil, err
	}
	return credentialFor(token)
}

func credentialFor(token string) (*domain.MCPCredential, error) {
	claims, err := domain.DecodeOAuthAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("cannot route the stored OAuth token to an MCP server: %w", err)
	}
	endpoint, err := domain.MCPEndpointFromAudience(claims.Audience)
	if err != nil {
		return nil, err
	}
	return &domain.MCPCredential{
		Token:       token,
		Endpoint:    endpoint,
		GrantScoped: true,
		Grants:      claims.Grants,
	}, nil
}
