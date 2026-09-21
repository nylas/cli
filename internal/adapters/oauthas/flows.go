package oauthas

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// Register performs RFC 7591 dynamic client registration. The server defaults
// an omitted token_endpoint_auth_method to "none", but the CLI states it so
// the registration cannot silently become confidential.
func (c *Client) Register(ctx context.Context, req domain.OAuthClientRegistrationRequest) (*domain.OAuthClientRegistration, error) {
	metadata, err := c.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	if metadata.RegistrationEndpoint == "" {
		return nil, fmt.Errorf("%w: server does not advertise a registration endpoint", domain.ErrOAuthMetadata)
	}

	var registration domain.OAuthClientRegistration
	if err := c.postJSON(ctx, metadata.RegistrationEndpoint, req, &registration); err != nil {
		return nil, fmt.Errorf("client registration failed: %w", err)
	}
	if registration.ClientID == "" {
		return nil, fmt.Errorf("client registration failed: server returned no client_id")
	}
	return &registration, nil
}

// AuthorizationURL builds the authorization request URL.
func (c *Client) AuthorizationURL(ctx context.Context, params domain.OAuthAuthorizationParams) (string, error) {
	metadata, err := c.Metadata(ctx)
	if err != nil {
		return "", err
	}

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", params.ClientID)
	query.Set("redirect_uri", params.RedirectURI)
	query.Set("scope", strings.Join(params.Scopes, " "))
	query.Set("state", params.State)
	query.Set("code_challenge", params.CodeChallenge)
	query.Set("code_challenge_method", "S256")
	if params.Nonce != "" {
		query.Set("nonce", params.Nonce)
	}

	separator := "?"
	if strings.Contains(metadata.AuthorizationEndpoint, "?") {
		separator = "&"
	}
	return metadata.AuthorizationEndpoint + separator + query.Encode(), nil
}

// ExchangeCode redeems an authorization code for tokens.
func (c *Client) ExchangeCode(ctx context.Context, params domain.OAuthCodeExchange) (*domain.OAuthTokens, error) {
	metadata, err := c.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", params.Code)
	form.Set("redirect_uri", params.RedirectURI)
	form.Set("code_verifier", params.CodeVerifier)
	form.Set("client_id", params.ClientID)
	if params.ClientSecret != "" {
		form.Set("client_secret", params.ClientSecret)
	}

	return c.requestTokens(ctx, metadata.TokenEndpoint, form)
}

// Refresh exchanges a refresh token for a new token set.
func (c *Client) Refresh(ctx context.Context, clientID, refreshToken string) (*domain.OAuthTokens, error) {
	metadata, err := c.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)

	return c.requestTokens(ctx, metadata.TokenEndpoint, form)
}

func (c *Client) requestTokens(ctx context.Context, endpoint string, form url.Values) (*domain.OAuthTokens, error) {
	var tokens domain.OAuthTokens
	if err := c.postForm(ctx, endpoint, form, &tokens); err != nil {
		return nil, err
	}
	if tokens.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned no access_token")
	}
	if tokens.ExpiresIn > 0 {
		tokens.ExpiresAt = c.now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	}
	return &tokens, nil
}

// Revoke revokes an access or refresh token.
func (c *Client) Revoke(ctx context.Context, clientID, token string) error {
	metadata, err := c.Metadata(ctx)
	if err != nil {
		return err
	}
	if metadata.RevocationEndpoint == "" {
		return fmt.Errorf("%w: server does not advertise a revocation endpoint", domain.ErrOAuthMetadata)
	}

	form := url.Values{}
	form.Set("token", token)
	form.Set("client_id", clientID)

	return c.postForm(ctx, metadata.RevocationEndpoint, form, nil)
}

// UserInfo returns the OIDC claims for an access token.
func (c *Client) UserInfo(ctx context.Context, accessToken string) (*domain.OAuthUserInfo, error) {
	metadata, err := c.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	if metadata.UserInfoEndpoint == "" {
		return nil, fmt.Errorf("%w: server does not advertise a userinfo endpoint", domain.ErrOAuthMetadata)
	}

	var info domain.OAuthUserInfo
	if err := c.getJSON(ctx, metadata.UserInfoEndpoint, accessToken, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
