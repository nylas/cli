package mcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/mcp"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start the MCP server to enable AI assistants to interact with Nylas.

This command acts as a proxy to the official Nylas MCP server, providing
access to all Nylas email and calendar tools through the Model Context Protocol.

The server communicates via STDIO (standard input/output) and requires
Nylas credentials to be configured via 'nylas auth login'.

Available tools (from Nylas MCP):

  Messages:
    list_messages     - Search and retrieve emails
    list_threads      - List email threads
    create_draft      - Create a new draft
    update_draft      - Update an existing draft
    send_message      - Send a new email (requires confirmation)
    send_draft        - Send a draft (requires confirmation)

  Calendar:
    list_calendars    - List all calendars
    list_events       - List calendar events
    create_event      - Create a new event
    update_event      - Update an existing event
    availability      - Check availability

  Utilities:
    get_grant         - Get grant information
    get_folder_by_id  - Get folder details
    current_time      - Get current time
    epoch_to_datetime - Convert epoch to datetime

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

	// Set up grant store for local grant lookups (allows get_grant without email)
	// Try multiple secret store backends to ensure we can access grants
	var secretStore ports.SecretStore
	secretStore, err = keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		// Fallback to encrypted file store if keyring fails
		// This can happen when the MCP server runs in a sandboxed context
		secretStore, err = keyring.NewEncryptedFileStore(config.DefaultConfigDir())
	}
	if err == nil && secretStore != nil {
		grantStore := keyring.NewGrantStore(secretStore)
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
