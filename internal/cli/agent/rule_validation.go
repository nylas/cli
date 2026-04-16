package agent

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

const (
	ruleTriggerInbound  = "inbound"
	ruleTriggerOutbound = "outbound"

	ruleMatchOperatorAll = "all"
	ruleMatchOperatorAny = "any"

	ruleConditionOperatorIs       = "is"
	ruleConditionOperatorIsNot    = "is_not"
	ruleConditionOperatorContains = "contains"
	ruleConditionOperatorInList   = "in_list"

	ruleConditionFieldFromAddress      = "from.address"
	ruleConditionFieldFromDomain       = "from.domain"
	ruleConditionFieldFromTLD          = "from.tld"
	ruleConditionFieldRecipientAddress = "recipient.address"
	ruleConditionFieldRecipientDomain  = "recipient.domain"
	ruleConditionFieldRecipientTLD     = "recipient.tld"
	ruleConditionFieldOutboundType     = "outbound.type"

	ruleActionBlock          = "block"
	ruleActionMarkAsSpam     = "mark_as_spam"
	ruleActionAssignToFolder = "assign_to_folder"
	ruleActionMarkAsRead     = "mark_as_read"
	ruleActionMarkAsStarred  = "mark_as_starred"
	ruleActionArchive        = "archive"
	ruleActionTrash          = "trash"

	ruleOutboundTypeCompose = "compose"
	ruleOutboundTypeReply   = "reply"

	maxRuleConditions    = 50
	maxRuleActions       = 20
	maxRuleListsPerMatch = 10
	maxRuleValueLength   = 500
)

type normalizedRuleMatch struct {
	Operator   string
	Conditions []normalizedRuleCondition
}

type normalizedRuleCondition struct {
	Field    string
	Operator string
	Value    any
}

type normalizedRuleAction struct {
	Type  string
	Value any
}

func validateRulePayload(payload map[string]any, existingRule *domain.Rule, requireCreateFields bool) error {
	trigger, payloadHasTrigger, err := resolveRuleTrigger(payload, existingRule)
	if err != nil {
		return err
	}

	if requireCreateFields && strings.TrimSpace(asString(payload["name"])) == "" {
		return common.NewUserError("rule name is required", "Use --name or include a non-empty name in --data/--data-file")
	}

	if _, ok := payload["priority"]; ok {
		priority, err := parseRulePriority(payload["priority"])
		if err != nil {
			return err
		}
		if priority < 0 || priority > 1000 {
			return common.NewUserError("rule priority must be between 0 and 1000", "Use --priority with a value from 0 to 1000")
		}
	}

	shouldValidateMatch := requireCreateFields
	if _, ok := payload["match"]; ok || payloadHasTrigger {
		shouldValidateMatch = true
	}
	if shouldValidateMatch {
		match, err := resolveRuleMatch(payload, existingRule)
		if err != nil {
			return err
		}
		if err := validateRuleMatch(match, trigger); err != nil {
			return err
		}
	}

	shouldValidateActions := requireCreateFields
	if _, ok := payload["actions"]; ok {
		shouldValidateActions = true
	}
	if shouldValidateActions {
		actions, err := resolveRuleActions(payload, existingRule)
		if err != nil {
			return err
		}
		if err := validateRuleActions(actions); err != nil {
			return err
		}
	}

	return nil
}

func resolveRuleTrigger(payload map[string]any, existingRule *domain.Rule) (string, bool, error) {
	if rawTrigger, ok := payload["trigger"]; ok {
		trigger := strings.TrimSpace(asString(rawTrigger))
		switch trigger {
		case ruleTriggerInbound, ruleTriggerOutbound:
			return trigger, true, nil
		default:
			return "", true, common.NewUserError("invalid rule trigger", "Use --trigger inbound or --trigger outbound")
		}
	}

	if existingRule != nil {
		trigger := strings.TrimSpace(existingRule.Trigger)
		if trigger != "" {
			return trigger, false, nil
		}
	}

	return ruleTriggerInbound, false, nil
}

func parseRulePriority(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		if float32(int(v)) != v {
			return 0, common.NewUserError("invalid rule priority", "Use an integer between 0 and 1000")
		}
		return int(v), nil
	case float64:
		if float64(int(v)) != v {
			return 0, common.NewUserError("invalid rule priority", "Use an integer between 0 and 1000")
		}
		return int(v), nil
	default:
		return 0, common.NewUserError("invalid rule priority", "Use an integer between 0 and 1000")
	}
}

