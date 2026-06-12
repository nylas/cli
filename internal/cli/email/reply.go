package email

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newReplyCmd() *cobra.Command {
	var body string
	var all bool
	var interactive bool
	var noConfirm bool

	cmd := &cobra.Command{
		Use:   "reply <message-id> [grant-id]",
		Short: "Reply to an email",
		Long: `Reply to an email message, keeping it in the same thread.

The original message is fetched to populate the recipient and subject
automatically. By default the reply goes only to the original sender; use
--all to also include the other To/Cc recipients (excluding yourself).

Threading is preserved via the message's reply_to_message_id, so the reply
groups with the original conversation in mail clients.`,
		Example: `  # Reply to the sender
  nylas email reply <message-id> --body "Sounds good, thanks!"

  # Reply to everyone on the thread
  nylas email reply <message-id> --all --body "Looping everyone in."

  # Compose the body interactively
  nylas email reply <message-id> --interactive

  # Reply using a specific grant
  nylas email reply <message-id> <grant-id> --body "On it."`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			messageID := args[0]
			remainingArgs := args[1:]
			jsonOutput := common.IsJSON(cmd)

			if interactive && body == "" {
				body = promptReplyBody()
			}
			if strings.TrimSpace(body) == "" {
				return common.NewUserError("reply body is required", "Use --body to provide the reply text, or --interactive to compose it")
			}

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				grant, err := getGrantForSend(ctx, client, grantID)
				if err != nil {
					return struct{}{}, err
				}

				req, err := buildReplyRequest(ctx, client, grantID, grant, messageID, body, all)
				if err != nil {
					return struct{}{}, err
				}

				printReplyPreview(req)

				if !noConfirm {
					if !common.Confirm("\nSend this reply?", false) {
						fmt.Println("Cancelled.")
						return struct{}{}, nil
					}
				}

				spinner := common.NewSpinner("Sending reply...")
				spinner.Start()
				msg, err := sendMessageForGrant(ctx, client, grantID, grant, req)
				spinner.Stop()
				if err != nil {
					return struct{}{}, common.WrapSendError("reply", err)
				}

				if jsonOutput {
					return struct{}{}, common.PrintJSON(msg)
				}
				common.PrintSuccess("Reply sent successfully! Message ID: %s", msg.ID)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Reply body (HTML or plain text)")
	cmd.Flags().BoolVar(&all, "all", false, "Reply to all recipients (original To and Cc, excluding yourself)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Compose the reply body interactively")
	cmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

// buildReplyRequest fetches the original message and assembles a send request
// that threads as a reply to it.
func buildReplyRequest(
	ctx context.Context,
	client ports.NylasClient,
	grantID string,
	grant *domain.Grant,
	messageID, body string,
	all bool,
) (*domain.SendMessageRequest, error) {
	orig, err := client.GetMessage(ctx, grantID, messageID)
	if err != nil {
		return nil, common.WrapGetError("message", err)
	}

	selfEmail := ""
	if grant != nil {
		selfEmail = grant.Email
	}

	to, cc, err := buildReplyRecipients(orig, selfEmail, all)
	if err != nil {
		return nil, err
	}

	return &domain.SendMessageRequest{
		Subject:      replySubject(orig.Subject),
		Body:         body,
		To:           to,
		Cc:           cc,
		ReplyToMsgID: messageID,
	}, nil
}

// buildReplyRecipients determines who a reply should go to. The reply targets
// the original Reply-To header when present, otherwise the original sender. The
// replier's own address is always excluded so replying to a message you sent
// goes to the other participants rather than back to yourself. With all set, the
// original To and Cc recipients are added (de-duplicated). When the reply target
// was only yourself (a self-sent message), the other recipients are promoted to
// the To line.
func buildReplyRecipients(orig *domain.Message, selfEmail string, all bool) (to, cc []domain.EmailParticipant, err error) {
	noRecipients := common.NewUserError(
		"the original message has no one to reply to",
		"Check the message ID",
	)

	// Prefer the Reply-To header, but only when it carries a usable address;
	// a header present with only blank/empty entries must fall back to From.
	target := orig.From
	for _, p := range orig.ReplyTo {
		if normalizeEmail(p.Email) != "" {
			target = orig.ReplyTo
			break
		}
	}

	seen := make(map[string]bool)
	if self := normalizeEmail(selfEmail); self != "" {
		seen[self] = true
	}

	appendUnseen := func(dst *[]domain.EmailParticipant, list []domain.EmailParticipant) {
		for _, p := range list {
			key := normalizeEmail(p.Email)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			*dst = append(*dst, p)
		}
	}

	appendUnseen(&to, target)
	if !all {
		if len(to) == 0 {
			// target existed but resolved to only the replier (a self-sent message).
			if len(target) > 0 {
				return nil, nil, common.NewUserError(
					"replying to a message you sent would only address yourself",
					"Use --all to reply to the other recipients on the thread",
				)
			}
			return nil, nil, noRecipients
		}
		return to, nil, nil
	}

	appendUnseen(&cc, orig.To)
	appendUnseen(&cc, orig.Cc)

	// Replying to your own message: promote the other recipients to the To line.
	if len(to) == 0 {
		to, cc = cc, nil
	}
	if len(to) == 0 {
		return nil, nil, noRecipients
	}

	return to, cc, nil
}

// replySubject prefixes the original subject with "Re: " unless it already
// carries a reply prefix.
func replySubject(original string) string {
	trimmed := strings.TrimSpace(original)
	if strings.HasPrefix(strings.ToLower(trimmed), "re:") {
		return original
	}
	if trimmed == "" {
		return "Re:"
	}
	return "Re: " + original
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// promptReplyBody reads a multi-line reply body from stdin, terminated by a
// line containing only ".".
func promptReplyBody() string {
	fmt.Println("Body (end with a line containing only '.'):")
	return readReplyBody(os.Stdin)
}

// readReplyBody reads a multi-line body terminated by a line containing only
// "." or by EOF, so a closed/piped stdin cannot loop forever.
func readReplyBody(r io.Reader) string {
	reader := bufio.NewReader(r)
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "." {
			break
		}
		if err != nil {
			if trimmed != "" {
				lines = append(lines, trimmed)
			}
			break
		}
		lines = append(lines, trimmed)
	}
	return strings.Join(lines, "\n")
}

func printReplyPreview(req *domain.SendMessageRequest) {
	fmt.Println("\nReply preview:")
	if len(req.To) > 0 {
		fmt.Printf("  To:      %s\n", participantList(req.To))
	}
	if len(req.Cc) > 0 {
		fmt.Printf("  Cc:      %s\n", participantList(req.Cc))
	}
	fmt.Printf("  Subject: %s\n", req.Subject)
	if req.Body != "" {
		fmt.Printf("  Body:    %s\n", common.Truncate(req.Body, 50))
	}
}

func participantList(participants []domain.EmailParticipant) string {
	parts := make([]string, len(participants))
	for i, p := range participants {
		parts[i] = p.String()
	}
	return strings.Join(parts, ", ")
}
