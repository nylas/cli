## Email Operations

Full email management including reading, sending, searching, and organizing.

### List Emails

```bash
nylas email list [grant-id]           # List recent emails
nylas email list --limit 20           # Specify number of emails
nylas email list --unread             # Show only unread
nylas email list --starred            # Show only starred
nylas email list --from "sender@example.com"  # Filter by sender
nylas email list --metadata key1:value  # Filter by metadata (key1-key5 only)
```

**Example output:**
```bash
$ nylas email list --limit 5

Recent Emails
─────────────────────────────────────────────────────
  From: John Doe <john@example.com>
  Subject: Meeting Tomorrow
  Date: 2 hours ago
  ID: msg_abc123

  From: GitHub <noreply@github.com>
  Subject: [repo] New pull request #42
  Date: 5 hours ago
  ID: msg_def456

  From: Newsletter <news@company.com>
  Subject: Weekly Update
  Date: yesterday
  ID: msg_ghi789

Found 5 emails
```

### Read Email

```bash
nylas email read <message-id>         # Read a specific email
nylas email show <message-id>         # Alias for read
nylas email read <id> --mark-read     # Mark as read after reading
```

**Example output:**
```bash
$ nylas email read msg_abc123

From: John Doe <john@example.com>
To: you@example.com
Subject: Meeting Tomorrow
Date: Mon, Dec 16, 2024 2:30 PM

Hi,

Just a reminder about our meeting tomorrow at 10am.
Please bring the quarterly report.

Best,
John

─────────────────────────────────────────────────────
ID: msg_abc123
Thread: thread_xyz789
```

### Send Email

```bash
# Send immediately
nylas email send --to "to@example.com" --subject "Subject" --body "Body"

# Send with CC and BCC
nylas email send --to "to@example.com" --cc "cc@example.com" --bcc "bcc@example.com" \
  --subject "Subject" --body "Body"

# Schedule to send in 2 hours
nylas email send --to "to@example.com" --subject "Reminder" --body "..." --schedule 2h

# Schedule for tomorrow at 9am
nylas email send --to "to@example.com" --subject "Morning" --schedule "tomorrow 9am"

# Schedule for a specific date/time
nylas email send --to "to@example.com" --subject "Meeting" --schedule "2024-12-20 14:30"

# Skip confirmation prompt
nylas email send --to "to@example.com" --subject "Quick" --body "..." --yes

# Send with tracking (opens and link clicks)
nylas email send --to "to@example.com" --subject "Newsletter" --body "..." \
  --track-opens --track-links --track-label "campaign-q4"

# Send with custom metadata
nylas email send --to "to@example.com" --subject "Order Confirmation" --body "..." \
  --metadata "order_id=12345" --metadata "customer_id=cust_abc"
```

**Tracking Options:**
- `--track-opens` - Track when recipients open the email
- `--track-links` - Track when recipients click links in the email
- `--track-label` - Label for grouping tracked emails (for analytics)
- `--metadata` - Custom key=value metadata pairs (can be specified multiple times)

**Example output (scheduled):**
```bash
$ nylas email send --to "user@example.com" --subject "Reminder" --body "Don't forget!" --schedule 2h --yes

Email preview:
  To:      user@example.com
  Subject: Reminder
  Body:    Don't forget!
  Scheduled: Mon Dec 16, 2024 4:30 PM PST

✓ Email scheduled successfully! Message ID: msg_scheduled_123
Scheduled to send: Mon Dec 16, 2024 4:30 PM PST
```

### GPG Signing and Encryption

Sign and/or encrypt emails using GPG/PGP:

```bash
# Sign email with your GPG key
nylas email send --to "to@example.com" --subject "Signed" --body "..." --sign

# Encrypt email with recipient's public key
nylas email send --to "to@example.com" --subject "Encrypted" --body "..." --encrypt

# Sign AND encrypt (recommended for maximum security)
nylas email send --to "to@example.com" --subject "Secure" --body "..." --sign --encrypt

# List available GPG keys
nylas email send --list-gpg-keys
```

