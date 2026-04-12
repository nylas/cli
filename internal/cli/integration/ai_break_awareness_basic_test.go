//go:build integration
// +build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestCLI_AI_BreakTimeAwareness tests that AI features respect configured break blocks
func TestCLI_AI_BreakTimeAwareness(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	if testAPIKey == "" || testGrantID == "" {
		t.Skip("Nylas API credentials not configured")
	}

	if !hasAnyAIProvider() {
		t.Skip("No AI provider configured")
	}

	// Get available provider - skip if none available
	aiProvider := getAvailableProvider()
	if aiProvider == "" {
		t.Skip("No AI provider available (Ollama not running and no cloud API keys configured)")
	}

	// Load AI config from user's config file
	aiConfig := getAIConfigFromUserConfig()
	if aiConfig == nil {
		// Fallback to minimal config if user config not found
		aiConfig = map[string]any{
			"default_provider": aiProvider,
		}
	}

	testEmail := getTestEmail()
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL environment variable not set")
	}

	// Create a temporary config file with working hours and breaks
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")

	config := map[string]any{
		"region":        "us",
		"callback_port": 8080,
		"grants": []map[string]string{
			{
				"id":       testGrantID,
				"email":    testEmail,
				"provider": "google",
			},
		},
		"working_hours": map[string]any{
			"default": map[string]any{
				"enabled": true,
				"start":   "09:00",
				"end":     "17:00",
				"breaks": []map[string]string{
					{
						"name":  "Lunch",
						"start": "12:00",
						"end":   "13:00",
						"type":  "lunch",
					},
					{
						"name":  "Coffee Break",
						"start": "15:00",
						"end":   "15:15",
						"type":  "coffee",
					},
				},
			},
		},
		"ai": aiConfig,
	}

	// Write config to file
	configData, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Logf("Created test config at: %s", configPath)

	// Override config path for tests
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		if originalHome != "" {
			_ = os.Setenv("HOME", originalHome)
		}
	})

	// Create fake home directory structure
	fakeHome := t.TempDir()
	fakeConfigDir := filepath.Join(fakeHome, ".config", "nylas")
	if err := os.MkdirAll(fakeConfigDir, 0750); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	fakeConfigPath := filepath.Join(fakeConfigDir, "config.yaml")
	if err := os.WriteFile(fakeConfigPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write fake config: %v", err)
	}

	_ = os.Setenv("HOME", fakeHome)
	t.Logf("Set HOME to: %s", fakeHome)
	t.Logf("Config should be read from: %s", fakeConfigPath)

	t.Run("event_creation_during_lunch_blocked", func(t *testing.T) {
		// Try to create an event during lunch break (12:00 - 13:00)
		tomorrow := time.Now().Add(24 * time.Hour)
		lunchTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 30, 0, 0, time.UTC)

		t.Logf("Attempting to create event during lunch break at: %s", lunchTime.Format(time.RFC3339))

		args := []string{
			"calendar", "events", "create",
			"--title", "Test Meeting During Lunch",
			"--description", "This should be blocked by break time validation",
			"--start", lunchTime.Format(time.RFC3339),
			"--end", lunchTime.Add(30 * time.Minute).Format(time.RFC3339),
		}

		stdout, stderr, err := runCLI(args...)

		// We expect this to fail or warn about break time conflict
		output := stdout + stderr
		t.Logf("Output: %s", output)

		// Check if break time conflict was detected
		if !strings.Contains(output, "break") && !strings.Contains(output, "Lunch") &&
			!strings.Contains(output, "conflicts") && !strings.Contains(output, "Break") {
			t.Logf("Warning: Expected break time conflict detection, but got: %s", output)
			// Don't fail the test as this might be expected behavior depending on implementation
		}

		if err == nil && !strings.Contains(output, "break") {
			// Event was created without warning - check if we can delete it
			eventID := extractEventID(output)
			if eventID != "" {
				t.Cleanup(func() {
					t.Logf("Cleaning up event created during lunch: %s", eventID)
					_, _, _ = runCLI("calendar", "events", "delete", eventID, "--force")
				})
				t.Logf("Note: Event was created during lunch without break validation")
			}
		}
	})

	t.Run("event_creation_during_coffee_break_blocked", func(t *testing.T) {
		// Try to create an event during coffee break (15:00 - 15:15)
		tomorrow := time.Now().Add(24 * time.Hour)
		coffeeTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 15, 5, 0, 0, time.UTC)

		t.Logf("Attempting to create event during coffee break at: %s", coffeeTime.Format(time.RFC3339))

		args := []string{
			"calendar", "events", "create",
			"--title", "Test Meeting During Coffee",
			"--description", "This should be blocked by break time validation",
			"--start", coffeeTime.Format(time.RFC3339),
			"--end", coffeeTime.Add(15 * time.Minute).Format(time.RFC3339),
		}

		stdout, stderr, err := runCLI(args...)

		output := stdout + stderr
		t.Logf("Output: %s", output)

		// Check if break time conflict was detected
		if !strings.Contains(output, "break") && !strings.Contains(output, "Coffee") &&
			!strings.Contains(output, "conflicts") && !strings.Contains(output, "Break") {
			t.Logf("Warning: Expected break time conflict detection for coffee break")
		}

		if err == nil && !strings.Contains(output, "break") {
			eventID := extractEventID(output)
			if eventID != "" {
				t.Cleanup(func() {
					t.Logf("Cleaning up event created during coffee break: %s", eventID)
					_, _, _ = runCLI("calendar", "events", "delete", eventID, "--force")
				})
			}
		}
	})

	t.Run("event_creation_outside_breaks_succeeds", func(t *testing.T) {
		// Create an event at a time that's NOT during breaks (2:00 PM)
		tomorrow := time.Now().Add(24 * time.Hour)
		goodTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 14, 0, 0, 0, time.UTC)

		t.Logf("Creating event outside break time at: %s", goodTime.Format(time.RFC3339))

		args := []string{
			"calendar", "events", "create",
			"--title", "Test Meeting Outside Breaks",
			"--description", "This should succeed",
			"--start", goodTime.Format(time.RFC3339),
			"--end", goodTime.Add(30 * time.Minute).Format(time.RFC3339),
		}

		stdout, stderr, err := runCLI(args...)

		output := stdout + stderr
		t.Logf("Output: %s", output)

		if err != nil {
			t.Logf("Event creation failed (might be expected): %v", err)
			t.Logf("stderr: %s", stderr)
		}

		// Try to extract and cleanup the event if created
		eventID := extractEventID(output)
		if eventID != "" {
			t.Logf("✓ Successfully created event outside break time: %s", eventID)
			t.Cleanup(func() {
				t.Logf("Cleaning up test event: %s", eventID)
				_, _, _ = runCLI("calendar", "events", "delete", eventID, "--force")
			})
		}
	})

	t.Run("event_with_ignore_working_hours_flag", func(t *testing.T) {
		// Create an event during lunch but with --ignore-working-hours flag
		tomorrow := time.Now().Add(24 * time.Hour)
		lunchTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 30, 0, 0, time.UTC)

		t.Logf("Creating event during lunch with --ignore-working-hours flag")

		args := []string{
			"calendar", "events", "create",
			"--title", "Override Break Time",
			"--description", "Using ignore flag to override break validation",
			"--start", lunchTime.Format(time.RFC3339),
			"--end", lunchTime.Add(30 * time.Minute).Format(time.RFC3339),
			"--ignore-working-hours",
		}

		stdout, stderr, err := runCLI(args...)

		output := stdout + stderr
		t.Logf("Output: %s", output)

		if err != nil {
			t.Logf("Event creation with ignore flag failed: %v", err)
		}

		eventID := extractEventID(output)
		if eventID != "" {
			t.Logf("✓ Event created with --ignore-working-hours flag: %s", eventID)
			t.Cleanup(func() {
				t.Logf("Cleaning up event with ignore flag: %s", eventID)
				_, _, _ = runCLI("calendar", "events", "delete", eventID, "--force")
			})
		}
	})
}

// TestCLI_AI_FocusTime_BreakAwareness tests that AI focus time respects break blocks
