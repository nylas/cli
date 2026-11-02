//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// skipIfMissingAppCreds skips tests of application-level routes, which need an API key but no grant.
func skipIfMissingAppCreds(t *testing.T) {
	t.Helper()
	if testBinary == "" {
		t.Skip("CLI binary not found - run 'go build -o bin/nylas ./cmd/nylas' first")
	}
	if testAPIKey == "" {
		t.Skip("NYLAS_API_KEY not set")
	}
}

func TestCLI_DeliverabilityHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("deliverability", "--help")
	if err != nil {
		t.Fatalf("deliverability --help failed: %v\nstderr: %s", err, stderr)
	}
	for _, expected := range []string{"send-events", "tracking-events", "send-alerts"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected help output to contain %q, got: %s", expected, stdout)
		}
	}
}

// TestCLI_DeliverabilityReads checks each read command decodes a live response. The app may have no
// recent sends, so the tests assert response shape, not content.
func TestCLI_DeliverabilityReads(t *testing.T) {
	skipIfMissingAppCreds(t)

	tests := []struct {
		name   string
		args   []string
		decode any
		check  func(t *testing.T, v any)
	}{
		{"send-events list", []string{"deliverability", "send-events", "list", "--limit", "5", "--json"},
			&domain.SendEventListResponse{}, func(t *testing.T, v any) {
				if v.(*domain.SendEventListResponse).Data == nil {
					t.Fatal("send events list has no data array")
				}
			}},
		{"tracking-events list", []string{"deliverability", "tracking-events", "list", "--limit", "5", "--json"},
			&domain.TrackingEventListResponse{}, func(t *testing.T, v any) {
				if v.(*domain.TrackingEventListResponse).Data == nil {
					t.Fatal("tracking events list has no data array")
				}
			}},
		{"send-events summary", []string{"deliverability", "send-events", "summary", "--json"},
			&domain.SendMetricsSummary{}, func(t *testing.T, v any) {
				if v.(*domain.SendMetricsSummary).Health.State == "" {
					t.Fatal("send summary has no health state")
				}
			}},
		{"send-events timeseries", []string{"deliverability", "send-events", "timeseries", "--json"},
			&domain.SendMetricsTimeseries{}, func(t *testing.T, v any) {
				// The default 7-day window resolves to daily buckets, one per day, empty or not.
				s := v.(*domain.SendMetricsTimeseries)
				if s.Granularity != "day" || len(s.Buckets) != 7 {
					t.Fatalf("expected 7 daily buckets, got %q with %d", s.Granularity, len(s.Buckets))
				}
			}},
		{"send-events breakdown", []string{"deliverability", "send-events", "breakdown", "--json"},
			&domain.SendMetricsBreakdown{}, func(t *testing.T, v any) {
				if v.(*domain.SendMetricsBreakdown).GroupBy != "sending_domain" {
					t.Fatal("send breakdown did not default to sending_domain")
				}
			}},
		{"tracking-events summary", []string{"deliverability", "tracking-events", "summary", "--json"},
			&domain.TrackingMetricsSummary{}, func(t *testing.T, v any) {}},
		{"tracking-events breakdown", []string{"deliverability", "tracking-events", "breakdown", "--group-by", "agent_account", "--json"},
			&domain.TrackingMetricsBreakdown{}, func(t *testing.T, v any) {
				if v.(*domain.TrackingMetricsBreakdown).GroupBy != "agent_account" {
					t.Fatal("tracking breakdown ignored --group-by")
				}
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acquireRateLimit(t)
			stdout, stderr, err := runCLI(tt.args...)
			if err != nil {
				t.Fatalf("%s failed: %v\nstderr: %s", tt.name, err, stderr)
			}
			if err := json.Unmarshal([]byte(stdout), tt.decode); err != nil {
				t.Fatalf("%s returned invalid JSON: %v\n%s", tt.name, err, stdout)
			}
			tt.check(t, tt.decode)
		})
	}
}

func TestCLI_SendAlertRuleLifecycle(t *testing.T) {
	skipIfMissingAppCreds(t)

	acquireRateLimit(t)
	stdout, stderr, err := runCLI("deliverability", "send-alerts", "list", "--json")
	if err != nil {
		t.Fatalf("send-alerts list failed: %v\nstderr: %s", err, stderr)
	}
	var existing []domain.SendAlertRule
	if err := json.Unmarshal([]byte(stdout), &existing); err != nil {
		t.Fatalf("send-alerts list returned invalid JSON: %v\n%s", err, stdout)
	}
	if len(existing) >= 20 {
		t.Skip("application already has the maximum of 20 send alert rules")
	}

	acquireRateLimit(t)
	stdout, stderr, err = runCLI("deliverability", "send-alerts", "create",
		"--metric", "bounce_rate", "--threshold", "99.5", "--window-hours", "24", "--min-volume", "1000000", "--json")
	if err != nil {
		t.Fatalf("send-alerts create failed: %v\nstderr: %s", err, stderr)
	}
	var created domain.SendAlertRule
	if err := json.Unmarshal([]byte(stdout), &created); err != nil {
		t.Fatalf("send-alerts create returned invalid JSON: %v\n%s", err, stdout)
	}
	t.Cleanup(func() {
		acquireRateLimit(t)
		_, _, _ = runCLI("deliverability", "send-alerts", "delete", created.ID, "--yes")
	})
	if created.Direction != "above" || created.Threshold != 99.5 || created.WindowHours != 24 {
		t.Fatalf("created rule does not match flags: %+v", created)
	}

	// A partial update must change only the given field and keep the rest.
	acquireRateLimit(t)
	stdout, stderr, err = runCLI("deliverability", "send-alerts", "update", created.ID, "--threshold", "98.25", "--json")
	if err != nil {
		t.Fatalf("send-alerts update failed: %v\nstderr: %s", err, stderr)
	}
	var updated domain.SendAlertRule
	if err := json.Unmarshal([]byte(stdout), &updated); err != nil {
		t.Fatalf("send-alerts update returned invalid JSON: %v\n%s", err, stdout)
	}
	if updated.Threshold != 98.25 || updated.WindowHours != 24 || updated.MinVolume != 1000000 {
		t.Fatalf("partial update changed other fields: %+v", updated)
	}

	acquireRateLimit(t)
	if _, stderr, err := runCLI("deliverability", "send-alerts", "show", created.ID); err != nil {
		t.Fatalf("send-alerts show failed: %v\nstderr: %s", err, stderr)
	}

	acquireRateLimit(t)
	if _, stderr, err := runCLI("deliverability", "send-alerts", "delete", created.ID, "--yes"); err != nil {
		t.Fatalf("send-alerts delete failed: %v\nstderr: %s", err, stderr)
	}

	acquireRateLimit(t)
	if _, _, err := runCLI("deliverability", "send-alerts", "show", created.ID); err == nil {
		t.Fatal("deleted rule can still be shown")
	}
}
