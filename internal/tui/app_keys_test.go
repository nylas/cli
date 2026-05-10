package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestAppQKeyQuitsStopsApplication(t *testing.T) {
	tests := []struct {
		name string
		r    rune
	}{
		{"lowercase q", 'q'},
		{"uppercase Q", 'Q'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp(t)
			capture := app.GetInputCapture()
			if capture == nil {
				t.Fatal("expected app input capture to be set")
			}

			app.mx.Lock()
			app.running = true
			app.mx.Unlock()

			event := tcell.NewEventKey(tcell.KeyRune, tt.r, tcell.ModNone)
			result := capture(event)
			if result != nil {
				t.Errorf("expected %q to be consumed (return nil), got %v", tt.r, result)
			}

			app.mx.RLock()
			running := app.running
			app.mx.RUnlock()
			if running {
				t.Errorf("expected %q to stop the app", tt.r)
			}
		})
	}
}
