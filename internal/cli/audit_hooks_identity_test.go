package cli

import (
	"os"
	"testing"
)

func TestGetInvokerIdentity(t *testing.T) {
	// Save original env vars
	origClaudeProjectDir := os.Getenv("CLAUDE_PROJECT_DIR")
	origCopilotModel := os.Getenv("COPILOT_MODEL")
	origGHCopilot := os.Getenv("GH_COPILOT")
	origNylasInvokerSource := os.Getenv("NYLAS_INVOKER_SOURCE")
	origSSHClient := os.Getenv("SSH_CLIENT")

	// Cleanup function to restore env vars
	cleanup := func() {
		setEnvOrUnset("CLAUDE_PROJECT_DIR", origClaudeProjectDir)
		setEnvOrUnset("COPILOT_MODEL", origCopilotModel)
		setEnvOrUnset("GH_COPILOT", origGHCopilot)
		setEnvOrUnset("NYLAS_INVOKER_SOURCE", origNylasInvokerSource)
		setEnvOrUnset("SSH_CLIENT", origSSHClient)
		// Clear any CLAUDE_CODE_ env vars we set
		for _, env := range os.Environ() {
			if len(env) > 12 && env[:12] == "CLAUDE_CODE_" {
				key := env[:len(env)-len(env[12:])]
				if idx := indexOf(env, '='); idx > 0 {
					key = env[:idx]
				}
				_ = os.Unsetenv(key)
			}
		}
	}
	defer cleanup()

	tests := []struct {
		name       string
		setup      func()
		wantSource string
	}{
		{
			name: "detects claude-code via CLAUDE_PROJECT_DIR",
			setup: func() {
				cleanup()
				_ = os.Setenv("CLAUDE_PROJECT_DIR", "/home/user/project")
			},
			wantSource: "claude-code",
		},
		{
			name: "detects claude-code via CLAUDE_CODE_ prefix",
			setup: func() {
				cleanup()
				_ = os.Setenv("CLAUDE_CODE_ENABLE_TELEMETRY", "1")
			},
			wantSource: "claude-code",
		},
		{
			name: "detects github-copilot via COPILOT_MODEL",
			setup: func() {
				cleanup()
				_ = os.Setenv("COPILOT_MODEL", "gpt-4")
			},
			wantSource: "github-copilot",
		},
		{
			name: "detects github-copilot via GH_COPILOT",
			setup: func() {
				cleanup()
				_ = os.Setenv("GH_COPILOT", "1")
			},
			wantSource: "github-copilot",
		},
		{
			name: "uses NYLAS_INVOKER_SOURCE override",
			setup: func() {
				cleanup()
				_ = os.Setenv("NYLAS_INVOKER_SOURCE", "custom-tool")
			},
			wantSource: "custom-tool",
		},
		{
			name: "detects ssh via SSH_CLIENT",
			setup: func() {
				cleanup()
				_ = os.Setenv("SSH_CLIENT", "192.168.1.1 12345 22")
			},
			wantSource: "ssh",
		},
		{
			name: "claude-code takes precedence over copilot",
			setup: func() {
				cleanup()
				_ = os.Setenv("CLAUDE_PROJECT_DIR", "/home/user/project")
				_ = os.Setenv("COPILOT_MODEL", "gpt-4")
			},
			wantSource: "claude-code",
		},
		{
			name: "copilot takes precedence over override",
			setup: func() {
				cleanup()
				_ = os.Setenv("COPILOT_MODEL", "gpt-4")
				_ = os.Setenv("NYLAS_INVOKER_SOURCE", "custom")
			},
			wantSource: "github-copilot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			_, gotSource := getInvokerIdentity()
			if gotSource != tt.wantSource {
				t.Errorf("getInvokerIdentity() source = %q, want %q", gotSource, tt.wantSource)
			}
		})
	}
}

