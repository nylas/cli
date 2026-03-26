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
	cmd.AddCommand(newAppsUseCmd())
	cmd.AddCommand(newAPIKeysCmd())

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

			if app.ClientSecret != "" {
				_, _ = common.Yellow.Println("\n  Client Secret (available once — save it now):")
				if err := handleSecretDelivery(app.ClientSecret, "Client Secret"); err != nil {
					return err
				}
			}

			fmt.Println("\nTo configure the CLI with this application:")
			fmt.Printf("  nylas auth config --api-key <your-api-key> --region %s\n", app.Region)

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Application name (required)")
	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (required: us or eu)")

	return cmd
}

func newAppsUseCmd() *cobra.Command {
	var region string

	cmd := &cobra.Command{
		Use:   "use [application-id]",
		Short: "Set the active application for subsequent commands",
		Long: `Set an application as active so you don't need to pass --app and --region
to every apikeys command.

When called without arguments, lists your applications and lets you pick one interactively.`,
		Example: `  # Interactive — choose from your apps
  nylas dashboard apps use

  # Set active app directly
  nylas dashboard apps use b09141da-ead2-46bd-8f4c-c9ec5af4c6cc --region us

  # Now apikeys commands use the active app automatically
  nylas dashboard apps apikeys list
  nylas dashboard apps apikeys create --name "My key"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID := ""
			if len(args) > 0 {
				appID = args[0]
			}

			// If no app ID provided, show interactive selector
			if appID == "" {
				selectedID, selectedRegion, err := selectApp(region)
				if err != nil {
					return wrapDashboardError(err)
				}
				appID = selectedID
				region = selectedRegion
			}

			if region == "" {
				return dashboardError("region is required", "Use --region us or --region eu")
			}

			_, secrets, err := createDPoPService()
			if err != nil {
				return wrapDashboardError(err)
			}

			if err := secrets.Set(ports.KeyDashboardAppID, appID); err != nil {
				return wrapDashboardError(err)
			}
			if err := secrets.Set(ports.KeyDashboardAppRegion, region); err != nil {
				return wrapDashboardError(err)
			}

			_, _ = common.Green.Printf("✓ Active app: %s (%s)\n", appID, region)
			return nil
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "Region of the application (us or eu)")

	return cmd
}

// selectApp fetches apps and presents an interactive selector.
// Returns the selected app ID and region.
func selectApp(regionFilter string) (appID, region string, err error) {
	appSvc, err := createAppService()
	if err != nil {
		return "", "", err
	}

	orgPublicID, err := getActiveOrgID()
	if err != nil {
		return "", "", err
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	var apps []domain.GatewayApplication
	err = common.RunWithSpinner("Loading applications...", func() error {
		apps, err = appSvc.ListApplications(ctx, orgPublicID, regionFilter)
		return err
	})
	if err != nil {
		return "", "", err
	}

	if len(apps) == 0 {
		return "", "", dashboardError(
			"no applications found",
			"Create one with: nylas dashboard apps create --name MyApp --region us",
		)
	}

	opts := make([]common.SelectOption[int], len(apps))
	for i, app := range apps {
		name := ""
		if app.Branding != nil {
			name = app.Branding.Name
		}
		label := fmt.Sprintf("%s (%s)", app.ApplicationID, app.Region)
		if name != "" {
			label = fmt.Sprintf("%s — %s (%s)", name, app.ApplicationID, app.Region)
		}
		opts[i] = common.SelectOption[int]{Label: label, Value: i}
	}

	idx, err := common.Select("Select application", opts)
	if err != nil {
		return "", "", err
	}

	selected := apps[idx]
	return selected.ApplicationID, selected.Region, nil
}

// getActiveApp returns the active app ID and region from the keyring.
// Flags take priority over the stored active app.
func getActiveApp(appFlag, regionFlag string) (appID, region string, err error) {
	if appFlag != "" && regionFlag != "" {
		return appFlag, regionFlag, nil
	}

	_, secrets, sErr := createDPoPService()
	if sErr != nil {
		return appFlag, regionFlag, sErr
	}

	if appFlag == "" {
		appID, _ = secrets.Get(ports.KeyDashboardAppID)
	} else {
		appID = appFlag
	}
	if regionFlag == "" {
		region, _ = secrets.Get(ports.KeyDashboardAppRegion)
	} else {
		region = regionFlag
	}

	if appID == "" {
		return "", "", dashboardError(
			"no active application",
			"Run 'nylas dashboard apps use <app-id> --region <region>' or pass --app and --region",
		)
	}
	if region == "" {
		return "", "", dashboardError(
			"no region set for active application",
			"Run 'nylas dashboard apps use <app-id> --region <region>' or pass --region",
		)
	}
	return appID, region, nil
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
