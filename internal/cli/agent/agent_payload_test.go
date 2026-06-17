package agent

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRulePayload(t *testing.T) {
	payload, err := loadRulePayload("", "", rulePayloadOptions{
		Name:       "Block Example",
		Conditions: []string{"from.domain,is,example.com"},
		Actions:    []string{"block"},
	}, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "Block Example", payload["name"])
		assert.Equal(t, true, payload["enabled"])
		assert.Equal(t, ruleTriggerInbound, payload["trigger"])

		matchPayload, ok := payload["match"].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, ruleMatchOperatorAll, matchPayload["operator"])
			assert.Len(t, matchPayload["conditions"], 1)
			conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
			if assert.True(t, ok) && assert.Len(t, conditions, 1) {
				assert.Equal(t, "example.com", conditions[0].Value)
			}
		}

		actions, ok := payload["actions"].([]domain.RuleAction)
		if assert.True(t, ok) && assert.Len(t, actions, 1) {
			assert.Equal(t, ruleActionBlock, actions[0].Type)
		}
	}

	payload, err = loadRulePayload(`{"name":"JSON Name","trigger":"inbound"}`, "", rulePayloadOptions{
		Name:          "Archive VIP sender",
		MatchOperator: "any",
		Conditions:    []string{"from.address,is,ceo@example.com", "from.domain,is,example.com"},
		Actions:       []string{"assign_to_folder=vip", "mark_as_starred"},
	}, true)
	if assert.NoError(t, err) {
		assert.Equal(t, "Archive VIP sender", payload["name"])
		matchPayload, ok := payload["match"].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, ruleMatchOperatorAny, matchPayload["operator"])
			conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
			if assert.True(t, ok) && assert.Len(t, conditions, 2) {
				assert.Equal(t, ruleFieldFromAddress, conditions[0].Field)
				assert.Equal(t, "ceo@example.com", conditions[0].Value)
				assert.Equal(t, ruleFieldFromDomain, conditions[1].Field)
				assert.Equal(t, "example.com", conditions[1].Value)
			}
		}
		actions, ok := payload["actions"].([]domain.RuleAction)
		if assert.True(t, ok) && assert.Len(t, actions, 2) {
			assert.Equal(t, ruleActionAssignToFolder, actions[0].Type)
			assert.Equal(t, "vip", actions[0].Value)
			assert.Equal(t, ruleActionMarkAsStarred, actions[1].Type)
		}
	}

	payload, err = loadRulePayload("", "", rulePayloadOptions{
		Name:       "List-based Inbound Rule",
		Conditions: []string{"from.tld,in_list,list-123,list-456", "from.domain,is,example.com"},
		Actions:    []string{"assign_to_folder=folder-123", "mark_as_read"},
	}, true)
	if assert.NoError(t, err) {
		matchPayload, ok := payload["match"].(map[string]any)
		if assert.True(t, ok) {
			conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
			if assert.True(t, ok) && assert.Len(t, conditions, 2) {
				assert.Equal(t, []string{"list-123", "list-456"}, conditions[0].Value)
				assert.Equal(t, "example.com", conditions[1].Value)
			}
		}
		actions, ok := payload["actions"].([]domain.RuleAction)
		if assert.True(t, ok) && assert.Len(t, actions, 2) {
			assert.Equal(t, "folder-123", actions[0].Value)
			assert.Nil(t, actions[1].Value)
		}
	}

	_, err = loadRulePayload("", "", rulePayloadOptions{}, true)
	assert.EqualError(t, err, "rule create requires a rule definition")

	_, err = loadRulePayload("", "", rulePayloadOptions{
		Name:    "Block Example",
		Actions: []string{"block"},
	}, true)
	assert.EqualError(t, err, "rule create is missing required fields")

	_, err = loadRulePayload("", "", rulePayloadOptions{
		Name:        "Block Example",
		Conditions:  []string{"from.domain,is,example.com"},
		Actions:     []string{"block"},
		EnabledSet:  true,
		DisabledSet: true,
	}, true)
	assert.EqualError(t, err, "cannot combine --enabled with --disabled")

	_, err = loadRulePayload("", "", rulePayloadOptions{
		Name:       "Block Example",
		Conditions: []string{"invalid"},
		Actions:    []string{"block"},
	}, true)
	assert.EqualError(t, err, "invalid --condition value")

	_, err = loadRulePayload("", "", rulePayloadOptions{
		Name:       "Block Example",
		Conditions: []string{"from.domain,is,example.com"},
		Actions:    []string{"=broken"},
	}, true)
	assert.EqualError(t, err, "invalid --action value")
}

