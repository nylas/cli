package mcp

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCredentials struct {
	cred *domain.MCPCredential
	err  error
}

func (s stubCredentials) Credential(context.Context) (*domain.MCPCredential, error) {
	return s.cred, s.err
}

func (s stubCredentials) Renew(context.Context, *domain.MCPCredential) (*domain.MCPCredential, error) {
	return nil, domain.ErrMCPCredentialNotRenewable
}

func withOAuthCredentials(t *testing.T, source ports.MCPCredentialSource, err error) {
	t.Helper()
	original := newOAuthCredentialsFn
	newOAuthCredentialsFn = func() (ports.MCPCredentialSource, error) { return source, err }
	t.Cleanup(func() { newOAuthCredentialsFn = original })
}

func TestServeCmd_AuthDefaultsToAPIKey(t *testing.T) {
	flag := newServeCmd().Flags().Lookup("auth")
	require.NotNil(t, flag)
	assert.Equal(t, authAPIKey, flag.DefValue, "existing installs must keep working unchanged")
}

func TestBuildProxy_RejectsUnknownAuthMode(t *testing.T) {
	_, err := buildProxy("password")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestBuildProxy_OAuthWithoutSessionNamesTheLoginCommand(t *testing.T) {
	withOAuthCredentials(t, stubCredentials{err: domain.ErrOAuthNotLoggedIn}, nil)

	_, err := buildProxy(authOAuth)

	require.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)
	assert.Contains(t, err.Error(), "nylas oauth login --for mcp")
}

func TestBuildProxy_OAuthWithSession(t *testing.T) {
	withOAuthCredentials(t, stubCredentials{cred: &domain.MCPCredential{
		Token: "t", Endpoint: domain.MCPResourceUS, GrantScoped: true,
	}}, nil)

	proxy, err := buildProxy(authOAuth)

	require.NoError(t, err)
	assert.NotNil(t, proxy)
}
