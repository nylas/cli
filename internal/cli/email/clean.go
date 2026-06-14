package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newCleanCmd() *cobra.Command {
	var (
		grantID          string
		keepLinks        bool
		keepImages       bool
		keepTables       bool
		imagesAsMarkdown bool
		keepSignatures   bool
	)

	cmd := &cobra.Command{
		Use:   "clean <message-id> [message-id...]",
		Short: "Strip quoted replies and signatures from messages",
		Long: `Parse one or more messages into clean, display-ready text.

The clean endpoint removes quoted reply/forward chains, signatures, and
conclusion phrases ("Best", "Regards"), returning just the meaningful message
text. Handy for piping a clean message body into AI tools or scripts.

By default links, images, tables, and signature phrases are stripped. Pass the
--keep-* flags to retain them. Up to 20 message IDs may be cleaned in one call.

Plain-text output has HTML tags stripped for readability; use --json to get the
raw cleaned HTML body in the "conversation" field.

API reference: https://developer.nylas.com/docs/v3/email/clean-conversation/`,
		Example: `  # Clean a single message
  nylas email clean <message-id>

  # Clean several messages, keep links
  nylas email clean <id-1> <id-2> --keep-links

  # JSON output for scripting (raw cleaned HTML)
  nylas email clean <message-id> --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > domain.CleanMessagesMaxIDs {
				return fmt.Errorf("clean accepts at most %d message IDs (got %d)", domain.CleanMessagesMaxIDs, len(args))
			}

			req := &domain.CleanMessagesRequest{MessageIDs: args}
			// Only send options that deviate from the API defaults (which strip
			// links/images/tables/signatures); leave the rest unset.
			no, yes := false, true
			if keepLinks {
				req.IgnoreLinks = &no
			}
			if keepImages {
				req.IgnoreImages = &no
			}
			if keepTables {
				req.IgnoreTables = &no
			}
			if imagesAsMarkdown {
				req.ImagesAsMarkdown = &yes
			}
			if keepSignatures {
				req.RemoveConclusionPhrases = &no
			}

			// Message IDs are variadic positionals, so the grant can't ride the
			// usual trailing [grant-id] arg — it comes from --grant (or the
			// active grant when empty).
			cleaned, err := common.WithClient([]string{grantID}, func(ctx context.Context, client ports.NylasClient, gid string) ([]domain.CleanedMessage, error) {
				return client.CleanMessages(ctx, gid, req)
			})
			if err != nil {
				return common.WrapListError("cleaned messages", err)
			}

			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(cleaned)
			}

			if len(cleaned) == 0 {
				common.PrintEmptyState("cleaned messages")
				return nil
			}

			out := cmd.OutOrStdout()
			for i, msg := range cleaned {
				if i > 0 {
					_, _ = fmt.Fprintln(out)
				}
				if len(cleaned) > 1 {
					_, _ = fmt.Fprintf(out, "── %s ──\n", msg.ID)
				}
				_, _ = fmt.Fprintln(out, strings.TrimSpace(common.StripHTML(msg.Conversation)))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&grantID, "grant", "g", "", "Grant ID or email (defaults to the active grant)")
	cmd.Flags().BoolVar(&keepLinks, "keep-links", false, "Keep links instead of stripping them")
	cmd.Flags().BoolVar(&keepImages, "keep-images", false, "Keep images instead of stripping them")
	cmd.Flags().BoolVar(&keepTables, "keep-tables", false, "Keep tables instead of stripping them")
	cmd.Flags().BoolVar(&imagesAsMarkdown, "images-as-markdown", false, "Return images as Markdown links")
	cmd.Flags().BoolVar(&keepSignatures, "keep-signatures", false, `Keep conclusion phrases like "Best" and "Regards"`)

	return cmd
}
