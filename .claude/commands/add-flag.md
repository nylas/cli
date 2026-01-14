# Add New Command Flag

Add a new flag to an existing CLI command.

## Instructions

1. Ask me for:
   - Which command to modify (e.g., "email list", "webhook create")
   - Flag name (e.g., "--full-ids", "--verbose")
   - Flag type (bool, string, int, stringSlice)
   - Short flag if desired (e.g., "-v")
   - Default value
   - Description

2. Then update the command file:

### For Boolean Flag:
```go
func newListCmd() *cobra.Command {
    var (
        // existing flags...
        newFlag bool  // Add new flag variable
    )

    cmd := &cobra.Command{
        // ... existing config
        RunE: func(cmd *cobra.Command, args []string) error {
            // Use the flag in logic
            if newFlag {
                // Do something different
            }
            return nil
        },
    }

    // Add flag registration
    cmd.Flags().BoolVar(&newFlag, "new-flag", false, "Description of what this flag does")
    // Or with shorthand:
    cmd.Flags().BoolVarP(&newFlag, "new-flag", "n", false, "Description")

    return cmd
}
```

### For String Flag:
```go
var newFlag string
cmd.Flags().StringVarP(&newFlag, "new-flag", "n", "default", "Description")
```

### For Int Flag:
```go
var newFlag int
cmd.Flags().IntVarP(&newFlag, "new-flag", "n", 50, "Description")
```

### For Optional Boolean (pointer pattern):
```go
var newFlag bool
cmd.Flags().BoolVar(&newFlag, "new-flag", false, "Description")

// In RunE, check if explicitly set:
if cmd.Flags().Changed("new-flag") {
    params.NewFlag = &newFlag
}
```

3. Update tests in `{command}_test.go`:
```go
t.Run("has_new_flag", func(t *testing.T) {
    flag := cmd.Flags().Lookup("new-flag")
    assert.NotNil(t, flag)
    assert.Equal(t, "false", flag.DefValue)  // or expected default
})
```

4. Update help test if exists:
```go
func TestCommandHelp(t *testing.T) {
    stdout, _, _ := executeCommand(cmd, "--help")
    assert.Contains(t, stdout, "--new-flag")
}
```

5. Update docs/COMMANDS.md if user-facing:
```markdown
### Command Name
\`\`\`bash
nylas command action --new-flag    # Description
\`\`\`
```

6. Run tests:
   - `go test ./internal/cli/{package}/...`
