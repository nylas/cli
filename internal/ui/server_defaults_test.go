package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// =============================================================================
// GetDefaultCommands Tests
// =============================================================================

func TestGetDefaultCommands(t *testing.T) {
	t.Parallel()

	cmds := GetDefaultCommands()

	// Verify all categories have commands
	if len(cmds.Auth) == 0 {
		t.Error("Auth commands should not be empty")
	}
	if len(cmds.Email) == 0 {
		t.Error("Email commands should not be empty")
	}
	if len(cmds.Calendar) == 0 {
		t.Error("Calendar commands should not be empty")
	}
	if len(cmds.Contacts) == 0 {
		t.Error("Contacts commands should not be empty")
	}
	if len(cmds.Scheduler) == 0 {
		t.Error("Scheduler commands should not be empty")
	}
	if len(cmds.Timezone) == 0 {
		t.Error("Timezone commands should not be empty")
	}
	if len(cmds.Webhook) == 0 {
		t.Error("Webhook commands should not be empty")
	}
	if len(cmds.OTP) == 0 {
		t.Error("OTP commands should not be empty")
	}
	if len(cmds.Admin) == 0 {
		t.Error("Admin commands should not be empty")
	}
	if len(cmds.Notetaker) == 0 {
		t.Error("Notetaker commands should not be empty")
	}
}

func TestGetDefaultCommands_RequiredFields(t *testing.T) {
	t.Parallel()

	cmds := GetDefaultCommands()

	// Check all commands have required fields
	allCommands := []Command{}
	allCommands = append(allCommands, cmds.Auth...)
	allCommands = append(allCommands, cmds.Email...)
	allCommands = append(allCommands, cmds.Calendar...)
	allCommands = append(allCommands, cmds.Contacts...)
	allCommands = append(allCommands, cmds.Scheduler...)
	allCommands = append(allCommands, cmds.Timezone...)
	allCommands = append(allCommands, cmds.Webhook...)
	allCommands = append(allCommands, cmds.OTP...)
	allCommands = append(allCommands, cmds.Admin...)
	allCommands = append(allCommands, cmds.Notetaker...)

	for _, cmd := range allCommands {
		if cmd.Key == "" {
			t.Errorf("Command has empty Key: %+v", cmd)
		}
		if cmd.Title == "" {
			t.Errorf("Command %q has empty Title", cmd.Key)
		}
		if cmd.Cmd == "" {
			t.Errorf("Command %q has empty Cmd", cmd.Key)
		}
		if cmd.Desc == "" {
			t.Errorf("Command %q has empty Desc", cmd.Key)
		}
	}
}

func TestGetDefaultCommands_ParamCommands(t *testing.T) {
	t.Parallel()

	cmds := GetDefaultCommands()

	// Find commands that require parameters
	allCommands := []Command{}
	allCommands = append(allCommands, cmds.Auth...)
	allCommands = append(allCommands, cmds.Email...)
	allCommands = append(allCommands, cmds.Calendar...)
	allCommands = append(allCommands, cmds.Contacts...)
	allCommands = append(allCommands, cmds.Scheduler...)
	allCommands = append(allCommands, cmds.Timezone...)
	allCommands = append(allCommands, cmds.Webhook...)
	allCommands = append(allCommands, cmds.OTP...)
	allCommands = append(allCommands, cmds.Admin...)
	allCommands = append(allCommands, cmds.Notetaker...)

	paramCommands := 0
	for _, cmd := range allCommands {
		if cmd.ParamName != "" {
			paramCommands++
			if cmd.Placeholder == "" {
				t.Errorf("Command %q has ParamName but no Placeholder", cmd.Key)
			}
		}
	}

	// Verify we have some commands that take parameters
	if paramCommands == 0 {
		t.Error("Expected some commands to have parameters (read, show, search)")
	}
}

// =============================================================================
// PageData JSON Methods Tests
// =============================================================================

