// Package audit provides CLI commands for audit logging.
package audit

import (
	"github.com/spf13/cobra"
)

// NewAuditCmd creates the audit command with all subcommands.
func NewAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Manage audit logging",
		Long: `Audit logging records command execution history for compliance and debugging.

Initialize audit logging first, then enable it to start recording:
  nylas audit init      # Configure audit logging
  nylas audit logs enable   # Start recording

Use 'nylas audit logs show' to view command history with Nylas request IDs
for API traceability.`,
		Example: `  # Initialize with defaults
  nylas audit init --enable

  # Show recent commands
  nylas audit logs show

  # Filter by command
  nylas audit logs show --command email

  # Find by Nylas request ID
  nylas audit logs show --request-id req_abc123

  # View statistics
  nylas audit logs summary --days 7

  # Export logs
  nylas audit export --output audit.json`,
	}

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newExportCmd())

	return cmd
}
