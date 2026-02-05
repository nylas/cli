package audit

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/adapters/audit"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage audit configuration",
		Long:  `View and modify audit logging configuration.`,
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show audit configuration",
		Long:  `Display current audit logging configuration and storage statistics.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := audit.NewFileStore("")
			if err != nil {
				return fmt.Errorf("open audit store: %w", err)
			}

			cfg, err := store.GetConfig()
			if err != nil {
				fmt.Println("Audit logging: not configured")
				fmt.Println()
				_, _ = common.Yellow.Println("Run 'nylas audit init' to set up audit logging.")
				return nil
			}

			_, _ = common.Bold.Println("Audit Configuration")
			fmt.Println()
			fmt.Printf("  Enabled:         %s\n", yesNo(cfg.Enabled))
			fmt.Printf("  Path:            %s\n", cfg.Path)
			fmt.Printf("  Retention:       %d days\n", cfg.RetentionDays)
			fmt.Printf("  Max Size:        %d MB\n", cfg.MaxSizeMB)
			fmt.Printf("  Format:          %s\n", cfg.Format)
			fmt.Printf("  Daily Rotation:  %s\n", yesNo(cfg.RotateDaily))
			fmt.Printf("  Compress Old:    %s\n", yesNo(cfg.CompressOld))
			fmt.Printf("  Log Request ID:  %s\n", yesNo(cfg.LogRequestID))
			fmt.Printf("  Log API Details: %s\n", yesNo(cfg.LogAPIDetails))

			// Storage stats
			fileCount, totalSize, oldestEntry, err := store.Stats()
			if err == nil && fileCount > 0 {
				fmt.Println()
				fmt.Println("Storage:")
				fmt.Printf("  Current size:    %s\n", FormatSize(totalSize))
				fmt.Printf("  Files:           %d\n", fileCount)
				if oldestEntry != nil {
					fmt.Printf("  Oldest entry:    %s\n", oldestEntry.Timestamp.Format("2006-01-02"))
				}
			}

			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Update an audit logging configuration setting.

Available keys:
  retention_days   - Days to keep logs (integer)
  max_size_mb      - Maximum storage in MB (integer)
  rotate_daily     - Create new file each day (true/false)
  compress_old     - Compress files older than 7 days (true/false)
  log_request_id   - Log Nylas request IDs (true/false)
  log_api_details  - Log API endpoint and status (true/false)`,
		Example: `  # Set retention to 30 days
  nylas audit config set retention_days 30

  # Enable compression
  nylas audit config set compress_old true

  # Disable request ID logging
  nylas audit config set log_request_id false`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			store, err := audit.NewFileStore("")
			if err != nil {
				return fmt.Errorf("open audit store: %w", err)
			}

			cfg, err := store.GetConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Update the setting
			switch key {
			case "retention_days":
				n, err := strconv.Atoi(value)
				if err != nil || n < 1 {
					return fmt.Errorf("retention_days must be a positive integer")
				}
				cfg.RetentionDays = n

			case "max_size_mb":
				n, err := strconv.Atoi(value)
				if err != nil || n < 1 {
					return fmt.Errorf("max_size_mb must be a positive integer")
				}
				cfg.MaxSizeMB = n

			case "rotate_daily":
				cfg.RotateDaily = parseBool(value)

			case "compress_old":
				cfg.CompressOld = parseBool(value)

			case "log_request_id":
				cfg.LogRequestID = parseBool(value)

			case "log_api_details":
				cfg.LogAPIDetails = parseBool(value)

			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			if err := store.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			_, _ = common.Green.Printf("✓ Set %s = %s\n", key, value)
			return nil
		},
	}
}

func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "yes" || s == "1" || s == "on"
}
