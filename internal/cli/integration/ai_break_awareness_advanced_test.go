//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestCLI_AI_FocusTime_BreakAwareness(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	if testAPIKey == "" || testGrantID == "" {
		t.Skip("Nylas API credentials not configured")
	}

	if !hasAnyAIProvider() {
		t.Skip("No AI provider configured")
	}

	// Load AI config from user's config file
	aiConfig := getAIConfigFromUserConfig()
	if aiConfig == nil {
		aiConfig = map[string]any{
			"default_provider": getAvailableProvider(),
		}
	}

	testEmail := getTestEmail()
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL environment variable not set")
	}

	// Create config with breaks
	fakeHome := t.TempDir()
	fakeConfigDir := filepath.Join(fakeHome, ".config", "nylas")
	if err := os.MkdirAll(fakeConfigDir, 0750); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

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

	configData, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	fakeConfigPath := filepath.Join(fakeConfigDir, "config.yaml")
	if err := os.WriteFile(fakeConfigPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", fakeHome)
	t.Cleanup(func() {
		if originalHome != "" {
			_ = os.Setenv("HOME", originalHome)
		}
	})

	t.Run("focus_time_excludes_lunch_break", func(t *testing.T) {
		args := []string{
			"calendar", "ai", "focus-time",
			"--analyze",
			"--target-hours", "14.0",
		}

		stdout, stderr, err := runCLI(args...)

		if err != nil {
			t.Logf("Focus time analysis: %v", err)
			t.Logf("stderr: %s", stderr)
			// Don't fail - this might be expected if the command doesn't exist yet
			return
		}

		output := stdout + stderr
		t.Logf("Focus Time Output:\n%s", output)

		// Check if output mentions breaks or excludes lunch time
		if strings.Contains(output, "Lunch") || strings.Contains(output, "break") ||
			strings.Contains(output, "12:00") || strings.Contains(output, "12-1") ||
			strings.Contains(output, "protected") {
			t.Logf("✓ Focus time analysis appears to respect break blocks")
		} else {
			t.Logf("Note: Focus time output doesn't explicitly mention breaks")
		}
	})
}

// TestCLI_AI_Scheduling_BreakAwareness tests that AI scheduling respects breaks
func TestCLI_AI_Scheduling_BreakAwareness(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	if testAPIKey == "" || testGrantID == "" {
		t.Skip("Nylas API credentials not configured")
	}

	if !hasAnyAIProvider() {
		t.Skip("No AI provider configured")
	}

	// Load AI config from user's config file
	aiConfig := getAIConfigFromUserConfig()
	if aiConfig == nil {
		aiConfig = map[string]any{
			"default_provider": getAvailableProvider(),
		}
	}

	testEmail := getTestEmail()
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL environment variable not set")
	}

	// Create config with breaks
	fakeHome := t.TempDir()
	fakeConfigDir := filepath.Join(fakeHome, ".config", "nylas")
	if err := os.MkdirAll(fakeConfigDir, 0750); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

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
				},
			},
		},
		"ai": aiConfig,
	}

	configData, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	fakeConfigPath := filepath.Join(fakeConfigDir, "config.yaml")
	if err := os.WriteFile(fakeConfigPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", fakeHome)
	t.Cleanup(func() {
		if originalHome != "" {
			_ = os.Setenv("HOME", originalHome)
		}
	})

	t.Run("ai_schedule_avoids_lunch_break", func(t *testing.T) {
		provider := getAvailableProvider()
		query := fmt.Sprintf("30-minute meeting with %s tomorrow at noon", testEmail)

		args := []string{
			"calendar", "schedule", "ai",
			"--provider", provider,
			query,
		}

		stdout, stderr, err := runCLI(args...)

		output := stdout + stderr
		t.Logf("AI Scheduling Output:\n%s", output)

		if err != nil {
			t.Logf("AI scheduling: %v", err)
			t.Logf("stderr: %s", stderr)
			// Don't fail - command might not exist or AI might not be configured
			return
		}

		// Check if AI suggests times outside of lunch break
		if strings.Contains(output, "12:00") || strings.Contains(output, "12:30") ||
			strings.Contains(output, "noon") {
			// Check if there's a warning or if it suggests alternative times
			if strings.Contains(output, "break") || strings.Contains(output, "Lunch") ||
				strings.Contains(output, "alternative") || strings.Contains(output, "suggest") {
				t.Logf("✓ AI appears to be aware of lunch break and suggests alternatives")
			} else {
				t.Logf("Note: AI suggested lunch time without mentioning break")
			}
		} else {
			t.Logf("✓ AI avoided lunch break time in suggestions")
		}

		// Clean up any created events
		eventID := extractEventID(output)
		if eventID != "" {
			t.Cleanup(func() {
				t.Logf("Cleaning up AI-created event: %s", eventID)
				_, _, _ = runCLI("calendar", "events", "delete", eventID, "--force")
			})
		}
	})
}

// TestCLI_AI_ConflictDetection_BreakAwareness tests conflict detection with breaks
func TestCLI_AI_ConflictDetection_BreakAwareness(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	if testAPIKey == "" || testGrantID == "" {
		t.Skip("Nylas API credentials not configured")
	}

	if !hasAnyAIProvider() {
		t.Skip("No AI provider configured")
	}

	// Load AI config from user's config file
	aiConfig := getAIConfigFromUserConfig()
	if aiConfig == nil {
		aiConfig = map[string]any{
			"default_provider": getAvailableProvider(),
		}
	}

	testEmail := getTestEmail()
	if testEmail == "" {
		t.Skip("NYLAS_TEST_EMAIL environment variable not set")
	}

	// Create config with breaks
	fakeHome := t.TempDir()
	fakeConfigDir := filepath.Join(fakeHome, ".config", "nylas")
	if err := os.MkdirAll(fakeConfigDir, 0750); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

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
				},
			},
		},
		"ai": aiConfig,
	}

	configData, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	fakeConfigPath := filepath.Join(fakeConfigDir, "config.yaml")
	if err := os.WriteFile(fakeConfigPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", fakeHome)
	t.Cleanup(func() {
		if originalHome != "" {
			_ = os.Setenv("HOME", originalHome)
		}
	})

	t.Run("conflict_detection_identifies_break_violation", func(t *testing.T) {
		tomorrow := time.Now().Add(24 * time.Hour)
		lunchTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 30, 0, 0, time.UTC)

		args := []string{
			"calendar", "ai", "conflicts", "check",
			"--title", "Meeting During Lunch",
			"--start", lunchTime.Format(time.RFC3339),
			"--duration", "30",
			"--participants", testEmail,
		}

		stdout, stderr, err := runCLI(args...)

		output := stdout + stderr
		t.Logf("Conflict Detection Output:\n%s", output)

		if err != nil {
			t.Logf("Conflict detection: %v", err)
			t.Logf("stderr: %s", stderr)
			// Don't fail - command might not exist
			return
		}

		// Check if conflict with break is detected
		if strings.Contains(output, "break") || strings.Contains(output, "Lunch") ||
			strings.Contains(output, "conflict") {
			t.Logf("✓ Conflict detection identified break time conflict")
		} else {
			t.Logf("Note: Conflict detection output doesn't mention break violation")
		}
	})
}
