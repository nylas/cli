package agent

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRulePayload_PreservesOutboundTrigger(t *testing.T) {
	payload, err := loadRulePayload("", "", rulePayloadOptions{
		Name:       "Block Replies",
		Trigger:    ruleTriggerOutbound,
		Conditions: []string{"outbound.type,is,reply"},
		Actions:    []string{"block"},
	}, true)
	require.NoError(t, err)

	assert.Equal(t, ruleTriggerOutbound, payload["trigger"])

	matchPayload, ok := payload["match"].(map[string]any)
	require.True(t, ok)

	conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
	require.True(t, ok)
	require.Len(t, conditions, 1)
	assert.Equal(t, ruleConditionFieldOutboundType, conditions[0].Field)
	assert.Equal(t, ruleConditionOperatorIs, conditions[0].Operator)
	assert.Equal(t, ruleOutboundTypeReply, conditions[0].Value)
}

func TestValidateRulePayload_RejectsRecipientConditionsOnInboundRules(t *testing.T) {
	err := validateRulePayload(map[string]any{
		"name":    "Inbound Recipient Rule",
		"trigger": ruleTriggerInbound,
		"match": map[string]any{
			"conditions": []map[string]any{{
				"field":    ruleConditionFieldRecipientDomain,
				"operator": ruleConditionOperatorIs,
				"value":    "example.com",
			}},
		},
		"actions": []map[string]any{{
			"type": ruleActionArchive,
		}},
	}, nil, true)

	require.Error(t, err)
	assert.EqualError(t, err, "condition 1 uses recipient.* on an inbound rule")
}

func TestValidateRulePayload_RejectsInvalidOutboundTypeOperator(t *testing.T) {
	err := validateRulePayload(map[string]any{
		"name":    "Outbound Type Rule",
		"trigger": ruleTriggerOutbound,
		"match": map[string]any{
			"conditions": []map[string]any{{
				"field":    ruleConditionFieldOutboundType,
				"operator": ruleConditionOperatorContains,
				"value":    ruleOutboundTypeReply,
			}},
		},
		"actions": []map[string]any{{
			"type": ruleActionMarkAsStarred,
		}},
	}, nil, true)

	require.Error(t, err)
	assert.EqualError(t, err, "outbound.type only supports is and is_not operators")
}

func TestValidateRulePayload_RejectsBlockWithOtherActions(t *testing.T) {
	err := validateRulePayload(map[string]any{
		"name": "Mixed Actions Rule",
		"match": map[string]any{
			"conditions": []map[string]any{{
				"field":    ruleConditionFieldFromDomain,
				"operator": ruleConditionOperatorIs,
				"value":    "example.com",
			}},
		},
		"actions": []map[string]any{
			{"type": ruleActionBlock},
			{"type": ruleActionArchive},
		},
	}, nil, true)

	require.Error(t, err)
	assert.EqualError(t, err, "block cannot be combined with other actions")
}

func TestValidateRulePayload_UpdateUsesExistingTrigger(t *testing.T) {
	err := validateRulePayload(map[string]any{
		"match": map[string]any{
			"conditions": []map[string]any{{
				"field":    ruleConditionFieldRecipientAddress,
				"operator": ruleConditionOperatorContains,
				"value":    "vip",
			}},
		},
	}, &domain.Rule{
		Trigger: ruleTriggerOutbound,
		Match: &domain.RuleMatch{
			Operator: ruleMatchOperatorAny,
			Conditions: []domain.RuleCondition{{
				Field:    ruleConditionFieldOutboundType,
				Operator: ruleConditionOperatorIs,
				Value:    ruleOutboundTypeCompose,
			}},
		},
		Actions: []domain.RuleAction{{
			Type: ruleActionArchive,
		}},
	}, false)

	require.NoError(t, err)
}