func TestGetUsername(t *testing.T) {
	// Save original SUDO_USER
	origSudoUser := os.Getenv("SUDO_USER")
	defer func() {
		setEnvOrUnset("SUDO_USER", origSudoUser)
	}()

	tests := []struct {
		name      string
		sudoUser  string
		wantEmpty bool
	}{
		{
			name:      "uses SUDO_USER when set",
			sudoUser:  "originaluser",
			wantEmpty: false,
		},
		{
			name:      "falls back to current user",
			sudoUser:  "",
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvOrUnset("SUDO_USER", tt.sudoUser)
			got := getUsername()

			if tt.wantEmpty && got != "" {
				t.Errorf("getUsername() = %q, want empty", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Error("getUsername() = empty, want non-empty")
			}
			if tt.sudoUser != "" && got != tt.sudoUser {
				t.Errorf("getUsername() = %q, want %q", got, tt.sudoUser)
			}
		})
	}
}

func TestHasClaudeCodeEnv(t *testing.T) {
	// Clear any existing CLAUDE_CODE_ vars first
	for _, env := range os.Environ() {
		if len(env) > 12 && env[:12] == "CLAUDE_CODE_" {
			if idx := indexOf(env, '='); idx > 0 {
				_ = os.Unsetenv(env[:idx])
			}
		}
	}

	tests := []struct {
		name    string
		setup   func()
		cleanup func()
		want    bool
	}{
		{
			name:    "no CLAUDE_CODE_ vars",
			setup:   func() {},
			cleanup: func() {},
			want:    false,
		},
		{
			name: "has CLAUDE_CODE_ENABLE_TELEMETRY",
			setup: func() {
				_ = os.Setenv("CLAUDE_CODE_ENABLE_TELEMETRY", "1")
			},
			cleanup: func() {
				_ = os.Unsetenv("CLAUDE_CODE_ENABLE_TELEMETRY")
			},
			want: true,
		},
		{
			name: "has CLAUDE_CODE_SHELL",
			setup: func() {
				_ = os.Setenv("CLAUDE_CODE_SHELL", "/bin/bash")
			},
			cleanup: func() {
				_ = os.Unsetenv("CLAUDE_CODE_SHELL")
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			got := hasClaudeCodeEnv()
			if got != tt.want {
				t.Errorf("hasClaudeCodeEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInvokerIdentity_TerminalAndScript(t *testing.T) {
	// Save original env vars
	origClaudeProjectDir := os.Getenv("CLAUDE_PROJECT_DIR")
	origCopilotModel := os.Getenv("COPILOT_MODEL")
	origGHCopilot := os.Getenv("GH_COPILOT")
	origNylasInvokerSource := os.Getenv("NYLAS_INVOKER_SOURCE")
	origSSHClient := os.Getenv("SSH_CLIENT")

	// Cleanup function
	defer func() {
		setEnvOrUnset("CLAUDE_PROJECT_DIR", origClaudeProjectDir)
		setEnvOrUnset("COPILOT_MODEL", origCopilotModel)
		setEnvOrUnset("GH_COPILOT", origGHCopilot)
		setEnvOrUnset("NYLAS_INVOKER_SOURCE", origNylasInvokerSource)
		setEnvOrUnset("SSH_CLIENT", origSSHClient)
		// Clear CLAUDE_CODE_ env vars
		_ = os.Unsetenv("CLAUDE_CODE_ENABLE_TELEMETRY")
	}()

	// Clear all detection env vars
	_ = os.Unsetenv("CLAUDE_PROJECT_DIR")
	_ = os.Unsetenv("COPILOT_MODEL")
	_ = os.Unsetenv("GH_COPILOT")
	_ = os.Unsetenv("NYLAS_INVOKER_SOURCE")
	_ = os.Unsetenv("SSH_CLIENT")
	_ = os.Unsetenv("CLAUDE_CODE_ENABLE_TELEMETRY")

	// When no env vars are set, should detect terminal or script
	// (depends on whether stdin is a TTY during test execution)
	invoker, source := getInvokerIdentity()

	if invoker == "" {
		t.Error("getInvokerIdentity() returned empty invoker")
	}
	if source != "terminal" && source != "script" {
		t.Errorf("getInvokerIdentity() source = %q, want terminal or script", source)
	}
}

func TestGetUsername_Unknown(t *testing.T) {
	// This tests the function returns a non-empty value
	// Even if we can't force user.Current() to fail, we verify it doesn't return empty
	result := getUsername()
	if result == "" {
		t.Error("getUsername() returned empty string")
	}
}

// Helper functions

func setEnvOrUnset(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, value)
	}
}

func indexOf(s string, c rune) int {
	for i, r := range s {
		if r == c {
			return i
		}
	}
	return -1
}
