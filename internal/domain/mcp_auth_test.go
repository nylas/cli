//go:build !integration

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPResourceForRegion(t *testing.T) {
	for region, want := range map[string]string{"": MCPResourceUS, "us": MCPResourceUS, "US": MCPResourceUS, "eu": MCPResourceEU} {
		got, err := MCPResourceForRegion(region)
		require.NoError(t, err, region)
		assert.Equal(t, want, got, region)
	}

	// An unknown region must not silently become US: the token would be
	// issued for a server holding none of the user's data.
	_, err := MCPResourceForRegion("ap")
	require.ErrorIs(t, err, ErrMCPResource)
}

func TestValidateMCPResource(t *testing.T) {
	require.NoError(t, ValidateMCPResource("https://mcp.us.nylas.com"))
	require.NoError(t, ValidateMCPResource("https://mcp.eu.nylas.com"))
	for _, bad := range []string{"", "https://mcp.us.nylas.com/", "http://mcp.us.nylas.com", "https://evil.example.com", "https://mcp.us.nylas.com.evil.example"} {
		assert.ErrorIs(t, ValidateMCPResource(bad), ErrMCPResource, bad)
	}
}

func TestMCPResource_Override(t *testing.T) {
	t.Setenv(EnvMCPURLUS, "https://mcp-staging.us.example.test/")
	got, err := MCPResourceForRegion("us")
	require.NoError(t, err)
	assert.Equal(t, "https://mcp-staging.us.example.test", got, "one trailing slash is dropped")
	got, err = MCPResourceForRegion("eu")
	require.NoError(t, err)
	assert.Equal(t, MCPResourceEU, got, "the other region keeps production")

	// The override REPLACES the region's server: production US is no longer
	// trusted, so a token can only go to the one server per region.
	require.NoError(t, ValidateMCPResource("https://mcp-staging.us.example.test"))
	require.NoError(t, ValidateMCPResource(MCPResourceEU))
	require.ErrorIs(t, ValidateMCPResource(MCPResourceUS), ErrMCPResource)
	endpoint, err := MCPEndpointFromAudience([]string{"https://mcp-staging.us.example.test"})
	require.NoError(t, err)
	assert.Equal(t, "https://mcp-staging.us.example.test", endpoint)
}

func TestMCPResource_OverrideLoopbackHTTP(t *testing.T) {
	for _, origin := range []string{"http://localhost:8080", "http://127.0.0.1:8080", "http://[::1]:8080"} {
		t.Setenv(EnvMCPURLEU, origin)
		got, err := MCPResourceForRegion("eu")
		require.NoError(t, err, origin)
		assert.Equal(t, origin, got)
	}
}

// A bad override fails closed instead of quietly using production, and never
// becomes trusted itself.
func TestMCPResource_OverrideRejected(t *testing.T) {
	for _, bad := range []string{
		"http://mcp.example.test",
		"ftp://mcp.example.test",
		"https://user@mcp.example.test",
		"https://mcp.example.test/path",
		"https://mcp.example.test?x=1",
		"https://mcp.example.test#f",
		"mcp.example.test",
		"https://",
	} {
		t.Setenv(EnvMCPURLUS, bad)
		_, err := MCPResourceForRegion("us")
		require.ErrorIs(t, err, ErrMCPResource, bad)
		assert.ErrorIs(t, ValidateMCPResource(bad), ErrMCPResource, bad)
		assert.ErrorIs(t, ValidateMCPResource(MCPResourceUS), ErrMCPResource, bad)
	}
}

func TestMCPEndpointFromAudience(t *testing.T) {
	got, err := MCPEndpointFromAudience([]string{"https://mcp.eu.nylas.com"})
	require.NoError(t, err)
	assert.Equal(t, MCPResourceEU, got)

	got, err = MCPEndpointFromAudience([]string{"account", "https://mcp.us.nylas.com", "https://mcp.us.nylas.com"})
	require.NoError(t, err, "unrelated audiences and duplicates do not make it ambiguous")
	assert.Equal(t, MCPResourceUS, got)

	for name, aud := range map[string][]string{
		"none":      nil,
		"unknown":   {"https://attacker.example"},
		"ambiguous": {MCPResourceUS, MCPResourceEU},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := MCPEndpointFromAudience(aud)
			require.ErrorIs(t, err, ErrMCPResource)
		})
	}
}

func TestMCPOAuthScopes_AreNarrow(t *testing.T) {
	scopes := MCPOAuthScopes()
	assert.Contains(t, scopes, OAuthScopeOfflineAccess, "serve is long-running and must refresh")
	assert.Contains(t, scopes, OAuthScopeGrantsRead)
	assert.NotContains(t, scopes, OAuthScopeGrantsWrite, "the MCP tools never create or delete a grant")
}

func TestMCPCredential_AllowsGrantHint(t *testing.T) {
	apiKey := &MCPCredential{Token: "k"}
	assert.True(t, apiKey.AllowsGrantHint("any-grant"), "an API key is not grant scoped")
	assert.False(t, apiKey.AllowsGrantHint(""))

	oauth := &MCPCredential{Token: "t", GrantScoped: true, Grants: []OAuthTokenGrant{{ID: "g1", ApplicationID: "a1"}}}
	assert.True(t, oauth.AllowsGrantHint("g1"))
	assert.False(t, oauth.AllowsGrantHint("g2"))

	none := &MCPCredential{Token: "t", GrantScoped: true}
	assert.False(t, none.AllowsGrantHint("g1"), "a token listing no grants allows no hint")
}

func TestParseBearerChallenge(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   *BearerChallenge
	}{
		{"absent", "", nil},
		{"other scheme", `Basic realm="x"`, nil},
		{"bare", "Bearer", &BearerChallenge{}},
		{
			"insufficient scope",
			`Bearer error="insufficient_scope", scope="email.send calendar.write", error_description="needs more"`,
			&BearerChallenge{Error: "insufficient_scope", Scope: "email.send calendar.write", ErrorDescription: "needs more"},
		},
		{
			"invalid token with resource metadata",
			`Bearer resource_metadata="https://mcp.us.nylas.com/.well-known/oauth-protected-resource", error="invalid_token"`,
			&BearerChallenge{Error: "invalid_token", ResourceMetadata: "https://mcp.us.nylas.com/.well-known/oauth-protected-resource"},
		},
		{"case-insensitive scheme, unquoted", `bearer error=invalid_token`, &BearerChallenge{Error: "invalid_token"}},
		{"comma and escape in quotes", `Bearer error_description="a, \"b\"", error="x"`, &BearerChallenge{Error: "x", ErrorDescription: `a, "b"`}},
		{"after another challenge", `Basic realm="r", Bearer error="invalid_token"`, &BearerChallenge{Error: "invalid_token"}},
		{"scheme only as a substring", `NotBearer error="x"`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseBearerChallenge(tt.header))
		})
	}
	assert.Equal(t, []string{"email.send", "calendar.write"},
		ParseBearerChallenge(`Bearer scope="email.send calendar.write"`).Scopes())
}
