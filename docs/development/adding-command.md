# Adding a New CLI Command

Step-by-step guide for adding new commands to the Nylas CLI.

---

## Overview

This project uses **Cobra** for CLI commands and follows **hexagonal architecture** (ports and adapters pattern).

**Layers:**
- **CLI** (`internal/cli/<feature>/`) - User interface layer
- **Port** (`internal/ports/nylas.go`) - Interface contracts
- **Adapter** (`internal/adapters/nylas/`) - API implementations
- **Domain** (`internal/domain/`) - Business entities

---

## Quick Start

### 1. Define Domain Model

**File:** `internal/domain/<feature>.go`

```go
package domain

// Widget represents a widget resource
type Widget struct {
    ID          string `json:"id"`
    GrantID     string `json:"grant_id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    CreatedAt   int64  `json:"created_at"`
}

// WidgetListParams for listing widgets
type WidgetListParams struct {
    Limit  int
    Offset int
}
```

---

### 2. Add Port Interface

**File:** `internal/ports/nylas.go`

```go
// Add to NylasClient interface
type NylasClient interface {
    // ... existing methods

    // Widget operations
    ListWidgets(ctx context.Context, grantID string, params *domain.WidgetListParams) ([]*domain.Widget, error)
    GetWidget(ctx context.Context, grantID, widgetID string) (*domain.Widget, error)
    CreateWidget(ctx context.Context, grantID string, widget *domain.Widget) (*domain.Widget, error)
    DeleteWidget(ctx context.Context, grantID, widgetID string) error
}
```

---

### 3. Implement Adapter

**File:** `internal/adapters/nylas/widgets.go`

```go
package nylas

import (
    "context"
    "fmt"
    "net/http"

    "github.com/nylas/cli/internal/domain"
)

// ListWidgets implements ports.NylasClient
func (c *Client) ListWidgets(ctx context.Context, grantID string, params *domain.WidgetListParams) ([]*domain.Widget, error) {
    url := fmt.Sprintf("%s/v3/grants/%s/widgets", c.baseURL, grantID)

    // Build query params
    query := make(map[string]string)
    if params.Limit > 0 {
        query["limit"] = fmt.Sprintf("%d", params.Limit)
    }

    req, err := c.newRequest(ctx, http.MethodGet, url, query, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    var response struct {
        Data []*domain.Widget `json:"data"`
    }

    if err := c.do(req, &response); err != nil {
        return nil, fmt.Errorf("failed to list widgets: %w", err)
    }

    return response.Data, nil
}

// GetWidget implements ports.NylasClient
func (c *Client) GetWidget(ctx context.Context, grantID, widgetID string) (*domain.Widget, error) {
    url := fmt.Sprintf("%s/v3/grants/%s/widgets/%s", c.baseURL, grantID, widgetID)

    req, err := c.newRequest(ctx, http.MethodGet, url, nil, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    var widget domain.Widget
    if err := c.do(req, &widget); err != nil {
        return nil, fmt.Errorf("failed to get widget: %w", err)
    }

    return &widget, nil
}

// CreateWidget implements ports.NylasClient
func (c *Client) CreateWidget(ctx context.Context, grantID string, widget *domain.Widget) (*domain.Widget, error) {
    url := fmt.Sprintf("%s/v3/grants/%s/widgets", c.baseURL, grantID)

    req, err := c.newRequest(ctx, http.MethodPost, url, nil, widget)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    var created domain.Widget
    if err := c.do(req, &created); err != nil {
        return nil, fmt.Errorf("failed to create widget: %w", err)
    }

    return &created, nil
}

// DeleteWidget implements ports.NylasClient
func (c *Client) DeleteWidget(ctx context.Context, grantID, widgetID string) error {
    url := fmt.Sprintf("%s/v3/grants/%s/widgets/%s", c.baseURL, grantID, widgetID)

    req, err := c.newRequest(ctx, http.MethodDelete, url, nil, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    if err := c.do(req, nil); err != nil {
        return fmt.Errorf("failed to delete widget: %w", err)
    }

    return nil
}
```

---

### 4. Create CLI Commands

**Directory structure:**
```
internal/cli/widgets/
├── widgets.go      # Root command
├── list.go         # List subcommand
├── show.go         # Show subcommand
├── create.go       # Create subcommand
├── delete.go       # Delete subcommand
└── helpers.go      # Shared helpers
```

**File:** `internal/cli/widgets/widgets.go`

```go
package widgets

import (
    "github.com/spf13/cobra"
)

// NewWidgetCmd creates the widget command
func NewWidgetCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "widget",
        Short: "Manage widgets",
        Long:  `Manage widgets including listing, creating, and deleting.`,
    }

    // Add subcommands
    cmd.AddCommand(newListCmd())
    cmd.AddCommand(newShowCmd())
    cmd.AddCommand(newCreateCmd())
    cmd.AddCommand(newDeleteCmd())

    return cmd
}
```

**File:** `internal/cli/widgets/list.go`

```go
package widgets

import (
    "context"
    "fmt"

    "github.com/nylas/cli/internal/domain"
    "github.com/nylas/cli/internal/ports"
    "github.com/spf13/cobra"
)

type listOptions struct {
    limit int
}

func newListCmd() *cobra.Command {
    opts := &listOptions{}

    cmd := &cobra.Command{
        Use:   "list [grant-id]",
        Short: "List widgets",
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runList(cmd.Context(), args, opts)
        },
    }

    cmd.Flags().IntVar(&opts.limit, "limit", 10, "Maximum number of widgets to return")

    return cmd
}

