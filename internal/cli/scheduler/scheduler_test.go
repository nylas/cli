package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchedulerCmd(t *testing.T) {
	cmd := NewSchedulerCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "scheduler", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "sched")
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"configurations", "sessions", "bookings"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

// Configurations Tests
func TestNewConfigurationsCmd(t *testing.T) {
	cmd := newConfigurationsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "configurations", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "config")
		assert.Contains(t, cmd.Aliases, "configs")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		expectedCmds := []string{"list", "show", "create", "update", "delete"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestConfigListCmd(t *testing.T) {
	cmd := newConfigListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list [grant-id]", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

}

func TestConfigShowCmd(t *testing.T) {
	cmd := newConfigShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <config-id> [grant-id]", cmd.Use)
	})

}

func TestConfigCreateCmd(t *testing.T) {
	cmd := newConfigCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create [grant-id]", cmd.Use)
	})

	t.Run("has_base_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("participants"))
		assert.NotNil(t, cmd.Flags().Lookup("title"))
		assert.NotNil(t, cmd.Flags().Lookup("description"))
		assert.NotNil(t, cmd.Flags().Lookup("location"))
	})

	t.Run("has_duration_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("duration")
		assert.NotNil(t, flag)
		assert.Equal(t, "30", flag.DefValue)
	})

	t.Run("has_availability_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("interval"))
		assert.NotNil(t, cmd.Flags().Lookup("round-to"))
		assert.NotNil(t, cmd.Flags().Lookup("availability-method"))
		assert.NotNil(t, cmd.Flags().Lookup("buffer-before"))
		assert.NotNil(t, cmd.Flags().Lookup("buffer-after"))
	})

	t.Run("has_event_booking_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("timezone"))
		assert.NotNil(t, cmd.Flags().Lookup("booking-type"))
		assert.NotNil(t, cmd.Flags().Lookup("conferencing-provider"))
		assert.NotNil(t, cmd.Flags().Lookup("disable-emails"))
		assert.NotNil(t, cmd.Flags().Lookup("reminder-minutes"))
	})

	t.Run("has_scheduler_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("min-booking-notice"))
		assert.NotNil(t, cmd.Flags().Lookup("min-cancellation-notice"))
		assert.NotNil(t, cmd.Flags().Lookup("confirmation-method"))
		assert.NotNil(t, cmd.Flags().Lookup("available-days-in-future"))
		assert.NotNil(t, cmd.Flags().Lookup("cancellation-policy"))
	})

	t.Run("has_file_flag", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("file"))
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
	})
}

func TestConfigUpdateCmd(t *testing.T) {
	cmd := newConfigUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <config-id> [grant-id]", cmd.Use)
	})

	t.Run("has_base_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("duration"))
		assert.NotNil(t, cmd.Flags().Lookup("title"))
		assert.NotNil(t, cmd.Flags().Lookup("description"))
	})

	t.Run("has_availability_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("interval"))
		assert.NotNil(t, cmd.Flags().Lookup("round-to"))
		assert.NotNil(t, cmd.Flags().Lookup("availability-method"))
		assert.NotNil(t, cmd.Flags().Lookup("buffer-before"))
		assert.NotNil(t, cmd.Flags().Lookup("buffer-after"))
	})

	t.Run("has_event_booking_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("timezone"))
		assert.NotNil(t, cmd.Flags().Lookup("booking-type"))
		assert.NotNil(t, cmd.Flags().Lookup("conferencing-provider"))
		assert.NotNil(t, cmd.Flags().Lookup("disable-emails"))
		assert.NotNil(t, cmd.Flags().Lookup("reminder-minutes"))
	})

	t.Run("has_scheduler_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("min-booking-notice"))
		assert.NotNil(t, cmd.Flags().Lookup("min-cancellation-notice"))
		assert.NotNil(t, cmd.Flags().Lookup("confirmation-method"))
		assert.NotNil(t, cmd.Flags().Lookup("available-days-in-future"))
		assert.NotNil(t, cmd.Flags().Lookup("cancellation-policy"))
	})

	t.Run("has_file_flag", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("file"))
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
	})
}

