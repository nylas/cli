package rpc

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/rpcserver"
	"github.com/nylas/cli/internal/cli/common"
)

func newTokenCmd() *cobra.Command {
	var copyToClipboard bool

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Show or copy the RPC session token",
		Long: "Print the JSON-RPC WebSocket session token used to authenticate against " +
			"'nylas rpc serve'. Resolves the same way the server does: NYLAS_WS_TOKEN if set, " +
			"otherwise the keyring, generating and persisting one if none exists.",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := keyring.NewSecretStore(config.DefaultConfigDir())
			if err != nil {
				return fmt.Errorf("open secret store: %w", err)
			}
			token, err := rpcserver.ResolveToken(store, os.Getenv)
			if err != nil {
				return err
			}

			if jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json"); jsonOutput {
				return common.PrintJSON(map[string]string{"token": token})
			}

			if copyToClipboard {
				if err := common.CopyToClipboard(token); err != nil {
					return common.WrapWriteError("clipboard", err)
				}
				_, _ = common.Green.Println("✓ RPC token copied to clipboard")
				return nil
			}

			fmt.Println(token)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&copyToClipboard, "copy", "c", false, "Copy to clipboard")

	return cmd
}
