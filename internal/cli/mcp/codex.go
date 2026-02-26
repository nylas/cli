package mcp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const nylasMCPServerName = "nylas"

var runCommand = func(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

type codexServerConfig struct {
	Name      string `json:"name"`
	Transport struct {
		Command string `json:"command"`
	} `json:"transport"`
}

func getCodexNylasConfig() (bool, string) {
	out, err := runCommand("codex", "mcp", "get", nylasMCPServerName, "--json")
	if err != nil {
		return false, ""
	}

	var cfg codexServerConfig
	if err := json.Unmarshal(out, &cfg); err != nil {
		return false, ""
	}
	if cfg.Name != nylasMCPServerName || cfg.Transport.Command == "" {
		return false, ""
	}

	return true, cfg.Transport.Command
}

func installForCodex(binaryPath string) error {
	_, _ = runCommand("codex", "mcp", "remove", nylasMCPServerName)

	out, err := runCommand("codex", "mcp", "add", nylasMCPServerName, "--", binaryPath, "mcp", "serve")
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("codex mcp add failed: %s", msg)
		}
		return fmt.Errorf("codex mcp add failed: %w", err)
	}

	return nil
}

func uninstallFromCodex() error {
	out, err := runCommand("codex", "mcp", "remove", nylasMCPServerName)
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("codex mcp remove failed: %s", msg)
		}
		return fmt.Errorf("codex mcp remove failed: %w", err)
	}
	return nil
}
