package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAvailabilityCmd(t *testing.T) {
	cmd := newAvailabilityCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "availability", cmd.Use)
	})

	t.Run("has_aliases", func(t *testing.T) {
		assert.Contains(t, cmd.Aliases, "avail")
		assert.Contains(t, cmd.Aliases, "freebusy")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"check", "find"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestFreeBusyCmd(t *testing.T) {
	cmd := newFreeBusyCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "check [grant-id]", cmd.Use)
	})

	t.Run("has_emails_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("emails")
		assert.NotNil(t, flag)
	})

	t.Run("has_start_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("start")
		assert.NotNil(t, flag)
	})

	t.Run("has_end_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("end")
		assert.NotNil(t, flag)
	})

	t.Run("has_duration_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("duration")
		assert.NotNil(t, flag)
	})

	t.Run("has_format_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "text", flag.DefValue)
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "availability check")
	})
}

func TestFindSlotsCmd(t *testing.T) {
	cmd := newFindSlotsCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "find", cmd.Use)
	})

	t.Run("has_participants_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("participants")
		assert.NotNil(t, flag)
	})

	t.Run("has_start_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("start")
		assert.NotNil(t, flag)
	})

	t.Run("has_end_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("end")
		assert.NotNil(t, flag)
	})

	t.Run("has_duration_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("duration")
		assert.NotNil(t, flag)
		assert.Equal(t, "30", flag.DefValue)
	})

	t.Run("has_interval_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("interval")
		assert.NotNil(t, flag)
		assert.Equal(t, "15", flag.DefValue)
	})

	t.Run("has_format_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("format")
		assert.NotNil(t, flag)
		assert.Equal(t, "text", flag.DefValue)
	})

	t.Run("has_examples", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Example)
		assert.Contains(t, cmd.Example, "availability find")
	})
}
