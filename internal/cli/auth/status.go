package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			grantSvc, configSvc, err := createGrantService()
			if err != nil {
				return err
			}

			status, err := configSvc.GetStatus()
			if err != nil {
				return err
			}

			// Get current grant info
			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get grant count and default grant from grant service
			if grants, err := grantSvc.ListGrants(ctx); err == nil {
				status.GrantCount = len(grants)
			}
			if defaultID, err := grantSvc.GetDefaultGrantID(); err == nil {
				status.DefaultGrant = defaultID
			}

			var grantInfo struct {
				ID       string `json:"id"`
				Email    string `json:"email"`
				Provider string `json:"provider"`
				Status   string `json:"status"`
			}

			grant, err := grantSvc.GetCurrentGrant(ctx)
			if err == nil {
				grantInfo.ID = grant.ID
				grantInfo.Email = grant.Email
				grantInfo.Provider = string(grant.Provider)
				grantInfo.Status = grant.Status
			}

			jsonOutput, _ := cmd.Root().PersistentFlags().GetBool("json")
			if jsonOutput {
				output := map[string]any{
					"configured":    status.IsConfigured,
					"region":        status.Region,
					"config_path":   status.ConfigPath,
					"secret_store":  status.SecretStore,
					"grant_count":   status.GrantCount,
					"default_grant": status.DefaultGrant,
				}
				if grantInfo.ID != "" {
					output["grant"] = grantInfo
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(output)
			}

			_, _ = common.Bold.Println("Authentication Status")
			fmt.Println()

			if grantInfo.ID != "" {
				_, _ = common.Bold.Println("Current Account:")
				fmt.Printf("  Email: %s\n", grantInfo.Email)
				fmt.Printf("  Provider: %s\n", grantInfo.Provider)
				fmt.Printf("  Grant ID: %s\n", grantInfo.ID)
				if grantInfo.Status == "valid" {
					_, _ = common.Green.Printf("  Status: ✓ Valid\n")
				} else {
					_, _ = common.Yellow.Printf("  Status: %s\n", grantInfo.Status)
				}
				fmt.Println()
			}

			_, _ = common.Bold.Println("Configuration:")
			fmt.Printf("  Region: %s\n", status.Region)
			fmt.Printf("  Config Path: %s\n", status.ConfigPath)
			fmt.Printf("  Secret Store: %s\n", status.SecretStore)

			return nil
		},
	}
}
