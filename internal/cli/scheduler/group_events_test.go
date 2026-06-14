package scheduler

import (
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupEventsCmd_Structure(t *testing.T) {
	cmd := newGroupEventsCmd()
	assert.Equal(t, "group-events", cmd.Use)

	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "create", "update", "delete", "import"} {
		assert.True(t, names[want], "missing group-events subcommand: %s", want)
	}
}

func TestSchedulerCmd_RegistersGroupEvents(t *testing.T) {
	cmd := NewSchedulerCmd()
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["group-events"], "scheduler must register group-events")
}

// The list endpoint requires calendar_id/start_time/end_time; the CLI must
// enforce --calendar locally (the API 400s without it).
func TestGroupEventList_RequiresCalendar(t *testing.T) {
	cmd := newGroupEventsListCmd()
	_, _, err := testutil.ExecuteCommand(cmd, "cfg-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "calendar")
}

// Create requires calendar/title/start/end before any API call.
func TestGroupEventCreate_RequiresFields(t *testing.T) {
	cmd := newGroupEventCreateCmd()
	_, _, err := testutil.ExecuteCommand(cmd, "cfg-1", "--title", "Only title")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// Update with no fields must fail locally rather than send an empty PUT.
func TestGroupEventUpdate_RequiresAField(t *testing.T) {
	cmd := newGroupEventUpdateCmd()
	_, _, err := testutil.ExecuteCommand(cmd, "cfg-1", "ge-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
}

// Import needs either a file or inline calendar+event.
func TestGroupEventImport_RequiresInput(t *testing.T) {
	cmd := newGroupEventsImportCmd()
	_, _, err := testutil.ExecuteCommand(cmd, "cfg-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "import")
}

func TestParseGroupParticipant(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		organizer bool
		wantName  string
		wantEmail string
		wantOK    bool
	}{
		{name: "email only", input: "nyla@example.com", wantEmail: "nyla@example.com", wantOK: true},
		{name: "name and email", input: "Nyla:nyla@example.com", wantName: "Nyla", wantEmail: "nyla@example.com", wantOK: true},
		{name: "organizer flag carried", input: "boss@example.com", organizer: true, wantEmail: "boss@example.com", wantOK: true},
		{name: "empty is dropped", input: "  ", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := parseGroupParticipant(tt.input, tt.organizer)
			assert.Equal(t, tt.wantOK, ok)
			if !ok {
				return
			}
			assert.Equal(t, tt.wantName, p.Name)
			assert.Equal(t, tt.wantEmail, p.Email)
			assert.Equal(t, tt.organizer, p.IsOrganizer)
		})
	}
}

func TestParseGroupEventTime(t *testing.T) {
	got, err := parseGroupEventTime("start", "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), got, "empty defers to API default")

	got, err = parseGroupEventTime("start", "1744286400")
	require.NoError(t, err)
	assert.Equal(t, int64(1744286400), got)

	_, err = parseGroupEventTime("start", "nonsense")
	assert.Error(t, err)
}
