package dashboard

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func newAppsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "Manage Nylas applications",
		Long:  `List and create Nylas applications via the Dashboard API.`,
	}

	cmd.AddCommand(newAppsListCmd())
	cmd.AddCommand(newAppsCreateCmd())

	return cmd
}

// appRow is a flat struct for table output.
type appRow struct {
	ApplicationID string `json:"application_id"`
	Region        string `json:"region"`
	Environment   string `json:"environment"`
	Name          string `json:"name"`
}

var appColumns = []ports.Column{
	{Header: "APPLICATION ID", Field: "ApplicationID"},
	{Header: "REGION", Field: "Region"},
	{Header: "ENVIRONMENT", Field: "Environment"},
	{Header: "NAME", Field: "Name"},
}

func newAppsListCmd() *cobra.Command {
	var region string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List applications",
		Long: `List all Nylas applications in your organization.

By default, queries both US and EU regions and merges results.
Use --region to filter to a specific region.`,
		Example: `  # List all applications
  nylas dashboard apps list

  # List only US applications
  nylas dashboard apps list --region us`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appSvc, err := createAppService()
			if err != nil {
				return wrapDashboardError(err)
			}

			orgPublicID, err := getActiveOrgID()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			apps, err := appSvc.ListApplications(ctx, orgPublicID, region)
			if err != nil {
				return wrapDashboardError(err)
			}

			if len(apps) == 0 {
				fmt.Println("No applications found.")
				fmt.Println("\nCreate one with: nylas dashboard apps create --name MyApp --region us")
				return nil
			}

			rows := toAppRows(apps)
			return common.WriteListWithColumns(cmd, rows, appColumns)
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "Filter by region (us or eu)")

	return cmd
}

func newAppsCreateCmd() *cobra.Command {
	var (
		name   string
		region string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new application",
		Long:  `Create a new Nylas application in the specified region.`,
		Example: `  # Create a US application
  nylas dashboard apps create --name "My App" --region us

  # Create an EU application
  nylas dashboard apps create --name "EU App" --region eu`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return dashboardError("application name is required", "Use --name to specify the application name")
			}
			if region == "" {
				return dashboardError("region is required", "Use --region us or --region eu")
			}
			if region != "us" && region != "eu" {
				return dashboardError("invalid region", "Use --region us or --region eu")
			}

			appSvc, err := createAppService()
			if err != nil {
				return wrapDashboardError(err)
			}

			orgPublicID, err := getActiveOrgID()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			app, err := appSvc.CreateApplication(ctx, orgPublicID, region, name)
			if err != nil {
				return wrapDashboardError(err)
			}

			_, _ = common.Green.Println("✓ Application created!")
			fmt.Printf("  Application ID: %s\n", app.ApplicationID)
			fmt.Printf("  Region:         %s\n", app.Region)
			if app.Environment != "" {
				fmt.Printf("  Environment:    %s\n", app.Environment)
			}

			_, _ = common.Yellow.Println("\n  Client Secret (shown once — save it now):")
			fmt.Printf("  %s\n", app.ClientSecret)

			fmt.Println("\nTo configure the CLI with this application:")
			fmt.Printf("  nylas auth config --api-key <your-api-key> --region %s\n", app.Region)

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Application name (required)")
	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (required: us or eu)")

	return cmd
}

// getActiveOrgID retrieves the active organization ID from the keyring.
func getActiveOrgID() (string, error) {
	_, secrets, err := createDPoPService()
	if err != nil {
		return "", err
	}

	orgID, err := secrets.Get(ports.KeyDashboardOrgPublicID)
	if err != nil || orgID == "" {
		return "", dashboardError(
			"no active organization",
			"Run 'nylas dashboard login' first",
		)
	}
	return orgID, nil
}

// toAppRows converts gateway applications to flat display rows.
func toAppRows(apps []domain.GatewayApplication) []appRow {
	rows := make([]appRow, len(apps))
	for i, app := range apps {
		name := ""
		if app.Branding != nil {
			name = app.Branding.Name
		}
		env := app.Environment
		if env == "" {
			env = "production"
		}
		rows[i] = appRow{
			ApplicationID: app.ApplicationID,
			Region:        app.Region,
			Environment:   env,
			Name:          name,
		}
	}
	return rows
}
