package otp

import (
	"bytes"
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/spf13/cobra"
)

// executeCommand executes a command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	return testutil.ExecuteCommand(root, args...)
}

// TestNewOTPCmd tests the otp command creation.
func TestNewOTPCmd(t *testing.T) {
	cmd := NewOTPCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "otp" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "otp")
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
			t.Error("OTP command should have subcommands")
		}
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"get", "watch", "list", "messages"}

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

// TestGetCommand tests the get subcommand.
func TestGetCommand(t *testing.T) {
	cmd := newGetCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "get [email]" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "get [email]")
		}
	})

	t.Run("has_no_copy_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("no-copy")
		if flag == nil {
			t.Error("Expected --no-copy flag")
		}
	})

	t.Run("has_raw_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("raw")
		if flag == nil {
			t.Error("Expected --raw flag")
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
	})
}

// TestWatchCommand tests the watch subcommand.
func TestWatchCommand(t *testing.T) {
	cmd := newWatchCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "watch [email]" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "watch [email]")
		}
	})

	t.Run("has_interval_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("interval")
		if flag == nil {
			t.Error("Expected --interval flag")
			return
		}
		if flag.DefValue != "10" {
			t.Errorf("--interval default = %q, want %q", flag.DefValue, "10")
		}
	})

	t.Run("has_interval_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("i")
		if flag == nil {
			t.Error("Expected -i shorthand for --interval")
		}
	})

	t.Run("has_no_copy_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("no-copy")
		if flag == nil {
			t.Error("Expected --no-copy flag")
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

	t.Run("has_short_description", func(t *testing.T) {
		if cmd.Short == "" {
			t.Error("Command should have Short description")
		}
	})
}

// TestMessagesCommand tests the messages subcommand.
func TestMessagesCommand(t *testing.T) {
	cmd := newMessagesCmd()

	t.Run("command_name", func(t *testing.T) {
		if cmd.Use != "messages [email]" {
			t.Errorf("Command Use = %q, want %q", cmd.Use, "messages [email]")
		}
	})

	t.Run("has_limit_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		if flag == nil {
			t.Error("Expected --limit flag")
			return
		}
		if flag.DefValue != "10" {
			t.Errorf("--limit default = %q, want %q", flag.DefValue, "10")
		}
	})

	t.Run("has_limit_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("l")
		if flag == nil {
			t.Error("Expected -l shorthand for --limit")
		}
	})

	t.Run("email_is_optional", func(t *testing.T) {
		// Email argument should be optional (uses default account when not provided)
		if cmd.Use != "messages [email]" {
			t.Error("Email should be optional (indicated by [email])")
		}
	})
}

// TestOTPCommandHelp tests help output for otp command.
func TestOTPCommandHelp(t *testing.T) {
	cmd := NewOTPCmd()
	stdout, _, err := executeCommand(cmd, "--help")

	if err != nil {
		t.Fatalf("Help failed: %v", err)
	}

	// Check that help contains expected content
	expectedStrings := []string{
		"otp",
		"get",
		"watch",
		"list",
		"messages",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(stdout), []byte(expected)) {
			t.Errorf("Help output should contain %q", expected)
		}
	}
}

// Note: FormatTimeAgo tests are in common/time_test.go
