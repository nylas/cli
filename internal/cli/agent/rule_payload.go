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

type loadedRulePayload struct {
	Payload   map[string]any
	UsingJSON bool
	PureJSON  bool
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
	loaded, err := loadRulePayloadDetails(data, dataFile, opts, requireBody)
	if err != nil {
		return nil, err
	}
	return loaded.Payload, nil
}

func loadRulePayloadDetails(data, dataFile string, opts rulePayloadOptions, requireBody bool) (loadedRulePayload, error) {
	payload, err := common.ReadJSONStringMap(data, dataFile)
	if err != nil {
		return loadedRulePayload{}, err
	}

	if opts.EnabledSet && opts.DisabledSet {
		return loadedRulePayload{}, common.NewUserError(
			"cannot combine --enabled with --disabled",
			"Use only one of the two flags",
		)
	}

	usingJSON := strings.TrimSpace(data) != "" || strings.TrimSpace(dataFile) != ""
	pureJSON := usingJSON && !opts.hasFlagInput()
	if requireBody && !usingJSON && !opts.hasFlagInput() {
		return loadedRulePayload{}, common.NewUserError(
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
			return loadedRulePayload{}, err
		}
		payload["match"] = matchPayload
	}

	if len(opts.Actions) > 0 {
		actions, err := parseRuleActions(opts.Actions)
		if err != nil {
			return loadedRulePayload{}, err
		}
		payload["actions"] = actions
	}

	if requireBody && !pureJSON {
		if _, ok := payload["enabled"]; !ok {
			payload["enabled"] = true
		}
		if strings.TrimSpace(asString(payload["trigger"])) == "" {
			payload["trigger"] = ruleTriggerInbound
		}
		if err := applyDefaultMatchOperator(payload); err != nil {
			return loadedRulePayload{}, err
		}
		if err := validateRuleCreatePayload(payload); err != nil {
			return loadedRulePayload{}, err
		}
	}

	if !pureJSON {
		if err := validateRulePayload(payload, nil); err != nil {
			return loadedRulePayload{}, err
		}
	}

	return loadedRulePayload{Payload: payload, UsingJSON: usingJSON, PureJSON: pureJSON}, nil
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

		parsedValue, ok := parseRuleConditionValue(operator, value)
		if !ok {
			return nil, invalidRuleConditionError(raw)
		}

		conditions = append(conditions, domain.RuleCondition{
			Field:    field,
			Operator: operator,
			Value:    parsedValue,
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
		actionType = canonicalRuleActionType(actionType)

		action := domain.RuleAction{Type: actionType}
		if hasValue && actionValue != "" {
			action.Value = parseRuleValue(actionValue)
		}

		actions = append(actions, action)
	}

	return actions, nil
}

func parseRuleConditionValue(operator, raw string) (any, bool) {
	if strings.TrimSpace(operator) != ruleConditionOperatorInList {
		return raw, true
	}

	items := strings.Split(raw, ",")
	values := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		values = append(values, item)
	}
	if len(values) == 0 {
		return nil, false
	}

	return values, true
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

func validateRuleCreatePayload(payload map[string]any) error {
	missing := make([]string, 0, 3)
	if strings.TrimSpace(asString(payload["name"])) == "" {
		missing = append(missing, "name")
	}

	matchPayload, err := createRuleMatchPayload(payload["match"])
	if err != nil {
		return err
	}

	conditions, err := extractRuleConditions(matchPayload["conditions"])
	if err != nil {
		return err
	}
	if len(conditions) == 0 {
		missing = append(missing, "match.conditions")
	}

	actions, err := extractRuleActions(payload["actions"])
	if err != nil {
		return err
	}
	if len(actions) == 0 {
		missing = append(missing, "actions")
	}

	if len(missing) == 0 {
		return nil
	}

	return common.NewUserError(
		"rule create is missing required fields",
		fmt.Sprintf("Provide %s, either with flags or in --data/--data-file JSON", strings.Join(missing, ", ")),
	)
}

func createRuleMatchPayload(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}

	matchPayload, ok := value.(map[string]any)
	if !ok {
		return nil, common.NewUserError(
			"invalid rule match payload",
			"match must be an object with operator and conditions fields",
		)
	}

	return copyStringAnyMap(matchPayload), nil
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