func runList(ctx context.Context, args []string, opts *listOptions) error {
    // Get grant ID
    grantID, err := getGrantID(args)
    if err != nil {
        return err
    }

    // Get client
    client, err := getNylasClient()
    if err != nil {
        return fmt.Errorf("failed to get client: %w", err)
    }

    // List widgets
    params := &domain.WidgetListParams{
        Limit: opts.limit,
    }

    widgets, err := client.ListWidgets(ctx, grantID, params)
    if err != nil {
        return fmt.Errorf("failed to list widgets: %w", err)
    }

    // Display results
    displayWidgets(widgets)

    return nil
}

func displayWidgets(widgets []*domain.Widget) {
    fmt.Printf("Found %d widget(s):\n\n", len(widgets))

    for _, w := range widgets {
        fmt.Printf("ID: %s\n", w.ID)
        fmt.Printf("Name: %s\n", w.Name)
        fmt.Printf("Description: %s\n", w.Description)
        fmt.Println()
    }
}
```

**File:** `internal/cli/widgets/create.go`

```go
package widgets

import (
    "context"
    "fmt"

    "github.com/nylas/cli/internal/domain"
    "github.com/spf13/cobra"
)

type createOptions struct {
    name        string
    description string
}

func newCreateCmd() *cobra.Command {
    opts := &createOptions{}

    cmd := &cobra.Command{
        Use:   "create [grant-id]",
        Short: "Create a new widget",
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runCreate(cmd.Context(), args, opts)
        },
    }

    cmd.Flags().StringVar(&opts.name, "name", "", "Widget name")
    cmd.Flags().StringVar(&opts.description, "description", "", "Widget description")

    _ = cmd.MarkFlagRequired("name")

    return cmd
}

func runCreate(ctx context.Context, args []string, opts *createOptions) error {
    grantID, err := getGrantID(args)
    if err != nil {
        return err
    }

    client, err := getNylasClient()
    if err != nil {
        return fmt.Errorf("failed to get client: %w", err)
    }

    // Create widget
    widget := &domain.Widget{
        Name:        opts.name,
        Description: opts.description,
    }

    created, err := client.CreateWidget(ctx, grantID, widget)
    if err != nil {
        return fmt.Errorf("failed to create widget: %w", err)
    }

    fmt.Printf("✅ Widget created successfully!\n")
    fmt.Printf("ID: %s\n", created.ID)
    fmt.Printf("Name: %s\n", created.Name)

    return nil
}
```

**File:** `internal/cli/widgets/helpers.go`

```go
package widgets

import (
    "fmt"
    "os"

    "github.com/nylas/cli/internal/adapters/nylas"
    "github.com/nylas/cli/internal/config"
    "github.com/nylas/cli/internal/ports"
)

// getGrantID gets grant ID from args or config
func getGrantID(args []string) (string, error) {
    if len(args) > 0 {
        return args[0], nil
    }

    cfg, err := config.Load()
    if err != nil {
        return "", fmt.Errorf("failed to load config: %w", err)
    }

    if cfg.GrantID == "" {
        return "", fmt.Errorf("grant ID required")
    }

    return cfg.GrantID, nil
}

