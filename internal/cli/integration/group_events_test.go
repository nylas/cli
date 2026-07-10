//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCLI_SchedulerGroupEventsHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}
	stdout, stderr, err := runCLI("scheduler", "group-events", "--help")
	if err != nil {
		t.Fatalf("scheduler group-events --help failed: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{"list", "create", "update", "delete", "import"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected group-events help to list subcommand %q, got: %s", want, stdout)
		}
	}
}

// TestCLI_GroupEventsListGuard verifies the list command requires --calendar
// (the API needs calendar_id/start_time/end_time) before any API call.
func TestCLI_GroupEventsListGuard(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}
	_, _, err := runCLI("scheduler", "group-events", "list", "cfg-1")
	if err == nil {
		t.Error("expected error when --calendar is omitted from group-events list")
	}
}

// TestGroupEvents_Integration lists group events for an existing Scheduler
// configuration. Skips when no configuration is set up for the account.
func TestGroupEvents_Integration(t *testing.T) {
	skipIfMissingCreds(t)
	client := getTestClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	acquireRateLimit(t)
	configs, err := client.ListSchedulerConfigurations(ctx, testGrantID)
	if err != nil {
		if isUnavailableErr(err) {
			t.Skipf("scheduler not available for this account: %v", err)
		}
		t.Fatalf("ListSchedulerConfigurations() error = %v", err)
	}
	if len(configs) == 0 {
		t.Skip("no scheduler configurations available to list group events")
	}

	now := time.Now()
	acquireRateLimit(t)
	events, err := client.ListGroupEvents(ctx, testGrantID, configs[0].ID, "primary", now.Unix(), now.AddDate(0, 1, 0).Unix())
	if err != nil {
		if isUnavailableErr(err) {
			t.Skipf("group events not available for this configuration: %v", err)
		}
		t.Fatalf("ListGroupEvents() error = %v", err)
	}
	t.Logf("ListGroupEvents returned %d event(s) for config %s", len(events), configs[0].ID)
}
