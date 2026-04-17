package ui

import (
	"strings"
	"testing"
)

// =============================================================================
// Command Validation Tests
// =============================================================================

func TestAllowedCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		allowed bool
	}{
		// Auth commands
		{"auth login", "auth login", true},
		{"auth logout", "auth logout", true},
		{"auth status", "auth status", true},
		{"auth whoami", "auth whoami", true},
		{"auth list", "auth list", true},
		{"auth show", "auth show", true},
		{"auth switch", "auth switch", true},
		{"auth config", "auth config", true},
		{"auth providers", "auth providers", true},

		// Email commands
		{"email list", "email list", true},
		{"email read", "email read", true},
		{"email send", "email send", true},
		{"email search", "email search", true},
		{"email delete", "email delete", true},
		{"email folders list", "email folders list", true},
		{"email threads list", "email threads list", true},
		{"email drafts create", "email drafts create", true},

		// Calendar commands
		{"calendar list", "calendar list", true},
		{"calendar events list", "calendar events list", true},
		{"calendar events show", "calendar events show", true},
		{"calendar availability check", "calendar availability check", true},

		// Contacts commands
		{"contacts list", "contacts list", true},
		{"contacts list --id", "contacts list --id", true},
		{"contacts show", "contacts show", true},
		{"contacts create", "contacts create", true},
		{"contacts search", "contacts search", true},
		{"contacts groups", "contacts groups", true},

		// Scheduler commands
		{"scheduler configurations", "scheduler configurations", true},
		{"scheduler sessions", "scheduler sessions", true},
		{"scheduler bookings", "scheduler bookings", true},
		{"scheduler pages", "scheduler pages", true},

		// Timezone commands
		{"timezone list", "timezone list", true},
		{"timezone info", "timezone info", true},
		{"timezone convert", "timezone convert", true},
		{"timezone find-meeting", "timezone find-meeting", true},
		{"timezone dst", "timezone dst", true},

		// Webhook commands
		{"webhook list", "webhook list", true},
		{"webhook show", "webhook show", true},
		{"webhook create", "webhook create", true},
		{"webhook update", "webhook update", true},
		{"webhook delete", "webhook delete", true},
		{"webhook triggers", "webhook triggers", true},
		{"webhook test", "webhook test", true},
		{"webhook server", "webhook server", true},

		// OTP commands
		{"otp get", "otp get", true},
		{"otp watch", "otp watch", true},
		{"otp list", "otp list", true},
		{"otp messages", "otp messages", true},

		// Admin commands
		{"admin applications", "admin applications", true},
		{"admin connectors", "admin connectors", true},
		{"admin credentials", "admin credentials", true},
		{"admin grants", "admin grants", true},

		// Notetaker commands
		{"notetaker list", "notetaker list", true},
		{"notetaker show", "notetaker show", true},
		{"notetaker create", "notetaker create", true},
		{"notetaker delete", "notetaker delete", true},
		{"notetaker media", "notetaker media", true},

		// Version
		{"version", "version", true},

		// Blocked commands
		{"rm command", "rm -rf /", false},
		{"shell injection", "email list; rm -rf /", false},
		{"unknown command", "unknown command", false},
		{"empty command", "", false},
		{"sudo", "sudo anything", false},
		{"curl", "curl http://evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommandAllowed(tt.command)
			if result != tt.allowed {
				t.Errorf("isCommandAllowed(%q) = %v, want %v", tt.command, result, tt.allowed)
			}
		})
	}
}

// isCommandAllowed checks if a command is in the allowlist.
// This is extracted for testing purposes.
func isCommandAllowed(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}

	// Check for shell metacharacters (defense in depth)
	if containsDangerousChars(cmd) {
		return false
	}

	args := strings.Fields(cmd)
	if len(args) == 0 {
		return false
	}

	// Try 3-word command first
	if len(args) >= 3 {
		baseCmd := args[0] + " " + args[1] + " " + args[2]
		if allowedCommands[baseCmd] {
			return true
		}
	}

	// Try 2-word command
	if len(args) >= 2 {
		baseCmd := args[0] + " " + args[1]
		if allowedCommands[baseCmd] {
			return true
		}
	}

	// Try 1-word command
	if len(args) >= 1 {
		baseCmd := args[0]
		if allowedCommands[baseCmd] {
			return true
		}
	}

	return false
}

// =============================================================================

// =============================================================================
// Command Whitelist Completeness Tests
// =============================================================================

func TestAllowedCommandsCompleteness(t *testing.T) {
	t.Parallel()

	// Verify all expected command categories are present
	expectedPrefixes := []string{
		"auth",
		"email",
		"calendar",
		"contacts",
		"scheduler",
		"timezone",
		"webhook",
		"otp",
		"admin",
		"notetaker",
		"version",
	}

	for _, prefix := range expectedPrefixes {
		found := false
		for cmd := range allowedCommands {
			if strings.HasPrefix(cmd, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("No commands found with prefix %q", prefix)
		}
	}
}

func TestAllowedCommands_NoShellCharacters(t *testing.T) {
	t.Parallel()

	// Verify no allowed commands contain shell metacharacters
	dangerousChars := []string{";", "|", "&", "`", "$", "(", ")", "<", ">", "\\"}

	for cmd := range allowedCommands {
		for _, char := range dangerousChars {
			if strings.Contains(cmd, char) {
				t.Errorf("Allowed command %q contains dangerous character %q", cmd, char)
			}
		}
	}
}

// =============================================================================
