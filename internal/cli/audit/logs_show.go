package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/adapters/audit"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var (
		limit         int
		since         string
		until         string
		command       string
		status        string
		grantID       string
		requestID     string
		invoker       string
		invokerSource string
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show recent audit entries",
		Long: `Display recent audit log entries with optional filtering.

When filtering by request ID, shows detailed information about that specific entry.`,
		Example: `  # Show last 20 entries
  nylas audit logs show

  # Show last 50 entries
  nylas audit logs show --limit 50

  # Filter by command prefix
  nylas audit logs show --command email

  # Filter by status
  nylas audit logs show --status error

  # Filter by date range
  nylas audit logs show --since 2024-01-01 --until 2024-01-31

  # Find by Nylas request ID
  nylas audit logs show --request-id req_abc123

  # Filter by user
  nylas audit logs show --invoker alice

  # Filter by source platform (claude-code, github-actions, terminal, etc.)
  nylas audit logs show --source github-actions`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := audit.NewFileStore("")
			if err != nil {
				return fmt.Errorf("open audit store: %w", err)
			}

			cfg, err := store.GetConfig()
			if err != nil || !cfg.Initialized {
				return fmt.Errorf("audit logging not initialized. Run: nylas audit init")
			}

			// Build query options
			opts := &domain.AuditQueryOptions{
				Limit:         limit,
				Command:       command,
				Status:        status,
				GrantID:       grantID,
				RequestID:     requestID,
				Invoker:       invoker,
				InvokerSource: invokerSource,
			}

			// Parse dates
			if since != "" {
				t, err := parseDate(since)
				if err != nil {
					return fmt.Errorf("invalid --since date: %w", err)
				}
				opts.Since = t
			}
			if until != "" {
				t, err := parseDate(until)
				if err != nil {
					return fmt.Errorf("invalid --until date: %w", err)
				}
				opts.Until = t.Add(24 * time.Hour) // Include full day
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			entries, err := store.Query(ctx, opts)
			if err != nil {
				return fmt.Errorf("query logs: %w", err)
			}

			if len(entries) == 0 {
				fmt.Println("No audit entries found.")
				return nil
			}

			// If filtering by request ID, show detailed view
			if requestID != "" && len(entries) == 1 {
				return showDetailedEntry(entries[0])
			}

			// Table view
			return showEntryTable(cmd, entries)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Number of entries to show")
	cmd.Flags().StringVar(&since, "since", "", "Show entries after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&until, "until", "", "Show entries before this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&command, "command", "", "Filter by command prefix")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (success/error)")
	cmd.Flags().StringVar(&grantID, "grant", "", "Filter by grant ID")
	cmd.Flags().StringVar(&requestID, "request-id", "", "Filter by Nylas request ID")
	cmd.Flags().StringVar(&invoker, "invoker", "", "Filter by username")
	cmd.Flags().StringVar(&invokerSource, "source", "", "Filter by source platform (claude-code, github-actions, terminal)")

	return cmd
}

func showEntryTable(cmd *cobra.Command, entries []domain.AuditEntry) error {
	columns := []ports.Column{
		{Header: "TIMESTAMP", Field: "Timestamp", Width: 19},
		{Header: "COMMAND", Field: "Command", Width: 16},
		{Header: "GRANT", Field: "GrantDisplay", Width: 12},
		{Header: "INVOKER", Field: "InvokerDisplay", Width: 12},
		{Header: "SOURCE", Field: "SourceDisplay", Width: 12},
		{Header: "STATUS", Field: "Status", Width: 8},
		{Header: "DURATION", Field: "Duration", Width: 10},
	}

	type row struct {
		Timestamp  string `json:"timestamp"`
		Command    string `json:"command"`
		Grant      string `json:"grant,omitempty"`
		Invoker    string `json:"invoker"`
		Source     string `json:"source"`
		Status     string `json:"status"`
		Duration   string `json:"duration"`
		RequestID  string `json:"request_id,omitempty"`
		HTTPStatus int    `json:"http_status,omitempty"`

		// Table display fields (not in JSON)
		GrantDisplay   string `json:"-"`
		InvokerDisplay string `json:"-"`
		SourceDisplay  string `json:"-"`
	}

	rows := make([]row, len(entries))
	for i, e := range entries {
		rows[i] = row{
			Timestamp:      e.Timestamp.Format("2006-01-02 15:04:05"),
			Command:        e.Command,
			Grant:          orDash(e.GrantID),
			Invoker:        orDash(e.Invoker),
			Source:         orDash(e.InvokerSource),
			Status:         string(e.Status),
			Duration:       FormatDuration(e.Duration),
			RequestID:      e.RequestID,
			HTTPStatus:     e.HTTPStatus,
			GrantDisplay:   truncate(orDash(e.GrantID), 12),
			InvokerDisplay: truncate(orDash(e.Invoker), 12),
			SourceDisplay:  truncate(orDash(e.InvokerSource), 12),
		}
	}

	return common.WriteListWithColumns(cmd, rows, columns)
}

func showDetailedEntry(entry domain.AuditEntry) error {
	fmt.Println("Entry Details")
	fmt.Println()
	fmt.Printf("  ID:           %s\n", entry.ID)
	fmt.Printf("  Timestamp:    %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Command:      %s\n", entry.Command)

	if len(entry.Args) > 0 {
		fmt.Printf("  Arguments:    %s\n", strings.Join(entry.Args, " "))
	}

	if entry.GrantEmail != "" {
		fmt.Printf("  Account:      %s\n", entry.GrantEmail)
	} else if entry.GrantID != "" {
		fmt.Printf("  Grant ID:     %s\n", entry.GrantID)
	}

	fmt.Printf("  Status:       %s\n", entry.Status)
	fmt.Printf("  Duration:     %s\n", FormatDuration(entry.Duration))

	if entry.Error != "" {
		fmt.Println()
		_, _ = common.Red.Printf("  Error: %s\n", entry.Error)
	}

	// Invoker details
	if entry.Invoker != "" || entry.InvokerSource != "" {
		fmt.Println()
		fmt.Println("  Invoker Details:")
		if entry.Invoker != "" {
			fmt.Printf("    User:        %s\n", entry.Invoker)
		}
		if entry.InvokerSource != "" {
			fmt.Printf("    Source:      %s\n", entry.InvokerSource)
		}
	}

	if entry.RequestID != "" || entry.HTTPStatus > 0 {
		fmt.Println()
		fmt.Println("  API Details:")
		if entry.RequestID != "" {
			fmt.Printf("    Request ID:  %s\n", entry.RequestID)
		}
		if entry.HTTPStatus > 0 {
			fmt.Printf("    HTTP Status: %d\n", entry.HTTPStatus)
		}
	}

	return nil
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// orDash returns "-" if s is empty, otherwise returns s.
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
