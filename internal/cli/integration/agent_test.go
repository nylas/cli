//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
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
		// Use the ID we already have — no need for a list lookup. GET-by-id
		// is consistent against fresh grants while the list endpoint can
		// lag tens of seconds.
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = client.DeleteAgentAccount(ctx, created.ID)
	})

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "account", "create", email, "--app-password", appPassword, "--json")
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
	env["NYLAS_GRANT_ID"] = account.ID

	listStdout, listStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "account", "list", "--json")
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

	deleteStdout, deleteStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "account", "delete", email, "--yes")
	if err != nil {
		t.Fatalf("agent delete by email failed: %v\nstdout: %s\nstderr: %s", err, deleteStdout, deleteStderr)
	}
	if !strings.Contains(strings.ToLower(deleteStdout), "deleted successfully") {
		t.Fatalf("expected delete confirmation in stdout, got: %s", deleteStdout)
	}

	if exists, _ := waitForAgentByID(t, client, account.ID, false); exists {
		t.Fatalf("agent account %q still exists after delete", email)
	}
	created = nil
}

func TestCLI_AgentStatus(t *testing.T) {
	skipIfMissingCreds(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	connectors, err := client.ListConnectors(ctx)
	cancel()
	if err != nil {
		t.Fatalf("failed to list connectors for status test: %v", err)
	}

	expectedConfigured := false
	expectedConnectorID := ""
	for _, connector := range connectors {
		if connector.Provider == string(domain.ProviderNylas) {
			expectedConfigured = true
			expectedConnectorID = connector.ID
			break
		}
	}

	acquireRateLimit(t)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	accountsBefore, err := client.ListAgentAccounts(ctx)
	cancel()
	if err != nil {
		t.Fatalf("failed to list agent accounts for status test: %v", err)
	}

	jsonStdout, jsonStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "status", "--json")
	if err != nil {
		t.Fatalf("agent status --json failed: %v\nstdout: %s\nstderr: %s", err, jsonStdout, jsonStderr)
	}

	var result struct {
		ConnectorConfigured bool                  `json:"connector_configured"`
		ConnectorID         string                `json:"connector_id"`
		AccountCount        int                   `json:"account_count"`
		Accounts            []domain.AgentAccount `json:"accounts"`
	}
	if err := json.Unmarshal([]byte(jsonStdout), &result); err != nil {
		t.Fatalf("failed to parse agent status JSON: %v\noutput: %s", err, jsonStdout)
	}

	if result.ConnectorConfigured != expectedConfigured {
		t.Fatalf("connector_configured = %t, want %t", result.ConnectorConfigured, expectedConfigured)
	}
	if expectedConfigured && result.ConnectorID != expectedConnectorID {
		t.Fatalf("connector_id = %q, want %q", result.ConnectorID, expectedConnectorID)
	}
	if result.AccountCount != len(result.Accounts) {
		t.Fatalf("account_count = %d, accounts length = %d", result.AccountCount, len(result.Accounts))
	}

	if err := waitForAgentStatusSnapshotMatch(t, client, result.Accounts); err != nil {
		t.Fatalf("agent status accounts did not converge to a live snapshot: %v (before=%v, status=%v)", err, agentAccountIDs(accountsBefore), agentAccountIDs(result.Accounts))
	}

	textStdout, textStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "status")
	if err != nil {
		t.Fatalf("agent status failed: %v\nstdout: %s\nstderr: %s", err, textStdout, textStderr)
	}
	if !strings.Contains(textStdout, "Agent Status") {
		t.Fatalf("expected status heading, got: %s", textStdout)
	}
	if !strings.Contains(textStdout, "Connector:") {
		t.Fatalf("expected connector line, got: %s", textStdout)
	}
	if !strings.Contains(textStdout, "Accounts:") {
		t.Fatalf("expected accounts line, got: %s", textStdout)
	}
}

