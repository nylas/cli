package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		assistantID string
		binaryPath  string
		installAll  bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install MCP configuration for AI assistants",
		Long: `Configure MCP for AI assistants like Claude Desktop, Cursor, or Windsurf.

This command automatically adds the Nylas MCP server configuration to your
AI assistant's config file, enabling it to interact with your email and calendar.

Supported assistants:
  - claude-desktop  Claude Desktop app
  - claude-code     Claude Code (~/.claude.json)
  - cursor          Cursor IDE
  - windsurf        Windsurf IDE
  - vscode          VS Code (project-level .vscode/mcp.json)`,
		Example: `  # Interactive mode - prompts for assistant selection
  nylas mcp install

  # Install for specific assistant
  nylas mcp install --assistant claude-desktop

  # Install for all detected assistants
  nylas mcp install --all

  # Specify custom binary path
  nylas mcp install --assistant cursor --binary /usr/local/bin/nylas`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(assistantID, binaryPath, installAll)
		},
	}

	cmd.Flags().StringVarP(&assistantID, "assistant", "a", "", "Target assistant (claude-desktop, cursor, windsurf, vscode, claude-code)")
	cmd.Flags().StringVarP(&binaryPath, "binary", "b", "", "Path to nylas binary (default: auto-detect)")
	cmd.Flags().BoolVar(&installAll, "all", false, "Install for all detected assistants")

	return cmd
}

func runInstall(assistantID, binaryPath string, installAll bool) error {
	// Detect binary path if not provided
	if binaryPath == "" {
		var err error
		binaryPath, err = detectBinaryPath()
		if err != nil {
			return fmt.Errorf("could not detect nylas binary path: %w\n\nPlease specify with --binary flag", err)
		}
	}

	// Validate binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("binary not found at %s: %w", binaryPath, err)
	}

	var assistants []Assistant

	if installAll {
		// Install for all assistants
		assistants = Assistants
	} else if assistantID != "" {
		// Install for specific assistant
		a := GetAssistantByID(assistantID)
		if a == nil {
			return fmt.Errorf("unknown assistant: %s\n\nSupported: claude-desktop, claude-code, cursor, windsurf, vscode", assistantID)
		}
		assistants = []Assistant{*a}
	} else {
		// Interactive mode
		a, err := selectAssistant()
		if err != nil {
			return err
		}
		if a == nil {
			return nil // User cancelled
		}
		assistants = []Assistant{*a}
	}

	// Install for each assistant
	successCount := 0

	for _, a := range assistants {
		configPath := a.GetConfigPath()
		if configPath == "" {
			_, _ = common.Yellow.Printf("  ! %s: unsupported on this platform\n", a.Name)
			continue
		}

		// Check if app is installed (for non-project configs)
		if !a.IsProjectConfig() && !a.IsInstalled() {
			_, _ = common.Yellow.Printf("  ! %s: application not installed\n", a.Name)
			continue
		}

		err := installForAssistant(a, binaryPath)
		if err != nil {
			_, _ = common.Yellow.Printf("  ! %s: %v\n", a.Name, err)
			continue
		}

		_, _ = common.Green.Printf("  ✓ %s: configured at %s\n", a.Name, configPath)
		if a.ID == "claude-code" {
			_, _ = common.Green.Printf("  ✓ %s: permissions added to ~/.claude/settings.json\n", a.Name)
		}
		successCount++
	}

	if successCount > 0 {
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Restart your AI assistant to load the new configuration")
		fmt.Println("  2. The Nylas tools will be available for email and calendar access")
	}

	return nil
}