func resolveRuleMatch(payload map[string]any, existingRule *domain.Rule) (*normalizedRuleMatch, error) {
	if rawMatch, ok := payload["match"]; ok && rawMatch != nil {
		return normalizeRuleMatch(rawMatch)
	}

	if existingRule != nil && existingRule.Match != nil {
		return normalizeRuleMatch(existingRule.Match)
	}

	return nil, nil
}

func normalizeRuleMatch(value any) (*normalizedRuleMatch, error) {
	switch v := value.(type) {
	case map[string]any:
		conditions, err := normalizeRuleConditions(v["conditions"])
		if err != nil {
			return nil, err
		}
		return &normalizedRuleMatch{
			Operator:   strings.TrimSpace(asString(v["operator"])),
			Conditions: conditions,
		}, nil
	case *domain.RuleMatch:
		if v == nil {
			return nil, nil
		}
		return &normalizedRuleMatch{
			Operator:   strings.TrimSpace(v.Operator),
			Conditions: normalizeDomainRuleConditions(v.Conditions),
		}, nil
	case domain.RuleMatch:
		return &normalizedRuleMatch{
			Operator:   strings.TrimSpace(v.Operator),
			Conditions: normalizeDomainRuleConditions(v.Conditions),
		}, nil
	default:
		return nil, common.NewUserError("invalid rule match payload", "Provide match as an object with operator and conditions")
	}
}

func normalizeDomainRuleConditions(conditions []domain.RuleCondition) []normalizedRuleCondition {
	normalized := make([]normalizedRuleCondition, 0, len(conditions))
	for _, condition := range conditions {
		normalized = append(normalized, normalizedRuleCondition{
			Field:    strings.TrimSpace(condition.Field),
			Operator: strings.TrimSpace(condition.Operator),
			Value:    condition.Value,
		})
	}
	return normalized
}

func normalizeRuleConditions(value any) ([]normalizedRuleCondition, error) {
	if value == nil {
		return nil, nil
	}

	switch conditions := value.(type) {
	case []domain.RuleCondition:
		return normalizeDomainRuleConditions(conditions), nil
	case []any:
		return normalizeRuleConditionsFromAny(conditions)
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return nil, common.NewUserError("invalid rule conditions payload", "Provide conditions as an array")
		}

		items := make([]any, 0, rv.Len())
		for i := range rv.Len() {
			items = append(items, rv.Index(i).Interface())
		}
		return normalizeRuleConditionsFromAny(items)
	}
}

func normalizeRuleConditionsFromAny(items []any) ([]normalizedRuleCondition, error) {
	conditions := make([]normalizedRuleCondition, 0, len(items))
	for _, item := range items {
		switch condition := item.(type) {
		case domain.RuleCondition:
			conditions = append(conditions, normalizedRuleCondition{
				Field:    strings.TrimSpace(condition.Field),
				Operator: strings.TrimSpace(condition.Operator),
				Value:    condition.Value,
			})
		case map[string]any:
			conditions = append(conditions, normalizedRuleCondition{
				Field:    strings.TrimSpace(asString(condition["field"])),
				Operator: strings.TrimSpace(asString(condition["operator"])),
				Value:    condition["value"],
			})
		default:
			return nil, common.NewUserError("invalid rule condition payload", "Provide each condition as an object with field, operator, and value")
		}
	}
	return conditions, nil
}

func resolveRuleActions(payload map[string]any, existingRule *domain.Rule) ([]normalizedRuleAction, error) {
	if rawActions, ok := payload["actions"]; ok && rawActions != nil {
		return normalizeRuleActions(rawActions)
	}

	if existingRule != nil {
		return normalizeRuleActions(existingRule.Actions)
	}

	return nil, nil
}

