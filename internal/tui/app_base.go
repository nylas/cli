// Package tui provides a k9s-style terminal user interface for Nylas.
package tui

import (
	"sync"
	"time"

	"github.com/nylas/cli/internal/ports"
	"github.com/rivo/tview"
)

// Config holds the TUI configuration.
type Config struct {
	Client          ports.NylasClient
	GrantStore      ports.GrantStore  // Optional: enables grant switching in TUI
	ConfigStore     ports.ConfigStore // Optional: when set, default-grant changes mirror to config.yaml
	GrantID         string
	Email           string
	Provider        string
	RefreshInterval time.Duration
	InitialView     string    // Initial view to navigate to (messages, events, contacts, webhooks, grants)
	Theme           ThemeName // Theme name (k9s, amber, green, apple2, vintage, ibm, futuristic, matrix)
}

// App is the main TUI application using tview (like k9s).
type App struct {
	*tview.Application

	// Layout components (k9s style)
	main    *tview.Flex
	header  *tview.Flex
	logo    *Logo
	status  *StatusIndicator
	crumbs  *Crumbs
	menu    *Menu
	prompt  *Prompt         // For filter mode (/)
	palette *CommandPalette // For command mode (:) with autocomplete

	// Content area with page stack (like k9s)
	content *PageStack

	// Command registry for help and autocomplete
	cmdRegistry *CommandRegistry

	// State
	config      Config
	styles      *Styles
	running     bool
	mx          sync.RWMutex
	cmdActive   bool
	filterMode  bool
	lastKey     rune      // For vim-style 'gg' command
	lastKeyTime time.Time // Timeout for key sequences

	// View registry
	views map[string]ResourceView
}

// NewApp creates a new TUI application.
func NewApp(cfg Config) *App {
	// Use theme from config, default to k9s if not specified
	var styles *Styles
	if cfg.Theme != "" {
		styles = GetThemeStyles(cfg.Theme)
	} else {
		styles = DefaultStyles()
	}

	app := &App{
		Application: tview.NewApplication(),
		config:      cfg,
		styles:      styles,
		views:       make(map[string]ResourceView),
	}

	app.init()
	return app
}
