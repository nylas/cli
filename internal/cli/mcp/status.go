package mcp

import (
	"fmt"
	"io"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

type assistantStatus struct {
	Name             string `json:"name" yaml:"name"`
	ID               string `json:"id" yaml:"id"`
	Status           string `json:"status" yaml:"status"`
	ConfigPath       string `json:"config_path,omitempty" yaml:"config_path,omitempty"`
	ProjectConfig    bool   `json:"project_config" yaml:"project_config"`
	Supported        bool   `json:"supported" yaml:"supported"`
	Installed        bool   `json:"installed" yaml:"installed"`
	ConfigFileExists bool   `json:"config_file_exists" yaml:"config_file_exists"`
	Configured       bool   `json:"configured" yaml:"configured"`
	BinaryPath       string `json:"binary_path,omitempty" yaml:"binary_path,omitempty"`
	SchemaKey        string `json:"schema_key,omitempty" yaml:"schema_key,omitempty"`
}

func (s assistantStatus) QuietField() string {
	return s.ID
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show MCP installation status",
		Long: `Show the MCP configuration status for all supported AI assistants.

This command checks which AI assistants have Nylas MCP configured and
displays the configuration path for each.`,
		Example: `  nylas mcp status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd)
		},
	}

	return cmd
}

func runStatus(cmd *cobra.Command) error {
	statuses := collectAssistantStatuses(Assistants)
	if common.IsStructuredOutput(cmd) {
		return common.GetOutputWriter(cmd).WriteList(statuses, nil)
	}

	renderHumanStatus(cmd.OutOrStdout(), statuses)
	return nil
}

func collectAssistantStatuses(assistants []Assistant) []assistantStatus {
	statuses := make([]assistantStatus, 0, len(assistants))

	for _, a := range assistants {
		statuses = append(statuses, getAssistantStatus(a))
	}

	return statuses
}

func getAssistantStatus(a Assistant) assistantStatus {
	configPath := a.GetConfigPath()
	status := assistantStatus{
		Name:          a.Name,
		ID:            a.ID,
		Status:        "unsupported",
		ConfigPath:    configPath,
		ProjectConfig: a.IsProjectConfig(),
	}

	if configPath == "" {
		return status
	}

	status.Supported = true
	status.Installed = true

	if !a.IsProjectConfig() && !a.IsInstalled() {
		status.Status = "not_installed"
		status.Installed = false
		return status
	}

	configState := inspectAssistantConfig(a, configPath)
	status.ConfigFileExists = configState.ConfigFileExists
	status.Configured = configState.HasNylas
	status.BinaryPath = configState.BinaryPath
	status.SchemaKey = configState.SchemaKey

	if configState.HasNylas {
		status.Status = "configured"
		return status
	}

	status.Status = "not_configured"
	return status
}

func renderHumanStatus(w io.Writer, statuses []assistantStatus) {
	_, _ = fmt.Fprintln(w, "MCP Installation Status:")
	_, _ = fmt.Fprintln(w)

	for _, status := range statuses {
		switch status.Status {
		case "unsupported":
			_, _ = fmt.Fprintf(w, "  - %-16s  unsupported on this platform\n", status.Name)
		case "not_installed":
			_, _ = fmt.Fprintf(w, "  - %-16s  application not installed\n", status.Name)
		case "configured":
			_, _ = fmt.Fprintf(w, "  ✓ %-16s  configured  %s\n", status.Name, status.ConfigPath)
			if status.BinaryPath != "" {
				_, _ = fmt.Fprintf(w, "                       binary: %s\n", status.BinaryPath)
			}
		default:
			message := "not configured"
			if status.ConfigFileExists {
				message = "config exists, nylas not added"
			}
			_, _ = fmt.Fprintf(w, "  ○ %-16s  %s  %s\n", status.Name, message, status.ConfigPath)
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Legend:")
	_, _ = fmt.Fprintln(w, "  ✓ Nylas MCP configured")
	_, _ = fmt.Fprintln(w, "  ○ Available but not configured")
	_, _ = fmt.Fprintln(w, "  - Not available")
}
