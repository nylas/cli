package calendar

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/utilities/scheduling"
)

func TestResolveParticipantTimezones(t *testing.T) {
	t.Parallel()

	participants := []string{"alice@example.com", "bob@example.com"}

	t.Run("uses explicit timezones in participant order", func(t *testing.T) {
		timezones, usedFallback, err := resolveParticipantTimezones(participants, []string{"America/New_York", "Europe/London"})
		if err != nil {
			t.Fatalf("resolveParticipantTimezones() error = %v", err)
		}
		if usedFallback {
			t.Fatal("expected explicit timezones without fallback")
		}
		if len(timezones) != 2 {
			t.Fatalf("timezones length = %d, want 2", len(timezones))
		}
	})

	t.Run("fills all participants with local timezone when omitted", func(t *testing.T) {
		timezones, usedFallback, err := resolveParticipantTimezones(participants, nil)
		if err != nil {
			t.Fatalf("resolveParticipantTimezones() error = %v", err)
		}
		if !usedFallback {
			t.Fatal("expected fallback timezones when none are provided")
		}
		if len(timezones) != len(participants) {
			t.Fatalf("timezones length = %d, want %d", len(timezones), len(participants))
		}
		if timezones[0] == "" || timezones[1] == "" {
			t.Fatal("expected fallback timezones to be populated")
		}
	})

	t.Run("rejects mismatched timezone counts", func(t *testing.T) {
		if _, _, err := resolveParticipantTimezones(participants, []string{"America/New_York"}); err == nil {
			t.Fatal("expected error for mismatched participant/timezone counts")
		}
	})
}

func TestParseWorkingTime(t *testing.T) {
	t.Parallel()

	t.Run("keeps minute precision", func(t *testing.T) {
		got, err := parseWorkingTime("09:30")
		if err != nil {
			t.Fatalf("parseWorkingTime() error = %v", err)
		}
		if got != 9*60+30 {
			t.Fatalf("parseWorkingTime() = %d, want %d", got, 9*60+30)
		}
	})

	t.Run("rejects invalid minutes", func(t *testing.T) {
		if _, err := parseWorkingTime("09:99"); err == nil {
			t.Fatal("expected error for invalid minutes")
		}
	})
}

func TestFindMeetingSlots_RespectsWorkingHourMinutes(t *testing.T) {
	t.Parallel()

	slots, err := findMeetingSlots(
		t.Context(),
		[]string{"UTC", "UTC"},
		30*time.Minute,
		"09:30",
		"17:00",
		1,
		false,
	)
	if err != nil {
		t.Fatalf("findMeetingSlots() error = %v", err)
	}
	if len(slots) == 0 {
		t.Fatal("expected at least one slot")
	}
	if slots[0].Breakdown.WorkingHours != scheduling.ScoreWorkingHoursMax {
		t.Fatalf("first slot working hours = %.0f, want %.0f", slots[0].Breakdown.WorkingHours, scheduling.ScoreWorkingHoursMax)
	}
}
