package agent

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentCmd(t *testing.T) {
	cmd := NewAgentCmd()

	assert.Equal(t, "agent", cmd.Use)
	assert.Contains(t, cmd.Aliases, "agents")
	assert.Contains(t, cmd.Short, "agent")
	assert.Contains(t, cmd.Long, "account subcommand")

	expected := []string{"account", "policy", "rule", "status"}
	cmdMap := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		cmdMap[sub.Name()] = true
	}
	for _, name := range expected {
		assert.True(t, cmdMap[name], "missing subcommand %s", name)
	}
}

func TestCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	assert.Equal(t, "create <email>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("app-password"))
	assert.NotNil(t, cmd.Flags().Lookup("policy-id"))
	assert.Contains(t, cmd.Long, "provider=nylas")
}

func TestAccountCmd(t *testing.T) {
	cmd := newAccountCmd()

	assert.Equal(t, "account", cmd.Use)
	assert.Contains(t, cmd.Short, "accounts")

	expected := []string{"create", "list", "get", "delete"}
	cmdMap := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		cmdMap[sub.Name()] = true
	}
	for _, name := range expected {
		assert.True(t, cmdMap[name], "missing subcommand %s", name)
	}
}

func TestGetCmd(t *testing.T) {
	cmd := newGetCmd()

	assert.Equal(t, "get <agent-id|email>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.Contains(t, cmd.Long, "grant ID or by email address")
}

func TestPolicyCmd(t *testing.T) {
	cmd := newPolicyCmd()

	assert.Equal(t, "policy", cmd.Use)
	assert.Contains(t, cmd.Short, "policies")

	expected := []string{"list", "get", "read", "create", "update", "delete"}
	cmdMap := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		cmdMap[sub.Name()] = true
	}
	for _, name := range expected {
		assert.True(t, cmdMap[name], "missing subcommand %s", name)
	}
}

func TestRuleCmd(t *testing.T) {
	cmd := newRuleCmd()

	assert.Equal(t, "rule", cmd.Use)
	assert.Contains(t, cmd.Short, "rules")

	expected := []string{"list", "get", "read", "create", "update", "delete"}
	cmdMap := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		cmdMap[sub.Name()] = true
	}
	for _, name := range expected {
		assert.True(t, cmdMap[name], "missing subcommand %s", name)
	}
}

func TestLoadPolicyPayload(t *testing.T) {
	payload, err := loadPolicyPayload("", "", "Test Policy", true)
	assert.NoError(t, err)
	assert.Equal(t, "Test Policy", payload["name"])

	_, err = loadPolicyPayload("", "", "", true)
	assert.EqualError(t, err, "policy name is required")
}

func TestPolicyListCmd(t *testing.T) {
	cmd := newPolicyListCmd()

	assert.Equal(t, "list", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("all"))
	assert.Contains(t, cmd.Long, "provider=nylas account")
	assert.Contains(t, cmd.Flags().Lookup("all").Usage, "provider=nylas accounts")
}

