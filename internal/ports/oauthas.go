package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// OAuthAuthServerClient talks to the Nylas OAuth 2.1 authorization server
// hosted by dashboard-account.
//
// Distinct from AuthClient, which drives Nylas hosted auth to create a
// provider grant. This one authenticates the person operating the CLI and
// yields an OIDC identity plus an access token.
type OAuthAuthServerClient interface {
	// Metadata fetches the RFC 8414 authorization server metadata, caching it
	// for the lifetime of the client. Every other method resolves its
	// endpoint from this document rather than assuming a path.
	Metadata(ctx context.Context) (*domain.OAuthServerMetadata, error)

	// AuthorizationURL builds the URL to open in the browser to start a
	// PKCE authorization code flow.
	AuthorizationURL(ctx context.Context, params domain.OAuthAuthorizationParams) (string, error)

	// ExchangeCode redeems an authorization code for a token set. The
	// returned tokens carry an absolute ExpiresAt derived from expires_in,
	// which is what callers check before reusing an access token.
	ExchangeCode(ctx context.Context, params domain.OAuthCodeExchange) (*domain.OAuthTokens, error)

	// Refresh exchanges a refresh token for a new token set. The server
	// rotates refresh tokens and detects replay, so the returned
	// RefreshToken must replace the one passed in. resource is the RFC 8707
	// resource indicator, empty for none.
	Refresh(ctx context.Context, clientID, refreshToken, resource string) (*domain.OAuthTokens, error)

	// Revoke revokes an access or refresh token (RFC 7009).
	Revoke(ctx context.Context, clientID, token string) error

	// UserInfo returns the OIDC claims associated with an access token.
	UserInfo(ctx context.Context, accessToken string) (*domain.OAuthUserInfo, error)
}
