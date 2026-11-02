package deliverability

import (
	"context"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

type metricsRunner func(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.MetricsParams) error

// newMetricsCmd builds a metrics command with the shared --start/--end/--sending-domain flags.
func newMetricsCmd(use, short, long, example string, run metricsRunner, extraFlags func(*cobra.Command, *domain.MetricsParams)) *cobra.Command {
	var params domain.MetricsParams
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateMetricsDate("start", params.Start); err != nil {
				return err
			}
			if err := validateMetricsDate("end", params.End); err != nil {
				return err
			}
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				return struct{}{}, run(ctx, cmd, client, &params)
			})
			return err
		},
	}
	cmd.Flags().StringVar(&params.Start, "start", "", "First UTC day, YYYY-MM-DD (default: 6 days before --end)")
	cmd.Flags().StringVar(&params.End, "end", "", "Last UTC day, YYYY-MM-DD, inclusive (default: today)")
	cmd.Flags().StringVar(&params.SendingDomain, "sending-domain", "", "Only this sending domain")
	if extraFlags != nil {
		extraFlags(cmd, &params)
	}
	return cmd
}

func addGranularityFlag(cmd *cobra.Command, params *domain.MetricsParams) {
	cmd.Flags().StringVar(&params.Granularity, "granularity", "",
		"Bucket size: "+strings.Join(domain.MetricsGranularities, ", ")+" (default auto: day ≤21 days, week ≤92, else month)")
}

func addBreakdownFlags(groupBy []string, sortKeys string) func(*cobra.Command, *domain.MetricsParams) {
	return func(cmd *cobra.Command, params *domain.MetricsParams) {
		cmd.Flags().StringVar(&params.GroupBy, "group-by", "", "Dimension: "+strings.Join(groupBy, ", ")+" (default sending_domain)")
		cmd.Flags().StringVar(&params.Sort, "sort", "", "Sort as <key>:<asc|desc>; keys: "+sortKeys)
		cmd.Flags().IntVarP(&params.Limit, "limit", "n", 25, "Rows to show before the rest are summed into (others), 1-100")
	}
}

func newSendSummaryCmd() *cobra.Command {
	return newMetricsCmd("summary", "Show send health, totals, and rates",
		longHelp("Show send health, totals, and rates. Health always grades the last 7 days."),
		`  nylas deliverability send-events summary
  nylas deliverability send-events summary --start 2026-10-01 --end 2026-10-31 --sending-domain mail.example.com`,
		runSendSummary, nil)
}

func newSendTimeseriesCmd() *cobra.Command {
	return newMetricsCmd("timeseries", "Show send counts and rates per day, week, or month",
		longHelp("Show send counts and rates per day, week, or month."),
		`  nylas deliverability send-events timeseries --granularity week --start 2026-08-01`,
		runSendTimeseries, addGranularityFlag)
}

func newSendBreakdownCmd() *cobra.Command {
	return newMetricsCmd("breakdown", "Show send counts and rates per sending domain, agent account, or recipient domain",
		longHelp("Show send counts and rates per sending domain, agent account, or recipient domain."),
		`  nylas deliverability send-events breakdown --group-by recipient_domain --sort bounce_rate:desc`,
		runSendBreakdown, addBreakdownFlags(domain.SendMetricsGroupBy, "key, sent, delivery_rate, bounce_rate, complaint_rate (default sent:desc)"))
}

func newTrackingSummaryCmd() *cobra.Command {
	return newMetricsCmd("summary", "Show engagement totals and open and click rates",
		longHelp("Show engagement totals and open and click rates.", trackingNote),
		`  nylas deliverability tracking-events summary --start 2026-10-01`,
		runTrackingSummary, nil)
}

func newTrackingTimeseriesCmd() *cobra.Command {
	return newMetricsCmd("timeseries", "Show engagement counts and rates per day, week, or month",
		longHelp("Show engagement counts and rates per day, week, or month.", trackingNote),
		`  nylas deliverability tracking-events timeseries --granularity day`,
		runTrackingTimeseries, addGranularityFlag)
}

func newTrackingBreakdownCmd() *cobra.Command {
	return newMetricsCmd("breakdown", "Show engagement per sending domain or agent account",
		longHelp("Show engagement per sending domain or agent account.", trackingNote),
		`  nylas deliverability tracking-events breakdown --group-by agent_account --sort open_rate:desc`,
		runTrackingBreakdown, addBreakdownFlags(domain.TrackingMetricsGroupBy, "key, tracked_opens, tracked_links, open_rate, click_rate (default tracked_opens:desc)"))
}

func runSendSummary(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.MetricsParams) error {
	summary, err := client.GetSendMetricsSummary(ctx, params)
	if err != nil {
		return common.WrapGetError("send metrics summary", err)
	}
	if done, err := writeStructured(cmd, summary); done {
		return err
	}
	renderSendSummary(cmd.OutOrStdout(), summary)
	return nil
}

func runSendTimeseries(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.MetricsParams) error {
	series, err := client.GetSendMetricsTimeseries(ctx, params)
	if err != nil {
		return common.WrapGetError("send metrics timeseries", err)
	}
	if done, err := writeStructured(cmd, series); done {
		return err
	}
	renderSendTimeseries(cmd.OutOrStdout(), series)
	return nil
}

func runSendBreakdown(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.MetricsParams) error {
	breakdown, err := client.GetSendMetricsBreakdown(ctx, params)
	if err != nil {
		return common.WrapGetError("send metrics breakdown", err)
	}
	if done, err := writeStructured(cmd, breakdown); done {
		return err
	}
	renderSendBreakdown(cmd.OutOrStdout(), breakdown)
	return nil
}

func runTrackingSummary(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.MetricsParams) error {
	summary, err := client.GetTrackingMetricsSummary(ctx, params)
	if err != nil {
		return common.WrapGetError("tracking metrics summary", err)
	}
	if done, err := writeStructured(cmd, summary); done {
		return err
	}
	renderTrackingSummary(cmd.OutOrStdout(), summary)
	return nil
}

func runTrackingTimeseries(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.MetricsParams) error {
	series, err := client.GetTrackingMetricsTimeseries(ctx, params)
	if err != nil {
		return common.WrapGetError("tracking metrics timeseries", err)
	}
	if done, err := writeStructured(cmd, series); done {
		return err
	}
	renderTrackingTimeseries(cmd.OutOrStdout(), series)
	return nil
}

func runTrackingBreakdown(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, params *domain.MetricsParams) error {
	breakdown, err := client.GetTrackingMetricsBreakdown(ctx, params)
	if err != nil {
		return common.WrapGetError("tracking metrics breakdown", err)
	}
	if done, err := writeStructured(cmd, breakdown); done {
		return err
	}
	renderTrackingBreakdown(cmd.OutOrStdout(), breakdown)
	return nil
}
