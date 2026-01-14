package cli

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

// TestRootCommand tests the root command.
func TestRootCommand(t *testing.T) {
	t.Run("help_flag", func(t *testing.T) {
		rootCmd := GetRootCmd()
		_, _, err := executeCommand(rootCmd, "--help")
		if err != nil {
			t.Fatalf("Help command failed: %v", err)
		}
	})

	t.Run("version_command", func(t *testing.T) {
		rootCmd := GetRootCmd()
		stdout, stderr, err := executeCommand(rootCmd, "version")
		if err != nil {
			t.Fatalf("Version command failed: %v", err)
		}

		// Version may output to stdout or stderr
		output := stdout + stderr
		if output == "" {
			t.Error("Version output should not be empty")
		}
	})

	t.Run("unknown_command_returns_error", func(t *testing.T) {
		rootCmd := GetRootCmd()
		_, _, err := executeCommand(rootCmd, "unknowncommand")
		if err == nil {
			t.Error("Unknown command should return error")
		}
	})
}

// TestGlobalFlags tests global flags.
func TestGlobalFlags(t *testing.T) {
	t.Run("json_flag_exists", func(t *testing.T) {
		rootCmd := GetRootCmd()
		flag := rootCmd.PersistentFlags().Lookup("json")
		if flag == nil {
			t.Error("Expected --json flag to exist")
		}
	})

	t.Run("verbose_flag_exists", func(t *testing.T) {
		rootCmd := GetRootCmd()
		flag := rootCmd.PersistentFlags().Lookup("verbose")
		if flag == nil {
			t.Error("Expected --verbose flag to exist")
		}

		// Check short flag
		shortFlag := rootCmd.PersistentFlags().ShorthandLookup("v")
		if shortFlag == nil {
			t.Error("Expected -v shorthand flag to exist")
		}
	})

	t.Run("no_color_flag_exists", func(t *testing.T) {
		rootCmd := GetRootCmd()
		flag := rootCmd.PersistentFlags().Lookup("no-color")
		if flag == nil {
			t.Error("Expected --no-color flag to exist")
		}
	})

	t.Run("config_flag_exists", func(t *testing.T) {
		rootCmd := GetRootCmd()
		flag := rootCmd.PersistentFlags().Lookup("config")
		if flag == nil {
			t.Error("Expected --config flag to exist")
		}
	})
}

// TestCommandDescriptions ensures all commands have proper descriptions.
func TestCommandDescriptions(t *testing.T) {
	rootCmd := GetRootCmd()

	if rootCmd.Short == "" {
		t.Error("Root command should have Short description")
	}

	if rootCmd.Long == "" {
		t.Error("Root command should have Long description")
	}

	// Check version command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Short == "" {
			t.Errorf("Command %q should have Short description", cmd.Name())
		}
	}
}

// TestCommandUsage ensures commands have proper usage strings.
func TestCommandUsage(t *testing.T) {
	rootCmd := GetRootCmd()

	if rootCmd.Use != "nylas" {
		t.Errorf("Root command Use = %q, want %q", rootCmd.Use, "nylas")
	}
}
