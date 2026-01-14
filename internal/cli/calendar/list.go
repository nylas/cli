package calendar

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [grant-id]",
		Aliases: []string{"ls"},
		Short:   "List calendars",
		Long:    "List all calendars for the specified grant or default account.",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := getGrantID(args)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			calendars, err := client.GetCalendars(ctx, grantID)
			if err != nil {
				return common.WrapListError("calendars", err)
			}

			if len(calendars) == 0 {
				common.PrintEmptyState("calendars")
				return nil
			}

			fmt.Printf("Found %d calendar(s):\n\n", len(calendars))

			table := common.NewTable("NAME", "ID", "PRIMARY", "READ-ONLY")
			for _, cal := range calendars {
				primary := ""
				if cal.IsPrimary {
					primary = common.Green.Sprint("Yes")
				}
				readOnly := ""
				if cal.ReadOnly {
					readOnly = common.Dim.Sprint("Yes")
				}
				table.AddRow(common.Cyan.Sprint(cal.Name), cal.ID, primary, readOnly)
			}
			table.Render()

			return nil
		},
	}

	return cmd
}