func TestConfigDeleteCmd(t *testing.T) {
	cmd := newConfigDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <config-id> [grant-id]", cmd.Use)
	})

	t.Run("has_yes_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_yes_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("y")
		assert.NotNil(t, flag)
	})
}

// Sessions Tests
func TestNewSessionsCmd(t *testing.T) {
	cmd := newSessionsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "sessions", cmd.Use)
	})

	t.Run("has_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "session")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		expectedCmds := []string{"create", "show"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestSessionCreateCmd(t *testing.T) {
	cmd := newSessionCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_config_id_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("config-id")
		assert.NotNil(t, flag)
	})

	t.Run("has_ttl_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("ttl")
		assert.NotNil(t, flag)
		assert.Equal(t, "30", flag.DefValue)
	})
}

func TestSessionShowCmd(t *testing.T) {
	cmd := newSessionShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <session-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

}

// Bookings Tests
func TestNewBookingsCmd(t *testing.T) {
	cmd := newBookingsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "bookings", cmd.Use)
	})

	t.Run("has_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "booking")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		expectedCmds := []string{"show", "confirm", "reschedule", "cancel"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestBookingShowCmd(t *testing.T) {
	cmd := newBookingShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <booking-id>", cmd.Use)
	})

}

// Booking endpoints authenticate with a session token minted from the
// configuration, so every booking command must require --configuration-id.
func TestBookingCommands_RequireConfigurationID(t *testing.T) {
	for _, newCmd := range []func() *cobra.Command{
		newBookingShowCmd, newBookingConfirmCmd, newBookingRescheduleCmd, newBookingCancelCmd,
	} {
		cmd := newCmd()
		flag := cmd.Flags().Lookup("configuration-id")
		require.NotNil(t, flag, "%s must expose --configuration-id", cmd.Name())
		annotations := flag.Annotations[cobra.BashCompOneRequiredFlag]
		assert.Contains(t, annotations, "true", "%s must mark --configuration-id required", cmd.Name())
	}
}

func TestBookingConfirmCmd(t *testing.T) {
	cmd := newBookingConfirmCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "confirm <booking-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	// Salt is a server-issued token that cannot be looked up from the booking
	// ID, so a missing --salt must fail locally with guidance on where to find
	// it — never reach the API.
	t.Run("missing_salt_returns_actionable_error", func(t *testing.T) {
		_, _, err := testutil.ExecuteCommand(newBookingConfirmCmd(), "booking-1", "--configuration-id", "config-1")
		require.Error(t, err)
		// The custom message (not cobra's generic "required flag" error)
		// confirms the salt guard fired locally before any API call.
		assert.Contains(t, err.Error(), "salt is required to confirm a booking")
	})

	// --reason was dropped from the v3 confirm payload. It is kept as a
	// deprecated no-op so existing `confirm <id> --reason ...` scripts degrade
	// gracefully instead of hitting cobra's "unknown flag" error on upgrade.
	t.Run("reason_flag_is_deprecated_not_removed", func(t *testing.T) {
		flag := cmd.Flags().Lookup("reason")
		require.NotNil(t, flag, "--reason must remain as a compatibility shim")
		assert.NotEmpty(t, flag.Deprecated, "--reason must be marked deprecated")
	})

	// Passing the deprecated --reason must not crash with "unknown flag"; it
	// still falls through to the salt guard (reason alone can't confirm).
	t.Run("deprecated_reason_flag_is_accepted", func(t *testing.T) {
		_, _, err := testutil.ExecuteCommand(newBookingConfirmCmd(), "booking-1", "--configuration-id", "config-1", "--reason", "obsolete")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "unknown flag")
		assert.Contains(t, err.Error(), "salt is required to confirm a booking")
	})
}

func TestBookingRescheduleCmd(t *testing.T) {
	cmd := newBookingRescheduleCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "reschedule <booking-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})

	t.Run("has_start_time_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("start-time")
		assert.NotNil(t, flag)
		assert.Equal(t, "0", flag.DefValue)
	})

	t.Run("has_end_time_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("end-time")
		assert.NotNil(t, flag)
		assert.Equal(t, "0", flag.DefValue)
	})

	t.Run("has_timezone_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("timezone")
		assert.NotNil(t, flag)
	})

	t.Run("has_reason_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("reason")
		assert.NotNil(t, flag)
	})
}

