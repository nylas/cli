package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nylas/cli/internal/adapters/audit"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var (
		output string
		format string
		since  string
		until  string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export audit logs to a file",
		Long: `Export audit logs to JSON or CSV format.

If no output file is specified, exports to stdout.`,
		Example: `  # Export to JSON file
  nylas audit export --output audit.json

  # Export to CSV
  nylas audit export --output audit.csv --format csv

  # Export with date filter
  nylas audit export --since 2024-01-01 --until 2024-01-31 --output january.json`,
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
				Limit: limit,
			}

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
				opts.Until = t.Add(24 * time.Hour)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			entries, err := store.Query(ctx, opts)
			if err != nil {
				return fmt.Errorf("query logs: %w", err)
			}

			if len(entries) == 0 {
				fmt.Println("No entries to export.")
				return nil
			}

			// Determine format from output file extension if not specified
			if format == "" && output != "" {
				ext := strings.ToLower(filepath.Ext(output))
				if ext == ".csv" {
					format = "csv"
				} else {
					format = "json"
				}
			}
			if format == "" {
				format = "json"
			}

			// Get output writer
			var w *os.File
			if output == "" {
				w = os.Stdout
			} else {
				w, err = os.Create(output)
				if err != nil {
					return fmt.Errorf("create output file: %w", err)
				}
				defer func() { _ = w.Close() }()
			}

			// Export
			switch format {
			case "json":
				err = exportJSON(w, entries)
			case "csv":
				err = exportCSV(w, entries)
			default:
				return fmt.Errorf("unsupported format: %s (use json or csv)", format)
			}

			if err != nil {
				return fmt.Errorf("export: %w", err)
			}

			if output != "" {
				_, _ = common.Green.Printf("✓ Exported %d entries to %s\n", len(entries), output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().StringVar(&format, "format", "", "Output format: json, csv (default: auto-detect from extension)")
	cmd.Flags().StringVar(&since, "since", "", "Export entries after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&until, "until", "", "Export entries before this date (YYYY-MM-DD)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 10000, "Maximum entries to export")

	return cmd
}

func exportJSON(w *os.File, entries []domain.AuditEntry) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

func exportCSV(w *os.File, entries []domain.AuditEntry) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	header := []string{
		"id", "timestamp", "command", "args", "grant_id", "grant_email",
		"invoker", "invoker_source", "status", "duration_ms", "error",
		"request_id", "http_status",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Rows
	for _, e := range entries {
		row := []string{
			e.ID,
			e.Timestamp.Format(time.RFC3339),
			e.Command,
			strings.Join(e.Args, " "),
			e.GrantID,
			e.GrantEmail,
			e.Invoker,
			e.InvokerSource,
			string(e.Status),
			fmt.Sprintf("%d", e.Duration.Milliseconds()),
			e.Error,
			e.RequestID,
			fmt.Sprintf("%d", e.HTTPStatus),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
