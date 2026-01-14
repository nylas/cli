# CLI Command Patterns

Reference patterns for implementing CLI commands in `internal/cli/`.

---

## File Structure

```
internal/cli/{resource}/
├── {resource}.go      # Root command with New{Resource}Cmd()
├── list.go            # newListCmd()
├── show.go            # newShowCmd()
├── create.go          # newCreateCmd()
├── update.go          # newUpdateCmd()
├── delete.go          # newDeleteCmd()
├── helpers.go         # getClient(), getGrantID(), createContext()
└── {resource}_test.go # Unit tests
```

---

## Root Command

```go
package resource

import (
    "github.com/spf13/cobra"
)

// NewResourceCmd creates the resource command.
func NewResourceCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "resource",
        Short: "Manage resources",
        Long:  `Commands for managing resources in your Nylas account.`,
    }

    cmd.AddCommand(newListCmd())
    cmd.AddCommand(newShowCmd())
    cmd.AddCommand(newCreateCmd())
    cmd.AddCommand(newUpdateCmd())
    cmd.AddCommand(newDeleteCmd())

    return cmd
}
```

---

## List Command

```go
func newListCmd() *cobra.Command {
    var limit int
    var format string

    cmd := &cobra.Command{
        Use:   "list",
        Short: "List resources",
        Long:  `List all resources for the current grant.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, cancel := createContext()
            defer cancel()

            client, err := getClient()
            if err != nil {
                return err
            }

            grantID, err := getGrantID()
            if err != nil {
                return err
            }

            params := &domain.ResourceQueryParams{
                Limit: limit,
            }

            resources, err := client.GetResources(ctx, grantID, params)
            if err != nil {
                return fmt.Errorf("failed to list resources: %w", err)
            }

            return outputResources(resources, format)
        },
    }

    cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of resources to return")
    cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table, json, yaml)")

    return cmd
}
```

---

## Show Command

```go
func newShowCmd() *cobra.Command {
    var format string

    cmd := &cobra.Command{
        Use:   "show <resource-id>",
        Short: "Show resource details",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, cancel := createContext()
            defer cancel()

            client, err := getClient()
            if err != nil {
                return err
            }

            grantID, err := getGrantID()
            if err != nil {
                return err
            }

            resource, err := client.GetResource(ctx, grantID, args[0])
            if err != nil {
                return fmt.Errorf("failed to get resource: %w", err)
            }

            return outputResource(resource, format)
        },
    }

    cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table, json, yaml)")

    return cmd
}
```

---

## Create Command

```go
func newCreateCmd() *cobra.Command {
    var name string
    var description string

    cmd := &cobra.Command{
        Use:   "create",
        Short: "Create a new resource",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, cancel := createContext()
            defer cancel()

            client, err := getClient()
            if err != nil {
                return err
            }

            grantID, err := getGrantID()
            if err != nil {
                return err
            }

            spinner := pterm.DefaultSpinner.Start("Creating resource...")

            req := &domain.CreateResourceRequest{
                Name:        name,
                Description: description,
            }

            resource, err := client.CreateResource(ctx, grantID, req)
            if err != nil {
                spinner.Fail("Failed to create resource")
                return err
            }

            spinner.Success("Resource created successfully")
            fmt.Printf("ID: %s\n", resource.ID)
            return nil
        },
    }

    cmd.Flags().StringVarP(&name, "name", "n", "", "Resource name (required)")
    cmd.Flags().StringVarP(&description, "description", "d", "", "Resource description")
    _ = cmd.MarkFlagRequired("name")

    return cmd
}
```

---

## Helpers File

```go
package resource

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/nylas/cli/internal/adapters/nylas"
    "github.com/nylas/cli/internal/ports"
)

const defaultTimeout = 30 * time.Second

func getClient() (ports.NylasClient, error) {
    apiKey := os.Getenv("NYLAS_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("NYLAS_API_KEY environment variable is required")
    }
    return nylas.NewClient(apiKey), nil
}

func getGrantID() (string, error) {
    grantID := os.Getenv("NYLAS_GRANT_ID")
    if grantID == "" {
        return "", fmt.Errorf("NYLAS_GRANT_ID environment variable is required")
    }
    return grantID, nil
}

func createContext() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), defaultTimeout)
}
```

---

## Output Helpers

```go
func outputResources(resources []domain.Resource, format string) error {
    switch format {
    case "json":
        return outputJSON(resources)
    case "yaml":
        return outputYAML(resources)
    default:
        return outputTable(resources)
    }
}

func outputTable(resources []domain.Resource) error {
    if len(resources) == 0 {
        fmt.Println("No resources found")
        return nil
    }

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"ID", "Name", "Created"})
    table.SetBorder(false)

    for _, r := range resources {
        table.Append([]string{
            r.ID,
            r.Name,
            time.Unix(r.CreatedAt, 0).Format(time.RFC3339),
        })
    }

    table.Render()
    return nil
}
```

---

## Registration in main.go

```go
// cmd/nylas/main.go
import (
    "github.com/nylas/cli/internal/cli/resource"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "nylas",
        Short: "Nylas CLI",
    }

    // Add command
    rootCmd.AddCommand(resource.NewResourceCmd())

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

---

## Checklist

- [ ] Root command with subcommands
- [ ] --format flag for list/show commands
- [ ] Spinner for long operations
- [ ] Context with timeout
- [ ] Helpful error messages
- [ ] Required flags marked
- [ ] Tests for each command
- [ ] Registered in main.go
