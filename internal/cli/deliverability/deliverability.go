// Package deliverability implements `nylas deliverability`: send and tracking
// activity, send and tracking metrics, and send alert rules.
package deliverability

import (
	"strings"

	"github.com/spf13/cobra"
)

// scopeNote matches the API reference: these routes cover Agent Account and Transactional send only.
const scopeNote = "Covers only messages sent from an Agent Account or through Transactional send. " +
	"Messages from connected grants, such as Google or Microsoft accounts, are not included."

// trackingNote narrows scopeNote for tracking events.
const trackingNote = "Only messages sent with open or link tracking turned on have tracking events."

// longHelp builds a command's Long help: the short description, the scope note, then any extra lines.
func longHelp(short string, extra ...string) string {
	return strings.Join(append([]string{short, scopeNote}, extra...), "\n\n")
}

// NewDeliverabilityCmd creates the deliverability command group.
func NewDeliverabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deliverability",
		Short: "Inspect send and tracking events, metrics, and send alerts",
		Long: `Inspect what happened to the email your application sends.

  send-events      Send activity (accepted, delivered, bounced, complaint, rejected) and send metrics
  tracking-events  Open and link-click activity and engagement metrics
  send-alerts      Rules that fire the alert.triggered webhook when a send rate crosses a threshold

These commands are application-level: they use your API key and do not take a grant.
Activity covers the last 90 days. Metrics accept windows of up to 365 days.

` + scopeNote,
	}

	cmd.AddCommand(newSendEventsCmd())
	cmd.AddCommand(newTrackingEventsCmd())
	cmd.AddCommand(newSendAlertsCmd())

	return cmd
}

func newSendEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-events",
		Short: "Send activity and send metrics",
		Long:  longHelp("Send activity and send metrics."),
	}
	cmd.AddCommand(newSendEventsListCmd())
	cmd.AddCommand(newSendEventsShowCmd())
	cmd.AddCommand(newSendEventsMessageCmd())
	cmd.AddCommand(newSendSummaryCmd())
	cmd.AddCommand(newSendTimeseriesCmd())
	cmd.AddCommand(newSendBreakdownCmd())
	return cmd
}

func newTrackingEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tracking-events",
		Short: "Open and click activity and engagement metrics",
		Long:  longHelp("Open and click activity and engagement metrics.", trackingNote),
	}
	cmd.AddCommand(newTrackingEventsListCmd())
	cmd.AddCommand(newTrackingEventsShowCmd())
	cmd.AddCommand(newTrackingEventsMessageCmd())
	cmd.AddCommand(newTrackingSummaryCmd())
	cmd.AddCommand(newTrackingTimeseriesCmd())
	cmd.AddCommand(newTrackingBreakdownCmd())
	return cmd
}

func newSendAlertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-alerts",
		Short: "Manage send alert rules",
		Long: longHelp(`Manage send alert rules. Each rule is evaluated per sending domain. When it fires,
Nylas sends the alert.triggered webhook to your application's webhook subscriptions.
An application can have at most 20 rules.`, "Rules are evaluated for those messages only."),
	}
	cmd.AddCommand(newSendAlertsListCmd())
	cmd.AddCommand(newSendAlertsShowCmd())
	cmd.AddCommand(newSendAlertsCreateCmd())
	cmd.AddCommand(newSendAlertsUpdateCmd())
	cmd.AddCommand(newSendAlertsDeleteCmd())
	return cmd
}
