package common

import (
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/ports"
)

// GetConfigStore returns the appropriate config store based on the --config flag.
// It walks up the parent command chain to find the flag value.
// If no custom config path is found, returns the default file store.
func GetConfigStore(cmd *cobra.Command) ports.ConfigStore {
	configPath := GetConfigPath(cmd)
	if configPath != "" {
		return config.NewFileStore(configPath)
	}
	return config.NewDefaultFileStore()
}

// GetConfigPath walks up the parent command chain to find the --config flag value.
// Returns empty string if no config path is set.
func GetConfigPath(cmd *cobra.Command) string {
	// Try current command first
	if configPath, err := cmd.Flags().GetString("config"); err == nil && configPath != "" {
		return configPath
	}

	// Walk up parent chain to find config flag
	for parent := cmd.Parent(); parent != nil; parent = parent.Parent() {
		if configPath, err := parent.Flags().GetString("config"); err == nil && configPath != "" {
			return configPath
		}
	}

	return ""
}
