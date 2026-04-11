package scheduler

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isolateSchedulerCommandEnv(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_API_KEY", "")
	t.Setenv("NYLAS_CLIENT_ID", "")
	t.Setenv("NYLAS_CLIENT_SECRET", "")
	t.Setenv("NYLAS_GRANT_ID", "")

	common.ResetCachedClient()
	t.Cleanup(common.ResetCachedClient)
}

func executeSchedulerCommand(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs(args)

	return cmd.Execute()
}

func TestConfigCreateCmd_ValidatesMissingNameBeforeAuth(t *testing.T) {
	isolateSchedulerCommandEnv(t)

	err := executeSchedulerCommand(t, newConfigCreateCmd(),
		"--participants", "alice@example.com",
		"--title", "Team Sync",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name flag is required")
	assert.NotContains(t, err.Error(), "API key not configured")
}

func TestConfigCreateCmd_ValidatesMissingTitleBeforeAuth(t *testing.T) {
	isolateSchedulerCommandEnv(t)

	err := executeSchedulerCommand(t, newConfigCreateCmd(),
		"--name", "Quick Chat",
		"--participants", "alice@example.com",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--title flag is required")
	assert.NotContains(t, err.Error(), "API key not configured")
}

func TestConfigCreateCmd_ValidatesFileBeforeAuth(t *testing.T) {
	isolateSchedulerCommandEnv(t)

	missingPath := filepath.Join(t.TempDir(), "missing.json")
	err := executeSchedulerCommand(t, newConfigCreateCmd(), "--file", missingPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read "+missingPath)
	assert.NotContains(t, err.Error(), "API key not configured")
}

func TestConfigCreateCmd_ValidatesInvalidJSONBeforeAuth(t *testing.T) {
	isolateSchedulerCommandEnv(t)

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.json")
	require.NoError(t, os.WriteFile(filePath, []byte("{invalid"), 0o600))

	err := executeSchedulerCommand(t, newConfigCreateCmd(), "--file", filePath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse "+filePath)
	assert.NotContains(t, err.Error(), "API key not configured")
}

func TestConfigUpdateCmd_ValidatesFileBeforeAuth(t *testing.T) {
	isolateSchedulerCommandEnv(t)

	missingPath := filepath.Join(t.TempDir(), "missing.json")
	err := executeSchedulerCommand(t, newConfigUpdateCmd(), "config-123", "--file", missingPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read "+missingPath)
	assert.NotContains(t, err.Error(), "API key not configured")
}

func TestConfigUpdateCmd_ValidatesInvalidJSONBeforeAuth(t *testing.T) {
	isolateSchedulerCommandEnv(t)

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.json")
	require.NoError(t, os.WriteFile(filePath, []byte("{invalid"), 0o600))

	err := executeSchedulerCommand(t, newConfigUpdateCmd(), "config-123", "--file", filePath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse "+filePath)
	assert.NotContains(t, err.Error(), "API key not configured")
}

func TestConfigUpdateCmd_RequiresUpdateFieldsBeforeAuth(t *testing.T) {
	isolateSchedulerCommandEnv(t)

	err := executeSchedulerCommand(t, newConfigUpdateCmd(), "config-123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "No update fields provided")
	assert.NotContains(t, err.Error(), "API key not configured")
}
