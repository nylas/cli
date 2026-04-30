package air

import (
	"testing"
	"time"
)

// TestIsFocusModeActive_ExpiresAtEndsAt is a regression test for the bug
// where ShouldAllowNotification kept returning false long after the focus
// session window had elapsed because the read handler only updated a local
// copy of the state. Now both IsFocusModeActive and ShouldAllowNotification
// honour EndsAt at call time.
func TestIsFocusModeActive_ExpiresAtEndsAt(t *testing.T) {
	original := fmStore
	t.Cleanup(func() { fmStore = original })

	now := time.Now()

	t.Run("active before EndsAt", func(t *testing.T) {
		fmStore = &focusModeStore{
			state: &FocusModeState{
				IsActive:  true,
				StartedAt: now.Add(-10 * time.Minute),
				EndsAt:    now.Add(15 * time.Minute),
			},
			settings: original.settings,
		}
		if !IsFocusModeActive() {
			t.Error("expected active session to report true")
		}
	})

	t.Run("expired after EndsAt", func(t *testing.T) {
		fmStore = &focusModeStore{
			state: &FocusModeState{
				IsActive:  true,
				StartedAt: now.Add(-2 * time.Hour),
				EndsAt:    now.Add(-1 * time.Hour),
			},
			settings: original.settings,
		}
		if IsFocusModeActive() {
			t.Error("expected expired session to report false")
		}
	})

	t.Run("not started", func(t *testing.T) {
		fmStore = &focusModeStore{
			state:    &FocusModeState{IsActive: false},
			settings: original.settings,
		}
		if IsFocusModeActive() {
			t.Error("expected inactive session to report false")
		}
	})
}

func TestShouldAllowNotification_HonoursExpiry(t *testing.T) {
	original := fmStore
	t.Cleanup(func() { fmStore = original })

	now := time.Now()
	fmStore = &focusModeStore{
		state: &FocusModeState{
			IsActive:  true,
			StartedAt: now.Add(-2 * time.Hour),
			EndsAt:    now.Add(-1 * time.Hour),
		},
		settings: &FocusModeSettings{
			HideNotifications: true,
			AllowedSenders:    []string{"vip@example.com"},
		},
	}

	// Once the session is past EndsAt, *every* sender should pass through
	// — the user is no longer in focus mode.
	if !ShouldAllowNotification("random@example.com") {
		t.Error("expired session must allow notifications")
	}
}

func TestShouldAllowNotification_VIPDuringActiveSession(t *testing.T) {
	original := fmStore
	t.Cleanup(func() { fmStore = original })

	now := time.Now()
	fmStore = &focusModeStore{
		state: &FocusModeState{
			IsActive:  true,
			StartedAt: now,
			EndsAt:    now.Add(15 * time.Minute),
		},
		settings: &FocusModeSettings{
			HideNotifications: true,
			AllowedSenders:    []string{"vip@example.com"},
		},
	}

	if !ShouldAllowNotification("vip@example.com") {
		t.Error("VIP sender must always pass during active focus")
	}
	if ShouldAllowNotification("noise@example.com") {
		t.Error("non-VIP sender must be blocked during active focus")
	}
}
