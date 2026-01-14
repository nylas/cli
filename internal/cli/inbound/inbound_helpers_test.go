package inbound

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// HELPER FUNCTION TESTS
// =============================================================================

func TestGetInboxID(t *testing.T) {
	t.Run("returns_first_arg_when_provided", func(t *testing.T) {
		id, err := getInboxID([]string{"test-inbox-id"})
		assert.NoError(t, err)
		assert.Equal(t, "test-inbox-id", id)
	})

	t.Run("returns_error_when_no_args_and_no_env", func(t *testing.T) {
		// Ensure env var is not set
		t.Setenv("NYLAS_INBOUND_GRANT_ID", "")
		_, err := getInboxID([]string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inbox ID required")
	})

	t.Run("returns_env_var_when_no_args", func(t *testing.T) {
		t.Setenv("NYLAS_INBOUND_GRANT_ID", "env-inbox-id")
		id, err := getInboxID([]string{})
		assert.NoError(t, err)
		assert.Equal(t, "env-inbox-id", id)
	})

	t.Run("prefers_arg_over_env_var", func(t *testing.T) {
		t.Setenv("NYLAS_INBOUND_GRANT_ID", "env-inbox-id")
		id, err := getInboxID([]string{"arg-inbox-id"})
		assert.NoError(t, err)
		assert.Equal(t, "arg-inbox-id", id)
	})
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "just now"},
		{1 * time.Minute, "1 minute ago"},
		{5 * time.Minute, "5 minutes ago"},
		{1 * time.Hour, "1 hour ago"},
		{5 * time.Hour, "5 hours ago"},
		{24 * time.Hour, "1 day ago"},
		{48 * time.Hour, "2 days ago"},
		{72 * time.Hour, "3 days ago"},
	}

	for _, tt := range tests {
		past := time.Now().Add(-tt.duration)
		got := common.FormatTimeAgo(past)
		assert.Equal(t, tt.expected, got)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"short", 5, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"longer than max", 10, "longer ..."},
	}

	for _, tt := range tests {
		got := common.Truncate(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, got)
	}
}

func TestFormatParticipant(t *testing.T) {
	tests := []struct {
		contact  domain.EmailParticipant
		expected string
	}{
		{domain.EmailParticipant{Name: "John", Email: "john@example.com"}, "John"},
		{domain.EmailParticipant{Name: "", Email: "jane@example.com"}, "jane@example.com"},
		{domain.EmailParticipant{Name: "Alice", Email: ""}, "Alice"},
	}

	for _, tt := range tests {
		got := common.FormatParticipant(tt.contact)
		assert.Equal(t, tt.expected, got)
	}
}

func TestFormatParticipants(t *testing.T) {
	contacts := []domain.EmailParticipant{
		{Name: "John", Email: "john@example.com"},
		{Name: "", Email: "jane@example.com"},
	}
	got := common.FormatParticipants(contacts)
	assert.Equal(t, "John, jane@example.com", got)
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"valid", "active"},
		{"invalid", "invalid"},
		{"pending", "pending"},
	}

	for _, tt := range tests {
		got := formatStatus(tt.status)
		assert.Contains(t, got, tt.contains)
	}
}

// =============================================================================
// HELP OUTPUT TESTS
// =============================================================================

func TestInboundCommandHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	expectedStrings := []string{
		"inbound",
		"list",
		"show",
		"create",
		"delete",
		"messages",
		"monitor",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestInboundListHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "list", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "list")
	assert.Contains(t, stdout, "--json")
}

func TestInboundCreateHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "create", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "--json")
	assert.Contains(t, stdout, "email-prefix")
}

func TestInboundDeleteHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "delete", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "delete")
	assert.Contains(t, stdout, "--yes")
	assert.Contains(t, stdout, "--force")
}

func TestInboundMessagesHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "messages", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "messages")
	assert.Contains(t, stdout, "--limit")
	assert.Contains(t, stdout, "--unread")
	assert.Contains(t, stdout, "--json")
}

func TestInboundMonitorHelp(t *testing.T) {
	cmd := NewInboundCmd()
	stdout, _, err := executeCommand(cmd, "monitor", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "monitor")
	assert.Contains(t, stdout, "--port")
	assert.Contains(t, stdout, "--tunnel")
	assert.Contains(t, stdout, "--secret")
	assert.Contains(t, stdout, "--json")
	assert.Contains(t, stdout, "--quiet")
}
