package timezone

import (
	"strings"
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
)

func TestDSTCmd_Help(t *testing.T) {
	cmd := newDSTCmd()

	if cmd.Use != "dst" {
		t.Errorf("Expected Use to be 'dst', got '%s'", cmd.Use)
	}

	zoneFlag := cmd.Flags().Lookup("zone")
	if zoneFlag == nil {
		t.Error("Expected --zone flag to exist")
	}

	yearFlag := cmd.Flags().Lookup("year")
	if yearFlag == nil {
		t.Error("Expected --year flag to exist")
	}
}

func TestDSTCmd_WithDSTZone(t *testing.T) {
	cmd := newDSTCmd()
	stdout, _, err := executeCommand(cmd,
		"--zone", "America/New_York",
		"--year", "2026")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "DST Transitions") {
		t.Error("Expected output to contain 'DST Transitions'")
	}

	if !strings.Contains(stdout, "America/New_York") {
		t.Error("Expected output to contain zone name")
	}
}

func TestDSTCmd_WithNonDSTZone(t *testing.T) {
	cmd := newDSTCmd()
	stdout, _, err := executeCommand(cmd,
		"--zone", "America/Phoenix",
		"--year", "2026")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "No DST transitions found") {
		t.Error("Expected output to indicate no DST transitions")
	}
}

func TestDSTCmd_JSONOutput(t *testing.T) {
	stdout, _, err := testutil.ExecuteSubCommand(newDSTCmd(),
		"--zone", "America/New_York",
		"--year", "2026",
		"--json")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, `"zone"`) {
		t.Error("Expected JSON output to contain 'zone' field")
	}

	if !strings.Contains(stdout, `"year"`) {
		t.Error("Expected JSON output to contain 'year' field")
	}
}

func TestListCmd_Help(t *testing.T) {
	cmd := newListCmd()

	if cmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got '%s'", cmd.Use)
	}

	filterFlag := cmd.Flags().Lookup("filter")
	if filterFlag == nil {
		t.Error("Expected --filter flag to exist")
	}
}

func TestListCmd_AllZones(t *testing.T) {
	cmd := newListCmd()
	stdout, _, err := executeCommand(cmd)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "IANA Time Zones") {
		t.Error("Expected output to contain 'IANA Time Zones'")
	}

	if !strings.Contains(stdout, "Total:") {
		t.Error("Expected output to contain total count")
	}
}

func TestListCmd_WithFilter(t *testing.T) {
	cmd := newListCmd()
	stdout, _, err := executeCommand(cmd, "--filter", "America")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "America") {
		t.Error("Expected output to contain 'America'")
	}

	if !strings.Contains(stdout, "filtered") {
		t.Error("Expected output to indicate filtering")
	}
}

func TestListCmd_JSONOutput(t *testing.T) {
	stdout, _, err := testutil.ExecuteSubCommand(newListCmd(), "--json")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, `"zones"`) {
		t.Error("Expected JSON output to contain 'zones' field")
	}

	if !strings.Contains(stdout, `"count"`) {
		t.Error("Expected JSON output to contain 'count' field")
	}
}

func TestInfoCmd_Help(t *testing.T) {
	cmd := newInfoCmd()

	if cmd.Use != "info [ZONE]" {
		t.Errorf("Expected Use to be 'info [ZONE]', got '%s'", cmd.Use)
	}

	zoneFlag := cmd.Flags().Lookup("zone")
	if zoneFlag == nil {
		t.Error("Expected --zone flag to exist")
	}
}

func TestInfoCmd_WithPositionalArg(t *testing.T) {
	cmd := newInfoCmd()
	stdout, _, err := executeCommand(cmd, "America/New_York")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Time Zone Information") {
		t.Error("Expected output to contain 'Time Zone Information'")
	}

	if !strings.Contains(stdout, "America/New_York") {
		t.Error("Expected output to contain zone name")
	}

	if !strings.Contains(stdout, "Abbreviation:") {
		t.Error("Expected output to contain abbreviation")
	}

	if !strings.Contains(stdout, "UTC Offset:") {
		t.Error("Expected output to contain UTC offset")
	}
}

