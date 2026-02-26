package mcp

import (
	"errors"
	"testing"
)

func TestGetCodexNylasConfig(t *testing.T) {
	orig := runCommand
	t.Cleanup(func() { runCommand = orig })

	tests := []struct {
		name       string
		out        []byte
		err        error
		wantOK     bool
		wantBinary string
	}{
		{
			name:       "returns command when codex config exists",
			out:        []byte(`{"name":"nylas","transport":{"type":"stdio","command":"/usr/local/bin/nylas","args":["mcp","serve"]}}`),
			wantOK:     true,
			wantBinary: "/usr/local/bin/nylas",
		},
		{
			name:   "returns not configured when output is non-json",
			out:    []byte("configured"),
			wantOK: false,
		},
		{
			name: "returns not configured on command error",
			err:  errors.New("not found"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			runCommand = func(_ string, _ ...string) ([]byte, error) {
				return tt.out, tt.err
			}

			gotOK, gotBinary := getCodexNylasConfig()
			if gotOK != tt.wantOK {
				t.Errorf("configured = %v, want %v", gotOK, tt.wantOK)
			}
			if gotBinary != tt.wantBinary {
				t.Errorf("binary = %q, want %q", gotBinary, tt.wantBinary)
			}
		})
	}
}

func TestInstallForCodex(t *testing.T) {
	orig := runCommand
	t.Cleanup(func() { runCommand = orig })

	t.Run("calls remove then add", func(t *testing.T) {
		var calls [][]string
		runCommand = func(name string, args ...string) ([]byte, error) {
			call := append([]string{name}, args...)
			calls = append(calls, call)
			return nil, nil
		}

		if err := installForCodex("/usr/local/bin/nylas"); err != nil {
			t.Fatalf("installForCodex() error = %v", err)
		}

		if len(calls) != 2 {
			t.Fatalf("expected 2 command calls, got %d", len(calls))
		}
	})

	t.Run("returns command output on add failure", func(t *testing.T) {
		callNum := 0
		runCommand = func(_ string, _ ...string) ([]byte, error) {
			callNum++
			if callNum == 2 {
				return []byte("add failed"), errors.New("exit 1")
			}
			return nil, nil
		}

		if err := installForCodex("/usr/local/bin/nylas"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestUninstallFromCodex(t *testing.T) {
	orig := runCommand
	t.Cleanup(func() { runCommand = orig })

	t.Run("success", func(t *testing.T) {
		runCommand = func(_ string, _ ...string) ([]byte, error) {
			return nil, nil
		}

		if err := uninstallFromCodex(); err != nil {
			t.Fatalf("uninstallFromCodex() error = %v", err)
		}
	})

	t.Run("failure includes command output", func(t *testing.T) {
		runCommand = func(_ string, _ ...string) ([]byte, error) {
			return []byte("remove failed"), errors.New("exit 1")
		}

		if err := uninstallFromCodex(); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
