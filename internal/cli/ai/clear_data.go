package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newClearDataCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clear-data",
		Short: "Clear all AI data and learned patterns",
		Long: `Clear all AI-related data including learned patterns, usage statistics,
and cached responses.

This command will:
  - Delete learned scheduling patterns
  - Clear usage statistics
  - Remove cached AI responses
  - Reset privacy-related data

Note: This does not affect your AI configuration (providers, API keys, etc.)

Examples:
  # Clear all AI data (with confirmation)
  nylas ai clear-data

  # Clear without confirmation
  nylas ai clear-data --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Confirm deletion unless --force is used
			if !force {
				fmt.Println("⚠️  This will delete all AI data including:")
				fmt.Println("   - Learned scheduling patterns")
				fmt.Println("   - Usage statistics")
				fmt.Println("   - Cached responses")
				fmt.Println()

				if !common.Confirm("Are you sure?", false) {
					fmt.Println("\n❌ Cancelled")
					return nil
				}
			}

			// Get data directory
			configDir, err := os.UserConfigDir()
			if err != nil {
				return common.WrapGetError("config directory", err)
			}

			dataDir := filepath.Join(configDir, "nylas", "ai-data")

			// Check if directory exists
			if _, err := os.Stat(dataDir); os.IsNotExist(err) {
				fmt.Println("✓ No AI data found (already clean)")
				return nil
			}

			// Remove AI data directory
			if err := os.RemoveAll(dataDir); err != nil {
				return common.WrapDeleteError("AI data", err)
			}

			fmt.Println("✓ AI data cleared successfully")
			fmt.Println()
			fmt.Println("The following have been deleted:")
			fmt.Println("  - Learned scheduling patterns")
			fmt.Println("  - Usage statistics")
			fmt.Println("  - Cached responses")
			fmt.Println()
			fmt.Println("Your AI configuration (providers, settings) has been preserved.")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
