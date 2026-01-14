package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAdminCmd(t *testing.T) {
	cmd := NewAdminCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "admin", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		// TODO: Add "credentials" back when implemented
		expectedCmds := []string{"applications", "connectors", "grants"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

// Applications Tests
func TestNewApplicationsCmd(t *testing.T) {
	cmd := newApplicationsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "applications", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "app")
		assert.Contains(t, cmd.Aliases, "apps")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "update", "delete"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestAppListCmd(t *testing.T) {
	cmd := newAppListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestAppShowCmd(t *testing.T) {
	cmd := newAppShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <app-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestAppCreateCmd(t *testing.T) {
	cmd := newAppCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_name_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("name")
		assert.NotNil(t, flag)
	})

	t.Run("has_region_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("region")
		assert.NotNil(t, flag)
		assert.Equal(t, "us", flag.DefValue)
	})

	t.Run("has_branding_name_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("branding-name")
		assert.NotNil(t, flag)
	})

	t.Run("has_website_url_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("website-url")
		assert.NotNil(t, flag)
	})

	t.Run("has_callback_uris_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("callback-uris")
		assert.NotNil(t, flag)
	})
}

func TestAppUpdateCmd(t *testing.T) {
	cmd := newAppUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <app-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_name_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("name")
		assert.NotNil(t, flag)
	})

	t.Run("has_branding_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("branding-name"))
		assert.NotNil(t, cmd.Flags().Lookup("website-url"))
	})
}

func TestAppDeleteCmd(t *testing.T) {
	cmd := newAppDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <app-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_yes_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_yes_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("y")
		assert.NotNil(t, flag)
	})
}

// Connectors Tests
func TestNewConnectorsCmd(t *testing.T) {
	cmd := newConnectorsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "connectors", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "connector")
		assert.Contains(t, cmd.Aliases, "conn")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "update", "delete"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestConnectorListCmd(t *testing.T) {
	cmd := newConnectorListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestConnectorShowCmd(t *testing.T) {
	cmd := newConnectorShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <connector-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestConnectorCreateCmd(t *testing.T) {
	cmd := newConnectorCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_required_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("provider"))
	})

	t.Run("has_oauth_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("client-id"))
		assert.NotNil(t, cmd.Flags().Lookup("scopes"))
	})

	t.Run("has_imap_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("imap-host"))
		assert.NotNil(t, cmd.Flags().Lookup("imap-port"))
		assert.NotNil(t, cmd.Flags().Lookup("smtp-host"))
		assert.NotNil(t, cmd.Flags().Lookup("smtp-port"))
	})

	t.Run("has_correct_default_ports", func(t *testing.T) {
		imapPort := cmd.Flags().Lookup("imap-port")
		assert.Equal(t, "993", imapPort.DefValue)

		smtpPort := cmd.Flags().Lookup("smtp-port")
		assert.Equal(t, "587", smtpPort.DefValue)
	})
}

func TestConnectorUpdateCmd(t *testing.T) {
	cmd := newConnectorUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <connector-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_update_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("scopes"))
	})
}

func TestConnectorDeleteCmd(t *testing.T) {
	cmd := newConnectorDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <connector-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_yes_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

// TODO: Credentials Tests - Uncomment when credentials.go is implemented
/*
func TestNewCredentialsCmd(t *testing.T) {
	cmd := newCredentialsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "credentials", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "credential")
		assert.Contains(t, cmd.Aliases, "creds")
		assert.Contains(t, cmd.Aliases, "cred")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "update", "delete"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestCredentialListCmd(t *testing.T) {
	cmd := newCredentialListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestCredentialShowCmd(t *testing.T) {
	cmd := newCredentialShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <credential-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestCredentialCreateCmd(t *testing.T) {
	cmd := newCredentialCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_required_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("type"))
	})

	t.Run("has_connector_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("connector-id")
		assert.NotNil(t, flag)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})
}

func TestCredentialUpdateCmd(t *testing.T) {
	cmd := newCredentialUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <credential-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_name_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("name")
		assert.NotNil(t, flag)
	})
}

func TestCredentialDeleteCmd(t *testing.T) {
	cmd := newCredentialDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <credential-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_yes_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}
*/

// Grants Tests
func TestNewGrantsCmd(t *testing.T) {
	cmd := newGrantsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "grants", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "grant")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "stats"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestGrantListCmd(t *testing.T) {
	cmd := newGrantListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_filter_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("connector-id"))
		assert.NotNil(t, cmd.Flags().Lookup("status"))
		assert.NotNil(t, cmd.Flags().Lookup("limit"))
		assert.NotNil(t, cmd.Flags().Lookup("offset"))
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})

	t.Run("has_correct_limit_default", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.Equal(t, "50", flag.DefValue)
	})
}

func TestGrantStatsCmd(t *testing.T) {
	cmd := newGrantStatsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "stats", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})
}
