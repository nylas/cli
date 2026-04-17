package tui

// registerAllCommands registers all TUI commands.
func (r *CommandRegistry) registerAllCommands() {
	// =========================================================================
	// Navigation Commands
	// =========================================================================
	r.Register(Command{
		Name:        "messages",
		Aliases:     []string{"m", "msg"},
		Description: "Go to messages view",
		Category:    CategoryNavigation,
	})
	r.Register(Command{
		Name:        "events",
		Aliases:     []string{"e", "ev", "cal", "calendar"},
		Description: "Go to calendar events view",
		Category:    CategoryNavigation,
	})
	r.Register(Command{
		Name:        "contacts",
		Aliases:     []string{"c", "ct"},
		Description: "Go to contacts view",
		Category:    CategoryNavigation,
	})
	r.Register(Command{
		Name:        "webhooks",
		Aliases:     []string{"w", "wh"},
		Description: "Go to webhooks view",
		Category:    CategoryNavigation,
	})
	r.Register(Command{
		Name:        "webhook-server",
		Aliases:     []string{"ws", "whs", "server"},
		Description: "Go to webhook server view",
		Category:    CategoryNavigation,
	})
	r.Register(Command{
		Name:        "grants",
		Aliases:     []string{"g", "gr"},
		Description: "Go to grants/accounts view",
		Category:    CategoryNavigation,
	})
	r.Register(Command{
		Name:        "dashboard",
		Aliases:     []string{"d", "dash", "home"},
		Description: "Go to dashboard",
		Category:    CategoryNavigation,
	})

	// =========================================================================
	// Message Commands
	// =========================================================================
	r.Register(Command{
		Name:        "compose",
		Aliases:     []string{"n", "new"},
		Description: "Compose new email",
		Category:    CategoryMessages,
		Shortcut:    "n",
	})
	r.Register(Command{
		Name:        "reply",
		Aliases:     []string{"r"},
		Description: "Reply to current message",
		Category:    CategoryMessages,
		Shortcut:    "R",
		ContextView: "messages",
	})
	r.Register(Command{
		Name:        "replyall",
		Aliases:     []string{"ra", "reply-all"},
		Description: "Reply all to message",
		Category:    CategoryMessages,
		Shortcut:    "A",
		ContextView: "messages",
	})
	r.Register(Command{
		Name:        "forward",
		Aliases:     []string{"f", "fwd"},
		Description: "Forward message",
		Category:    CategoryMessages,
		ContextView: "messages",
	})
	r.Register(Command{
		Name:        "star",
		Aliases:     []string{"s"},
		Description: "Toggle star on message",
		Category:    CategoryMessages,
		Shortcut:    "s",
		ContextView: "messages",
	})
	r.Register(Command{
		Name:        "unstar",
		Aliases:     []string{},
		Description: "Remove star from message",
		Category:    CategoryMessages,
		ContextView: "messages",
	})
	r.Register(Command{
		Name:        "read",
		Aliases:     []string{"mr"},
		Description: "Mark as read",
		Category:    CategoryMessages,
		ContextView: "messages",
	})
	r.Register(Command{
		Name:        "unread",
		Aliases:     []string{"mu"},
		Description: "Mark as unread",
		Category:    CategoryMessages,
		Shortcut:    "u",
		ContextView: "messages",
	})
	r.Register(Command{
		Name:        "delete",
		Aliases:     []string{"del", "rm"},
		Description: "Delete current item",
		Category:    CategoryMessages,
		Shortcut:    "dd",
	})
	r.Register(Command{
		Name:        "archive",
		Aliases:     []string{},
		Description: "Archive message",
		Category:    CategoryMessages,
		ContextView: "messages",
	})

	// =========================================================================
	// Calendar Commands (with sub-commands)
	// =========================================================================
	r.Register(Command{
		Name:        "event",
		Aliases:     []string{},
		Description: "Event management",
		Category:    CategoryCalendar,
		ContextView: "events",
		SubCommands: []Command{
			{Name: "new", Aliases: []string{"create"}, Description: "Create new event"},
			{Name: "edit", Aliases: []string{"update"}, Description: "Edit current event"},
			{Name: "delete", Aliases: []string{"del"}, Description: "Delete current event"},
		},
	})
	r.Register(Command{
		Name:        "rsvp",
		Aliases:     []string{},
		Description: "RSVP to event",
		Category:    CategoryCalendar,
		ContextView: "events",
		SubCommands: []Command{
			{Name: "yes", Description: "RSVP yes to event"},
			{Name: "no", Description: "RSVP no to event"},
			{Name: "maybe", Description: "RSVP maybe to event"},
		},
	})
	r.Register(Command{
		Name:        "availability",
		Aliases:     []string{"avail"},
		Description: "Check availability",
		Category:    CategoryCalendar,
	})
	r.Register(Command{
		Name:        "find-time",
		Aliases:     []string{"findtime"},
		Description: "Find meeting time",
		Category:    CategoryCalendar,
	})

	// =========================================================================
	// Contact Commands (with sub-commands)
	// =========================================================================
	r.Register(Command{
		Name:        "contact",
		Aliases:     []string{},
		Description: "Contact management",
		Category:    CategoryContacts,
		ContextView: "contacts",
		SubCommands: []Command{
			{Name: "new", Aliases: []string{"create"}, Description: "Create new contact"},
			{Name: "edit", Aliases: []string{"update"}, Description: "Edit current contact"},
			{Name: "delete", Aliases: []string{"del"}, Description: "Delete current contact"},
		},
	})

	// =========================================================================
	// Webhook Commands (with sub-commands)
	// =========================================================================
	r.Register(Command{
		Name:        "webhook",
		Aliases:     []string{},
		Description: "Webhook management",
		Category:    CategoryWebhooks,
		ContextView: "webhooks",
		SubCommands: []Command{
			{Name: "new", Aliases: []string{"create"}, Description: "Create new webhook"},
			{Name: "edit", Aliases: []string{"update"}, Description: "Edit current webhook"},
			{Name: "delete", Aliases: []string{"del"}, Description: "Delete current webhook"},
			{Name: "test", Description: "Test current webhook"},
		},
	})

	// =========================================================================
	// Folder Commands (with sub-commands)
	// =========================================================================
	r.Register(Command{
		Name:        "folder",
		Aliases:     []string{},
		Description: "Folder management",
		Category:    CategoryFolders,
		ContextView: "messages",
		SubCommands: []Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List all folders"},
			{Name: "create", Aliases: []string{"new"}, Description: "Create new folder"},
			{Name: "delete", Aliases: []string{"del"}, Description: "Delete folder"},
		},
	})
	r.Register(Command{
		Name:        "inbox",
		Aliases:     []string{},
		Description: "Go to inbox folder",
		Category:    CategoryFolders,
	})
	r.Register(Command{
		Name:        "sent",
		Aliases:     []string{},
		Description: "Go to sent folder",
		Category:    CategoryFolders,
	})
	r.Register(Command{
		Name:        "trash",
		Aliases:     []string{},
		Description: "Go to trash folder",
		Category:    CategoryFolders,
	})
	r.Register(Command{
		Name:        "drafts",
		Aliases:     []string{"dr"},
		Description: "Go to drafts",
		Category:    CategoryFolders,
	})

	// =========================================================================
	// Vim Commands
	// =========================================================================
	r.Register(Command{
		Name:        "quit",
		Aliases:     []string{"q", "exit"},
		Description: "Quit application",
		Category:    CategoryVim,
	})
	r.Register(Command{
		Name:        "quit!",
		Aliases:     []string{"q!"},
		Description: "Force quit",
		Category:    CategoryVim,
	})
	r.Register(Command{
		Name:        "wq",
		Aliases:     []string{"x"},
		Description: "Save and quit",
		Category:    CategoryVim,
	})
	r.Register(Command{
		Name:        "help",
		Aliases:     []string{"h"},
		Description: "Show help",
		Category:    CategoryVim,
		Shortcut:    "?",
	})
	r.Register(Command{
		Name:        "top",
		Aliases:     []string{"first", "gg"},
		Description: "Go to first row",
		Category:    CategoryVim,
		Shortcut:    "gg",
	})
	r.Register(Command{
		Name:        "bottom",
		Aliases:     []string{"last", "G"},
		Description: "Go to last row",
		Category:    CategoryVim,
		Shortcut:    "G",
	})

	// =========================================================================
	// System Commands
	// =========================================================================
	r.Register(Command{
		Name:        "refresh",
		Aliases:     []string{"reload"},
		Description: "Refresh current view",
		Category:    CategorySystem,
		Shortcut:    "r",
	})
}
