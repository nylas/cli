//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCLI_AgentPolicyLifecycle_CreateGetListUpdateDelete(t *testing.T) {
	skipIfMissingCreds(t)

	env := newAgentSandboxEnv(t)
	name := newPolicyTestName("create")
	updatedName := newPolicyTestName("updated")
	var created *domain.Policy
	client := getTestClient()

	t.Cleanup(func() {
		if created == nil || created.ID == "" {
			return
		}
		acquireRateLimit(t)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = client.DeletePolicy(ctx, created.ID)
	})

	createStdout, createStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "create", "--name", name, "--json")
	if err != nil {
		t.Fatalf("policy create failed: %v\nstdout: %s\nstderr: %s", err, createStdout, createStderr)
	}

	var policy domain.Policy
	if err := json.Unmarshal([]byte(createStdout), &policy); err != nil {
		t.Fatalf("failed to parse policy create JSON: %v\noutput: %s", err, createStdout)
	}
	if strings.TrimSpace(policy.ID) == "" {
		t.Fatalf("expected created policy ID, got empty output: %s", createStdout)
	}
	if policy.Name != name {
		t.Fatalf("created policy name = %q, want %q", policy.Name, name)
	}
	created = &policy

	getStdout, getStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "get", policy.ID, "--json")
	if err != nil {
		t.Fatalf("policy get failed: %v\nstdout: %s\nstderr: %s", err, getStdout, getStderr)
	}

	var fetched domain.Policy
	if err := json.Unmarshal([]byte(getStdout), &fetched); err != nil {
		t.Fatalf("failed to parse policy get JSON: %v\noutput: %s", err, getStdout)
	}
	if fetched.ID != policy.ID {
		t.Fatalf("policy get returned ID %q, want %q", fetched.ID, policy.ID)
	}

	readStdout, readStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "read", policy.ID, "--json")
	if err != nil {
		t.Fatalf("policy read failed: %v\nstdout: %s\nstderr: %s", err, readStdout, readStderr)
	}

	var readPolicy domain.Policy
	if err := json.Unmarshal([]byte(readStdout), &readPolicy); err != nil {
		t.Fatalf("failed to parse policy read JSON: %v\noutput: %s", err, readStdout)
	}
	if readPolicy.ID != policy.ID {
		t.Fatalf("policy read returned ID %q, want %q", readPolicy.ID, policy.ID)
	}

	readTextStdout, readTextStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "read", policy.ID)
	if err != nil {
		t.Fatalf("policy read text failed: %v\nstdout: %s\nstderr: %s", err, readTextStdout, readTextStderr)
	}
	if !strings.Contains(readTextStdout, "Limits:") {
		t.Fatalf("policy read text output should include limits section\noutput: %s", readTextStdout)
	}
	if !strings.Contains(readTextStdout, "Options:") {
		t.Fatalf("policy read text output should include options section\noutput: %s", readTextStdout)
	}
	if !strings.Contains(readTextStdout, "Spam detection:") {
		t.Fatalf("policy read text output should include spam detection section\noutput: %s", readTextStdout)
	}

	listStdout, listStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "list", "--all", "--json")
	if err != nil {
		t.Fatalf("policy list failed: %v\nstdout: %s\nstderr: %s", err, listStdout, listStderr)
	}

	var policies []domain.Policy
	if err := json.Unmarshal([]byte(listStdout), &policies); err != nil {
		t.Fatalf("failed to parse policy list JSON: %v\noutput: %s", err, listStdout)
	}

	for _, listed := range policies {
		if listed.ID == policy.ID {
			t.Fatalf("unattached policy %q should not appear in agent policy list --all\noutput: %s", policy.ID, listStdout)
		}
	}

	updateStdout, updateStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "update", policy.ID, "--name", updatedName, "--json")
	if err != nil {
		t.Fatalf("policy update failed: %v\nstdout: %s\nstderr: %s", err, updateStdout, updateStderr)
	}

	var updated domain.Policy
	if err := json.Unmarshal([]byte(updateStdout), &updated); err != nil {
		t.Fatalf("failed to parse policy update JSON: %v\noutput: %s", err, updateStdout)
	}
	if updated.ID != policy.ID {
		t.Fatalf("policy update returned ID %q, want %q", updated.ID, policy.ID)
	}

	confirmStdout, confirmStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "get", policy.ID, "--json")
	if err != nil {
		t.Fatalf("policy confirm get failed: %v\nstdout: %s\nstderr: %s", err, confirmStdout, confirmStderr)
	}

	var confirmed domain.Policy
	if err := json.Unmarshal([]byte(confirmStdout), &confirmed); err != nil {
		t.Fatalf("failed to parse policy confirm JSON: %v\noutput: %s", err, confirmStdout)
	}
	if confirmed.Name != updatedName {
		t.Fatalf("policy name after update = %q, want %q", confirmed.Name, updatedName)
	}

	deleteStdout, deleteStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "delete", policy.ID, "--yes")
	if err != nil {
		t.Fatalf("policy delete failed: %v\nstdout: %s\nstderr: %s", err, deleteStdout, deleteStderr)
	}
	if !strings.Contains(strings.ToLower(deleteStdout), "deleted") {
		t.Fatalf("expected delete confirmation in stdout, got: %s", deleteStdout)
	}

	created = nil
}

