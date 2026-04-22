package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	clitestutil "github.com/nylas/cli/internal/cli/testutil"
)

func TestInstallForAssistantUsesAssistantSpecificSchema(t *testing.T) {
	t.Run("vscode uses servers and removes legacy entry", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "mcp.json")
		err := os.WriteFile(configPath, []byte(`{"mcpServers":{"nylas":{"command":"old","args":["mcp","serve"]}}}`), 0600)
		if err != nil {
			t.Fatalf("writing config: %v", err)
		}

		assistant := testAssistant("vscode", configPath)
		if err := installForAssistant(assistant, "/usr/local/bin/nylas"); err != nil {
			t.Fatalf("installForAssistant returned error: %v", err)
		}

		config := readMCPTestConfig(t, configPath)
		servers := getConfigSection(t, config, "servers")
		if _, ok := servers["nylas"]; !ok {
			t.Fatalf("expected nylas server in servers section, got %+v", servers)
		}
		if _, ok := config["mcpServers"]; ok {
			t.Fatalf("expected legacy mcpServers section to be removed, got %+v", config["mcpServers"])
		}
	})

	t.Run("cursor uses mcpServers", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "mcp.json")
		assistant := testAssistant("cursor", configPath)

		if err := installForAssistant(assistant, "/usr/local/bin/nylas"); err != nil {
			t.Fatalf("installForAssistant returned error: %v", err)
		}

		config := readMCPTestConfig(t, configPath)
		mcpServers := getConfigSection(t, config, "mcpServers")
		if _, ok := mcpServers["nylas"]; !ok {
			t.Fatalf("expected nylas server in mcpServers section, got %+v", mcpServers)
		}
	})
}

func TestUninstallFromAssistantRemovesConfiguredServer(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "mcp.json")
	err := os.WriteFile(configPath, []byte(`{
  "servers":{"nylas":{"command":"new","args":["mcp","serve"]}},
  "mcpServers":{"nylas":{"command":"old","args":["mcp","serve"]}}
}`), 0600)
	if err != nil {
		t.Fatalf("writing config: %v", err)
	}

	assistant := testAssistant("vscode", configPath)
	if err := uninstallFromAssistant(assistant); err != nil {
		t.Fatalf("uninstallFromAssistant returned error: %v", err)
	}

	config := readMCPTestConfig(t, configPath)
	if _, ok := config["servers"]; ok {
		t.Fatalf("expected servers section removed, got %+v", config["servers"])
	}
	if _, ok := config["mcpServers"]; ok {
		t.Fatalf("expected mcpServers section removed, got %+v", config["mcpServers"])
	}
}

func TestInspectAssistantConfigAcceptsLegacyVSCodeSchema(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "mcp.json")
	err := os.WriteFile(configPath, []byte(`{"mcpServers":{"nylas":{"command":"legacy","args":["mcp","serve"]}}}`), 0600)
	if err != nil {
		t.Fatalf("writing config: %v", err)
	}

	assistant := testAssistant("vscode", configPath)
	state := inspectAssistantConfig(assistant, configPath)

	if !state.ConfigFileExists || !state.HasNylas {
		t.Fatalf("unexpected config state: %+v", state)
	}
	if state.SchemaKey != "mcpServers" {
		t.Fatalf("state.SchemaKey = %q, want %q", state.SchemaKey, "mcpServers")
	}
	if state.BinaryPath != "legacy" {
		t.Fatalf("state.BinaryPath = %q, want %q", state.BinaryPath, "legacy")
	}
}

func TestStatusCommandStructuredOutput(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "mcp.json")
	assistant := testAssistant("vscode", configPath)

	config := map[string]any{}
	setAssistantServer(config, assistant, nylasServerName, map[string]any{
		"command": "/usr/local/bin/nylas",
		"args":    []string{"mcp", "serve"},
	})
	if err := writeConfig(configPath, config); err != nil {
		t.Fatalf("writeConfig returned error: %v", err)
	}

	restore := swapAssistantsForTest(t, []Assistant{assistant})
	defer restore()

	root := newMCPStatusTestRoot()
	stdout, stderr, err := clitestutil.ExecuteCommand(root, "status", "--json")
	if err != nil {
		t.Fatalf("status --json returned error: %v\nstderr: %s", err, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("status --json wrote stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, "MCP Installation Status") {
		t.Fatalf("status --json output contained human-readable prose: %q", stdout)
	}

	var statuses []assistantStatus
	if err := json.Unmarshal([]byte(stdout), &statuses); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, stdout)
	}
	if len(statuses) != 1 {
		t.Fatalf("len(statuses) = %d, want 1", len(statuses))
	}
	if statuses[0].Status != "configured" {
		t.Fatalf("statuses[0].Status = %q, want %q", statuses[0].Status, "configured")
	}
	if statuses[0].SchemaKey != "servers" {
		t.Fatalf("statuses[0].SchemaKey = %q, want %q", statuses[0].SchemaKey, "servers")
	}
}

func TestStatusCommandDefaultOutputRemainsHumanReadable(t *testing.T) {
	restore := swapAssistantsForTest(t, []Assistant{testAssistant("cursor", filepath.Join(t.TempDir(), "cursor.json"))})
	defer restore()

	root := newMCPStatusTestRoot()
	stdout, _, err := clitestutil.ExecuteCommand(root, "status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if !strings.Contains(stdout, "MCP Installation Status") {
		t.Fatalf("default output missing header: %q", stdout)
	}
	if !strings.Contains(stdout, "Not available") {
		t.Fatalf("default output missing legend: %q", stdout)
	}
}

func newMCPStatusTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "test",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	common.AddOutputFlags(root)
	root.AddCommand(newStatusCmd())
	return root
}

func swapAssistantsForTest(t *testing.T, assistants []Assistant) func() {
	t.Helper()

	original := Assistants
	Assistants = assistants

	return func() {
		Assistants = original
	}
}

func testAssistant(id, configPath string) Assistant {
	return Assistant{
		Name:        "Test Assistant",
		ID:          id,
		ConfigPaths: map[string]string{runtime.GOOS: configPath},
		AppPaths:    map[string]string{runtime.GOOS: ""},
	}
}

func readMCPTestConfig(t *testing.T, configPath string) map[string]any {
	t.Helper()

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}
	return config
}

func getConfigSection(t *testing.T, config map[string]any, key string) map[string]any {
	t.Helper()

	section, ok := config[key].(map[string]any)
	if !ok {
		t.Fatalf("expected %s section in %+v", key, config)
	}
	return section
}
