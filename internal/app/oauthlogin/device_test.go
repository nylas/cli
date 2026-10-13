//go:build !integration

package oauthlogin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/oauthas"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deviceTestClient struct {
	*oauthas.MockClient
	authorization *domain.OAuthDeviceAuthorization
	errors        []error
	polls         int
	scopes        []string
	resource      string
}

func (c *deviceTestClient) StartDeviceAuthorization(_ context.Context, _ string, scopes []string, resource string) (*domain.OAuthDeviceAuthorization, error) {
	c.scopes, c.resource = scopes, resource
	return c.authorization, nil
}

func (c *deviceTestClient) PollDeviceTokens(_ context.Context, _, _, _ string) (*domain.OAuthTokens, error) {
	c.polls++
	if len(c.errors) > 0 {
		err := c.errors[0]
		c.errors = c.errors[1:]
		return nil, err
	}
	return &domain.OAuthTokens{AccessToken: "device-access", RefreshToken: "device-refresh", Scope: "openid email offline_access dashboard.session", ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func deviceFixture(t *testing.T) (*fixture, *deviceTestClient, *LoginOptions, *[]time.Duration) {
	t.Helper()
	f := newFixture(t)
	c := &deviceTestClient{MockClient: f.client, authorization: &domain.OAuthDeviceAuthorization{
		DeviceCode: "device-code-in-memory", UserCode: "BCDF-GHJK",
		VerificationURI: oauthas.MockIssuer + "/oauth/device", ExpiresIn: 600, Interval: 5,
	}}
	c.MetadataFunc = func(context.Context) (*domain.OAuthServerMetadata, error) {
		return &domain.OAuthServerMetadata{Issuer: oauthas.MockIssuer, DeviceAuthorizationEndpoint: oauthas.MockIssuer + "/oauth/device_authorization"}, nil
	}
	f.service.client = c
	intervals := []time.Duration{}
	f.service.waitDevice = func(ctx context.Context, d time.Duration) error {
		intervals = append(intervals, d)
		f.advance(d)
		return ctx.Err()
	}
	opts := &LoginOptions{Device: true, OnDeviceAuthorization: func(domain.OAuthDevicePrompt) error { return nil }}
	return f, c, opts, &intervals
}

func TestDeviceLogin_CumulativeSlowDownAndPersistence(t *testing.T) {
	f, c, opts, intervals := deviceFixture(t)
	c.errors = []error{&domain.OAuthError{Code: "authorization_pending"}, &domain.OAuthError{Code: "slow_down"}, &domain.OAuthError{Code: "slow_down"}, domain.ErrNetworkError}
	opts.Resource = "https://mcp.us.nylas.com"
	var prompt domain.OAuthDevicePrompt
	opts.OnDeviceAuthorization = func(p domain.OAuthDevicePrompt) error { prompt = p; return nil }
	result, err := f.service.Login(context.Background(), *opts)
	require.NoError(t, err)
	assert.Equal(t, []time.Duration{5 * time.Second, 5 * time.Second, 10 * time.Second, 15 * time.Second, 30 * time.Second}, *intervals)
	assert.True(t, result.HasRefresh)
	assert.Equal(t, "BCDF-GHJK", prompt.UserCode)
	assert.Equal(t, "device-access", f.secrets.GetAll()[ports.KeyOAuthAccessToken])
	assert.Equal(t, opts.Resource, f.secrets.GetAll()[ports.KeyOAuthResource])
	assert.Equal(t, domain.DefaultOAuthScopes(), c.scopes)
	assert.Empty(t, f.browser.openedURL)
	assert.Empty(t, f.client.AuthorizationCalls)
}

func TestDeviceLogin_TerminalErrorsStopWithoutStoring(t *testing.T) {
	for _, code := range []string{"access_denied", "expired_token", "invalid_grant", "unauthorized_client", "server_error"} {
		t.Run(code, func(t *testing.T) {
			f, c, opts, _ := deviceFixture(t)
			c.errors = []error{&domain.OAuthError{Code: code, Description: "must-not-appear"}}
			_, err := f.service.Login(context.Background(), *opts)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "must-not-appear")
			assert.Equal(t, 1, c.polls)
			assert.Empty(t, f.secrets.GetAll())
		})
	}
}

func TestDeviceLogin_ExpiryStopsBeforeNextPoll(t *testing.T) {
	f, c, opts, _ := deviceFixture(t)
	c.authorization.ExpiresIn = 5
	_, err := f.service.Login(context.Background(), *opts)
	require.ErrorContains(t, err, "run `nylas oauth login --device` again")
	assert.Zero(t, c.polls)
}

func TestDeviceLogin_CancellationStopsPolling(t *testing.T) {
	f, c, opts, _ := deviceFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	f.service.waitDevice = func(context.Context, time.Duration) error { cancel(); return context.Canceled }
	_, err := f.service.Login(ctx, *opts)
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, c.polls)
}

func TestDeviceLogin_PromptFailureDoesNotPoll(t *testing.T) {
	f, c, opts, _ := deviceFixture(t)
	opts.OnDeviceAuthorization = func(domain.OAuthDevicePrompt) error { return errors.New("output unavailable") }
	_, err := f.service.Login(context.Background(), *opts)
	require.ErrorContains(t, err, "output unavailable")
	assert.Zero(t, c.polls)
}

func TestDeviceLogin_RefusesUnadvertisedAndNonFirstPartyClients(t *testing.T) {
	f, c, opts, _ := deviceFixture(t)
	f.service.clientID = "different-client"
	_, err := f.service.Login(context.Background(), *opts)
	require.ErrorContains(t, err, "built-in CLI")
	f.service.clientID = domain.DefaultOAuthClientID
	c.MetadataFunc = func(context.Context) (*domain.OAuthServerMetadata, error) { return &domain.OAuthServerMetadata{}, nil }
	_, err = f.service.Login(context.Background(), *opts)
	require.ErrorContains(t, err, "does not offer")
	assert.Zero(t, c.polls)
}

func TestWaitDevicePoll_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, waitDevicePoll(ctx, time.Hour), context.Canceled)
}

func TestDeviceLogin_RegionConstrainsSignupAudience(t *testing.T) {
	f, c, opts, _ := deviceFixture(t)
	opts.Region = "eu"
	result, err := f.service.Login(context.Background(), *opts)
	require.NoError(t, err)
	assert.Equal(t, domain.MCPResourceEU, c.resource)
	assert.Equal(t, domain.MCPResourceEU, result.Resource)
	assert.Equal(t, domain.MCPResourceEU, f.secrets.GetAll()[ports.KeyOAuthResource])
}

func TestDeviceLogin_RetriesTransient503WithoutRestart(t *testing.T) {
	f, c, opts, intervals := deviceFixture(t)
	c.errors = []error{&domain.OAuthError{Code: "temporarily_unavailable", StatusCode: 503}}
	_, err := f.service.Login(context.Background(), *opts)
	require.NoError(t, err)
	assert.Equal(t, []time.Duration{5 * time.Second, 10 * time.Second}, *intervals)
	assert.Equal(t, 2, c.polls)
}
