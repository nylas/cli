package agent

import (
	"errors"
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapRuleMutationError_CreateOutboundUnsupported(t *testing.T) {
	err := wrapRuleMutationError(
		"create",
		map[string]any{"trigger": "outbound"},
		nil,
		errors.New("nylas API error: Invalid rule trigger. Must be 'inbound'"),
	)

	var cliErr *common.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, "outbound rules are not enabled on this API environment", cliErr.Message)
	assert.Equal(t, "Retry on an environment with outbound rule support, or remove --trigger outbound", cliErr.Suggestion)
}

func TestWrapRuleMutationError_UpdateUsesExistingOutboundTrigger(t *testing.T) {
	err := wrapRuleMutationError(
		"update",
		map[string]any{"name": "updated"},
		&domain.Rule{Trigger: "outbound"},
		errors.New("nylas API error: Invalid rule trigger. Must be 'inbound'"),
	)

	var cliErr *common.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, "outbound rules are not enabled on this API environment", cliErr.Message)
}

func TestWrapRuleMutationError_LeavesInboundErrorsWrapped(t *testing.T) {
	err := wrapRuleMutationError(
		"create",
		map[string]any{"trigger": "inbound"},
		nil,
		errors.New("boom"),
	)

	assert.EqualError(t, err, "failed to create rule: boom")
}
