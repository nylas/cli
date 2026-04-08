package tui

import (
	"testing"
)

func TestNewCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry()

	if registry == nil {
		t.Fatal("NewCommandRegistry() returned nil")
		return
	}

	// Verify commands were registered
	if len(registry.commands) == 0 {
		t.Error("Registry has no commands")
	}

	// Verify byName map is populated
	if len(registry.byName) == 0 {
		t.Error("Registry byName map is empty")
	}

	// Verify byCategory map is populated
	if len(registry.byCategory) == 0 {
		t.Error("Registry byCategory map is empty")
	}
}

func TestCommandRegistry_Get(t *testing.T) {
	registry := NewCommandRegistry()

	tests := []struct {
		name      string
		query     string
		wantName  string
		wantFound bool
	}{
		{"primary name", "messages", "messages", true},
		{"alias m", "m", "messages", true},
		{"alias msg", "msg", "messages", true},
		{"events primary", "events", "events", true},
		{"events alias e", "e", "events", true},
		{"events alias cal", "cal", "events", true},
		{"not found", "nonexistent", "", false},
		{"empty string", "", "", false},
		{"with spaces", "  messages  ", "messages", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := registry.Get(tt.query)
			if tt.wantFound {
				if cmd == nil {
					t.Errorf("Get(%q) returned nil, want command", tt.query)
					return
				}
				if cmd.Name != tt.wantName {
					t.Errorf("Get(%q).Name = %q, want %q", tt.query, cmd.Name, tt.wantName)
				}
			} else {
				if cmd != nil {
					t.Errorf("Get(%q) = %v, want nil", tt.query, cmd.Name)
				}
			}
		})
	}
}

func TestCommandRegistry_Search(t *testing.T) {
	registry := NewCommandRegistry()

	tests := []struct {
		name      string
		query     string
		wantFirst string
		wantMin   int // Minimum expected results
	}{
		{"exact match", "messages", "messages", 1},
		{"alias match", "m", "messages", 1},
		{"prefix match", "mes", "messages", 1},
		{"fuzzy match msg", "msg", "messages", 1},
		{"empty returns all", "", "", 5}, // At least 5 commands
		{"no match", "xyz123", "", 0},
		{"compose", "compose", "compose", 1},
		{"partial comp", "comp", "compose", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := registry.Search(tt.query)

			if len(results) < tt.wantMin {
				t.Errorf("Search(%q) returned %d results, want at least %d", tt.query, len(results), tt.wantMin)
			}

			if tt.wantFirst != "" && len(results) > 0 {
				if results[0].Name != tt.wantFirst {
					t.Errorf("Search(%q)[0].Name = %q, want %q", tt.query, results[0].Name, tt.wantFirst)
				}
			}
		})
	}
}

func TestCommandRegistry_GetByCategory(t *testing.T) {
	registry := NewCommandRegistry()

	groups := registry.GetByCategory()

	if len(groups) == 0 {
		t.Fatal("GetByCategory() returned no groups")
	}

	// Check that Navigation category exists and has commands
	foundNavigation := false
	for _, group := range groups {
		if group.Category == CategoryNavigation {
			foundNavigation = true
			if len(group.Commands) == 0 {
				t.Error("Navigation category has no commands")
			}
		}
	}

	if !foundNavigation {
		t.Error("Navigation category not found")
	}
}

func TestCommandRegistry_GetSubCommands(t *testing.T) {
	registry := NewCommandRegistry()

	tests := []struct {
		name    string
		parent  string
		wantLen int
	}{
		{"folder subcommands", "folder", 3},   // list, create, delete
		{"event subcommands", "event", 3},     // new, edit, delete
		{"rsvp subcommands", "rsvp", 3},       // yes, no, maybe
		{"webhook subcommands", "webhook", 4}, // new, edit, delete, test
		{"no subcommands", "messages", 0},
		{"nonexistent", "nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subs := registry.GetSubCommands(tt.parent)
			if len(subs) != tt.wantLen {
				t.Errorf("GetSubCommands(%q) returned %d commands, want %d", tt.parent, len(subs), tt.wantLen)
			}
		})
	}
}

