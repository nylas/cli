package deliverability

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newTrackingEventsListCmd() *cobra.Command {
	var window activityWindowFlags
	var params domain.TrackingEventListParams

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List opens and link clicks (newest first)",
		Example: `  nylas deliverability tracking-events list --event-type link_clicked --start 7d
  nylas deliverability tracking-events list --label welcome-campaign --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			if params.Start, params.End, err = window.resolve(time.Now(), params.PageToken); err != nil {
				return err
			}
			_, err = common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				return struct{}{}, runTrackingEventsList(ctx, cmd, client, &params, window.pageFlags)
			})
			return err
		},
	}

	cmd.Flags().StringArrayVar(&params.EventTypes, "event-type", nil, "Event type, repeatable: opened, link_clicked")
	cmd.Flags().StringVar(&params.SendingDomain, "sending-domain", "", "Sending domain")
	cmd.Flags().StringVar(&params.From, "from", "", "From address")
	cmd.Flags().StringVar(&params.GrantID, "grant-id", "", "Only events for this grant (agent account)")
	cmd.Flags().StringVar(&params.MessageID, "message-id", "", "Message ID")
	cmd.Flags().StringVar(&params.ThreadID, "thread-id", "", "Thread ID")
	cmd.Flags().StringVar(&params.Label, "label", "", "Tracking label set when the message was sent")
	addActivityWindowFlags(cmd, &window, &params.Order, &params.Limit, &params.PageToken)

	return cmd
}

func runTrackingEventsList(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.TrackingEventListParams, pageFlags string) error {
	resp, err := client.ListTrackingEvents(ctx, params)
	if err != nil {
		return common.WrapListError("tracking events", err)
	}
	if done, err := writeStructuredList(cmd, resp, resp.Data); done {
		return err
	}
	if len(resp.Data) == 0 {
		common.PrintEmptyStateWithHint("tracking events", "Widen the window with --start (activity is kept for 90 days)")
		return nil
	}

	table := newTable(cmd.OutOrStdout(), "TIME", "EVENT", "FIRST", "FROM", "LABEL", "URL", "EVENT ID")
	table.SetMaxWidth(5, 40)
	for _, e := range resp.Data {
		table.AddRow(formatUnix(e.Timestamp), e.EventType, yesNo(e.IsFirst), e.From, e.Label, derefString(e.URL), e.ID)
	}
	table.Render()
	nextCursorNote(cmd, resp.NextCursor, pageFlags)
	return nil
}

func newTrackingEventsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <event-id>",
		Short: "Show one open or link click",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateEventID(args[0]); err != nil {
				return err
			}
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				event, err := client.GetTrackingEvent(ctx, args[0])
				if err != nil {
					return struct{}{}, common.WrapGetError("tracking event", err)
				}
				if done, err := writeStructured(cmd, event); done {
					return struct{}{}, err
				}
				printTrackingEvent(cmd, event)
				return struct{}{}, nil
			})
			return err
		},
	}
}

func printTrackingEvent(cmd *cobra.Command, e *domain.TrackingEvent) {
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "Event ID:    %s\n", e.ID)
	_, _ = fmt.Fprintf(w, "Event:       %s\n", e.EventType)
	_, _ = fmt.Fprintf(w, "Time:        %s\n", formatUnix(e.Timestamp))
	_, _ = fmt.Fprintf(w, "First:       %s\n", yesNo(e.IsFirst))
	_, _ = fmt.Fprintf(w, "Message ID:  %s\n", e.MessageID)
	_, _ = fmt.Fprintf(w, "From:        %s\n", e.From)
	if e.Label != "" {
		_, _ = fmt.Fprintf(w, "Label:       %s\n", e.Label)
	}
	if e.URL != nil {
		_, _ = fmt.Fprintf(w, "URL:         %s\n", *e.URL)
	}
	if e.LinkIndex != nil {
		_, _ = fmt.Fprintf(w, "Link index:  %d\n", *e.LinkIndex)
	}
	if e.IP != "" {
		_, _ = fmt.Fprintf(w, "IP:          %s\n", e.IP)
	}
	if e.UserAgent != "" {
		_, _ = fmt.Fprintf(w, "User agent:  %s\n", e.UserAgent)
	}
}

func newTrackingEventsMessageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "message <message-id>",
		Short: "Show open and click totals and the newest 100 interactions for one message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				activity, err := client.GetTrackingMessageActivity(ctx, args[0])
				if err != nil {
					return struct{}{}, common.WrapGetError("tracking activity for message", err)
				}
				if done, err := writeStructured(cmd, activity); done {
					return struct{}{}, err
				}
				printTrackingMessageActivity(cmd, activity)
				return struct{}{}, nil
			})
			return err
		},
	}
}

func printTrackingMessageActivity(cmd *cobra.Command, a *domain.TrackingMessageActivity) {
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "Message ID:  %s\n", a.MessageID)
	_, _ = fmt.Fprintf(w, "From:        %s\n", a.From)
	if a.Label != "" {
		_, _ = fmt.Fprintf(w, "Label:       %s\n", a.Label)
	}
	_, _ = fmt.Fprintf(w, "Tracking:    opens %s, links %s\n", onOff(a.Tracking.Opens), onOff(a.Tracking.Links))
	_, _ = fmt.Fprintf(w, "Opens:       %d%s\n", a.Opens.Total, totalsRange(a.Opens))
	_, _ = fmt.Fprintf(w, "Clicks:      %d%s\n", a.Clicks.Total, totalsRange(a.Clicks.TrackingTotals))

	if len(a.Clicks.Links) > 0 {
		_, _ = fmt.Fprintln(w)
		table := newTable(w, "LINK", "URL", "CLICKS")
		table.AlignRight(2)
		for _, l := range a.Clicks.Links {
			table.AddRow(formatCount(l.LinkIndex), l.URL, formatCount(l.Clicks))
		}
		table.Render()
	}

	if len(a.Events) > 0 {
		_, _ = fmt.Fprintln(w)
		table := newTable(w, "TIME", "EVENT", "FIRST", "URL", "USER AGENT")
		table.SetMaxWidth(3, 40).SetMaxWidth(4, 40)
		for _, e := range a.Events {
			table.AddRow(formatUnix(e.Timestamp), e.EventType, yesNo(e.IsFirst), derefString(e.URL), e.UserAgent)
		}
		table.Render()
	}
}

func totalsRange(t domain.TrackingTotals) string {
	if t.FirstAt == nil || t.LastAt == nil {
		return ""
	}
	return fmt.Sprintf(" (first %s, last %s)", formatUnix(*t.FirstAt), formatUnix(*t.LastAt))
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