func waitForAgentStatusSnapshotMatch(t *testing.T, client interface {
	ListAgentAccounts(context.Context) ([]domain.AgentAccount, error)
}, want []domain.AgentAccount) error {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	var lastIDs []string

	for time.Now().Before(deadline) {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		accounts, err := client.ListAgentAccounts(ctx)
		cancel()
		if err != nil {
			return err
		}

		lastIDs = agentAccountIDs(accounts)
		if sameAgentAccountSet(accounts, want) {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("latest live accounts = %v, want %v", lastIDs, agentAccountIDs(want))
}

func sameAgentAccountSet(got, want []domain.AgentAccount) bool {
	if len(got) != len(want) {
		return false
	}

	gotIDs := agentAccountIDs(got)
	wantIDs := agentAccountIDs(want)
	slices.Sort(gotIDs)
	slices.Sort(wantIDs)
	return slices.Equal(gotIDs, wantIDs)
}

func agentAccountIDs(accounts []domain.AgentAccount) []string {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.ID)
	}
	return ids
}

func TestCLI_AgentCreate_WithPolicyID(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	email := newAgentTestEmail(t, "policy-create")
	policyName := newPolicyTestName("account-create")
	client := getTestClient()

	var createdPolicy *domain.Policy
	var created *domain.AgentAccount
	t.Cleanup(func() {
		if created != nil {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = client.DeleteAgentAccount(ctx, created.ID)
			cancel()
		}
		if createdPolicy != nil {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeletePolicy(ctx, createdPolicy.ID)
		}
	})

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	policy, err := client.CreatePolicy(ctx, map[string]any{"name": policyName})
	cancel()
	if err != nil {
		t.Fatalf("failed to create policy for agent account test: %v", err)
	}
	createdPolicy = policy

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "account", "create", email, "--policy-id", policy.ID, "--json")
	if err != nil {
		t.Fatalf("agent create with policy failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var account domain.AgentAccount
	if err := json.Unmarshal([]byte(stdout), &account); err != nil {
		t.Fatalf("failed to parse agent create with policy JSON: %v\noutput: %s", err, stdout)
	}
	if account.Email != email {
		t.Fatalf("created email = %q, want %q", account.Email, email)
	}
	if account.Settings.PolicyID != "" && account.Settings.PolicyID != policy.ID {
		t.Fatalf("created policy_id = %q, want %q or empty response field", account.Settings.PolicyID, policy.ID)
	}
	created = &account

	if exists, fetched := waitForAgentByID(t, client, account.ID, true); !exists {
		t.Fatalf("created agent account %q did not appear via GET-by-id", email)
	} else if fetched.Settings.PolicyID != policy.ID {
		t.Fatalf("fetched policy_id = %q, want %q", fetched.Settings.PolicyID, policy.ID)
	}
}

func TestCLI_AgentUpdate_ByEmail(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	email := newAgentTestEmail(t, "update")
	appPassword := validAgentTestPassword()
	client := getTestClient()

	var created *domain.AgentAccount
	t.Cleanup(func() {
		if created != nil {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, created.ID)
		}
	})

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	account, err := client.CreateAgentAccount(ctx, email, "", "")
	cancel()
	if err != nil {
		t.Fatalf("failed to create agent account %q for update test: %v", email, err)
	}
	created = account
	env["NYLAS_GRANT_ID"] = account.ID

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"account",
		"update",
		email,
		"--app-password", appPassword,
		"--json",
	)
	if err != nil {
		t.Fatalf("agent update failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var updated domain.AgentAccount
	if err := json.Unmarshal([]byte(stdout), &updated); err != nil {
		t.Fatalf("failed to parse agent update JSON: %v\noutput: %s", err, stdout)
	}
	if updated.ID != created.ID {
		t.Fatalf("updated agent ID = %q, want %q", updated.ID, created.ID)
	}

	acquireRateLimit(t)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	refetched, err := client.GetAgentAccount(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to refetch agent account after update: %v", err)
	}
	if refetched.ID != created.ID {
		t.Fatalf("refetched agent ID = %q, want %q", refetched.ID, created.ID)
	}
}

func TestCLI_AgentUpdate_PreservesPolicyID(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	email := newAgentTestEmail(t, "update-policy")
	appPassword := validAgentTestPassword()
	policyName := newPolicyTestName("account-update")
	client := getTestClient()

	var createdPolicy *domain.Policy
	var created *domain.AgentAccount
	t.Cleanup(func() {
		if created != nil {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = client.DeleteAgentAccount(ctx, created.ID)
			cancel()
		}
		if createdPolicy != nil {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeletePolicy(ctx, createdPolicy.ID)
		}
	})

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	policy, err := client.CreatePolicy(ctx, map[string]any{"name": policyName})
	cancel()
	if err != nil {
		t.Fatalf("failed to create policy for agent update test: %v", err)
	}
	createdPolicy = policy

	acquireRateLimit(t)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	account, err := client.CreateAgentAccount(ctx, email, "", policy.ID)
	cancel()
	if err != nil {
		t.Fatalf("failed to create agent account %q for update test: %v", email, err)
	}
	created = account
	env["NYLAS_GRANT_ID"] = account.ID
	if exists, fetched := waitForAgentByID(t, client, account.ID, true); !exists {
		t.Fatalf("created agent account %q not retrievable by id", email)
	} else if fetched.Settings.PolicyID != policy.ID {
		t.Fatalf("fetched policy_id before update = %q, want %q", fetched.Settings.PolicyID, policy.ID)
	}

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(
		t,
		2*time.Minute,
		env,
		"agent",
		"account",
		"update",
		email,
		"--app-password", appPassword,
		"--json",
	)
	if err != nil {
		t.Fatalf("agent update with policy failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var updated domain.AgentAccount
	if err := json.Unmarshal([]byte(stdout), &updated); err != nil {
		t.Fatalf("failed to parse agent update with policy JSON: %v\noutput: %s", err, stdout)
	}
	if updated.ID != created.ID {
		t.Fatalf("updated agent ID = %q, want %q", updated.ID, created.ID)
	}

	acquireRateLimit(t)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	refetched, err := client.GetAgentAccount(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to refetch agent account after update: %v", err)
	}
	if refetched.Settings.PolicyID != policy.ID {
		t.Fatalf("refetched policy_id after update = %q, want %q", refetched.Settings.PolicyID, policy.ID)
	}
}

func TestCLI_AgentDelete_ByID(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "delete-id")
	account := createAgentForTest(t, client, email)

	t.Cleanup(func() {
		// Test deletes the account itself; cleanup is a best-effort
		// safety net. GET-by-id is consistent so we use it instead of
		// the lag-prone list endpoint.
		if exists, _ := waitForAgentByID(t, client, account.ID, true); exists {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, account.ID)
		}
	})

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "account", "delete", account.ID, "--yes")
	if err != nil {
		t.Fatalf("agent delete by ID failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if exists, _ := waitForAgentByID(t, client, account.ID, false); exists {
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
	env["NYLAS_GRANT_ID"] = account.ID

	t.Cleanup(func() {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = client.DeleteAgentAccount(ctx, account.ID)
	})

	acquireRateLimit(t)
	stdout, stderr, err := runCLIWithInputAndOverrides(2*time.Minute, "n\n", env, "agent", "account", "delete", account.Email)
	if err != nil {
		t.Fatalf("agent delete cancel flow failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Deletion cancelled.") {
		t.Fatalf("expected cancellation message, got stdout: %s", stdout)
	}

	// Cancellation must leave the grant intact — verify via GET-by-id
	// (consistent) instead of the lag-prone list endpoint.
	if exists, fetched := waitForAgentByID(t, client, account.ID, true); !exists || fetched.ID != account.ID {
		t.Fatalf("agent account %q should still exist after cancellation", email)
	}
}

func TestCLI_AgentGet_ByEmailAndID(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "get")
	account := createAgentForTest(t, client, email)
	env["NYLAS_GRANT_ID"] = account.ID

	t.Cleanup(func() {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = client.DeleteAgentAccount(ctx, account.ID)
	})

	byEmailStdout, byEmailStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "account", "get", email, "--json")
	if err != nil {
		t.Fatalf("agent get by email failed: %v\nstdout: %s\nstderr: %s", err, byEmailStdout, byEmailStderr)
	}

	var byEmail domain.AgentAccount
	if err := json.Unmarshal([]byte(byEmailStdout), &byEmail); err != nil {
		t.Fatalf("failed to parse agent get by email JSON: %v\noutput: %s", err, byEmailStdout)
	}
	if byEmail.ID != account.ID {
		t.Fatalf("agent get by email returned ID %q, want %q", byEmail.ID, account.ID)
	}

	byIDStdout, byIDStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "account", "get", account.ID, "--json")
	if err != nil {
		t.Fatalf("agent get by ID failed: %v\nstdout: %s\nstderr: %s", err, byIDStdout, byIDStderr)
	}

	var byID domain.AgentAccount
	if err := json.Unmarshal([]byte(byIDStdout), &byID); err != nil {
		t.Fatalf("failed to parse agent get by ID JSON: %v\noutput: %s", err, byIDStdout)
	}
	if byID.Email != email {
		t.Fatalf("agent get by ID returned email %q, want %q", byID.Email, email)
	}
}

