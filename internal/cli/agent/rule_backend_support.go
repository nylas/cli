package agent

import (
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func wrapRuleMutationError(operation string, payload map[string]any, existingRule *domain.Rule, err error) error {
	if outboundRuleUnsupportedError(payload, existingRule, err) {
		return &common.CLIError{
			Err:        err,
			Message:    "outbound rules are not enabled on this API environment",
			Suggestion: "Retry on an environment with outbound rule support, or remove --trigger outbound",
		}
	}

	switch operation {
	case "create":
		return common.WrapCreateError("rule", err)
	case "update":
		return common.WrapUpdateError("rule", err)
	default:
		return err
	}
}

func outboundRuleUnsupportedError(payload map[string]any, existingRule *domain.Rule, err error) bool {
	if err == nil {
		return false
	}

	trigger, _, resolveErr := resolveRuleTrigger(payload, existingRule)
	if resolveErr != nil || trigger != ruleTriggerOutbound {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "invalid rule trigger") &&
		strings.Contains(message, "must be 'inbound'")
}
