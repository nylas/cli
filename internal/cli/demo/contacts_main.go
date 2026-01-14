package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
)

func newDemoContactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Explore contacts features with sample data",
		Long:  "Demo contacts commands showing sample contacts and simulated operations.",
	}

	// Basic CRUD
	cmd.AddCommand(newDemoContactsListCmd())
	cmd.AddCommand(newDemoContactsShowCmd())
	cmd.AddCommand(newDemoContactsCreateCmd())
	cmd.AddCommand(newDemoContactsUpdateCmd())
	cmd.AddCommand(newDemoContactsDeleteCmd())

	// Search
	cmd.AddCommand(newDemoContactsSearchCmd())

	// Groups
	cmd.AddCommand(newDemoContactsGroupsCmd())

	// Photo
	cmd.AddCommand(newDemoContactsPhotoCmd())

	// Sync
	cmd.AddCommand(newDemoContactsSyncCmd())

	return cmd
}

// ============================================================================
// BASIC CRUD COMMANDS
// ============================================================================

// newDemoContactsListCmd lists sample contacts.
func newDemoContactsListCmd() *cobra.Command {
	var limit int
	var showID bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sample contacts",
		Long:  "Display a list of realistic sample contacts.",
		Example: `  # List sample contacts
  nylas demo contacts list

  # List with IDs shown
  nylas demo contacts list --id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			contacts, err := client.GetContacts(ctx, "demo-grant", nil)
			if err != nil {
				return common.WrapListError("contacts", err)
			}

			if limit > 0 && limit < len(contacts) {
				contacts = contacts[:limit]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("👤 Demo Mode - Sample Contacts"))
			fmt.Println(common.Dim.Sprint("These are sample contacts for demonstration purposes."))
			fmt.Println()
			fmt.Printf("Found %d contacts:\n\n", len(contacts))

			for _, contact := range contacts {
				printDemoContact(contact, showID)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To connect your real contacts: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of contacts to show")
	cmd.Flags().BoolVar(&showID, "id", false, "Show contact IDs")

	return cmd
}

// newDemoContactsShowCmd shows a sample contact.
func newDemoContactsShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show [contact-id]",
		Aliases: []string{"read"},
		Short:   "Show a sample contact",
		Long:    "Display a sample contact to see the full contact format.",
		Example: `  # Show first sample contact
  nylas demo contacts show

  # Show specific contact
  nylas demo contacts show contact-001`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			contactID := "contact-001"
			if len(args) > 0 {
				contactID = args[0]
			}

			contact, err := client.GetContact(ctx, "demo-grant", contactID)
			if err != nil {
				return common.WrapGetError("contact", err)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("👤 Demo Mode - Sample Contact"))
			fmt.Println()
			printDemoContactFull(*contact)

			fmt.Println(common.Dim.Sprint("To connect your real contacts: nylas auth login"))

			return nil
		},
	}

	return cmd
}

// newDemoContactsCreateCmd simulates creating a contact.
func newDemoContactsCreateCmd() *cobra.Command {
	var firstName string
	var lastName string
	var email string
	var phone string
	var company string
	var jobTitle string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Simulate creating a contact",
		Long:  "Simulate creating a contact to see how the create command works.",
		Example: `  # Create a basic contact
  nylas demo contacts create --first-name "John" --last-name "Doe" --email "john@example.com"

  # Create with company info
  nylas demo contacts create --first-name "Jane" --last-name "Smith" --email "jane@company.com" --company "Acme Inc" --title "Engineer"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if firstName == "" {
				firstName = "Demo"
			}
			if lastName == "" {
				lastName = "Contact"
			}
			if email == "" {
				email = "demo@example.com"
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("👤 Demo Mode - Simulated Contact Creation"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Printf("Name:     %s %s\n", firstName, lastName)
			fmt.Printf("Email:    %s\n", email)
			if phone != "" {
				fmt.Printf("Phone:    %s\n", phone)
			}
			if company != "" {
				fmt.Printf("Company:  %s\n", company)
			}
			if jobTitle != "" {
				fmt.Printf("Title:    %s\n", jobTitle)
			}
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()
			_, _ = common.Green.Println("✓ Contact would be created (demo mode - no actual contact created)")
			_, _ = common.Dim.Printf("  Contact ID: contact-demo-%d\n", time.Now().Unix())
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To create real contacts, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&firstName, "first-name", "", "First name")
	cmd.Flags().StringVar(&lastName, "last-name", "", "Last name")
	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	cmd.Flags().StringVar(&company, "company", "", "Company name")
	cmd.Flags().StringVar(&jobTitle, "title", "", "Job title")

	return cmd
}

// newDemoContactsUpdateCmd simulates updating a contact.
func newDemoContactsUpdateCmd() *cobra.Command {
	var email string
	var phone string
	var company string
	var jobTitle string

	cmd := &cobra.Command{
		Use:   "update [contact-id]",
		Short: "Simulate updating a contact",
		Long:  "Simulate updating a contact to see how the update command works.",
		Example: `  # Update contact email
  nylas demo contacts update contact-001 --email "newemail@example.com"

  # Update company info
  nylas demo contacts update contact-001 --company "New Company" --title "Senior Engineer"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID := "contact-demo-123"
			if len(args) > 0 {
				contactID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("👤 Demo Mode - Simulated Contact Update"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.Dim.Printf("Contact ID: %s\n", contactID)
			fmt.Println()
			_, _ = common.BoldWhite.Println("Changes:")
			if email != "" {
				fmt.Printf("  Email:   %s\n", email)
			}
			if phone != "" {
				fmt.Printf("  Phone:   %s\n", phone)
			}
			if company != "" {
				fmt.Printf("  Company: %s\n", company)
			}
			if jobTitle != "" {
				fmt.Printf("  Title:   %s\n", jobTitle)
			}
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()
			_, _ = common.Green.Println("✓ Contact would be updated (demo mode - no actual changes made)")
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To update real contacts, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "New email address")
	cmd.Flags().StringVar(&phone, "phone", "", "New phone number")
	cmd.Flags().StringVar(&company, "company", "", "New company name")
	cmd.Flags().StringVar(&jobTitle, "title", "", "New job title")

	return cmd
}

// newDemoContactsDeleteCmd simulates deleting a contact.
func newDemoContactsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [contact-id]",
		Short: "Simulate deleting a contact",
		Long:  "Simulate deleting a contact to see how the delete command works.",
		Example: `  # Delete a contact
  nylas demo contacts delete contact-001

  # Force delete without confirmation
  nylas demo contacts delete contact-001 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID := "contact-demo-123"
			if len(args) > 0 {
				contactID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("👤 Demo Mode - Simulated Contact Deletion"))
			fmt.Println()

			if !force {
				_, _ = common.Yellow.Println("⚠ Would prompt for confirmation in real mode")
			}

			fmt.Printf("Contact ID: %s\n", contactID)
			fmt.Println()
			_, _ = common.Green.Println("✓ Contact would be deleted (demo mode - no actual deletion)")
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To delete real contacts, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

// ============================================================================
// SEARCH COMMAND
// ============================================================================

// newDemoContactsSearchCmd simulates searching contacts.
func newDemoContactsSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search sample contacts",
		Long:  "Search through sample contacts by name, email, or company.",
		Example: `  # Search by name
  nylas demo contacts search "John"

  # Search by email domain
  nylas demo contacts search "@acme.com"

  # Search by company
  nylas demo contacts search "Acme"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := "John"
			if len(args) > 0 {
				query = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("👤 Demo Mode - Contact Search"))
			fmt.Println()
			fmt.Printf("Search query: %s\n\n", common.Cyan.Sprint(query))

			// Sample search results
			results := []struct {
				name    string
				email   string
				company string
			}{
				{"John Smith", "john.smith@acme.com", "Acme Corp"},
				{"Johnny Appleseed", "johnny@example.com", "Example Inc"},
				{"Sarah Johnson", "sarah.johnson@acme.com", "Acme Corp"},
			}

			fmt.Printf("Found %d contacts:\n\n", len(results))

			for _, r := range results {
				fmt.Printf("  %s %s\n", "👤", common.BoldWhite.Sprint(r.name))
				fmt.Printf("    📧 %s\n", r.email)
				fmt.Printf("    💼 %s\n", common.Dim.Sprint(r.company))
				fmt.Println()
			}

			fmt.Println(common.Dim.Sprint("To search your real contacts: nylas auth login"))

			return nil
		},
	}

	return cmd
}

// ============================================================================
// GROUPS COMMAND
// ============================================================================

// newDemoContactsGroupsCmd creates the groups subcommand.
