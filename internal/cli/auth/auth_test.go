package auth

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// executeCommand executes a command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	err := root.Execute()

	return stdout.String(), stderr.String(), err
}

// TestNewAuthCmd tests the auth command creation.
func TestNewAuthCmd(t *testing.T) {
	cmd := NewAuthCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "auth" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "auth")
		}
	})

	t.Run("has_short_description", func(t *testing.T) {
		if cmd.Short == "" {
			t.Error("Command should have Short description")
		}
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		if len(subcommands) == 0 {
			t.Error("Auth command should have subcommands")
		}
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"config", "login", "logout", "status", "whoami", "list", "show", "switch", "token", "revoke", "add", "remove"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			if !cmdMap[expected] {
				t.Errorf("Missing expected subcommand: %s", expected)
			}
		}
	})
}

// TestShowCommand tests the show subcommand.
func TestShowCommand(t *testing.T) {
	cmd := newShowCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "show [grant-id]" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "show [grant-id]")
		}
	})

	t.Run("has_short_description", func(t *testing.T) {
		if cmd.Short == "" {
			t.Error("Expected short description")
		}
	})

	t.Run("has_long_description", func(t *testing.T) {
		if cmd.Long == "" {
			t.Error("Expected long description")
		}
	})
}

// TestConfigCommand tests the config subcommand.
func TestConfigCommand(t *testing.T) {
	cmd := newConfigCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "config" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "config")
		}
	})

	t.Run("has_region_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("region")
		if flag == nil {
			t.Error("Expected --region flag")
			return
		}
		if flag.DefValue != "us" {
			t.Errorf("--region default = %q, want %q", flag.DefValue, "us")
		}
	})

	t.Run("has_client_id_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("client-id")
		if flag == nil {
			t.Error("Expected --client-id flag")
		}
	})

	t.Run("has_api_key_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("api-key")
		if flag == nil {
			t.Error("Expected --api-key flag")
		}
	})

	t.Run("has_reset_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("reset")
		if flag == nil {
			t.Error("Expected --reset flag")
		}
	})
}

// TestLoginCommand tests the login subcommand.
func TestLoginCommand(t *testing.T) {
	cmd := newLoginCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "login" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "login")
		}
	})

	t.Run("has_provider_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("provider")
		if flag == nil {
			t.Error("Expected --provider flag")
			return
		}
		if flag.DefValue != "google" {
			t.Errorf("--provider default = %q, want %q", flag.DefValue, "google")
		}
	})

	t.Run("has_provider_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("p")
		if flag == nil {
			t.Error("Expected -p shorthand for --provider")
		}
	})
}

// TestLogoutCommand tests the logout subcommand.
func TestLogoutCommand(t *testing.T) {
	cmd := newLogoutCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "logout" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "logout")
		}
	})

	t.Run("has_short_description", func(t *testing.T) {
		if cmd.Short == "" {
			t.Error("Command should have Short description")
		}
	})
}

// TestStatusCommand tests the status subcommand.
func TestStatusCommand(t *testing.T) {
	cmd := newStatusCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "status" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "status")
		}
	})

	t.Run("has_short_description", func(t *testing.T) {
		if cmd.Short == "" {
			t.Error("Command should have Short description")
		}
	})
}

// TestWhoamiCommand tests the whoami subcommand.
func TestWhoamiCommand(t *testing.T) {
	cmd := newWhoamiCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "whoami" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "whoami")
		}
	})
}

// TestListCommand tests the list subcommand.
func TestListCommand(t *testing.T) {
	cmd := newListCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "list" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "list")
		}
	})
}

// TestSwitchCommand tests the switch subcommand.
func TestSwitchCommand(t *testing.T) {
	cmd := newSwitchCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "switch <email-or-grant-id>" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "switch <email-or-grant-id>")
		}
	})

	t.Run("requires_argument", func(t *testing.T) {
		// Switch command requires exactly 1 argument
		if cmd.Args == nil {
			t.Error("Command should have Args validator")
		}
	})
}