func TestCLI_AgentCreate_RequiresEmail(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("agent", "account", "create")
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
	stdout, stderr, err := runCLIWithOverrides(30*time.Second, env, "agent", "account", "create", "bad email")
	if err == nil {
		t.Fatalf("expected invalid email with spaces to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "should not contain spaces") {
		t.Fatalf("expected invalid email error, got stderr: %s", stderr)
	}
}

func TestCLI_AgentGet_InvalidIdentifier(t *testing.T) {
	skipIfMissingCreds(t)

	env := newAgentSandboxEnv(t)
	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 30*time.Second, env, "agent", "account", "get", "invalid-agent-id", "--json")
	if err == nil {
		t.Fatalf("expected invalid agent get to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "not found") && !strings.Contains(strings.ToLower(stderr), "failed to get agent account") {
		t.Fatalf("expected not found error, got stderr: %s", stderr)
	}
}

func TestCLI_AgentDelete_InvalidIdentifier(t *testing.T) {
	skipIfMissingCreds(t)

	env := newAgentSandboxEnv(t)
	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 30*time.Second, env, "agent", "account", "delete", "invalid-agent-id", "--yes")
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
		"NYLAS_AGENT_GRANT_ID":        "",
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
	CreateAgentAccount(context.Context, string, string, string) (*domain.AgentAccount, error)
}, email string) *domain.AgentAccount {
	t.Helper()
	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	account, err := client.CreateAgentAccount(ctx, email, "", "")
	if err != nil {
		t.Fatalf("failed to create agent account %q for test setup: %v", email, err)
	}
	return account
}

