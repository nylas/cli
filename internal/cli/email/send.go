package email

import (
	"bufio"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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

	cmd := &cobra.Command{
		Use:   "send [grant-id]",
		Short: "Send an email",
		Long: `Compose and send an email message.

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
			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := common.GetGrantID(args)
			if err != nil {
				return err
			}

			// Interactive mode
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
				var err error
				scheduledTime, err = parseScheduleTime(scheduleAt)
				if err != nil {
					return err // parseScheduleTime already returns CLIError
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
			ctx, cancel := common.CreateContext()
			defer cancel()

			msg, err := client.SendMessage(ctx, grantID, req)
			if err != nil {
				return common.WrapSendError("email", err)
			}

			if !scheduledTime.IsZero() {
				printSuccess("Email scheduled successfully! Message ID: %s", msg.ID)
				fmt.Printf("Scheduled to send: %s\n", scheduledTime.Format(common.DisplayWeekdayFullWithTZ))
			} else {
				printSuccess("Email sent successfully! Message ID: %s", msg.ID)
			}
			return nil
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
