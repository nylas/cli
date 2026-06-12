package tui

// FlashLoadError flashes an error from a Load() call in the status bar.
// Safe to call from any goroutine, including from inside a QueueUpdateDraw
// callback — Flash dispatches the status mutation in a new goroutine so
// callers never block the event loop waiting on itself.
func (a *App) FlashLoadError(fallback string, err error) {
	a.Flash(FlashError, "%s: %v", fallback, err)
}
