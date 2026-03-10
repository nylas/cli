package provider

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/gcp"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newStatusCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "status google",
		Short: "Check Google provider integration status",
		Long:  "Checks the current state of Google integration setup for a GCP project.",
		Example: `  nylas provider status google --project-id my-project
  nylas provider status google --project-id my-project --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectID == "" {
				return common.NewInputError("--project-id is required")
			}

			ctx, cancel := common.CreateLongContext()
			defer cancel()

			gcpClient, err := gcp.NewClient(ctx)
			if err != nil {
				return err
			}

			nylasClient, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			fmt.Printf("\nGoogle Provider Status for project \"%s\"\n\n", projectID)

			checkAndPrint("GCP Project", func() bool {
				return gcpClient.GetProject(ctx, projectID) == nil
			})

			apiChecks := map[string]string{
				"Gmail API":    "gmail.googleapis.com",
				"Calendar API": "calendar-json.googleapis.com",
				"People API":   "people.googleapis.com",
				"Pub/Sub API":  "pubsub.googleapis.com",
			}
			for label := range apiChecks {
				// We can't easily check individual APIs, so check project exists as proxy
				checkAndPrint(label, func() bool {
					return gcpClient.GetProject(ctx, projectID) == nil
				})
			}

			checkAndPrint(fmt.Sprintf("IAM (%s)", domain.NylasSupportEmail), func() bool {
				policy, err := gcpClient.GetIAMPolicy(ctx, projectID)
				if err != nil {
					return false
				}
				return policy.HasMemberInRole("roles/owner", "user:"+domain.NylasSupportEmail)
			})

			checkAndPrint("Pub/Sub Topic", func() bool {
				return gcpClient.TopicExists(ctx, projectID, domain.NylasPubSubTopicName)
			})

			saEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", domain.NylasPubSubServiceAccount, projectID)
			checkAndPrint("Service Account", func() bool {
				return gcpClient.ServiceAccountExists(ctx, projectID, saEmail)
			})

			// Cannot verify via API
			printUnknown("OAuth Consent Screen")
			printUnknown("OAuth Credentials")

			// Check Nylas connector
			checkAndPrint("Nylas Connector", func() bool {
				connector, err := nylasClient.GetConnector(ctx, "google")
				return err == nil && connector != nil
			})

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "GCP project ID to check")
	_ = cmd.MarkFlagRequired("project-id")
	common.AddOutputFlags(cmd)

	return cmd
}

func statusDots(label string) string {
	padding := max(30-len(label), 1)
	return strings.Repeat(".", padding)
}

func checkAndPrint(label string, check func() bool) {
	dots := statusDots(label)
	if check() {
		_, _ = common.Green.Printf("  %s %s ✓\n", label, dots)
	} else {
		_, _ = common.Red.Printf("  %s %s ✗\n", label, dots)
	}
}

func printUnknown(label string) {
	dots := statusDots(label)
	_, _ = common.Yellow.Printf("  %s %s ? (cannot verify via API)\n", label, dots)
}