func normalizeRuleActions(value any) ([]normalizedRuleAction, error) {
	if value == nil {
		return nil, nil
	}

	switch actions := value.(type) {
	case []domain.RuleAction:
		normalized := make([]normalizedRuleAction, 0, len(actions))
		for _, action := range actions {
			normalized = append(normalized, normalizedRuleAction{
				Type:  strings.TrimSpace(action.Type),
				Value: action.Value,
			})
		}
		return normalized, nil
	case []any:
		return normalizeRuleActionsFromAny(actions)
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return nil, common.NewUserError("invalid rule actions payload", "Provide actions as an array")
		}

		items := make([]any, 0, rv.Len())
		for i := range rv.Len() {
			items = append(items, rv.Index(i).Interface())
		}
		return normalizeRuleActionsFromAny(items)
	}
}

func normalizeRuleActionsFromAny(items []any) ([]normalizedRuleAction, error) {
	actions := make([]normalizedRuleAction, 0, len(items))
	for _, item := range items {
		switch action := item.(type) {
		case domain.RuleAction:
			actions = append(actions, normalizedRuleAction{
				Type:  strings.TrimSpace(action.Type),
				Value: action.Value,
			})
		case map[string]any:
			actions = append(actions, normalizedRuleAction{
				Type:  strings.TrimSpace(asString(action["type"])),
				Value: action["value"],
			})
		default:
			return nil, common.NewUserError("invalid rule action payload", "Provide each action as an object with type and optional value")
		}
	}
	return actions, nil
}

func validateRuleMatch(match *normalizedRuleMatch, trigger string) error {
	if match == nil {
		return common.NewUserError("rule match is required", "Add at least one condition or provide a full rule body with match conditions")
	}

	if match.Operator != "" {
		switch match.Operator {
		case ruleMatchOperatorAll, ruleMatchOperatorAny:
		default:
			return common.NewUserError("invalid rule match operator", "Use --match-operator all or --match-operator any")
		}
	}

	if len(match.Conditions) == 0 {
		return common.NewUserError("at least one rule condition is required", "Add one or more --condition entries, or provide conditions in --data/--data-file")
	}
	if len(match.Conditions) > maxRuleConditions {
		return common.NewUserError(
			fmt.Sprintf("rule cannot have more than %d conditions", maxRuleConditions),
			"Reduce the number of conditions in the rule body",
		)
	}

	for i, condition := range match.Conditions {
		if err := validateRuleCondition(condition, trigger, i+1); err != nil {
			return err
		}
	}

	return nil
}

func validateRuleCondition(condition normalizedRuleCondition, trigger string, index int) error {
	if condition.Field == "" {
		return common.NewUserError(
			fmt.Sprintf("condition %d is missing a field", index),
			"Provide field,operator,value or use JSON with field/operator/value keys",
		)
	}
	if !isValidRuleConditionField(condition.Field) {
		return common.NewUserError(
			fmt.Sprintf("condition %d uses an unsupported field", index),
			"Use from.address, from.domain, from.tld, recipient.address, recipient.domain, recipient.tld, or outbound.type",
		)
	}

	if trigger != ruleTriggerOutbound {
		switch condition.Field {
		case ruleConditionFieldRecipientAddress, ruleConditionFieldRecipientDomain, ruleConditionFieldRecipientTLD:
			return common.NewUserError(
				fmt.Sprintf("condition %d uses recipient.* on an inbound rule", index),
				"Set --trigger outbound or use only from.* fields",
			)
		case ruleConditionFieldOutboundType:
			return common.NewUserError(
				fmt.Sprintf("condition %d uses outbound.type on an inbound rule", index),
				"Set --trigger outbound when matching outbound.type",
			)
		}
	}

	if condition.Operator == "" {
		return common.NewUserError(
			fmt.Sprintf("condition %d is missing an operator", index),
			"Use is, is_not, contains, or in_list",
		)
	}
	if !isValidRuleConditionOperator(condition.Operator) {
		return common.NewUserError(
			fmt.Sprintf("condition %d uses an unsupported operator", index),
			"Use is, is_not, contains, or in_list",
		)
	}

	if condition.Field == ruleConditionFieldOutboundType &&
		condition.Operator != ruleConditionOperatorIs &&
		condition.Operator != ruleConditionOperatorIsNot {
		return common.NewUserError(
			"outbound.type only supports is and is_not operators",
			"Use --condition outbound.type,is,compose or --condition outbound.type,is_not,reply",
		)
	}

	if condition.Operator == ruleConditionOperatorInList {
		listCount, ok := listValueLen(condition.Value)
		if !ok {
			return common.NewUserError(
				fmt.Sprintf("condition %d requires an array value for in_list", index),
				"Use --data/--data-file to pass a JSON array of list IDs for in_list conditions",
			)
		}
		if listCount == 0 {
			return common.NewUserError(
				fmt.Sprintf("condition %d requires at least one list ID", index),
				"Provide one or more list IDs in the in_list array",
			)
		}
		if listCount > maxRuleListsPerMatch {
			return common.NewUserError(
				fmt.Sprintf("condition %d cannot reference more than %d lists", index, maxRuleListsPerMatch),
				"Reduce the number of list IDs in the in_list array",
			)
		}
		return nil
	}

	value, ok := condition.Value.(string)
	if !ok {
		return common.NewUserError(
			fmt.Sprintf("condition %d value must be a string", index),
			"Use a string value, or use --data/--data-file for structured values",
		)
	}
	if strings.TrimSpace(value) == "" {
		return common.NewUserError(
			fmt.Sprintf("condition %d value cannot be empty", index),
			"Provide a non-empty condition value",
		)
	}
	if len(value) > maxRuleValueLength {
		return common.NewUserError(
			fmt.Sprintf("condition %d value is too long", index),
			fmt.Sprintf("Keep condition values at %d characters or fewer", maxRuleValueLength),
		)
	}

	if condition.Field == ruleConditionFieldOutboundType {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case ruleOutboundTypeCompose, ruleOutboundTypeReply:
		default:
			return common.NewUserError("outbound.type must be compose or reply", "Use outbound.type with value compose or reply")
		}
	}

	return nil
}

