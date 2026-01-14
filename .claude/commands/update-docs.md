# Update Documentation

Auto-detect code changes and update relevant documentation files.

**See also:** `.claude/rules/documentation-maintenance.md` for documentation rules.

Context: $ARGUMENTS

## Instructions

1. **Detect what changed**

   Run these commands to understand recent changes:
   ```bash
   # Uncommitted changes
   git diff --name-only
   git diff --cached --name-only

   # Recent commits (if looking at committed changes)
   git log --oneline -10
   git diff HEAD~1 --name-only
   ```

2. **Identify documentation impact**

   | Changed File Pattern | Docs to Update |
   |---------------------|----------------|
   | `internal/cli/*/` | `docs/COMMANDS.md` |
   | `internal/domain/*.go` | `docs/COMMANDS.md` (if affects CLI output) |
   | `cmd/nylas/main.go` | `docs/COMMANDS.md` (new commands) |
   | `internal/adapters/nylas/*.go` | `docs/ARCHITECTURE.md` (if new adapter) |
   | Major features | `README.md` |
   | New flags | `docs/COMMANDS.md` |

3. **Update docs/COMMANDS.md**

   For CLI changes, update the relevant sections:

   ```markdown
   ## {command}

   {Brief description}

   ### Usage

   ```bash
   nylas {command} [flags]
   nylas {command} {subcommand} [flags]
   ```

   ### Subcommands

   | Subcommand | Description |
   |------------|-------------|
   | list | List all {resources} |
   | show | Show {resource} details |
   | create | Create a new {resource} |
   | delete | Delete a {resource} |

   ### Flags

   | Flag | Short | Description | Default |
   |------|-------|-------------|---------|
   | --format | -f | Output format (table, json, yaml) | table |
   | --limit | -l | Maximum results to return | 50 |

   ### Examples

   ```bash
   # List all {resources}
   nylas {command} list

   # Show specific {resource}
   nylas {command} show {resource-id}

   # Create with JSON
   nylas {command} create --json '{"field": "value"}'
   ```
   ```

4. **Update README.md** (for major features)

   Only update README for:
   - New command groups (e.g., adding `nylas scheduler`)
   - Major new capabilities
   - Breaking changes

5. **Verify documentation accuracy**

   ```bash
   # Build and test the commands
   make build

   # Verify help text matches docs
   ./bin/nylas {command} --help
   ./bin/nylas {command} {subcommand} --help

   # Test example commands from docs
   ./bin/nylas {command} list --format json
   ```

## Auto-Detection Checklist

Run this analysis:

```bash
# 1. Find all CLI commands
grep -r "Use:" internal/cli/*/

# 2. Find all flags
grep -r "Flags().String\|Flags().Bool\|Flags().Int" internal/cli/*/

# 3. Compare with COMMANDS.md
# Check if all commands are documented
```

## Documentation Templates

### New Command Section

```markdown
## {command}

{One-line description of what this command does.}

### Usage

```bash
nylas {command} [subcommand] [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| list | List {resources} |
| show | Show {resource} details |

### Global Flags

| Flag | Description |
|------|-------------|
| --format | Output format: table, json, yaml, csv |
| --grant-id | Specify grant ID (overrides default) |

### Examples

```bash
# Basic usage
nylas {command} list

# With formatting
nylas {command} list --format json

# Specific item
nylas {command} show <id>
```
```

### New Flag Documentation

```markdown
| Flag | Short | Type | Description | Default |
|------|-------|------|-------------|---------|
| --{flag-name} | -{short} | {string/bool/int} | {Description} | {default} |
```

## Files to Check

1. **docs/COMMANDS.md** - Primary CLI documentation
2. **docs/ARCHITECTURE.md** - Architecture and structure (if major changes)
3. **README.md** - Project overview (major changes only)
4. **CLAUDE.md** - AI assistant guide (if patterns change)

## Verification

```bash
# Ensure docs are consistent with code
make build
./bin/nylas --help

# Check markdown formatting
# (visual inspection or use markdownlint if available)
```

## Checklist

- [ ] Identified all changed files
- [ ] Determined documentation impact
- [ ] Updated `docs/COMMANDS.md` for CLI changes
- [ ] Updated `docs/ARCHITECTURE.md` for structural changes
- [ ] Updated `README.md` for major features
- [ ] Verified examples work correctly
- [ ] Help text matches documentation
- [ ] No stale/outdated documentation remains
