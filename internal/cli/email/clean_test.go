package email

import (
	"fmt"
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestCleanCommand_Structure(t *testing.T) {
	cmd := newCleanCmd()

	assert.Equal(t, "clean <message-id> [message-id...]", cmd.Use)
	for _, flag := range []string{"grant", "keep-links", "keep-images", "keep-tables", "images-as-markdown", "keep-signatures"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "missing --%s flag", flag)
	}
}

// The CLI must reject more than the API maximum before making a request, so the
// user gets a clear local error instead of an opaque 4xx.
func TestCleanCommand_RejectsTooManyIDs(t *testing.T) {
	cmd := newCleanCmd()

	args := make([]string, domain.CleanMessagesMaxIDs+1)
	for i := range args {
		args[i] = fmt.Sprintf("msg-%d", i)
	}

	_, _, err := testutil.ExecuteCommand(cmd, args...)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at most")
}

func TestEmailCmd_RegistersClean(t *testing.T) {
	cmd := NewEmailCmd()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["clean"], "email command must register the clean subcommand")
}
