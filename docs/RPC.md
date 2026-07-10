# Nylas CLI — RPC Server (`nylas rpc serve`)

A local **JSON-RPC 2.0 server over WebSocket** that exposes the Nylas CLI's full
capability surface to a thin client (for example a desktop app). The CLI binary is the
engine — it holds the credentials, runs the live pollers, and owns all business logic.
Clients are thin: they send requests and render the streamed results and notifications.

- **Endpoint:** `ws://127.0.0.1:7369/ws`
- **Protocol:** JSON-RPC 2.0 (bidirectional) over WebSocket
- **Auth:** per-session bearer token, loopback-only bind
- **Surface:** ~108 methods across 18 domains + live push notifications

---

## Table of contents

1. [Quick start](#quick-start)
2. [Architecture](#architecture)
3. [Transport & message format](#transport--message-format)
4. [Authentication & security](#authentication--security)
5. [Configuration](#configuration)
6. [Error codes](#error-codes)
7. [Method reference](#method-reference)
8. [Notifications (server → client push)](#notifications-server--client-push)
9. [Examples](#examples)
10. [Testing](#testing)
11. [Limitations & scope](#limitations--scope)

---

## Quick start

Start the server:

```bash
nylas rpc serve                      # binds 127.0.0.1:7369
nylas rpc serve --addr 127.0.0.1:9000
```

On first run the server generates a session token, stores it in the OS keyring, and
prints how to authenticate. Print the current token (generates one if none exists):

```bash
nylas rpc token            # prints the token
nylas rpc token --json     # {"token":"…"}
nylas rpc token --copy     # copy to clipboard
```

To inject a known token instead (headless / scripting):

```bash
NYLAS_WS_TOKEN=my-secret nylas rpc serve
```

Connect a WebSocket client to `ws://127.0.0.1:7369/ws` with the token, then send a request:

```json
{ "jsonrpc": "2.0", "id": 1, "method": "email.list", "params": { "limit": 10 } }
```

---

## Architecture

The CLI follows a hexagonal architecture (CLI → Port → Adapter). The RPC server is a
**second inbound adapter** alongside the cobra CLI, calling the same `ports.NylasClient`:

```
client ──ws (JSON-RPC request)──▶ rpcserver ──▶ ports.NylasClient ──▶ Nylas API
client ◀─ws (JSON-RPC notification)── rpcserver ◀── pollers (received_after / updated_after / …)
        127.0.0.1 + session token + Origin check ; creds from the OS keyring
```

- **No business logic in the client.** It can't run an operation itself; it only talks to the server.
- **The server owns:** credentials (keyring), the incremental pollers, session/connection state.
- **Code:** `internal/cli/rpc/` (command) and `internal/adapters/rpcserver/` (server + handlers + pollers).

---

## Transport & message format

WebSocket at path `/ws`. Messages are JSON-RPC 2.0 objects, one per WebSocket text frame.

**Request** (has `id`):
```json
{ "jsonrpc": "2.0", "id": 1, "method": "email.list", "params": { "limit": 5 } }
```

**Response** (matches `id`):
```json
{ "jsonrpc": "2.0", "id": 1, "result": { "messages": [ … ], "next_cursor": "…", "has_more": true } }
```

**Error response:**
```json
{ "jsonrpc": "2.0", "id": 1, "error": { "code": -32602, "message": "message_id required" } }
```

**Notification** (server → client, no `id`):
```json
{ "jsonrpc": "2.0", "method": "message.received", "params": { "id": "…", "subject": "…" } }
```

Notes:
- Parse-error / invalid-request responses carry `"id": null` per the spec.
- A **notification from the client** (a request with no `id`) is executed but gets **no reply** —
  used for `client.focus` (see [adaptive polling](#adaptive-polling)).
- `page_token` / `next_cursor` are **opaque** — store and replay verbatim per grant; the format
  differs by provider (e.g. base64-JSON for the `nylas` provider vs numeric for Google).

---

## Authentication & security

The server holds live Nylas credentials, so the local socket is a real trust boundary.

- **Bearer token** on the WebSocket upgrade, via either:
  - `Authorization: Bearer <token>` header, or
  - `?token=<token>` query parameter.
  Wrong/missing token → **401**. Comparison is constant-time over SHA-256 digests (no length leak).
- **Token lifecycle:** generated once (32 bytes, `crypto/rand`, base64url), persisted in the OS
  keyring (`rpc_session_token`); `NYLAS_WS_TOKEN` overrides. Reused across restarts (no rotation/expiry).
- **Loopback only:** binds `127.0.0.1`. A non-loopback `--addr` is **refused** unless `--allow-remote`
  is passed (then it warns). Never expose a credential-holding socket to the network unauthenticated.
- **Origin check:** non-empty `Origin` headers are rejected (blocks browser-based CSWSH / DNS-rebinding).
- **Generic client errors:** internal/upstream error detail is logged to the server's stderr; the
  client receives a generic `-32603 "internal error"` (intentional RPC errors like `-32602` pass through).
- **`config.read` is whitelisted** — it never returns secrets, grants, or AI/GPG/dashboard sub-objects
  (only boolean presence flags).
- **Writes execute immediately** — there is no server-side confirmation prompt. The **client** is
  responsible for confirming destructive/outbound operations before calling.

---

## Configuration

| Flag / Env | Purpose | Default |
|---|---|---|
| `--addr` | bind address | `127.0.0.1:7369` |
| `NYLAS_WS_ADDR` | bind address (env; `--addr` wins) | `127.0.0.1:7369` |
| `--allow-remote` | permit a non-loopback bind (warns) | `false` |
| `NYLAS_WS_TOKEN` | inject the session token (headless/CI) | auto-generated, keyring-brokered |
| `NYLAS_DISABLE_KEYRING` | store token/creds in `~/.config/nylas` instead of the keyring | `false` |
| `NYLAS_WS_POLL_FAST` | message/thread/event poll interval while focused (Go duration) | `5s` |
| `NYLAS_WS_POLL_IDLE` | message/thread/event poll interval while idle (Go duration) | `30s` |
| `NYLAS_WS_POLL_CONTACTS` | contact refetch interval (Go duration) | `60s` |

Invalid or non-positive poll durations fall back to the default. The intervals can also be changed
at runtime via the [`client.pollConfig`](#adaptive-polling) method — no restart required.

The server resolves the Nylas API credentials and default grant the same way the rest of the
CLI does (keyring, or env/file when `NYLAS_DISABLE_KEYRING=true`). Live pollers run only when a
**default grant** is configured; otherwise the server still serves requests and prints a notice.

---

## Error codes

Standard JSON-RPC 2.0 codes:

| Code | Meaning | When |
|---|---|---|
| `-32700` | Parse error | malformed JSON (response `id` is `null`) |
| `-32600` | Invalid request | missing/incorrect `jsonrpc` field |
| `-32601` | Method not found | unknown method |
| `-32602` | Invalid params | missing required param / bad value (e.g. `message_id required`) |
| `-32603` | Internal error | upstream/handler failure (detail logged server-side, generic to client) |

---

## Method reference

Conventions:
- `grant_id` is **optional** on per-grant methods — it falls back to the server's default grant.
  App-level methods (admin, etc.) take **no** grant. Scheduler configs are **per-grant** (the
  Nylas v3 configuration endpoints are grant-scoped).
- Required ids return `-32602` when missing.
- Create/update params **embed the corresponding `domain.*Request` struct** at the top level — i.e.
  the request fields sit alongside `grant_id`/ids (see `internal/domain` for exact fields).
- Delete-style methods return `{ "deleted": true }` (or `{ "revoked": true }` / `{ "cancelled": true }`).

### Email
| Method | Params | Result |
|---|---|---|
| `email.list` | `grant_id?, limit?, page_token?, received_after?` | `{ messages, next_cursor, has_more }` |
| `email.get` | `grant_id?, message_id` | message |
| `email.send` | `grant_id?` + `SendMessageRequest` | message |
| `email.update` | `grant_id?, message_id` + `UpdateMessageRequest` | message |
| `email.delete` | `grant_id?, message_id` | `{ deleted }` |
| `email.clean` | `grant_id?` + `CleanMessagesRequest` | `{ messages }` (cleaned) |
| `email.folder.list` | `grant_id?` | `{ folders }` |
| `email.folder.get` | `grant_id?, folder_id` | folder |
| `email.folder.create` | `grant_id?` + `CreateFolderRequest` | folder |
| `email.folder.update` | `grant_id?, folder_id` + `UpdateFolderRequest` | folder |
| `email.folder.delete` | `grant_id?, folder_id` | `{ deleted }` |
| `email.attachment.list` | `grant_id?, message_id` | `{ attachments }` |
| `email.attachment.get` | `grant_id?, message_id, attachment_id` | attachment (metadata) |
| `email.attachment.download` | `grant_id?, message_id, attachment_id` | `{ content (base64), size }` |
| `email.signature.list` | `grant_id?` | `{ signatures }` |
| `email.signature.get` | `grant_id?, signature_id` | signature |
| `email.signature.create` | `grant_id?` + `CreateSignatureRequest` | signature |
| `email.signature.update` | `grant_id?, signature_id` + `UpdateSignatureRequest` | signature |
| `email.signature.delete` | `grant_id?, signature_id` | `{ deleted }` |
| `email.scheduled.list` | `grant_id?` | `{ scheduled }` |
| `email.scheduled.get` | `grant_id?, schedule_id` | scheduled message |
| `email.scheduled.cancel` | `grant_id?, schedule_id` | `{ cancelled }` |

### Drafts
| Method | Params | Result |
|---|---|---|
| `draft.list` | `grant_id?, limit?` | `{ drafts }` |
| `draft.get` | `grant_id?, draft_id` | draft |
| `draft.create` | `grant_id?` + `CreateDraftRequest` | draft |
| `draft.update` | `grant_id?, draft_id` + `CreateDraftRequest` | draft |
| `draft.delete` | `grant_id?, draft_id` | `{ deleted }` |
| `draft.send` | `grant_id?, draft_id` + `SendDraftRequest` | message |

### Threads
| Method | Params | Result |
|---|---|---|
| `thread.list` | `grant_id?, limit?, page_token?, latest_message_after?, unread?` | `{ threads, next_cursor, has_more }` |
| `thread.get` | `grant_id?, thread_id` | thread |
| `thread.update` | `grant_id?, thread_id` + `UpdateMessageRequest` (unread/starred/folders) | thread |
| `thread.delete` | `grant_id?, thread_id` | `{ deleted }` |

### Calendar & events
| Method | Params | Result |
|---|---|---|
| `calendar.list` | `grant_id?` | `{ calendars }` |
| `event.list` | `grant_id?, calendar_id=primary, limit?, page_token?, updated_after?, start?, end?` | `{ events, next_cursor, has_more }` |
| `event.get` | `grant_id?, calendar_id=primary, event_id` | event |
| `event.create` | `grant_id?, calendar_id=primary` + `CreateEventRequest` | event |
| `event.update` | `grant_id?, calendar_id=primary, event_id` + `UpdateEventRequest` | event |
| `event.delete` | `grant_id?, calendar_id=primary, event_id` | `{ deleted }` |
| `event.rsvp` | `grant_id?, calendar_id=primary, event_id` + `SendRSVPRequest` | `{ ok }` |
| `event.import` | `grant_id?` + `EventQueryParams` (incl. `calendar_id`, `start`, `end`) | `{ events }` |
| `event.recurring.list` | `grant_id?, calendar_id=primary, master_event_id` + `EventQueryParams` | `{ events }` (instances) |
| `event.recurring.update` | `grant_id?, calendar_id=primary, event_id` + `UpdateEventRequest` | event |
| `event.recurring.delete` | `grant_id?, calendar_id=primary, event_id` | `{ deleted }` |
| `calendar.get` | `grant_id?, calendar_id` | calendar |
| `calendar.create` | `grant_id?` + `CreateCalendarRequest` | calendar |
| `calendar.update` | `grant_id?, calendar_id` + `UpdateCalendarRequest` | calendar |
| `calendar.delete` | `grant_id?, calendar_id` | `{ deleted }` |
| `calendar.freeBusy` | `grant_id?` + `FreeBusyRequest` | free/busy response |
| `calendar.availability` | `AvailabilityRequest` (**no grant**) | availability response |
| `calendar.resources` | `grant_id?` | `{ resources }` (bookable rooms) |
| `calendar.virtual.create` | `email` (**no grant**) | virtual calendar grant |
| `calendar.virtual.list` | — (**no grant**) | `{ grants }` |
| `calendar.virtual.get` | `grant_id` (the virtual grant id) | virtual calendar grant |
| `calendar.virtual.delete` | `grant_id` (the virtual grant id) | `{ deleted }` |

### Contacts
| Method | Params | Result |
|---|---|---|
| `contact.list` | `grant_id?, limit?, page_token?` | `{ contacts, next_cursor, has_more }` |
| `contact.get` | `grant_id?, contact_id` | contact |
| `contact.create` | `grant_id?` + `CreateContactRequest` | contact |
| `contact.update` | `grant_id?, contact_id` + `UpdateContactRequest` | contact |
| `contact.delete` | `grant_id?, contact_id` | `{ deleted }` |
| `contact.getWithPicture` | `grant_id?, contact_id, include_picture?` | contact (with base64 `picture` when requested) |
| `contact.group.list` | `grant_id?` | `{ groups }` |
| `contact.group.get` | `grant_id?, group_id` | contact group |
| `contact.group.create` | `grant_id?` + `CreateContactGroupRequest` | contact group |
| `contact.group.update` | `grant_id?, group_id` + `UpdateContactGroupRequest` | contact group |
| `contact.group.delete` | `grant_id?, group_id` | `{ deleted }` |

### Agent accounts / grants / config
| Method | Params | Result |
|---|---|---|
| `agentAccount.list` | — | `{ accounts }` |
| `agentAccount.get` | `grant_id` (the agent account's grant) | account |
| `grant.list` | — | `{ grants }` (local store: id/email/provider) |
| `config.read` | — | whitelisted config (region, default_grant, callback_port, tui_theme, api{base_url,timeout}, working_hours, ai/gpg/dashboard `*_configured` booleans). **No secrets.** |

### Notetaker
`notetaker.list` (`grant_id?` + query) · `notetaker.get` · `notetaker.create` · `notetaker.update`
· `notetaker.delete` → `{ deleted }` · `notetaker.leave` → `{ left }` · `notetaker.media`
(all per-grant; `notetaker_id` required where applicable).

### Scheduler
- Configs (per-grant): `scheduler.config.list` / `.get` / `.create` / `.update` / `.delete`
- Sessions: `scheduler.session.create` (requires `configuration_id` or `slug`; `time_to_live` in
  minutes, max 30, legacy alias `ttl`) / `.get`
- Bookings (all require `configuration_id` — booking endpoints authenticate with a Scheduler
  session token minted from the configuration):
  `scheduler.booking.get` / `.confirm` (`salt` + `status` required; `cancellation_reason`, legacy
  alias `reason`, applies when declining) / `.reschedule` (returns the booking; adds `warning`
  when the reschedule was applied but the record could not be read back) / `.cancel`
  (`{ cancelled }`; `cancellation_reason`, legacy alias `reason`)
- Group events (per-grant): `scheduler.groupEvent.list` (requires `config_id`, `calendar_id`,
  `start_time`, `end_time`) / `.create` / `.update` / `.delete` / `.import`

### Templates & workflows
A `scope` param selects `"app"` (default) or `"grant"`; only `"grant"` scope requires a grant.
- Templates: `template.list` / `.get` / `.create` / `.update` / `.delete` / `.render` / `.renderHTML`
- Workflows: `workflow.list` / `.get` / `.create` / `.update` / `.delete`

### Admin & workspaces (app-level; no grant)
- Applications: `admin.app.list` / `.get` / `.create` / `.update` / `.delete`
- Callback URIs: `admin.callbackUri.list` / `.get` / `.create` / `.update` / `.delete`
- Connectors: `admin.connector.list` / `.get` / `.create` / `.update` / `.delete`
- **Credentials (secret material):** `admin.credential.list` / `.get` / `.create` / `.update` / `.delete`
- Workspaces: `workspace.list` / `.get` / `.create` / `.update` / `.delete` / `workspace.assignGrants`
- Grants admin: `admin.grants.listAll` / `admin.grants.stats`

### Auth
| Method | Params | Result |
|---|---|---|
| `auth.grant.get` | `grant_id?` | grant |
| `auth.grant.revoke` | `grant_id?` | `{ revoked }` |
| `auth.grant.createCustom` | `provider, settings` | grant |
| `auth.url` | `provider, redirect_uri, state?, code_challenge?` | `{ url }` (pure builder; no API call) |
| `auth.grant.exchange` | `code, redirect_uri, code_verifier?` | grant (completes the OAuth code→grant round-trip) |

> The interactive OAuth login flow (opening a browser + running a local callback server) is **not**
> exposed over RPC. A GUI runs its own redirect, then calls `auth.url` → `auth.grant.exchange`.
> Local CLI session commands (`whoami`, `switch`, `token`, `status`) are intentionally CLI-only.

### Audit (local audit log)
`audit.list` (`limit?`) · `audit.query` (`AuditQueryOptions`) · `audit.summary` (`days?`) ·
`audit.stats` · `audit.config.read` · `audit.config.save` (`{ ok }`) · `audit.path` ·
`audit.clear` (`{ cleared }`) · `audit.cleanup` (`{ ok }`).

### OTP
| Method | Params | Result |
|---|---|---|
| `otp.get` | `email?` (omit → default grant) | `OTPResult` (code/from/subject/received/message_id) |

---

## Notifications (server → client push)

When a default grant is configured, the server runs incremental pollers and pushes notifications
(no `id`) to all connected clients:

| Method | Fires when |
|---|---|
| `message.received` | a new message arrives |
| `thread.updated` | a thread has new activity |
| `event.updated` | a calendar event is created or edited (per calendar) |
| `contact.updated` | a contact is created or its content changes (SHA-256 fingerprint diff) |
| `contact.deleted` | a contact disappears from the address book |

Polling cursors: messages use `received_after`, threads `latest_message_after`, events
`updated_after`; contacts have no server-side time filter so the poller refetches and diffs on a
content fingerprint. Filters are **exclusive**, so pollers query `cursor-1` and dedupe boundary
records by id.

### Adaptive polling

Send a `client.focus` **notification** (no `id`) to scale the poll interval:

```json
{ "jsonrpc": "2.0", "method": "client.focus", "params": { "focused": true } }
```

- `focused: true` → fast interval (default 5s) for message/thread/event pollers.
- `focused: false` → idle interval (default 30s). Contacts poll on their own cadence (default 60s).

To change the interval **values** themselves at runtime, call `client.pollConfig` (a request, not a
notification). All fields are optional Go durations (`"2s"`, `"1m"`); omitted fields are left
unchanged, and the result reports the effective values. `fast`/`idle` drive the message/thread/event
pollers; `contacts` drives the contact poller.

```jsonc
// → request
{ "jsonrpc": "2.0", "id": 9, "method": "client.pollConfig",
  "params": { "fast": "2s", "idle": "45s", "contacts": "90s" } }
// ← result
{ "jsonrpc": "2.0", "id": 9, "result": { "fast": "2s", "idle": "45s", "contacts": "1m30s" } }
```

A non-positive or unparseable duration returns `-32602 invalid params`. Startup defaults come from
the `NYLAS_WS_POLL_*` env vars (see [Configuration](#configuration)).

---

## Examples

Authenticate, list mail, and subscribe to live notifications (pseudocode):

```js
const ws = new WebSocket("ws://127.0.0.1:7369/ws", {
  headers: { Authorization: `Bearer ${token}` },
});

// request / response
ws.send(JSON.stringify({ jsonrpc: "2.0", id: 1, method: "email.list", params: { limit: 10 } }));

// tell the server we're focused → faster polling
ws.send(JSON.stringify({ jsonrpc: "2.0", method: "client.focus", params: { focused: true } }));

ws.onmessage = (e) => {
  const msg = JSON.parse(e.data);
  if (msg.id === 1) console.log("emails:", msg.result.messages);
  else if (msg.method === "message.received") console.log("new mail:", msg.params.subject);
};
```

Send an email (the client confirms first; the server executes immediately):

```json
{ "jsonrpc": "2.0", "id": 2, "method": "email.send",
  "params": { "to": [{ "email": "a@b.com" }], "subject": "Hi", "body": "Hello" } }
```

---

## Testing

| Layer | What |
|---|---|
| Unit (`-race`) | every handler, all pollers (boundary/truncation/fingerprint/deleted/seed), dispatcher, auth, adaptive intervals, WebSocket concurrency |
| Adapter (httptest) | query-building (cursors, filters) |
| Integration (live API) | `make test-integration-rpc` — protocol/auth edge cases, all reads, reversible write round-trips (draft/contact/event create→delete), a live `message.received` round-trip, extended-domain reads |

```bash
make test-integration-rpc    # requires NYLAS_API_KEY + NYLAS_GRANT_ID
```

Integration tests live in `internal/cli/integration/rpc_*_test.go` (build tag `integration`).

---

## Limitations & scope

- **Dashboard is not exposed** — its auth is interactive (login/MFA/SSO) and its app/API-key ops
  return secret material; that needs a separate, dedicated design.
- **Extended-domain writes** (admin/scheduler/template/workflow/notetaker create/update/delete) are
  **unit-tested only**, not live-integration-tested — exercising them creates real app-level or
  secret resources, which isn't safe in a test suite. Their reads are integration-verified.
- **CLI-only conveniences are not exposed:** GPG sign/encrypt, hosted-template rendering shortcuts,
  recipient-string parsing, raw-MIME send, attachment-from-path.
- **No write-path rate limiting** and **no distinct error codes** for not-found vs rate-limited
  (everything upstream maps to `-32603`) — acceptable within the loopback+token threat model; both
  are candidate follow-ups.
- **Token has no rotation/expiry** — rotate by deleting the keyring entry (or changing
  `NYLAS_WS_TOKEN`) and restarting.

---

*Code: `internal/cli/rpc/`, `internal/adapters/rpcserver/`. Tracked in Jira TW-5722 (epic TW-5721).*
