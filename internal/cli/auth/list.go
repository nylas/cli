package auth

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

const (
	authListGrantIDWidth  = 38
	authListEmailWidth    = 24
	authListProviderWidth = 12
	authListStatusWidth   = 12
)

type grantListCell struct {
	raw     string
	display string
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all authenticated accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			grantSvc, _, err := createGrantService()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			grants, err := grantSvc.ListGrants(ctx)
			if err != nil {
				return err
			}

			if len(grants) == 0 {
				common.PrintEmptyState("accounts")
				return nil
			}

			// Check if we should use structured output
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(grants)
			}

			verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

			renderGrantListTable(cmd.OutOrStdout(), grants, verbose)

			return nil
		},
	}
}

func renderGrantListTable(w io.Writer, grants []domain.GrantStatus, verbose bool) {
	headers := []grantListCell{
		{raw: "GRANT ID", display: common.Bold.Sprint("GRANT ID")},
		{raw: "EMAIL", display: common.Bold.Sprint("EMAIL")},
		{raw: "PROVIDER", display: common.Bold.Sprint("PROVIDER")},
		{raw: "STATUS", display: common.Bold.Sprint("STATUS")},
		{raw: "DEFAULT", display: common.Bold.Sprint("DEFAULT")},
	}
	widths := []int{
		authListGrantIDWidth,
		authListEmailWidth,
		authListProviderWidth,
		authListStatusWidth,
		utf8.RuneCountInString(headers[4].raw),
	}

	rows := make([][]grantListCell, 0, len(grants))
	for _, grant := range grants {
		row := []grantListCell{
			{raw: grant.ID, display: grant.ID},
			{raw: grant.Email, display: grant.Email},
			{raw: grant.Provider.DisplayName(), display: grant.Provider.DisplayName()},
			grantStatusCell(grant.Status),
			defaultGrantCell(grant.IsDefault),
		}
		rows = append(rows, row)
		for i, cell := range row {
			if width := displayWidth(cell.raw); width > widths[i] {
				widths[i] = width
			}
		}
	}

	renderGrantListRow(w, headers, widths)
	for i, row := range rows {
		renderGrantListRow(w, row, widths)
		if verbose && grants[i].Error != "" {
			_, _ = common.Dim.Fprintf(w, "    Error: %s\n", grants[i].Error)
		}
	}
}

func renderGrantListRow(w io.Writer, row []grantListCell, widths []int) {
	_, _ = fmt.Fprint(w, "  ")
	for i, cell := range row {
		if i > 0 {
			_, _ = fmt.Fprint(w, "  ")
		}
		_, _ = fmt.Fprint(w, cell.display)
		if i < len(row)-1 {
			padding := widths[i] - displayWidth(cell.raw)
			if padding > 0 {
				_, _ = fmt.Fprint(w, strings.Repeat(" ", padding))
			}
		}
	}
	_, _ = fmt.Fprintln(w)
}

func grantStatusCell(status string) grantListCell {
	switch status {
	case "valid":
		return grantListCell{raw: "✓ valid", display: common.Green.Sprint("✓ valid")}
	case "error":
		return grantListCell{raw: "✗ error", display: common.Red.Sprint("✗ error")}
	case "revoked":
		return grantListCell{raw: "✗ revoked", display: common.Red.Sprint("✗ revoked")}
	default:
		return grantListCell{raw: status, display: common.Yellow.Sprint(status)}
	}
}

func defaultGrantCell(isDefault bool) grantListCell {
	if !isDefault {
		return grantListCell{}
	}
	return grantListCell{raw: "✓", display: common.Green.Sprint("✓")}
}

func displayWidth(s string) int {
	return utf8.RuneCountInString(s)
}
