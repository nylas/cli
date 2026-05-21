package agent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type workspaceUpdateTestClient struct {
	workspaceID string
	req         *domain.UpdateWorkspaceRequest
}

func (c *workspaceUpdateTestClient) UpdateWorkspace(ctx context.Context, workspaceID string, req *domain.UpdateWorkspaceRequest) (*domain.Workspace, error) {
	c.workspaceID = workspaceID
	c.req = req
	return &domain.Workspace{ID: workspaceID}, nil
}

func TestAttachPolicyToAgentWorkspacePatchesWorkspaceWithoutMutatingGrantSettings(t *testing.T) {
	client := &workspaceUpdateTestClient{}
	account := &domain.AgentAccount{
		ID:          "grant-1",
		WorkspaceID: "workspace-1",
		Settings: domain.AgentAccountSettings{
			PolicyID: "legacy-policy",
		},
	}

	updated, err := attachPolicyToAgentWorkspace(context.Background(), client, account, "workspace-policy")

	require.NoError(t, err)
	require.Same(t, account, updated)
	assert.Equal(t, "workspace-1", client.workspaceID)
	require.NotNil(t, client.req)
	require.NotNil(t, client.req.PolicyID)
	assert.Equal(t, "workspace-policy", *client.req.PolicyID)
	assert.Equal(t, "legacy-policy", updated.Settings.PolicyID)
}

func TestGetAgentIdentifier(t *testing.T) {
	t.Run("uses explicit argument", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)
		t.Setenv("NYLAS_AGENT_GRANT_ID", "env-agent-grant")

		identifier, err := getAgentIdentifier([]string{"  agent-123  "})

		require.NoError(t, err)
		assert.Equal(t, "agent-123", identifier)
	})

	t.Run("uses NYLAS_AGENT_GRANT_ID before stored default", func(t *testing.T) {
		configDir := setupAgentIdentifierTestEnv(t)
		seedAgentIdentifierDefaultGrant(t, configDir, domain.GrantInfo{
			ID:       "stored-default",
			Email:    "stored@example.com",
			Provider: domain.ProviderNylas,
		})
		t.Setenv("NYLAS_AGENT_GRANT_ID", "env-agent-grant")

		identifier, err := getAgentIdentifier(nil)

		require.NoError(t, err)
		assert.Equal(t, "env-agent-grant", identifier)
	})

	t.Run("falls back to configured default grant", func(t *testing.T) {
		configDir := setupAgentIdentifierTestEnv(t)
		seedAgentIdentifierDefaultGrant(t, configDir, domain.GrantInfo{
			ID:       "stored-default",
			Email:    "stored@example.com",
			Provider: domain.ProviderNylas,
		})

		identifier, err := getAgentIdentifier(nil)

		require.NoError(t, err)
		assert.Equal(t, "stored-default", identifier)
	})

	t.Run("falls back to the unique stored agent grant when the global default is not nylas", func(t *testing.T) {
		configDir := setupAgentIdentifierTestEnv(t)
		seedAgentIdentifierStoredGrant(t, configDir, domain.GrantInfo{
			ID:       "google-default",
			Email:    "user@gmail.com",
			Provider: domain.ProviderGoogle,
		})
		seedAgentIdentifierStoredGrant(t, configDir, domain.GrantInfo{
			ID:       "agent-default",
			Email:    "agent@example.com",
			Provider: domain.ProviderNylas,
		})
		seedAgentIdentifierDefaultGrantOnly(t, configDir, "google-default")

		identifier, err := getAgentIdentifier(nil)

		require.NoError(t, err)
		assert.Equal(t, "agent-default", identifier)
	})

	t.Run("rejects ambiguous managed agent fallback", func(t *testing.T) {
		configDir := setupAgentIdentifierTestEnv(t)
		seedAgentIdentifierStoredGrant(t, configDir, domain.GrantInfo{
			ID:       "google-default",
			Email:    "user@gmail.com",
			Provider: domain.ProviderGoogle,
		})
		seedAgentIdentifierStoredGrant(t, configDir, domain.GrantInfo{
			ID:       "agent-a",
			Email:    "agent-a@example.com",
			Provider: domain.ProviderNylas,
		})
		seedAgentIdentifierStoredGrant(t, configDir, domain.GrantInfo{
			ID:       "agent-b",
			Email:    "agent-b@example.com",
			Provider: domain.ProviderNylas,
		})
		seedAgentIdentifierDefaultGrantOnly(t, configDir, "google-default")

		identifier, err := getAgentIdentifier(nil)

		require.Error(t, err)
		assert.Empty(t, identifier)
		assert.Contains(t, err.Error(), "multiple provider=nylas agent grants available")
	})

	t.Run("returns standard grant resolution error when unset", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)

		identifier, err := getAgentIdentifier(nil)

		require.Error(t, err)
		assert.Empty(t, identifier)
		assert.Contains(t, err.Error(), "no provider=nylas agent grant configured")
	})

	t.Run("rejects explicit blank identifier", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)
		t.Setenv("NYLAS_AGENT_GRANT_ID", "env-agent-grant")

		identifier, err := getAgentIdentifier([]string{"   "})

		require.Error(t, err)
		assert.Empty(t, identifier)
		assert.Contains(t, err.Error(), "agent ID required")
	})
}

