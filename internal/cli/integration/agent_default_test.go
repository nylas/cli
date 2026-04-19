//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
)

func TestCLI_AgentUpdate_UsesDefaultGrant(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	email := newAgentTestEmail(t, "default-update")
	appPassword := validAgentTestPassword()
	client := getTestClient()

	var created *domain.AgentAccount
	t.Cleanup(func() {
		if created == nil {
			return
		}
		if exists, account := waitForAgentByEmail(t, client, email, true); exists {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, account.ID)
		}
	})

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	account, err := client.CreateAgentAccount(ctx, email, "", "")
	cancel()
	if err != nil {
		t.Fatalf("failed to create agent account %q for default update test: %v", email, err)
	}
	created = account

	setAgentSandboxDefaultGrant(t, env, account)

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"account",
		"update",
		"--app-password", appPassword,
		"--json",
	)
	if err != nil {
		t.Fatalf("agent update with default grant failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var updated domain.AgentAccount
	if err := json.Unmarshal([]byte(stdout), &updated); err != nil {
		t.Fatalf("failed to parse agent update JSON: %v\noutput: %s", err, stdout)
	}
	if updated.ID != created.ID {
		t.Fatalf("updated agent ID = %q, want %q", updated.ID, created.ID)
	}
}

func setAgentSandboxDefaultGrant(t *testing.T, env map[string]string, account *domain.AgentAccount) {
	t.Helper()

	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", env["NYLAS_FILE_STORE_PASSPHRASE"])

	configDir := filepath.Join(env["XDG_CONFIG_HOME"], "nylas")
	secretStore, err := keyring.NewEncryptedFileStore(configDir)
	if err != nil {
		t.Fatalf("failed to create sandbox secret store: %v", err)
	}

	grantStore := keyring.NewGrantStore(secretStore)
	if err := grantStore.SaveGrant(domain.GrantInfo{
		ID:       account.ID,
		Email:    account.Email,
		Provider: domain.ProviderNylas,
	}); err != nil {
		t.Fatalf("failed to save sandbox default grant: %v", err)
	}
	if err := grantStore.SetDefaultGrant(account.ID); err != nil {
		t.Fatalf("failed to set sandbox default grant: %v", err)
	}
}
