package mcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/mcp"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start the MCP server to enable AI assistants to interact with Nylas.

This command acts as a proxy to the official Nylas MCP server, providing
access to all Nylas email, calendar, and contacts tools through the
Model Context Protocol.

The proxy dynamically discovers available tools from the upstream Nylas
MCP server and adds local enhancements:
  - Automatic grant_id injection (no need to specify which account)
  - Local grant lookup (get_grant without email)
  - Timezone-aware timestamp display
  - Secure credential handling via system keyring

The server communicates via STDIO (standard input/output) and requires
Nylas credentials to be configured via 'nylas auth login'.

For more information: https://developer.nylas.com/docs/dev-guide/mcp/`,
		RunE: runServe,
	}

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	// Get API key from credentials
	apiKey, err := common.GetAPIKey()
	if err != nil {
		return fmt.Errorf("failed to get API key: %w\n\nPlease run 'nylas auth login' first", err)
	}

	// Get region from config (defaults to "us")
	region := "us"
	configStore := config.NewDefaultFileStore()
	if cfg, err := configStore.Load(); err == nil && cfg.Region != "" {
		region = cfg.Region
	}

	// Get default grant ID (optional - helps Claude know which account to use)
	grantID, _ := common.GetGrantID(nil)

	// Create MCP proxy with region
	proxy := mcp.NewProxy(apiKey, region)
	if grantID != "" {
		proxy.SetDefaultGrant(grantID)
	}

	// Set up grant store for local grant lookups (allows get_grant without email).
	if grantStore, err := common.NewDefaultGrantStore(); err == nil {
		proxy.SetGrantStore(grantStore)
	}

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Run the proxy (blocks until context is cancelled or error)
	return proxy.Run(ctx)
}
