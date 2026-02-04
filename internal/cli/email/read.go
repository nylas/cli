package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newReadCmd() *cobra.Command {
	var markAsRead bool
	var rawOutput bool
	var mimeOutput bool
	var headersOutput bool
	var verifySignature bool
	var decryptMessage bool

	cmd := &cobra.Command{
		Use:     "read <message-id> [grant-id]",
		Aliases: []string{"show"},
		Short:   "Read a specific email",
		Long: `Read and display the full content of a specific email message.

Supports GPG/PGP encrypted and signed messages:
- --decrypt: Decrypt PGP/MIME encrypted emails
- --verify: Verify GPG/PGP signature of signed emails`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			messageID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Determine which fields to request
				var fields string
				switch {
				case mimeOutput, verifySignature, decryptMessage:
					// --mime, --verify, and --decrypt need raw MIME data
					fields = "raw_mime"
				case headersOutput:
					fields = "include_headers"
				}

				// Fetch message with appropriate fields
				var msg *domain.Message
				var err error
				if fields != "" {
					msg, err = client.GetMessageWithFields(ctx, grantID, messageID, fields)
				} else {
					msg, err = client.GetMessage(ctx, grantID, messageID)
				}
				if err != nil {
					return struct{}{}, common.WrapGetError("message", err)
				}

				// Handle JSON output
				jsonOutput, _ := cmd.Flags().GetBool("json")
				if jsonOutput {
					data, err := json.MarshalIndent(msg, "", "  ")
					if err != nil {
						return struct{}{}, common.WrapMarshalError("JSON", err)
					}
					fmt.Println(string(data))
					return struct{}{}, nil
				}

				// Handle --decrypt flag
				if decryptMessage {
					// Fetch full message for header display
					fullMsg, err := client.GetMessage(ctx, grantID, messageID)
					if err == nil {
						printMessageHeaders(*fullMsg)
					}
					// Decrypt the message
					result, err := decryptGPGEmail(ctx, msg)
					if err != nil {
						return struct{}{}, fmt.Errorf("GPG decryption failed: %w", err)
					}
					// Display decryption result (signature info only if --verify also passed)
					printDecryptResult(result, verifySignature)
					printDecryptedContent(result.Plaintext)
					return struct{}{}, nil
				}

				// Handle --verify flag
				if verifySignature {
					// Fetch full message for display (raw_mime request returns minimal fields)
					fullMsg, err := client.GetMessage(ctx, grantID, messageID)
					if err == nil {
						printMessage(*fullMsg, true)
					}
					// Verify signature using raw MIME
					if err := verifyGPGSignature(ctx, msg); err != nil {
						return struct{}{}, fmt.Errorf("GPG verification failed: %w", err)
					}
					return struct{}{}, nil
				}

				// Display logic: --mime > --headers > --raw > default
				switch {
				case mimeOutput:
					// Get provider info to show better error message for Microsoft
					provider := getProviderForGrant(grantID)
					printMessageMIMEWithProvider(*msg, provider)
				case headersOutput:
					printMessageHeaders(*msg)
				case rawOutput:
					printMessageRaw(*msg)
				default:
					printMessage(*msg, true)
				}

				// Mark as read if requested
				if markAsRead && msg.Unread {
					unread := false
					_, err := client.UpdateMessage(ctx, grantID, messageID, &domain.UpdateMessageRequest{
						Unread: &unread,
					})
					if err != nil {
						_, _ = common.Dim.Printf("(Failed to mark as read: %v)\n", err)
					} else {
						_, _ = common.Dim.Println("(Marked as read)")
					}
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&markAsRead, "mark-read", "r", false, "Mark the message as read after viewing")
	cmd.Flags().BoolVar(&rawOutput, "raw", false, "Show raw email body without HTML processing")
	cmd.Flags().BoolVar(&mimeOutput, "mime", false, "Show raw RFC822/MIME message format")
	cmd.Flags().BoolVar(&headersOutput, "headers", false, "Show email headers (works with all providers)")
	cmd.Flags().BoolVar(&verifySignature, "verify", false, "Verify GPG/PGP signature of the message")
	cmd.Flags().BoolVar(&decryptMessage, "decrypt", false, "Decrypt PGP/MIME encrypted message")

	return cmd
}

// getProviderForGrant retrieves the provider type for a grant ID.
// Returns empty string if provider cannot be determined.
func getProviderForGrant(grantID string) domain.Provider {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return ""
	}

	grantStore := keyring.NewGrantStore(secretStore)
	grant, err := grantStore.GetGrant(grantID)
	if err != nil || grant == nil {
		return ""
	}

	return grant.Provider
}
