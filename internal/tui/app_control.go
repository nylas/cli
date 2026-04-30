// Package tui provides a k9s-style terminal user interface for Nylas.
package tui

import (
	"fmt"
	"time"

	authapp "github.com/nylas/cli/internal/app/auth"
)

func (a *App) Run() error {
	a.mx.Lock()
	a.running = true
	a.mx.Unlock()

	// Start status update ticker
	go a.statusTicker()

	return a.Application.Run()
}

func (a *App) statusTicker() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		a.mx.RLock()
		running := a.running
		a.mx.RUnlock()

		if !running {
			return
		}

		<-ticker.C
		a.QueueUpdateDraw(func() {
			a.status.Update()
		})
	}
}

// Stop stops the application.
func (a *App) Stop() {
	a.mx.Lock()
	a.running = false
	a.mx.Unlock()
	a.Application.Stop()
}

// Flash displays a temporary message.
func (a *App) Flash(level FlashLevel, msg string, args ...any) {
	a.status.Flash(level, fmt.Sprintf(msg, args...))
}

// Styles returns the app styles.
func (a *App) Styles() *Styles {
	return a.styles
}

// Config returns the app config.
func (a *App) GetConfig() Config {
	return a.config
}

// SwitchGrant switches to a different grant and updates the UI.
// Returns an error if GrantStore is not configured or the switch fails.
func (a *App) SwitchGrant(grantID, email, provider string) error {
	if a.config.GrantStore == nil {
		return fmt.Errorf("grant switching not available (no grant store)")
	}

	// Set the new default grant. Mirror to config.yaml via PersistDefaultGrant
	// so the TUI matches every other write path (auth switch, Air, login flow).
	if err := authapp.PersistDefaultGrant(a.config.ConfigStore, a.config.GrantStore, grantID); err != nil {
		return fmt.Errorf("failed to switch grant: %w", err)
	}

	// Update config
	a.config.GrantID = grantID
	a.config.Email = email
	a.config.Provider = provider

	// Update status indicator
	a.status.UpdateGrant(email, provider, grantID)

	// Refresh the current view to load data for the new grant
	if view := a.getCurrentView(); view != nil {
		go func() {
			a.QueueUpdateDraw(func() {
				view.Refresh()
			})
		}()
	}

	return nil
}

// CanSwitchGrant returns true if grant switching is available.
func (a *App) CanSwitchGrant() bool {
	return a.config.GrantStore != nil
}

// ============================================================================
// Vim-style Navigation Helpers
// ============================================================================

// pageMove moves the selection up or down by the specified amount.
