package webhook

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPubSubCommand(t *testing.T) {
	cmd := newPubSubCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "pubsub", cmd.Use)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expected := []string{"list", "show", "create", "update", "delete"}
		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}
		for _, name := range expected {
			assert.True(t, cmdMap[name], "Missing expected subcommand: %s", name)
		}
	})
}

func TestPubSubCreateCommand(t *testing.T) {
	cmd := newPubSubCreateCmd()

	assert.Equal(t, "create", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("topic"))
	assert.NotNil(t, cmd.Flags().Lookup("triggers"))
	assert.NotNil(t, cmd.Flags().Lookup("notify"))
	assert.NotNil(t, cmd.Flags().Lookup("encryption-key"))
	assert.Contains(t, cmd.Flags().Lookup("topic").Annotations, cobra.BashCompOneRequiredFlag)
	assert.Contains(t, cmd.Flags().Lookup("triggers").Annotations, cobra.BashCompOneRequiredFlag)
}

func TestPubSubUpdateCommand(t *testing.T) {
	cmd := newPubSubUpdateCmd()

	assert.Equal(t, "update <channel-id>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("topic"))
	assert.NotNil(t, cmd.Flags().Lookup("triggers"))
	assert.NotNil(t, cmd.Flags().Lookup("status"))
}

func TestValidatePubSubTopic(t *testing.T) {
	assert.NoError(t, validatePubSubTopic("projects/demo/topics/events"))
	assert.Error(t, validatePubSubTopic("demo/topics/events"))
	assert.Error(t, validatePubSubTopic(""))
	assert.Error(t, validatePubSubTopic("projects/demo/subscriptions/events"))
}

func TestParseAndValidateTriggers(t *testing.T) {
	triggers, err := parseAndValidateTriggers([]string{"message.created, message.updated", "thread.replied"})
	require.NoError(t, err)
	assert.Equal(t, []string{"message.created", "message.updated", "thread.replied"}, triggers)

	_, err = parseAndValidateTriggers([]string{"  ", ","})
	require.Error(t, err)

	_, err = parseAndValidateTriggers([]string{"message.created", "totally.invalid"})
	require.Error(t, err)
}

func TestPubSubCreateCommand_Validation(t *testing.T) {
	_, _, err := executeCommand(newPubSubCreateCmd(), "--topic", "projects/demo/topics/events")
	require.Error(t, err)

	_, _, err = executeCommand(newPubSubCreateCmd(), "--triggers", "message.created")
	require.Error(t, err)
}

func TestPubSubUpdateCommand_Validation(t *testing.T) {
	_, _, err := executeCommand(newPubSubUpdateCmd(), "pubsub-1")
	require.Error(t, err)

	_, _, err = executeCommand(newPubSubUpdateCmd(), "pubsub-1", "--status", "paused")
	require.Error(t, err)
}

func TestPubSubDeleteCommand_RequiresYes(t *testing.T) {
	_, _, err := executeCommand(newPubSubDeleteCmd(), "pubsub-1")
	require.Error(t, err)
}

func TestPubSubCreateCommand_RejectsMalformedTopic(t *testing.T) {
	_, _, err := executeCommand(newPubSubCreateCmd(),
		"--topic", "projects/demo/subscriptions/events",
		"--triggers", "message.created",
	)
	require.Error(t, err)
}
