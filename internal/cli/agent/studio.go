package agent

import (
	"fmt"

	browserpkg "github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/studio"
	"github.com/spf13/cobra"
)

func newStudioCmd() *cobra.Command {
	var (
		port      int
		noBrowser bool
	)

	cmd := &cobra.Command{
		Use:   "studio",
		Short: "Launch Agent Studio, the visual agent management UI",
		Long: `Launch Agent Studio, a local web UI for managing agent resources.

The board shows every workspace with its attached policy, rules, and member
accounts. Drag policies, rules, and accounts between workspaces; create and
edit policies, rules, lists, and accounts without leaving the page.

The server runs on localhost only.

API reference: https://developer.nylas.com/docs/v3/agent-accounts/

Examples:
  nylas agent studio
  nylas agent studio --port 8080
  nylas agent studio --no-browser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			addr := fmt.Sprintf("localhost:%d", port)
			url := fmt.Sprintf("http://%s", addr)

			fmt.Printf("Starting Agent Studio at %s\n", url)
			fmt.Println("Press Ctrl+C to stop")

			if !noBrowser {
				b := browserpkg.NewDefaultBrowser()
				if err := b.Open(url); err != nil {
					fmt.Printf("Could not open browser: %v\n", err)
					fmt.Printf("Please open %s manually\n", url)
				}
			}

			server := studio.NewServer(addr, client)
			return server.Start(cmd.Context())
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 7368, "Port to run the server on")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")

	return cmd
}
