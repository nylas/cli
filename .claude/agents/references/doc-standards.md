# Documentation Standards

Writing style, formatting rules, and common patterns for documentation.

---

## Writing Style

| Principle | Example |
|-----------|---------|
| **Active voice** | "Run `nylas email list`" not "The command can be run" |
| **Imperative mood** | "Configure the API key" not "You should configure" |
| **Concise** | Remove filler words (just, simply, basically) |
| **Scannable** | Use tables, bullets, code blocks |
| **Example-driven** | Show don't tell - include runnable examples |

---

## Formatting Rules

```markdown
# H1 - Document title only (one per file)
## H2 - Major sections
### H3 - Subsections
#### H4 - Rarely needed

**Bold** for emphasis, UI elements, important terms
`code` for commands, flags, file paths, code references
> Blockquotes for notes, warnings, tips

| Tables | For | Structured | Data |
|--------|-----|------------|------|
```

---

## Code Block Standards

```bash
# Always include language identifier
nylas email list --limit 10

# Show expected output when helpful
# Output:
# ID                    Subject                  From
# abc123               Meeting Tomorrow          alice@example.com
```

---

## Command Documentation Pattern

```markdown
## Command Name

Brief description of what it does.

### Usage

\`\`\`bash
nylas <resource> <action> [flags]
\`\`\`

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--format` | `-f` | Output format (json, table, csv) | table |
| `--limit` | `-l` | Maximum results | 50 |

### Examples

\`\`\`bash
# Basic usage
nylas email list

# With filters
nylas email list --from "alice@example.com" --limit 10

# JSON output for scripting
nylas email list --format json | jq '.[] | .subject'
\`\`\`

### Related Commands

- `nylas email show` - View single email
- `nylas email send` - Send new email
```

---

## Common Patterns

### Adding a New Command (Two-Level)

**Step 1: Quick Reference** (`docs/COMMANDS.md`)
```markdown
## Feature Name

\`\`\`bash
nylas feature action --key-flag VALUE    # Brief description
nylas feature other --flag VALUE         # Another action
\`\`\`

**Details:** \`docs/commands/feature.md\`
```

**Step 2: Detailed Docs** (`docs/commands/feature.md`)
```markdown
## Feature Operations

Full description of the feature and its capabilities.

### Action Name

\`\`\`bash
nylas feature action [grant-id]           # Basic usage
nylas feature action --flag1 VALUE        # With option
nylas feature action --flag2 --flag3      # Multiple flags
\`\`\`

**Example output:**
\`\`\`bash
$ nylas feature action --flag1 "test"

Feature Results
-----------------------------------------------------
  Name: Example Item
  ID: item_abc123
  Status: active

Found 1 item
\`\`\`
```

### Adding to Existing Command

1. **COMMANDS.md** - Add brief mention under existing section
2. **README.md** - Add to features list if major feature
3. **docs/commands/<feature>.md** - Add workflow example if complex

### Documenting Breaking Changes

```markdown
## Breaking Changes in vX.Y.Z

### `nylas command` flag renamed

**Before:**
\`\`\`bash
nylas command --old-flag value
\`\`\`

**After:**
\`\`\`bash
nylas command --new-flag value
\`\`\`

**Migration:** Update scripts to use `--new-flag`.
```

### Adding Troubleshooting Entry

```markdown
### Error: "specific error message"

**Cause:** Explanation of why this happens.

**Solution:**
1. Step one
2. Step two

\`\`\`bash
# Fix command
nylas auth login
\`\`\`
```

---

## Quality Checklist

### Before Submitting Doc Changes

- [ ] **Accurate** - Matches current code behavior
- [ ] **Complete** - All flags, options, behaviors documented
- [ ] **Examples work** - Tested the code examples
- [ ] **Links valid** - No broken internal/external links
- [ ] **Consistent** - Follows existing style and patterns
- [ ] **Spell-checked** - No typos
- [ ] **TOC updated** - If document has table of contents

### Public Repo Standards

- [ ] **No internal references** - No internal URLs, team names, or private info
- [ ] **No TODO placeholders** - Complete or remove
- [ ] **No WIP sections** - Either complete or remove
- [ ] **Inclusive language** - No exclusionary terms
- [ ] **Accessible** - Alt text for images, semantic headers

---

## Anti-Patterns

| Don't | Do Instead |
|-------|------------|
| Document implementation details | Document behavior and usage |
| Use internal jargon | Use user-facing terminology |
| Write walls of text | Use bullets, tables, examples |
| Leave TODO comments | Complete or remove |
| Copy code comments as docs | Write user-focused explanation |
| Document obvious things | Focus on non-obvious behavior |
| Use passive voice | Use active, imperative voice |