func TestBookingRescheduleCmd_Validation(t *testing.T) {
	tests := []struct {
		name          string
		startTime     string
		endTime       string
		expectError   bool
		errorContains string
	}{
		{
			name:          "missing both times",
			startTime:     "0",
			endTime:       "0",
			expectError:   true,
			errorContains: "--start-time and --end-time are required",
		},
		{
			name:          "missing start time",
			startTime:     "0",
			endTime:       "1704070800",
			expectError:   true,
			errorContains: "--start-time and --end-time are required",
		},
		{
			name:          "missing end time",
			startTime:     "1704067200",
			endTime:       "0",
			expectError:   true,
			errorContains: "--start-time and --end-time are required",
		},
		{
			name:          "end time before start time",
			startTime:     "1704070800",
			endTime:       "1704067200",
			expectError:   true,
			errorContains: "end-time must be after start-time",
		},
		{
			name:          "end time equals start time",
			startTime:     "1704067200",
			endTime:       "1704067200",
			expectError:   true,
			errorContains: "end-time must be after start-time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newBookingRescheduleCmd()
			cmd.SetArgs([]string{"test-booking-id", "--configuration-id", "config-1", "--start-time", tt.startTime, "--end-time", tt.endTime})

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			}
		})
	}
}

func TestBookingCancelCmd(t *testing.T) {
	cmd := newBookingCancelCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "cancel <booking-id>", cmd.Use)
	})

	t.Run("has_reason_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("reason")
		assert.NotNil(t, flag)
	})

	t.Run("has_yes_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
	})
}

func TestResolveRescheduleResult(t *testing.T) {
	applied := &domain.Booking{BookingID: "booking-1"}

	t.Run("verified success passes through without warning", func(t *testing.T) {
		booking, warning, err := resolveRescheduleResult(applied, nil)
		require.NoError(t, err)
		assert.Equal(t, applied, booking)
		assert.Empty(t, warning)
	})

	t.Run("read-back failure becomes success with warning", func(t *testing.T) {
		// The reschedule was applied; failing the command would make scripts
		// retry or alert on a change that took effect.
		sentinelErr := fmt.Errorf("%w: transient failure", domain.ErrBookingReadBackFailed)
		booking, warning, err := resolveRescheduleResult(applied, sentinelErr)
		require.NoError(t, err)
		assert.Equal(t, applied, booking)
		assert.Contains(t, warning, "unverified")
	})

	t.Run("other errors stay errors", func(t *testing.T) {
		booking, warning, err := resolveRescheduleResult(nil, errors.New("PATCH failed"))
		require.Error(t, err)
		assert.Nil(t, booking)
		assert.Empty(t, warning)
	})

	t.Run("sentinel without booking stays an error", func(t *testing.T) {
		// Defensive: an implementer violating the port contract must not cause
		// a nil-booking success.
		sentinelErr := fmt.Errorf("%w: transient failure", domain.ErrBookingReadBackFailed)
		booking, warning, err := resolveRescheduleResult(nil, sentinelErr)
		require.Error(t, err)
		assert.Nil(t, booking)
		assert.Empty(t, warning)
	})
}

func TestRescheduleJSONPayload(t *testing.T) {
	booking := &domain.Booking{BookingID: "booking-1"}

	t.Run("verified success omits warning key", func(t *testing.T) {
		data, err := json.Marshal(rescheduleJSONPayload(booking, ""))
		require.NoError(t, err)
		var raw map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &raw))
		_, ok := raw["warning"]
		assert.False(t, ok, "warning key must be absent on verified success")
	})

	t.Run("partial success carries warning key", func(t *testing.T) {
		// --json consumers must be able to tell an unverified reschedule from a
		// verified one without parsing stderr.
		data, err := json.Marshal(rescheduleJSONPayload(booking, "record unverified"))
		require.NoError(t, err)
		var raw map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &raw))
		assert.Contains(t, raw, "warning")
		assert.Contains(t, raw, "booking_id")
	})
}
