package cli

import (
	"os"
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

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
