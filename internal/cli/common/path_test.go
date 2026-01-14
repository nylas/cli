package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateExecutablePath(t *testing.T) {
	// Create a temporary executable file for testing
	tmpDir := t.TempDir()
	validExec := filepath.Join(tmpDir, "test-executable")
	//nolint:gosec // G306: 0755 is intentional - test requires executable file
	if err := os.WriteFile(validExec, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create test executable: %v", err)
	}

	// Create a non-executable file
	nonExec := filepath.Join(tmpDir, "test-non-executable")
	//nolint:gosec // G306: 0644 is intentional - test requires non-executable file
	if err := os.WriteFile(nonExec, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid executable",
			path:      validExec,
			wantError: false,
		},
		{
			name:      "non-executable file",
			path:      nonExec,
			wantError: true,
			errorMsg:  "not executable",
		},
		{
			name:      "non-existent file",
			path:      filepath.Join(tmpDir, "does-not-exist"),
			wantError: true,
			errorMsg:  "not found",
		},
		{
			name:      "empty path",
			path:      "",
			wantError: true,
			errorMsg:  "empty",
		},
		{
			name:      "path with traversal",
			path:      "../../../etc/passwd",
			wantError: true,
			errorMsg:  "path traversal",
		},
		{
			name:      "directory instead of file",
			path:      tmpDir,
			wantError: true,
			errorMsg:  "not a regular file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExecutablePath(tt.path)

			if tt.wantError && err == nil {
				t.Error("ValidateExecutablePath() expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("ValidateExecutablePath() unexpected error: %v", err)
			}

			if tt.wantError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error should contain %q, got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

func TestFindExecutableInPath(t *testing.T) {
	tests := []struct {
		name      string
		execName  string
		wantError bool
	}{
		{
			name:      "find sh",
			execName:  "sh",
			wantError: false,
		},
		{
			name:      "non-existent executable",
			execName:  "this-command-does-not-exist-xyz123",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := FindExecutableInPath(tt.execName)

			if tt.wantError && err == nil {
				t.Error("FindExecutableInPath() expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("FindExecutableInPath() unexpected error: %v", err)
			}

			if !tt.wantError && path == "" {
				t.Error("FindExecutableInPath() returned empty path")
			}

			if !tt.wantError && !filepath.IsAbs(path) {
				t.Errorf("FindExecutableInPath() should return absolute path, got: %s", path)
			}
		})
	}
}

func TestSafeCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		args      []string
		wantError bool
	}{
		{
			name:      "valid command",
			command:   "sh",
			args:      []string{"-c", "echo test"},
			wantError: false,
		},
		{
			name:      "non-existent command",
			command:   "this-does-not-exist-xyz123",
			args:      []string{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := SafeCommand(tt.command, tt.args...)

			if tt.wantError && err == nil {
				t.Error("SafeCommand() expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("SafeCommand() unexpected error: %v", err)
			}

			if !tt.wantError && cmd == nil {
				t.Error("SafeCommand() returned nil command")
			}

			if !tt.wantError && cmd != nil {
				// Verify the command path is absolute
				if !filepath.IsAbs(cmd.Path) {
					t.Errorf("SafeCommand() should set absolute path, got: %s", cmd.Path)
				}

				// Verify args match
				if len(cmd.Args) != len(tt.args)+1 { // +1 for command itself
					t.Errorf("SafeCommand() args length = %d, want %d", len(cmd.Args), len(tt.args)+1)
				}
			}
		})
	}
}
