// Package rpc provides JSON-RPC server commands.
package rpc

import "github.com/spf13/cobra"

// NewRPCCmd creates the rpc command with all subcommands.
func NewRPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rpc",
		Short: "JSON-RPC WebSocket server for Nylas",
	}

	cmd.AddCommand(newServeCmd())

	return cmd
}
