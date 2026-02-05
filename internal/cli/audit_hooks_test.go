package cli

import (
	"os"
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func TestSetAuditRequestInfo(t *testing.T) {
	tests := []struct {
		name       string
		setup      func()
		requestID  string
		httpStatus int
		wantID     string
		wantStatus int
	}{
		{
			name: "sets request info when audit context exists",
			setup: func() {
				auditMu.Lock()
				currentAudit = &AuditContext{}
				auditMu.Unlock()
			},
			requestID:  "req-123",
			httpStatus: 200,
			wantID:     "req-123",
			wantStatus: 200,
		},
		{
			name: "does nothing when audit context is nil",
			setup: func() {
				auditMu.Lock()
				currentAudit = nil
				auditMu.Unlock()
			},
			requestID:  "req-456",
			httpStatus: 500,
			wantID:     "",
			wantStatus: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			SetAuditRequestInfo(tt.requestID, tt.httpStatus)

			auditMu.Lock()
			defer auditMu.Unlock()

			if currentAudit != nil {
				if currentAudit.RequestID != tt.wantID {
					t.Errorf("RequestID = %q, want %q", currentAudit.RequestID, tt.wantID)
				}
				if currentAudit.HTTPStatus != tt.wantStatus {
					t.Errorf("HTTPStatus = %d, want %d", currentAudit.HTTPStatus, tt.wantStatus)
				}
			}
		})
	}

	// Cleanup
	auditMu.Lock()
	currentAudit = nil
	auditMu.Unlock()
}

func TestSetAuditGrantInfo(t *testing.T) {
	tests := []struct {
		name       string
		setup      func()
		grantID    string
		grantEmail string
		wantID     string
		wantEmail  string
	}{
		{
			name: "sets grant info when audit context exists",
			setup: func() {
				auditMu.Lock()
				currentAudit = &AuditContext{}
				auditMu.Unlock()
			},
			grantID:    "grant-123",
			grantEmail: "alice@example.com",
			wantID:     "grant-123",
			wantEmail:  "alice@example.com",
		},
		{
			name: "does nothing when audit context is nil",
			setup: func() {
				auditMu.Lock()
				currentAudit = nil
				auditMu.Unlock()
			},
			grantID:    "grant-456",
			grantEmail: "bob@example.com",
			wantID:     "",
			wantEmail:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			SetAuditGrantInfo(tt.grantID, tt.grantEmail)

			auditMu.Lock()
			defer auditMu.Unlock()

			if currentAudit != nil {
				if currentAudit.GrantID != tt.wantID {
					t.Errorf("GrantID = %q, want %q", currentAudit.GrantID, tt.wantID)
				}
				if currentAudit.GrantEmail != tt.wantEmail {
					t.Errorf("GrantEmail = %q, want %q", currentAudit.GrantEmail, tt.wantEmail)
				}
			}
		})
	}

	// Cleanup
	auditMu.Lock()
	currentAudit = nil
	auditMu.Unlock()
}