**Reading encrypted/signed emails:**

```bash
# Decrypt an encrypted email
nylas email read <message-id> --decrypt

# Verify a signed email
nylas email read <message-id> --verify

# Decrypt and verify (for sign+encrypt emails)
nylas email read <message-id> --decrypt --verify
```

**See also:**
- [GPG Email Signing](email-signing.md) - Detailed signing documentation
- [GPG Email Encryption](encryption.md) - Detailed encryption documentation

### Search Emails

```bash
nylas email search "query"            # Search emails
nylas email search "query" --limit 50 # Search with custom limit
nylas email search "query" --from "sender@example.com"
nylas email search "query" --after "2024-01-01"
nylas email search "query" --before "2024-12-31"
nylas email search "query" --unread   # Only unread messages
nylas email search "query" --starred  # Only starred messages
nylas email search "query" --in INBOX # Search in specific folder
nylas email search "query" --has-attachment  # Only with attachments
```

**Example output:**
```bash
$ nylas email search "invoice" --limit 3

Search Results for "invoice"
─────────────────────────────────────────────────────
  From: Billing <billing@service.com>
  Subject: Your December Invoice
  Date: 3 days ago
  ID: msg_inv001

  From: Accounting <accounting@company.com>
  Subject: Invoice #2024-156 Approved
  Date: 1 week ago
  ID: msg_inv002

Found 3 matching emails
```

### Mark Operations

```bash
nylas email mark read <message-id>      # Mark as read
nylas email mark unread <message-id>    # Mark as unread
nylas email mark starred <message-id>   # Star a message
nylas email mark unstarred <message-id> # Unstar a message
```

### Delete Email

```bash
nylas email delete <message-id>       # Delete an email
nylas email delete <message-id> -f    # Delete without confirmation
```

### Smart Compose (AI Email Generation)

Generate AI-powered email drafts using Nylas Smart Compose (requires Plus package):

```bash
# Generate a new email draft from scratch
nylas email smart-compose --prompt "Draft a thank you email for yesterday's meeting"

# Generate a reply to a specific message
nylas email smart-compose --message-id <msg-id> --prompt "Reply accepting the invitation"

# Output as JSON
nylas email smart-compose --prompt "Write a follow-up email" --json
```

**Features:**
- AI-powered email composition based on natural language prompts
- Context-aware replies to existing messages
- Max prompt length: 1000 tokens
- Requires Nylas Plus package subscription

**Note:** Smart Compose leverages AI to draft professional emails quickly. Always review and edit the generated content before sending.

### Email Tracking

Track email opens, link clicks, and replies via webhooks:

```bash
# View tracking information and setup guide
nylas email tracking-info

# Send an email with tracking enabled
nylas email send --to user@example.com \\
  --subject "Meeting Invite" \\
  --body "Let's schedule a meeting" \\
  --track-opens \\
  --track-links
```

**Tracking Features:**
- **Opens:** Track when recipients open your emails
- **Clicks:** Track when recipients click links in your emails
- **Replies:** Track when recipients reply to your messages

**Data Delivery:**
Tracking data is delivered via webhooks. Set up webhooks to receive notifications:

```bash
# Create webhook for tracking events
nylas webhook create --url https://your-server.com/webhooks \\
  --triggers message.opened,message.link_clicked,thread.replied
```

For detailed information about tracking setup and webhook payloads, run:
```bash
nylas email tracking-info
```

### Message Metadata

Manage custom metadata on messages for organization and filtering:

```bash
# View metadata information and usage guide
nylas email metadata info

# Show metadata for a specific message
nylas email metadata show <message-id>
nylas email metadata show <message-id> --json

# Filter messages by metadata (when listing)
nylas email list --metadata key1:project-alpha
nylas email list --metadata key2:urgent --limit 20
```

**Indexed Keys (Searchable):**
Only five keys support filtering in queries:
- `key1`, `key2`, `key3`, `key4`, `key5`

**Setting Metadata:**
Metadata can only be set when sending messages or creating drafts:

