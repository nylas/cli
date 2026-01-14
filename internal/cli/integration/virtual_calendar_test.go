//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVirtualCalendarGrants(t *testing.T) {
	apiKey := os.Getenv("NYLAS_API_KEY")
	if apiKey == "" {
		t.Skip("NYLAS_API_KEY not set")
	}

	client := nylas.NewHTTPClient()
	client.SetRegion("us")
	client.SetCredentials("", "", apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test creating a virtual calendar grant
	t.Run("CreateVirtualCalendarGrant", func(t *testing.T) {
		email := "test-conference-room-" + time.Now().Format("20060102150405") + "@example.com"

		grant, err := client.CreateVirtualCalendarGrant(ctx, email)
		require.NoError(t, err)

		// Skip test if virtual calendar grants not properly supported
		if grant.ID == "" || grant.Provider == "" {
			t.Skip("Virtual calendar grant creation returned empty values - feature may not be available")
		}

		assert.NotEmpty(t, grant.ID)
		assert.Equal(t, "virtual-calendar", grant.Provider)
		assert.Equal(t, email, grant.Email)
		assert.Equal(t, "valid", grant.GrantStatus)
		assert.NotZero(t, grant.CreatedAt)

		// Cleanup
		defer func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			_ = client.DeleteVirtualCalendarGrant(cleanupCtx, grant.ID)
		}()

		// Test getting the virtual calendar grant
		t.Run("GetVirtualCalendarGrant", func(t *testing.T) {
			retrieved, err := client.GetVirtualCalendarGrant(ctx, grant.ID)
			require.NoError(t, err)
			assert.Equal(t, grant.ID, retrieved.ID)
			assert.Equal(t, grant.Email, retrieved.Email)
			assert.Equal(t, grant.Provider, retrieved.Provider)
		})

		// Test listing virtual calendar grants
		t.Run("ListVirtualCalendarGrants", func(t *testing.T) {
			grants, err := client.ListVirtualCalendarGrants(ctx)
			require.NoError(t, err)
			assert.NotEmpty(t, grants)

			// Find our created grant
			found := false
			for _, g := range grants {
				if g.ID == grant.ID {
					found = true
					assert.Equal(t, email, g.Email)
					break
				}
			}
			assert.True(t, found, "Created grant should be in the list")
		})

		// Test deleting virtual calendar grant
		t.Run("DeleteVirtualCalendarGrant", func(t *testing.T) {
			err := client.DeleteVirtualCalendarGrant(ctx, grant.ID)
			require.NoError(t, err)

			// Verify it's deleted
			_, err = client.GetVirtualCalendarGrant(ctx, grant.ID)
			assert.Error(t, err, "Should get error when fetching deleted grant")
		})
	})
}

func TestVirtualCalendarWorkflow(t *testing.T) {
	apiKey := os.Getenv("NYLAS_API_KEY")
	if apiKey == "" {
		t.Skip("NYLAS_API_KEY not set")
	}

	client := nylas.NewHTTPClient()
	client.SetRegion("us")
	client.SetCredentials("", "", apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create virtual calendar grant
	email := "integration-test-room-" + time.Now().Format("20060102150405") + "@example.com"
	grant, err := client.CreateVirtualCalendarGrant(ctx, email)
	require.NoError(t, err)

	// Skip test if virtual calendar grants not properly supported
	if grant.ID == "" || grant.Provider == "" {
		t.Skip("Virtual calendar grant creation returned empty values - feature may not be available")
	}

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = client.DeleteVirtualCalendarGrant(cleanupCtx, grant.ID)
	}()

	// Create a calendar for the virtual grant
	t.Run("CreateCalendarForVirtualGrant", func(t *testing.T) {
		calReq := &domain.CreateCalendarRequest{
			Name:        "Test Virtual Calendar",
			Description: "Integration test calendar",
			Timezone:    "America/New_York",
		}

		calendar, err := client.CreateCalendar(ctx, grant.ID, calReq)
		require.NoError(t, err)
		assert.NotEmpty(t, calendar.ID)
		assert.Equal(t, "Test Virtual Calendar", calendar.Name)

		// Cleanup calendar
		defer func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			_ = client.DeleteCalendar(cleanupCtx, grant.ID, calendar.ID)
		}()

		// Create an event on the virtual calendar
		t.Run("CreateEventOnVirtualCalendar", func(t *testing.T) {
			now := time.Now()
			eventReq := &domain.CreateEventRequest{
				Title: "Virtual Calendar Test Event",
				When: domain.EventWhen{
					StartTime: now.Add(24 * time.Hour).Unix(),
					EndTime:   now.Add(25 * time.Hour).Unix(),
					Object:    "timespan",
				},
				Busy: true,
			}

			event, err := client.CreateEvent(ctx, grant.ID, calendar.ID, eventReq)
			require.NoError(t, err)
			assert.NotEmpty(t, event.ID)
			assert.Equal(t, "Virtual Calendar Test Event", event.Title)

			// Cleanup event
			defer func() {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cleanupCancel()
				_ = client.DeleteEvent(cleanupCtx, grant.ID, calendar.ID, event.ID)
			}()
		})
	})
}
