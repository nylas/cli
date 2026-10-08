package oauthlogin

import (
	"context"
	"fmt"
	"sync"
	"time"

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
//
// The credential is kept in memory until it nears expiry. Reading the session
// is several keyring reads, and on the encrypted file store each one derives
// the key again, which cost about 185ms on every MCP request.
type MCPCredentials struct {
	service *Service

	mu        sync.Mutex
	cached    *domain.MCPCredential
	expiresAt time.Time
}

// NewMCPCredentials returns a credential source backed by s.
func NewMCPCredentials(s *Service) *MCPCredentials {
	return &MCPCredentials{service: s}
}

// Credential returns the current access token, refreshed when it is at or
// near expiry, and the MCP server it was issued for.
func (m *MCPCredentials) Credential(ctx context.Context) (*domain.MCPCredential, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cached != nil && !(&domain.OAuthTokens{ExpiresAt: m.expiresAt}).ExpiresWithin(m.service.now(), 0) {
		return m.cached, nil
	}
	token, err := m.service.AccessToken(ctx)
	if err != nil {
		m.cached = nil
		return nil, err
	}
	return m.remember(token)
}

// Renew refreshes after the server refused rejected with 401.
func (m *MCPCredentials) Renew(ctx context.Context, rejected *domain.MCPCredential) (*domain.MCPCredential, error) {
	previous := ""
	if rejected != nil {
		previous = rejected.Token
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cached = nil
	token, err := m.service.RefreshAccessToken(ctx, previous)
	if err != nil {
		return nil, err
	}
	return m.remember(token)
}

// remember caches the credential for token until the token's own expiry. A
// token without one is not cached, so every request reads the session.
func (m *MCPCredentials) remember(token string) (*domain.MCPCredential, error) {
	cred, err := credentialFor(token)
	if err != nil {
		return nil, err
	}
	m.cached, m.expiresAt = nil, time.Time{}
	if claims, err := domain.DecodeOAuthAccessToken(token); err == nil && !claims.ExpiresAt.IsZero() {
		m.cached, m.expiresAt = cred, claims.ExpiresAt
	}
	return cred, nil
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
