package agent

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentCmd(t *testing.T) {
	cmd := NewAgentCmd()

	assert.Equal(t, "agent", cmd.Use)
	assert.Contains(t, cmd.Aliases, "agents")
	assert.Contains(t, cmd.Short, "agent")
	assert.Contains(t, cmd.Long, "provider=nylas")

	expected := []string{"create", "list", "delete", "status"}
	cmdMap := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		cmdMap[sub.Name()] = true
	}
	for _, name := range expected {
		assert.True(t, cmdMap[name], "missing subcommand %s", name)
	}
}

func TestCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	assert.Equal(t, "create <email>", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("json"))
	assert.NotNil(t, cmd.Flags().Lookup("app-password"))
	assert.Contains(t, cmd.Long, "provider=nylas")
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
