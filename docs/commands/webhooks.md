## Webhook Management

Create and manage webhooks for real-time event notifications.

### Built-in Webhook Server

Start a local webhook server for development and testing:

```bash
# Start server (local only)
nylas webhook server

# Start with public tunnel (cloudflared required)
nylas webhook server --tunnel cloudflared

# Custom port
nylas webhook server --port 8080 --tunnel cloudflared
```

**Install cloudflared:**
```bash
brew install cloudflared                    # macOS
# Or download from: github.com/cloudflare/cloudflared
```

---

### List Webhooks

```bash
nylas webhook list
nylas webhook list --full-ids         # Show full webhook IDs (for copy/paste)
nylas webhook list --format json
nylas webhook list --format yaml
nylas webhook list --format csv
```

**Example output:**
```bash
$ nylas webhook list

ID                    Description              URL                                    Status    Triggers
────────────────────────────────────────────────────────────────────────────────────────────────────────────
webhook_abc123        Message notifications    https://api.myapp.com/webhooks/nylas   ● active  message.created, message.updated
webhook_def456        Calendar sync            https://api.myapp.com/calendar         ● active  event.created, event.updated
webhook_ghi789        Contact updates          https://api.myapp.com/contacts         ○ inactive contact.created

Total: 3 webhooks
```

### Show Webhook

```bash
nylas webhook show <webhook-id>
nylas webhook show <webhook-id> --format json
```

**Example output:**
```bash
$ nylas webhook show webhook_abc123

Webhook: webhook_abc123
────────────────────────────────────────────────────────────
Description:  Message notifications
URL:          https://api.myapp.com/webhooks/nylas
Status:       ● active
Secret:       wh_s****************************cret

Trigger Types:
  • message.created
  • message.updated
  • message.opened

Notification Emails:
  • admin@myapp.com

Timestamps:
  Created:        2024-12-01T10:00:00Z
  Updated:        2024-12-15T14:30:00Z
  Status Updated: 2024-12-15T14:30:00Z
```

### Create Webhook

```bash
# Create with required fields
nylas webhook create --url https://example.com/webhook --triggers message.created

# Create with multiple triggers
nylas webhook create --url https://example.com/webhook \
  --triggers message.created,event.created,contact.created

# Create with description and notification email
nylas webhook create --url https://example.com/webhook \
  --triggers message.created \
  --description "My message webhook" \
  --notify admin@example.com
```

**Example output:**
```bash
$ nylas webhook create --url https://api.myapp.com/webhook --triggers message.created --description "New messages"

✓ Webhook created successfully!

  ID:     webhook_new_123
  URL:    https://api.myapp.com/webhook
  Status: active

Triggers:
  • message.created

Important: Save your webhook secret - it won't be shown again:
  Secret: wh_secret_abc123xyz789

Use this secret to verify webhook signatures.
```

### Update Webhook

```bash
# Update URL
nylas webhook update <webhook-id> --url https://new.example.com/webhook

# Update triggers
nylas webhook update <webhook-id> --triggers message.created,message.updated

# Pause/resume webhook
nylas webhook update <webhook-id> --status inactive
nylas webhook update <webhook-id> --status active

# Update multiple properties
nylas webhook update <webhook-id> \
  --description "Updated webhook" \
  --triggers event.created,event.updated
```

**Example output:**
```bash
$ nylas webhook update webhook_abc123 --status inactive

✓ Webhook updated successfully!

  ID:     webhook_abc123
  URL:    https://api.myapp.com/webhooks/nylas
  Status: ○ inactive

Triggers:
  • message.created
  • message.updated
```

### Delete Webhook

```bash
nylas webhook delete <webhook-id>
nylas webhook delete <webhook-id> --force   # Skip confirmation
```

**Example output:**
```bash
$ nylas webhook delete webhook_abc123

Webhook to delete:
  ID:  webhook_abc123
  URL: https://api.myapp.com/webhooks/nylas
  Description: Message notifications
  Triggers: [message.created message.updated]

Are you sure you want to delete this webhook? [y/N] y
✓ Webhook deleted successfully!
```

### List Trigger Types

```bash
nylas webhook triggers
nylas webhook triggers --format json
nylas webhook triggers --format list
nylas webhook triggers --category message   # Filter by category
nylas webhook triggers --category notetaker # Filter by notetaker
```

