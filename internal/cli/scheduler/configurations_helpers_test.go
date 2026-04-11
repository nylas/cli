package scheduler

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(v string) *string {
	return &v
}

// newTestCmd creates a command with all config flags registered for testing.
func newTestCmd() (*cobra.Command, *configFlags) {
	cmd := &cobra.Command{Use: "test"}
	f := &configFlags{}
	registerConfigFlags(cmd, f)
	return cmd, f
}

func TestRegisterConfigFlags_Create(t *testing.T) {
	cmd, _ := newTestCmd()

	expectedFlags := []string{
		// Availability
		"interval",
		"round-to",
		"availability-method",
		"buffer-before",
		"buffer-after",
		// Event booking
		"timezone",
		"booking-type",
		"conferencing-provider",
		"disable-emails",
		"reminder-minutes",
		// Scheduler settings
		"min-booking-notice",
		"min-cancellation-notice",
		"confirmation-method",
		"available-days-in-future",
		"cancellation-policy",
		// File input
		"file",
	}

	for _, flagName := range expectedFlags {
		t.Run(flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "expected flag %q to be registered", flagName)
		})
	}
}

func TestValidateConfigFlags(t *testing.T) {
	tests := []struct {
		name        string
		flags       configFlags
		expectError bool
		errContains string
	}{
		{
			name:        "empty flags are valid",
			flags:       configFlags{},
			expectError: false,
		},
		{
			name:        "valid availability method max-fairness",
			flags:       configFlags{availabilityMethod: "max-fairness"},
			expectError: false,
		},
		{
			name:        "valid availability method max-availability",
			flags:       configFlags{availabilityMethod: "max-availability"},
			expectError: false,
		},
		{
			name:        "invalid availability method",
			flags:       configFlags{availabilityMethod: "random"},
			expectError: true,
			errContains: "availability-method",
		},
		{
			name:        "valid booking type booking",
			flags:       configFlags{bookingType: "booking"},
			expectError: false,
		},
		{
			name:        "valid booking type organizer-confirmation",
			flags:       configFlags{bookingType: "organizer-confirmation"},
			expectError: false,
		},
		{
			name:        "invalid booking type",
			flags:       configFlags{bookingType: "instant"},
			expectError: true,
			errContains: "booking-type",
		},
		{
			name:        "valid confirmation method automatic",
			flags:       configFlags{confirmationMethod: "automatic"},
			expectError: false,
		},
		{
			name:        "valid confirmation method manual",
			flags:       configFlags{confirmationMethod: "manual"},
			expectError: false,
		},
		{
			name:        "invalid confirmation method",
			flags:       configFlags{confirmationMethod: "none"},
			expectError: true,
			errContains: "confirmation-method",
		},
		{
			name:        "valid conferencing provider Google Meet",
			flags:       configFlags{conferencingProvider: "Google Meet"},
			expectError: false,
		},
		{
			name:        "valid conferencing provider Zoom",
			flags:       configFlags{conferencingProvider: "Zoom"},
			expectError: false,
		},
		{
			name:        "valid conferencing provider Microsoft Teams",
			flags:       configFlags{conferencingProvider: "Microsoft Teams"},
			expectError: false,
		},
		{
			name:        "invalid conferencing provider",
			flags:       configFlags{conferencingProvider: "Webex"},
			expectError: true,
			errContains: "conferencing-provider",
		},
		{
			name: "all valid enum values together",
			flags: configFlags{
				availabilityMethod:   "max-fairness",
				bookingType:          "booking",
				confirmationMethod:   "automatic",
				conferencingProvider: "Zoom",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.flags
			err := validateConfigFlags(&f)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildCreateRequest_FromFileOnly(t *testing.T) {
	fileData := domain.CreateSchedulerConfigurationRequest{
		Name: "File Config",
		Participants: []domain.ConfigurationParticipant{
			{Email: "organizer@example.com", IsOrganizer: true},
		},
		Availability: domain.AvailabilityRules{DurationMinutes: 45},
		EventBooking: domain.EventBooking{
			Title:    "File Meeting",
			Timezone: "America/Chicago",
		},
	}

	data, err := json.Marshal(fileData)
	require.NoError(t, err)

	dir := t.TempDir()
	filePath := dir + "/config.json"
	require.NoError(t, os.WriteFile(filePath, data, 0600))

	cmd, f := newTestCmd()
	f.file = filePath

	req, err := buildCreateRequest(cmd, f, "", nil, 0, "", "", "")
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "File Config", req.Name)
	assert.Equal(t, 45, req.Availability.DurationMinutes)
	assert.Equal(t, "File Meeting", req.EventBooking.Title)
	assert.Equal(t, "America/Chicago", req.EventBooking.Timezone)
	require.Len(t, req.Participants, 1)
	assert.Equal(t, "organizer@example.com", req.Participants[0].Email)
}

func TestBuildCreateRequest_FlagsOverrideFile(t *testing.T) {
	fileData := domain.CreateSchedulerConfigurationRequest{
		Name:         "File Config",
		Availability: domain.AvailabilityRules{DurationMinutes: 45},
		EventBooking: domain.EventBooking{Title: "File Meeting"},
	}

	data, err := json.Marshal(fileData)
	require.NoError(t, err)

	dir := t.TempDir()
	filePath := dir + "/config.json"
	require.NoError(t, os.WriteFile(filePath, data, 0600))

	cmd, f := newTestCmd()
	f.file = filePath

	// Simulate flag being explicitly set
	require.NoError(t, cmd.Flags().Set("interval", "15"))

	req, err := buildCreateRequest(cmd, f, "Flag Config", nil, 0, "Flag Meeting", "", "")
	require.NoError(t, err)
	require.NotNil(t, req)

	// Flag-provided name and title override file values
	assert.Equal(t, "Flag Config", req.Name)
	assert.Equal(t, "Flag Meeting", req.EventBooking.Title)
	// File value preserved when not overridden by flag
	assert.Equal(t, 45, req.Availability.DurationMinutes)
	// Interval set via flag
	assert.Equal(t, 15, req.Availability.IntervalMinutes)
}

func TestBuildCreateRequest_FlagsOnly(t *testing.T) {
	cmd, f := newTestCmd()

	require.NoError(t, cmd.Flags().Set("timezone", "America/New_York"))
	require.NoError(t, cmd.Flags().Set("buffer-before", "5"))
	require.NoError(t, cmd.Flags().Set("buffer-after", "10"))
	require.NoError(t, cmd.Flags().Set("availability-method", "max-availability"))

	req, err := buildCreateRequest(
		cmd, f,
		"My Config",
		[]string{"alice@example.com", "bob@example.com"},
		60,
		"Team Meeting", "Monthly sync", "Conference Room A",
	)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "My Config", req.Name)
	assert.Equal(t, 60, req.Availability.DurationMinutes)
	assert.Equal(t, "Team Meeting", req.EventBooking.Title)
	assert.Equal(t, "Monthly sync", req.EventBooking.Description)
	assert.Equal(t, "Conference Room A", req.EventBooking.Location)
	assert.Equal(t, "America/New_York", req.EventBooking.Timezone)
	assert.Equal(t, "max-availability", req.Availability.AvailabilityMethod)

	require.NotNil(t, req.Availability.Buffer)
	assert.Equal(t, 5, req.Availability.Buffer.Before)
	assert.Equal(t, 10, req.Availability.Buffer.After)

	require.Len(t, req.Participants, 2)
	assert.Equal(t, "alice@example.com", req.Participants[0].Email)
	assert.True(t, req.Participants[0].IsOrganizer)
	assert.Equal(t, "bob@example.com", req.Participants[1].Email)
	assert.False(t, req.Participants[1].IsOrganizer)
}

func TestBuildUpdateRequest_FromFileOnly(t *testing.T) {
	name := "File Update Config"
	fileData := domain.UpdateSchedulerConfigurationRequest{
		Name: &name,
		Availability: &domain.AvailabilityRules{
			DurationMinutes: 30,
			IntervalMinutes: 15,
		},
		EventBooking: &domain.EventBooking{
			Title:    "Updated Meeting",
			Timezone: "Europe/London",
		},
	}

	data, err := json.Marshal(fileData)
	require.NoError(t, err)

	dir := t.TempDir()
	filePath := dir + "/update.json"
	require.NoError(t, os.WriteFile(filePath, data, 0600))

	cmd, f := newTestCmd()
	f.file = filePath

	req, err := buildUpdateRequest(cmd, f, "", 0, "", "")
	require.NoError(t, err)
	require.NotNil(t, req)

	require.NotNil(t, req.Name)
	assert.Equal(t, "File Update Config", *req.Name)
	require.NotNil(t, req.Availability)
	assert.Equal(t, 30, req.Availability.DurationMinutes)
	assert.Equal(t, 15, req.Availability.IntervalMinutes)
	require.NotNil(t, req.EventBooking)
	assert.Equal(t, "Updated Meeting", req.EventBooking.Title)
	assert.Equal(t, "Europe/London", req.EventBooking.Timezone)
}

func TestBuildUpdateRequest_FlagsOnly(t *testing.T) {
	cmd, f := newTestCmd()

	require.NoError(t, cmd.Flags().Set("interval", "20"))
	require.NoError(t, cmd.Flags().Set("min-booking-notice", "60"))

	req, err := buildUpdateRequest(cmd, f, "Updated Name", 0, "", "")
	require.NoError(t, err)
	require.NotNil(t, req)

	require.NotNil(t, req.Name)
	assert.Equal(t, "Updated Name", *req.Name)

	require.NotNil(t, req.Availability)
	assert.Equal(t, 20, req.Availability.IntervalMinutes)

	require.NotNil(t, req.Scheduler)
	assert.Equal(t, 60, req.Scheduler.MinBookingNotice)
}

func TestBuildUpdateRequest_FlagsOverrideFile(t *testing.T) {
	origName := "File Name"
	fileData := domain.UpdateSchedulerConfigurationRequest{
		Name: &origName,
		Availability: &domain.AvailabilityRules{
			DurationMinutes: 45,
		},
	}

	data, err := json.Marshal(fileData)
	require.NoError(t, err)

	dir := t.TempDir()
	filePath := dir + "/update.json"
	require.NoError(t, os.WriteFile(filePath, data, 0600))

	cmd, f := newTestCmd()
	// Register a duration flag as the real update command would have
	var dur int
	cmd.Flags().IntVar(&dur, "duration", 0, "Meeting duration in minutes")
	f.file = filePath

	// Duration flag overrides file value
	require.NoError(t, cmd.Flags().Set("duration", "90"))

	req, err := buildUpdateRequest(cmd, f, "Flag Name", 90, "", "")
	require.NoError(t, err)
	require.NotNil(t, req)

	// Flag-provided name overrides file name
	require.NotNil(t, req.Name)
	assert.Equal(t, "Flag Name", *req.Name)

	// Duration flag overrides file value
	require.NotNil(t, req.Availability)
	assert.Equal(t, 90, req.Availability.DurationMinutes)
}

func TestValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *domain.CreateSchedulerConfigurationRequest
		errContains string
	}{
		{
			name: "missing name",
			req: &domain.CreateSchedulerConfigurationRequest{
				Participants: []domain.ConfigurationParticipant{{Email: "alice@example.com"}},
				EventBooking: domain.EventBooking{Title: "Team Sync"},
			},
			errContains: "--name flag is required",
		},
		{
			name: "missing participants",
			req: &domain.CreateSchedulerConfigurationRequest{
				Name:         "Config",
				EventBooking: domain.EventBooking{Title: "Team Sync"},
			},
			errContains: "at least one participant is required",
		},
		{
			name: "missing title",
			req: &domain.CreateSchedulerConfigurationRequest{
				Name:         "Config",
				Participants: []domain.ConfigurationParticipant{{Email: "alice@example.com"}},
			},
			errContains: "--title flag is required",
		},
		{
			name: "valid request",
			req: &domain.CreateSchedulerConfigurationRequest{
				Name:         "Config",
				Participants: []domain.ConfigurationParticipant{{Email: "alice@example.com"}},
				EventBooking: domain.EventBooking{Title: "Team Sync"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateRequest(tt.req)
			if tt.errContains == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *domain.UpdateSchedulerConfigurationRequest
		errContains string
	}{
		{
			name:        "empty request",
			req:         &domain.UpdateSchedulerConfigurationRequest{},
			errContains: "No update fields provided",
		},
		{
			name: "name update",
			req: &domain.UpdateSchedulerConfigurationRequest{
				Name: strPtr("Updated Name"),
			},
		},
		{
			name: "explicit participants update counts as change",
			req: &domain.UpdateSchedulerConfigurationRequest{
				Participants: []domain.ConfigurationParticipant{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateRequest(tt.req)
			if tt.errContains == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestFormatConfigDetails(t *testing.T) {
	config := &domain.SchedulerConfiguration{
		ID:   "cfg-123",
		Name: "Team Sync",
		Slug: "team-sync",
		Availability: domain.AvailabilityRules{
			DurationMinutes:    30,
			IntervalMinutes:    15,
			RoundTo:            10,
			AvailabilityMethod: "max-fairness",
			Buffer: &domain.AvailabilityBuffer{
				Before: 5,
				After:  10,
			},
		},
		Participants: []domain.ConfigurationParticipant{
			{Name: "Alice", Email: "alice@example.com", IsOrganizer: true},
			{Name: "Bob", Email: "bob@example.com"},
		},
		EventBooking: domain.EventBooking{
			Title:       "Team Meeting",
			Description: "Weekly sync",
			Location:    "Conference Room",
			Timezone:    "America/New_York",
			BookingType: "booking",
			Conferencing: &domain.ConferencingSettings{
				Provider:   "Google Meet",
				Autocreate: true,
			},
			DisableEmails:   false,
			ReminderMinutes: []int{10, 60},
		},
		Scheduler: domain.SchedulerSettings{
			AvailableDaysInFuture: 30,
			MinBookingNotice:      60,
			MinCancellationNotice: 120,
			ConfirmationMethod:    "automatic",
			CancellationPolicy:    "No refunds",
		},
		AppearanceSettings: &domain.AppearanceSettings{
			CompanyName:     "Acme Corp",
			Color:           "#ff5733",
			SubmitText:      "Book Now",
			ThankYouMessage: "Thanks!",
		},
	}

	var buf bytes.Buffer
	formatConfigDetails(&buf, config)
	output := buf.String()

	assert.Contains(t, output, "Team Sync")
	assert.Contains(t, output, "cfg-123")
	assert.Contains(t, output, "team-sync")
	assert.Contains(t, output, "30 minutes")
	assert.Contains(t, output, "Interval: 15 minutes")
	assert.Contains(t, output, "Round To: 10 minutes")
	assert.Contains(t, output, "max-fairness")
	assert.Contains(t, output, "5 min before")
	assert.Contains(t, output, "10 min after")
	assert.Contains(t, output, "Participants (2)")
	assert.Contains(t, output, "alice@example.com")
	assert.Contains(t, output, "bob@example.com")
	assert.Contains(t, output, "(Organizer)")
	assert.Contains(t, output, "Team Meeting")
	assert.Contains(t, output, "Weekly sync")
	assert.Contains(t, output, "Conference Room")
	assert.Contains(t, output, "America/New_York")
	assert.Contains(t, output, "booking")
	assert.Contains(t, output, "Google Meet (autocreate)")
	assert.Contains(t, output, "10, 60")
	assert.Contains(t, output, "Available Days: 30")
	assert.Contains(t, output, "Min Booking Notice: 60 minutes")
	assert.Contains(t, output, "Min Cancellation Notice: 120 minutes")
	assert.Contains(t, output, "automatic")
	assert.Contains(t, output, "No refunds")
	assert.Contains(t, output, "Acme Corp")
	assert.Contains(t, output, "#ff5733")
	assert.Contains(t, output, "Book Now")
	assert.Contains(t, output, "Thanks!")
}

func TestFormatConfigDetails_MinimalConfig(t *testing.T) {
	config := &domain.SchedulerConfiguration{
		ID:   "cfg-456",
		Name: "Minimal Config",
		Availability: domain.AvailabilityRules{
			DurationMinutes: 30,
		},
		EventBooking: domain.EventBooking{
			Title: "Quick Call",
		},
	}

	var buf bytes.Buffer
	formatConfigDetails(&buf, config)
	output := buf.String()

	// Required fields present
	assert.Contains(t, output, "Minimal Config")
	assert.Contains(t, output, "cfg-456")
	assert.Contains(t, output, "Quick Call")

	// Optional sections NOT present
	assert.NotContains(t, output, "Slug:")
	assert.NotContains(t, output, "Interval:")
	assert.NotContains(t, output, "Round To:")
	assert.NotContains(t, output, "Availability Method:")
	assert.NotContains(t, output, "Buffer:")
	assert.NotContains(t, output, "Participants (")
	assert.NotContains(t, output, "Timezone:")
	assert.NotContains(t, output, "Booking Type:")
	assert.NotContains(t, output, "Conferencing:")
	assert.NotContains(t, output, "Emails: disabled")
	assert.NotContains(t, output, "Reminders:")
	assert.NotContains(t, output, "Scheduler Settings:")
	assert.NotContains(t, output, "Appearance:")
}

func TestBuildParticipants(t *testing.T) {
	t.Run("first is organizer", func(t *testing.T) {
		participants := buildParticipants([]string{"a@example.com", "b@example.com", "c@example.com"})
		require.Len(t, participants, 3)
		assert.True(t, participants[0].IsOrganizer)
		assert.False(t, participants[1].IsOrganizer)
		assert.False(t, participants[2].IsOrganizer)
	})

	t.Run("single participant is organizer", func(t *testing.T) {
		participants := buildParticipants([]string{"only@example.com"})
		require.Len(t, participants, 1)
		assert.True(t, participants[0].IsOrganizer)
		assert.Equal(t, "only@example.com", participants[0].Email)
	})

	t.Run("empty returns nil", func(t *testing.T) {
		participants := buildParticipants(nil)
		assert.Nil(t, participants)
	})
}

func TestHasAvailabilityFlags(t *testing.T) {
	t.Run("no flags changed returns false", func(t *testing.T) {
		cmd, _ := newTestCmd()
		assert.False(t, hasAvailabilityFlags(cmd))
	})

	t.Run("interval changed returns true", func(t *testing.T) {
		cmd, _ := newTestCmd()
		require.NoError(t, cmd.Flags().Set("interval", "10"))
		assert.True(t, hasAvailabilityFlags(cmd))
	})

	t.Run("buffer-before changed returns true", func(t *testing.T) {
		cmd, _ := newTestCmd()
		require.NoError(t, cmd.Flags().Set("buffer-before", "5"))
		assert.True(t, hasAvailabilityFlags(cmd))
	})
}

func TestHasEventBookingFlags(t *testing.T) {
	t.Run("no flags changed returns false", func(t *testing.T) {
		cmd, _ := newTestCmd()
		assert.False(t, hasEventBookingFlags(cmd))
	})

	t.Run("timezone changed returns true", func(t *testing.T) {
		cmd, _ := newTestCmd()
		require.NoError(t, cmd.Flags().Set("timezone", "UTC"))
		assert.True(t, hasEventBookingFlags(cmd))
	})
}

func TestHasSchedulerFlags(t *testing.T) {
	t.Run("no flags changed returns false", func(t *testing.T) {
		cmd, _ := newTestCmd()
		assert.False(t, hasSchedulerFlags(cmd))
	})

	t.Run("confirmation-method changed returns true", func(t *testing.T) {
		cmd, _ := newTestCmd()
		require.NoError(t, cmd.Flags().Set("confirmation-method", "manual"))
		assert.True(t, hasSchedulerFlags(cmd))
	})
}

func TestFormatConfigDetails_DisableEmails(t *testing.T) {
	config := &domain.SchedulerConfiguration{
		ID:   "cfg-789",
		Name: "No Email Config",
		Availability: domain.AvailabilityRules{
			DurationMinutes: 30,
		},
		EventBooking: domain.EventBooking{
			Title:         "Silent Meeting",
			DisableEmails: true,
		},
	}

	var buf bytes.Buffer
	formatConfigDetails(&buf, config)
	output := buf.String()

	assert.True(t, strings.Contains(output, "Emails: disabled"))
}
