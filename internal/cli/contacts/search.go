package contacts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func newSearchCmd() *cobra.Command {
	var (
		companyName string
		email       string
		phoneNumber string
		source      string
		group       string
		hasEmail    bool
		limit       int
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search contacts with advanced filters",
		Long: `Search contacts using various filters like company name, email, phone number, and source.

Advanced Search Options:
  --company: Filter by company name (searches in company_name field)
  --email: Filter by email address
  --phone: Filter by phone number
  --source: Filter by contact source (address_book, inbox, domain)
  --group: Filter by contact group ID
  --has-email: Only show contacts with email addresses

Note: Company name filtering searches the company_name field. For more advanced
text searches, use the regular list command with additional filtering.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				params := &domain.ContactQueryParams{
					Limit:       limit,
					Email:       email,
					PhoneNumber: phoneNumber,
					Source:      source,
					Group:       group,
				}

				contacts, err := client.GetContacts(ctx, grantID, params)
				if err != nil {
					return struct{}{}, common.WrapSearchError("contacts", err)
				}

				// Apply client-side filters
				var filtered []domain.Contact
				for _, contact := range contacts {
					// Filter by company name (case-insensitive)
					if companyName != "" && !strings.Contains(strings.ToLower(contact.CompanyName), strings.ToLower(companyName)) {
						continue
					}

					// Filter by has-email
					if hasEmail && len(contact.Emails) == 0 {
						continue
					}

					filtered = append(filtered, contact)
				}

				if common.IsJSON(cmd) {
					encoder := json.NewEncoder(os.Stdout)
					encoder.SetIndent("", "  ")
					return struct{}{}, encoder.Encode(filtered)
				}

				// Print results as table
				table := common.NewTable("ID", "Name", "Email", "Company", "Job Title")
				for _, contact := range filtered {
					name := contact.DisplayName()
					email := contact.PrimaryEmail()
					company := contact.CompanyName
					if company == "" {
						company = "-"
					}
					jobTitle := contact.JobTitle
					if jobTitle == "" {
						jobTitle = "-"
					}
					table.AddRow(contact.ID, name, email, company, jobTitle)
				}
				table.Render()

				fmt.Printf("\nFound %d contacts\n", len(filtered))

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&companyName, "company", "", "Filter by company name (partial match)")
	cmd.Flags().StringVar(&email, "email", "", "Filter by email address")
	cmd.Flags().StringVar(&phoneNumber, "phone", "", "Filter by phone number")
	cmd.Flags().StringVar(&source, "source", "", "Filter by source (address_book, inbox, domain)")
	cmd.Flags().StringVar(&group, "group", "", "Filter by contact group ID")
	cmd.Flags().BoolVar(&hasEmail, "has-email", false, "Only show contacts with email addresses")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of contacts to retrieve")

	return cmd
}
