package deliverability

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

const activityRetention = 90 * 24 * time.Hour

type activityWindowFlags struct {
	start, end string
	// pageFlags repeats a look-back window as epochs, because the page token is bound to the exact start and end.
	pageFlags string
}

func addActivityWindowFlags(cmd *cobra.Command, f *activityWindowFlags, order *string, limit *int, pageToken *string) {
	cmd.Flags().StringVar(&f.start, "start", "", `Window start: Unix seconds, YYYY-MM-DD, "YYYY-MM-DD HH:MM" (UTC), RFC3339, or look-back like 24h/7d (default: UTC midnight 6 days before --end, or before today)`)
	cmd.Flags().StringVar(&f.end, "end", "", "Window end, inclusive, within the last 90 days; a date means the end of that UTC day (same formats; default: now)")
	cmd.Flags().StringVar(order, "order", "", "Sort order: desc (newest first, default) or asc")
	common.AddLimitFlag(cmd, limit, 50)
	common.AddPageTokenFlag(cmd, pageToken)
}

// resolve returns the start and end to send, in Unix seconds; 0 leaves the server default.
func (f *activityWindowFlags) resolve(now time.Time, pageToken string) (int64, int64, error) {
	start, startRelative, err := parseActivityTime("start", f.start, now, false)
	if err != nil {
		return 0, 0, err
	}
	end, endRelative, err := parseActivityTime("end", f.end, now, true)
	if err != nil {
		return 0, 0, err
	}
	// The server clamps start to its 90-day retention, so an older --end would return nothing with no error.
	if end != 0 && end < now.Add(-activityRetention).Unix() {
		return 0, 0, common.NewUserError("--end is more than 90 days ago", "Activity is kept for 90 days; use a more recent --end")
	}
	// Mirror the server's default start, anchored on --end, so an older --end is not before the default start.
	if start == 0 && end != 0 {
		endDay := time.Unix(end, 0).UTC().Truncate(24 * time.Hour)
		start = endDay.AddDate(0, 0, -6).Unix()
	}
	if !startRelative && !endRelative {
		return start, end, nil
	}
	if pageToken != "" {
		return 0, 0, common.NewUserError(
			"a look-back --start/--end cannot be used with --page-token",
			"Look-backs move between runs; repeat the --start/--end values printed with the page token",
		)
	}
	f.pageFlags = strings.TrimSpace(epochFlag("start", start) + " " + epochFlag("end", end))
	return start, end, nil
}

func epochFlag(name string, ts int64) string {
	if ts == 0 {
		return ""
	}
	return fmt.Sprintf("--%s %d", name, ts)
}

