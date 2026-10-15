//go:build !integration

package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildProxyWith_CredentialRequiresOAuth(t *testing.T) {
	_, err := buildProxyWith(authAPIKey, "0123456789ab")
	require.ErrorContains(t, err, "--credential requires --auth oauth")
}

func TestBuildProxyWith_CredentialSelectsThatCredential(t *testing.T) {
	withOAuthCredentials(t, stubCredentials{err: domain.ErrOAuthNotLoggedIn}, nil)
	original := newOAuthCredentialFn
	var asked string
	newOAuthCredentialFn = func(id string) (ports.MCPCredentialSource, error) {
		asked = id
		return stubCredentials{cred: &domain.MCPCredential{Token: "t", Endpoint: domain.MCPResourceUS, GrantScoped: true}}, nil
	}
	t.Cleanup(func() { newOAuthCredentialFn = original })

	proxy, err := buildProxyWith(authOAuth, "0123456789ab")
	require.NoError(t, err)
	assert.NotNil(t, proxy)
	assert.Equal(t, "0123456789ab", asked)
}

func TestServeArgsWith(t *testing.T) {
	args, err := serveArgsWith(authOAuth, "0123456789ab")
	require.NoError(t, err)
	assert.Equal(t, []string{"mcp", "serve", "--auth", "oauth", "--credential", "0123456789ab"}, args)

	_, err = serveArgsWith(authAPIKey, "0123456789ab")
	require.Error(t, err)

	args, err = serveArgsWith(authOAuth, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"mcp", "serve", "--auth", "oauth"}, args)
}

func TestInstallServer_WithoutAuthKeepsThePinnedCredential(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	configPath := filepath.Join(t.TempDir(), "config.json")
	a := testAssistant("cursor", configPath)
	args, err := serveArgsWith(authOAuth, "0123456789ab")
	require.NoError(t, err)
	require.NoError(t, installServer(a, "/usr/local/bin/nylas", args))

	require.NoError(t, installServer(a, "/opt/bin/nylas", nil))

	raw, err := os.ReadFile(configPath) // #nosec G304 -- test temp file
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"--credential"`)
	assert.Contains(t, string(raw), `"0123456789ab"`)
}
