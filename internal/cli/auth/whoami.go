package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current user info",
		RunE: func(cmd *cobra.Command, args []string) error {
			grantSvc, _, err := createGrantService()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			grant, err := grantSvc.GetCurrentGrant(ctx)
			if err != nil {
				return err
			}

			jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")
			if jsonOutput {
				output := map[string]string{
					"email":    grant.Email,
					"provider": string(grant.Provider),
					"grant_id": grant.ID,
					"status":   grant.Status,
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(output)
			}

			fmt.Println(grant.Email)
			fmt.Printf("Provider: %s\n", grant.Provider.DisplayName())
			fmt.Printf("Grant ID: %s\n", grant.ID)

			return nil
		},
	}
}
