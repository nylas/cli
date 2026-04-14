//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCLI_AgentLifecycle_CreateListDeleteByEmail(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	email := newAgentTestEmail(t, "lifecycle")
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

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "create", email, "--app-password", appPassword, "--json")
	if err != nil {
		t.Fatalf("agent create failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var account domain.AgentAccount
	if err := json.Unmarshal([]byte(stdout), &account); err != nil {
		t.Fatalf("failed to parse agent create JSON: %v\noutput: %s", err, stdout)
	}
	if account.Email != email {
		t.Fatalf("created email = %q, want %q", account.Email, email)
	}
	if account.Provider != domain.ProviderNylas {
		t.Fatalf("created provider = %q, want %q", account.Provider, domain.ProviderNylas)
	}
	if strings.TrimSpace(account.ID) == "" {
		t.Fatalf("expected created agent account ID, got empty output: %s", stdout)
	}
	created = &account

	if exists, _ := waitForAgentByEmail(t, client, email, true); !exists {
		t.Fatalf("created agent account %q did not appear in list", email)
	}

	listStdout, listStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "list", "--json")
	if err != nil {
		t.Fatalf("agent list failed: %v\nstdout: %s\nstderr: %s", err, listStdout, listStderr)
	}

	var accounts []domain.AgentAccount
	if err := json.Unmarshal([]byte(listStdout), &accounts); err != nil {
		t.Fatalf("failed to parse agent list JSON: %v\noutput: %s", err, listStdout)
	}

	found := false
	for _, listed := range accounts {
		if listed.Provider != domain.ProviderNylas {
			t.Fatalf("agent list returned non-nylas provider %q in %+v", listed.Provider, listed)
		}
		if listed.Email == email {
			found = true
		}
	}
	if !found {
		t.Fatalf("agent list did not include created account %q\noutput: %s", email, listStdout)
	}

	deleteStdout, deleteStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "delete", email, "--yes")
	if err != nil {
		t.Fatalf("agent delete by email failed: %v\nstdout: %s\nstderr: %s", err, deleteStdout, deleteStderr)
	}
	if !strings.Contains(strings.ToLower(deleteStdout), "deleted successfully") {
		t.Fatalf("expected delete confirmation in stdout, got: %s", deleteStdout)
	}

	if exists, _ := waitForAgentByEmail(t, client, email, false); exists {
		t.Fatalf("agent account %q still exists after delete", email)
	}
	created = nil
}

func TestCLI_AgentDelete_ByID(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "delete-id")
	account := createAgentForTest(t, client, email)

	t.Cleanup(func() {
		if exists, listed := waitForAgentByEmail(t, client, email, true); exists {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, listed.ID)
		}
	})

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "delete", account.ID, "--yes")
	if err != nil {
		t.Fatalf("agent delete by ID failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if exists, _ := waitForAgentByEmail(t, client, email, false); exists {
		t.Fatalf("agent account %q still exists after delete by ID", email)
	}
}

func TestCLI_AgentDelete_CancelKeepsAccount(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "cancel")
	account := createAgentForTest(t, client, email)

	t.Cleanup(func() {
		if exists, listed := waitForAgentByEmail(t, client, email, true); exists {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, listed.ID)
		}
	})

	acquireRateLimit(t)
	stdout, stderr, err := runCLIWithInputAndOverrides(2*time.Minute, "n\n", env, "agent", "delete", account.Email)
	if err != nil {
		t.Fatalf("agent delete cancel flow failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Deletion cancelled.") {
		t.Fatalf("expected cancellation message, got stdout: %s", stdout)
	}

	if exists, listed := waitForAgentByEmail(t, client, email, true); !exists || listed.ID != account.ID {
		t.Fatalf("agent account %q should still exist after cancellation", email)
	}
}

func TestCLI_AgentCreate_RequiresEmail(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("agent", "create")
	if err == nil {
		t.Fatalf("expected agent create without email to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "accepts 1 arg") && !strings.Contains(strings.ToLower(stderr), "argument") {
		t.Fatalf("expected missing argument error, got stderr: %s", stderr)
	}
}

func TestCLI_AgentCreate_InvalidEmailWithSpaces(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	stdout, stderr, err := runCLIWithOverrides(30*time.Second, env, "agent", "create", "bad email")
	if err == nil {
		t.Fatalf("expected invalid email with spaces to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "should not contain spaces") {
		t.Fatalf("expected invalid email error, got stderr: %s", stderr)
	}
}

func TestCLI_AgentDelete_InvalidIdentifier(t *testing.T) {
	skipIfMissingCreds(t)

	env := newAgentSandboxEnv(t)
	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 30*time.Second, env, "agent", "delete", "invalid-agent-id", "--yes")
	if err == nil {
		t.Fatalf("expected invalid agent delete to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "not found") && !strings.Contains(strings.ToLower(stderr), "failed to get agent account") {
		t.Fatalf("expected not found error, got stderr: %s", stderr)
	}
}

func skipIfMissingAgentDomain(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(os.Getenv("NYLAS_AGENT_DOMAIN")) == "" {
		t.Skip("NYLAS_AGENT_DOMAIN not set - skipping agent account lifecycle tests")
	}
}

func newAgentSandboxEnv(t *testing.T) map[string]string {
	t.Helper()
	configHome := t.TempDir()
	return map[string]string{
		"XDG_CONFIG_HOME":             configHome,
		"HOME":                        configHome,
		"NYLAS_DISABLE_KEYRING":       "true",
		"NYLAS_FILE_STORE_PASSPHRASE": "integration-test-file-store-passphrase",
		"NYLAS_GRANT_ID":              "",
	}
}

func newAgentTestEmail(t *testing.T, prefix string) string {
	t.Helper()
	domainName := strings.TrimSpace(os.Getenv("NYLAS_AGENT_DOMAIN"))
	domainName = strings.Trim(domainName, `"'`)
	domainName = strings.TrimPrefix(domainName, "@")
	if domainName == "" {
		t.Fatal("NYLAS_AGENT_DOMAIN is empty")
	}
	slug := strings.ReplaceAll(prefix, "_", "-")
	return fmt.Sprintf("it-%s-%d@%s", slug, time.Now().UnixNano(), domainName)
}

func createAgentForTest(t *testing.T, client interface {
	CreateAgentAccount(context.Context, string, string) (*domain.AgentAccount, error)
}, email string) *domain.AgentAccount {
	t.Helper()
	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	account, err := client.CreateAgentAccount(ctx, email, "")
	if err != nil {
		t.Fatalf("failed to create agent account %q for test setup: %v", email, err)
	}
	return account
}

func validAgentTestPassword() string {
	return "ValidAgentPass123ABC!"
}

func waitForAgentByEmail(t *testing.T, client interface {
	ListAgentAccounts(context.Context) ([]domain.AgentAccount, error)
}, email string, wantPresent bool) (bool, *domain.AgentAccount) {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		accounts, err := client.ListAgentAccounts(ctx)
		cancel()
		if err == nil {
			for i := range accounts {
				if strings.EqualFold(accounts[i].Email, email) {
					if wantPresent {
						return true, &accounts[i]
					}
					time.Sleep(500 * time.Millisecond)
					goto next
				}
			}
			if !wantPresent {
				return false, nil
			}
		}

	next:
		time.Sleep(500 * time.Millisecond)
	}

	if wantPresent {
		return false, nil
	}
	return true, nil
}

func runCLIWithInputAndOverrides(timeout time.Duration, input string, envOverrides map[string]string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, testBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = cliTestEnv(envOverrides)

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
