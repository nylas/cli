package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/adapters/audit"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Manage and view audit logs",
		Long:  `Commands for enabling, disabling, viewing, and managing audit logs.`,
	}

	cmd.AddCommand(newEnableCmd())
	cmd.AddCommand(newDisableCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newSummaryCmd())
	cmd.AddCommand(newClearCmd())

	return cmd
}

func newEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable audit logging",
		Long:  `Enable audit logging to start recording command execution history.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := audit.NewFileStore("")
			if err != nil {
				return fmt.Errorf("open audit store: %w", err)
			}

			cfg, err := store.GetConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if !cfg.Initialized {
				return fmt.Errorf("audit logging not initialized. Run: nylas audit init")
			}

			cfg.Enabled = true
			if err := store.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			_, _ = common.Green.Println("✓ Audit logging enabled")
			fmt.Printf("  Logs will be written to: %s\n", store.Path())
			return nil
		},
	}
}

func newDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable audit logging",
		Long:  `Disable audit logging. Existing logs are preserved.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := audit.NewFileStore("")
			if err != nil {
				return fmt.Errorf("open audit store: %w", err)
			}

			cfg, err := store.GetConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			cfg.Enabled = false
			if err := store.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			_, _ = common.Yellow.Println("⏸ Audit logging disabled")
			fmt.Println("  Existing logs are preserved.")
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show audit logging status",
		Long:  `Show the current status of audit logging including configuration and storage statistics.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := audit.NewFileStore("")
			if err != nil {
				return fmt.Errorf("open audit store: %w", err)
			}

			cfg, err := store.GetConfig()
			if err != nil {
				fmt.Println("Audit logging: not initialized")
				fmt.Println()
				_, _ = common.Yellow.Println("Run 'nylas audit init' to set up audit logging.")
				return nil
			}

			// Status header
			if cfg.Enabled {
				_, _ = common.Green.Println("● Audit logging: enabled")
			} else {
				_, _ = common.Yellow.Println("○ Audit logging: disabled")
			}

			if !cfg.Initialized {
				fmt.Println()
				_, _ = common.Yellow.Println("Run 'nylas audit init' to complete setup.")
				return nil
			}

			fmt.Println()
			fmt.Println("Configuration:")
			fmt.Printf("  Path:           %s\n", cfg.Path)
			fmt.Printf("  Retention:      %d days\n", cfg.RetentionDays)
			fmt.Printf("  Max Size:       %d MB\n", cfg.MaxSizeMB)
			fmt.Printf("  Format:         %s\n", cfg.Format)
			fmt.Printf("  Daily Rotation: %s\n", yesNo(cfg.RotateDaily))
			fmt.Printf("  Compress Old:   %s\n", yesNo(cfg.CompressOld))
			fmt.Printf("  Log Request ID: %s\n", yesNo(cfg.LogRequestID))
			fmt.Printf("  Log API Details: %s\n", yesNo(cfg.LogAPIDetails))

			// Storage stats
			fileCount, totalSize, oldestEntry, err := store.Stats()
			if err == nil && fileCount > 0 {
				fmt.Println()
				fmt.Println("Storage:")
				fmt.Printf("  Current size:   %s\n", FormatSize(totalSize))
				fmt.Printf("  Files:          %d\n", fileCount)
				if oldestEntry != nil {
					fmt.Printf("  Oldest entry:   %s\n", oldestEntry.Timestamp.Format("2006-01-02"))
				}
			}

			return nil
		},
	}
}

func newClearCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all audit logs",
		Long:  `Remove all audit log files. Configuration is preserved.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := audit.NewFileStore("")
			if err != nil {
				return fmt.Errorf("open audit store: %w", err)
			}

			fileCount, totalSize, _, err := store.Stats()
			if err != nil {
				return fmt.Errorf("get stats: %w", err)
			}

			if fileCount == 0 {
				fmt.Println("No audit logs to clear.")
				return nil
			}

			if !force {
				fmt.Printf("This will delete %d log files (%s).\n", fileCount, FormatSize(totalSize))
				fmt.Print("Are you sure? [y/N]: ")

				var confirm string
				if _, err := fmt.Scanln(&confirm); err != nil {
					return nil // No input, assume no
				}
				if confirm != "y" && confirm != "Y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := store.Clear(ctx); err != nil {
				return fmt.Errorf("clear logs: %w", err)
			}

			_, _ = common.Green.Printf("✓ Cleared %d log files\n", fileCount)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
