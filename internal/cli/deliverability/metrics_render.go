package deliverability

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// healthColor maps send health states and tile statuses to colors.
func healthColor(state string) *color.Color {
	switch state {
	case "healthy", "ok":
		return common.Green
	case "at_risk", "warn":
		return common.Yellow
	case "critical", "crit":
		return common.Red
	default:
		return common.Dim
	}
}

func windowString(w *domain.MetricsWindow) string {
	if w == nil {
		return "—"
	}
	return w.Start + " → " + w.End
}

func renderSendSummary(w io.Writer, s *domain.SendMetricsSummary) {
	_, _ = fmt.Fprintf(w, "Health:        %s (last 7 days, %s)\n",
		healthColor(s.Health.State).Sprint(s.Health.State), windowString(&s.Health.Window))
	if s.Health.State == "insufficient_data" {
		_, _ = fmt.Fprintf(w, "               fewer than %d sends in the last 7 days, so health is not graded\n", s.Health.MinAssessVolume)
	}
	_, _ = fmt.Fprintf(w, "Prior period:  %s\n\n", windowString(s.PriorPeriod))

	totals := newTable(w, "SENT", "DELIVERED", "HARD BOUNCES", "SOFT BOUNCES", "UNDETERMINED", "REJECTED", "COMPLAINTS")
	t := s.Totals
	totals.AddRow(formatCount(t.Sent), formatCount(t.Delivered), formatCount(t.BouncedHard), formatCount(t.BouncedSoft),
		formatCount(t.BouncedUndetermined), formatCount(t.Rejected), formatCount(t.Complaints))
	totals.Render()
	_, _ = fmt.Fprintln(w)

	rates := newTable(w, "RATE", "VALUE", "CHANGE", "PRIOR", "STATUS")
	addGradedTile(rates, "delivery", s.Metrics.DeliveryRate)
	addGradedTile(rates, "bounce (hard)", s.Metrics.BounceRate)
	addGradedTile(rates, "complaint", s.Metrics.ComplaintRate)
	rates.Render()

	if len(s.Health.Findings) > 0 {
		_, _ = fmt.Fprintln(w, "\nFindings:")
		for _, f := range s.Health.Findings {
			_, _ = fmt.Fprintf(w, "  [%s] %s\n", f.Severity, f.Title)
			if f.Recommendation != "" {
				_, _ = fmt.Fprintf(w, "         %s\n", f.Recommendation)
			}
		}
	}
}

func addGradedTile(table *common.Table, name string, tile domain.GradedRateTile) {
	status := "—"
	if tile.Status != nil {
		status = healthColor(*tile.Status).Sprint(*tile.Status)
	}
	table.AddRow(name, formatRate(tile.Value), formatDelta(tile.DeltaPP), formatRate(tile.PriorValue), status)
}

func renderSendTimeseries(w io.Writer, s *domain.SendMetricsTimeseries) {
	_, _ = fmt.Fprintf(w, "Granularity: %s\n\n", s.Granularity)
	table := newTable(w, "START", "END", "SENT", "DELIVERED", "HARD BOUNCES", "COMPLAINTS", "BOUNCE RATE", "COMPLAINT RATE")
	for _, b := range s.Buckets {
		table.AddRow(b.Start, b.End, formatCount(b.Sent), formatCount(b.Delivered), formatCount(b.BouncedHard),
			formatCount(b.Complaints), formatRate(b.BounceRate), formatRate(b.ComplaintRate))
	}
	table.Render()
}

func renderSendBreakdown(w io.Writer, b *domain.SendMetricsBreakdown) {
	_, _ = fmt.Fprintf(w, "Grouped by: %s\nWindow:     %s\n\n", b.GroupBy, windowString(b.Window))
	if b.Window == nil {
		_, _ = fmt.Fprintln(w, "No data: recipient_domain breakdowns only cover the last 90 days.")
		return
	}
	table := newTable(w, "KEY", "SENT", "DELIVERED", "HARD BOUNCES", "COMPLAINTS", "DELIVERY", "BOUNCE", "COMPLAINT")
	for _, r := range b.Rows {
		table.AddRow(r.Key, formatCount(r.Sent), formatCount(r.Delivered), formatCount(r.BouncedHard), formatCount(r.Complaints),
			formatRate(r.DeliveryRate), formatRate(r.BounceRate), formatRate(r.ComplaintRate))
	}
	table.Render()
}

func renderTrackingSummary(w io.Writer, s *domain.TrackingMetricsSummary) {
	_, _ = fmt.Fprintf(w, "Prior period:  %s\n\n", windowString(s.PriorPeriod))
	t := s.Totals
	totals := newTable(w, "TRACKED OPENS", "OPENED", "OPENS TOTAL", "TRACKED LINKS", "CLICKED", "CLICKS TOTAL")
	totals.AddRow(formatCount(t.TrackedOpens), formatCount(t.Opened), formatCount(t.OpensTotal),
		formatCount(t.TrackedLinks), formatCount(t.Clicked), formatCount(t.ClicksTotal))
	totals.Render()
	_, _ = fmt.Fprintln(w)

	rates := newTable(w, "RATE", "VALUE", "CHANGE", "PRIOR")
	for _, r := range []struct {
		name string
		tile domain.RateTile
	}{{"open", s.Metrics.OpenRate}, {"click", s.Metrics.ClickRate}} {
		rates.AddRow(r.name, formatRate(r.tile.Value), formatDelta(r.tile.DeltaPP), formatRate(r.tile.PriorValue))
	}
	rates.Render()
}

func renderTrackingTimeseries(w io.Writer, s *domain.TrackingMetricsTimeseries) {
	_, _ = fmt.Fprintf(w, "Granularity: %s\n\n", s.Granularity)
	table := newTable(w, "START", "END", "TRACKED OPENS", "OPENED", "OPEN RATE", "TRACKED LINKS", "CLICKED", "CLICK RATE")
	for _, b := range s.Buckets {
		table.AddRow(b.Start, b.End, formatCount(b.TrackedOpens), formatCount(b.Opened), formatRate(b.OpenRate),
			formatCount(b.TrackedLinks), formatCount(b.Clicked), formatRate(b.ClickRate))
	}
	table.Render()
}

func renderTrackingBreakdown(w io.Writer, b *domain.TrackingMetricsBreakdown) {
	_, _ = fmt.Fprintf(w, "Grouped by: %s\nWindow:     %s\n\n", b.GroupBy, windowString(b.Window))
	table := newTable(w, "KEY", "TRACKED OPENS", "OPENED", "OPEN RATE", "TRACKED LINKS", "CLICKED", "CLICK RATE")
	for _, r := range b.Rows {
		table.AddRow(r.Key, formatCount(r.TrackedOpens), formatCount(r.Opened), formatRate(r.OpenRate),
			formatCount(r.TrackedLinks), formatCount(r.Clicked), formatRate(r.ClickRate))
	}
	table.Render()
}
