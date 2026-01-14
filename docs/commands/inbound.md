## Inbound Email Management

Manage Nylas Inbound email inboxes for receiving emails at managed addresses.

Nylas Inbound enables your application to receive emails at dedicated managed addresses (e.g., `support@yourapp.nylas.email`) without requiring OAuth authentication or third-party mailbox connections.

**Use cases:**
- Capturing messages sent to specific addresses (intake@, leads@, tickets@)
- Triggering automated workflows from incoming mail
- Real-time message delivery to workers, LLMs, or downstream systems

**Aliases:** `nylas inbox`

---

### List Inbound Inboxes

```bash
nylas inbound list
nylas inbound list --json
```

**Example output:**
```bash
$ nylas inbound list

Inbound Inboxes (3)

1. support@yourapp.nylas.email
   ID:      inbox_abc123
   Status:  active
   Created: 2024-12-01T10:00:00Z

2. leads@yourapp.nylas.email
   ID:      inbox_def456
   Status:  active
   Created: 2024-12-10T14:30:00Z

3. tickets@yourapp.nylas.email
   ID:      inbox_ghi789
   Status:  active
   Created: 2024-12-15T09:15:00Z

Use 'nylas inbound messages [inbox-id]' to view messages
```

---

### Show Inbound Inbox

```bash
nylas inbound show <inbox-id>
nylas inbound show <inbox-id> --json

# Use environment variable for inbox ID
export NYLAS_INBOUND_GRANT_ID=inbox_abc123
nylas inbound show
```

**Example output:**
```bash
$ nylas inbound show inbox_abc123

Inbound Inbox: inbox_abc123
────────────────────────────────────────────────────────────
Email:    support@yourapp.nylas.email
ID:       inbox_abc123
Status:   active
Created:  2024-12-01T10:00:00Z
Updated:  2024-12-15T14:30:00Z

Configuration:
  Domain:     yourapp.nylas.email
  Prefix:     support
  Forwarding: enabled

Statistics:
  Messages Received: 142
  Last Message:      2024-12-20T16:45:00Z
```

---

### Create Inbound Inbox

```bash
# Create a support inbox
nylas inbound create support
# Creates: support@yourapp.nylas.email

# Create a leads inbox
nylas inbound create leads
# Creates: leads@yourapp.nylas.email

# Create and output as JSON
nylas inbound create tickets --json
```

**Example output:**
```bash
$ nylas inbound create support

✓ Inbound inbox created successfully!

Inbound Inbox: inbox_new_123
────────────────────────────────────────────────────────────
Email:    support@yourapp.nylas.email
ID:       inbox_new_123
Status:   active
Created:  2024-12-20T10:30:00Z

Next steps:
  1. Set up a webhook: nylas webhooks create --url <your-url> --triggers message.created
  2. View messages: nylas inbound messages inbox_new_123
  3. Monitor in real-time: nylas inbound monitor inbox_new_123
```

---

### Delete Inbound Inbox

```bash
# Delete with confirmation
nylas inbound delete <inbox-id>

# Delete without confirmation
nylas inbound delete <inbox-id> --yes
nylas inbound delete <inbox-id> --force

# Use environment variable for inbox ID
export NYLAS_INBOUND_GRANT_ID=inbox_abc123
nylas inbound delete --yes
```

**Example output:**
```bash
$ nylas inbound delete inbox_abc123

You are about to delete the inbound inbox:
  Email: support@yourapp.nylas.email
  ID:    inbox_abc123

This action cannot be undone. All messages in this inbox will be deleted.

Type 'delete' to confirm: delete
✓ Inbox support@yourapp.nylas.email deleted successfully!
```

---

### List Messages

```bash
# List messages for an inbox
nylas inbound messages <inbox-id>

# List only unread messages
nylas inbound messages <inbox-id> --unread

# Limit to 5 messages
nylas inbound messages <inbox-id> --limit 5

# Output as JSON
nylas inbound messages <inbox-id> --json

# Use environment variable for inbox ID
export NYLAS_INBOUND_GRANT_ID=inbox_abc123
nylas inbound messages
```