func selectAssistant() (*Assistant, error) {
	fmt.Println("Select an AI assistant to configure:")
	fmt.Println()

	available := make([]Assistant, 0, len(Assistants))
	for _, a := range Assistants {
		status := ""
		if a.IsProjectConfig() {
			status = " (project-level)"
		} else if !a.IsInstalled() {
			status = " (not installed)"
		} else if a.IsConfigured() {
			status = " (already configured)"
		}
		available = append(available, a)
		fmt.Printf("  %d. %s%s\n", len(available), a.Name, status)
	}

	fmt.Println()
	fmt.Print("Enter number (or 'q' to quit): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "q" || input == "" {
		return nil, nil
	}

	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(available) {
		return nil, common.NewInputError(fmt.Sprintf("invalid selection: %s", input))
	}

	return &available[num-1], nil
}

func installForAssistant(a Assistant, binaryPath string) error {
	configPath := a.GetConfigPath()

	// Ensure parent directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Read existing config or create new
	config := make(map[string]any)
	// #nosec G304 -- configPath from Assistant.GetConfigPath() returns validated AI assistant config paths
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing existing config: %w", err)
		}
	}

	// Get or create mcpServers section
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		mcpServers = make(map[string]any)
	}

	// Add nylas server config
	mcpServers["nylas"] = map[string]any{
		"command": binaryPath,
		"args":    []string{"mcp", "serve"},
	}

	config["mcpServers"] = mcpServers

	// Write config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// For Claude Code, also configure permissions
	if a.ID == "claude-code" {
		if err := installClaudeCodePermissions(); err != nil {
			// Non-fatal: permissions can be granted interactively
			return fmt.Errorf("configured MCP server, but failed to set permissions: %w\n  You may need to grant permissions interactively", err)
		}
	}

	return nil
}

// installClaudeCodePermissions adds Nylas MCP tool permissions to Claude Code settings.
// This allows Claude Code to use Nylas tools without prompting for each one.
func installClaudeCodePermissions() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	// Claude Code user-level settings
	settingsDir := filepath.Join(home, ".claude")
	settingsPath := filepath.Join(settingsDir, "settings.json")

	// Ensure directory exists
	if err := os.MkdirAll(settingsDir, 0750); err != nil {
		return fmt.Errorf("creating settings directory: %w", err)
	}

	// Read existing settings or create new
	settings := make(map[string]any)
	// #nosec G304 -- settingsPath is constructed from home directory + ".claude/settings.json"
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parsing existing settings: %w", err)
		}
	}

	// Get or create permissions section
	permissions, ok := settings["permissions"].(map[string]any)
	if !ok {
		permissions = make(map[string]any)
	}

	// Get or create allow list
	var allowList []string
	if existing, ok := permissions["allow"].([]any); ok {
		for _, item := range existing {
			if s, ok := item.(string); ok {
				allowList = append(allowList, s)
			}
		}
	}

	// Nylas MCP tools to allow
	nylasPermission := "mcp__nylas__*"

	// Add if not already present
	if !slices.Contains(allowList, nylasPermission) {
		allowList = append(allowList, nylasPermission)
		permissions["allow"] = allowList
		settings["permissions"] = permissions

		// Write settings
		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding settings: %w", err)
		}

		if err := os.WriteFile(settingsPath, data, 0600); err != nil {
			return fmt.Errorf("writing settings: %w", err)
		}
	}

	return nil
}

func detectBinaryPath() (string, error) {
	// Try os.Executable() first
	exePath, err := os.Executable()
	if err == nil {
		// Resolve symlinks
		resolved, err := filepath.EvalSymlinks(exePath)
		if err == nil {
			return resolved, nil
		}
		return exePath, nil
	}

	// Fallback: try to find nylas in PATH
	path, err := exec.LookPath("nylas")
	if err == nil {
		return path, nil
	}

	// Fallback: common installation paths
	candidates := []string{
		"/usr/local/bin/nylas",
		"/opt/homebrew/bin/nylas",
		filepath.Join(os.Getenv("HOME"), ".local/bin/nylas"),
		filepath.Join(os.Getenv("HOME"), "go/bin/nylas"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf("could not find nylas binary")
}