func TestCLI_AgentPolicyCreate_RequiresNameOrData(t *testing.T) {
	skipIfMissingCreds(t)

	env := newAgentSandboxEnv(t)
	stdout, stderr, err := runCLIWithOverrides(30*time.Second, env, "agent", "policy", "create")
	if err == nil {
		t.Fatalf("expected policy create without payload to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "policy name is required") {
		t.Fatalf("expected missing payload error, got stderr: %s", stderr)
	}
}

func TestCLI_AgentPolicyUpdate_RequiresChanges(t *testing.T) {
	skipIfMissingCreds(t)

	env := newAgentSandboxEnv(t)
	stdout, stderr, err := runCLIWithOverrides(30*time.Second, env, "agent", "policy", "update", "policy-id")
	if err == nil {
		t.Fatalf("expected policy update without changes to fail\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "requires at least one field") {
		t.Fatalf("expected missing update field error, got stderr: %s", stderr)
	}
}

func TestCLI_AgentPolicyList_ShowsAttachedAgentAccount(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "policy-list")
	policyName := newPolicyTestName("attached")

	var createdPolicy *domain.Policy
	var createdAccount *domain.AgentAccount

	t.Cleanup(func() {
		if createdAccount != nil && createdAccount.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, createdAccount.ID)
		}
		if createdPolicy != nil && createdPolicy.ID != "" {
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
		t.Fatalf("failed to create policy for list test: %v", err)
	}
	createdPolicy = policy

	createdAccount = createAgentWithPolicyForTest(t, email, createdPolicy.ID)
	env["NYLAS_GRANT_ID"] = createdAccount.ID

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "list")
	if err != nil {
		t.Fatalf("policy list failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, createdPolicy.Name) {
		t.Fatalf("policy list output missing policy name %q\noutput: %s", createdPolicy.Name, stdout)
	}
	if !strings.Contains(stdout, createdAccount.Email) {
		t.Fatalf("policy list output missing agent email %q\noutput: %s", createdAccount.Email, stdout)
	}
	if !strings.Contains(stdout, createdAccount.ID) {
		t.Fatalf("policy list output missing agent grant ID %q\noutput: %s", createdAccount.ID, stdout)
	}

	allStdout, allStderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "list", "--all")
	if err != nil {
		t.Fatalf("policy list --all failed: %v\nstdout: %s\nstderr: %s", err, allStdout, allStderr)
	}
	if !strings.Contains(allStdout, createdPolicy.Name) {
		t.Fatalf("policy list --all output missing policy name %q\noutput: %s", createdPolicy.Name, allStdout)
	}
	if !strings.Contains(allStdout, createdAccount.Email) {
		t.Fatalf("policy list --all output missing agent email %q\noutput: %s", createdAccount.Email, allStdout)
	}
	if !strings.Contains(allStdout, "Agent:") {
		t.Fatalf("policy list --all should include agent annotations\noutput: %s", allStdout)
	}
	if !strings.Contains(allStdout, fmt.Sprintf("(%s)", createdAccount.ID)) {
		t.Fatalf("policy list --all output missing agent grant ID %q\noutput: %s", createdAccount.ID, allStdout)
	}
	if strings.Contains(allStdout, "Agent: none") {
		t.Fatalf("policy list --all should not show policies without a provider=nylas account\noutput: %s", allStdout)
	}
}

func TestCLI_AgentPolicyDelete_RejectsAttachedPolicy(t *testing.T) {
	skipIfMissingCreds(t)
	skipIfMissingAgentDomain(t)

	env := newAgentSandboxEnv(t)
	client := getTestClient()
	email := newAgentTestEmail(t, "policy-delete")
	policyName := newPolicyTestName("delete-guard")

	var createdPolicy *domain.Policy
	var createdAccount *domain.AgentAccount

	t.Cleanup(func() {
		if createdAccount != nil && createdAccount.ID != "" {
			acquireRateLimit(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = client.DeleteAgentAccount(ctx, createdAccount.ID)
		}
		if createdPolicy != nil && createdPolicy.ID != "" {
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
		t.Fatalf("failed to create policy for delete guard test: %v", err)
	}
	createdPolicy = policy

	createdAccount = createAgentWithPolicyForTest(t, email, createdPolicy.ID)
	env["NYLAS_GRANT_ID"] = createdAccount.ID

	stdout, stderr, err := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, env, "agent", "policy", "delete", createdPolicy.ID, "--yes")
	if err == nil {
		t.Fatalf("expected policy delete to fail while attached\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "policy is attached to agent accounts") {
		t.Fatalf("expected attached policy error, got stderr: %s", stderr)
	}
	if !strings.Contains(stderr, createdAccount.Email) {
		t.Fatalf("expected attached agent email %q in stderr: %s", createdAccount.Email, stderr)
	}

	acquireRateLimit(t)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	confirmed, err := client.GetPolicy(ctx, createdPolicy.ID)
	if err != nil {
		t.Fatalf("expected policy to remain after rejected delete: %v", err)
	}
	if confirmed.ID != createdPolicy.ID {
		t.Fatalf("policy after rejected delete = %q, want %q", confirmed.ID, createdPolicy.ID)
	}
}

func createAgentWithPolicyForTest(t *testing.T, email, policyID string) *domain.AgentAccount {
	t.Helper()

	acquireRateLimit(t)
	ctx, cancel := context.WithTimeout(context.Background(), domain.TimeoutAPI)
	defer cancel()

	client := getTestClient()
	account, err := client.CreateAgentAccount(ctx, email, "", policyID)
	if err != nil {
		t.Fatalf("failed to create agent with policy: %v", err)
	}
	return account
}

func newPolicyTestName(prefix string) string {
	return fmt.Sprintf("it-policy-%s-%d", prefix, time.Now().UnixNano())
}
