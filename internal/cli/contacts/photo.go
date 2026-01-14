package contacts

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newPhotoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "photo",
		Short: "Manage contact profile photos",
		Long:  `Download and view contact profile pictures from the email provider.`,
	}

	cmd.AddCommand(newPhotoDownloadCmd())
	cmd.AddCommand(newPhotoInfoCmd())

	return cmd
}

func newPhotoDownloadCmd() *cobra.Command {
	var outputFile string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "download <contact-id>",
		Short: "Download contact profile picture",
		Long: `Download a contact's profile picture as a Base64-encoded image.

The Nylas API returns profile pictures as Base64-encoded data when you include
the profile_picture=true query parameter. This command retrieves the picture
and saves it to a file or displays the Base64 data.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := getGrantID(args[1:])
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			contact, err := client.GetContactWithPicture(ctx, grantID, contactID, true)
			if err != nil {
				return common.WrapGetError("contact", err)
			}

			if contact.Picture == "" {
				fmt.Println("No profile picture available for this contact")
				return nil
			}

			if outputFile != "" {
				// Decode Base64 and save to file
				imageData, err := base64.StdEncoding.DecodeString(contact.Picture)
				if err != nil {
					return common.WrapDecodeError("image data", err)
				}

				// Use restrictive permissions (owner-only) for contact photos
				if err := os.WriteFile(outputFile, imageData, 0600); err != nil {
					return common.WrapWriteError("file", err)
				}

				fmt.Printf("Profile picture saved to: %s\n", outputFile)
				fmt.Printf("Size: %d bytes\n", len(imageData))
			} else if jsonOutput {
				// Print as JSON
				fmt.Printf(`{"contact_id":"%s","picture":"%s"}`+"\n", contactID, contact.Picture)
			} else {
				// Print Base64 data
				fmt.Println("Base64-encoded profile picture:")
				fmt.Println(contact.Picture)
				fmt.Println("\nTo save to a file, use the --output flag")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (decodes and saves image)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newPhotoInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show information about profile pictures",
		Long: `Display information about how profile pictures work in Nylas API v3.

Profile Picture Management in Nylas API v3
==========================================

Retrieval:
  - Include ?profile_picture=true in GET contact requests
  - API returns Base64-encoded image data in the 'picture' field
  - Image comes directly from the email provider

Upload:
  - Not supported in Nylas API v3
  - Profile pictures must be managed through the email provider
  - For Gmail: Visit https://contacts.google.com
  - For Outlook: Visit https://outlook.office.com/people
  - For other providers: Use their native contact management

Limitations:
  - Picture availability depends on email provider
  - Not all contacts have profile pictures
  - Image format and size controlled by provider

Best Practices:
  - Cache pictures locally if using frequently
  - Handle missing pictures gracefully
  - Decode Base64 data when saving to disk

Example Usage:
  # Download profile picture to file
  nylas contacts photo download <contact-id> --output photo.jpg

  # Get Base64 data
  nylas contacts photo download <contact-id>

  # Get as JSON
  nylas contacts photo download <contact-id> --json
`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmd.Long)
		},
	}

	return cmd
}