func TestInfoCmd_WithFlag(t *testing.T) {
	cmd := newInfoCmd()
	stdout, _, err := executeCommand(cmd, "--zone", "Europe/London")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Europe/London") {
		t.Error("Expected output to contain zone name")
	}
}

func TestInfoCmd_WithAbbreviation(t *testing.T) {
	cmd := newInfoCmd()
	stdout, _, err := executeCommand(cmd, "PST")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "America/Los_Angeles") {
		t.Error("Expected PST to be expanded to America/Los_Angeles")
	}

	if !strings.Contains(stdout, "expanded from 'PST'") {
		t.Error("Expected output to show abbreviation expansion")
	}
}

func TestInfoCmd_MissingZone(t *testing.T) {
	cmd := newInfoCmd()
	_, _, err := executeCommand(cmd)

	if err == nil {
		t.Error("Expected error when zone is not provided")
	}
}

func TestInfoCmd_JSONOutput(t *testing.T) {
	stdout, _, err := testutil.ExecuteSubCommand(newInfoCmd(), "--zone", "UTC", "--json")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, `"zone"`) {
		t.Error("Expected JSON output to contain 'zone' field")
	}

	if !strings.Contains(stdout, `"abbreviation"`) {
		t.Error("Expected JSON output to contain 'abbreviation' field")
	}

	if !strings.Contains(stdout, `"is_dst"`) {
		t.Error("Expected JSON output to contain 'is_dst' field")
	}
}

func TestConvertCmd_SameTimezone(t *testing.T) {
	cmd := newConvertCmd()
	stdout, _, err := executeCommand(cmd, "--from", "UTC", "--to", "UTC")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "same offset") {
		t.Error("Expected output to indicate same offset")
	}
}

func TestFindMeetingCmd_InvalidDateFormat(t *testing.T) {
	cmd := newFindMeetingCmd()
	_, _, err := executeCommand(cmd,
		"--zones", "UTC",
		"--start-date", "invalid-date")

	if err == nil {
		t.Error("Expected error for invalid start date format")
	}
}

func TestFindMeetingCmd_InvalidEndDate(t *testing.T) {
	cmd := newFindMeetingCmd()
	_, _, err := executeCommand(cmd,
		"--zones", "UTC",
		"--end-date", "not-a-date")

	if err == nil {
		t.Error("Expected error for invalid end date format")
	}
}

func TestFindMeetingCmd_NoZones(t *testing.T) {
	cmd := newFindMeetingCmd()
	_, _, err := executeCommand(cmd,
		"--zones", "",
		"--duration", "1h")

	if err == nil {
		t.Error("Expected error when no zones provided")
	}
}

func TestFindMeetingCmd_InvalidWorkingHours(t *testing.T) {
	cmd := newFindMeetingCmd()
	_, _, err := executeCommand(cmd,
		"--zones", "UTC",
		"--start-hour", "invalid",
		"--end-hour", "17:00")

	if err == nil {
		t.Error("Expected error for invalid working hours")
	}
}

func TestDSTCmd_InvalidYear(t *testing.T) {
	cmd := newDSTCmd()
	_, _, err := executeCommand(cmd,
		"--zone", "America/New_York",
		"--year", "0")

	// Service should handle gracefully even with year 0
	// This test verifies the command doesn't panic
	_ = err
}

func TestListCmd_EmptyFilter(t *testing.T) {
	cmd := newListCmd()
	stdout, _, err := executeCommand(cmd, "--filter", "NonExistentZone12345")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "No zones found") {
		t.Error("Expected output to indicate no zones found")
	}
}

func TestInfoCmd_InvalidTimeFormat(t *testing.T) {
	cmd := newInfoCmd()
	_, _, err := executeCommand(cmd,
		"--zone", "UTC",
		"--time", "invalid-time")

	if err == nil {
		t.Error("Expected error for invalid time format")
	}
}

func TestInfoCmd_InvalidTimezone(t *testing.T) {
	cmd := newInfoCmd()
	_, _, err := executeCommand(cmd, "Invalid/Timezone")

	if err == nil {
		t.Error("Expected error for invalid timezone")
	}
}
