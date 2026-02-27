package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewMCPCmd_HasSubcommands(t *testing.T) {
	cmd := NewMCPCmd()
	if cmd == nil {
		t.Fatal("NewMCPCmd returned nil")
	}
	if cmd.Use != "mcp" {
		t.Fatalf("Use = %q, want mcp", cmd.Use)
	}

	found := map[string]bool{}
	for _, c := range cmd.Commands() {
		found[c.Use] = true
	}
	for _, use := range []string{"serve", "install", "uninstall", "status"} {
		if !found[use] {
			t.Fatalf("missing subcommand %q", use)
		}
	}
}

func TestAssistant_IsInstalled(t *testing.T) {
	a := Assistant{
		Name: "Test",
		ID:   "test",
		AppPaths: map[string]string{
			runtime.GOOS: "",
		},
	}
	if !a.IsInstalled() {
		t.Fatal("assistant with empty app path should be considered installed")
	}

	a2 := Assistant{
		Name: "Test",
		ID:   "test2",
		AppPaths: map[string]string{
			runtime.GOOS: filepath.Join(t.TempDir(), "does-not-exist"),
		},
	}
	if a2.IsInstalled() {
		t.Fatal("assistant with missing app path should not be installed")
	}
}

func TestAssistant_IsConfigured(t *testing.T) {
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "mcp.json")
	content := map[string]any{
		"mcpServers": map[string]any{
			"nylas": map[string]any{
				"command": "/bin/nylas",
				"args":    []string{"mcp", "serve"},
			},
		},
	}
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(cfg, data, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	a := Assistant{
		Name: "Test",
		ID:   "test",
		ConfigPaths: map[string]string{
			runtime.GOOS: cfg,
		},
	}
	if !a.IsConfigured() {
		t.Fatal("expected configured assistant")
	}
}

func TestCheckNylasInConfig(t *testing.T) {
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "mcp.json")
	content := map[string]any{
		"mcpServers": map[string]any{
			"nylas": map[string]any{
				"command": "/usr/local/bin/nylas",
				"args":    []string{"mcp", "serve"},
			},
		},
	}
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(cfg, data, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	a := Assistant{ID: "test"}
	ok, bin := checkNylasInConfig(a, cfg)
	if !ok {
		t.Fatal("expected nylas config to be found")
	}
	if bin != "/usr/local/bin/nylas" {
		t.Fatalf("binary = %q, want /usr/local/bin/nylas", bin)
	}
}

func TestRunInstallAndUninstall_ClaudeCode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	bin := filepath.Join(home, "nylas")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	if err := runInstall("claude-code", bin, false); err != nil {
		t.Fatalf("runInstall: %v", err)
	}

	cfg := filepath.Join(home, ".claude.json")
	raw, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	mcpServers, ok := parsed["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers missing")
	}
	if _, ok := mcpServers["nylas"]; !ok {
		t.Fatal("nylas server missing")
	}

	if err := runUninstall("claude-code", false); err != nil {
		t.Fatalf("runUninstall: %v", err)
	}
	raw, err = os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read config after uninstall: %v", err)
	}
	parsed = map[string]any{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal config after uninstall: %v", err)
	}
	if ms, ok := parsed["mcpServers"].(map[string]any); ok {
		if _, has := ms["nylas"]; has {
			t.Fatal("nylas server should be removed after uninstall")
		}
	}
}

func TestRunInstall_UnknownAssistant(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bin := filepath.Join(home, "nylas")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	if err := runInstall("does-not-exist", bin, false); err == nil {
		t.Fatal("expected error for unknown assistant")
	}
}

func TestRunUninstall_RequiresTarget(t *testing.T) {
	if err := runUninstall("", false); err == nil {
		t.Fatal("expected error when neither --assistant nor --all is set")
	}
}

func TestInstallClaudeCodePermissions_Idempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := installClaudeCodePermissions(); err != nil {
		t.Fatalf("first installClaudeCodePermissions: %v", err)
	}
	if err := installClaudeCodePermissions(); err != nil {
		t.Fatalf("second installClaudeCodePermissions: %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	perms, ok := parsed["permissions"].(map[string]any)
	if !ok {
		t.Fatal("permissions missing")
	}
	allow, ok := perms["allow"].([]any)
	if !ok || len(allow) == 0 {
		t.Fatal("permissions.allow missing")
	}
}

func TestDetectBinaryPath(t *testing.T) {
	path, err := detectBinaryPath()
	if err != nil {
		t.Fatalf("detectBinaryPath: %v", err)
	}
	if path == "" {
		t.Fatal("detectBinaryPath returned empty path")
	}
}

func TestRunStatus_NoError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := runStatus(); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
}

func TestRunServe_NotConfiguredReturnsError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NYLAS_API_KEY", "")
	t.Setenv("NYLAS_GRANT_ID", "")

	if err := runServe(nil, nil); err == nil {
		t.Fatal("expected runServe to fail when not configured")
	}
}
