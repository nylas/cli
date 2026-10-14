package oauthas

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

// StartDeviceAuthorization requests a first-party device authorization.
func (c *Client) StartDeviceAuthorization(ctx context.Context, clientID string, scopes []string, resource string) (*domain.OAuthDeviceAuthorization, error) {
	metadata, err := c.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	if metadata.DeviceAuthorizationEndpoint == "" {
		return nil, domain.ErrOAuthMetadata
	}
	form := url.Values{"client_id": {clientID}, "scope": {strings.Join(scopes, " ")}}
	if resource != "" {
		form.Set("resource", resource)
	}
	var result domain.OAuthDeviceAuthorization
	if err := c.postForm(ctx, metadata.DeviceAuthorizationEndpoint, form, &result); err != nil {
		return nil, safeDeviceError(err)
	}
	if err := result.ValidateFor(metadata.Issuer); err != nil {
		return nil, err
	}
	return &result, nil
}

// PollDeviceTokens performs one poll; the application service owns the cadence.
func (c *Client) PollDeviceTokens(ctx context.Context, clientID, deviceCode, resource string) (*domain.OAuthTokens, error) {
	metadata, err := c.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{"grant_type": {domain.OAuthDeviceGrant}, "client_id": {clientID}, "device_code": {deviceCode}}
	if resource != "" {
		form.Set("resource", resource)
	}
	tokens, err := c.requestTokens(ctx, metadata.TokenEndpoint, form)
	if err != nil {
		return nil, safeDeviceError(err)
	}
	return tokens, nil
}

// Device errors never echo a response body that could contain bearer codes.
func safeDeviceError(err error) error {
	if errors.Is(err, domain.ErrNetworkError) {
		return domain.ErrNetworkError
	}
	var oauthErr *domain.OAuthError
	if errors.As(err, &oauthErr) {
		code := "server_error"
		switch oauthErr.Code {
		case "authorization_pending", "slow_down", "access_denied", "expired_token", "invalid_grant", "invalid_client", "unauthorized_client", "unsupported_grant_type", "invalid_scope", "invalid_target", "invalid_request", "server_error", "temporarily_unavailable":
			code = oauthErr.Code
		}
		return &domain.OAuthError{Code: code, StatusCode: oauthErr.StatusCode}
	}
	return errors.New("device sign-in failed")
}
