package demo

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newDemoContactsPhotoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "photo",
		Short: "Manage contact photos",
		Long:  "Demo commands for managing contact photos.",
	}

	cmd.AddCommand(newDemoPhotoGetCmd())
	cmd.AddCommand(newDemoPhotoSetCmd())
	cmd.AddCommand(newDemoPhotoRemoveCmd())

	return cmd
}

func newDemoPhotoGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [contact-id]",
		Short: "Get contact photo",
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID := "contact-demo-123"
			if len(args) > 0 {
				contactID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📷 Demo Mode - Contact Photo"))
			fmt.Println()
			fmt.Printf("Contact ID: %s\n", contactID)
			fmt.Printf("Photo URL:  https://example.com/photos/%s.jpg\n", contactID)
			fmt.Printf("Size:       128x128\n")
			fmt.Printf("Format:     JPEG\n")

			return nil
		},
	}
}

func newDemoPhotoSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [contact-id] [photo-path]",
		Short: "Set contact photo",
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID := "contact-demo-123"
			photoPath := "photo.jpg"
			if len(args) > 0 {
				contactID = args[0]
			}
			if len(args) > 1 {
				photoPath = args[1]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Photo '%s' would be set for contact %s (demo mode)\n", photoPath, contactID)

			return nil
		},
	}
}

func newDemoPhotoRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [contact-id]",
		Short: "Remove contact photo",
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID := "contact-demo-123"
			if len(args) > 0 {
				contactID = args[0]
			}

			fmt.Println()
			_, _ = common.Green.Printf("✓ Photo would be removed from contact %s (demo mode)\n", contactID)

			return nil
		},
	}
}

// ============================================================================
// SYNC COMMAND
// ============================================================================

// newDemoContactsSyncCmd simulates syncing contacts.
func newDemoContactsSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync contacts",
		Long:  "Demo contact synchronization features.",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show sync status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println(common.Dim.Sprint("🔄 Demo Mode - Contact Sync Status"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			fmt.Printf("  Last sync:     %s\n", common.Green.Sprint("2 minutes ago"))
			fmt.Printf("  Total contacts: 247\n")
			fmt.Printf("  Added:         5\n")
			fmt.Printf("  Updated:       12\n")
			fmt.Printf("  Deleted:       2\n")
			fmt.Printf("  Sync status:   %s\n", common.Green.Sprint("Up to date"))
			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "now",
		Short: "Trigger sync now",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println(common.Dim.Sprint("🔄 Demo Mode - Contact Sync"))
			fmt.Println()
			fmt.Println("Syncing contacts...")
			fmt.Println()
			_, _ = common.Green.Println("✓ Sync would be triggered (demo mode)")
			fmt.Printf("  Estimated time: 30 seconds\n")

			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "export",
		Short: "Export contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println(common.Dim.Sprint("📤 Demo Mode - Export Contacts"))
			fmt.Println()
			fmt.Println("Available export formats:")
			fmt.Printf("  • CSV  - Comma-separated values\n")
			fmt.Printf("  • VCF  - vCard format\n")
			fmt.Printf("  • JSON - JSON format\n")
			fmt.Println()
			_, _ = common.Green.Println("✓ Export would be generated (demo mode)")

			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "import",
		Short: "Import contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println(common.Dim.Sprint("📥 Demo Mode - Import Contacts"))
			fmt.Println()
			fmt.Println("Supported import formats:")
			fmt.Printf("  • CSV  - Comma-separated values\n")
			fmt.Printf("  • VCF  - vCard format\n")
			fmt.Printf("  • JSON - JSON format\n")
			fmt.Println()
			_, _ = common.Green.Println("✓ Import would be processed (demo mode)")

			return nil
		},
	})

	return cmd
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// printDemoContact prints a contact summary.
func printDemoContact(contact domain.Contact, showID bool) {
	name := fmt.Sprintf("%s %s", contact.GivenName, contact.Surname)
	name = strings.TrimSpace(name)

	fmt.Printf("  %s %s\n", "👤", common.BoldWhite.Sprint(name))

	if len(contact.Emails) > 0 {
		fmt.Printf("    📧 %s\n", contact.Emails[0].Email)
	}

	if contact.CompanyName != "" || contact.JobTitle != "" {
		company := contact.CompanyName
		if contact.JobTitle != "" {
			if company != "" {
				company = contact.JobTitle + " at " + company
			} else {
				company = contact.JobTitle
			}
		}
		fmt.Printf("    💼 %s\n", common.Dim.Sprint(company))
	}

	if len(contact.PhoneNumbers) > 0 {
		fmt.Printf("    📱 %s\n", contact.PhoneNumbers[0].Number)
	}

	if showID {
		_, _ = common.Dim.Printf("    ID: %s\n", contact.ID)
	}

	fmt.Println()
}

// printDemoContactFull prints a full contact.
func printDemoContactFull(contact domain.Contact) {
	name := fmt.Sprintf("%s %s", contact.GivenName, contact.Surname)
	name = strings.TrimSpace(name)

	fmt.Println(strings.Repeat("─", 50))
	_, _ = common.BoldWhite.Printf("Name: %s\n", name)

	if contact.CompanyName != "" {
		fmt.Printf("Company: %s\n", contact.CompanyName)
	}
	if contact.JobTitle != "" {
		fmt.Printf("Title: %s\n", contact.JobTitle)
	}

	if len(contact.Emails) > 0 {
		fmt.Println("\nEmails:")
		for _, e := range contact.Emails {
			emailType := e.Type
			if emailType == "" {
				emailType = "email"
			}
			fmt.Printf("  %s: %s\n", emailType, e.Email)
		}
	}

	if len(contact.PhoneNumbers) > 0 {
		fmt.Println("\nPhone Numbers:")
		for _, p := range contact.PhoneNumbers {
			phoneType := p.Type
			if phoneType == "" {
				phoneType = "phone"
			}
			fmt.Printf("  %s: %s\n", phoneType, p.Number)
		}
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
}
