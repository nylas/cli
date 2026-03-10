package provider

import (
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/gcp"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

type googleSetupOpts struct {
	region    string
	projectID string
	email     bool
	calendar  bool
	contacts  bool
	pubsub    bool
	yes       bool
	fresh     bool
}

func (o *googleSetupOpts) hasFeatureFlags() bool {
	return o.email || o.calendar || o.contacts || o.pubsub
}

func (o *googleSetupOpts) selectedFeatures() []string {
	var features []string
	if o.email {
		features = append(features, domain.FeatureEmail)
	}
	if o.calendar {
		features = append(features, domain.FeatureCalendar)
	}
	if o.contacts {
		features = append(features, domain.FeatureContacts)
	}
	if o.pubsub {
		features = append(features, domain.FeaturePubSub)
	}
	return features
}

func newGoogleSetupCmd() *cobra.Command {
	opts := &googleSetupOpts{}

	cmd := &cobra.Command{
		Use:   "google",
		Short: "Set up Google provider integration",
		Long: `Automated setup wizard for Google provider integration.

Creates a GCP project, enables APIs, configures Pub/Sub, guides you through
OAuth consent screen setup, and creates a Nylas connector.

Requires the gcloud CLI and Google Application Default Credentials.`,
		Example: `  # Interactive wizard
  nylas provider setup google

  # Non-interactive with flags
  nylas provider setup google --project-id my-project --region us --email --calendar --pubsub --yes

  # Resume a previous setup
  nylas provider setup google

  # Start fresh (ignore saved state)
  nylas provider setup google --fresh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate region flag
			if opts.region != "" && opts.region != "us" && opts.region != "eu" {
				return common.NewInputError("region must be 'us' or 'eu'")
			}

			ctx, cancel := common.CreateLongContext()
			defer cancel()

			// Create GCP client (uses ADC)
			gcpClient, err := gcp.NewClient(ctx)
			if err != nil {
				return err
			}

			// Get Nylas client
			nylasClient, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			return runGoogleSetup(ctx, gcpClient, nylasClient, opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "Nylas region (us or eu)")
	cmd.Flags().StringVar(&opts.projectID, "project-id", "", "GCP project ID (skip project selection)")
	cmd.Flags().BoolVar(&opts.email, "email", false, "Enable Email (Gmail API)")
	cmd.Flags().BoolVar(&opts.calendar, "calendar", false, "Enable Calendar (Google Calendar API)")
	cmd.Flags().BoolVar(&opts.contacts, "contacts", false, "Enable Contacts (People API)")
	cmd.Flags().BoolVar(&opts.pubsub, "pubsub", false, "Enable real-time sync via Pub/Sub")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&opts.fresh, "fresh", false, "Ignore saved state and start fresh")

	return cmd
}
