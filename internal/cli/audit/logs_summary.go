package audit

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/nylas/cli/internal/adapters/audit"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newSummaryCmd() *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Show audit log statistics",
		Long:  `Display aggregate statistics for audit logs over a specified period.`,
		Example: `  # Summary for last 7 days (default)
  nylas audit logs summary

  # Summary for last 30 days
  nylas audit logs summary --days 30`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := audit.NewFileStore("")
			if err != nil {
				return fmt.Errorf("open audit store: %w", err)
			}

			cfg, err := store.GetConfig()
			if err != nil || !cfg.Initialized {
				return fmt.Errorf("audit logging not initialized. Run: nylas audit init")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			summary, err := store.Summary(ctx, days)
			if err != nil {
				return fmt.Errorf("generate summary: %w", err)
			}

			if summary.TotalCommands == 0 {
				fmt.Printf("No audit entries found in the last %d days.\n", days)
				return nil
			}

			// Header
			_, _ = common.Bold.Printf("Audit Log Summary (Last %d days)\n", days)
			fmt.Println()

			// Total stats
			fmt.Printf("Total Commands:  %d\n", summary.TotalCommands)
			_, _ = common.Green.Printf("  ✓ Success:     %d (%.0f%%)\n",
				summary.SuccessCount, summary.SuccessPercent)
			if summary.ErrorCount > 0 {
				_, _ = common.Red.Printf("  ✗ Errors:      %d (%.0f%%)\n",
					summary.ErrorCount, 100-summary.SuccessPercent)
			} else {
				fmt.Printf("  ✗ Errors:      0\n")
			}
			fmt.Println()

			// Most used commands
			if len(summary.CommandCounts) > 0 {
				fmt.Println("Most Used:")
				printTopItems(summary.CommandCounts, 5)
				fmt.Println()
			}

			// Accounts
			if len(summary.AccountCounts) > 0 {
				fmt.Println("Accounts:")
				printTopItems(summary.AccountCounts, 5)
				fmt.Println()
			}

			// Invoker breakdown
			if len(summary.InvokerCounts) > 0 {
				fmt.Println("Invoker Breakdown:")
				printTopItems(summary.InvokerCounts, 5)
				fmt.Println()
			}

			// API statistics
			if summary.TotalAPICalls > 0 {
				fmt.Println("API Statistics:")
				fmt.Printf("  Total API calls:  %d\n", summary.TotalAPICalls)
				fmt.Printf("  Avg response time: %s\n", FormatDuration(summary.AvgResponseTime))
				if summary.APIErrorRate > 0 {
					_, _ = common.Yellow.Printf("  Error rate:       %.1f%%\n", summary.APIErrorRate)
				} else {
					fmt.Printf("  Error rate:       0%%\n")
				}
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&days, "days", 7, "Number of days to include in summary")

	return cmd
}

func printTopItems(counts map[string]int, limit int) {
	// Sort by count descending
	type item struct {
		name  string
		count int
	}

	items := make([]item, 0, len(counts))
	for name, count := range counts {
		items = append(items, item{name, count})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].count > items[j].count
	})

	// Print top items
	for i, it := range items {
		if i >= limit {
			break
		}
		fmt.Printf("  %-20s %d\n", it.name, it.count)
	}
}