```bash
# Send with metadata
nylas email send --to user@example.com \\
  --subject "Project Update" \\
  --body "Status report" \\
  --metadata key1=project-alpha \\
  --metadata key2=status-update
```

**Features:**
- Store up to 50 custom key-value pairs per message
- Only `key1`-`key5` are indexed and searchable
- Cannot update metadata on existing messages
- Useful for categorization, tracking, and custom workflows

**Example filtering workflow:**
```bash
# Send emails with metadata tags
nylas email send --to team@company.com \\
  --subject "Sprint Planning" \\
  --metadata key1=sprint-23 \\
  --metadata key2=planning

# Later, filter by metadata
nylas email list --metadata key1:sprint-23
nylas email list --metadata key2:planning --unread
```

For detailed information about metadata usage and best practices, run:
```bash
nylas email metadata info
```

---

## Workflows & Automation

### Daily Email Check

```bash
#!/bin/bash
# morning-email-check.sh

echo "=== Morning Email Summary ==="

# Count unread emails
unread=$(nylas email list --unread | grep -c "From:")
echo "Unread emails: $unread"

# Check for urgent/important
urgent=$(nylas email list --unread | grep -ci "urgent\|important\|asap")
echo "Urgent emails: $urgent"

# List recent unread
echo ""
echo "Recent unread emails:"
nylas email list --unread --limit 10
```

---

### Bulk Email Operations

```bash
#!/bin/bash
# bulk-send.sh

# Read emails from file (one per line)
while IFS= read -r email; do
  echo "Sending to: $email"

  nylas email send \
    --to "$email" \
    --subject "Newsletter - $(date +%B\ %Y)" \
    --body "Dear subscriber, ..." \
    --yes

  sleep 2  # Rate limiting
done < email-list.txt
```

---

### Scheduled Email Automation

```bash
# Send in 2 hours
nylas email send \
  --to "team@company.com" \
  --subject "Meeting Reminder" \
  --body "Reminder: Team meeting in 1 hour" \
  --schedule 2h

# Send tomorrow morning
nylas email send \
  --to "team@company.com" \
  --subject "Daily Standup" \
  --body "Good morning! Today's standup at 9:30 AM" \
  --schedule "tomorrow 9am"
```

---

### Customer Onboarding Automation

```bash
#!/bin/bash
# customer-onboarding.sh

CUSTOMER_EMAIL="$1"
CUSTOMER_NAME="$2"

# Day 1: Welcome email
nylas email send \
  --to "$CUSTOMER_EMAIL" \
  --subject "Welcome to Our Service!" \
  --body "Hi $CUSTOMER_NAME, welcome aboard!" \
  --yes

# Day 3: Getting started tips (scheduled)
nylas email send \
  --to "$CUSTOMER_EMAIL" \
  --subject "Getting Started Tips" \
  --body "Hi $CUSTOMER_NAME, here are some tips..." \
  --schedule "3 days" \
  --yes

# Day 7: Check-in (scheduled)
nylas email send \
  --to "$CUSTOMER_EMAIL" \
  --subject "How's It Going?" \
  --body "Hi $CUSTOMER_NAME, any questions?" \
  --schedule "7 days" \
  --yes
```

---

### Email Notification Monitor

```bash
#!/bin/bash
# email-monitor.sh

while true; do
  urgent=$(nylas email list --unread | grep -ci "urgent\|asap\|important")

  if [ $urgent -gt 0 ]; then
    # macOS notification
    osascript -e "display notification \"You have $urgent urgent emails\" with title \"Email Alert\""
  fi

  sleep 300  # Check every 5 minutes
done
```

---

### Best Practices

**Rate limiting:**
```bash
for email in "${emails[@]}"; do
  nylas email send --to "$email" --subject "..." --body "..."
  sleep 2  # Wait between sends
done
```

**Error handling:**
```bash
if nylas email send --to "user@example.com" --subject "Test" --body "Test"; then
  echo "Email sent successfully"
else
  echo "Failed to send email" >&2
  exit 1
fi
```

---

