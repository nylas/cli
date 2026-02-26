package mcp

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	mcpserver "github.com/nylas/cli/internal/adapters/mcp"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start a native MCP server that calls the Nylas API directly.

The server communicates via STDIO (standard input/output) and requires
Nylas credentials to be configured via 'nylas auth login'.

Available tools (37 total):

  Email:        list/get/send/update/delete messages, smart compose
  Drafts:       list/get/create/update/send drafts
  Threads:      list/get threads
  Folders:      list/get/create folders
  Attachments:  list/get attachment metadata
  Scheduled:    list/cancel scheduled messages
  Calendar:     list/get/create calendars
  Events:       list/get/create/update/delete events
  Availability: free/busy check, find available slots
  Contacts:     list/get/create contacts
  Utilities:    current_time, epoch_to_datetime, datetime_to_epoch

For more information: https://developer.nylas.com/docs/dev-guide/mcp/`,
		RunE: runServe,
	}

	return cmd
}

func runServe(_ *cobra.Command, _ []string) error {
	client, err := common.GetNylasClient()
	if err != nil {
		return err
	}

	grantID, _ := common.GetGrantID(nil)

	server := mcpserver.NewServer(client, grantID)

	// Set up grant store for local grant lookups
	var secretStore ports.SecretStore
	secretStore, err = keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		secretStore, err = keyring.NewEncryptedFileStore(config.DefaultConfigDir())
	}
	if err == nil && secretStore != nil {
		grantStore := keyring.NewGrantStore(secretStore)
		server.SetGrantStore(grantStore)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	return server.Run(ctx)
}