func TestPolicyReadCmd(t *testing.T) {
	cmd := newPolicyReadCmd()

	assert.Equal(t, "read <policy-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.Contains(t, cmd.Long, "Read details for a single policy")
}

func TestRuleListCmd(t *testing.T) {
	cmd := newRuleListCmd()

	assert.Equal(t, "list", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("all"))
	assert.NotNil(t, cmd.Flags().Lookup("policy-id"))
	assert.Contains(t, cmd.Long, "default grant")
}

func TestRuleReadCmd(t *testing.T) {
	cmd := newRuleReadCmd()

	assert.Equal(t, "read <rule-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("all"))
	assert.NotNil(t, cmd.Flags().Lookup("policy-id"))
	assert.Contains(t, cmd.Long, "Read details for a single rule")
}

func TestRuleCreateCmd(t *testing.T) {
	cmd := newRuleCreateCmd()

	assert.Equal(t, "create", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("description"))
	assert.NotNil(t, cmd.Flags().Lookup("priority"))
	assert.NotNil(t, cmd.Flags().Lookup("enabled"))
	assert.NotNil(t, cmd.Flags().Lookup("disabled"))
	assert.NotNil(t, cmd.Flags().Lookup("condition"))
	assert.NotNil(t, cmd.Flags().Lookup("action"))
	assert.Contains(t, cmd.Long, "--condition")
}

func TestRuleUpdateCmd(t *testing.T) {
	cmd := newRuleUpdateCmd()

	assert.Equal(t, "update <rule-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("description"))
	assert.NotNil(t, cmd.Flags().Lookup("priority"))
	assert.NotNil(t, cmd.Flags().Lookup("enabled"))
	assert.NotNil(t, cmd.Flags().Lookup("disabled"))
	assert.NotNil(t, cmd.Flags().Lookup("condition"))
	assert.NotNil(t, cmd.Flags().Lookup("action"))
	assert.Contains(t, cmd.Long, "--condition")
}

func TestPrintPolicyDetails(t *testing.T) {
	attachmentSize := int64(50480000)
	attachmentCount := 10
	allowedTypes := []string{"application/pdf", "text/plain"}
	totalMIME := int64(50480000)
	dailyMessages := int64(500)
	inboxRetention := 30
	spamRetention := 7
	additionalFolders := []string{"archive", "support"}
	useCidrAliasing := false
	useDNSBL := false
	useHeaderAnomaly := true
	spamSensitivity := 1.0

	policy := domain.Policy{
		ID:             "policy-123",
		Name:           "Default Policy",
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
		Rules:          []string{"rule-a", "rule-b"},
		Limits: &domain.PolicyLimits{
			LimitAttachmentSizeInBytes:      &attachmentSize,
			LimitAttachmentCount:            &attachmentCount,
			LimitAttachmentAllowedTypes:     &allowedTypes,
			LimitSizeTotalMimeInBytes:       &totalMIME,
			LimitCountDailyMessagePerGrant:  &dailyMessages,
			LimitInboxRetentionPeriodInDays: &inboxRetention,
			LimitSpamRetentionPeriodInDays:  &spamRetention,
		},
		Options: &domain.PolicyOptions{
			AdditionalFolders: &additionalFolders,
			UseCidrAliasing:   &useCidrAliasing,
		},
		SpamDetection: &domain.PolicySpamDetection{
			UseListDNSBL:              &useDNSBL,
			UseHeaderAnomalyDetection: &useHeaderAnomaly,
			SpamSensitivity:           &spamSensitivity,
		},
		CreatedAt: domain.UnixTime{Time: time.Date(2026, time.April, 13, 16, 49, 44, 0, time.UTC)},
		UpdatedAt: domain.UnixTime{Time: time.Date(2026, time.April, 13, 16, 49, 44, 0, time.UTC)},
	}

	output := captureStdout(t, func() {
		printPolicyDetails(policy)
	})

	assert.Contains(t, output, "Policy:       Default Policy")
	assert.Contains(t, output, "Rules:")
	assert.Contains(t, output, "1. rule-a")
	assert.Contains(t, output, "Limits:")
	assert.Contains(t, output, "Attachment size:")
	assert.Contains(t, output, "50480000 bytes")
	assert.Contains(t, output, "Allowed types:")
	assert.Contains(t, output, "application/pdf")
	assert.Contains(t, output, "Options:")
	assert.Contains(t, output, "Additional folders:")
	assert.Contains(t, output, "archive")
	assert.Contains(t, output, "CIDR aliasing:")
	assert.Contains(t, output, "Spam detection:")
	assert.Contains(t, output, "Use DNSBL:")
	assert.Contains(t, output, "Header anomaly detection:")
	assert.Contains(t, output, "Spam sensitivity:")
}

func TestPrintRuleDetails(t *testing.T) {
	priority := 10
	enabled := true
	rule := domain.Rule{
		ID:             "rule-123",
		Name:           "Block Example",
		Description:    "Blocks example.com",
		Priority:       &priority,
		Enabled:        &enabled,
		Trigger:        "inbound",
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
		Match: &domain.RuleMatch{
			Operator: "all",
			Conditions: []domain.RuleCondition{{
				Field:    "from.domain",
				Operator: "is",
				Value:    "example.com",
			}},
		},
		Actions: []domain.RuleAction{{
			Type: "mark_as_spam",
		}},
		CreatedAt: domain.UnixTime{Time: time.Date(2026, time.April, 13, 16, 49, 44, 0, time.UTC)},
		UpdatedAt: domain.UnixTime{Time: time.Date(2026, time.April, 13, 16, 49, 44, 0, time.UTC)},
	}

	output := captureStdout(t, func() {
		printRuleDetails(rule, []rulePolicyRef{{
			PolicyID:   "policy-123",
			PolicyName: "Default Policy",
			Accounts: []policyAgentAccountRef{{
				GrantID: "grant-123",
				Email:   "agent@example.com",
			}},
		}})
	})

	assert.Contains(t, output, "Rule:         Block Example")
	assert.Contains(t, output, "Policies:")
	assert.Contains(t, output, "Default Policy")
	assert.Contains(t, output, "agent@example.com")
	assert.Contains(t, output, "Match:")
	assert.Contains(t, output, "from.domain is example.com")
	assert.Contains(t, output, "Actions:")
	assert.Contains(t, output, "mark_as_spam")
}

func TestBuildPolicyAccountRefs(t *testing.T) {
	refsByPolicyID := buildPolicyAccountRefs([]domain.AgentAccount{
		{
			ID:       "grant-b",
			Email:    "beta@example.com",
			Provider: domain.ProviderNylas,
			Settings: domain.AgentAccountSettings{PolicyID: "policy-1"},
		},
		{
			ID:       "grant-a",
			Email:    "alpha@example.com",
			Provider: domain.ProviderNylas,
			Settings: domain.AgentAccountSettings{PolicyID: "policy-1"},
		},
		{
			ID:       "grant-empty",
			Email:    "empty@example.com",
			Provider: domain.ProviderNylas,
		},
	})

	if assert.Len(t, refsByPolicyID["policy-1"], 2) {
		assert.Equal(t, "alpha@example.com", refsByPolicyID["policy-1"][0].Email)
		assert.Equal(t, "grant-a", refsByPolicyID["policy-1"][0].GrantID)
		assert.Equal(t, "beta@example.com", refsByPolicyID["policy-1"][1].Email)
	}
	assert.Empty(t, refsByPolicyID[""])
}

func TestFormatPolicyAgentAccounts(t *testing.T) {
	assert.Equal(t, "", formatPolicyAgentAccounts(nil))
	assert.Equal(t,
		"alpha@example.com (grant-a), beta@example.com (grant-b)",
		formatPolicyAgentAccounts([]policyAgentAccountRef{
			{GrantID: "grant-a", Email: "alpha@example.com"},
			{GrantID: "grant-b", Email: "beta@example.com"},
		}),
	)
}

func TestFilterPoliciesWithAgentAccounts(t *testing.T) {
	policies := filterPoliciesWithAgentAccounts(
		[]domain.Policy{
			{ID: "policy-1", Name: "Attached"},
			{ID: "policy-2", Name: "Unused"},
		},
		map[string][]policyAgentAccountRef{
			"policy-1": {{
				GrantID: "grant-1",
				Email:   "agent@example.com",
			}},
		},
	)

	if assert.Len(t, policies, 1) {
		assert.Equal(t, "policy-1", policies[0].ID)
	}
}

func TestResolvePolicyForAgentOps(t *testing.T) {
	scope := &agentPolicyScope{
		AllPolicies: []domain.Policy{
			{ID: "policy-agent", Name: "Agent"},
			{ID: "policy-unattached", Name: "Unattached"},
		},
		PolicyRefsByID: map[string][]policyAgentAccountRef{
			"policy-agent": {{
				GrantID: "grant-agent",
				Email:   "agent@example.com",
			}},
		},
	}

	resolved, err := resolvePolicyForAgentOps(scope, "policy-agent")
	if assert.NoError(t, err) {
		assert.Equal(t, "policy-agent", resolved.Policy.ID)
		assert.True(t, resolved.AttachedToAgent)
	}

	resolved, err = resolvePolicyForAgentOps(scope, "policy-unattached")
	if assert.NoError(t, err) {
		assert.Equal(t, "policy-unattached", resolved.Policy.ID)
		assert.False(t, resolved.AttachedToAgent)
	}
}

func TestBuildRuleRefsByID(t *testing.T) {
	refsByRuleID := buildRuleRefsByID(
		[]domain.Policy{
			{ID: "policy-b", Name: "Beta", Rules: []string{"rule-1"}},
			{ID: "policy-a", Name: "Alpha", Rules: []string{"rule-1", "rule-2", "rule-1"}},
		},
		map[string][]policyAgentAccountRef{
			"policy-a": {{
				GrantID: "grant-a",
				Email:   "alpha@example.com",
			}},
			"policy-b": {{
				GrantID: "grant-b",
				Email:   "beta@example.com",
			}},
		},
	)

	if assert.Len(t, refsByRuleID["rule-1"], 2) {
		assert.Equal(t, "Alpha", refsByRuleID["rule-1"][0].PolicyName)
		assert.Equal(t, "Beta", refsByRuleID["rule-1"][1].PolicyName)
	}
	if assert.Len(t, refsByRuleID["rule-2"], 1) {
		assert.Equal(t, "policy-a", refsByRuleID["rule-2"][0].PolicyID)
	}
}

func TestRuleReferencedOutsideAgentScope(t *testing.T) {
	allPolicies := []domain.Policy{
		{ID: "policy-agent", Rules: []string{"rule-1"}},
		{ID: "policy-other", Rules: []string{"rule-1"}},
	}
	agentPolicies := []domain.Policy{
		{ID: "policy-agent", Rules: []string{"rule-1"}},
	}

	assert.True(t, ruleReferencedOutsideAgentScope(allPolicies, agentPolicies, "rule-1"))
	assert.False(t, ruleReferencedOutsideAgentScope(allPolicies, agentPolicies, "rule-2"))
}

func TestPoliciesLeftEmptyByRuleRemoval(t *testing.T) {
	blocking := policiesLeftEmptyByRuleRemoval([]domain.Policy{
		{ID: "policy-last", Name: "Last Rule", Rules: []string{"rule-1"}},
		{ID: "policy-shared", Name: "Has Spare", Rules: []string{"rule-1", "rule-2"}},
		{ID: "policy-other", Name: "Other Rule", Rules: []string{"rule-3"}},
	}, "rule-1")

	if assert.Len(t, blocking, 1) {
		assert.Equal(t, "policy-last", blocking[0].ID)
		assert.Equal(t, "Last Rule", blocking[0].Name)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	_ = r.Close()

	return buf.String()
}