func TestLoadRulePayload_ValidatesSupportedRuleSurface(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		opts    rulePayloadOptions
		wantErr string
	}{
		{
			name: "unsupported trigger",
			opts: rulePayloadOptions{
				Name:       "Bad Trigger",
				Trigger:    "scheduled",
				Conditions: []string{"from.domain,is,example.com"},
				Actions:    []string{"block"},
			},
			wantErr: "unsupported rule trigger",
		},
		{
			name: "inbound rejects recipient field",
			opts: rulePayloadOptions{
				Name:       "Bad Inbound",
				Trigger:    ruleTriggerInbound,
				Conditions: []string{"recipient.domain,is,example.com"},
				Actions:    []string{"block"},
			},
			wantErr: "inbound rules only support from.* conditions",
		},
		{
			name: "assign_to_folder requires value",
			opts: rulePayloadOptions{
				Name:       "Missing Folder Value",
				Conditions: []string{"from.domain,is,example.com"},
				Actions:    []string{"assign_to_folder"},
			},
			wantErr: "assign_to_folder requires a folder value",
		},
		{
			name: "block cannot combine",
			opts: rulePayloadOptions{
				Name:       "Mixed Block",
				Conditions: []string{"from.domain,is,example.com"},
				Actions:    []string{"block", "archive"},
			},
			wantErr: "block cannot be combined with other actions",
		},
		{
			name: "flags create requires conditions and actions",
			opts: rulePayloadOptions{
				Name: "JSON Name",
			},
			wantErr: "rule create is missing required fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadRulePayload(tt.data, "", tt.opts, true)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestLoadRulePayload_FlagSurfacePreservesLegacyAndFutureValues(t *testing.T) {
	payload, err := loadRulePayload("", "", rulePayloadOptions{
		Name:       "Legacy Rule",
		Conditions: []string{"subject.contains,contains,vip"},
		Actions:    []string{"move_to_folder=vip", "tag=important"},
	}, true)
	require.NoError(t, err)

	matchPayload, ok := payload["match"].(map[string]any)
	require.True(t, ok)

	conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
	require.True(t, ok)
	require.Len(t, conditions, 1)
	assert.Equal(t, "subject.contains", conditions[0].Field)
	assert.Equal(t, ruleConditionOperatorContains, conditions[0].Operator)
	assert.Equal(t, "vip", conditions[0].Value)

	actions, ok := payload["actions"].([]domain.RuleAction)
	require.True(t, ok)
	require.Len(t, actions, 2)
	assert.Equal(t, ruleActionAssignToFolder, actions[0].Type)
	assert.Equal(t, "vip", actions[0].Value)
	assert.Equal(t, "tag", actions[1].Type)
	assert.Equal(t, "important", actions[1].Value)
}

func TestLoadRulePayloadDetails_MixedJSONAndFlagsAreNotPureJSON(t *testing.T) {
	loaded, err := loadRulePayloadDetails(`{"name":"Mixed","description":"Mixed"}`, "", rulePayloadOptions{
		Conditions: []string{"from.domain,is,example.com"},
		Actions:    []string{"archive"},
	}, true)
	require.NoError(t, err)
	assert.True(t, loaded.UsingJSON)
	assert.False(t, loaded.PureJSON)

	matchPayload, ok := loaded.Payload["match"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, ruleMatchOperatorAll, matchPayload["operator"])
}

func TestLoadRulePayload_RawJSONRemainsOpaque(t *testing.T) {
	payload, err := loadRulePayload(`{
		"name":"Legacy JSON Rule",
		"trigger":"inbound",
		"match":{"conditions":[{"field":"subject.contains","operator":"contains","value":"vip"}]},
		"actions":[{"type":"move_to_folder","value":"vip"}],
		"future_field":"allowed-through"
	}`, "", rulePayloadOptions{}, true)
	require.NoError(t, err)

	assert.Equal(t, "Legacy JSON Rule", payload["name"])
	assert.Equal(t, "inbound", payload["trigger"])
	assert.Equal(t, "allowed-through", payload["future_field"])

	matchPayload, ok := payload["match"].(map[string]any)
	if assert.True(t, ok) {
		_, hasOperator := matchPayload["operator"]
		assert.False(t, hasOperator)
	}
}

func TestLoadRulePayload_SupportsOutboundRuleSurface(t *testing.T) {
	payload, err := loadRulePayload("", "", rulePayloadOptions{
		Name:          "Outbound Rule",
		Trigger:       ruleTriggerOutbound,
		MatchOperator: ruleMatchOperatorAny,
		Conditions: []string{
			"recipient.domain,is,example.com",
			"outbound.type,is,compose",
		},
		Actions: []string{"archive"},
	}, true)
	require.NoError(t, err)

	assert.Equal(t, ruleTriggerOutbound, payload["trigger"])

	matchPayload, ok := payload["match"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, ruleMatchOperatorAny, matchPayload["operator"])

		conditions, ok := matchPayload["conditions"].([]domain.RuleCondition)
		if assert.True(t, ok) {
			if assert.Len(t, conditions, 2) {
				assert.Equal(t, ruleFieldRecipientDomain, conditions[0].Field)
				assert.Equal(t, "example.com", conditions[0].Value)
				assert.Equal(t, ruleFieldOutboundType, conditions[1].Field)
				assert.Equal(t, ruleOutboundTypeCompose, conditions[1].Value)
			}
		}
	}
}

