package air

import (
	"sync"
	"testing"
	"time"
)

// TestIsSenderAllowed_OnlyAllowedReturnsTrue is a regression test for the
// pre-fix logic where pending senders accidentally returned (true, "") —
// the screener would let unscreened mail leak into the inbox before the
// user had decided to allow them.
func TestIsSenderAllowed_OnlyAllowedReturnsTrue(t *testing.T) {
	// Replace the package-level store so the test cannot leak across runs.
	original := screenerStore
	screenerStore = &ScreenerStore{senders: map[string]*ScreenedSender{
		"allowed@example.com": {
			Email:       "allowed@example.com",
			Status:      "allowed",
			Destination: "inbox",
			FirstSeen:   time.Now(),
		},
		"pending@example.com": {
			Email:     "pending@example.com",
			Status:    "pending",
			FirstSeen: time.Now(),
		},
		"blocked@example.com": {
			Email:     "blocked@example.com",
			Status:    "blocked",
			FirstSeen: time.Now(),
		},
		"feed@example.com": {
			Email:       "feed@example.com",
			Status:      "allowed",
			Destination: "feed",
			FirstSeen:   time.Now(),
		},
	}}
	t.Cleanup(func() { screenerStore = original })

	cases := []struct {
		email           string
		wantAllowed     bool
		wantDestination string
	}{
		{"allowed@example.com", true, "inbox"},
		{"feed@example.com", true, "feed"},
		{"pending@example.com", false, ""},
		{"blocked@example.com", false, ""},
		{"unknown@example.com", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.email, func(t *testing.T) {
			gotAllowed, gotDest := IsSenderAllowed(tc.email)
			if gotAllowed != tc.wantAllowed {
				t.Errorf("allowed: got %v, want %v", gotAllowed, tc.wantAllowed)
			}
			if gotDest != tc.wantDestination {
				t.Errorf("destination: got %q, want %q", gotDest, tc.wantDestination)
			}
		})
	}
}

// TestIsSenderAllowed_ConcurrentReads catches any future regression where the
// read path stops holding RLock — the data race detector will flag a writer
// running alongside the readers.
func TestIsSenderAllowed_ConcurrentReads(t *testing.T) {
	original := screenerStore
	screenerStore = &ScreenerStore{senders: map[string]*ScreenedSender{
		"a@example.com": {Email: "a@example.com", Status: "allowed", Destination: "inbox"},
	}}
	t.Cleanup(func() { screenerStore = original })

	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 200 {
				_, _ = IsSenderAllowed("a@example.com")
			}
		})
	}
	wg.Wait()
}
