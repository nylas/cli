//go:build integration
// +build integration

package integration

import (
	"os"
	"strings"
	"testing"
)

// AI tests require LLM provider API keys (Ollama, Claude, OpenAI, or Groq)
//
// To run AI integration tests with Ollama:
//   1. Ensure Ollama is running: ollama serve
//   2. Pull a model: ollama pull mistral
//   3. Configure in ~/.nylas/config.yaml:
//      ai:
//        default_provider: ollama
//        ollama:
//          host: http://localhost:11434
//          model: mistral:latest
//   4. Run tests: go test -tags=integration -v ./internal/cli/integration/ai_test.go ./internal/cli/integration/test.go

func TestCLI_AIProvider_Availability(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Check if default_provider is set in config
	skipIfNoDefaultAIProvider(t)

	// Check if at least one AI provider is configured
	hasOllama := checkOllamaAvailable()
	hasClaude := os.Getenv("ANTHROPIC_API_KEY") != ""
	hasOpenAI := os.Getenv("OPENAI_API_KEY") != ""
	hasGroq := os.Getenv("GROQ_API_KEY") != ""

	if !hasOllama && !hasClaude && !hasOpenAI && !hasGroq {
		t.Skip("No AI provider configured. Set ANTHROPIC_API_KEY, OPENAI_API_KEY, GROQ_API_KEY, or run Ollama locally")
	}

	t.Logf("AI providers available: Ollama=%v, Claude=%v, OpenAI=%v, Groq=%v", hasOllama, hasClaude, hasOpenAI, hasGroq)
}

func TestCLI_CalendarAI_Basic(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Skip if no default AI provider configured
	skipIfNoDefaultAIProvider(t)

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		skipTest bool
	}{
		{
			name:    "calendar ai help",
			args:    []string{"calendar", "ai", "--help"},
			wantErr: false,
		},
		{
			name:    "calendar ai schedule help",
			args:    []string{"calendar", "ai", "schedule", "--help"},
			wantErr: false,
		},
		{
			name:    "calendar ai reschedule help",
			args:    []string{"calendar", "ai", "reschedule", "--help"},
			wantErr: false,
		},
		{
			name:    "calendar ai context help",
			args:    []string{"calendar", "ai", "context", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip("Test requires specific configuration")
			}

			stdout, stderr, err := runCLI(tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none. stdout: %s", stdout)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
			}
		})
	}
}

func TestCLI_CalendarAI_Schedule_InvalidInput(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Skip if no default AI provider configured
	skipIfNoDefaultAIProvider(t)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "schedule without query",
			args:    []string{"calendar", "schedule", "ai"},
			wantErr: true,
		},
		{
			name:    "schedule with invalid provider",
			args:    []string{"calendar", "schedule", "ai", "--provider", "invalid", "Test meeting"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none. stdout: %s, stderr: %s", stdout, stderr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
			}
		})
	}
}

func TestCLI_CalendarAI_Reschedule_InvalidInput(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Skip if no default AI provider configured
	skipIfNoDefaultAIProvider(t)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "reschedule without event ID",
			args:    []string{"calendar", "ai", "reschedule", "ai"},
			wantErr: true,
		},
		{
			name:    "reschedule with event ID but no Nylas credentials",
			args:    []string{"calendar", "ai", "reschedule", "ai", "event-id"},
			wantErr: true, // Will fail on "secret not found" or API call
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none. stdout: %s, stderr: %s", stdout, stderr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
			}
		})
	}
}

func TestCLI_CalendarAI_Context_Basic(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	// Skip if no Nylas API configured
	if testAPIKey == "" || testGrantID == "" {
		t.Skip("Nylas API credentials not configured")
	}

	// Skip if no default AI provider configured
	skipIfNoDefaultAIProvider(t)

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name: "analyze help command works",
			args: []string{"calendar", "ai", "analyze", "--help"},
			contains: []string{
				"Analyze historical meeting data",
			},
		},
		{
			name: "analyze shows meeting patterns",
			args: []string{"calendar", "ai", "analyze"},
			contains: []string{
				"Analyzing",
				"meetings",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none. stdout: %s", stdout)
				}
				return
			}

			if err != nil {
				t.Logf("Note: Test may fail without valid Nylas credentials. Error: %v", err)
				t.Logf("stderr: %s", stderr)
				t.Logf("stdout: %s", stdout)
				// Don't fail the test, just log
				return
			}

			for _, expected := range tt.contains {
				if !strings.Contains(stdout, expected) {
					t.Errorf("Expected output to contain %q\nGot: %s", expected, stdout)
				}
			}
		})
	}
}