func TestValidateRulePayload_UsesExistingTriggerOnUpdate(t *testing.T) {
	err := validateRulePayload(map[string]any{
		"match": map[string]any{
			"conditions": []domain.RuleCondition{{
				Field:    ruleFieldFromDomain,
				Operator: ruleConditionOperatorIs,
				Value:    "example.com",
			}},
		},
		"actions": []domain.RuleAction{{
			Type: ruleActionMarkAsSpam,
		}},
	}, &domain.Rule{
		Trigger: ruleTriggerInbound,
	})
	assert.NoError(t, err)

	err = validateRulePayload(map[string]any{
		"match": map[string]any{
			"conditions": []domain.RuleCondition{{
				Field:    ruleFieldRecipientDomain,
				Operator: ruleConditionOperatorIs,
				Value:    "example.com",
			}, {
				Field:    ruleFieldOutboundType,
				Operator: ruleConditionOperatorIs,
				Value:    ruleOutboundTypeReply,
			}},
		},
		"actions": []domain.RuleAction{{
			Type: ruleActionArchive,
		}},
	}, &domain.Rule{
		Trigger: ruleTriggerOutbound,
	})
	assert.NoError(t, err)
}

func TestValidateRulePayload_RejectsTriggerOnlyUpdateWhenExistingConditionsAreIncompatible(t *testing.T) {
	err := validateRulePayload(map[string]any{
		"trigger": ruleTriggerInbound,
	}, &domain.Rule{
		Trigger: ruleTriggerOutbound,
		Match: &domain.RuleMatch{
			Conditions: []domain.RuleCondition{{
				Field:    ruleFieldRecipientDomain,
				Operator: ruleConditionOperatorIs,
				Value:    "example.com",
			}},
		},
	})

	assert.EqualError(t, err, "inbound rules only support from.* conditions")
}

func TestLoadRulePayload_FlagCreateDefaultsTriggerToInbound(t *testing.T) {
	payload, err := loadRulePayload("", "", rulePayloadOptions{
		Name:       "JSON Rule",
		Conditions: []string{"from.domain,is,example.com"},
		Actions:    []string{"archive"},
	}, true)
	require.NoError(t, err)

	assert.Equal(t, ruleTriggerInbound, payload["trigger"])

	matchPayload, ok := payload["match"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, ruleMatchOperatorAll, matchPayload["operator"])
	}
}

