package auth

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
)

func newScopesCmd() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "scopes [grant-id]",
		Short: "Show OAuth scopes for a grant",
		Long: `Display the OAuth scopes (permissions) for an authenticated grant.

Scopes define what data and operations are permitted for a grant:
- Email scopes: Read messages, send emails, manage drafts
- Calendar scopes: Read events, create/update events
- Contacts scopes: Read/write contact information

If no grant ID is provided, shows scopes for the currently active grant.`,
		Example: `  # Show scopes for current grant
  nylas auth scopes

  # Show scopes for specific grant
  nylas auth scopes grant-123

  # Output as JSON
  nylas auth scopes --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.CreateContext()
			defer cancel()

			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			// Determine grant ID
			var grantID string
			if len(args) > 0 {
				grantID = args[0]
			} else {
				configStore := config.NewDefaultFileStore()
				cfg, err := configStore.Load()
				if err != nil {
					return fmt.Errorf("no grant ID provided and no configuration found")
				}
				grantID = cfg.DefaultGrant
				if grantID == "" {
					return fmt.Errorf("no grant ID provided and no default grant configured")
				}
			}

			// Get grant details
			grant, err := client.GetGrant(ctx, grantID)
			if err != nil {
				return common.WrapGetError("grant scopes", err)
			}

			result := struct {
				GrantID  string   `json:"grant_id"`
				Email    string   `json:"email"`
				Provider string   `json:"provider"`
				Status   string   `json:"status"`
				Scopes   []string `json:"scopes"`
			}{
				GrantID:  grant.ID,
				Email:    grant.Email,
				Provider: string(grant.Provider),
				Status:   grant.GrantStatus,
				Scopes:   grant.Scope,
			}

			if outputJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			// Display as formatted text
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Grant ID:  %s\n", result.GrantID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Email:     %s\n", result.Email)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider:  %s\n", result.Provider)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status:    %s\n", result.Status)
			_, _ = fmt.Fprintln(cmd.OutOrStdout())

			if len(result.Scopes) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No scopes configured.")
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "OAuth Scopes (%d):\n", len(result.Scopes))
			for i, scope := range result.Scopes {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s\n", i+1, scope)
				if description := describeScopeCategory(scope); description != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "     %s\n", description)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}

// describeScopeCategory provides a brief description for common scope patterns
func describeScopeCategory(scope string) string {
	// Google scopes
	if strings.Contains(scope, "gmail") {
		if strings.Contains(scope, "readonly") {
			return "→ Read-only access to Gmail"
		}
		if strings.Contains(scope, "send") {
			return "→ Send email via Gmail"
		}
		if strings.Contains(scope, "modify") || strings.Contains(scope, "compose") {
			return "→ Read and modify Gmail messages"
		}
		return "→ Gmail access"
	}

	if strings.Contains(scope, "calendar") {
		if strings.Contains(scope, "readonly") {
			return "→ Read-only access to Google Calendar"
		}
		if strings.Contains(scope, "events") {
			return "→ Manage calendar events"
		}
		return "→ Calendar access"
	}

	if strings.Contains(scope, "contacts") {
		if strings.Contains(scope, "readonly") {
			return "→ Read-only access to contacts"
		}
		return "→ Manage contacts"
	}

	// Microsoft scopes
	if strings.Contains(scope, "Mail.") {
		if strings.Contains(scope, "Read") {
			return "→ Read email messages"
		}
		if strings.Contains(scope, "Send") {
			return "→ Send email messages"
		}
		return "→ Email access"
	}

	if strings.Contains(scope, "Calendars.") {
		if strings.Contains(scope, "Read") {
			return "→ Read calendar events"
		}
		return "→ Manage calendar events"
	}

	if strings.Contains(scope, "Contacts.") {
		if strings.Contains(scope, "Read") {
			return "→ Read contacts"
		}
		return "→ Manage contacts"
	}

	if strings.Contains(scope, "User.Read") {
		return "→ Read user profile information"
	}

	return ""
}