// getNylasClient creates a Nylas client
func getNylasClient() (ports.NylasClient, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }

    if cfg.APIKey == "" {
        return nil, fmt.Errorf("API key not configured")
    }

    return nylas.NewClient(cfg.APIKey), nil
}
```

---

### 5. Register Command

**File:** `cmd/nylas/main.go`

```go
import (
    // ... existing imports
    "github.com/nylas/cli/internal/cli/widgets"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "nylas",
        Short: "Nylas CLI",
    }

    // Register commands
    rootCmd.AddCommand(auth.NewAuthCmd())
    rootCmd.AddCommand(email.NewEmailCmd())
    rootCmd.AddCommand(widgets.NewWidgetCmd())  // Add this line
    // ... other commands

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

---

### 6. Add Unit Tests

**File:** `internal/cli/widgets/list_test.go`

```go
package widgets

import (
    "context"
    "testing"

    "github.com/nylas/cli/internal/domain"
)

func TestListWidgets(t *testing.T) {
    tests := []struct {
        name    string
        opts    *listOptions
        wantErr bool
    }{
        {
            name: "successful list",
            opts: &listOptions{limit: 10},
            wantErr: false,
        },
        {
            name: "with custom limit",
            opts: &listOptions{limit: 50},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

---

### 7. Add Integration Tests

**File:** `internal/cli/integration/widgets_test.go`

```go
//go:build integration
// +build integration

package integration

import (
    "testing"
)

func TestWidgets(t *testing.T) {
    skipIfMissingCreds(t)
    t.Parallel()

    t.Run("ListWidgets", func(t *testing.T) {
        acquireRateLimit(t)

        stdout, stderr, err := runCLIWithRateLimit(t, "widget", "list")

        if err != nil {
            t.Fatalf("Command failed: %v\nStderr: %s", err, stderr)
        }

        if len(stdout) == 0 {
            t.Error("Expected output, got none")
        }
    })

    t.Run("CreateWidget", func(t *testing.T) {
        acquireRateLimit(t)

        stdout, stderr, err := runCLIWithRateLimit(t,
            "widget", "create",
            "--name", "Test Widget",
            "--description", "Integration test",
        )

        if err != nil {
            t.Fatalf("Command failed: %v\nStderr: %s", err, stderr)
        }

        // Verify output contains success message
        if !contains(stdout, "created successfully") {
            t.Error("Expected success message not found")
        }

        // Cleanup
        t.Cleanup(func() {
            acquireRateLimit(t)
            // Delete created widget
        })
    })
}
```

---

### 8. Update Documentation

**File:** `docs/COMMANDS.md`

Add widget command documentation:

```markdown
### Widget Commands

```bash
nylas widget list              # List widgets
nylas widget show <id>         # Show widget details
nylas widget create --name "..." # Create widget
nylas widget delete <id>       # Delete widget
```

**Full guide:** See `docs/commands/widgets.md`
```

**File:** `docs/commands/widgets.md`

Create detailed command guide with examples.

---

### 9. Run Quality Checks

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run --timeout=5m

# Run tests
go test ./... -short

# Run integration tests
make test-integration

# Full check
make ci
```

---

## Best Practices

### Command Design

1. **Use consistent naming:** `nylas <resource> <action>`
2. **Provide aliases:** Common shortcuts (e.g., `ls` for `list`)
3. **Clear help text:** Useful descriptions and examples
4. **Sensible defaults:** Make common use cases easy
5. **Progressive disclosure:** Basic → advanced options

### Error Handling

```go
// ✅ Good - wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create widget: %w", err)
}

// ❌ Bad - lose error context
if err != nil {
    return err
}
```

### Flag Naming

```go
// ✅ Good - descriptive, kebab-case
cmd.Flags().StringVar(&opts.sortBy, "sort-by", "name", "Sort widgets by field")

// ❌ Bad - unclear
cmd.Flags().StringVar(&opts.s, "s", "", "Sort")
```

---

## Complete Example: Full CRUD Command

See `internal/cli/email/` for a complete, production-ready example of CRUD operations.

---

## More Resources

- **Architecture:** [ARCHITECTURE.md](../ARCHITECTURE.md)
- **Testing Guide:** [testing-guide.md](testing-guide.md)
- **Code Style:** [CONTRIBUTING.md](../../CONTRIBUTING.md)
- **Cobra Docs:** https://github.com/spf13/cobra
