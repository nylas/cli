// Package mcp provides MCP (Model Context Protocol) server functionality for AI integration.
package mcp

import (
	"github.com/spf13/cobra"
)

// NewMCPCmd creates the mcp command with all subcommands.
func NewMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP (Model Context Protocol) server for AI integration",
		Long: `Start an MCP server to enable AI assistants like Claude to interact with
your Nylas email, calendar, and contacts.

This command proxies requests to the official Nylas MCP server, providing
seamless access to all Nylas tools through your locally configured credentials.

Example configuration for Claude Desktop (~/Library/Application Support/Claude/claude_desktop_config.json):

  {
    "mcpServers": {
      "nylas": {
        "command": "nylas",
        "args": ["mcp", "serve"]
      }
    }
  }

Example configuration for Claude Code (.mcp.json):

  {
    "mcpServers": {
      "nylas": {
        "command": "nylas",
        "args": ["mcp", "serve"]
      }
    }
  }

For more information about Nylas MCP: https://developer.nylas.com/docs/dev-guide/mcp/`,
	}

	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}
