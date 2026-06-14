//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCLI_CalendarResourcesHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("calendar", "resources", "--help")
	if err != nil {
		t.Fatalf("calendar resources --help failed: %v\nstderr: %s", err, stderr)
	}
	// The command's value proposition (email doubles as calendar ID) must surface.
	if !strings.Contains(stdout, "calendar ID") {
		t.Errorf("expected resources help to mention 'calendar ID', got: %s", stdout)
	}
}

// TestCalendarResources_Integration exercises the read-only room-resources
// endpoint against the live API.
func TestCalendarResources_Integration(t *testing.T) {
	skipIfMissingCreds(t)
	client := getTestClient()
	acquireRateLimit(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resources, err := client.ListRoomResources(ctx, testGrantID)
	if err != nil {
		// Not every account/provider exposes room resources.
		if isUnavailableErr(err) {
			t.Skipf("room resources not available for this account: %v", err)
		}
		t.Fatalf("ListRoomResources() error = %v", err)
	}
	// A returned resource must carry the email that doubles as its calendar ID.
	for _, r := range resources {
		if r.Email == "" {
			t.Errorf("room resource missing email (its calendar ID): %+v", r)
		}
	}
	t.Logf("ListRoomResources returned %d resource(s)", len(resources))
}
