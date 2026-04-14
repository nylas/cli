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
	var signatureID string
	var templateOpts hostedTemplateSendOptions

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
- --metadata key=value: Add custom key-value metadata (can be repeated)

Supports hosted templates:
- --template-id <id>: Render and send a Nylas-hosted template
- --template-data <json>: Provide template variables as inline JSON
- --template-data-file <path>: Load template variables from a JSON file
- --render-only: Preview the rendered template without sending`,
		Example: `  # Send immediately
  nylas email send --to user@example.com --subject "Hello" --body "Hi there!"

  # Send using a hosted template
  nylas email send --to user@example.com --template-id tpl_123 --template-data '{"user":{"name":"Ada"}}'

  # Preview a hosted template render without sending
  nylas email send --template-id tpl_123 --template-data-file data.json --render-only

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

			// Interactive mode (runs before client setup).
			if shouldUseInteractiveSendMode(interactive, to, subject, body, templateOpts) {
				reader := bufio.NewReader(os.Stdin)

				if len(to) == 0 && !templateOpts.RenderOnly {
					fmt.Print("To (comma-separated): ")
					input, _ := reader.ReadString('\n')
					to = parseEmails(strings.TrimSpace(input))
				}

				if templateOpts.TemplateID == "" && subject == "" {
					fmt.Print("Subject: ")
					subject, _ = reader.ReadString('\n')
					subject = strings.TrimSpace(subject)
				}

				if templateOpts.TemplateID == "" && body == "" {
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

			if err := validateHostedTemplateSendOptions(templateOpts, subject, body); err != nil {
				return err
			}

			if len(to) == 0 && !templateOpts.RenderOnly {
				return common.NewUserError("at least one recipient is required", "Use --to to specify recipient email addresses")
			}

			if templateOpts.TemplateID == "" && subject == "" {
				return common.NewUserError("subject is required", "Use --subject to specify the email subject")
			}

			var toContacts []domain.EmailParticipant
			var ccContacts []domain.EmailParticipant
			var bccContacts []domain.EmailParticipant

			// Parse and validate recipients.
			if len(to) > 0 {
				var err error
				toContacts, err = parseContacts(to)
				if err != nil {
					return common.WrapRecipientError("to", err)
				}
			}
			if len(cc) > 0 {
				var err error
				ccContacts, err = parseContacts(cc)
				if err != nil {
					return common.WrapRecipientError("cc", err)
				}
			}
			if len(bcc) > 0 {
				var err error
				bccContacts, err = parseContacts(bcc)
				if err != nil {
					return common.WrapRecipientError("bcc", err)
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
			}

			sendNeedsGrant, err := hostedTemplateSendNeedsGrant(templateOpts)
			if err != nil {
				return err
			}

			sendWithClient := func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				activeSubject := subject
				activeBody := body
				var templatePreviewLabel string

				if templateOpts.TemplateID != "" {
					rendered, err := renderHostedTemplateForSend(ctx, client, grantID, templateOpts)
					if err != nil {
						return struct{}{}, err
					}

					activeSubject = rendered.Subject
					activeBody = rendered.Body
					templatePreviewLabel = templateOpts.TemplateID

					if templateOpts.RenderOnly {
						if jsonOutput {
							return struct{}{}, common.PrintJSON(rendered.Result)
						}
						printHostedTemplatePreview(templateOpts.TemplateID, rendered.Subject, rendered.Body, to, cc, bcc)
						return struct{}{}, nil
					}
				}

				req := &domain.SendMessageRequest{
					Subject:     activeSubject,
					Body:        activeBody,
					To:          toContacts,
					Cc:          ccContacts,
					Bcc:         bccContacts,
					SignatureID: signatureID,
				}
				if replyTo != "" {
					req.ReplyToMsgID = replyTo
				}
				if trackOpens || trackLinks || trackLabel != "" {
					req.TrackingOpts = &domain.TrackingOptions{
						Opens: trackOpens,
						Links: trackLinks,
						Label: trackLabel,
					}
				}
				if len(metadata) > 0 {
					req.Metadata = make(map[string]string)
					for _, m := range metadata {
						parts := strings.SplitN(m, "=", 2)
						if len(parts) == 2 {
							req.Metadata[parts[0]] = parts[1]
							continue
						}
						return struct{}{}, common.NewInputError(fmt.Sprintf("invalid metadata format: %s (expected key=value)", m))
					}
				}
				if !scheduledTime.IsZero() {
					req.SendAt = scheduledTime.Unix()
				}

				fmt.Println("\nEmail preview:")
				if templatePreviewLabel != "" {
					fmt.Printf("  Template: %s\n", templatePreviewLabel)
				}
				if len(to) > 0 {
					fmt.Printf("  To:      %s\n", strings.Join(to, ", "))
				}
				if len(cc) > 0 {
					fmt.Printf("  Cc:      %s\n", strings.Join(cc, ", "))
				}
				if len(bcc) > 0 {
					fmt.Printf("  Bcc:     %s\n", strings.Join(bcc, ", "))
				}
				fmt.Printf("  Subject: %s\n", activeSubject)
				if activeBody != "" {
					fmt.Printf("  Body:    %s\n", common.Truncate(activeBody, 50))
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
					signingInfo := "default key from git config"
					if gpgKeyID != "" {
						signingInfo = fmt.Sprintf("key %s", gpgKeyID)
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
				if signatureID != "" {
					fmt.Printf("  %s %s\n", common.Cyan.Sprint("Signature:"), signatureID)
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
						return struct{}{}, nil
					}
				}

				var msg *domain.Message
				var err error

				// Get grant info to determine provider and email
				grant, err := getGrantForSend(ctx, client, grantID)
				if err != nil {
					return struct{}{}, err
				}
				if signatureID != "" {
					if err := validateSendSignatureSupport(signatureID, sign, encrypt, grant); err != nil {
						return struct{}{}, err
					}
					if _, err := validateSignatureSelection(ctx, client, grantID, signatureID, grant); err != nil {
						return struct{}{}, err
					}
				}

				if sign || encrypt {
					if err := validateManagedSecureSendSupport(sign, encrypt, grant); err != nil {
						return struct{}{}, err
					}
					if grant.Email != "" {
						// Populate From field with grant's email address
						req.From = []domain.EmailParticipant{
							{Email: grant.Email},
						}
					}

					// GPG signing and/or encryption flow
					msg, err = sendSecureEmail(ctx, client, grantID, req, gpgKeyID, recipientKey, toContacts, activeSubject, activeBody, sign, encrypt)
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

					msg, err = sendMessageForGrant(ctx, client, grantID, grant, req)
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
			}

			if sendNeedsGrant {
				_, sendErr := common.WithClient(args, sendWithClient)
				return sendErr
			}

			_, sendErr := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				return sendWithClient(ctx, client, "")
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
	cmd.Flags().StringVar(&signatureID, "signature-id", "", "Stored signature ID to append when sending")
	cmd.Flags().StringVar(&templateOpts.TemplateID, "template-id", "", "Hosted template ID to render and send")
	cmd.Flags().StringVar(&templateOpts.TemplateScope, "template-scope", string(domain.ScopeApplication), "Hosted template scope: app or grant")
	cmd.Flags().StringVar(&templateOpts.TemplateGrantID, "template-grant-id", "", "Grant ID or email for grant-scoped hosted templates")
	cmd.Flags().StringVar(&templateOpts.TemplateData, "template-data", "", "Inline JSON object with hosted template variables")
	cmd.Flags().StringVar(&templateOpts.TemplateDataFile, "template-data-file", "", "Path to a JSON file with hosted template variables")
	cmd.Flags().BoolVar(&templateOpts.RenderOnly, "render-only", false, "Render a hosted template preview without sending")
	cmd.Flags().BoolVar(&templateOpts.Strict, "template-strict", true, "Fail if a hosted template references missing variables")

	return cmd
}

func getGrantForSend(ctx context.Context, client ports.NylasClient, grantID string) (*domain.Grant, error) {
	grant, err := client.GetGrant(ctx, grantID)
	if err != nil {
		return nil, common.WrapGetError("grant", err)
	}
	return grant, nil
}

func sendMessageForGrant(
	ctx context.Context,
	client ports.NylasClient,
	grantID string,
	grant *domain.Grant,
	req *domain.SendMessageRequest,
) (*domain.Message, error) {
	if isManagedTransactionalGrant(grant) {
		emailDomain := common.ExtractDomain(grant.Email)
		if emailDomain == "" {
			return nil, common.NewUserError(
				"could not extract domain from grant email",
				"Ensure the grant has a valid email address",
			)
		}
		req.From = []domain.EmailParticipant{{Email: grant.Email}}
		return client.SendTransactionalMessage(ctx, emailDomain, req)
	}

	return client.SendMessage(ctx, grantID, req)
}

func shouldUseInteractiveSendMode(
	interactive bool,
	to []string,
	subject, body string,
	templateOpts hostedTemplateSendOptions,
) bool {
	if interactive {
		return true
	}
	if templateOpts.TemplateID != "" {
		return len(to) == 0 && !templateOpts.RenderOnly
	}
	return len(to) == 0 && subject == "" && body == ""
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
