package email

import (
	"bufio"
	"context"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	configAdapter "github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newSendCmd() *cobra.Command {
	var to []string
	var cc []string
	var bcc []string
	var subject string
	var body string
	var replyTo string
	var interactive bool
	var scheduleAt string
	var noConfirm bool
	var trackOpens bool
	var trackLinks bool
	var trackLabel string
	var metadata []string
	var jsonOutput bool
	var sign bool
	var gpgKeyID string
	var listGPGKeys bool
	var encrypt bool
	var recipientKey string

	cmd := &cobra.Command{
		Use:   "send [grant-id]",
		Short: "Send an email",
		Long: `Compose and send an email message.

Supports GPG/PGP email signing:
- --sign: Sign email with your GPG key (uses default key from git config)
- --gpg-key <key-id>: Sign with a specific GPG key
- --list-gpg-keys: List available GPG signing keys

Supports GPG/PGP email encryption:
- --encrypt: Encrypt email with recipient's GPG public key (auto-fetched if needed)
- --recipient-key <key-id>: Use specific GPG key for encryption
- --sign --encrypt: Sign AND encrypt for maximum security

Supports scheduled sending with the --schedule flag. You can specify:
- Duration: "30m", "2h", "1d" (minutes, hours, days from now)
- Time: "14:30" or "2:30pm" (today or tomorrow if past)
- Date/time: "2024-01-15 14:30" or "tomorrow 9am"
- Unix timestamp: "1705320600"

Supports email tracking:
- --track-opens: Track when recipients open the email
- --track-links: Track when recipients click links
- --track-label: Add a label to identify tracked emails

Supports custom metadata:
- --metadata key=value: Add custom key-value metadata (can be repeated)`,
		Example: `  # Send immediately
  nylas email send --to user@example.com --subject "Hello" --body "Hi there!"

  # Send with GPG signature (uses default key from git config)
  nylas email send --to user@example.com --subject "Secure" --body "Signed email" --sign

  # Send with specific GPG key
  nylas email send --to user@example.com --subject "Secure" --body "Signed" --sign --gpg-key 601FEE9B1D60185F

  # Encrypt email (auto-fetches recipient's public key)
  nylas email send --to bob@example.com --subject "Confidential" --body "Secret message" --encrypt

  # Encrypt with specific recipient key
  nylas email send --to bob@example.com --subject "Confidential" --body "Secret" --encrypt --recipient-key ABCD1234

  # Sign AND encrypt (maximum security)
  nylas email send --to bob@example.com --subject "Top Secret" --body "Secret message" --sign --encrypt

  # List available GPG keys
  nylas email send --list-gpg-keys

  # Send in 2 hours
  nylas email send --to user@example.com --subject "Reminder" --schedule 2h

  # Send tomorrow at 9am
  nylas email send --to user@example.com --subject "Morning" --schedule "tomorrow 9am"

  # Send at a specific time
  nylas email send --to user@example.com --subject "Meeting" --schedule "2024-01-15 14:30"

  # Send with open and link tracking
  nylas email send --to user@example.com --subject "Newsletter" --track-opens --track-links

  # Send with custom metadata
  nylas email send --to user@example.com --subject "Invoice" --metadata campaign=q4 --metadata type=invoice`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --list-gpg-keys early (no client needed)
			if listGPGKeys {
				return handleListGPGKeys(cmd.Context())
			}

			// Check auto-sign config if --sign flag not explicitly set
			if !cmd.Flags().Changed("sign") {
				configStore := configAdapter.NewDefaultFileStore()
				cfg, err := configStore.Load()
				if err == nil && cfg != nil && cfg.GPG != nil && cfg.GPG.AutoSign {
					sign = true
				}
			}

			// Interactive mode (runs before WithClient)
			if interactive || (len(to) == 0 && subject == "" && body == "") {
				reader := bufio.NewReader(os.Stdin)

				if len(to) == 0 {
					fmt.Print("To (comma-separated): ")
					input, _ := reader.ReadString('\n')
					to = parseEmails(strings.TrimSpace(input))
				}

				if subject == "" {
					fmt.Print("Subject: ")
					subject, _ = reader.ReadString('\n')
					subject = strings.TrimSpace(subject)
				}

				if body == "" {
					fmt.Println("Body (end with a line containing only '.'):")
					var bodyLines []string
					for {
						line, _ := reader.ReadString('\n')
						line = strings.TrimSuffix(line, "\n")
						if line == "." {
							break
						}
						bodyLines = append(bodyLines, line)
					}
					body = strings.Join(bodyLines, "\n")
				}
			}

			if len(to) == 0 {
				return common.NewUserError("at least one recipient is required", "Use --to to specify recipient email addresses")
			}

			if subject == "" {
				return common.NewUserError("subject is required", "Use --subject to specify the email subject")
			}

			// Parse and validate recipients
			toContacts, err := parseContacts(to)
			if err != nil {
				return common.WrapRecipientError("to", err)
			}

			// Build request
			req := &domain.SendMessageRequest{
				Subject: subject,
				Body:    body,
				To:      toContacts,
			}

			if len(cc) > 0 {
				ccContacts, err := parseContacts(cc)
				if err != nil {
					return common.WrapRecipientError("cc", err)
				}
				req.Cc = ccContacts
			}
			if len(bcc) > 0 {
				bccContacts, err := parseContacts(bcc)
				if err != nil {
					return common.WrapRecipientError("bcc", err)
				}
				req.Bcc = bccContacts
			}
			if replyTo != "" {
				req.ReplyToMsgID = replyTo
			}

			// Add tracking options if specified
			if trackOpens || trackLinks || trackLabel != "" {
				req.TrackingOpts = &domain.TrackingOptions{
					Opens: trackOpens,
					Links: trackLinks,
					Label: trackLabel,
				}
			}

			// Parse metadata key=value pairs
			if len(metadata) > 0 {
				req.Metadata = make(map[string]string)
				for _, m := range metadata {
					parts := strings.SplitN(m, "=", 2)
					if len(parts) == 2 {
						req.Metadata[parts[0]] = parts[1]
					} else {
						return common.NewInputError(fmt.Sprintf("invalid metadata format: %s (expected key=value)", m))
					}
				}
			}

			// Parse schedule time if provided
			var scheduledTime time.Time
			if scheduleAt != "" {
				var parseErr error
				scheduledTime, parseErr = parseScheduleTime(scheduleAt)
				if parseErr != nil {
					return parseErr // parseScheduleTime already returns CLIError
				}
				req.SendAt = scheduledTime.Unix()
			}

			// Confirmation
			fmt.Println("\nEmail preview:")
			fmt.Printf("  To:      %s\n", strings.Join(to, ", "))
			if len(cc) > 0 {
				fmt.Printf("  Cc:      %s\n", strings.Join(cc, ", "))
			}
			if len(bcc) > 0 {
				fmt.Printf("  Bcc:     %s\n", strings.Join(bcc, ", "))
			}
			fmt.Printf("  Subject: %s\n", subject)
			if body != "" {
				fmt.Printf("  Body:    %s\n", common.Truncate(body, 50))
			}
			if !scheduledTime.IsZero() {
				fmt.Printf("  %s %s\n", common.Yellow.Sprint("Scheduled:"), scheduledTime.Format(common.DisplayWeekdayFullWithTZ))
			}
			if trackOpens || trackLinks {
				tracking := []string{}
				if trackOpens {
					tracking = append(tracking, "opens")
				}
				if trackLinks {
					tracking = append(tracking, "links")
				}
				fmt.Printf("  %s %s\n", common.Cyan.Sprint("Tracking:"), strings.Join(tracking, ", "))
			}
			if len(metadata) > 0 {
				fmt.Printf("  %s %s\n", common.Cyan.Sprint("Metadata:"), strings.Join(metadata, ", "))
			}
			if sign {
				var signingInfo string
				if gpgKeyID != "" {
					// Explicit key ID provided
					signingInfo = fmt.Sprintf("key %s", gpgKeyID)
				} else {
					// Auto-detect from From address
					fromEmail := ""
					if len(toContacts) > 0 && len(req.From) > 0 {
						fromEmail = req.From[0].Email
					}
					if fromEmail != "" {
						signingInfo = fmt.Sprintf("as %s", fromEmail)
					} else {
						signingInfo = "default key from git config"
					}
				}
				fmt.Printf("  %s %s\n", common.Green.Sprint("GPG Signed:"), signingInfo)
			}
			if encrypt {
				var encryptInfo string
				if recipientKey != "" {
					encryptInfo = fmt.Sprintf("with key %s", recipientKey)
				} else {
					encryptInfo = fmt.Sprintf("for %s (auto-fetch)", strings.Join(to, ", "))
				}
				fmt.Printf("  %s %s\n", common.Blue.Sprint("GPG Encrypted:"), encryptInfo)
			}

			if !noConfirm {
				if scheduledTime.IsZero() {
					fmt.Print("\nSend this email? [y/N]: ")
				} else {
					fmt.Print("\nSchedule this email? [y/N]: ")
				}

				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.ToLower(strings.TrimSpace(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			// Send
			_, sendErr := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				var msg *domain.Message
				var err error

				// Get grant info to determine provider and email
				grant, grantErr := client.GetGrant(ctx, grantID)

				if sign || encrypt {
					if grantErr == nil && grant != nil && grant.Email != "" {
						// Populate From field with grant's email address
						req.From = []domain.EmailParticipant{
							{Email: grant.Email},
						}
					}

					// GPG signing and/or encryption flow
					msg, err = sendSecureEmail(ctx, client, grantID, req, gpgKeyID, recipientKey, toContacts, subject, body, sign, encrypt)
				} else {
					// Standard flow
					var sendMsg string
					if scheduledTime.IsZero() {
						sendMsg = "Sending email..."
					} else {
						sendMsg = "Scheduling email..."
					}

					spinner := common.NewSpinner(sendMsg)
					spinner.Start()

					// Auto-detect inbox provider and use transactional endpoint
					if grantErr == nil && grant != nil && grant.Provider == domain.ProviderInbox {
						// Inbox provider - use domain-based transactional send
						emailDomain := common.ExtractDomain(grant.Email)
						if emailDomain == "" {
							spinner.Stop()
							return struct{}{}, common.NewUserError(
								"could not extract domain from grant email",
								"Ensure the grant has a valid email address",
							)
						}
						// Set From field for transactional send (required)
						req.From = []domain.EmailParticipant{{Email: grant.Email}}
						msg, err = client.SendTransactionalMessage(ctx, emailDomain, req)
					} else {
						// Standard provider (Google/Microsoft/IMAP) - use grant-based send
						msg, err = client.SendMessage(ctx, grantID, req)
					}
					spinner.Stop()
				}

				if err != nil {
					return struct{}{}, common.WrapSendError("email", err)
				}

				if jsonOutput {
					return struct{}{}, common.PrintJSON(msg)
				}

				if !scheduledTime.IsZero() {
					printSuccess("Email scheduled successfully! Message ID: %s", msg.ID)
					fmt.Printf("Scheduled to send: %s\n", scheduledTime.Format(common.DisplayWeekdayFullWithTZ))
				} else {
					if sign && encrypt {
						printSuccess("Signed and encrypted email sent successfully! Message ID: %s", msg.ID)
					} else if encrypt {
						printSuccess("Encrypted email sent successfully! Message ID: %s", msg.ID)
					} else if sign {
						printSuccess("Signed email sent successfully! Message ID: %s", msg.ID)
					} else {
						printSuccess("Email sent successfully! Message ID: %s", msg.ID)
					}
				}
				return struct{}{}, nil
			})
			return sendErr
		},
	}

	cmd.Flags().StringSliceVarP(&to, "to", "t", nil, "Recipient email addresses")
	cmd.Flags().StringSliceVar(&cc, "cc", nil, "CC email addresses")
	cmd.Flags().StringSliceVar(&bcc, "bcc", nil, "BCC email addresses")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "Email subject")
	cmd.Flags().StringVarP(&body, "body", "b", "", "Email body (HTML or plain text)")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Message ID to reply to")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")
	cmd.Flags().StringVar(&scheduleAt, "schedule", "", "Schedule sending (e.g., '2h', 'tomorrow 9am', '2024-01-15 14:30')")
	cmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&trackOpens, "track-opens", false, "Track email opens")
	cmd.Flags().BoolVar(&trackLinks, "track-links", false, "Track link clicks")
	cmd.Flags().StringVar(&trackLabel, "track-label", "", "Label for tracking (used to group tracked emails)")
	cmd.Flags().StringSliceVar(&metadata, "metadata", nil, "Custom metadata as key=value (can be repeated)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&sign, "sign", false, "Sign email with GPG (uses default key from git config)")
	cmd.Flags().StringVar(&gpgKeyID, "gpg-key", "", "Specific GPG key ID to use for signing")
	cmd.Flags().BoolVar(&listGPGKeys, "list-gpg-keys", false, "List available GPG signing keys and exit")
	cmd.Flags().BoolVar(&encrypt, "encrypt", false, "Encrypt email with recipient's GPG public key")
	cmd.Flags().StringVar(&recipientKey, "recipient-key", "", "Specific GPG key ID for encryption (auto-detected from recipient email if not specified)")

	return cmd
}

