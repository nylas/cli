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
| **Output** | `GetOutputWriter(cmd)` | Get output writer based on flags |
| **Output** | `GetOutputOptions(cmd)` | Extract output options from flags |
| **Output** | `AddOutputFlags(cmd)` | Add global output flags to command |
| **Output** | `IsJSON(cmd)` | Check if JSON/YAML/quiet output |
| **Output** | `IsWide(cmd)` | Check if wide mode enabled |
| **Output** | `WriteListWithColumns(cmd, data, cols)` | Write list with table/JSON |
| **Output** | `WriteListWithWideColumns(cmd, data, normal, wide)` | Write with wide support |
| **Client** | `WithClient[T](args, fn)` | Execute with client+grant setup |
| **Client** | `WithClientNoGrant[T](fn)` | Execute with client (no grant) |
| **Flags** | `AddJSONFlag(cmd, &target)` | Add --json flag |
| **Flags** | `AddLimitFlag(cmd, &target, default)` | Add --limit/-n flag |
| **Flags** | `AddYesFlag(cmd, &target)` | Add --yes/-y flag |
| **Flags** | `AddFormatFlag(cmd, &target)` | Add --format/-f flag |
| **Flags** | `AddIDFlag(cmd, &target)` | Add --id flag |
| **Flags** | `AddPageTokenFlag(cmd, &target)` | Add --page-token flag |
| **Flags** | `AddForceFlag(cmd, &target)` | Add --force/-f flag |
| **Flags** | `AddVerboseFlag(cmd, &target)` | Add --verbose/-v flag |
| **Validation** | `ValidateRequired(name, value)` | Validate required argument |
| **Validation** | `ValidateRequiredFlag(flag, value)` | Validate required flag |
| **Validation** | `ValidateRequiredArg(args, name)` | Validate args not empty |
| **Validation** | `ValidateURL(name, value)` | Validate HTTP/HTTPS URL |
| **Validation** | `ValidateEmail(name, value)` | Validate email format |
| **Validation** | `ValidateOneOf(name, value, allowed)` | Validate value in list |
| **Validation** | `ValidateAtLeastOne(name, values...)` | Validate at least one set |
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
| Field validation | `common.ValidateRequired()`, `ValidateRequiredFlag()` |
| URL validation | `common.ValidateURL(name, value)` |
| Email validation | `common.ValidateEmail(name, value)` |
| Pagination handling | `common.FetchAllPages[T]()` |
| Error formatting for CLI | `common.WrapError(err)` or wrap with `fmt.Errorf` |
| Retry with backoff | `common.WithRetry(ctx, config, fn)` |
| Progress indicators | `common.NewSpinner()`, `NewProgressBar()` |
| --json flag | `common.AddJSONFlag(cmd, &jsonOutput)` |
| --limit flag | `common.AddLimitFlag(cmd, &limit, default)` |
| --yes flag | `common.AddYesFlag(cmd, &yes)` |

**Rule:** If you're about to write something from this table, STOP and use the existing helper.

---

## Helper Location Guide

| Helper Type | Location |
|-------------|----------|
| CLI-wide utilities | `internal/cli/common/` |
| HTTP/API helpers | `internal/adapters/nylas/client.go` |
| Feature-specific | `internal/cli/{feature}/helpers.go` |
| Secret storage | `internal/adapters/keyring/keyring.go` |
| Secret constants | `internal/ports/secrets.go` |

---

## Credential Storage (Keyring)

Credentials are stored in the system keyring under service name `"nylas"`.

### Key Constants (`internal/ports/secrets.go`)

| Constant | Key Value | Description |
|----------|-----------|-------------|
| `ports.KeyClientID` | `"client_id"` | Nylas Application/Client ID |
| `ports.KeyAPIKey` | `"api_key"` | Nylas API key (Bearer auth) |
| `ports.KeyClientSecret` | `"client_secret"` | Provider OAuth secret (Google/Microsoft) |
| `ports.KeyOrgID` | `"org_id"` | Nylas Organization ID |
| `ports.GrantTokenKey(id)` | `"grant_token_<id>"` | Per-grant access tokens |

### Grant Keys (`internal/adapters/keyring/grants.go`)

| Key | Description |
|-----|-------------|
| `"grants"` | JSON array of grant info (ID, email, provider) |
| `"default_grant"` | Default grant ID for CLI operations |

### Usage Pattern

```go
// Get secret store
secretStore, _ := keyring.NewSecretStore(configDir)

// Read credentials
apiKey, _ := secretStore.Get(ports.KeyAPIKey)
clientID, _ := secretStore.Get(ports.KeyClientID)
orgID, _ := secretStore.Get(ports.KeyOrgID)

// Save credentials (done in auth/config.go)
secretStore.Set(ports.KeyAPIKey, apiKey)
secretStore.Set(ports.KeyClientID, clientID)
secretStore.Set(ports.KeyOrgID, orgID)

// Grant operations (via GrantStore)
grantStore := keyring.NewGrantStore(secretStore)
grants, _ := grantStore.ListGrants()
defaultID, _ := grantStore.GetDefaultGrant()
```

### Platform Backends
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **macOS**: Keychain
- **Windows**: Windows Credential Manager
- **Fallback**: Encrypted file (`~/.config/nylas/`)
