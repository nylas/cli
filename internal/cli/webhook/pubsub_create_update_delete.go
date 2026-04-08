package webhook

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newPubSubCreateCmd() *cobra.Command {
	var (
		description   string
		topic         string
		encryptionKey string
		triggers      []string
		notifyEmails  []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Pub/Sub notification channel",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validatePubSubTopic(topic); err != nil {
				return err
			}
			allTriggers, err := parseAndValidateTriggers(triggers)
			if err != nil {
				return err
			}

			req := &domain.CreatePubSubChannelRequest{
				Description:                description,
				TriggerTypes:               allTriggers,
				Topic:                      topic,
				EncryptionKey:              encryptionKey,
				NotificationEmailAddresses: notifyEmails,
			}

			_, err = common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				channel, err := client.CreatePubSubChannel(ctx, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("pub/sub channel", err)
				}
				if common.IsStructuredOutput(cmd) {
					return struct{}{}, common.GetOutputWriter(cmd).Write(channel)
				}
				printPubSubChannel(channel)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Channel description")
	cmd.Flags().StringVar(&topic, "topic", "", "Google Cloud Pub/Sub topic path (required)")
	cmd.Flags().StringVar(&encryptionKey, "encryption-key", "", "Optional encryption key identifier")
	cmd.Flags().StringSliceVarP(&triggers, "triggers", "t", nil, "Trigger types (required, comma-separated or repeated)")
	cmd.Flags().StringSliceVarP(&notifyEmails, "notify", "n", nil, "Notification email addresses")
	_ = cmd.MarkFlagRequired("topic")
	_ = cmd.MarkFlagRequired("triggers")

	return cmd
}

func newPubSubUpdateCmd() *cobra.Command {
	var (
		description   string
		topic         string
		encryptionKey string
		triggers      []string
		notifyEmails  []string
		status        string
	)

	cmd := &cobra.Command{
		Use:   "update <channel-id>",
		Short: "Update a Pub/Sub notification channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &domain.UpdatePubSubChannelRequest{}

			if topic != "" {
				if err := validatePubSubTopic(topic); err != nil {
					return err
				}
				req.Topic = topic
			}
			if len(triggers) > 0 {
				allTriggers, err := parseAndValidateTriggers(triggers)
				if err != nil {
					return err
				}
				req.TriggerTypes = allTriggers
			}
			if status != "" {
				if err := common.ValidateOneOf("status", status, []string{"active", "inactive"}); err != nil {
					return common.NewUserError("invalid status", "Use --status active or --status inactive")
				}
				req.Status = status
			}
			if description != "" {
				req.Description = description
			}
			if encryptionKey != "" {
				req.EncryptionKey = encryptionKey
			}
			if len(notifyEmails) > 0 {
				req.NotificationEmailAddresses = notifyEmails
			}

			if err := common.ValidateAtLeastOne(
				"pub/sub channel field",
				description, topic, encryptionKey, status,
			); err != nil && len(triggers) == 0 && len(notifyEmails) == 0 {
				return err
			}

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				channel, err := client.UpdatePubSubChannel(ctx, args[0], req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("pub/sub channel", err)
				}
				if common.IsStructuredOutput(cmd) {
					return struct{}{}, common.GetOutputWriter(cmd).Write(channel)
				}
				printPubSubChannel(channel)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Updated channel description")
	cmd.Flags().StringVar(&topic, "topic", "", "Updated Google Cloud Pub/Sub topic path")
	cmd.Flags().StringVar(&encryptionKey, "encryption-key", "", "Updated encryption key identifier")
	cmd.Flags().StringSliceVarP(&triggers, "triggers", "t", nil, "Updated trigger types (comma-separated or repeated)")
	cmd.Flags().StringSliceVarP(&notifyEmails, "notify", "n", nil, "Updated notification email addresses")
	cmd.Flags().StringVar(&status, "status", "", "Updated status (active or inactive)")

	return cmd
}

func newPubSubDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <channel-id>",
		Short: "Delete a Pub/Sub notification channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return common.NewUserError(
					"deletion requires confirmation",
					"Re-run with --yes to delete the Pub/Sub channel",
				)
			}

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				if err := client.DeletePubSubChannel(ctx, args[0]); err != nil {
					return struct{}{}, common.WrapDeleteError("pub/sub channel", err)
				}
				if !common.IsStructuredOutput(cmd) {
					common.PrintSuccess("Pub/Sub channel deleted")
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	common.AddYesFlag(cmd, &yes)

	return cmd
}