**Example output:**
```bash
$ nylas inbound messages inbox_abc123

Messages (10 total, 3 unread)

1. [NEW] Feature Request: Dark Mode
   From:    alice@example.com
   Date:    2024-12-20 16:45
   Preview: Hi team, I'd like to request a dark mode feature for the app...
   ID:      msg_xyz123

2. [NEW] Bug Report: Login Issue
   From:    bob@company.com
   Date:    2024-12-20 15:30
   Preview: I'm experiencing issues logging in on mobile devices...
   ID:      msg_abc456

3. Integration Question
   From:    charlie@startup.io
   Date:    2024-12-20 14:15
   Preview: Does your API support webhook retries? We need to ensure...
   ID:      msg_def789

Use 'nylas email read <message-id> [inbox-id]' to view full message
```

---

### Monitor for New Messages

Start a local webhook server to receive real-time notifications when new emails arrive.

```bash
# Start monitoring with default settings
nylas inbound monitor <inbox-id>

# Monitor with cloudflared tunnel (for public access)
nylas inbound monitor <inbox-id> --tunnel cloudflared

# Monitor on custom port
nylas inbound monitor <inbox-id> --port 8080

# Output events as JSON
nylas inbound monitor <inbox-id> --tunnel cloudflared --json

# Quiet mode (only show events)
nylas inbound monitor <inbox-id> --quiet

# Use environment variable for inbox ID
export NYLAS_INBOUND_GRANT_ID=inbox_abc123
nylas inbound monitor --tunnel cloudflared
```

**Example output:**
```bash
$ nylas inbound monitor inbox_abc123 --tunnel cloudflared

╔══════════════════════════════════════════════════════════════╗
║            Nylas Inbound Monitor                             ║
╚══════════════════════════════════════════════════════════════╝

Monitoring: support@yourapp.nylas.email

Starting tunnel...
✓ Monitor started successfully!

  Local URL:    http://localhost:3000/webhook
  Public URL:   https://random-words.trycloudflare.com/webhook

  Tunnel:       cloudflared (connected)

To receive events, register this webhook URL with Nylas:
  nylas webhooks create --url https://random-words.trycloudflare.com/webhook --triggers message.created

Press Ctrl+C to stop

─────────────────────────────────────────────────────────────────
Incoming Messages:

[16:45:32] NEW MESSAGE [verified]
  Subject: Feature Request: Dark Mode
  From: Alice Smith <alice@example.com>
  Preview: Hi team, I'd like to request a dark mode feature...
  ID: msg_xyz123

[16:48:15] NEW MESSAGE [verified]
  Subject: Urgent: Production Issue
  From: bob@company.com
  Preview: We're seeing errors in production...
  ID: msg_abc456
```

**Tunnel providers:**
- `cloudflared` - Cloudflare Tunnel (requires `cloudflared` installed)

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--port` | `-p` | Port to listen on (default: 3000) |
| `--tunnel` | `-t` | Tunnel provider (cloudflared) |
| `--secret` | `-s` | Webhook secret for signature verification |
| `--json` | | Output events as JSON |
| `--quiet` | `-q` | Suppress startup messages, only show events |

---

### Environment Variables

| Variable | Description |
|----------|-------------|
| `NYLAS_INBOUND_GRANT_ID` | Default inbox ID for commands |

---

### Workflow Example

Complete workflow for setting up inbound email processing:

```bash
# 1. Create an inbound inbox
nylas inbound create support
# → Creates support@yourapp.nylas.email

# 2. Start monitoring (in another terminal)
nylas inbound monitor inbox_abc123 --tunnel cloudflared
# → Provides public webhook URL

# 3. Register the webhook URL
nylas webhook create --url https://random-words.trycloudflare.com/webhook \
  --triggers message.created

# 4. Send a test email to support@yourapp.nylas.email
# → Watch the monitor for incoming message

# 5. View messages
nylas inbound messages inbox_abc123

# 6. Read a specific message
nylas email read msg_xyz123 inbox_abc123
```

---