func TestLoadRulePayload_JSONCreateDoesNotInjectDefaults(t *testing.T) {
	payload, err := loadRulePayload(`{
		"name":"JSON Rule",
		"match":{"conditions":[{"field":"from.domain","operator":"is","value":"example.com"}]},
		"actions":[{"type":"archive"}]
	}`, "", rulePayloadOptions{}, true)
	require.NoError(t, err)

	_, hasTrigger := payload["trigger"]
	assert.False(t, hasTrigger)

	matchPayload, ok := payload["match"].(map[string]any)
	if assert.True(t, ok) {
		_, hasOperator := matchPayload["operator"]
		assert.False(t, hasOperator)
	}
}

func TestLoadRulePayload_AllowsMalformedJSONMatchPayloadThrough(t *testing.T) {
	payload, err := loadRulePayload(`{
		"name":"JSON Rule",
		"match":"invalid",
		"actions":[{"type":"archive"}]
	}`, "", rulePayloadOptions{}, true)
	require.NoError(t, err)

	assert.Equal(t, "invalid", payload["match"])
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

func TestFinalizeRuleUpdatePayload_SkipsValidationAndMutationForPureJSON(t *testing.T) {
	payload := map[string]any{
		"match": map[string]any{
			"conditions": []domain.RuleCondition{{
				Field:    ruleFieldRecipientDomain,
				Operator: ruleConditionOperatorIs,
				Value:    "example.com",
			}},
		},
	}

	err := finalizeRuleUpdatePayload(payload, &domain.Rule{
		Trigger: ruleTriggerInbound,
		Match: &domain.RuleMatch{
			Operator: ruleMatchOperatorAny,
		},
	}, true)
	require.NoError(t, err)

	matchPayload, ok := payload["match"].(map[string]any)
	require.True(t, ok)
	_, hasOperator := matchPayload["operator"]
	assert.False(t, hasOperator)
}

func TestFinalizeRuleUpdatePayload_MixedJSONAndFlagsPreservesOperator(t *testing.T) {
	payload := map[string]any{
		"description": "Updated",
		"match": map[string]any{
			"conditions": []domain.RuleCondition{{
				Field:    ruleFieldFromDomain,
				Operator: ruleConditionOperatorIs,
				Value:    "example.org",
			}},
		},
	}

	err := finalizeRuleUpdatePayload(payload, &domain.Rule{
		Trigger: ruleTriggerInbound,
		Match: &domain.RuleMatch{
			Operator: ruleMatchOperatorAny,
		},
	}, false)
	require.NoError(t, err)

	matchPayload, ok := payload["match"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, ruleMatchOperatorAny, matchPayload["operator"])
}

func TestFinalizeRuleUpdatePayload_MixedJSONAndFlagsStillValidateAgainstExistingRule(t *testing.T) {
	payload := map[string]any{
		"trigger": ruleTriggerInbound,
		"match": map[string]any{
			"conditions": []domain.RuleCondition{{
				Field:    ruleFieldRecipientDomain,
				Operator: ruleConditionOperatorIs,
				Value:    "example.com",
			}},
		},
	}

	err := finalizeRuleUpdatePayload(payload, &domain.Rule{
		Trigger: ruleTriggerOutbound,
		Match: &domain.RuleMatch{
			Operator: ruleMatchOperatorAny,
		},
	}, false)
	assert.EqualError(t, err, "inbound rules only support from.* conditions")
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

func TestValidateAgentName(t *testing.T) {
	assert.NoError(t, validateAgentName(""), "empty name is valid (field omitted)")
	assert.NoError(t, validateAgentName("Support Bot"))
	assert.NoError(t, validateAgentName(strings.Repeat("a", 256)), "256 chars is the upper bound")
	// Length is measured in runes, not bytes: 256 multi-byte chars are valid
	// even though they span far more than 256 bytes.
	assert.NoError(t, validateAgentName(strings.Repeat("あ", 256)), "256 multi-byte runes must be accepted")

	assert.EqualError(t, validateAgentName(strings.Repeat("a", 257)), "name must be 256 characters or fewer")
	assert.EqualError(t, validateAgentName(strings.Repeat("あ", 257)), "name must be 256 characters or fewer")
}

func TestCreateCmd_HasNameFlag(t *testing.T) {
	cmd := newCreateCmd()
	assert.NotNil(t, cmd.Flags().Lookup("name"), "create command must expose a --name flag")
}

func TestUpdateCmd_HasNameFlag(t *testing.T) {
	cmd := newUpdateCmd()
	assert.NotNil(t, cmd.Flags().Lookup("name"), "update command must expose a --name flag")
}

func TestResolveEffectiveName(t *testing.T) {
	// Not provided: the existing name is preserved (the grant update replaces
	// the whole record, so omitting it would clear the name).
	assert.Equal(t, "Existing Bot", resolveEffectiveName("Existing Bot", "", false))
	assert.Equal(t, "Existing Bot", resolveEffectiveName("Existing Bot", "Ignored", false))

	// Provided: the caller's value wins. (An explicit empty value resolves to ""
	// here, but the adapter omits an empty name from the payload, so this does
	// not clear an existing name end-to-end — clearing is unsupported.)
	assert.Equal(t, "Renamed Bot", resolveEffectiveName("Existing Bot", "Renamed Bot", true))
	assert.Equal(t, "", resolveEffectiveName("Existing Bot", "", true))
}

func TestUpdateAgentAccount_RetryPathPreservesName(t *testing.T) {
	// When create is retried without the app password and then patched to set
	// it, the patch must carry the name so it is not wiped.
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "settings.app_password is an unknown field",
	}
	var createNames, updateNames []string
	createCalls := 0

	client := stubAgentClient{
		listFn: func(ctx context.Context) ([]domain.AgentAccount, error) { return nil, nil },
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			createCalls++
			createNames = append(createNames, name)
			if appPassword != "" {
				return nil, initialErr
			}
			return &domain.AgentAccount{ID: "agent-new", Email: email, Name: name, Provider: domain.ProviderNylas}, nil
		},
		updateFn: func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
			updateNames = append(updateNames, name)
			return &domain.AgentAccount{ID: grantID, Email: email, Name: name, Provider: domain.ProviderNylas}, nil
		},
	}

	account, err := createAgentAccountWithFallback(context.Background(), client, "agent@example.com", "Support Bot", "ValidAgentPass123ABC!")
	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, []string{"Support Bot", "Support Bot"}, createNames, "both create attempts carry the name")
	assert.Equal(t, []string{"Support Bot"}, updateNames, "the password-setting patch preserves the name")
	assert.Equal(t, "Support Bot", account.Name)
}

