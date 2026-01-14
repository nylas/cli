package calendar

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// newRecurringCmd creates the recurring events command.
func newRecurringCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recurring",
		Short: "Manage recurring events",
		Long: `Manage recurring calendar events, including viewing all instances,
updating or deleting specific occurrences.`,
	}

	cmd.AddCommand(newRecurringListCmd())
	cmd.AddCommand(newRecurringUpdateCmd())
	cmd.AddCommand(newRecurringDeleteCmd())

	return cmd
}

// newRecurringListCmd creates the list recurring event instances command.
func newRecurringListCmd() *cobra.Command {
	var (
		calendarID string
		grantID    string
		jsonOutput bool
		limit      int
		startUnix  int64
		endUnix    int64
	)

	cmd := &cobra.Command{
		Use:   "list <master-event-id> [grant-id]",
		Short: "List all instances of a recurring event",
		Long: `List all occurrences of a recurring event series.
The master event ID is the ID of the parent recurring event.`,
		Example: `  # List all instances of a recurring event
  nylas calendar recurring list event-master-123 --calendar cal-456

  # List instances with a date range
  nylas calendar recurring list event-master-123 --calendar cal-456 --start 1704067200 --end 1706745600

  # List with custom limit
  nylas calendar recurring list event-master-123 --calendar cal-456 --limit 100`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			masterEventID := args[0]

			if len(args) > 1 {
				grantID = args[1]
			}

			if calendarID == "" {
				return common.NewUserError("calendar ID is required", "Use --calendar to specify the calendar")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if grantID == "" {
				grantID, err = getGrantID([]string{})
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			params := &domain.EventQueryParams{
				Limit:           limit,
				ExpandRecurring: true,
				Start:           startUnix,
				End:             endUnix,
			}

			instances, err := client.GetRecurringEventInstances(ctx, grantID, calendarID, masterEventID, params)
			if err != nil {
				return common.WrapFetchError("recurring event instances", err)
			}

			if jsonOutput {
				return json.NewEncoder(os.Stdout).Encode(instances)
			}

			if len(instances) == 0 {
				common.PrintEmptyState("recurring event instances")
				return nil
			}

			table := common.NewTable("INSTANCE ID", "TITLE", "START TIME", "STATUS")
			for _, event := range instances {
				startTime := time.Unix(event.When.StartTime, 0).Format("2006-01-02 15:04")
				table.AddRow(event.ID, event.Title, startTime, event.Status)
			}
			table.Render()

			fmt.Printf("\nTotal instances: %d\n", len(instances))
			if len(instances) > 0 && instances[0].MasterEventID != "" {
				fmt.Printf("Master Event ID: %s\n", instances[0].MasterEventID)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (required)")
	cmd.Flags().StringVarP(&grantID, "grant", "g", "", "Grant ID (uses default if not specified)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of instances to retrieve")
	cmd.Flags().Int64Var(&startUnix, "start", 0, "Start time (Unix timestamp)")
	cmd.Flags().Int64Var(&endUnix, "end", 0, "End time (Unix timestamp)")
	// Note: --calendar validation is done in RunE for better error messages

	return cmd
}

// newRecurringUpdateCmd creates the update recurring event instance command.
func newRecurringUpdateCmd() *cobra.Command {
	var (
		calendarID  string
		grantID     string
		title       string
		description string
		location    string
		startTime   string
		endTime     string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "update <instance-id> [grant-id]",
		Short: "Update a single instance of a recurring event",
		Long: `Update a specific occurrence of a recurring event series.
This creates an exception for that particular instance.`,
		Example: `  # Update the title of a specific instance
  nylas calendar recurring update event-instance-123 --calendar cal-456 --title "Rescheduled Meeting"

  # Update time and location
  nylas calendar recurring update event-instance-123 --calendar cal-456 \
    --start "2024-01-15T14:00:00" --end "2024-01-15T15:30:00" \
    --location "Conference Room B"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceID := args[0]

			if len(args) > 1 {
				grantID = args[1]
			}

			if calendarID == "" {
				return common.NewUserError("calendar ID is required", "Use --calendar to specify the calendar")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if grantID == "" {
				grantID, err = getGrantID([]string{})
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.UpdateEventRequest{}

			if title != "" {
				req.Title = &title
			}
			if description != "" {
				req.Description = &description
			}
			if location != "" {
				req.Location = &location
			}

			if startTime != "" || endTime != "" {
				when := &domain.EventWhen{Object: "timespan"}
				if startTime != "" {
					t, err := time.Parse(time.RFC3339, startTime)
					if err != nil {
						return common.NewUserError("invalid start time format", "use RFC3339 (e.g., 2024-01-15T14:00:00Z)")
					}
					when.StartTime = t.Unix()
				}
				if endTime != "" {
					t, err := time.Parse(time.RFC3339, endTime)
					if err != nil {
						return common.NewUserError("invalid end time format", "use RFC3339 (e.g., 2024-01-15T15:00:00Z)")
					}
					when.EndTime = t.Unix()
				}
				req.When = when
			}

			event, err := client.UpdateRecurringEventInstance(ctx, grantID, calendarID, instanceID, req)
			if err != nil {
				return common.WrapUpdateError("recurring event instance", err)
			}

			if jsonOutput {
				return json.NewEncoder(os.Stdout).Encode(event)
			}

			fmt.Printf("✓ Updated recurring event instance\n")
			fmt.Printf("  ID:    %s\n", event.ID)
			fmt.Printf("  Title: %s\n", event.Title)
			if event.When.StartTime > 0 {
				fmt.Printf("  Start: %s\n", time.Unix(event.When.StartTime, 0).Format(common.DateTimeFormat))
			}
			if event.When.EndTime > 0 {
				fmt.Printf("  End:   %s\n", time.Unix(event.When.EndTime, 0).Format(common.DateTimeFormat))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (required)")
	cmd.Flags().StringVarP(&grantID, "grant", "g", "", "Grant ID (uses default if not specified)")
	cmd.Flags().StringVar(&title, "title", "", "New title for this instance")
	cmd.Flags().StringVar(&description, "description", "", "New description for this instance")
	cmd.Flags().StringVar(&location, "location", "", "New location for this instance")
	cmd.Flags().StringVar(&startTime, "start", "", "New start time (RFC3339 format)")
	cmd.Flags().StringVar(&endTime, "end", "", "New end time (RFC3339 format)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	_ = cmd.MarkFlagRequired("calendar") // Hardcoded flag name, won't fail

	return cmd
}

// newRecurringDeleteCmd creates the delete recurring event instance command.
func newRecurringDeleteCmd() *cobra.Command {
	var (
		calendarID  string
		grantID     string
		skipConfirm bool
	)

	cmd := &cobra.Command{
		Use:   "delete <instance-id> [grant-id]",
		Short: "Delete a single instance of a recurring event",
		Long: `Delete a specific occurrence of a recurring event series.
This adds an exception to the recurrence rule.`,
		Example: `  # Delete a specific instance (with confirmation)
  nylas calendar recurring delete event-instance-123 --calendar cal-456

  # Delete without confirmation
  nylas calendar recurring delete event-instance-123 --calendar cal-456 -y`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceID := args[0]

			if len(args) > 1 {
				grantID = args[1]
			}

			if calendarID == "" {
				return common.NewUserError("calendar ID is required", "Use --calendar to specify the calendar")
			}

			if !skipConfirm {
				fmt.Printf("Are you sure you want to delete this recurring event instance? (y/N): ")
				var response string
				_, _ = fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if grantID == "" {
				grantID, err = getGrantID([]string{})
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if err := client.DeleteRecurringEventInstance(ctx, grantID, calendarID, instanceID); err != nil {
				return common.WrapDeleteError("recurring event instance", err)
			}

			fmt.Printf("✓ Deleted recurring event instance %s\n", instanceID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (required)")
	cmd.Flags().StringVarP(&grantID, "grant", "g", "", "Grant ID (uses default if not specified)")
	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	_ = cmd.MarkFlagRequired("calendar") // Hardcoded flag name, won't fail

	return cmd
}
