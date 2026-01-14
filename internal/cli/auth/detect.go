package auth

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newDetectCmd() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "detect <email>",
		Short: "Detect provider from email address",
		Long: `Detect the authentication provider based on an email address.

This command analyzes the email domain and suggests the appropriate provider:
- Gmail addresses → google
- Google Workspace domains → google
- Outlook/Hotmail addresses → microsoft
- Microsoft 365 domains → microsoft
- iCloud addresses → icloud
- Yahoo addresses → yahoo
- Other domains → imap

Note: This is a client-side heuristic. For Google Workspace or Microsoft 365
custom domains, verify the provider with your IT administrator.`,
		Example: `  # Detect provider for Gmail
  nylas auth detect user@gmail.com

  # Detect provider for corporate email
  nylas auth detect john@company.com

  # Output as JSON
  nylas auth detect user@outlook.com --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := strings.ToLower(strings.TrimSpace(args[0]))

			if !strings.Contains(email, "@") {
				return common.NewInputError(fmt.Sprintf("invalid email address: %s", email))
			}

			parts := strings.Split(email, "@")
			if len(parts) != 2 {
				return common.NewInputError(fmt.Sprintf("invalid email address format: %s", email))
			}

			domainPart := parts[1]
			provider := detectProvider(domainPart)

			result := struct {
				Email    string `json:"email"`
				Domain   string `json:"domain"`
				Provider string `json:"provider"`
				Note     string `json:"note,omitempty"`
			}{
				Email:    email,
				Domain:   domainPart,
				Provider: string(provider),
			}

			// Add notes for specific cases
			switch provider {
			case domain.ProviderGoogle:
				if !isGoogleDomain(domainPart) {
					result.Note = "Custom domain detected. Verify this is a Google Workspace account."
				}
			case domain.ProviderMicrosoft:
				if !isMicrosoftDomain(domainPart) {
					result.Note = "Custom domain detected. Verify this is a Microsoft 365 account."
				}
			case domain.ProviderIMAP:
				result.Note = "Use IMAP for generic email providers. Configure IMAP/SMTP settings during authentication."
			}

			if outputJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			// Display as formatted text
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Email:    %s\n", result.Email)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Domain:   %s\n", result.Domain)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider: %s\n", result.Provider)

			if result.Note != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNote: %s\n", result.Note)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nTo authenticate:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  nylas auth login --provider %s\n", result.Provider)

			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}

// detectProvider determines the provider based on email domain
func detectProvider(domainStr string) domain.Provider {
	domainStr = strings.ToLower(domainStr)

	// Google domains
	if isGoogleDomain(domainStr) {
		return domain.ProviderGoogle
	}

	// Microsoft domains
	if isMicrosoftDomain(domainStr) {
		return domain.ProviderMicrosoft
	}

	// iCloud domains
	if isICloudDomain(domainStr) {
		return domain.Provider("icloud")
	}

	// Yahoo domains
	if isYahooDomain(domainStr) {
		return domain.Provider("yahoo")
	}

	// Default to IMAP for unknown domains
	return domain.ProviderIMAP
}

func isGoogleDomain(domain string) bool {
	googleDomains := []string{
		"gmail.com",
		"googlemail.com",
		"google.com",
	}

	for _, d := range googleDomains {
		if domain == d {
			return true
		}
	}
	return false
}

func isMicrosoftDomain(domain string) bool {
	microsoftDomains := []string{
		"outlook.com",
		"hotmail.com",
		"live.com",
		"msn.com",
		"outlook.co.uk",
		"outlook.fr",
		"outlook.de",
		"outlook.jp",
		"outlook.kr",
		"microsoft.com",
		"office365.com",
	}

	for _, d := range microsoftDomains {
		if domain == d || strings.HasSuffix(domain, "."+d) {
			return true
		}
	}
	return false
}

func isICloudDomain(domain string) bool {
	icloudDomains := []string{
		"icloud.com",
		"me.com",
		"mac.com",
	}

	for _, d := range icloudDomains {
		if domain == d {
			return true
		}
	}
	return false
}

func isYahooDomain(domain string) bool {
	yahooDomains := []string{
		"yahoo.com",
		"yahoo.co.uk",
		"yahoo.fr",
		"yahoo.de",
		"yahoo.jp",
		"yahoo.co.jp",
		"ymail.com",
		"rocketmail.com",
	}

	for _, d := range yahooDomains {
		if domain == d || strings.HasSuffix(domain, "."+d) {
			return true
		}
	}
	return false
}