func TestUpdateAgentAccount_ExistingAccountFallbackPreservesName(t *testing.T) {
	// Initial create fails on app_password, an existing account is found, and
	// we set the password via update. With no --name supplied, the existing
	// account's name must be preserved (not wiped by the full-record PATCH).
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "settings.app_password is an unknown field",
	}
	var updateName string

	client := stubAgentClient{
		listFn: func(ctx context.Context) ([]domain.AgentAccount, error) {
			return []domain.AgentAccount{{
				ID:       "agent-existing",
				Email:    "agent@example.com",
				Name:     "Existing Bot",
				Provider: domain.ProviderNylas,
			}}, nil
		},
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			return nil, initialErr
		},
		updateFn: func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
			updateName = name
			return &domain.AgentAccount{ID: grantID, Email: email, Name: name, Provider: domain.ProviderNylas}, nil
		},
	}

	// No name supplied by the caller (empty) — must fall back to the existing name.
	account, err := createAgentAccountWithFallback(context.Background(), client, "agent@example.com", "", "ValidAgentPass123ABC!")
	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, "Existing Bot", updateName, "existing display name must be preserved when no name is supplied")
	assert.Equal(t, "Existing Bot", account.Name)
}

func TestCreateAgentAccountWithFallback_PassesNameToCreate(t *testing.T) {
	var gotName string
	client := stubAgentClient{
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			gotName = name
			return &domain.AgentAccount{ID: "agent-new", Email: email, Name: name, Provider: domain.ProviderNylas}, nil
		},
	}

	account, err := createAgentAccountWithFallback(context.Background(), client, "agent@example.com", "Support Bot", "")
	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, "Support Bot", gotName)
	assert.Equal(t, "Support Bot", account.Name)
}

func TestDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	assert.Equal(t, "delete [agent-id|email]", cmd.Use)
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
