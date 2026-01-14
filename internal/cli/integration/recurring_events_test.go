//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecurringEventOperations(t *testing.T) {
	apiKey := os.Getenv("NYLAS_API_KEY")
	grantID := os.Getenv("NYLAS_GRANT_ID")

	if apiKey == "" || grantID == "" {
		t.Skip("NYLAS_API_KEY and NYLAS_GRANT_ID required")
	}

	client := nylas.NewHTTPClient()
	client.SetRegion("us")
	client.SetCredentials("", "", apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Get a calendar to work with
	calendars, err := client.GetCalendars(ctx, grantID)
	if err != nil {
		t.Skipf("Could not get calendars: %v", err)
	}
	if len(calendars) == 0 {
		t.Skip("No calendars available for testing")
	}
	calendarID := calendars[0].ID

	// Create a recurring event
	t.Run("CreateRecurringEvent", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(24 * time.Hour)
		endTime := startTime.Add(1 * time.Hour)

		eventReq := &domain.CreateEventRequest{
			Title:       "Integration Test Recurring Meeting",
			Description: "Weekly recurring event for integration testing",
			When: domain.EventWhen{
				StartTime: startTime.Unix(),
				EndTime:   endTime.Unix(),
				Object:    "timespan",
			},
			Recurrence: []string{"RRULE:FREQ=WEEKLY;COUNT=5;BYDAY=MO"},
			Busy:       true,
		}

		event, err := client.CreateEvent(ctx, grantID, calendarID, eventReq)
		if err != nil {
			errStr := err.Error()
			// Skip if calendar not found, forbidden, or not supported
			if strings.Contains(errStr, "Not Found") || strings.Contains(errStr, "not found") ||
				strings.Contains(errStr, "Forbidden") || strings.Contains(errStr, "forbidden") {
				t.Skipf("Calendar not available or operation not permitted: %v", err)
			}
			require.NoError(t, err)
		}
		assert.NotEmpty(t, event.ID)
		assert.Equal(t, "Integration Test Recurring Meeting", event.Title)
		assert.NotEmpty(t, event.Recurrence)

		masterEventID := event.ID

		// Cleanup
		defer func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			_ = client.DeleteEvent(cleanupCtx, grantID, calendarID, masterEventID)
		}()

		// Test getting recurring event instances
		t.Run("GetRecurringEventInstances", func(t *testing.T) {
			params := &domain.EventQueryParams{
				ExpandRecurring: true,
				Limit:           10,
			}

			instances, err := client.GetRecurringEventInstances(ctx, grantID, calendarID, masterEventID, params)
			require.NoError(t, err)
			assert.NotEmpty(t, instances, "Should have at least one instance")

			// Verify all instances share the same master event ID
			for _, instance := range instances {
				assert.Equal(t, masterEventID, instance.MasterEventID,
					"All instances should have the same master_event_id")
			}

			// Test updating a single instance
			if len(instances) > 0 {
				t.Run("UpdateRecurringEventInstance", func(t *testing.T) {
					instanceID := instances[0].ID
					newTitle := "Updated Instance - Integration Test"

					updateReq := &domain.UpdateEventRequest{
						Title: &newTitle,
					}

					updated, err := client.UpdateRecurringEventInstance(ctx, grantID, calendarID, instanceID, updateReq)
					require.NoError(t, err)
					assert.Equal(t, newTitle, updated.Title)
					assert.Equal(t, instanceID, updated.ID)
				})

				// Test deleting a single instance
				if len(instances) > 1 {
					t.Run("DeleteRecurringEventInstance", func(t *testing.T) {
						instanceID := instances[1].ID

						err := client.DeleteRecurringEventInstance(ctx, grantID, calendarID, instanceID)
						require.NoError(t, err)

						// Verify the instance is deleted
						remainingInstances, err := client.GetRecurringEventInstances(ctx, grantID, calendarID, masterEventID, params)
						require.NoError(t, err)

						// Check that the deleted instance is not in the list
						found := false
						for _, inst := range remainingInstances {
							if inst.ID == instanceID {
								found = true
								break
							}
						}
						assert.False(t, found, "Deleted instance should not be in the list")
					})
				}
			}
		})
	})
}

func TestRecurringEventPatterns(t *testing.T) {
	apiKey := os.Getenv("NYLAS_API_KEY")
	grantID := os.Getenv("NYLAS_GRANT_ID")

	if apiKey == "" || grantID == "" {
		t.Skip("NYLAS_API_KEY and NYLAS_GRANT_ID required")
	}

	client := nylas.NewHTTPClient()
	client.SetRegion("us")
	client.SetCredentials("", "", apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Get a calendar
	calendars, err := client.GetCalendars(ctx, grantID)
	if err != nil {
		t.Skipf("Could not get calendars: %v", err)
	}
	if len(calendars) == 0 {
		t.Skip("No calendars available for testing")
	}
	calendarID := calendars[0].ID

	now := time.Now()
	startTime := now.Add(24 * time.Hour)
	endTime := startTime.Add(1 * time.Hour)

	testCases := []struct {
		name       string
		title      string
		recurrence []string
	}{
		{
			name:       "DailyRecurrence",
			title:      "Daily Standup",
			recurrence: []string{"RRULE:FREQ=DAILY;COUNT=3"},
		},
		{
			name:       "WeeklyRecurrence",
			title:      "Weekly Team Meeting",
			recurrence: []string{"RRULE:FREQ=WEEKLY;COUNT=3;BYDAY=TU"},
		},
		{
			name:       "MonthlyRecurrence",
			title:      "Monthly Review",
			recurrence: []string{"RRULE:FREQ=MONTHLY;COUNT=2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eventReq := &domain.CreateEventRequest{
				Title: tc.title,
				When: domain.EventWhen{
					StartTime: startTime.Unix(),
					EndTime:   endTime.Unix(),
					Object:    "timespan",
				},
				Recurrence: tc.recurrence,
				Busy:       true,
			}

			event, err := client.CreateEvent(ctx, grantID, calendarID, eventReq)
			if err != nil {
				errStr := err.Error()
				// Skip if calendar not found, forbidden, or not supported
				if strings.Contains(errStr, "Not Found") || strings.Contains(errStr, "not found") ||
					strings.Contains(errStr, "Forbidden") || strings.Contains(errStr, "forbidden") {
					t.Skipf("Calendar not available or operation not permitted: %v", err)
				}
				require.NoError(t, err)
			}
			assert.NotEmpty(t, event.ID)

			// Cleanup
			defer func() {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cleanupCancel()
				_ = client.DeleteEvent(cleanupCtx, grantID, calendarID, event.ID)
			}()

			// Get instances
			params := &domain.EventQueryParams{
				ExpandRecurring: true,
				Limit:           10,
			}

			instances, err := client.GetRecurringEventInstances(ctx, grantID, calendarID, event.ID, params)
			require.NoError(t, err)
			assert.NotEmpty(t, instances, "Should have recurring instances")
		})
	}
}