**Example output:**
```bash
$ nylas webhook triggers

Available Webhook Trigger Types
================================

🔑 Grant
   Authentication grant events

   • grant.created
   • grant.updated
   • grant.deleted
   • grant.expired
   • grant.imap_sync_completed

📧 Message
   Email message events

   • message.created
   • message.updated
   • message.opened
   • message.bounce_detected
   • message.send_success
   • message.send_failed
   • message.opened.truncated
   • message.link_clicked

💬 Thread
   Email thread events

   • thread.replied

📅 Event
   Calendar event events

   • event.created
   • event.updated
   • event.deleted

👤 Contact
   Contact events

   • contact.created
   • contact.updated
   • contact.deleted

📆 Calendar
   Calendar events

   • calendar.created
   • calendar.updated
   • calendar.deleted

📁 Folder
   Email folder events

   • folder.created
   • folder.updated
   • folder.deleted

📝 Notetaker
   Meeting notetaker events

   • notetaker.media

Usage:
  nylas webhook create --url <URL> --triggers message.created
  nylas webhook create --url <URL> --triggers message.created,event.created
```

### Test Webhook

```bash
# Send a test event to a URL
nylas webhook test send https://example.com/webhook

# Get mock payload for a trigger type
nylas webhook test payload message.created
nylas webhook test payload event.created --format json
```

**Example output:**
```bash
$ nylas webhook test send https://api.myapp.com/webhook

✓ Test event sent successfully!

  URL: https://api.myapp.com/webhook

Check your webhook endpoint logs to verify the event was received.
```

---

## Local Development

### Using ngrok for local testing:

```bash
# 1. Install ngrok
brew install ngrok

# 2. Start local webhook server
python webhook-server.py &

# 3. Create ngrok tunnel
ngrok http 8080

# 4. Create webhook with ngrok URL
nylas webhook create \
  --url "https://abc123.ngrok.io/webhook" \
  --triggers "message.created"
```

---

### Simple webhook server (Python):

```python
#!/usr/bin/env python3
# webhook-server.py

from http.server import BaseHTTPRequestHandler, HTTPServer
import json

class WebhookHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        body = self.rfile.read(content_length)
        data = json.loads(body)

        print(f"Received: {data.get('trigger')}")

        # Handle different event types
        if data.get('trigger') == 'message.created':
            self.handle_new_message(data)

        self.send_response(200)
        self.end_headers()

    def handle_new_message(self, data):
        print("New message received!")

if __name__ == '__main__':
    HTTPServer(('', 8080), WebhookHandler).serve_forever()
```

---

### Simple webhook server (Node.js):

```javascript
const express = require('express');
const app = express();

app.use(express.json());

app.post('/webhook', (req, res) => {
  const { trigger, data } = req.body;

  switch (trigger) {
    case 'message.created':
      console.log('New message:', data);
      break;
    case 'calendar.created':
      console.log('New event:', data);
      break;
  }

  res.json({ status: 'ok' });
});

app.listen(8080, () => console.log('Webhook server on port 8080'));
```

---

## Integration Examples

### Production webhook server (Flask):

```python
from flask import Flask, request, jsonify
import hmac
import hashlib
import os

app = Flask(__name__)
WEBHOOK_SECRET = os.environ.get('WEBHOOK_SECRET', '')

def verify_webhook(request):
    if not WEBHOOK_SECRET:
        return True
    signature = request.headers.get('X-Nylas-Signature', '')
    expected = hmac.new(
        WEBHOOK_SECRET.encode(),
        request.get_data(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)

@app.route('/webhook', methods=['POST'])
def webhook():
    if not verify_webhook(request):
        return jsonify({"error": "Invalid signature"}), 401

    data = request.get_json()
    trigger = data.get('trigger')

    handlers = {
        'message.created': handle_message,
        'calendar.created': handle_event,
    }

    if trigger in handlers:
        handlers[trigger](data)

    return jsonify({"status": "ok"})

def handle_message(data):
    print("New message")

def handle_event(data):
    print("New event")

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
```

---

## Best Practices

**Security:**
1. Always verify webhook signatures
2. Use HTTPS for production webhooks
3. Implement rate limiting on your endpoint

**Performance:**
1. Respond with 200 OK quickly, process async
2. Queue webhooks for background processing
3. Monitor for failures and latency

**Debugging:**
```bash
# Test webhook locally
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"trigger": "message.created", "data": {}}'

# Test with Nylas CLI
nylas webhook test <webhook-id>
```

---

