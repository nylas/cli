package scheduler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
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
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.CreateSchedulerSessionRequest{
					ConfigurationID: configID,
					TimeToLive:      ttl,
				}

				session, err := client.CreateSchedulerSession(ctx, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("session", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(session)
				}

				_, _ = common.Green.Println("✓ Created scheduler session")
				fmt.Printf("  Session ID: %s\n", common.Cyan.Sprint(session.SessionID))
				fmt.Printf("  Configuration ID: %s\n", session.ConfigurationID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "Configuration ID (required)")
	cmd.Flags().IntVar(&ttl, "ttl", 30, "Time to live in minutes")

	_ = cmd.MarkFlagRequired("config-id")

	return cmd
}

func newSessionShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <session-id>",
		Short: "Show scheduler session details",
		Long:  "Show detailed information about a scheduler session.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				session, err := client.GetSchedulerSession(ctx, sessionID)
				if err != nil {
					return struct{}{}, common.WrapGetError("session", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(session)
				}

				_, _ = common.Bold.Println("Scheduler Session")
				fmt.Printf("  Session ID: %s\n", common.Cyan.Sprint(session.SessionID))
				fmt.Printf("  Configuration ID: %s\n", session.ConfigurationID)

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}