func TestPageData_GrantsJSON_Empty(t *testing.T) {
	t.Parallel()

	data := PageData{
		Grants: []Grant{},
	}

	result := string(data.GrantsJSON())
	if result != "[]" {
		t.Errorf("Expected '[]' for empty grants, got: %s", result)
	}
}

func TestPageData_GrantsJSON_WithData(t *testing.T) {
	t.Parallel()

	data := PageData{
		Grants: []Grant{
			{ID: "id-1", Email: "user1@example.com", Provider: "google"},
			{ID: "id-2", Email: "user2@example.com", Provider: "microsoft"},
		},
	}

	result := string(data.GrantsJSON())

	// Verify it's valid JSON
	var grants []Grant
	if err := json.Unmarshal([]byte(result), &grants); err != nil {
		t.Fatalf("Failed to unmarshal GrantsJSON result: %v", err)
	}

	if len(grants) != 2 {
		t.Errorf("Expected 2 grants, got %d", len(grants))
	}
}

func TestPageData_CommandsJSON(t *testing.T) {
	t.Parallel()

	data := PageData{
		Commands: GetDefaultCommands(),
	}

	result := string(data.CommandsJSON())

	// Verify it's valid JSON
	var cmds Commands
	if err := json.Unmarshal([]byte(result), &cmds); err != nil {
		t.Fatalf("Failed to unmarshal CommandsJSON result: %v", err)
	}

	if len(cmds.Auth) == 0 {
		t.Error("Expected auth commands in JSON")
	}
}

// =============================================================================
// getDemoCommandOutput Tests
// =============================================================================

func TestGetDemoCommandOutput_AllCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		command  string
		contains []string
	}{
		// Email commands
		{"email list", []string{"Demo Mode", "alice@example.com", "Showing"}},
		{"email threads", []string{"Demo Mode", "Standup", "threads"}},

		// Calendar commands
		{"calendar list", []string{"Demo Mode", "Work Calendar", "PRIMARY"}},
		{"calendar events", []string{"Demo Mode", "Team Standup", "upcoming events"}},

		// Auth commands
		{"auth status", []string{"Demo Mode", "Configured", "alice@example.com"}},
		{"auth list", []string{"Demo Mode", "Connected Accounts", "demo-grant"}},

		// Contacts commands
		{"contacts list", []string{"Demo Mode", "Alice Johnson", "contact"}},
		{"contacts list --id", []string{"Demo Mode", "demo-contact-001"}},
		{"contacts groups", []string{"Demo Mode", "Contact Groups", "Work"}},

		// Scheduler commands
		{"scheduler configurations", []string{"Demo Mode", "30-min Meeting", "DURATION"}},
		{"scheduler bookings", []string{"Demo Mode", "Bookings", "UPCOMING"}},
		{"scheduler sessions", []string{"Demo Mode", "Sessions", "Active"}},
		{"scheduler pages", []string{"Demo Mode", "Scheduling Pages", "meet-with-alice"}},

		// Timezone commands
		{"timezone list", []string{"Demo Mode", "Time Zones", "America/New_York"}},
		{"timezone info", []string{"Demo Mode", "Time Zone Info", "DST"}},
		{"timezone convert", []string{"Demo Mode", "Time Conversion", "FROM", "TO"}},
		{"timezone find-meeting", []string{"Demo Mode", "Meeting Time Finder", "Best meeting times"}},
		{"timezone dst", []string{"Demo Mode", "DST Transitions", "Spring Forward", "Fall Back"}},

		// Webhook commands
		{"webhook list", []string{"Demo Mode", "Webhooks", "wh-001"}},
		{"webhook triggers", []string{"Demo Mode", "Webhook Triggers", "message.created"}},
		{"webhook test", []string{"Demo Mode", "Webhook Test", "200 OK"}},
		{"webhook server", []string{"Demo Mode", "Webhook Server", "localhost"}},

		// OTP commands
		{"otp get", []string{"Demo Mode", "OTP Code", "GitHub"}},
		{"otp watch", []string{"Demo Mode", "Watching for OTP"}},
		{"otp list", []string{"Demo Mode", "Configured OTP Accounts"}},
		{"otp messages", []string{"Demo Mode", "Recent OTP Messages", "GitHub"}},

		// Admin commands
		{"admin applications", []string{"Demo Mode", "Applications", "Production App"}},
		{"admin connectors", []string{"Demo Mode", "Connectors", "Google Workspace"}},
		{"admin credentials", []string{"Demo Mode", "Credentials", "oauth2"}},
		{"admin grants", []string{"Demo Mode", "Grants", "alice@example.com"}},

		// Notetaker commands
		{"notetaker list", []string{"Demo Mode", "Notetakers", "Team Standup"}},
		{"notetaker create", []string{"Demo Mode", "Create Notetaker", "nt-004"}},
		{"notetaker media", []string{"Demo Mode", "Notetaker Media", "Video Recording"}},

		// Version command
		{"version", []string{"nylas version dev", "demo mode"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := getDemoCommandOutput(tt.command)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("getDemoCommandOutput(%q) missing %q in output:\n%s",
						tt.command, expected, result)
				}
			}
		})
	}
}

