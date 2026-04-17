// Package tui provides a k9s-style terminal user interface for Nylas.
package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) getCurrentView() ResourceView {
	name := a.content.Top()
	if view, ok := a.views[name]; ok {
		return view
	}
	return nil
}

func (a *App) showPrompt() {
	// Add prompt to layout (before menu)
	a.main.RemoveItem(a.menu)
	a.main.AddItem(a.prompt, 1, 0, true)
	a.main.AddItem(a.menu, 1, 0, false)
	a.SetFocus(a.prompt)
}

func (a *App) hidePrompt() {
	a.main.RemoveItem(a.prompt)
	a.cmdActive = false
	a.filterMode = false

	// Refocus current view
	if view := a.getCurrentView(); view != nil {
		a.SetFocus(view.Primitive())
	}
}

func (a *App) showPalette() {
	// Add palette to layout (before menu)
	a.main.RemoveItem(a.menu)
	a.main.AddItem(a.palette, 12, 0, true) // Give palette more height for dropdown
	a.main.AddItem(a.menu, 1, 0, false)
	a.palette.Show()
	a.cmdActive = true
}

func (a *App) hidePalette() {
	a.main.RemoveItem(a.palette)
	a.palette.Hide()
	a.cmdActive = false

	// Refocus current view
	if view := a.getCurrentView(); view != nil {
		a.SetFocus(view.Primitive())
	}
}

func (a *App) onPaletteExecute(cmd string) {
	a.hidePalette()
	if cmd != "" {
		a.onCommand(cmd)
	}
}

func (a *App) onPaletteCancel() {
	a.hidePalette()
}

func (a *App) onCommand(cmd string) {
	a.hidePrompt()

	if cmd == "" {
		return
	}

	// Handle numeric commands (go to row number)
	if isNumeric(cmd) {
		a.goToRow(parseInt(cmd))
		return
	}

	switch cmd {
	// Navigation - vim style
	case "m", "messages", "msg":
		a.navigateTo("messages")
	case "dr", "drafts":
		a.navigateTo("drafts")
	case "e", "events", "ev", "cal", "calendar":
		a.navigateTo("events")
	case "av", "avail", "availability":
		a.navigateTo("availability")
	case "c", "contacts", "ct":
		a.navigateTo("contacts")
	case "w", "webhooks", "wh":
		a.navigateTo("webhooks")
	case "ws", "webhook-server", "whs", "server":
		a.navigateTo("webhook-server")
	case "g", "grants", "gr":
		a.navigateTo("grants")
	case "d", "dashboard", "dash", "home":
		a.navigateTo("dashboard")

	// Quit commands - vim style
	case "q", "quit", "exit":
		a.Stop()
	case "q!", "quit!":
		a.Stop() // Force quit
	case "wq", "x":
		a.Stop() // Write and quit (just quit for TUI)

	// Help - vim style
	case "h", "help":
		a.showHelp()

	// Actions on current item
	case "delete", "del", "rm":
		a.executeCommand("delete")
	case "star", "s":
		a.executeCommand("star")
	case "unstar":
		a.executeCommand("unstar")
	case "read", "mr":
		a.executeCommand("read")
	case "unread", "mu":
		a.executeCommand("unread")

	// Compose/Reply - vim style
	case "new", "n", "compose":
		a.executeCommand("compose")
	case "reply", "r":
		a.executeCommand("reply")
	case "replyall", "ra", "reply-all":
		a.executeCommand("replyall")
	case "forward", "f", "fwd":
		a.executeCommand("forward")

	// View commands
	case "refresh", "reload":
		if view := a.getCurrentView(); view != nil {
			go func() {
				view.Refresh()
				a.QueueUpdateDraw(func() {})
			}()
		}
	case "top", "first", "gg":
		a.goToTop()
	case "bottom", "last", "G":
		a.goToBottom()

	// Set commands (vim-style :set)
	default:
		// Check for :e <view> pattern (vim-style edit)
		if len(cmd) > 2 && cmd[:2] == "e " {
			viewName := cmd[2:]
			switch viewName {
			case "messages", "m":
				a.navigateTo("messages")
			case "drafts", "dr":
				a.navigateTo("drafts")
			case "events", "ev", "cal":
				a.navigateTo("events")
			case "availability", "av", "avail":
				a.navigateTo("availability")
			case "contacts", "c":
				a.navigateTo("contacts")
			case "webhooks", "w":
				a.navigateTo("webhooks")
			case "grants", "g":
				a.navigateTo("grants")
			}
		}
	}
}

func (a *App) onFilter(filter string) {
	a.hidePrompt()
	if view := a.getCurrentView(); view != nil {
		view.Filter(filter)
		go func() {
			view.Refresh()
			a.QueueUpdateDraw(func() {})
		}()
	}
}

func (a *App) navigateTo(name string) {
	view, ok := a.views[name]
	if !ok {
		view = a.createView(name)
		a.views[name] = view
	}

	// Use page stack for navigation
	a.content.SwitchTo(name, view.Primitive())

	// Update UI
	a.crumbs.SetPath(view.Title())
	a.menu.SetHints(view.Hints())
	a.SetFocus(view.Primitive())

	// Load data asynchronously
	go func() {
		view.Load()
		a.QueueUpdateDraw(func() {})
	}()
}

func (a *App) goBack() *tcell.EventKey {
	// Need at least 2 items to go back (current + previous)
	if a.content.Len() <= 1 {
		return nil
	}

	// Pop current
	a.content.Pop()

	// Update UI for new top
	name := a.content.Top()
	if view, ok := a.views[name]; ok {
		a.crumbs.SetPath(view.Title())
		a.menu.SetHints(view.Hints())
		a.SetFocus(view.Primitive())
	}

	return nil
}

func (a *App) createView(name string) ResourceView {
	switch name {
	case "messages":
		return NewMessagesView(a)
	case "drafts":
		return NewDraftsView(a)
	case "events":
		return NewEventsView(a)
	case "availability":
		return NewAvailabilityView(a)
	case "contacts":
		return NewContactsView(a)
	case "webhooks":
		return NewWebhooksView(a)
	case "webhook-server":
		return NewWebhookServerView(a)
	case "grants":
		return NewGrantsView(a)
	default:
		return NewDashboardView(a)
	}
}

func (a *App) showHelp() {
	// Create help view with callbacks
	onClose := func() {
		a.content.Pop()
		if view := a.getCurrentView(); view != nil {
			a.SetFocus(view.Primitive())
		}
	}
	onExecute := func(cmd string) {
		a.executeCommand(cmd)
	}

	help := NewHelpView(a, a.cmdRegistry, onClose, onExecute)

	// Push help as a page
	a.content.Push("help", help)
	a.SetFocus(help)
}

// PushDetail pushes a detail view onto the stack (for message detail, etc.)
func (a *App) PushDetail(name string, view tview.Primitive) {
	a.content.Push(name, view)
	a.SetFocus(view)
}

// PopDetail pops a detail view from the stack
func (a *App) PopDetail() {
	if a.content.Len() > 1 {
		a.content.Pop()
		if view := a.getCurrentView(); view != nil {
			a.SetFocus(view.Primitive())
		}
	}
}

// Run starts the application.
