package agent

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

type rulePayloadOptions struct {
	Name          string
	Description   string
	Priority      int
	PrioritySet   bool
	EnabledSet    bool
	DisabledSet   bool
	Trigger       string
	MatchOperator string
	Conditions    []string
	Actions       []string
}

func (o rulePayloadOptions) hasFlagInput() bool {
	return strings.TrimSpace(o.Name) != "" ||
		strings.TrimSpace(o.Description) != "" ||
		o.PrioritySet ||
		o.EnabledSet ||
		o.DisabledSet ||
		strings.TrimSpace(o.Trigger) != "" ||
		strings.TrimSpace(o.MatchOperator) != "" ||
		len(o.Conditions) > 0 ||
		len(o.Actions) > 0
}

func loadRulePayload(data, dataFile string, opts rulePayloadOptions, requireBody bool) (map[string]any, error) {
	payload, err := common.ReadJSONStringMap(data, dataFile)
	if err != nil {
		return nil, err
	}

	if opts.EnabledSet && opts.DisabledSet {
		return nil, common.NewUserError(
			"cannot combine --enabled with --disabled",
			"Use only one of the two flags",
		)
	}

	usingJSON := strings.TrimSpace(data) != "" || strings.TrimSpace(dataFile) != ""
	if requireBody && !usingJSON && !opts.hasFlagInput() {
		return nil, common.NewUserError(
			"rule create requires a rule definition",
			"Use --condition/--action for the common case or --data/--data-file for full JSON",
		)
	}

	if strings.TrimSpace(opts.Name) != "" {
		payload["name"] = strings.TrimSpace(opts.Name)
	}
	if strings.TrimSpace(opts.Description) != "" {
		payload["description"] = strings.TrimSpace(opts.Description)
	}
	if opts.PrioritySet {
		payload["priority"] = opts.Priority
	}
	if opts.EnabledSet {
		payload["enabled"] = true
	}
	if opts.DisabledSet {
		payload["enabled"] = false
	}
	if strings.TrimSpace(opts.Trigger) != "" {
		payload["trigger"] = strings.TrimSpace(opts.Trigger)
	}

	if strings.TrimSpace(opts.MatchOperator) != "" || len(opts.Conditions) > 0 {
		matchPayload, err := mergeRuleMatchPayload(payload["match"], opts)
		if err != nil {
			return nil, err
		}
		payload["match"] = matchPayload
	}

	if len(opts.Actions) > 0 {
		actions, err := parseRuleActions(opts.Actions)
		if err != nil {
			return nil, err
		}
		payload["actions"] = actions
	}

	if requireBody && !usingJSON {
		if _, ok := payload["enabled"]; !ok {
			payload["enabled"] = true
		}
		if strings.TrimSpace(asString(payload["trigger"])) == "" {
			payload["trigger"] = "inbound"
		}
		if err := applyDefaultMatchOperator(payload); err != nil {
			return nil, err
		}
		if err := validateFlagBuiltRuleCreatePayload(payload); err != nil {
			return nil, err
		}
	}

	if requireBody {
		if err := validateRulePayload(payload, nil, true); err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func mergeRuleMatchPayload(existing any, opts rulePayloadOptions) (map[string]any, error) {
	matchPayload := copyStringAnyMap(existing)

	if strings.TrimSpace(opts.MatchOperator) != "" {
		matchPayload["operator"] = strings.TrimSpace(opts.MatchOperator)
	}
	if len(opts.Conditions) > 0 {
		conditions, err := parseRuleConditions(opts.Conditions)
		if err != nil {
			return nil, err
		}
		matchPayload["conditions"] = conditions
	}

	return matchPayload, nil
}

func parseRuleConditions(rawConditions []string) ([]domain.RuleCondition, error) {
	conditions := make([]domain.RuleCondition, 0, len(rawConditions))
	for _, raw := range rawConditions {
		field, remainder, ok := strings.Cut(raw, ",")
		if !ok {
			return nil, invalidRuleConditionError(raw)
		}

		operator, value, ok := strings.Cut(remainder, ",")
		if !ok {
			return nil, invalidRuleConditionError(raw)
		}

		field = strings.TrimSpace(field)
		operator = strings.TrimSpace(operator)
		value = strings.TrimSpace(value)
		if field == "" || operator == "" || value == "" {
			return nil, invalidRuleConditionError(raw)
		}

		conditions = append(conditions, domain.RuleCondition{
			Field:    field,
			Operator: operator,
			Value:    parseRuleValue(value),
		})
	}

	return conditions, nil
}

func parseRuleActions(rawActions []string) ([]domain.RuleAction, error) {
	actions := make([]domain.RuleAction, 0, len(rawActions))
	for _, raw := range rawActions {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, invalidRuleActionError(raw)
		}

		actionType, actionValue, hasValue := strings.Cut(raw, "=")
		actionType = strings.TrimSpace(actionType)
		actionValue = strings.TrimSpace(actionValue)
		if actionType == "" {
			return nil, invalidRuleActionError(raw)
		}

		action := domain.RuleAction{Type: actionType}
		if hasValue && actionValue != "" {
			action.Value = parseRuleValue(actionValue)
		}

		actions = append(actions, action)
	}

	return actions, nil
}

func parseRuleValue(raw string) any {
	return raw
}

func preserveRuleMatchOperator(payload map[string]any, existingRule *domain.Rule) {
	if existingRule == nil || existingRule.Match == nil {
		return
	}

	matchPayload := copyStringAnyMap(payload["match"])
	if len(matchPayload) == 0 {
		return
	}
	if strings.TrimSpace(asString(matchPayload["operator"])) != "" {
		return
	}
	if sliceLen(matchPayload["conditions"]) == 0 {
		return
	}

	existingOperator := strings.TrimSpace(existingRule.Match.Operator)
	if existingOperator == "" {
		return
	}

	matchPayload["operator"] = existingOperator
	payload["match"] = matchPayload
}

func applyDefaultMatchOperator(payload map[string]any) error {
	matchPayload := copyStringAnyMap(payload["match"])
	if len(matchPayload) == 0 {
		return nil
	}
	if strings.TrimSpace(asString(matchPayload["operator"])) != "" {
		return nil
	}
	if sliceLen(matchPayload["conditions"]) == 0 {
		return nil
	}
	matchPayload["operator"] = "all"
	payload["match"] = matchPayload
	return nil
}

func validateFlagBuiltRuleCreatePayload(payload map[string]any) error {
	missing := make([]string, 0, 3)
	if strings.TrimSpace(asString(payload["name"])) == "" {
		missing = append(missing, "--name")
	}

	matchPayload := copyStringAnyMap(payload["match"])
	if sliceLen(matchPayload["conditions"]) == 0 {
		missing = append(missing, "--condition")
	}
	if sliceLen(payload["actions"]) == 0 {
		missing = append(missing, "--action")
	}

	if len(missing) == 0 {
		return nil
	}

	return common.NewUserError(
		"rule create is missing required fields",
		fmt.Sprintf("Use %s, or provide a full rule body with --data/--data-file", strings.Join(missing, ", ")),
	)
}

func copyStringAnyMap(value any) map[string]any {
	existing, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(existing))
	for key, entry := range existing {
		cloned[key] = entry
	}
	return cloned
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func sliceLen(value any) int {
	if value == nil {
		return 0
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return 0
	}
	return rv.Len()
}

func invalidRuleConditionError(raw string) error {
	return common.NewUserError(
		"invalid --condition value",
		fmt.Sprintf("Use field,operator,value. Got %q", raw),
	)
}

func invalidRuleActionError(raw string) error {
	return common.NewUserError(
		"invalid --action value",
		fmt.Sprintf("Use type or type=value. Got %q", raw),
	)
}
