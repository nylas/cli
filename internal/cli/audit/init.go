package audit

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/adapters/audit"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		path      string
		retention int
		maxSize   int
		format    string
		enable    bool
		noPrompt  bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize audit logging",
		Long: `Initialize audit logging with storage location, retention, and other options.

By default, runs interactively to configure settings. Use flags to skip prompts.`,
		Example: `  # Interactive setup
  nylas audit init

  # Non-interactive with defaults and enable immediately
  nylas audit init --enable

  # Custom configuration
  nylas audit init --path /custom/path --retention 30 --max-size 50 --enable`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := audit.NewFileStore(path)
			if err != nil {
				return fmt.Errorf("create audit store: %w", err)
			}

			cfg, err := store.GetConfig()
			if err != nil {
				cfg = domain.DefaultAuditConfig()
			}

			// Set path
			if path != "" {
				cfg.Path = path
			} else if cfg.Path == "" {
				cfg.Path = audit.DefaultAuditPath()
			}

			// Interactive mode if no flags set
			if !noPrompt && !cmd.Flags().Changed("retention") && !cmd.Flags().Changed("max-size") {
				cfg = runInteractiveSetup(cfg)
			} else {
				// Apply flag values
				if cmd.Flags().Changed("retention") {
					cfg.RetentionDays = retention
				}
				if cmd.Flags().Changed("max-size") {
					cfg.MaxSizeMB = maxSize
				}
				if cmd.Flags().Changed("format") {
					cfg.Format = format
				}
				if cmd.Flags().Changed("enable") {
					cfg.Enabled = enable
				}
			}

			cfg.Initialized = true

			// Create store with the configured path
			store, err = audit.NewFileStore(cfg.Path)
			if err != nil {
				return fmt.Errorf("create audit store: %w", err)
			}

			if err := store.SaveConfig(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			// Print summary
			fmt.Println()
			_, _ = common.Green.Println("✓ Audit logging initialized" + enabledSuffix(cfg.Enabled))
			fmt.Printf("  Path: %s\n", cfg.Path)
			fmt.Printf("  Retention: %d days\n", cfg.RetentionDays)
			fmt.Printf("  Max size: %d MB\n", cfg.MaxSizeMB)
			fmt.Printf("  Request IDs: %s\n", enabledDisabled(cfg.LogRequestID))

			if !cfg.Enabled {
				fmt.Println()
				_, _ = common.Yellow.Println("Tip: Enable logging with: nylas audit logs enable")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "Custom log directory (default: ~/.config/nylas/audit)")
	cmd.Flags().IntVar(&retention, "retention", 90, "Log retention period in days")
	cmd.Flags().IntVar(&maxSize, "max-size", 100, "Max storage size in MB")
	cmd.Flags().StringVar(&format, "format", "jsonl", "Log format: jsonl, json")
	cmd.Flags().BoolVar(&enable, "enable", false, "Enable audit logging immediately")
	cmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "Skip interactive prompts")

	return cmd
}

func runInteractiveSetup(cfg *domain.AuditConfig) *domain.AuditConfig {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Initializing Nylas CLI Audit Logging")
	fmt.Println()

	// Path
	fmt.Printf("? Log directory [%s]: ", cfg.Path)
	if input := readLine(reader); input != "" {
		cfg.Path = input
	}

	// Retention
	fmt.Printf("? Retention period in days [%d]: ", cfg.RetentionDays)
	if input := readLine(reader); input != "" {
		if n, err := strconv.Atoi(input); err == nil && n > 0 {
			cfg.RetentionDays = n
		}
	}

	// Max size
	fmt.Printf("? Max storage size in MB [%d]: ", cfg.MaxSizeMB)
	if input := readLine(reader); input != "" {
		if n, err := strconv.Atoi(input); err == nil && n > 0 {
			cfg.MaxSizeMB = n
		}
	}

	// Log request IDs
	fmt.Printf("? Log Nylas request IDs? [Y/n]: ")
	if input := strings.ToLower(readLine(reader)); input == "n" || input == "no" {
		cfg.LogRequestID = false
	} else {
		cfg.LogRequestID = true
	}

	// Compress old files
	fmt.Printf("? Compress old log files? [y/N]: ")
	if input := strings.ToLower(readLine(reader)); input == "y" || input == "yes" {
		cfg.CompressOld = true
	}

	// Enable now
	fmt.Printf("? Enable audit logging now? [Y/n]: ")
	if input := strings.ToLower(readLine(reader)); input == "n" || input == "no" {
		cfg.Enabled = false
	} else {
		cfg.Enabled = true
	}

	return cfg
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func enabledSuffix(enabled bool) string {
	if enabled {
		return " and enabled"
	}
	return ""
}

func enabledDisabled(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}