func validAgentTestPassword() string {
	return "ValidAgentPass123ABC!"
}

// waitForAgentByEmail polls /v3/grants?provider=nylas (the LIST endpoint)
// for a managed agent account matching `email`. The list endpoint has been
// observed to lag freshly-created grants by tens of seconds — sometimes
// >70s — while GET-by-id is consistent immediately. Use waitForAgentByID
// for ID-based checks; reserve this helper for tests that genuinely
// validate the email→list resolution path (delete-by-email, get-by-email).
//
// Deadline is set to 90s to absorb the observed list-propagation lag.
func waitForAgentByEmail(t *testing.T, client interface {
	ListAgentAccounts(context.Context) ([]domain.AgentAccount, error)
}, email string, wantPresent bool) (bool, *domain.AgentAccount) {
	t.Helper()

	deadline := time.Now().Add(90 * time.Second)
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

// waitForAgentByID polls /v3/grants/<id> (the GET endpoint) for a managed
// agent account by its grant ID. GET-by-id is strongly consistent against
// freshly-created grants — verified by direct probing — so this is the
// preferred helper for visibility checks where the test already has the
// ID returned from create.
//
// wantPresent=true returns (true, account) on first hit, (false, nil) if
// the deadline expires without a hit. wantPresent=false inverts: returns
// (false, nil) once the API stops returning the grant, (true, last) if
// the deadline expires while it's still returned.
func waitForAgentByID(t *testing.T, client interface {
	GetAgentAccount(context.Context, string) (*domain.AgentAccount, error)
}, id string, wantPresent bool) (bool, *domain.AgentAccount) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	var last *domain.AgentAccount
	for time.Now().Before(deadline) {
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		account, err := client.GetAgentAccount(ctx, id)
		cancel()
		if err == nil && account != nil {
			last = account
			if wantPresent {
				return true, account
			}
		} else if errors.Is(err, domain.ErrGrantNotFound) {
			if !wantPresent {
				return false, nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if wantPresent {
		return false, nil
	}
	return true, last
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
