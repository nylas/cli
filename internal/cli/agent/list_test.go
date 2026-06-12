package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentListCmd(t *testing.T) {
	cmd := newAgentListCmd()

	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Aliases, "lists")
	assert.Contains(t, cmd.Short, "lists")
	assert.Contains(t, cmd.Long, "/v3/lists")

	expected := []string{"list", "get", "create", "update", "delete", "items", "add", "remove"}
	cmdMap := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		cmdMap[sub.Name()] = true
	}
	for _, name := range expected {
		assert.True(t, cmdMap[name], "missing subcommand %s", name)
	}
}

func TestBuildListCreatePayload(t *testing.T) {
	payload, err := buildListCreatePayload("Blocked domains", "domain", "bad senders")
	assert.NoError(t, err)
	assert.Equal(t, "Blocked domains", payload["name"])
	assert.Equal(t, "domain", payload["type"])
	assert.Equal(t, "bad senders", payload["description"])

	// description is optional and omitted when empty
	payload, err = buildListCreatePayload("Blocked domains", "tld", "")
	assert.NoError(t, err)
	_, hasDescription := payload["description"]
	assert.False(t, hasDescription)

	// name is required
	_, err = buildListCreatePayload("", "domain", "")
	assert.ErrorContains(t, err, "name is required")

	// type must be one of domain, tld, address — it is immutable after
	// creation and determines which rule fields the list can match
	_, err = buildListCreatePayload("Blocked", "country", "")
	assert.ErrorContains(t, err, "invalid list type")
}

func TestAgentListCreateCmd(t *testing.T) {
	cmd := newAgentListCreateCmd()

	assert.Equal(t, "create", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("type"))
	assert.NotNil(t, cmd.Flags().Lookup("description"))
	assert.NotNil(t, cmd.Flags().Lookup("item"))
}

func TestAgentListDeleteCmd_RequiresYes(t *testing.T) {
	cmd := newAgentListDeleteCmd()
	cmd.SetArgs([]string{"list-123"})

	err := cmd.Execute()
	assert.ErrorContains(t, err, "confirmation")
}
