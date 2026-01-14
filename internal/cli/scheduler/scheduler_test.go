package scheduler

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		expectedCmds := []string{"configurations", "sessions", "bookings", "pages"}

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
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestConfigShowCmd(t *testing.T) {
	cmd := newConfigShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <config-id>", cmd.Use)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestConfigCreateCmd(t *testing.T) {
	cmd := newConfigCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_required_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("participants"))
		assert.NotNil(t, cmd.Flags().Lookup("title"))
	})

	t.Run("has_event_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("title"))
		assert.NotNil(t, cmd.Flags().Lookup("description"))
		assert.NotNil(t, cmd.Flags().Lookup("location"))
	})

	t.Run("has_duration_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("duration")
		assert.NotNil(t, flag)
		assert.Equal(t, "30", flag.DefValue)
	})
}

func TestConfigUpdateCmd(t *testing.T) {
	cmd := newConfigUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <config-id>", cmd.Use)
	})

	t.Run("has_update_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("duration"))
		assert.NotNil(t, cmd.Flags().Lookup("title"))
		assert.NotNil(t, cmd.Flags().Lookup("description"))
	})
}

func TestConfigDeleteCmd(t *testing.T) {
	cmd := newConfigDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <config-id>", cmd.Use)
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

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
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
		expectedCmds := []string{"list", "show", "confirm", "reschedule", "cancel"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestBookingListCmd(t *testing.T) {
	cmd := newBookingListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_filter_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("config-id"))
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestBookingShowCmd(t *testing.T) {
	cmd := newBookingShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <booking-id>", cmd.Use)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestBookingConfirmCmd(t *testing.T) {
	cmd := newBookingConfirmCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "confirm <booking-id>", cmd.Use)
	})

	t.Run("requires_exactly_one_arg", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
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
			cmd.SetArgs([]string{"test-booking-id", "--start-time", tt.startTime, "--end-time", tt.endTime})

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

// Pages Tests
func TestNewPagesCmd(t *testing.T) {
	cmd := newPagesCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "pages", cmd.Use)
	})

	t.Run("has_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "page")
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

func TestPageListCmd(t *testing.T) {
	cmd := newPageListCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
	})

	t.Run("has_ls_alias", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "ls")
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestPageShowCmd(t *testing.T) {
	cmd := newPageShowCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "show <page-id>", cmd.Use)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestPageCreateCmd(t *testing.T) {
	cmd := newPageCreateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "create", cmd.Use)
	})

	t.Run("has_required_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("config-id"))
	})

	t.Run("has_slug_flag", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("slug"))
	})
}

func TestPageUpdateCmd(t *testing.T) {
	cmd := newPageUpdateCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "update <page-id>", cmd.Use)
	})

	t.Run("has_update_flags", func(t *testing.T) {
		assert.NotNil(t, cmd.Flags().Lookup("name"))
		assert.NotNil(t, cmd.Flags().Lookup("slug"))
	})
}

func TestPageDeleteCmd(t *testing.T) {
	cmd := newPageDeleteCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "delete <page-id>", cmd.Use)
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
