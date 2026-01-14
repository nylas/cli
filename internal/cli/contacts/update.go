package contacts

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		givenName   string
		middleName  string
		surname     string
		suffix      string
		nickname    string
		birthday    string
		companyName string
		jobTitle    string
		managerName string
		notes       string
		emails      []string
		phones      []string
	)

	cmd := &cobra.Command{
		Use:   "update <contact-id> [grant-id]",
		Short: "Update a contact",
		Long: `Update an existing contact's information.

Examples:
  # Update contact name
  nylas contacts update <contact-id> --given-name "John" --surname "Smith"

  # Update contact company info
  nylas contacts update <contact-id> --company "Acme Inc" --job-title "Engineer"

  # Update contact email
  nylas contacts update <contact-id> --email "new@example.com"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.UpdateContactRequest{}

			if cmd.Flags().Changed("given-name") {
				req.GivenName = &givenName
			}
			if cmd.Flags().Changed("middle-name") {
				req.MiddleName = &middleName
			}
			if cmd.Flags().Changed("surname") {
				req.Surname = &surname
			}
			if cmd.Flags().Changed("suffix") {
				req.Suffix = &suffix
			}
			if cmd.Flags().Changed("nickname") {
				req.Nickname = &nickname
			}
			if cmd.Flags().Changed("birthday") {
				req.Birthday = &birthday
			}
			if cmd.Flags().Changed("company") {
				req.CompanyName = &companyName
			}
			if cmd.Flags().Changed("job-title") {
				req.JobTitle = &jobTitle
			}
			if cmd.Flags().Changed("manager") {
				req.ManagerName = &managerName
			}
			if cmd.Flags().Changed("notes") {
				req.Notes = &notes
			}

			// Handle emails
			if len(emails) > 0 {
				for _, email := range emails {
					req.Emails = append(req.Emails, domain.ContactEmail{
						Email: email,
						Type:  "work",
					})
				}
			}

			// Handle phone numbers
			if len(phones) > 0 {
				for _, phone := range phones {
					req.PhoneNumbers = append(req.PhoneNumbers, domain.ContactPhone{
						Number: phone,
						Type:   "work",
					})
				}
			}

			contact, err := client.UpdateContact(ctx, grantID, contactID, req)
			if err != nil {
				return common.WrapUpdateError("contact", err)
			}

			fmt.Printf("%s Contact updated successfully!\n\n", common.Green.Sprint("✓"))
			fmt.Printf("Name: %s\n", contact.DisplayName())
			if len(contact.Emails) > 0 {
				fmt.Printf("Email: %s\n", contact.Emails[0].Email)
			}
			fmt.Printf("ID: %s\n", contact.ID)

			return nil
		},
	}

	cmd.Flags().StringVar(&givenName, "given-name", "", "First name")
	cmd.Flags().StringVar(&middleName, "middle-name", "", "Middle name")
	cmd.Flags().StringVar(&surname, "surname", "", "Last name")
	cmd.Flags().StringVar(&suffix, "suffix", "", "Name suffix (e.g., Jr., Sr.)")
	cmd.Flags().StringVar(&nickname, "nickname", "", "Nickname")
	cmd.Flags().StringVar(&birthday, "birthday", "", "Birthday (YYYY-MM-DD)")
	cmd.Flags().StringVar(&companyName, "company", "", "Company name")
	cmd.Flags().StringVar(&jobTitle, "job-title", "", "Job title")
	cmd.Flags().StringVar(&managerName, "manager", "", "Manager name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringArrayVar(&emails, "email", nil, "Email address (can be used multiple times)")
	cmd.Flags().StringArrayVar(&phones, "phone", nil, "Phone number (can be used multiple times)")

	return cmd
}