func TestGetCommandPath(t *testing.T) {
	tests := []struct {
		name     string
		setupCmd func() *cobra.Command
		want     string
	}{
		{
			name: "single command",
			setupCmd: func() *cobra.Command {
				return &cobra.Command{Use: "list"}
			},
			want: "list",
		},
		{
			name: "nested command under nylas",
			setupCmd: func() *cobra.Command {
				root := &cobra.Command{Use: "nylas"}
				email := &cobra.Command{Use: "email"}
				list := &cobra.Command{Use: "list"}
				root.AddCommand(email)
				email.AddCommand(list)
				return list
			},
			want: "email list",
		},
		{
			name: "deeply nested command",
			setupCmd: func() *cobra.Command {
				root := &cobra.Command{Use: "nylas"}
				email := &cobra.Command{Use: "email"}
				attachments := &cobra.Command{Use: "attachments"}
				download := &cobra.Command{Use: "download"}
				root.AddCommand(email)
				email.AddCommand(attachments)
				attachments.AddCommand(download)
				return download
			},
			want: "email attachments download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.setupCmd()
			got := getCommandPath(cmd)
			if got != tt.want {
				t.Errorf("getCommandPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsExcludedCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmdName string
		want    bool
	}{
		{"help command excluded", "help", true},
		{"version command excluded", "version", true},
		{"completion command excluded", "completion", true},
		{"__complete excluded", "__complete", true},
		{"__completeNoDesc excluded", "__completeNoDesc", true},
		{"email command not excluded", "email", false},
		{"list command not excluded", "list", false},
		{"audit command not excluded", "audit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: tt.cmdName}
			got := isExcludedCommand(cmd)
			if got != tt.want {
				t.Errorf("isExcludedCommand(%q) = %v, want %v", tt.cmdName, got, tt.want)
			}
		})
	}
}

func TestSanitizeArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "empty args",
			args: []string{},
			want: []string{},
		},
		{
			name: "no sensitive args",
			args: []string{"--limit", "10", "--format", "json"},
			want: []string{"--limit", "10", "--format", "json"},
		},
		{
			name: "redacts --api-key value",
			args: []string{"--api-key", "secret123"},
			want: []string{"--api-key", "[REDACTED]"},
		},
		{
			name: "redacts --password value",
			args: []string{"--password", "mypassword"},
			want: []string{"--password", "[REDACTED]"},
		},
		{
			name: "redacts --token value",
			args: []string{"--token", "tok_abc123"},
			want: []string{"--token", "[REDACTED]"},
		},
		{
			name: "redacts --secret value",
			args: []string{"--secret", "supersecret"},
			want: []string{"--secret", "[REDACTED]"},
		},
		{
			name: "redacts --client-secret value",
			args: []string{"--client-secret", "clientsecret123"},
			want: []string{"--client-secret", "[REDACTED]"},
		},
		{
			name: "redacts --access-token value",
			args: []string{"--access-token", "access123"},
			want: []string{"--access-token", "[REDACTED]"},
		},
		{
			name: "redacts --refresh-token value",
			args: []string{"--refresh-token", "refresh456"},
			want: []string{"--refresh-token", "[REDACTED]"},
		},
		{
			name: "redacts --body value",
			args: []string{"--body", "sensitive content"},
			want: []string{"--body", "[REDACTED]"},
		},
		{
			name: "redacts --subject value",
			args: []string{"--subject", "Private email subject"},
			want: []string{"--subject", "[REDACTED]"},
		},
		{
			name: "redacts --html value",
			args: []string{"--html", "<html>content</html>"},
			want: []string{"--html", "[REDACTED]"},
		},
		{
			name: "redacts -p short flag",
			args: []string{"-p", "password123"},
			want: []string{"-p", "[REDACTED]"},
		},
		{
			name: "redacts --flag=value format",
			args: []string{"--api-key=secret123"},
			want: []string{"--api-key=[REDACTED]"},
		},
		{
			name: "redacts nyk_ prefixed tokens",
			args: []string{"nyk_abcdef123456789012345678901234567890"},
			want: []string{"[REDACTED]"},
		},
		{
			name: "redacts long base64 strings",
			args: []string{"YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkw"},
			want: []string{"[REDACTED]"},
		},
		{
			name: "mixed args with sensitive and non-sensitive",
			args: []string{"--limit", "10", "--api-key", "secret", "--format", "json"},
			want: []string{"--limit", "10", "--api-key", "[REDACTED]", "--format", "json"},
		},
		{
			name: "multiple sensitive flags",
			args: []string{"--password", "pass1", "--token", "tok1"},
			want: []string{"--password", "[REDACTED]", "--token", "[REDACTED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeArgs(tt.args)
			if len(got) != len(tt.want) {
				t.Errorf("sanitizeArgs() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("sanitizeArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsLongBase64(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"short string", "abc", false},
		{"exactly 39 chars", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklm", false},
		{"40 char base64", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmn", true},
		{"long base64 with numbers", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkw", true},
		{"base64 with plus", "ABCDEFGHIJKLMNOPQRSTUVWXYZ+abcdefghijklmn", true},
		{"base64 with slash", "ABCDEFGHIJKLMNOPQRSTUVWXYZ/abcdefghijklmn", true},
		{"base64 with equals", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk===", true},
		{"base64 with dash (URL safe)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ-abcdefghijklmn", true},
		{"base64 with underscore (URL safe)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmn", true},
		{"contains space", "ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmn", false},
		{"contains special char", "ABCDEFGHIJKLMNOPQRSTUVWXYZ!abcdefghijklmn", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLongBase64(tt.input)
			if got != tt.want {
				t.Errorf("isLongBase64(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

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

func TestLogAuditError(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		err   error
	}{
		{
			name: "does nothing when audit context is nil",
			setup: func() {
				auditMu.Lock()
				currentAudit = nil
				auditMu.Unlock()
			},
			err: nil,
		},
		{
			name: "captures error when audit context exists",
			setup: func() {
				auditMu.Lock()
				currentAudit = &AuditContext{
					Command: "test",
				}
				auditMu.Unlock()
			},
			err: os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			// LogAuditError should not panic
			if tt.err != nil {
				LogAuditError(tt.err)
			} else {
				LogAuditError(nil)
			}
		})
	}

	// Cleanup
	auditMu.Lock()
	currentAudit = nil
	auditMu.Unlock()
}

func TestAuditPreRun(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *cobra.Command
		args    []string
		wantNil bool
	}{
		{
			name: "excludes help command",
			cmd: func() *cobra.Command {
				return &cobra.Command{Use: "help"}
			}(),
			args:    []string{},
			wantNil: true,
		},
		{
			name: "excludes version command",
			cmd: func() *cobra.Command {
				return &cobra.Command{Use: "version"}
			}(),
			args:    []string{},
			wantNil: true,
		},
		{
			name: "excludes audit command",
			cmd: func() *cobra.Command {
				root := &cobra.Command{Use: "nylas"}
				audit := &cobra.Command{Use: "audit"}
				logs := &cobra.Command{Use: "logs"}
				root.AddCommand(audit)
				audit.AddCommand(logs)
				return logs
			}(),
			args:    []string{},
			wantNil: true,
		},
		{
			name: "processes regular command",
			cmd: func() *cobra.Command {
				root := &cobra.Command{Use: "nylas"}
				email := &cobra.Command{Use: "email"}
				list := &cobra.Command{Use: "list"}
				root.AddCommand(email)
				email.AddCommand(list)
				return list
			}(),
			args:    []string{"--limit", "10"},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			auditMu.Lock()
			currentAudit = nil
			auditMu.Unlock()

			err := auditPreRun(tt.cmd, tt.args)
			if err != nil {
				t.Errorf("auditPreRun() error = %v", err)
			}

			auditMu.Lock()
			gotNil := currentAudit == nil
			auditMu.Unlock()

			if gotNil != tt.wantNil {
				t.Errorf("auditPreRun() currentAudit nil = %v, want %v", gotNil, tt.wantNil)
			}
		})
	}

	// Cleanup
	auditMu.Lock()
	currentAudit = nil
	auditMu.Unlock()
}

func TestAuditPostRun(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
	}{
		{
			name: "does nothing when audit context is nil",
			setup: func() {
				auditMu.Lock()
				currentAudit = nil
				auditMu.Unlock()
			},
		},
		{
			name: "clears audit context after run",
			setup: func() {
				auditMu.Lock()
				currentAudit = &AuditContext{
					Command: "test",
				}
				auditMu.Unlock()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			cmd := &cobra.Command{Use: "test"}
			err := auditPostRun(cmd, []string{})
			if err != nil {
				t.Errorf("auditPostRun() error = %v", err)
			}

			auditMu.Lock()
			if currentAudit != nil {
				t.Error("auditPostRun() should clear currentAudit")
			}
			auditMu.Unlock()
		})
	}
}

func TestInitAuditHooks(t *testing.T) {
	rootCmd := &cobra.Command{Use: "nylas"}

	// Should not panic
	initAuditHooks(rootCmd)

	// Verify hooks are set
	if rootCmd.PersistentPreRunE == nil {
		t.Error("initAuditHooks() did not set PersistentPreRunE")
	}
	if rootCmd.PersistentPostRunE == nil {
		t.Error("initAuditHooks() did not set PersistentPostRunE")
	}

	// Verify grant hook is set - call it to test
	if common.AuditGrantHook == nil {
		t.Error("initAuditHooks() did not set AuditGrantHook")
	} else {
		// Test the grant hook (should not panic)
		auditMu.Lock()
		currentAudit = &AuditContext{}
		auditMu.Unlock()

		common.AuditGrantHook("test-grant-id")

		auditMu.Lock()
		if currentAudit.GrantID != "test-grant-id" {
			t.Errorf("AuditGrantHook did not set GrantID, got %q", currentAudit.GrantID)
		}
		currentAudit = nil
		auditMu.Unlock()
	}

	// Verify request hook is set
	if ports.AuditRequestHook == nil {
		t.Error("initAuditHooks() did not set AuditRequestHook")
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