func validateRuleActions(actions []normalizedRuleAction) error {
	if len(actions) == 0 {
		return common.NewUserError("rule actions are required", "Add one or more --action entries, or provide actions in --data/--data-file")
	}
	if len(actions) > maxRuleActions {
		return common.NewUserError(
			fmt.Sprintf("rule cannot have more than %d actions", maxRuleActions),
			"Reduce the number of actions in the rule body",
		)
	}

	hasBlock := false
	for i, action := range actions {
		if action.Type == "" {
			return common.NewUserError(
				fmt.Sprintf("action %d is missing a type", i+1),
				"Provide an action type such as block, mark_as_spam, or assign_to_folder",
			)
		}
		if !isValidRuleActionType(action.Type) {
			return common.NewUserError(
				fmt.Sprintf("action %d uses an unsupported type", i+1),
				"Use block, mark_as_spam, assign_to_folder, mark_as_read, mark_as_starred, archive, or trash",
			)
		}
		if action.Type == ruleActionAssignToFolder && isEmptyRuleValue(action.Value) {
			return common.NewUserError("assign_to_folder requires a value", "Use --action assign_to_folder=<folder-id>")
		}
		if action.Type == ruleActionBlock {
			hasBlock = true
		}
	}

	if hasBlock && len(actions) > 1 {
		return common.NewUserError("block cannot be combined with other actions", "Use block by itself or remove the other actions")
	}

	return nil
}

func isValidRuleConditionField(field string) bool {
	switch field {
	case ruleConditionFieldFromAddress,
		ruleConditionFieldFromDomain,
		ruleConditionFieldFromTLD,
		ruleConditionFieldRecipientAddress,
		ruleConditionFieldRecipientDomain,
		ruleConditionFieldRecipientTLD,
		ruleConditionFieldOutboundType:
		return true
	default:
		return false
	}
}

func isValidRuleConditionOperator(operator string) bool {
	switch operator {
	case ruleConditionOperatorIs,
		ruleConditionOperatorIsNot,
		ruleConditionOperatorContains,
		ruleConditionOperatorInList:
		return true
	default:
		return false
	}
}

func isValidRuleActionType(actionType string) bool {
	switch actionType {
	case ruleActionBlock,
		ruleActionMarkAsSpam,
		ruleActionAssignToFolder,
		ruleActionMarkAsRead,
		ruleActionMarkAsStarred,
		ruleActionArchive,
		ruleActionTrash:
		return true
	default:
		return false
	}
}

func listValueLen(value any) (int, bool) {
	switch v := value.(type) {
	case []string:
		return len(v), true
	case []any:
		return len(v), true
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return 0, false
		}
		return rv.Len(), true
	}
}

func isEmptyRuleValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}
