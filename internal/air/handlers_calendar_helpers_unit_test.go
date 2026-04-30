package air

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// TestEventToResponse_AllDayValidDate verifies that a well-formed all-day
// date is converted to the correct Unix-second window.
func TestEventToResponse_AllDayValidDate(t *testing.T) {
	e := domain.Event{
		ID:    "evt1",
		Title: "Holiday",
		When: domain.EventWhen{
			Object: "date",
			Date:   "2026-04-29",
		},
	}

	resp := eventToResponse(e)

	if !resp.IsAllDay {
		t.Fatalf("expected IsAllDay=true")
	}
	// 2026-04-29T00:00:00Z = 1777420800
	if resp.StartTime != 1777420800 {
		t.Errorf("StartTime: want 1777420800, got %d", resp.StartTime)
	}
	if resp.EndTime != 1777420800+24*60*60 {
		t.Errorf("EndTime: want %d, got %d", 1777420800+24*60*60, resp.EndTime)
	}
}

// TestEventToResponse_AllDayMalformedDate verifies that a malformed date
// does NOT clobber StartTime/EndTime with a year-1 Unix timestamp.
//
// Before the fix, time.Parse("2006-01-02", "not-a-date") returned the zero
// time, and resp.StartTime got -62135596800 (year 1). After the fix, the
// upstream-provided StartTime is preserved when the date string is bad.
func TestEventToResponse_AllDayMalformedDate(t *testing.T) {
	e := domain.Event{
		ID:    "evt-bad",
		Title: "Bad date",
		When: domain.EventWhen{
			Object:    "date",
			Date:      "not-a-date",
			StartTime: 1700000000, // upstream fallback
			EndTime:   1700086400,
		},
	}

	resp := eventToResponse(e)

	if resp.StartTime < 0 || resp.StartTime > 4102444800 {
		t.Errorf("StartTime should be a sane Unix second, got %d", resp.StartTime)
	}
	if resp.EndTime < 0 || resp.EndTime > 4102444800 {
		t.Errorf("EndTime should be a sane Unix second, got %d", resp.EndTime)
	}
	// We expect the upstream fallback to win, not year-1.
	if resp.StartTime != 1700000000 {
		t.Errorf("StartTime should preserve upstream fallback 1700000000, got %d", resp.StartTime)
	}
}

// TestEventToResponse_AllDayDatespanValidStart confirms that datespan events
// override start (and end if EndDate parses), while malformed end is ignored.
func TestEventToResponse_AllDayDatespanMixedValidity(t *testing.T) {
	e := domain.Event{
		ID:    "evt-mixed",
		Title: "Multi-day",
		When: domain.EventWhen{
			Object:    "datespan",
			StartDate: "2026-04-29",
			EndDate:   "totally-broken",
			StartTime: 1700000000, // would be overridden by valid StartDate
			EndTime:   9999999999, // should be preserved (EndDate fails to parse)
		},
	}

	resp := eventToResponse(e)
	if resp.StartTime != 1777420800 {
		t.Errorf("StartTime should reflect parsed StartDate, got %d", resp.StartTime)
	}
	if resp.EndTime != 9999999999 {
		t.Errorf("EndTime should preserve fallback when EndDate is malformed, got %d", resp.EndTime)
	}
}

// TestAllDayBounds_NotAllDay returns the supplied fallback unchanged.
func TestAllDayBounds_NotAllDay(t *testing.T) {
	w := domain.EventWhen{Object: "timespan", StartTime: 100, EndTime: 200}
	gotStart, gotEnd := allDayBounds(w, 100, 200)
	if gotStart != 100 || gotEnd != 200 {
		t.Errorf("non-all-day should return fallback (100, 200), got (%d, %d)", gotStart, gotEnd)
	}
}

// TestAllDayBounds_BadDateReturnsFallback documents the regression: bad date
// strings used to produce year-1 timestamps; now they leave the fallback in
// place so callers can keep using upstream StartTime/EndTime.
func TestAllDayBounds_BadDateReturnsFallback(t *testing.T) {
	w := domain.EventWhen{Object: "date", Date: "garbage"}
	gotStart, gotEnd := allDayBounds(w, 1234567890, 1234567890+86400)
	if gotStart != 1234567890 || gotEnd != 1234567890+86400 {
		t.Errorf("bad Date should preserve fallback, got (%d, %d)", gotStart, gotEnd)
	}
}

// TestAllDayBounds_GoodDateOverrides confirms the happy path overrides.
func TestAllDayBounds_GoodDateOverrides(t *testing.T) {
	w := domain.EventWhen{Object: "date", Date: "2026-04-29"}
	gotStart, gotEnd := allDayBounds(w, 0, 0)
	if gotStart != 1777420800 {
		t.Errorf("StartTime: want 1777420800, got %d", gotStart)
	}
	if gotEnd != 1777420800+86400 {
		t.Errorf("EndTime: want %d, got %d", 1777420800+86400, gotEnd)
	}
}
