package timezone

import (
	"strings"
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/spf13/cobra"
)

// executeCommand executes a command and captures output
func executeCommand(cmd *cobra.Command, args ...string) (stdout string, stderr string, err error) {
	return testutil.ExecuteCommand(cmd, args...)
}

func TestNewTimezoneCmd(t *testing.T) {
	cmd := NewTimezoneCmd()

	if cmd.Use != "timezone" {
		t.Errorf("Expected Use to be 'timezone', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	// Check that all subcommands are registered
	expectedCommands := []string{"convert", "find-meeting", "dst", "list", "info"}
	commands := cmd.Commands()

	if len(commands) != len(expectedCommands) {
		t.Errorf("Expected %d subcommands, got %d", len(expectedCommands), len(commands))
	}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range commands {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found", expected)
		}
	}
}

func TestConvertCmd_Help(t *testing.T) {
	cmd := newConvertCmd()

	if cmd.Use != "convert" {
		t.Errorf("Expected Use to be 'convert', got '%s'", cmd.Use)
	}

	// Check required flags
	fromFlag := cmd.Flags().Lookup("from")
	if fromFlag == nil {
		t.Error("Expected --from flag to exist")
	}

	toFlag := cmd.Flags().Lookup("to")
	if toFlag == nil {
		t.Error("Expected --to flag to exist")
	}

}

func TestConvertCmd_MissingRequiredFlags(t *testing.T) {
	cmd := newConvertCmd()
	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error when required flags are missing")
	}
}

func TestConvertCmd_ValidConversion(t *testing.T) {
	cmd := newConvertCmd()
	stdout, _, err := executeCommand(cmd, "--from", "UTC", "--to", "America/New_York")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Time Zone Conversion") {
		t.Error("Expected output to contain 'Time Zone Conversion'")
	}

	if !strings.Contains(stdout, "UTC") {
		t.Error("Expected output to contain 'UTC'")
	}

	if !strings.Contains(stdout, "America/New_York") {
		t.Error("Expected output to contain 'America/New_York'")
	}
}

func TestConvertCmd_WithAbbreviations(t *testing.T) {
	cmd := newConvertCmd()
	stdout, _, err := executeCommand(cmd, "--from", "PST", "--to", "EST")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that abbreviations were expanded
	if !strings.Contains(stdout, "America/Los_Angeles") {
		t.Error("Expected PST to be expanded to America/Los_Angeles")
	}

	if !strings.Contains(stdout, "America/New_York") {
		t.Error("Expected EST to be expanded to America/New_York")
	}
}

func TestConvertCmd_WithSpecificTime(t *testing.T) {
	cmd := newConvertCmd()
	testTime := "2025-01-01T12:00:00Z"

	stdout, _, err := executeCommand(cmd,
		"--from", "UTC",
		"--to", "America/Los_Angeles",
		"--time", testTime)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "2025-01-01") {
		t.Error("Expected output to contain the specified date")
	}
}

func TestConvertCmd_JSONOutput(t *testing.T) {
	stdout, _, err := testutil.ExecuteSubCommand(newConvertCmd(),
		"--from", "UTC",
		"--to", "America/New_York",
		"--json")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, `"from"`) {
		t.Error("Expected JSON output to contain 'from' field")
	}

	if !strings.Contains(stdout, `"to"`) {
		t.Error("Expected JSON output to contain 'to' field")
	}

	if !strings.Contains(stdout, `"zone"`) {
		t.Error("Expected JSON output to contain 'zone' field")
	}
}

func TestConvertCmd_InvalidTimeFormat(t *testing.T) {
	cmd := newConvertCmd()
	_, _, err := executeCommand(cmd,
		"--from", "UTC",
		"--to", "America/New_York",
		"--time", "invalid-time")

	if err == nil {
		t.Error("Expected error for invalid time format")
	}
}

func TestConvertCmd_InvalidTimeZone(t *testing.T) {
	cmd := newConvertCmd()
	_, _, err := executeCommand(cmd,
		"--from", "Invalid/Zone",
		"--to", "America/New_York")

	if err == nil {
		t.Error("Expected error for invalid time zone")
	}
}

func TestFindMeetingCmd_Help(t *testing.T) {
	cmd := newFindMeetingCmd()

	if cmd.Use != "find-meeting" {
		t.Errorf("Expected Use to be 'find-meeting', got '%s'", cmd.Use)
	}

	// Check required flags
	zonesFlag := cmd.Flags().Lookup("zones")
	if zonesFlag == nil {
		t.Error("Expected --zones flag to exist")
	}

	durationFlag := cmd.Flags().Lookup("duration")
	if durationFlag == nil {
		t.Error("Expected --duration flag to exist")
	}
}

func TestFindMeetingCmd_MissingZones(t *testing.T) {
	cmd := newFindMeetingCmd()
	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error when --zones flag is missing")
	}
}

func TestFindMeetingCmd_ValidRequest(t *testing.T) {
	cmd := newFindMeetingCmd()
	stdout, _, err := executeCommand(cmd,
		"--zones", "America/New_York,Europe/London")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Meeting Time Finder") {
		t.Error("Expected output to contain 'Meeting Time Finder'")
	}

	if !strings.Contains(stdout, "Time Zones:") {
		t.Error("Expected output to contain 'Time Zones:'")
	}
}

func TestFindMeetingCmd_WithAllOptions(t *testing.T) {
	cmd := newFindMeetingCmd()
	stdout, _, err := executeCommand(cmd,
		"--zones", "PST,EST,IST",
		"--duration", "30m",
		"--start-hour", "10:00",
		"--end-hour", "16:00",
		"--exclude-weekends")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "30m") {
		t.Error("Expected output to contain duration")
	}

	if !strings.Contains(stdout, "10:00 - 16:00") {
		t.Error("Expected output to contain working hours")
	}

	if !strings.Contains(stdout, "Weekends") {
		t.Error("Expected output to mention weekend exclusion")
	}
}

func TestFindMeetingCmd_InvalidDuration(t *testing.T) {
	cmd := newFindMeetingCmd()
	_, _, err := executeCommand(cmd,
		"--zones", "UTC",
		"--duration", "invalid")

	if err == nil {
		t.Error("Expected error for invalid duration format")
	}
}
