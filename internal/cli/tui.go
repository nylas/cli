package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/nylas/cli/internal/tui"
)

// NewTUICmd creates the tui command.
func NewTUICmd() *cobra.Command {
	var refreshInterval int
	var theme string

	cmd := &cobra.Command{
		Use:   "tui [resource]",
		Short: "Launch interactive terminal UI",
		Long: `Launch a k9s-style terminal interface for managing your Nylas email.

The TUI provides:
  - Real-time email list with auto-refresh
  - Keyboard-driven navigation (vim-style: j/k)
  - Read, star, and manage messages
  - Resource views for messages, events, contacts, webhooks, grants

Navigation:
  ↑/k, ↓/j    Move up/down
  g/G         Go to top/bottom
  enter       Open/select
  esc         Go back
  :           Command mode
  /           Filter
  ?           Help
  Ctrl+C      Quit

Themes:
  k9s         Default k9s style (blue/orange)
  amber       Amber phosphor CRT
  green       Green phosphor CRT
  apple2      Apple ][ style
  vintage     Vintage neon green
  ibm         IBM DOS white
  futuristic  Steel blue futuristic
  matrix      Matrix green
  norton      Norton Commander DOS (blue/cyan)

Resources:
  messages    Email messages
  events      Calendar events
  contacts    Contacts
  webhooks    Webhooks
  grants      Connected accounts`,
		Example: `  # Launch TUI at dashboard
  nylas tui

  # Launch directly to messages
  nylas tui messages

  # Launch with retro amber theme
  nylas tui --theme amber

  # Launch with green CRT theme
  nylas tui messages --theme green

  # Launch directly to events with custom refresh
  nylas tui events --refresh 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initialView := ""
			if len(args) > 0 {
				initialView = args[0]
			}
			themeExplicitlySet := cmd.Flags().Changed("theme")
			return runTUI(time.Duration(refreshInterval)*time.Second, initialView, tui.ThemeName(theme), themeExplicitlySet)
		},
	}

	cmd.Flags().IntVar(&refreshInterval, "refresh", 3, "Refresh interval in seconds")
	cmd.Flags().StringVar(&theme, "theme", "k9s", "Color theme (k9s, amber, green, apple2, vintage, ibm, futuristic, matrix, norton, or custom)")

	// Add subcommands for direct navigation
	cmd.AddCommand(newTUIResourceCmd("messages", "m", "Launch TUI directly to messages view"))
	cmd.AddCommand(newTUIResourceCmdWithAliases("events", []string{"e", "calendar", "cal"}, "Launch TUI directly to calendar/events view"))
	cmd.AddCommand(newTUIResourceCmd("contacts", "c", "Launch TUI directly to contacts view"))
	cmd.AddCommand(newTUIResourceCmd("webhooks", "w", "Launch TUI directly to webhooks view"))
	cmd.AddCommand(newTUIResourceCmd("grants", "g", "Launch TUI directly to grants view"))

	// Add theme management subcommand
	cmd.AddCommand(newThemeCmd())

	return cmd
}

// newThemeCmd creates the theme management command.
func newThemeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Manage TUI themes",
		Long: `Manage custom TUI themes (k9s-style YAML configuration).

Custom themes are loaded from ~/.config/nylas/themes/<name>.yaml
Use 'nylas tui theme init' to create a starter theme file.`,
	}

	cmd.AddCommand(newThemeInitCmd())
	cmd.AddCommand(newThemeListCmd())
	cmd.AddCommand(newThemeValidateCmd())
	cmd.AddCommand(newThemeSetDefaultCmd())

	return cmd
}

// newThemeInitCmd creates a starter theme file.
func newThemeInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [name]",
		Short: "Create a starter custom theme file",
		Long: `Create a starter custom theme file in ~/.config/nylas/themes/

The generated YAML file uses the k9s skin format and can be fully customized.
After creating, use: nylas tui --theme <name>`,
		Example: `  # Create a theme called "mytheme"
  nylas tui theme init mytheme

  # Then use it
  nylas tui --theme mytheme`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeName := args[0]

			// Get themes directory
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return common.WrapGetError("home directory", err)
			}

			themesDir := filepath.Join(homeDir, ".config", "nylas", "themes")
			themePath := filepath.Join(themesDir, themeName+".yaml")

			// Check if file already exists
			if _, err := os.Stat(themePath); err == nil {
				return fmt.Errorf("theme file already exists: %s", themePath)
			}

			// Create the theme file
			if err := tui.CreateDefaultThemeFile(themePath); err != nil {
				return common.WrapCreateError("theme file", err)
			}

			fmt.Printf("Created theme file: %s\n\n", themePath)
			fmt.Printf("To use this theme:\n")
			fmt.Printf("  nylas tui --theme %s\n\n", themeName)
			fmt.Printf("Edit the YAML file to customize colors.\n")
			fmt.Printf("See k9s skin documentation for more options:\n")
			fmt.Printf("  https://k9scli.io/topics/skins/\n")

			return nil
		},
	}
}

// newThemeListCmd lists available themes.
func newThemeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available themes",
		Long:  "List all built-in and custom themes available for the TUI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Built-in themes:")
			for _, theme := range tui.AvailableThemes() {
				fmt.Printf("  %s\n", theme)
			}

			customThemes := tui.ListCustomThemes()
			if len(customThemes) > 0 {
				fmt.Println("\nCustom themes (~/.config/nylas/themes/):")
				for _, theme := range customThemes {
					fmt.Printf("  %s\n", theme)
				}
			} else {
				fmt.Println("\nNo custom themes found.")
				fmt.Println("Create one with: nylas tui theme init <name>")
			}

			return nil
		},
	}
}

// newThemeValidateCmd validates a custom theme file.
func newThemeValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <name>",
		Short: "Validate a custom theme file",
		Long: `Validate a custom theme file and check for common errors.

This command checks:
  - File exists and is readable
  - YAML syntax is valid
  - Color values are valid (#RRGGBB hex or named colors)
  - Required color definitions are present`,
		Example: `  # Validate a custom theme
  nylas tui theme validate mytheme

  # Check if the testcustom theme is valid
  nylas tui theme validate testcustom`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeName := args[0]

			// Check if it's a built-in theme
			if tui.IsBuiltInTheme(tui.ThemeName(themeName)) {
				fmt.Printf("'%s' is a built-in theme (no validation needed)\n", themeName)
				return nil
			}

			// Validate the custom theme
			result, err := tui.ValidateTheme(themeName)
			if err != nil {
				return common.WrapError(err)
			}

			// Display results
			fmt.Printf("Theme: %s\n", result.ThemeName)
			fmt.Printf("File:  %s\n", result.FilePath)

			if result.FileSize > 0 {
				fmt.Printf("Size:  %d bytes\n", result.FileSize)
			}

			fmt.Println()

			if len(result.Errors) > 0 {
				fmt.Println(common.Red.Sprint("Errors:"))
				for _, err := range result.Errors {
					fmt.Printf("  ✗ %s\n", err)
				}
			}

			if len(result.Warnings) > 0 {
				fmt.Println(common.Yellow.Sprint("Warnings:"))
				for _, warn := range result.Warnings {
					fmt.Printf("  ! %s\n", warn)
				}
			}

			if len(result.ColorsFound) > 0 {
				fmt.Println(common.Green.Sprint("Colors found:"))
				for _, color := range result.ColorsFound {
					fmt.Printf("  ✓ %s\n", color)
				}
			}

			fmt.Println()

			if result.Valid {
				fmt.Println(common.Green.Sprint("✓ Theme is valid!"))
				fmt.Printf("\nTo use this theme:\n  nylas tui --theme %s\n", themeName)
			} else {
				fmt.Println(common.Red.Sprint("✗ Theme has errors"))
				fmt.Println("\nCommon fixes:")
				fmt.Println("  - Use proper YAML indentation (2 spaces, no tabs)")
				fmt.Println("  - Use hex colors like #RRGGBB (e.g., #FF0000 for red)")
				fmt.Println("  - Ensure at least 'foreground' or 'k9s.body.fgColor' is defined")
				return fmt.Errorf("theme validation failed")
			}

			return nil
		},
	}
}

// newThemeSetDefaultCmd sets the default theme in config.
func newThemeSetDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-default <name>",
		Short: "Set the default TUI theme",
		Long: `Set the default theme that will be used when launching the TUI.

This saves the theme preference to ~/.config/nylas/config.yaml.
You can still override it with --theme flag when launching.`,
		Example: `  # Set amber as the default theme
  nylas tui theme set-default amber

  # Set a custom theme as default
  nylas tui theme set-default mytheme

  # Clear the default (use built-in k9s theme)
  nylas tui theme set-default k9s`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeName := args[0]

			// Validate the theme exists (either built-in or custom)
			if !tui.IsBuiltInTheme(tui.ThemeName(themeName)) {
				// Check if it's a valid custom theme
				_, err := tui.ValidateTheme(themeName)
				if err != nil {
					return common.WrapError(err)
				}
				result, _ := tui.ValidateTheme(themeName)
				if !result.Valid {
					return fmt.Errorf("theme %q is not valid. Run 'nylas tui theme validate %s' for details", themeName, themeName)
				}
			}

			// Load current config
			configStore := config.NewDefaultFileStore()
			cfg, err := configStore.Load()
			if err != nil {
				return common.WrapLoadError("config", err)
			}

			// Update theme
			cfg.TUITheme = themeName

			// Save config
			if err := configStore.Save(cfg); err != nil {
				return common.WrapSaveError("config", err)
			}

			fmt.Printf("Default theme set to: %s\n", themeName)
			fmt.Printf("\nThis theme will be used when you run 'nylas tui'\n")
			fmt.Printf("Override with: nylas tui --theme <other-theme>\n")

			return nil
		},
	}
}

// newTUIResourceCmd creates a subcommand for a specific resource view.
func newTUIResourceCmd(resource, alias, desc string) *cobra.Command {
	return newTUIResourceCmdWithAliases(resource, []string{alias}, desc)
}

// newTUIResourceCmdWithAliases creates a subcommand with multiple aliases.
func newTUIResourceCmdWithAliases(resource string, aliases []string, desc string) *cobra.Command {
	var refreshInterval int
	var theme string

	cmd := &cobra.Command{
		Use:     resource,
		Aliases: aliases,
		Short:   desc,
		Example: fmt.Sprintf("  nylas tui %s\n  nylas tui %s --refresh 5\n  nylas tui %s --theme amber", resource, aliases[0], resource),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeExplicitlySet := cmd.Flags().Changed("theme")
			return runTUI(time.Duration(refreshInterval)*time.Second, resource, tui.ThemeName(theme), themeExplicitlySet)
		},
	}

	cmd.Flags().IntVar(&refreshInterval, "refresh", 3, "Refresh interval in seconds")
	cmd.Flags().StringVar(&theme, "theme", "k9s", "Color theme (k9s, amber, green, apple2, vintage, ibm, futuristic, matrix, norton, or custom)")
	return cmd
}

func runTUI(refreshInterval time.Duration, initialView string, theme tui.ThemeName, themeExplicitlySet bool) error {
	// Load config
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err != nil {
		return common.WrapLoadError("config", err)
	}
	if cfg == nil {
		cfg = &domain.Config{}
	}

	// Get Nylas client using the common helper (handles env vars and keyring)
	client, err := common.GetNylasClient()
	if err != nil {
		return err
	}

	// Get default grant
	grantStore, err := common.NewDefaultGrantStore()
	if err != nil {
		return common.WrapError(err)
	}
	grantID, err := grantStore.GetDefaultGrant()
	if err != nil {
		return fmt.Errorf("no default grant set. Run 'nylas auth login' first")
	}

	// Get grant email for display
	grantInfo, err := grantStore.GetGrant(grantID)
	if err != nil {
		return common.WrapGetError("grant info", err)
	}

	return runTViewTUI(client, grantStore, grantID, grantInfo, cfg, theme, themeExplicitlySet, refreshInterval, initialView)
}

func runTViewTUI(client ports.NylasClient, grantStore ports.GrantStore, grantID string, grantInfo *domain.GrantInfo, cfg *domain.Config, theme tui.ThemeName, themeExplicitlySet bool, refreshInterval time.Duration, initialView string) error {
	// Use config theme if no explicit --theme flag was provided
	if !themeExplicitlySet && cfg.TUITheme != "" {
		theme = tui.ThemeName(cfg.TUITheme)
	}

	// Check if theme loads correctly and show helpful error if not
	_, themeErr := tui.GetThemeStylesWithError(theme)
	if themeErr != nil {
		// Show error but continue with default theme
		fmt.Fprintf(os.Stderr, "%s %s\n", common.Yellow.Sprint("Warning:"), themeErr)
		fmt.Fprintf(os.Stderr, "Falling back to default theme (k9s)\n\n")
		fmt.Fprintf(os.Stderr, "To fix this, run: nylas tui theme validate %s\n\n", theme)
	}

	// Create TUI app (k9s-style using tview)
	app := tui.NewApp(tui.Config{
		Client:          client,
		GrantStore:      grantStore, // Enable grant switching in TUI
		GrantID:         grantID,
		Email:           grantInfo.Email,
		Provider:        string(grantInfo.Provider),
		RefreshInterval: refreshInterval,
		InitialView:     initialView,
		Theme:           theme,
	})

	// Run the application
	return app.Run()
}
