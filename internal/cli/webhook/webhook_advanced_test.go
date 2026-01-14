package webhook

import (
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/stretchr/testify/assert"
)

func TestTriggersCommand(t *testing.T) {
	cmd := newTriggersCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "triggers", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "trigger-types")
		assert.Contains(t, cmd.Aliases, "events")
	})

	t.Run("has_format_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "text", flag.DefValue)
	})

	t.Run("has_category_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("category")
		assert.NotNil(t, flag)
	})

	t.Run("has_category_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("c")
		assert.NotNil(t, flag)
		assert.Equal(t, "category", flag.Name)
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "webhook triggers")
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("truncate_short_string", func(t *testing.T) {
		result := common.Truncate("hello", 10)
		assert.Equal(t, "hello", result)
	})

	t.Run("truncate_long_string", func(t *testing.T) {
		result := common.Truncate("hello world this is long", 10)
		assert.Equal(t, "hello w...", result)
	})

	t.Run("truncate_exact_length", func(t *testing.T) {
		result := common.Truncate("hello", 5)
		assert.Equal(t, "hello", result)
	})

	t.Run("getStatusIcon_active", func(t *testing.T) {
		result := getStatusIcon("active")
		assert.Contains(t, result, "●")
		// Color codes are only present in TTY environments
	})

	t.Run("getStatusIcon_inactive", func(t *testing.T) {
		result := getStatusIcon("inactive")
		assert.Contains(t, result, "●")
		// Color codes are only present in TTY environments
	})

	t.Run("getStatusIcon_failing", func(t *testing.T) {
		result := getStatusIcon("failing")
		assert.Contains(t, result, "●")
		// Color codes are only present in TTY environments
	})

	t.Run("getStatusIcon_unknown", func(t *testing.T) {
		result := getStatusIcon("unknown")
		assert.Equal(t, "○", result)
	})

	t.Run("capitalize_empty", func(t *testing.T) {
		result := capitalize("")
		assert.Equal(t, "", result)
	})

	t.Run("capitalize_lowercase", func(t *testing.T) {
		result := capitalize("hello")
		assert.Equal(t, "Hello", result)
	})

	t.Run("capitalize_single_char", func(t *testing.T) {
		result := capitalize("a")
		assert.Equal(t, "A", result)
	})
}

func TestWebhookCommandHelp(t *testing.T) {
	cmd := NewWebhookCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	expectedStrings := []string{
		"webhook",
		"list",
		"show",
		"create",
		"update",
		"delete",
		"test",
		"triggers",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestWebhookListHelp(t *testing.T) {
	cmd := NewWebhookCmd()
	stdout, _, err := executeCommand(cmd, "list", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "list")
	assert.Contains(t, stdout, "--format")
	assert.Contains(t, stdout, "--full-ids")
}

func TestWebhookCreateHelp(t *testing.T) {
	cmd := NewWebhookCmd()
	stdout, _, err := executeCommand(cmd, "create", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "create")
	assert.Contains(t, stdout, "--url")
	assert.Contains(t, stdout, "--triggers")
}

func TestWebhookTriggersHelp(t *testing.T) {
	cmd := NewWebhookCmd()
	stdout, _, err := executeCommand(cmd, "triggers", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "triggers")
	assert.Contains(t, stdout, "--format")
	assert.Contains(t, stdout, "--category")
}

func TestWebhookTestHelp(t *testing.T) {
	cmd := NewWebhookCmd()
	stdout, _, err := executeCommand(cmd, "test", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "test")
	assert.Contains(t, stdout, "send")
	assert.Contains(t, stdout, "payload")
}

func TestServerCommand(t *testing.T) {
	cmd := newServerCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "server", cmd.Use)
	})

	t.Run("has_port_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("port")
		assert.NotNil(t, flag)
		assert.Equal(t, "3000", flag.DefValue)
	})

	t.Run("has_port_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("p")
		assert.NotNil(t, flag)
		assert.Equal(t, "port", flag.Name)
	})

	t.Run("has_tunnel_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("tunnel")
		assert.NotNil(t, flag)
	})

	t.Run("has_tunnel_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("t")
		assert.NotNil(t, flag)
		assert.Equal(t, "tunnel", flag.Name)
	})

	t.Run("has_secret_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("secret")
		assert.NotNil(t, flag)
	})

	t.Run("has_secret_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("s")
		assert.NotNil(t, flag)
		assert.Equal(t, "secret", flag.Name)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_quiet_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("quiet")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_quiet_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("q")
		assert.NotNil(t, flag)
		assert.Equal(t, "quiet", flag.Name)
	})

	t.Run("has_path_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("path")
		assert.NotNil(t, flag)
		assert.Equal(t, "/webhook", flag.DefValue)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "webhook")
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "cloudflared")
	})
}

func TestWebhookServerHelp(t *testing.T) {
	cmd := NewWebhookCmd()
	stdout, _, err := executeCommand(cmd, "server", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "server")
	assert.Contains(t, stdout, "--port")
	assert.Contains(t, stdout, "--tunnel")
	assert.Contains(t, stdout, "cloudflared")
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   string
	}{
		{
			name:   "short_secret_fully_masked",
			secret: "abc123",
			want:   "******",
		},
		{
			name:   "8_char_secret_fully_masked",
			secret: "12345678",
			want:   "********",
		},
		{
			name:   "9_char_secret_partial_mask",
			secret: "123456789",
			want:   "12*****89",
		},
		{
			name:   "12_char_secret_partial_mask",
			secret: "123456789012",
			want:   "12********12",
		},
		{
			name:   "13_char_secret_full_mask",
			secret: "1234567890123",
			want:   "1234*****0123",
		},
		{
			name:   "typical_webhook_secret",
			secret: "whsec_abcdefghijklmnopqrstuvwxyz123456",
			want:   "whse******************************3456",
		},
		{
			name:   "empty_secret",
			secret: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskSecret(tt.secret)
			assert.Equal(t, tt.want, got)
		})
	}
}
