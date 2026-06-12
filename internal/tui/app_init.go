// Package tui provides a k9s-style terminal user interface for Nylas.
package tui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) init() {
	// Create command registry
	a.cmdRegistry = NewCommandRegistry()

	// Create components (k9s style)
	a.logo = NewLogo(a.styles)
	a.status = NewStatusIndicator(a.styles, a.config)
	a.crumbs = NewCrumbs(a.styles)
	a.menu = NewMenu(a.styles)
	a.prompt = NewPrompt(a.styles, a.onCommand, a.onFilter)
	a.palette = NewCommandPalette(a, a.cmdRegistry, a.onPaletteExecute, a.onPaletteCancel)
	a.content = NewPageStack()

	// Header: Logo on left, Status on right (like k9s)
	a.header = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.logo, 12, 0, false).
		AddItem(a.status, 0, 1, false)

	// Main layout (vertical flex - like k9s)
	// Layout: Header -> Crumbs -> Content -> Menu
	a.main = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 1, 0, false).
		AddItem(a.crumbs, 1, 0, false).
		AddItem(a.content, 0, 1, true). // Content takes remaining space
		AddItem(a.menu, 1, 0, false)

	// Set up key bindings
	a.setupKeys()

	// Initialize with specified view or dashboard
	initialView := a.config.InitialView
	if initialView == "" {
		initialView = "dashboard"
	}
	a.navigateTo(initialView)

	// Set root and enable mouse
	a.SetRoot(a.main, true)
	a.EnableMouse(true)
}

func (a *App) setupKeys() {
	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If palette is active (command mode), let it handle input
		if a.cmdActive && a.palette.IsVisible() {
			// Palette handles its own input via SetInputCapture
			return event
		}

		// If filter mode is active, let prompt handle input
		if a.filterMode {
			return a.prompt.HandleKey(event)
		}

		// Check if we're in a detail view (compose, message-detail, etc.)
		// Detail views are pushed onto the stack but not registered in views map
		topPage := a.content.Top()
		currentView := a.getCurrentView()
		inDetailView := currentView == nil && a.content.Len() > 0

		// If in a detail view (compose, message-detail, help, etc.), only handle Escape and Ctrl+C
		if inDetailView {
			switch event.Key() {
			case tcell.KeyCtrlC:
				a.Stop()
				return nil
			case tcell.KeyEscape:
				// Pop the detail view
				a.content.Pop()
				// Re-focus the underlying view
				if view := a.getCurrentView(); view != nil {
					a.SetFocus(view.Primitive())
				}
				return nil
			}
			// Let the detail view handle all other keys (for typing in compose form)
			return event
		}

		switch event.Key() {
		case tcell.KeyCtrlC:
			// Quit with Ctrl+C
			a.Stop()
			return nil

		case tcell.KeyEscape:
			// First, let the current view handle Escape (for closing details, etc.)
			if currentView != nil {
				result := currentView.HandleKey(event)
				if result == nil {
					// View handled the Escape
					return nil
				}
			}

			// If view didn't handle it, go back in navigation
			return a.goBack()

		case tcell.KeyCtrlD:
			// Half page down (vim-style)
			a.pageMove(10, true)
			return nil

		case tcell.KeyCtrlU:
			// Half page up (vim-style)
			a.pageMove(10, false)
			return nil

		case tcell.KeyCtrlF:
			// Full page down (vim-style)
			a.pageMove(20, true)
			return nil

		case tcell.KeyCtrlB:
			// Full page up (vim-style)
			a.pageMove(20, false)
			return nil

		case tcell.KeyRune:
			switch event.Rune() {
			case 'q', 'Q':
				// Quit the application (k9s/less/top convention).
				// Suppressed in compose/forms via the inDetailView branch above
				// and in command/filter mode via the early returns above.
				a.Stop()
				return nil

			case ':':
				// Enter command mode with palette (autocomplete)
				a.showPalette()
				return nil

			case '/':
				// Enter filter mode
				a.filterMode = true
				a.prompt.Activate(PromptFilter)
				a.showPrompt()
				return nil

			case '?':
				// Show help
				a.showHelp()
				return nil

			case 'r':
				// Refresh (lowercase only - uppercase R is for reply)
				if currentView != nil {
					currentView.Refresh()
				}
				return nil

			case 'g':
				// Handle 'gg' sequence for go-to-top (vim-style)
				now := time.Now()
				if a.lastKey == 'g' && now.Sub(a.lastKeyTime) < 500*time.Millisecond {
					// 'gg' pressed - go to top
					a.goToTop()
					a.lastKey = 0
					return nil
				}
				a.lastKey = 'g'
				a.lastKeyTime = now
				return nil

			case 'G':
				// Go to bottom (vim-style)
				a.goToBottom()
				return nil

			case 'd':
				// Handle 'dd' sequence for delete (vim-style)
				now := time.Now()
				if a.lastKey == 'd' && now.Sub(a.lastKeyTime) < 500*time.Millisecond {
					// 'dd' pressed - delete current item
					a.executeCommand("delete")
					a.lastKey = 0
					return nil
				}
				a.lastKey = 'd'
				a.lastKeyTime = now
				return nil

			case 'x':
				// Delete/archive current item (vim-style)
				a.executeCommand("delete")
				return nil
			}
		}

		// Pass to current view
		if currentView != nil {
			return currentView.HandleKey(event)
		}

		// Handle topPage for debugging
		_ = topPage

		return event
	})
}
