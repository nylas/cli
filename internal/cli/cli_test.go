package cli

import (
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/spf13/cobra"
)

// executeCommand executes a command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	return testutil.ExecuteCommand(root, args...)
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

// getRootCmdWithTUI returns the root command with TUI command added (simulating main.go setup).
func getRootCmdWithTUI() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "nylas",
		Short: "Nylas CLI",
	}
	rootCmd.AddCommand(NewTUICmd())
	return rootCmd
}

// TestTUICommand tests the tui command structure.
func TestTUICommand(t *testing.T) {
	t.Run("tui_command_exists", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		tuiCmd, _, err := rootCmd.Find([]string{"tui"})
		if err != nil {
			t.Fatalf("TUI command not found: %v", err)
		}
		if tuiCmd.Name() != "tui" {
			t.Errorf("Expected command name 'tui', got %q", tuiCmd.Name())
		}
	})

	t.Run("tui_has_theme_flag", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		tuiCmd, _, _ := rootCmd.Find([]string{"tui"})
		flag := tuiCmd.Flags().Lookup("theme")
		if flag == nil {
			t.Error("Expected --theme flag on tui command")
			return
		}
		if flag.DefValue != "k9s" {
			t.Errorf("Expected default theme 'k9s', got %q", flag.DefValue)
		}
	})

	t.Run("tui_has_refresh_flag", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		tuiCmd, _, _ := rootCmd.Find([]string{"tui"})
		flag := tuiCmd.Flags().Lookup("refresh")
		if flag == nil {
			t.Error("Expected --refresh flag on tui command")
			return
		}
		if flag.DefValue != "3" {
			t.Errorf("Expected default refresh '3', got %q", flag.DefValue)
		}
	})

	t.Run("tui_has_resource_subcommands", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		tuiCmd, _, _ := rootCmd.Find([]string{"tui"})

		expectedSubcommands := []string{"messages", "events", "contacts", "webhooks", "grants", "theme"}
		for _, expected := range expectedSubcommands {
			found := false
			for _, cmd := range tuiCmd.Commands() {
				if cmd.Name() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected subcommand %q not found in tui command", expected)
			}
		}
	})
}

// TestTUIThemeCommand tests the tui theme subcommand.
func TestTUIThemeCommand(t *testing.T) {
	t.Run("theme_subcommand_exists", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		themeCmd, _, err := rootCmd.Find([]string{"tui", "theme"})
		if err != nil {
			t.Fatalf("TUI theme command not found: %v", err)
		}
		if themeCmd.Name() != "theme" {
			t.Errorf("Expected command name 'theme', got %q", themeCmd.Name())
		}
	})

	t.Run("theme_has_subcommands", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		themeCmd, _, _ := rootCmd.Find([]string{"tui", "theme"})

		expectedSubcommands := []string{"init", "list", "validate", "set-default"}
		for _, expected := range expectedSubcommands {
			found := false
			for _, cmd := range themeCmd.Commands() {
				if cmd.Name() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected subcommand %q not found in theme command", expected)
			}
		}
	})

	t.Run("theme_list_command", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		// Note: command uses fmt.Printf which goes to os.Stdout, not cmd.SetOut()
		// Just verify the command executes without error
		_, _, err := executeCommand(rootCmd, "tui", "theme", "list")
		if err != nil {
			t.Fatalf("Theme list command failed: %v", err)
		}
	})

	t.Run("theme_validate_builtin", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		// Note: command uses fmt.Printf which goes to os.Stdout
		// Just verify the command executes without error for built-in themes
		_, _, err := executeCommand(rootCmd, "tui", "theme", "validate", "k9s")
		if err != nil {
			t.Fatalf("Theme validate command failed: %v", err)
		}
	})

	t.Run("theme_validate_nonexistent", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		_, _, err := executeCommand(rootCmd, "tui", "theme", "validate", "nonexistent_theme_xyz")
		if err == nil {
			t.Error("Expected error for non-existent theme")
		}
	})

	t.Run("theme_set_default_requires_arg", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		_, _, err := executeCommand(rootCmd, "tui", "theme", "set-default")
		if err == nil {
			t.Error("Expected error when no theme name provided")
		}
	})

	t.Run("theme_init_requires_arg", func(t *testing.T) {
		rootCmd := getRootCmdWithTUI()
		_, _, err := executeCommand(rootCmd, "tui", "theme", "init")
		if err == nil {
			t.Error("Expected error when no theme name provided")
		}
	})
}

// TestTUIResourceAliases tests the resource command aliases.
func TestTUIResourceAliases(t *testing.T) {
	testCases := []struct {
		resource string
		aliases  []string
	}{
		{"messages", []string{"m"}},
		{"events", []string{"e", "calendar", "cal"}},
		{"contacts", []string{"c"}},
		{"webhooks", []string{"w"}},
		{"grants", []string{"g"}},
	}

	for _, tc := range testCases {
		t.Run(tc.resource+"_has_aliases", func(t *testing.T) {
			rootCmd := getRootCmdWithTUI()
			cmd, _, _ := rootCmd.Find([]string{"tui", tc.resource})

			for _, alias := range tc.aliases {
				found := false
				for _, a := range cmd.Aliases {
					if a == alias {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected alias %q for %s command", alias, tc.resource)
				}
			}
		})
	}
}
