package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

// ResourceArgs holds parsed arguments for CRUD operations.
type ResourceArgs struct {
	ResourceID string // The resource ID (first argument)
	GrantID    string // The grant ID (second argument or default)
}

// ParseResourceArgs parses standard resource arguments: <resource-id> [grant-id]
// Returns ResourceArgs with both IDs populated.
func ParseResourceArgs(args []string, minArgs int) (*ResourceArgs, error) {
	if len(args) < minArgs {
		return nil, fmt.Errorf("insufficient arguments")
	}

	result := &ResourceArgs{
		ResourceID: args[0],
	}

	// Grant ID from second argument or default
	var grantArgs []string
	if len(args) > 1 {
		grantArgs = args[1:]
	}

	grantID, err := GetGrantID(grantArgs)
	if err != nil {
		return nil, err
	}

	result.GrantID = grantID
	return result, nil
}

// DeleteConfig configures a delete operation.
type DeleteConfig struct {
	ResourceName string                                                      // e.g., "contact", "event"
	ResourceID   string                                                      // The ID to delete
	GrantID      string                                                      // Grant ID
	Force        bool                                                        // Skip confirmation
	DeleteFunc   func(ctx context.Context, grantID, resourceID string) error // Actual delete function
}

// RunDelete executes a standard delete operation with confirmation, spinner, and success message.
func RunDelete(config DeleteConfig) error {
	// Confirmation prompt
	if !config.Force {
		fmt.Printf("Are you sure you want to delete %s %s? [y/N] ", config.ResourceName, config.ResourceID)
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Create context
	ctx, cancel := CreateContext()
	defer cancel()

	// Delete with spinner
	err := RunWithSpinner(fmt.Sprintf("Deleting %s...", config.ResourceName), func() error {
		return config.DeleteFunc(ctx, config.GrantID, config.ResourceID)
	})
	if err != nil {
		return WrapDeleteError(config.ResourceName, err)
	}

	// Success message
	// Capitalize first letter manually
	resourceName := config.ResourceName
	if len(resourceName) > 0 {
		resourceName = strings.ToUpper(resourceName[:1]) + resourceName[1:]
	}
	fmt.Printf("%s %s deleted successfully.\n", Green.Sprint("✓"), resourceName)

	return nil
}

// ListSetup holds common setup for list commands.
type ListSetup struct {
	Client  ports.NylasClient
	GrantID string
	Ctx     context.Context
	Cancel  context.CancelFunc
}

// SetupListCommand performs standard setup for list commands:
// - Gets client
// - Gets grant ID from args
// - Creates context
// Returns ListSetup with all values populated.
func SetupListCommand(args []string) (*ListSetup, error) {
	client, err := GetNylasClient()
	if err != nil {
		return nil, err
	}

	grantID, err := GetGrantID(args)
	if err != nil {
		return nil, err
	}

	ctx, cancel := CreateContext()

	return &ListSetup{
		Client:  client,
		GrantID: grantID,
		Ctx:     ctx,
		Cancel:  cancel,
	}, nil
}

// NewDeleteCommand creates a standard delete command with common boilerplate handled.
//
// Example usage (with grantID):
//
//	cmd := common.NewDeleteCommand(common.DeleteCommandConfig{
//	    Use: "delete <contact-id> [grant-id]",
//	    Aliases: []string{"rm", "remove"},
//	    Short: "Delete a contact",
//	    ResourceName: "contact",
//	    DeleteFunc: client.DeleteContact,
//	    ShowDetailsFunc: func(ctx, grantID, resourceID string) (string, error) {
//	        contact, _ := client.GetContact(ctx, grantID, resourceID)
//	        return fmt.Sprintf("Name: %s\nEmail: %s", contact.Name, contact.Email), nil
//	    },
//	})
//
// Example usage (without grantID):
//
//	cmd := common.NewDeleteCommand(common.DeleteCommandConfig{
//	    Use: "delete <webhook-id>",
//	    Short: "Delete a webhook",
//	    ResourceName: "webhook",
//	    DeleteFuncNoGrant: func(ctx context.Context, resourceID string) error {
//	        client, _ := common.GetNylasClient()
//	        return client.DeleteWebhook(ctx, resourceID)
//	    },
//	    GetDetailsFunc: func(ctx context.Context, resourceID string) (string, error) {
//	        webhook, _ := client.GetWebhook(ctx, resourceID)
//	        return fmt.Sprintf("URL: %s", webhook.WebhookURL), nil
//	    },
//	})
type DeleteCommandConfig struct {
	Use               string                                                                // Cobra Use string
	Aliases           []string                                                              // Cobra aliases
	Short             string                                                                // Short description
	Long              string                                                                // Long description
	ResourceName      string                                                                // Resource name for messages
	DeleteFunc        func(ctx context.Context, grantID, resourceID string) error           // Delete function (with grantID)
	DeleteFuncNoGrant func(ctx context.Context, resourceID string) error                    // Delete function (without grantID)
	GetClient         func() (ports.NylasClient, error)                                     // Client getter function
	GetDetailsFunc    func(ctx context.Context, resourceID string) (string, error)          // Optional: Get details for confirmation (no grant)
	ShowDetailsFunc   func(ctx context.Context, grantID, resourceID string) (string, error) // Optional: Get details for confirmation (with grant)
	RequiresGrant     bool                                                                  // Whether this resource requires a grant ID
}

// NewDeleteCommand creates a fully configured delete command.
func NewDeleteCommand(config DeleteCommandConfig) *cobra.Command {
	var force bool

	if config.Long == "" {
		config.Long = fmt.Sprintf("Delete a %s by its ID.", config.ResourceName)
	}

	// Determine args based on whether grant is required
	args := cobra.ExactArgs(1)
	if config.RequiresGrant || config.DeleteFunc != nil {
		args = cobra.RangeArgs(1, 2)
	}

	cmd := &cobra.Command{
		Use:     config.Use,
		Aliases: config.Aliases,
		Short:   config.Short,
		Long:    config.Long,
		Args:    args,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get client
			_, err := config.GetClient()
			if err != nil {
				return err
			}

			// Handle resources that don't need grant ID
			if config.DeleteFuncNoGrant != nil {
				resourceID := args[0]

				// Show details if available
				if !force && config.GetDetailsFunc != nil {
					ctx, cancel := CreateContext()
					details, err := config.GetDetailsFunc(ctx, resourceID)
					cancel()
					if err == nil && details != "" {
						fmt.Println(details)
						fmt.Println()
					}
				}

				// Confirm deletion
				if !force {
					fmt.Printf("Are you sure you want to delete %s %s? [y/N] ", config.ResourceName, resourceID)
					var confirm string
					_, _ = fmt.Scanln(&confirm)
					if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
						fmt.Println("Cancelled.")
						return nil
					}
				}

				// Create context and delete
				ctx, cancel := CreateContext()
				defer cancel()

				err := RunWithSpinner(fmt.Sprintf("Deleting %s...", config.ResourceName), func() error {
					return config.DeleteFuncNoGrant(ctx, resourceID)
				})
				if err != nil {
					return WrapDeleteError(config.ResourceName, err)
				}

				// Success message
				resourceName := config.ResourceName
				if len(resourceName) > 0 {
					resourceName = strings.ToUpper(resourceName[:1]) + resourceName[1:]
				}
				fmt.Printf("%s %s deleted successfully.\n", Green.Sprint("✓"), resourceName)
				return nil
			}

			// Handle resources with grant ID (existing logic)
			resourceArgs, err := ParseResourceArgs(args, 1)
			if err != nil {
				return err
			}

			// Show details if available
			if !force && config.ShowDetailsFunc != nil {
				ctx, cancel := CreateContext()
				details, err := config.ShowDetailsFunc(ctx, resourceArgs.GrantID, resourceArgs.ResourceID)
				cancel()
				if err == nil && details != "" {
					fmt.Println(details)
					fmt.Println()
				}
			}

			// Run delete
			return RunDelete(DeleteConfig{
				ResourceName: config.ResourceName,
				ResourceID:   resourceArgs.ResourceID,
				GrantID:      resourceArgs.GrantID,
				Force:        force,
				DeleteFunc:   config.DeleteFunc,
			})
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// ShowCommandConfig configures a show command with custom display logic.
//
// Example usage:
//
//	cmd := common.NewShowCommand(common.ShowCommandConfig{
//	    Use: "show <contact-id> [grant-id]",
//	    Aliases: []string{"get", "read"},
//	    Short: "Show contact details",
//	    ResourceName: "contact",
//	    GetFunc: func(ctx context.Context, client ports.NylasClient, grantID, resourceID string) (any, error) {
//	        return client.GetContact(ctx, grantID, resourceID)
//	    },
//	    DisplayFunc: func(resource any) error {
//	        contact := resource.(*domain.Contact)
//	        fmt.Printf("Name: %s\n", contact.DisplayName())
//	        return nil
//	    },
//	    GetClient: getClient,
//	})
type ShowCommandConfig struct {
	Use          string                                                                                       // Cobra Use string
	Aliases      []string                                                                                     // Cobra aliases
	Short        string                                                                                       // Short description
	Long         string                                                                                       // Long description
	ResourceName string                                                                                       // Resource name for error messages
	GetFunc      func(ctx context.Context, client ports.NylasClient, grantID, resourceID string) (any, error) // Get function
	DisplayFunc  func(resource any) error                                                                     // Custom display function
	GetClient    func() (ports.NylasClient, error)                                                            // Client getter function
}

// NewShowCommand creates a fully configured show command with custom display logic.
func NewShowCommand(config ShowCommandConfig) *cobra.Command {
	if config.Long == "" {
		config.Long = fmt.Sprintf("Display detailed information about a specific %s.", config.ResourceName)
	}

	cmd := &cobra.Command{
		Use:     config.Use,
		Aliases: config.Aliases,
		Short:   config.Short,
		Long:    config.Long,
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse arguments
			resourceArgs, err := ParseResourceArgs(args, 1)
			if err != nil {
				return err
			}

			// Get client
			client, err := config.GetClient()
			if err != nil {
				return err
			}

			// Create context
			ctx, cancel := CreateContext()
			defer cancel()

			// Fetch resource
			resource, err := config.GetFunc(ctx, client, resourceArgs.GrantID, resourceArgs.ResourceID)
			if err != nil {
				return WrapGetError(config.ResourceName, err)
			}

			// Display with custom logic
			return config.DisplayFunc(resource)
		},
	}

	return cmd
}

// UpdateSetup holds common setup for update commands.
type UpdateSetup struct {
	Client     ports.NylasClient
	ResourceID string
	GrantID    string
	Ctx        context.Context
	Cancel     context.CancelFunc
}

// SetupUpdateCommand performs standard setup for update commands:
// - Parses resource ID and grant ID from args
// - Gets client
// - Creates context
// Returns UpdateSetup with all values populated.
func SetupUpdateCommand(args []string) (*UpdateSetup, error) {
	// Parse arguments
	resourceArgs, err := ParseResourceArgs(args, 1)
	if err != nil {
		return nil, err
	}

	// Get client
	client, err := GetNylasClient()
	if err != nil {
		return nil, err
	}

	// Create context
	ctx, cancel := CreateContext()

	return &UpdateSetup{
		Client:     client,
		ResourceID: resourceArgs.ResourceID,
		GrantID:    resourceArgs.GrantID,
		Ctx:        ctx,
		Cancel:     cancel,
	}, nil
}

// PrintUpdateSuccess prints a standardized success message for update operations.
func PrintUpdateSuccess(resourceName string, details ...string) {
	msg := fmt.Sprintf("%s %s updated successfully", Green.Sprint("✓"), capitalize(resourceName))
	if len(details) > 0 {
		msg += ": " + details[0]
	}
	fmt.Println(msg)
}

// capitalize capitalizes the first letter of a string.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
