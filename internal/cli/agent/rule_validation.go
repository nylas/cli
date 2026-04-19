package agent

import (
	"fmt"
	"slices"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

const (
	ruleTriggerInbound  = "inbound"
	ruleTriggerOutbound = "outbound"

	ruleFieldFromAddress      = "from.address"
	ruleFieldFromDomain       = "from.domain"
	ruleFieldFromTLD          = "from.tld"
	ruleFieldRecipientAddress = "recipient.address"
	ruleFieldRecipientDomain  = "recipient.domain"
	ruleFieldRecipientTLD     = "recipient.tld"
	ruleFieldOutboundType     = "outbound.type"

	ruleConditionOperatorIs       = "is"
	ruleConditionOperatorIsNot    = "is_not"
	ruleConditionOperatorContains = "contains"
	ruleConditionOperatorInList   = "in_list"

	ruleMatchOperatorAll = "all"
	ruleMatchOperatorAny = "any"

	ruleActionBlock          = "block"
	ruleActionMarkAsSpam     = "mark_as_spam"
	ruleActionAssignToFolder = "assign_to_folder"
	ruleActionMarkAsRead     = "mark_as_read"
	ruleActionMarkAsStarred  = "mark_as_starred"
	ruleActionArchive        = "archive"
	ruleActionTrash          = "trash"

	ruleOutboundTypeCompose = "compose"
	ruleOutboundTypeReply   = "reply"
)

var (
	supportedRuleTriggers = []string{
		ruleTriggerInbound,
		ruleTriggerOutbound,
	}
	supportedRuleFields = []string{
		ruleFieldFromAddress,
		ruleFieldFromDomain,
		ruleFieldFromTLD,
		ruleFieldRecipientAddress,
		ruleFieldRecipientDomain,
		ruleFieldRecipientTLD,
		ruleFieldOutboundType,
	}
	inboundRuleFields = []string{
		ruleFieldFromAddress,
		ruleFieldFromDomain,
		ruleFieldFromTLD,
	}
	outboundRuleFields = []string{
		ruleFieldFromAddress,
		ruleFieldFromDomain,
		ruleFieldFromTLD,
		ruleFieldRecipientAddress,
		ruleFieldRecipientDomain,
		ruleFieldRecipientTLD,
		ruleFieldOutboundType,
	}
	supportedRuleConditionOperators = []string{
		ruleConditionOperatorIs,
		ruleConditionOperatorIsNot,
		ruleConditionOperatorContains,
		ruleConditionOperatorInList,
	}
	outboundTypeOperators = []string{
		ruleConditionOperatorIs,
		ruleConditionOperatorIsNot,
	}
	supportedRuleActions = []string{
		ruleActionBlock,
		ruleActionMarkAsSpam,
		ruleActionAssignToFolder,
		ruleActionMarkAsRead,
		ruleActionMarkAsStarred,
		ruleActionArchive,
		ruleActionTrash,
	}
	supportedOutboundTypes = []string{
		ruleOutboundTypeCompose,
		ruleOutboundTypeReply,
	}
)

func canonicalRuleActionType(actionType string) string {
	switch strings.TrimSpace(actionType) {
	case "move_to_folder":
		return ruleActionAssignToFolder
	default:
		return strings.TrimSpace(actionType)
	}
}

func validateRulePayload(payload map[string]any, existingRule *domain.Rule) error {
	trigger := strings.TrimSpace(finalRuleTrigger(payload, existingRule))
	if trigger != "" && !slices.Contains(supportedRuleTriggers, trigger) {
		return common.NewUserError(
			"unsupported rule trigger",
			fmt.Sprintf("Use one of: %s", strings.Join(supportedRuleTriggers, ", ")),
		)
	}

	conditions, err := finalRuleConditions(payload, existingRule)
	if err != nil {
		return err
	}
	for _, condition := range conditions {
		if err := validateRuleCondition(trigger, condition); err != nil {
			return err
		}
	}

	actions, err := extractRuleActions(payload["actions"])
	if err != nil {
		return err
	}
	if err := validateRuleActions(actions); err != nil {
		return err
	}

	return nil
}

func finalRuleTrigger(payload map[string]any, existingRule *domain.Rule) string {
	if trigger := strings.TrimSpace(asString(payload["trigger"])); trigger != "" {
		return trigger
	}
	if existingRule != nil {
		return strings.TrimSpace(existingRule.Trigger)
	}
	return ""
}

func finalRuleConditions(payload map[string]any, existingRule *domain.Rule) ([]domain.RuleCondition, error) {
	matchPayload := copyStringAnyMap(payload["match"])
	if len(matchPayload) > 0 {
		if rawConditions, ok := matchPayload["conditions"]; ok {
			return extractRuleConditions(rawConditions)
		}
	}

	if !isRuleTriggerChanging(payload, existingRule) || existingRule == nil || existingRule.Match == nil {
		return nil, nil
	}

	return existingRule.Match.Conditions, nil
}

func isRuleTriggerChanging(payload map[string]any, existingRule *domain.Rule) bool {
	if existingRule == nil {
		return false
	}

	nextTrigger := strings.TrimSpace(asString(payload["trigger"]))
	if nextTrigger == "" {
		return false
	}

	return nextTrigger != strings.TrimSpace(existingRule.Trigger)
}

func validateRuleCondition(trigger string, condition domain.RuleCondition) error {
	field := strings.TrimSpace(condition.Field)
	if !slices.Contains(supportedRuleFields, field) {
		// Preserve compatibility for legacy and future server-side fields.
		return nil
	}

	operator := strings.TrimSpace(condition.Operator)
	operatorKnown := slices.Contains(supportedRuleConditionOperators, operator)

	switch trigger {
	case ruleTriggerInbound:
		if !slices.Contains(inboundRuleFields, field) {
			return common.NewUserError(
				"inbound rules only support from.* conditions",
				fmt.Sprintf("Allowed fields: %s", strings.Join(inboundRuleFields, ", ")),
			)
		}
	case ruleTriggerOutbound:
		if !slices.Contains(outboundRuleFields, field) {
			return common.NewUserError(
				"outbound rule field is not supported",
				fmt.Sprintf("Allowed fields: %s", strings.Join(outboundRuleFields, ", ")),
			)
		}
	}

	if field == ruleFieldOutboundType {
		if !operatorKnown {
			return nil
		}
		if !slices.Contains(outboundTypeOperators, operator) {
			return common.NewUserError(
				"outbound.type only supports is and is_not",
				"Use --condition outbound.type,is,compose or outbound.type,is,reply",
			)
		}

		value, ok := scalarRuleValue(condition.Value)
		if !ok || !slices.Contains(supportedOutboundTypes, value) {
			return common.NewUserError(
				"unsupported outbound.type value",
				fmt.Sprintf("Use one of: %s", strings.Join(supportedOutboundTypes, ", ")),
			)
		}
		return nil
	}

	if !operatorKnown {
		return nil
	}

	if operator == ruleConditionOperatorInList {
		if _, ok := listRuleValue(condition.Value); !ok {
			return common.NewUserError(
				"in_list conditions require one or more list IDs",
				"Use --condition field,in_list,list-id or provide a JSON array with --data/--data-file",
			)
		}
		return nil
	}

	if _, ok := scalarRuleValue(condition.Value); !ok {
		return common.NewUserError(
			"rule condition value must be a string",
			"Use field,operator,value for scalar comparisons",
		)
	}

	return nil
}

func validateRuleActions(actions []domain.RuleAction) error {
	if len(actions) == 0 {
		return nil
	}

	blockSeen := false
	knownActionCount := 0
	for _, action := range actions {
		actionType := canonicalRuleActionType(action.Type)
		if !slices.Contains(supportedRuleActions, actionType) {
			// Preserve compatibility for legacy and future server-side actions.
			continue
		}
		knownActionCount++

		if actionType == ruleActionBlock {
			blockSeen = true
		}

		if actionType == ruleActionAssignToFolder {
			if _, ok := scalarRuleValue(action.Value); !ok {
				return common.NewUserError(
					"assign_to_folder requires a folder value",
					"Use --action assign_to_folder=<folder-id-or-name>",
				)
			}
			continue
		}

		if hasRuleValue(action.Value) {
			return common.NewUserError(
				"rule action does not accept a value",
				fmt.Sprintf("Remove the value from action %q", actionType),
			)
		}
	}

	if blockSeen && knownActionCount > 1 {
		return common.NewUserError(
			"block cannot be combined with other actions",
			"Use block by itself because it is terminal",
		)
	}

	return nil
}

func extractRuleConditions(value any) ([]domain.RuleCondition, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case []domain.RuleCondition:
		return v, nil
	case []any:
		conditions := make([]domain.RuleCondition, 0, len(v))
		for _, item := range v {
			condition, ok := toRuleCondition(item)
			if !ok {
				return nil, common.NewUserError(
					"invalid rule match payload",
					"match.conditions must be an array of {field, operator, value} objects",
				)
			}
			conditions = append(conditions, condition)
		}
		return conditions, nil
	default:
		return nil, common.NewUserError(
			"invalid rule match payload",
			"match.conditions must be an array of {field, operator, value} objects",
		)
	}
}

