package contacts

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var (
		firstName string
		lastName  string
		email     string
		phone     string
		company   string
		jobTitle  string
		notes     string
	)

	cmd := &cobra.Command{
		Use:   "create [grant-id]",
		Short: "Create a new contact",
		Long: `Create a new contact in your address book.

Examples:
  # Create a contact with basic info
  nylas contacts create --first-name "John" --last-name "Doe" --email "john@example.com"

  # Create a contact with work info
  nylas contacts create --first-name "Jane" --last-name "Smith" \
    --email "jane@company.com" --phone "+1-555-123-4567" \
    --company "Acme Corp" --job-title "Engineer"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if firstName == "" && lastName == "" && email == "" {
				return common.NewUserError(
					"at least one of --first-name, --last-name, or --email is required",
					"Provide contact information, e.g.: --first-name 'John' --email 'john@example.com'",
				)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := getGrantID(args)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.CreateContactRequest{
				GivenName:   firstName,
				Surname:     lastName,
				CompanyName: company,
				JobTitle:    jobTitle,
				Notes:       notes,
			}

			if email != "" {
				req.Emails = []domain.ContactEmail{{Email: email, Type: "work"}}
			}
			if phone != "" {
				req.PhoneNumbers = []domain.ContactPhone{{Number: phone, Type: "mobile"}}
			}

			contact, err := common.RunWithSpinnerResult("Creating contact...", func() (*domain.Contact, error) {
				return client.CreateContact(ctx, grantID, req)
			})
			if err != nil {
				return common.WrapCreateError("contact", err)
			}

			fmt.Printf("%s Contact created successfully!\n\n", common.Green.Sprint("✓"))
			fmt.Printf("Name: %s\n", contact.DisplayName())
			if contact.PrimaryEmail() != "" {
				fmt.Printf("Email: %s\n", contact.PrimaryEmail())
			}
			fmt.Printf("ID: %s\n", contact.ID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&firstName, "first-name", "f", "", "First name")
	cmd.Flags().StringVarP(&lastName, "last-name", "l", "", "Last name")
	cmd.Flags().StringVarP(&email, "email", "e", "", "Email address")
	cmd.Flags().StringVarP(&phone, "phone", "p", "", "Phone number")
	cmd.Flags().StringVarP(&company, "company", "c", "", "Company name")
	cmd.Flags().StringVarP(&jobTitle, "job-title", "j", "", "Job title")
	cmd.Flags().StringVarP(&notes, "notes", "n", "", "Notes about the contact")

	return cmd
}
