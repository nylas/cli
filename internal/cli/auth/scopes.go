package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
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

			client, err := getScopesClient()
			if err != nil {
				return err
			}

			// Determine grant ID
			var grantID string
			if len(args) > 0 {
				grantID = args[0]
			} else {
				secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
				if err != nil {
					return fmt.Errorf("no grant ID provided and failed to access keyring: %w", err)
				}
				grantStore := keyring.NewGrantStore(secretStore)
				defaultID, err := grantStore.GetDefaultGrant()
				if err != nil {
					return fmt.Errorf("no grant ID provided and no default grant configured")
				}
				grantID = defaultID
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
	if contains(scope, "gmail") {
		if contains(scope, "readonly") {
			return "→ Read-only access to Gmail"
		}
		if contains(scope, "send") {
			return "→ Send email via Gmail"
		}
		if contains(scope, "modify") || contains(scope, "compose") {
			return "→ Read and modify Gmail messages"
		}
		return "→ Gmail access"
	}

	if contains(scope, "calendar") {
		if contains(scope, "readonly") {
			return "→ Read-only access to Google Calendar"
		}
		if contains(scope, "events") {
			return "→ Manage calendar events"
		}
		return "→ Calendar access"
	}

	if contains(scope, "contacts") {
		if contains(scope, "readonly") {
			return "→ Read-only access to contacts"
		}
		return "→ Manage contacts"
	}

	// Microsoft scopes
	if contains(scope, "Mail.") {
		if contains(scope, "Read") {
			return "→ Read email messages"
		}
		if contains(scope, "Send") {
			return "→ Send email messages"
		}
		return "→ Email access"
	}

	if contains(scope, "Calendars.") {
		if contains(scope, "Read") {
			return "→ Read calendar events"
		}
		return "→ Manage calendar events"
	}

	if contains(scope, "Contacts.") {
		if contains(scope, "Read") {
			return "→ Read contacts"
		}
		return "→ Manage contacts"
	}

	if contains(scope, "User.Read") {
		return "→ Read user profile information"
	}

	return ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			s[max(0, len(s)-len(substr)-1):len(s)-len(substr)] == substr ||
			findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getScopesClient() (ports.NylasClient, error) {
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err != nil {
		cfg = &domain.Config{Region: "us"}
	}

	// Check environment variables first (highest priority)
	apiKey := os.Getenv("NYLAS_API_KEY")
	clientID := os.Getenv("NYLAS_CLIENT_ID")
	clientSecret := os.Getenv("NYLAS_CLIENT_SECRET")

	// If API key not in env, try keyring/file store
	if apiKey == "" {
		secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
		if err == nil {
			apiKey, _ = secretStore.Get(ports.KeyAPIKey)
			if clientID == "" {
				clientID, _ = secretStore.Get(ports.KeyClientID)
			}
			if clientSecret == "" {
				clientSecret, _ = secretStore.Get(ports.KeyClientSecret)
			}
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key not configured. Set NYLAS_API_KEY environment variable or run 'nylas auth config'")
	}

	c := nylas.NewHTTPClient()
	c.SetRegion(cfg.Region)
	c.SetCredentials(clientID, clientSecret, apiKey)

	return c, nil
}
