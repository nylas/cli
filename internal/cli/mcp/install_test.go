package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeArgsFor(t *testing.T) {
	args, err := serveArgsFor(authAPIKey)
	require.NoError(t, err)
	assert.Equal(t, []string{"mcp", "serve"}, args, "the default is what earlier releases wrote")

	args, err = serveArgsFor(authOAuth)
	require.NoError(t, err)
	assert.Equal(t, []string{"mcp", "serve", "--auth", "oauth"}, args)

	_, err = serveArgsFor("token")
	require.Error(t, err)
}

func TestInstallServer_OAuthWritesNoCredential(t *testing.T) {
	// The config file is readable by anything running as the user; the
	// OAuth session stays in the keyring and the config only says how to
	// start the proxy.
	//
	// claude-code also writes ~/.claude/settings.json, so HOME must point at
	// a temp dir or the test edits the developer's real settings.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	for _, id := range []string{"claude-desktop", "claude-code", "cursor", "windsurf", "vscode"} {
		t.Run(id, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			args, err := serveArgsFor(authOAuth)
			require.NoError(t, err)

			require.NoError(t, installServer(testAssistant(id, configPath), "/usr/local/bin/nylas", args))

			raw, err := os.ReadFile(configPath) // #nosec G304 -- test temp file
			require.NoError(t, err)
			content := string(raw)
			assert.Contains(t, content, `"--auth"`)
			assert.Contains(t, content, `"oauth"`)
			for _, forbidden := range []string{"api_key", "apiKey", "Authorization", "Bearer", "NYLAS_API_KEY", "token"} {
				assert.False(t, strings.Contains(content, forbidden), "config must not carry %q", forbidden)
			}
		})
	}
}

func TestInstallCmd_AuthFlagDefaultsToAPIKey(t *testing.T) {
	flag := newInstallCmd().Flags().Lookup("auth")
	require.NotNil(t, flag)
	assert.Equal(t, authAPIKey, flag.DefValue)
}