// TestTokenCommand tests the token subcommand.
func TestTokenCommand(t *testing.T) {
	cmd := newTokenCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "token" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "token")
		}
	})

	t.Run("has_copy_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("copy")
		if flag == nil {
			t.Error("Expected --copy flag")
		}
	})

	t.Run("has_copy_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("c")
		if flag == nil {
			t.Error("Expected -c shorthand for --copy")
		}
	})
}

// TestRevokeCommand tests the revoke subcommand.
func TestRevokeCommand(t *testing.T) {
	cmd := newRevokeCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "revoke <grant-id>" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "revoke <grant-id>")
		}
	})

	t.Run("requires_argument", func(t *testing.T) {
		// Revoke command requires exactly 1 argument
		if cmd.Args == nil {
			t.Error("Command should have Args validator")
		}
	})
}

// TestRemoveCommand tests the remove subcommand.
func TestRemoveCommand(t *testing.T) {
	cmd := newRemoveCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "remove <grant-id>" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "remove <grant-id>")
		}
	})

	t.Run("has_short_description", func(t *testing.T) {
		if cmd.Short == "" {
			t.Error("Command should have Short description")
		}
	})

	t.Run("has_long_description", func(t *testing.T) {
		if cmd.Long == "" {
			t.Error("Command should have Long description")
		}
		// Long description should clarify it doesn't revoke on server
		if !bytes.Contains([]byte(cmd.Long), []byte("NOT revoke")) {
			t.Error("Long description should clarify this does NOT revoke on server")
		}
	})

	t.Run("requires_argument", func(t *testing.T) {
		if cmd.Args == nil {
			t.Error("Command should have Args validator")
		}
	})
}

// TestAddCommand tests the add subcommand.
func TestAddCommand(t *testing.T) {
	cmd := newAddCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "add <grant-id>" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "add <grant-id>")
		}
	})

	t.Run("has_short_description", func(t *testing.T) {
		if cmd.Short == "" {
			t.Error("Command should have Short description")
		}
	})

	t.Run("has_email_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("email")
		if flag == nil {
			t.Error("Expected --email flag")
		}
	})

	t.Run("has_email_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("e")
		if flag == nil {
			t.Error("Expected -e shorthand for --email")
		}
	})

	t.Run("has_provider_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("provider")
		if flag == nil {
			t.Error("Expected --provider flag")
		}
	})

	t.Run("has_default_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("default")
		if flag == nil {
			t.Error("Expected --default flag")
		}
	})

	t.Run("requires_argument", func(t *testing.T) {
		if cmd.Args == nil {
			t.Error("Command should have Args validator")
		}
	})
}

// TestAuthCommandHelp tests help output for auth command.
func TestAuthCommandHelp(t *testing.T) {
	cmd := NewAuthCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	if err != nil {
		t.Fatalf("Help failed: %v", err)
	}

	// Check that help contains expected content
	expectedStrings := []string{
		"auth",
		"config",
		"login",
		"logout",
		"status",
		"remove",
		"add",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(stdout), []byte(expected)) {
			t.Errorf("Help output should contain %q", expected)
		}
	}
}

// TestRemoveCommandHelp tests help output for remove command.
func TestRemoveCommandHelp(t *testing.T) {
	cmd := newRemoveCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	if err != nil {
		t.Fatalf("Help failed: %v", err)
	}

	// Should explain it's local only
	if !bytes.Contains([]byte(stdout), []byte("local")) {
		t.Error("Help should mention 'local' to clarify scope")
	}
}

// TestAddCommandHelp tests help output for add command.
func TestAddCommandHelp(t *testing.T) {
	cmd := newAddCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	if err != nil {
		t.Fatalf("Help failed: %v", err)
	}

	// Should mention auto-detection
	if !bytes.Contains([]byte(stdout), []byte("auto-detected")) {
		t.Error("Help should mention auto-detection of email/provider")
	}
}