func newSendEventsListCmd() *cobra.Command {
	var window activityWindowFlags
	var params domain.SendEventListParams

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List send events (newest first)",
		Example: `  nylas deliverability send-events list --event-type bounced --start 24h
  nylas deliverability send-events list --recipient-domain gmail.com --event-type bounced --event-type complaint
  nylas deliverability send-events list --message-id "<abc@mail.example.com>" --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			if params.Start, params.End, err = window.resolve(time.Now(), params.PageToken); err != nil {
				return err
			}
			_, err = common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				return struct{}{}, runSendEventsList(ctx, cmd, client, &params, window.pageFlags)
			})
			return err
		},
	}

	cmd.Flags().StringArrayVar(&params.EventTypes, "event-type", nil,
		"Event type, repeatable: accepted, delivered, bounced, complaint, rejected")
	cmd.Flags().StringVar(&params.BounceClass, "bounce-class", "", "Bounce class: hard, soft, or undetermined")
	cmd.Flags().StringVar(&params.SendingDomain, "sending-domain", "", "Sending domain")
	cmd.Flags().StringVar(&params.From, "from", "", "From address")
	cmd.Flags().StringVar(&params.GrantID, "grant-id", "", "Only events for this grant (agent account)")
	cmd.Flags().StringVar(&params.Recipient, "recipient", "", "Recipient address")
	cmd.Flags().StringVar(&params.RecipientDomain, "recipient-domain", "", "Recipient domain")
	cmd.Flags().StringVar(&params.MessageID, "message-id", "", "Message ID")
	cmd.Flags().StringVar(&params.ThreadID, "thread-id", "", "Thread ID")
	addActivityWindowFlags(cmd, &window, &params.Order, &params.Limit, &params.PageToken)

	return cmd
}

func runSendEventsList(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.SendEventListParams, pageFlags string) error {
	resp, err := client.ListSendEvents(ctx, params)
	if err != nil {
		return common.WrapListError("send events", err)
	}
	if done, err := writeStructuredList(cmd, resp, resp.Data); done {
		// Stdout stays the API envelope; a look-back's resolved window goes to stderr so scripts can page.
		if err == nil && pageFlags != "" {
			nextCursorNote(cmd.ErrOrStderr(), resp.NextCursor, pageFlags)
		}
		return err
	}
	if len(resp.Data) == 0 {
		common.PrintEmptyStateWithHint("send events", "Widen the window with --start (activity is kept for 90 days)")
		return nil
	}

	table := newTable(cmd.OutOrStdout(), "TIME", "EVENT", "RECIPIENT", "FROM", "REASON", "EVENT ID")
	table.SetMaxWidth(4, 40)
	for _, e := range resp.Data {
		table.AddRow(formatUnix(e.Timestamp), sendEventLabel(e.EventType, e.BounceClass), e.Recipient, e.From, e.Reason, e.ID)
	}
	table.Render()
	nextCursorNote(cmd.OutOrStdout(), resp.NextCursor, pageFlags)
	return nil
}

func newSendEventsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <event-id>",
		Short: "Show one send event with its provider detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventID(args[0]); err != nil {
				return err
			}
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				event, err := client.GetSendEvent(ctx, args[0])
				if err != nil {
					return struct{}{}, common.WrapGetError("send event", err)
				}
				if done, err := writeStructured(cmd, event); done {
					return struct{}{}, err
				}
				printSendEvent(cmd, event)
				return struct{}{}, nil
			})
			return err
		},
	}
}

func printSendEvent(cmd *cobra.Command, e *domain.SendEvent) {
	fields := [][2]string{
		{"Event ID", e.ID},
		{"Event", sendEventLabel(e.EventType, e.BounceClass)},
		{"Time", formatUnix(e.Timestamp)},
		{"Message ID", e.MessageID},
		{"Thread ID", e.ThreadID},
		{"Grant ID", e.GrantID},
		{"From", e.From},
		{"Recipient", e.Recipient},
		{"Subject", derefString(e.Subject)},
		{"Reason", e.Reason},
	}
	if d := e.Detail; d != nil {
		fields = append(fields,
			[2]string{"SMTP response", d.SMTPResponse},
			[2]string{"Diagnostic", d.DiagnosticCode},
			[2]string{"Bounce type", strings.TrimSpace(d.BounceType + " " + d.BounceSubtype)},
			[2]string{"Status", d.Status},
			[2]string{"Action", d.Action},
			[2]string{"Reporting MTA", d.ReportingMTA},
			[2]string{"Remote MTA IP", d.RemoteMTAIP},
			[2]string{"Feedback type", strings.TrimSpace(d.FeedbackType + " " + d.ComplaintSubtype)},
		)
		if d.ProcessingTimeMs > 0 {
			fields = append(fields, [2]string{"Processing", fmt.Sprintf("%d ms", d.ProcessingTimeMs)})
		}
	}
	for _, f := range fields {
		if f[1] != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-15s %s\n", f[0]+":", f[1])
		}
	}
}

func newSendEventsMessageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "message <message-id>",
		Short: "Show every recipient's send timeline for one message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				activity, err := client.GetSendMessageActivity(ctx, args[0])
				if err != nil {
					return struct{}{}, common.WrapGetError("send activity for message", err)
				}
				if done, err := writeStructured(cmd, activity); done {
					return struct{}{}, err
				}
				printSendMessageActivity(cmd, activity)
				return struct{}{}, nil
			})
			return err
		},
	}
}

func printSendMessageActivity(cmd *cobra.Command, a *domain.SendMessageActivity) {
	w := cmd.OutOrStdout()
	m := a.Message
	_, _ = fmt.Fprintf(w, "Message ID:  %s\n", m.MessageID)
	_, _ = fmt.Fprintf(w, "From:        %s\n", m.From)
	if m.Subject != nil {
		_, _ = fmt.Fprintf(w, "Subject:     %s\n", *m.Subject)
	}
	_, _ = fmt.Fprintf(w, "Sent:        %s\n", formatUnix(m.SentAt))
	_, _ = fmt.Fprintf(w, "Last event:  %s\n", formatUnix(m.LastEventAt))

	for _, r := range a.Recipients {
		_, _ = fmt.Fprintf(w, "\n%s  [%s]\n", r.Recipient, sendEventLabel(r.CurrentStatus, r.BounceClass))
		for _, e := range r.Events {
			line := fmt.Sprintf("  %s  %-18s", formatUnix(e.Timestamp), sendEventLabel(e.EventType, e.BounceClass))
			if e.Reason != "" {
				line += "  " + e.Reason
			}
			_, _ = fmt.Fprintln(w, line)
		}
	}
}

// sendEventLabel shows the bounce class next to bounced events.
func sendEventLabel(eventType, bounceClass string) string {
	if bounceClass == "" {
		return eventType
	}
	return eventType + " (" + bounceClass + ")"
}