// parseEmails parses a comma-separated list of emails.
func parseEmails(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// parseContacts converts email strings to EmailParticipant objects with validation.
func parseContacts(emails []string) ([]domain.EmailParticipant, error) {
	contacts := make([]domain.EmailParticipant, len(emails))
	for i, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			return nil, common.NewInputError("email address cannot be empty")
		}

		// Try parsing as RFC 5322 address (handles "Name <email>" format)
		addr, err := mail.ParseAddress(email)
		if err == nil {
			contacts[i] = domain.EmailParticipant{Name: addr.Name, Email: addr.Address}
		} else {
			// Check if it's a plain email without angle brackets
			if !strings.Contains(email, "@") {
				return nil, common.NewInputError(fmt.Sprintf("invalid email address: %s", email))
			}
			// Basic validation for plain email
			if strings.Count(email, "@") != 1 {
				return nil, common.NewInputError(fmt.Sprintf("invalid email address: %s", email))
			}
			contacts[i] = domain.EmailParticipant{Email: email}
		}
	}
	return contacts, nil
}

// errScheduleInPast is returned when the scheduled time is in the past.
var errScheduleInPast = common.NewUserError("scheduled time is in the past", "Specify a future time")

// parseScheduleTime parses various time formats for scheduling.
func parseScheduleTime(input string) (time.Time, error) {
	now := time.Now()
	input = strings.TrimSpace(input)
	lower := strings.ToLower(input)

	// Try Unix timestamp first
	if ts, err := strconv.ParseInt(input, 10, 64); err == nil && ts > 1000000000 {
		t := time.Unix(ts, 0)
		if t.Before(now) {
			return time.Time{}, errScheduleInPast
		}
		return t, nil
	}

	// Duration formats: 30m, 2h, 1d
	if len(input) >= 2 {
		numStr := input[:len(input)-1]
		unit := input[len(input)-1:]
		if num, err := strconv.Atoi(numStr); err == nil {
			switch unit {
			case "m":
				return now.Add(time.Duration(num) * time.Minute), nil
			case "h":
				return now.Add(time.Duration(num) * time.Hour), nil
			case "d":
				return now.AddDate(0, 0, num), nil
			}
		}
	}

	// "tomorrow" keyword
	if strings.HasPrefix(lower, "tomorrow") {
		tomorrow := now.AddDate(0, 0, 1)
		rest := strings.TrimPrefix(lower, "tomorrow")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			// Default to 9am tomorrow
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, now.Location()), nil
		}
		// Parse time part
		if t, err := common.ParseTimeOfDay(rest); err == nil {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), nil
		}
	}

	// "today" keyword
	if strings.HasPrefix(lower, "today") {
		rest := strings.TrimPrefix(lower, "today")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return time.Time{}, common.NewInputError("please specify a time, e.g., 'today 3pm'")
		}
		if t, err := common.ParseTimeOfDay(rest); err == nil {
			result := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			if result.Before(now) {
				return time.Time{}, errScheduleInPast
			}
			return result, nil
		}
	}

	// Try standard date/time formats
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02 3:04pm",
		"2006-01-02 3:04PM",
		"Jan 2 15:04",
		"Jan 2 3:04pm",
		"Jan 2, 2006 15:04",
		"Jan 2, 2006 3:04pm",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, now.Location()); err == nil {
			// If year wasn't specified, use current year
			if t.Year() == 0 {
				t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			}
			if t.Before(now) {
				return time.Time{}, errScheduleInPast
			}
			return t, nil
		}
	}

	// Try just time of day (today or tomorrow)
	if t, err := common.ParseTimeOfDay(lower); err == nil {
		result := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if result.Before(now) {
			// If time is in the past, assume tomorrow
			result = result.AddDate(0, 0, 1)
		}
		return result, nil
	}

	return time.Time{}, common.NewInputError(fmt.Sprintf("could not parse time format: %s", input))
}
