package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Assistant represents an AI assistant that supports MCP.
type Assistant struct {
	Name        string            // Display name
	ID          string            // Identifier (claude-desktop, cursor, etc.)
	ConfigPaths map[string]string // OS -> config path template
	AppPaths    map[string]string // OS -> app path (to check if installed)
}

// Assistants is the list of supported AI assistants.
var Assistants = []Assistant{
	{
		Name: "Claude Desktop",
		ID:   "claude-desktop",
		ConfigPaths: map[string]string{
			"darwin":  "~/Library/Application Support/Claude/claude_desktop_config.json",
			"linux":   "~/.config/Claude/claude_desktop_config.json",
			"windows": "%APPDATA%\\Claude\\claude_desktop_config.json",
		},
		AppPaths: map[string]string{
			"darwin":  "/Applications/Claude.app",
			"linux":   "",
			"windows": "",
		},
	},
	{
		Name: "Claude Code",
		ID:   "claude-code",
		ConfigPaths: map[string]string{
			"darwin":  "~/.claude.json",
			"linux":   "~/.claude.json",
			"windows": "%USERPROFILE%\\.claude.json",
		},
		AppPaths: map[string]string{
			"darwin":  "",
			"linux":   "",
			"windows": "",
		},
	},
	{
		Name: "Cursor",
		ID:   "cursor",
		ConfigPaths: map[string]string{
			"darwin":  "~/.cursor/mcp.json",
			"linux":   "~/.cursor/mcp.json",
			"windows": "%USERPROFILE%\\.cursor\\mcp.json",
		},
		AppPaths: map[string]string{
			"darwin":  "/Applications/Cursor.app",
			"linux":   "",
			"windows": "",
		},
	},
	{
		Name: "Windsurf",
		ID:   "windsurf",
		ConfigPaths: map[string]string{
			"darwin":  "~/.codeium/windsurf/mcp_config.json",
			"linux":   "~/.codeium/windsurf/mcp_config.json",
			"windows": "%USERPROFILE%\\.codeium\\windsurf\\mcp_config.json",
		},
		AppPaths: map[string]string{
			"darwin":  "/Applications/Windsurf.app",
			"linux":   "",
			"windows": "",
		},
	},
	{
		Name: "VS Code",
		ID:   "vscode",
		ConfigPaths: map[string]string{
			"darwin":  ".vscode/mcp.json",
			"linux":   ".vscode/mcp.json",
			"windows": ".vscode\\mcp.json",
		},
		AppPaths: map[string]string{
			"darwin":  "/Applications/Visual Studio Code.app",
			"linux":   "",
			"windows": "",
		},
	},
}

// GetConfigPath returns the config path for the current OS.
func (a Assistant) GetConfigPath() string {
	path, ok := a.ConfigPaths[runtime.GOOS]
	if !ok {
		return ""
	}
	return expandPath(path)
}

// IsProjectConfig returns true if the config is project-specific (not global).
func (a Assistant) IsProjectConfig() bool {
	return a.ID == "vscode"
}

// IsInstalled checks if the AI assistant application is installed.
func (a Assistant) IsInstalled() bool {
	appPath, ok := a.AppPaths[runtime.GOOS]
	if !ok || appPath == "" {
		// Can't determine installation status
		return true
	}
	_, err := os.Stat(appPath)
	return err == nil
}

// IsConfigured checks if Nylas MCP is already configured for this assistant.
func (a Assistant) IsConfigured() bool {
	configPath := a.GetConfigPath()
	if configPath == "" {
		return false
	}

	// #nosec G304 -- configPath from GetConfigPath() returns validated assistant config paths
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}

	// Check if mcpServers.nylas exists
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		return false
	}

	_, hasNylas := mcpServers["nylas"]
	return hasNylas
}

// GetAssistantByID returns an assistant by ID.
func GetAssistantByID(id string) *Assistant {
	for i := range Assistants {
		if Assistants[i].ID == id {
			return &Assistants[i]
		}
	}
	return nil
}

// expandPath expands ~ and environment variables in a path.
func expandPath(path string) string {
	// Expand ~
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path
}
