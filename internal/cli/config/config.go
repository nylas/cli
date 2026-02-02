package config

import (
	"fmt"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configStore *config.FileStore

func init() {
	configStore = config.NewDefaultFileStore()
}

// NewConfigCmd creates the config command.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `Manage Nylas CLI configuration settings.

Configuration is stored in ~/.config/nylas/config.yaml by default.
If the config file doesn't exist, sensible defaults are used automatically.`,
		Example: `  # Show all configuration
  nylas config list

  # Get a specific value
  nylas config get api.timeout

  # Set a value
  nylas config set api.timeout 120s

  # Set default grant ID
  nylas config set default_grant grant_abc123

  # Set GPG default key
  nylas config set gpg.default_key 601FEE9B1D60185F

  # Enable auto-sign all emails
  nylas config set gpg.auto_sign true

  # Initialize config with defaults
  nylas config init`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newPathCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "show"},
		Short:   "Show all configuration",
		Long:    "Display all configuration settings in YAML format.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configStore.Load()
			if err != nil {
				return common.WrapLoadError("configuration", err)
			}

			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			fmt.Println(string(data))
			return nil
		},
	}
}

func newPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		Long:  "Display the path to the configuration file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(configStore.Path())
			if !configStore.Exists() {
				fmt.Println(common.Yellow.Sprint("(file does not exist yet - using defaults)"))
			}
			return nil
		},
	}
}
