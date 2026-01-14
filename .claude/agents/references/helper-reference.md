# CLI Helper Reference

Complete reference for reusable helper functions. **ALWAYS check these before writing new code.**

---

## CLI Common Helpers (`internal/cli/common/`)

| Category | Helper | Purpose |
|----------|--------|---------|
| **Context** | `CreateContext()` | Standard API timeout context |
| **Context** | `CreateContextWithTimeout(d)` | Custom timeout context |
| **Config** | `GetConfigStore(cmd)` | Get config store from command |
| **Config** | `GetConfigPath(cmd)` | Get config file path |
| **Client** | `GetNylasClient()` | Get configured API client |
| **Client** | `GetAPIKey()` | Get API key from env/config |
| **Client** | `GetGrantID(args)` | Get grant ID from args/env |
| **Colors** | `Bold`, `Dim`, `Cyan`, `Green`, `Yellow`, `Red`, `Blue`, `BoldWhite` | Terminal colors |
| **Errors** | `WrapError(err)` | Wrap error with CLI context |
| **Errors** | `FormatError(err)` | Format error for display |
| **Errors** | `NewUserError(msg, suggestion)` | Create user-facing error |
| **Errors** | `NewInputError(msg)` | Create input validation error |
| **Format** | `NewFormatter(format)` | JSON/YAML/CSV output |
| **Format** | `NewTable(headers...)` | Create display table |
| **Format** | `PrintSuccess/Error/Warning/Info` | Colored output |
| **Format** | `Confirm(prompt, default)` | Y/N confirmation |
| **Pagination** | `FetchAllPages[T](ctx, config, fetcher)` | Paginated API calls |
| **Pagination** | `FetchAllWithProgress[T](...)` | With progress indicator |
| **Progress** | `NewSpinner(msg)` | Loading spinner |
| **Progress** | `NewProgressBar(total, msg)` | Progress bar |
| **Progress** | `NewCounter(msg)` | Item counter |
| **Retry** | `WithRetry(ctx, config, fn)` | Retry with backoff |
| **Retry** | `IsRetryable(err)` | Check if error is retryable |
| **Retry** | `IsRetryableStatusCode(code)` | Check HTTP status |
| **Time** | `FormatTimeAgo(t)` | "2 hours ago" format |
| **Time** | `ParseTimeOfDay(s)` | Parse "3:30 PM" |
| **Time** | `ParseDuration(s)` | Parse "2h30m" |
| **String** | `Truncate(s, maxLen)` | Truncate with ellipsis |
| **Path** | `ValidateExecutablePath(path)` | Validate executable |
| **Path** | `FindExecutableInPath(name)` | Find in PATH |
| **Path** | `SafeCommand(name, args...)` | Create safe exec.Cmd |
| **Logger** | `Debug/Info/Warn/Error(msg, args...)` | Structured logging |
| **Logger** | `DebugHTTP(method, url, status, dur)` | HTTP request logging |

---

## HTTP Client Helpers (`internal/adapters/nylas/client.go`)

| Helper | Purpose |
|--------|---------|
| `c.doJSONRequest(ctx, method, url, body, statuses...)` | JSON API request with error handling |
| `c.doJSONRequestNoAuth(ctx, method, url, body, statuses...)` | JSON request without auth (token exchange) |
| `c.decodeJSONResponse(resp, v)` | Decode response body to struct |
| `validateRequired(fieldName, value)` | Validate required string field |
| `validateGrantID(grantID)` | Validate grant ID not empty |
| `validateCalendarID(calendarID)` | Validate calendar ID not empty |
| `validateMessageID(messageID)` | Validate message ID not empty |
| `validateEventID(eventID)` | Validate event ID not empty |

---

## Common Duplicates to Avoid

These patterns have been duplicated before - ALWAYS check first:

| Pattern | Already Exists In |
|---------|-------------------|
| Context creation for CLI | `common.CreateContext()` |
| Config store retrieval | `common.GetConfigStore(cmd)` |
| Color formatting | `common.Bold`, `common.Cyan`, `common.Green`, etc. |
| JSON API requests (POST/PUT/PATCH) | `c.doJSONRequest(ctx, method, url, body)` |
| Response decoding | `c.decodeJSONResponse(resp, &result)` |
| Field validation | `validateRequired("fieldName", value)` |
| Pagination handling | `common.FetchAllPages[T]()` |
| Error formatting for CLI | `common.WrapError(err)` or wrap with `fmt.Errorf` |
| Retry with backoff | `common.WithRetry(ctx, config, fn)` |
| Progress indicators | `common.NewSpinner()`, `NewProgressBar()` |

**Rule:** If you're about to write something from this table, STOP and use the existing helper.

---

## Helper Location Guide

| Helper Type | Location |
|-------------|----------|
| CLI-wide utilities | `internal/cli/common/` |
| HTTP/API helpers | `internal/adapters/nylas/client.go` |
| Feature-specific | `internal/cli/{feature}/helpers.go` |
