# AI Coding Agent Guidelines

For AI coding agents (Cursor, Copilot, Windsurf, Codex, etc.) working on this codebase.

**Read these first — they are the source of truth:**
- `CLAUDE.md` — Working principles, critical rules, learnings from past mistakes
- `docs/ARCHITECTURE.md` — Hexagonal architecture, project structure, package inventory
- `docs/DEVELOPMENT.md` — Build, test, lint, and CI commands (`make ci`, `make ci-full`, etc.)
- `docs/COMMANDS.md` — CLI command reference
- `.claude/rules/go-quality.md` — Go style, imports, error handling, modern patterns
- `.claude/rules/testing.md` — Test organization, coverage targets, rate limiting

Everything below supplements those docs with quick-reference examples. If anything here conflicts with the above, the above wins.

---

## Quick Reference: Shared Helpers

Don't create package-local wrappers — use these directly:

```go
// CLI client
client := common.GetNylasClient()
grantID := common.GetGrantID(args)

// Client helpers (reduce boilerplate)
common.WithClient(args, func(ctx, client, grantID) (T, error) {
    return client.DoSomething(ctx, grantID)
})
common.WithClientNoGrant(func(ctx, client) (T, error) {
    return client.DoSomething(ctx)
})

// Output
common.PrintSuccess("Email sent successfully")
common.PrintError("Failed to send email", err)
common.FormatSize(bytes)      // "1.5 MB"
common.FormatTimeAgo(time)    // "2 hours ago"
common.PrintJSON(data)        // Pretty-print JSON
out := common.GetOutputWriter(cmd)  // --json/--yaml/--quiet

// Flags
common.AddJSONFlag(cmd, &jsonOutput)   // --json
common.AddLimitFlag(cmd, &limit, 25)   // --limit/-n
common.AddYesFlag(cmd, &yes)           // --yes/-y
common.AddFormatFlag(cmd, &format)     // --format/-f

// Validation
common.ValidateRequired("event ID", eventID)
common.ValidateRequiredFlag("--to", toEmail)
common.ValidateEmail("recipient", email)
common.ValidateURL("webhook URL", webhookURL)
common.ValidateOneOf("status", status, []string{"pending", "active"})

// HTTP (in adapters)
httputil.WriteJSON(w, http.StatusOK, data)
body, err := httputil.LimitedBody(r, maxSize)

// AI (in adapters/ai/)
ConvertMessagesToMaps(messages)
ConvertToolsOpenAIFormat(tools)
```

---

## Quick Reference: Adding a New Feature

1. **Domain:** `internal/domain/<feature>.go` — define types
2. **Port:** `internal/ports/nylas.go` — add interface methods
3. **Adapter:** `internal/adapters/nylas/<feature>.go` — implement
4. **Mock:** `internal/adapters/nylas/mock.go` — add mock methods
5. **CLI:** `internal/cli/<feature>/` — add commands
6. **Register:** `cmd/nylas/main.go` — wire command
7. **Tests:** unit + integration tests
8. **Docs:** update `docs/COMMANDS.md`

---

## Quick Reference: Credential Storage

Credentials stored in system keyring (service: `"nylas"`).

| Key | Description |
|-----|-------------|
| `client_id` | Nylas Application/Client ID |
| `api_key` | Nylas API key (Bearer auth) |
| `client_secret` | Provider OAuth client secret (optional) |
| `org_id` | Nylas Organization ID |
| `grants` | JSON array of grant info (ID, email, provider) |
| `default_grant` | Default grant ID for CLI operations |
| `grant_token_<id>` | Per-grant access tokens |

Key files: `internal/ports/secrets.go`, `internal/adapters/keyring/keyring.go`, `internal/adapters/keyring/grants.go`

Fallback: set `NYLAS_DISABLE_KEYRING=true` for encrypted file store (`~/.config/nylas/`).

---

**Nylas API v3 ONLY** — never use v1/v2.
