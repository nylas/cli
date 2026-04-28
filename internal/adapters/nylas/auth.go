// Package nylas provides the Nylas API client implementation.
package nylas

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// BuildAuthURL builds the OAuth authorization URL.
func (c *HTTPClient) BuildAuthURL(provider domain.Provider, redirectURI, state, codeChallenge string) string {
	baseURL := fmt.Sprintf("%s/v3/connect/auth", c.baseURL)
	query := NewQueryBuilder().
		Add("client_id", c.clientID).
		Add("redirect_uri", redirectURI).
		Add("response_type", "code").
		Add("provider", string(provider)).
		Add("access_type", "offline").
		Add("state", state)

	if codeChallenge != "" {
		query.Add("code_challenge", codeChallenge).
			Add("code_challenge_method", "S256")
	}

	return query.BuildURL(baseURL)
}

// ExchangeCode exchanges an authorization code for tokens.
func (c *HTTPClient) ExchangeCode(ctx context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error) {
	// In Nylas v3, client_secret is the API key
	secret := c.clientSecret
	if secret == "" {
		secret = c.apiKey
	}

	payload := map[string]string{
		"code":          code,
		"redirect_uri":  redirectURI,
		"grant_type":    "authorization_code",
		"client_id":     c.clientID,
		"client_secret": secret,
		"code_verifier": codeVerifier,
	}

	resp, err := c.doJSONRequestNoAuth(ctx, "POST", c.baseURL+"/v3/connect/token", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		GrantID      string `json:"grant_id"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Email        string `json:"email"`
		Provider     string `json:"provider"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &domain.Grant{
		ID:           result.GrantID,
		Email:        result.Email,
		Provider:     domain.Provider(result.Provider),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		GrantStatus:  "valid",
	}, nil
}

// ListGrants lists all grants for the application, transparently
// following next_cursor pagination so callers always see the complete
// result set. The Nylas v3 default page size (10) would otherwise
// silently truncate accounts with more than ten grants.
func (c *HTTPClient) ListGrants(ctx context.Context) ([]domain.Grant, error) {
	return c.ListAllGrants(ctx, nil)
}

// GetGrant retrieves a specific grant.
func (c *HTTPClient) GetGrant(ctx context.Context, grantID string) (*domain.Grant, error) {
	queryURL := c.baseURL + "/v3/grants/" + url.PathEscape(grantID)

	var result struct {
		Data domain.Grant `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrGrantNotFound); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// RevokeGrant revokes a grant.
func (c *HTTPClient) RevokeGrant(ctx context.Context, grantID string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+"/v3/grants/"+url.PathEscape(grantID), nil)
	if err != nil {
		return err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return domain.ErrGrantNotFound
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}