func TestCommandRegistry_HasSubCommands(t *testing.T) {
	registry := NewCommandRegistry()

	tests := []struct {
		name    string
		cmd     string
		wantHas bool
	}{
		{"folder has subcommands", "folder", true},
		{"event has subcommands", "event", true},
		{"messages no subcommands", "messages", false},
		{"quit no subcommands", "quit", false},
		{"nonexistent", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has := registry.HasSubCommands(tt.cmd)
			if has != tt.wantHas {
				t.Errorf("HasSubCommands(%q) = %v, want %v", tt.cmd, has, tt.wantHas)
			}
		})
	}
}

func TestCommandRegistry_SearchSubCommands(t *testing.T) {
	registry := NewCommandRegistry()

	tests := []struct {
		name    string
		parent  string
		query   string
		wantLen int
	}{
		{"all folder subs", "folder", "", 3},
		{"folder create", "folder", "create", 1},
		{"folder cr", "folder", "cr", 1},
		{"folder no match", "folder", "xyz", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := registry.SearchSubCommands(tt.parent, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("SearchSubCommands(%q, %q) returned %d results, want %d",
					tt.parent, tt.query, len(results), tt.wantLen)
			}
		})
	}
}

func TestCommand_AllNames(t *testing.T) {
	cmd := Command{
		Name:    "messages",
		Aliases: []string{"m", "msg"},
	}

	names := cmd.AllNames()

	if len(names) != 3 {
		t.Errorf("AllNames() returned %d names, want 3", len(names))
	}

	if names[0] != "messages" {
		t.Errorf("AllNames()[0] = %q, want 'messages'", names[0])
	}
}

func TestCommand_DisplayAliases(t *testing.T) {
	tests := []struct {
		name    string
		aliases []string
		want    string
	}{
		{"with aliases", []string{"m", "msg"}, "m, msg"},
		{"no aliases", []string{}, ""},
		{"single alias", []string{"m"}, "m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command{Aliases: tt.aliases}
			got := cmd.DisplayAliases()
			if got != tt.want {
				t.Errorf("DisplayAliases() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		target string
		query  string
		want   bool
	}{
		{"messages", "msg", true},
		{"messages", "mgs", true},
		{"messages", "mes", true},
		{"messages", "xyz", false},
		{"compose", "cps", true},
		{"compose", "abc", false},
		{"webhook-server", "ws", true},
		{"webhook-server", "whs", true},
	}

	for _, tt := range tests {
		t.Run(tt.target+"_"+tt.query, func(t *testing.T) {
			got := fuzzyMatch(tt.target, tt.query)
			if got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.target, tt.query, got, tt.want)
			}
		})
	}
}

func TestMatchScore(t *testing.T) {
	tests := []struct {
		target string
		query  string
		want   int
	}{
		{"messages", "messages", 0}, // Exact match
		{"messages", "mess", 1},     // Prefix match
		{"messages", "sage", 2},     // Contains match
		{"messages", "msg", 3},      // Fuzzy match
		{"messages", "xyz", -1},     // No match
	}

	for _, tt := range tests {
		t.Run(tt.target+"_"+tt.query, func(t *testing.T) {
			got := matchScore(tt.target, tt.query)
			if got != tt.want {
				t.Errorf("matchScore(%q, %q) = %d, want %d", tt.target, tt.query, got, tt.want)
			}
		})
	}
}

func TestCategoryOrder(t *testing.T) {
	// Verify category order includes all expected categories
	expectedCategories := []CommandCategory{
		CategoryNavigation,
		CategoryMessages,
		CategoryCalendar,
		CategoryContacts,
		CategoryWebhooks,
		CategoryFolders,
		CategoryVim,
		CategorySystem,
	}

	if len(categoryOrder) != len(expectedCategories) {
		t.Errorf("categoryOrder has %d categories, want %d", len(categoryOrder), len(expectedCategories))
	}

	for i, cat := range expectedCategories {
		if categoryOrder[i] != cat {
			t.Errorf("categoryOrder[%d] = %q, want %q", i, categoryOrder[i], cat)
		}
	}
}
