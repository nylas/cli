//go:build integration
// +build integration

package nylas_test

import (
	"os"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ListScheduledMessages(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	scheduled, err := client.ListScheduledMessages(ctx, grantID)
	require.NoError(t, err)

	t.Logf("Found %d scheduled message(s)", len(scheduled))
	for _, s := range scheduled {
		t.Logf("  Schedule ID: %s, Status: %s, CloseTime: %d",
			s.ScheduleID, s.Status, s.CloseTime)
	}
}

func TestIntegration_GetScheduledMessage_NotFound(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Try to get a non-existent scheduled message
	_, err := client.GetScheduledMessage(ctx, grantID, "nonexistent-schedule-id")
	require.Error(t, err)
	t.Logf("Expected error for non-existent schedule: %v", err)
}

// =============================================================================
// Notetaker Tests
// =============================================================================

func TestIntegration_ListNotetakers(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	notetakers, err := client.ListNotetakers(ctx, grantID, nil)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)

	t.Logf("Found %d notetaker(s)", len(notetakers))
	for _, nt := range notetakers {
		t.Logf("  ID: %s, State: %s, Meeting: %s",
			safeSubstring(nt.ID, 12), nt.State, safeSubstring(nt.MeetingTitle, 30))
	}
}

func TestIntegration_ListNotetakers_WithParams(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Test with limit
	params := &domain.NotetakerQueryParams{
		Limit: 5,
	}

	notetakers, err := client.ListNotetakers(ctx, grantID, params)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)

	assert.LessOrEqual(t, len(notetakers), 5, "Should respect limit")
	t.Logf("Found %d notetaker(s) with limit=5", len(notetakers))
}

func TestIntegration_ListNotetakers_ByState(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	// Only test with states known to be valid in the API
	states := []string{"scheduled", "attending", "media_processing"}

	for _, state := range states {
		t.Run("State_"+state, func(t *testing.T) {
			params := &domain.NotetakerQueryParams{
				Limit: 10,
				State: state,
			}

			notetakers, err := client.ListNotetakers(ctx, grantID, params)
			// Skip if state is not supported (API returns "invalid state" error)
			if err != nil && strings.Contains(err.Error(), "invalid state") {
				t.Skipf("State %s not supported by API: %v", state, err)
			}
			skipIfProviderNotSupported(t, err)
			require.NoError(t, err)

			t.Logf("Found %d notetaker(s) with state=%s", len(notetakers), state)
			for _, nt := range notetakers {
				assert.Equal(t, state, nt.State, "Notetaker should have requested state")
			}
		})
	}
}

func TestIntegration_GetNotetaker_NotFound(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	_, err := client.GetNotetaker(ctx, grantID, "nonexistent-notetaker-id-12345")
	assert.Error(t, err, "Should return error for non-existent notetaker")
	t.Logf("Expected error: %v", err)
}

func TestIntegration_CreateNotetaker(t *testing.T) {
	if os.Getenv("NYLAS_TEST_NOTETAKER") != "true" {
		t.Skip("Skipping notetaker create test - set NYLAS_TEST_NOTETAKER=true to enable")
	}

	client, grantID := getTestClient(t)
	ctx, cancel := createLongTestContext()
	defer cancel()

	meetingLink := os.Getenv("NYLAS_TEST_MEETING_LINK")
	if meetingLink == "" {
		t.Skip("NYLAS_TEST_MEETING_LINK not set")
	}

	req := &domain.CreateNotetakerRequest{
		MeetingLink: meetingLink,
		BotConfig: &domain.BotConfig{
			Name: "Integration Test Bot",
		},
	}

	notetaker, err := client.CreateNotetaker(ctx, grantID, req)
	skipIfProviderNotSupported(t, err)
	require.NoError(t, err)
	require.NotEmpty(t, notetaker.ID)

	t.Logf("Created notetaker: %s", notetaker.ID)
	t.Logf("State: %s", notetaker.State)
	t.Logf("Meeting Link: %s", notetaker.MeetingLink)

	// Clean up - delete the notetaker
	err = client.DeleteNotetaker(ctx, grantID, notetaker.ID)
	if err != nil {
		t.Logf("Warning: Could not clean up notetaker: %v", err)
	} else {
		t.Logf("Cleaned up notetaker: %s", notetaker.ID)
	}
}

func TestIntegration_NotetakerMedia_NotFound(t *testing.T) {
	client, grantID := getTestClient(t)
	ctx, cancel := createTestContext()
	defer cancel()

	_, err := client.GetNotetakerMedia(ctx, grantID, "nonexistent-notetaker-id-12345")
	assert.Error(t, err, "Should return error for non-existent notetaker media")
	t.Logf("Expected error: %v", err)
}

// =============================================================================
// Tracking Options Tests
// =============================================================================
