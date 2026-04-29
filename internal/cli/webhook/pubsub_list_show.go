package webhook

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newPubSubListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Pub/Sub notification channels",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				resp, err := client.ListPubSubChannels(ctx)
				if err != nil {
					return struct{}{}, common.WrapListError("pub/sub channels", err)
				}
				if len(resp.Data) == 0 {
					common.PrintEmptyStateWithHint(
						"pub/sub channels",
						"Create one with: nylas webhook pubsub create --topic <TOPIC> --triggers <TRIGGERS>",
					)
					return struct{}{}, nil
				}

				out := common.GetOutputWriter(cmd)
				if common.IsStructuredOutput(cmd) {
					return struct{}{}, out.Write(resp)
				}
				return struct{}{}, out.WriteList(resp.Data, pubSubColumns)
			})
			return err
		},
	}

	return cmd
}

func newPubSubShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <channel-id>",
		Short: "Show a Pub/Sub notification channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				channel, err := client.GetPubSubChannel(ctx, args[0])
				if err != nil {
					return struct{}{}, common.WrapGetError("pub/sub channel", err)
				}

				if common.IsStructuredOutput(cmd) {
					return struct{}{}, common.GetOutputWriter(cmd).Write(channel)
				}

				printPubSubChannel(channel, false)
				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}
