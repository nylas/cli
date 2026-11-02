// Package deliverability implements `nylas deliverability`: send and tracking
// activity, send and tracking metrics, and send alert rules.
package deliverability

import "github.com/spf13/cobra"

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
Activity covers the last 90 days. Metrics accept windows of up to 365 days.`,
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
		Long: `Manage send alert rules. Each rule is evaluated per sending domain. When it fires,
Nylas sends the alert.triggered webhook to your application's webhook subscriptions.
An application can have at most 20 rules.`,
	}
	cmd.AddCommand(newSendAlertsListCmd())
	cmd.AddCommand(newSendAlertsShowCmd())
	cmd.AddCommand(newSendAlertsCreateCmd())
	cmd.AddCommand(newSendAlertsUpdateCmd())
	cmd.AddCommand(newSendAlertsDeleteCmd())
	return cmd
}
