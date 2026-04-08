package webhook

import (
	"fmt"

	"github.com/nylas/cli/internal/adapters/webhookserver"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	var payload string
	var payloadFile string
	var signature string
	var secret string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a webhook signature locally",
		Long: `Verify a webhook payload against the x-nylas-signature header value.

The payload must be the exact raw body that Nylas sent. Do not reformat or
re-encode the JSON before verifying it.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.ValidateRequiredFlag("--signature", signature); err != nil {
				return err
			}
			if err := common.ValidateRequiredFlag("--secret", secret); err != nil {
				return err
			}

			payload, err := common.ReadStringOrFile("payload", payload, payloadFile, true)
			if err != nil {
				return err
			}

			if webhookserver.VerifySignature([]byte(payload), signature, secret) {
				fmt.Printf("%s Signature is valid.\n", common.Green.Sprint("✓"))
				return nil
			}

			return common.NewUserError(
				"signature verification failed",
				"Ensure you are using the exact raw request body, the x-nylas-signature header value, and the current webhook secret",
			)
		},
	}

	cmd.Flags().StringVar(&payload, "payload", "", "Inline webhook payload body")
	cmd.Flags().StringVar(&payloadFile, "payload-file", "", "Path to a file containing the raw webhook payload body")
	cmd.Flags().StringVar(&signature, "signature", "", "Webhook signature from the x-nylas-signature header")
	cmd.Flags().StringVar(&secret, "secret", "", "Webhook secret used to verify the signature")

	return cmd
}