func extractRuleActions(value any) ([]domain.RuleAction, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case []domain.RuleAction:
		return v, nil
	case []any:
		actions := make([]domain.RuleAction, 0, len(v))
		for _, item := range v {
			action, ok := toRuleAction(item)
			if !ok {
				return nil, common.NewUserError(
					"invalid rule actions payload",
					"actions must be an array of {type, value?} objects",
				)
			}
			actions = append(actions, action)
		}
		return actions, nil
	default:
		return nil, common.NewUserError(
			"invalid rule actions payload",
			"actions must be an array of {type, value?} objects",
		)
	}
}

func toRuleCondition(value any) (domain.RuleCondition, bool) {
	switch v := value.(type) {
	case domain.RuleCondition:
		return v, true
	case map[string]any:
		return domain.RuleCondition{
			Field:    strings.TrimSpace(asString(v["field"])),
			Operator: strings.TrimSpace(asString(v["operator"])),
			Value:    normalizeRuleValue(v["value"]),
		}, true
	default:
		return domain.RuleCondition{}, false
	}
}

func toRuleAction(value any) (domain.RuleAction, bool) {
	switch v := value.(type) {
	case domain.RuleAction:
		return v, true
	case map[string]any:
		return domain.RuleAction{
			Type:  strings.TrimSpace(asString(v["type"])),
			Value: normalizeRuleValue(v["value"]),
		}, true
	default:
		return domain.RuleAction{}, false
	}
}

func normalizeRuleValue(value any) any {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			text, ok := item.(string)
			if !ok {
				return value
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			values = append(values, text)
		}
		return values
	default:
		return value
	}
}

func scalarRuleValue(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	return text, true
}

func listRuleValue(value any) ([]string, bool) {
	switch v := value.(type) {
	case []string:
		values := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			values = append(values, item)
		}
		return values, len(values) > 0
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			values = append(values, text)
		}
		return values, len(values) > 0
	default:
		return nil, false
	}
}

func hasRuleValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(v) != ""
	case []string:
		return len(v) > 0
	case []any:
		return len(v) > 0
	default:
		return true
	}
}
