package tui

import "fmt"

// FlashLoadError flashes an error from a Load() call in the status bar.
// Safe to call from any goroutine, including from inside a QueueUpdateDraw
// callback — the inner QueueUpdateDraw is dispatched in a new goroutine
// so callers never block the event loop waiting on itself.
func (a *App) FlashLoadError(fallback string, err error) {
	msg := fmt.Sprintf("%s: %v", fallback, err)
	go func() {
		a.QueueUpdateDraw(func() {
			a.status.Flash(FlashError, msg)
		})
	}()
}