func TestGetRequiredAgentIdentifier(t *testing.T) {
	t.Run("uses explicit argument", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)

		identifier, err := getRequiredAgentIdentifier([]string{"agent-123"})

		require.NoError(t, err)
		assert.Equal(t, "agent-123", identifier)
	})

	t.Run("uses NYLAS_AGENT_GRANT_ID when argument omitted", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)
		t.Setenv("NYLAS_AGENT_GRANT_ID", "env-agent-grant")

		identifier, err := getRequiredAgentIdentifier(nil)

		require.NoError(t, err)
		assert.Equal(t, "env-agent-grant", identifier)
	})

	t.Run("does not fall back to stored default grant", func(t *testing.T) {
		configDir := setupAgentIdentifierTestEnv(t)
		seedAgentIdentifierDefaultGrant(t, configDir, domain.GrantInfo{
			ID:       "stored-default",
			Email:    "stored@example.com",
			Provider: domain.ProviderNylas,
		})

		identifier, err := getRequiredAgentIdentifier(nil)

		require.Error(t, err)
		assert.Empty(t, identifier)
		assert.Contains(t, err.Error(), "agent ID required")
	})

	t.Run("rejects explicit blank identifier", func(t *testing.T) {
		setupAgentIdentifierTestEnv(t)
		t.Setenv("NYLAS_AGENT_GRANT_ID", "env-agent-grant")

		identifier, err := getRequiredAgentIdentifier([]string{"   "})

		require.Error(t, err)
		assert.Empty(t, identifier)
		assert.Contains(t, err.Error(), "agent ID required")
	})
}

func setupAgentIdentifierTestEnv(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configDir := filepath.Join(configHome, "nylas")

	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase")
	t.Setenv("NYLAS_GRANT_ID", "")
	t.Setenv("NYLAS_AGENT_GRANT_ID", "")

	return configDir
}

func seedAgentIdentifierDefaultGrant(t *testing.T, configDir string, grant domain.GrantInfo) {
	t.Helper()
	_ = configDir

	grantStore, err := common.NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SaveGrant(grant))
	require.NoError(t, grantStore.SetDefaultGrant(grant.ID))
}

func seedAgentIdentifierStoredGrant(t *testing.T, configDir string, grant domain.GrantInfo) {
	t.Helper()
	_ = configDir

	grantStore, err := common.NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SaveGrant(grant))
}

func seedAgentIdentifierDefaultGrantOnly(t *testing.T, configDir, grantID string) {
	t.Helper()
	_ = configDir

	grantStore, err := common.NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SetDefaultGrant(grantID))
}
