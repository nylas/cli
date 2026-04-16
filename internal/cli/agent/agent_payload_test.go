package agent

import (
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestLoadRulePayload(t *testing.T) {
	payload, err := loadRulePayload("", "", rulePayloadOptions{
		Name:       "Block Example",
		Conditions: []string{"from.domain,is,example.com"},
		Actions:    []string{"mark_as_spam"},
	}, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "Block Example", payload["name"])
		assert.Equal(t, true, payload["enabled"])
		assert.Equal(t, "inbound", payload["trigger"])

		matchPayload, ok := payload["match"].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, "all", matchPayload["operator"])
			assert.Len(t, matchPayload["conditions"], 1)
			conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
			if assert.True(t, ok) && assert.Len(t, conditions, 1) {
				assert.Equal(t, "example.com", conditions[0].Value)
			}
		}

		actions, ok := payload["actions"].([]domain.RuleAction)
		if assert.True(t, ok) && assert.Len(t, actions, 1) {
			assert.Equal(t, "mark_as_spam", actions[0].Type)
		}
	}

	payload, err = loadRulePayload(`{"name":"JSON Name","trigger":"inbound"}`, "", rulePayloadOptions{
		Name:          "Flag Name",
		MatchOperator: "any",
		Conditions:    []string{"from.address,is,ceo@example.com"},
		Actions:       []string{"assign_to_folder=vip"},
	}, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "Flag Name", payload["name"])
		matchPayload, ok := payload["match"].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, "any", matchPayload["operator"])
			conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
			if assert.True(t, ok) && assert.Len(t, conditions, 1) {
				assert.Equal(t, "ceo@example.com", conditions[0].Value)
			}
		}
		actions, ok := payload["actions"].([]domain.RuleAction)
		if assert.True(t, ok) && assert.Len(t, actions, 1) {
			assert.Equal(t, "assign_to_folder", actions[0].Type)
			assert.Equal(t, "vip", actions[0].Value)
		}
	}

	payload, err = loadRulePayload("", "", rulePayloadOptions{
		Name:       "Preserve Strings",
		Conditions: []string{"from.address,is,true", "from.tld,is,123"},
		Actions:    []string{"assign_to_folder=123", "assign_to_folder=true"},
	}, true)
	if assert.NoError(t, err) {
		matchPayload, ok := payload["match"].(map[string]any)
		if assert.True(t, ok) {
			conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
			if assert.True(t, ok) && assert.Len(t, conditions, 2) {
				assert.Equal(t, "true", conditions[0].Value)
				assert.Equal(t, "123", conditions[1].Value)
			}
		}
		actions, ok := payload["actions"].([]domain.RuleAction)
		if assert.True(t, ok) && assert.Len(t, actions, 2) {
			assert.Equal(t, "123", actions[0].Value)
			assert.Equal(t, "true", actions[1].Value)
		}
	}

	_, err = loadRulePayload("", "", rulePayloadOptions{}, true)
	assert.EqualError(t, err, "rule create requires a rule definition")

	_, err = loadRulePayload("", "", rulePayloadOptions{
		Name:    "Block Example",
		Actions: []string{"mark_as_spam"},
	}, true)
	assert.EqualError(t, err, "rule create is missing required fields")

	_, err = loadRulePayload("", "", rulePayloadOptions{
		Name:        "Block Example",
		Conditions:  []string{"from.domain,is,example.com"},
		Actions:     []string{"mark_as_spam"},
		EnabledSet:  true,
		DisabledSet: true,
	}, true)
	assert.EqualError(t, err, "cannot combine --enabled with --disabled")

	_, err = loadRulePayload("", "", rulePayloadOptions{
		Name:       "Block Example",
		Conditions: []string{"invalid"},
		Actions:    []string{"mark_as_spam"},
	}, true)
	assert.EqualError(t, err, "invalid --condition value")

	_, err = loadRulePayload("", "", rulePayloadOptions{
		Name:       "Block Example",
		Conditions: []string{"from.domain,is,example.com"},
		Actions:    []string{"=broken"},
	}, true)
	assert.EqualError(t, err, "invalid --action value")
}

func TestPreserveRuleMatchOperator(t *testing.T) {
	payload := map[string]any{
		"match": map[string]any{
			"conditions": []domain.RuleCondition{{
				Field:    "from.domain",
				Operator: "is",
				Value:    "example.com",
			}},
		},
	}

	preserveRuleMatchOperator(payload, &domain.Rule{
		Match: &domain.RuleMatch{Operator: "any"},
	})

	matchPayload, ok := payload["match"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, "any", matchPayload["operator"])
	}
}

func TestPreserveRuleMatchOperator_NoOverride(t *testing.T) {
	payload := map[string]any{
		"match": map[string]any{
			"operator": "all",
			"conditions": []domain.RuleCondition{{
				Field:    "from.domain",
				Operator: "is",
				Value:    "example.com",
			}},
		},
	}

	preserveRuleMatchOperator(payload, &domain.Rule{
		Match: &domain.RuleMatch{Operator: "any"},
	})

	matchPayload, ok := payload["match"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, "all", matchPayload["operator"])
	}
}

func TestRuleJSONPreservesZeroAndFalseValues(t *testing.T) {
	priority := 0
	enabled := false
	rule := domain.Rule{
		ID:       "rule-123",
		Priority: &priority,
		Enabled:  &enabled,
	}

	output := captureStdout(t, func() {
		assert.NoError(t, common.PrintJSON(rule))
	})

	assert.Contains(t, output, `"priority": 0`)
	assert.Contains(t, output, `"enabled": false`)
}

func TestValidateAgentAppPassword(t *testing.T) {
	assert.NoError(t, validateAgentAppPassword(""))
	assert.NoError(t, validateAgentAppPassword("ValidAgentPass123ABC!"))

	assert.EqualError(t, validateAgentAppPassword("short"), "app password must be between 18 and 40 characters")
	assert.EqualError(t, validateAgentAppPassword("Invalid Agent Pass123"), "app password must use printable ASCII characters only and cannot contain spaces")
	assert.EqualError(t, validateAgentAppPassword("alllowercasepassword123"), "app password must include at least one uppercase letter, one lowercase letter, and one digit")
}

func TestDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	assert.Equal(t, "delete <agent-id|email>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("yes"))
	assert.NotNil(t, cmd.Flags().Lookup("force"))
}

func TestIsDeleteConfirmed(t *testing.T) {
	assert.True(t, isDeleteConfirmed("y\n"))
	assert.True(t, isDeleteConfirmed("yes\n"))
	assert.True(t, isDeleteConfirmed("delete\n"))
	assert.False(t, isDeleteConfirmed("n\n"))
	assert.False(t, isDeleteConfirmed("\n"))
}

func TestFindNylasConnector(t *testing.T) {
	connectors := []domain.Connector{
		{Provider: "google", ID: "conn-google"},
		{Provider: "nylas"},
	}

	connector := findNylasConnector(connectors)
	assert.NotNil(t, connector)
	assert.Equal(t, "nylas", connector.Provider)
	assert.Empty(t, connector.ID)
	assert.Nil(t, findNylasConnector([]domain.Connector{{Provider: "google"}}))
}

func TestFormatConnectorSummary(t *testing.T) {
	assert.Equal(t, "nylas", formatConnectorSummary(domain.Connector{Provider: "nylas"}))
	assert.Equal(t, "nylas (conn-nylas)", formatConnectorSummary(domain.Connector{
		Provider: "nylas",
		ID:       "conn-nylas",
	}))
}
