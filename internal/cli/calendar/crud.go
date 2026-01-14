package calendar

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <calendar-id> [grant-id]",
		Short: "Show calendar details",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			calendarID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			cal, err := client.GetCalendar(ctx, grantID, calendarID)
			if err != nil {
				return common.WrapGetError("calendar", err)
			}

			fmt.Println("════════════════════════════════════════════════════════════")
			_, _ = common.BoldWhite.Printf("Calendar: %s\n", cal.Name)
			fmt.Println("════════════════════════════════════════════════════════════")

			fmt.Printf("ID:          %s\n", cal.ID)
			fmt.Printf("Name:        %s\n", cal.Name)

			if cal.Description != "" {
				fmt.Printf("Description: %s\n", cal.Description)
			}
			if cal.Location != "" {
				fmt.Printf("Location:    %s\n", cal.Location)
			}
			if cal.Timezone != "" {
				fmt.Printf("Timezone:    %s\n", cal.Timezone)
			}

			if cal.IsPrimary {
				_, _ = common.Cyan.Printf("Primary:     Yes\n")
			}
			if cal.ReadOnly {
				_, _ = common.Dim.Printf("Read-only:   Yes\n")
			}
			if cal.IsOwner {
				fmt.Printf("Owner:       Yes\n")
			}
			if cal.HexColor != "" {
				fmt.Printf("Color:       %s\n", cal.HexColor)
			}

			return nil
		},
	}
}

func newCreateCmd() *cobra.Command {
	var description, location, timezone string

	cmd := &cobra.Command{
		Use:   "create <name> [grant-id]",
		Short: "Create a new calendar",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.CreateCalendarRequest{
				Name:        name,
				Description: description,
				Location:    location,
				Timezone:    timezone,
			}

			cal, err := client.CreateCalendar(ctx, grantID, req)
			if err != nil {
				return common.WrapCreateError("calendar", err)
			}

			_, _ = common.Green.Printf("✓ Created calendar '%s' (ID: %s)\n", cal.Name, cal.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Calendar description")
	cmd.Flags().StringVarP(&location, "location", "l", "", "Calendar location")
	cmd.Flags().StringVarP(&timezone, "timezone", "t", "", "Calendar timezone (e.g., America/New_York)")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var name, description, location, timezone, hexColor string

	cmd := &cobra.Command{
		Use:   "update <calendar-id> [grant-id]",
		Short: "Update a calendar",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			calendarID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.UpdateCalendarRequest{}

			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			if cmd.Flags().Changed("location") {
				req.Location = &location
			}
			if cmd.Flags().Changed("timezone") {
				req.Timezone = &timezone
			}
			if cmd.Flags().Changed("color") {
				req.HexColor = &hexColor
			}

			cal, err := client.UpdateCalendar(ctx, grantID, calendarID, req)
			if err != nil {
				return common.WrapUpdateError("calendar", err)
			}

			_, _ = common.Green.Printf("✓ Updated calendar '%s'\n", cal.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New calendar name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Calendar description")
	cmd.Flags().StringVarP(&location, "location", "l", "", "Calendar location")
	cmd.Flags().StringVarP(&timezone, "timezone", "t", "", "Calendar timezone")
	cmd.Flags().StringVarP(&hexColor, "color", "c", "", "Calendar color (hex, e.g., #FF5733)")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <calendar-id> [grant-id]",
		Short: "Delete a calendar",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			calendarID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if !force {
				cal, err := client.GetCalendar(ctx, grantID, calendarID)
				if err != nil {
					return common.WrapGetError("calendar", err)
				}

				fmt.Println("Delete this calendar?")
				fmt.Printf("  Name: %s\n", cal.Name)
				fmt.Printf("  ID:   %s\n", cal.ID)
				if cal.IsPrimary {
					_, _ = common.Yellow.Printf("  Warning: This is a PRIMARY calendar!\n")
				}
				fmt.Print("\n[y/N]: ")

				var confirm string
				_, _ = fmt.Scanln(&confirm) // Ignore error - empty string treated as "no"
				if confirm != "y" && confirm != "Y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			err = client.DeleteCalendar(ctx, grantID, calendarID)
			if err != nil {
				return common.WrapDeleteError("calendar", err)
			}

			_, _ = common.Green.Printf("✓ Calendar deleted\n")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}
