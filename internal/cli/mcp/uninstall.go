package mcp

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var (
		assistantID  string
		uninstallAll bool
	)

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove MCP configuration from AI assistants",
		Long: `Remove the Nylas MCP server configuration from AI assistants.

This command removes the nylas entry from the mcpServers section of
the AI assistant's configuration file.`,
		Example: `  # Uninstall from specific assistant
  nylas mcp uninstall --assistant claude-desktop

  # Uninstall from all configured assistants
  nylas mcp uninstall --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(assistantID, uninstallAll)
		},
	}

	cmd.Flags().StringVarP(&assistantID, "assistant", "a", "", "Target assistant (claude-desktop, claude-code, cursor, windsurf, vscode)")
	cmd.Flags().BoolVar(&uninstallAll, "all", false, "Uninstall from all configured assistants")

	return cmd
}

func runUninstall(assistantID string, uninstallAll bool) error {
	var assistants []Assistant

	if uninstallAll {
		assistants = Assistants
	} else if assistantID != "" {
		a := GetAssistantByID(assistantID)
		if a == nil {
			return fmt.Errorf("unknown assistant: %s\n\nSupported: claude-desktop, claude-code, cursor, windsurf, vscode", assistantID)
		}
		assistants = []Assistant{*a}
	} else {
		return fmt.Errorf("please specify --assistant or --all")
	}

	successCount := 0

	for _, a := range assistants {
		configPath := a.GetConfigPath()
		if configPath == "" {
			continue
		}

		configState := inspectAssistantConfig(a, configPath)
		if !configState.HasNylas {
			if !uninstallAll {
				msg := "not configured"
				if configState.ConfigFileExists {
					msg = "nylas not found in config"
				}
				_, _ = common.Yellow.Printf("  ! %s: %s\n", a.Name, msg)
			}
			continue
		}

		err := uninstallFromAssistant(a)
		if err != nil {
			_, _ = common.Yellow.Printf("  ! %s: %v\n", a.Name, err)
			continue
		}

		_, _ = common.Green.Printf("  ✓ %s: removed from %s\n", a.Name, configPath)
		successCount++
	}

	if successCount > 0 {
		fmt.Println()
		fmt.Println("Restart your AI assistant to apply the changes.")
	}

	return nil
}

func uninstallFromAssistant(a Assistant) error {
	configPath := a.GetConfigPath()

	// Read existing config
	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	removeAssistantServer(config, a, nylasServerName)

	// Write config back
	if err := writeConfig(configPath, config); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