func TestGetDemoCommandOutput_EmptyCommand(t *testing.T) {
	t.Parallel()

	result := getDemoCommandOutput("")
	if !strings.Contains(result, "no command specified") {
		t.Errorf("Expected 'no command specified' for empty command, got: %s", result)
	}
}

func TestGetDemoCommandOutput_WhitespaceCommand(t *testing.T) {
	t.Parallel()

	result := getDemoCommandOutput("   ")
	if !strings.Contains(result, "no command specified") {
		t.Errorf("Expected 'no command specified' for whitespace command, got: %s", result)
	}
}

func TestGetDemoCommandOutput_UnknownCommand(t *testing.T) {
	t.Parallel()

	result := getDemoCommandOutput("unknown command here")
	if !strings.Contains(result, "Demo Mode - Command:") {
		t.Errorf("Expected fallback message for unknown command, got: %s", result)
	}
	if !strings.Contains(result, "sample output") {
		t.Errorf("Expected sample output message, got: %s", result)
	}
}

func TestGetDemoCommandOutput_CommandWithFlags(t *testing.T) {
	t.Parallel()

	// Commands with flags should still match on base command
	result := getDemoCommandOutput("email list --limit 10 --unread")
	if !strings.Contains(result, "Demo Mode") {
		t.Errorf("Expected demo output for command with flags, got: %s", result)
	}
}

// =============================================================================
// grantFromDomain Tests
// =============================================================================

func TestGrantFromDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    domain.GrantInfo
		expected Grant
	}{
		{
			name: "google provider",
			input: domain.GrantInfo{
				ID:       "grant-123",
				Email:    "test@example.com",
				Provider: domain.ProviderGoogle,
			},
			expected: Grant{
				ID:       "grant-123",
				Email:    "test@example.com",
				Provider: "google",
			},
		},
		{
			name: "microsoft provider",
			input: domain.GrantInfo{
				ID:       "grant-456",
				Email:    "user@work.com",
				Provider: domain.ProviderMicrosoft,
			},
			expected: Grant{
				ID:       "grant-456",
				Email:    "user@work.com",
				Provider: "microsoft",
			},
		},
		{
			name: "empty fields",
			input: domain.GrantInfo{
				ID:       "",
				Email:    "",
				Provider: "",
			},
			expected: Grant{
				ID:       "",
				Email:    "",
				Provider: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grantFromDomain(tt.input)

			if result.ID != tt.expected.ID {
				t.Errorf("ID: got %q, want %q", result.ID, tt.expected.ID)
			}
			if result.Email != tt.expected.Email {
				t.Errorf("Email: got %q, want %q", result.Email, tt.expected.Email)
			}
			if result.Provider != tt.expected.Provider {
				t.Errorf("Provider: got %q, want %q", result.Provider, tt.expected.Provider)
			}
		})
	}
}

// =============================================================================
