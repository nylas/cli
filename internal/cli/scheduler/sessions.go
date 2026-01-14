package scheduler

import (
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newSessionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sessions",
		Aliases: []string{"session"},
		Short:   "Manage scheduler sessions",
		Long:    "Manage scheduler sessions for booking workflows.",
	}

	cmd.AddCommand(newSessionCreateCmd())
	cmd.AddCommand(newSessionShowCmd())

	return cmd
}

func newSessionCreateCmd() *cobra.Command {
	var (
		configID string
		ttl      int
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a scheduler session",
		Long:  "Create a new scheduler session for a configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			req := &domain.CreateSchedulerSessionRequest{
				ConfigurationID: configID,
				TimeToLive:      ttl,
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			session, err := client.CreateSchedulerSession(ctx, req)
			if err != nil {
				return common.WrapCreateError("session", err)
			}

			_, _ = common.Green.Println("✓ Created scheduler session")
			fmt.Printf("  Session ID: %s\n", common.Cyan.Sprint(session.SessionID))
			fmt.Printf("  Configuration ID: %s\n", session.ConfigurationID)

			return nil
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "Configuration ID (required)")
	cmd.Flags().IntVar(&ttl, "ttl", 30, "Time to live in minutes")

	_ = cmd.MarkFlagRequired("config-id")

	return cmd
}

func newSessionShowCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <session-id>",
		Short: "Show scheduler session details",
		Long:  "Show detailed information about a scheduler session.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			session, err := client.GetSchedulerSession(ctx, args[0])
			if err != nil {
				return common.WrapGetError("session", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(session)
			}

			_, _ = common.Bold.Println("Scheduler Session")
			fmt.Printf("  Session ID: %s\n", common.Cyan.Sprint(session.SessionID))
			fmt.Printf("  Configuration ID: %s\n", session.ConfigurationID)

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}
