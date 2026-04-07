## Email Templates

Manage locally-stored email templates for composing messages with variable substitution.

Templates are stored in `~/.config/nylas/templates.json` and support `{{variable}}` syntax for placeholders. Variables are automatically extracted from the subject and body when templates are created or updated.

For API-backed hosted templates shared across an application or grant, use the top-level `nylas template ...` commands.

---

### Hosted Templates Quick Reference

```bash
nylas template list
nylas template create --name "Welcome" --subject "Hello {{user.name}}" --body "<p>Hello {{user.name}}</p>"
nylas template render <template-id> --data '{"user":{"name":"Ada"}}'
nylas email send --to user@example.com --template-id <template-id> --template-data '{"user":{"name":"Ada"}}'
```

---

### List Templates

```bash
nylas email templates list
nylas email templates list --category sales
nylas email templates list --json
```

**Example output:**
```bash
$ nylas email templates list

EMAIL TEMPLATES
Storage: /home/user/.config/nylas/templates.json

  tpl_170012345...  Welcome Email              Welcome {{name}}!                    onboarding (2 vars)
  tpl_170012346...  Sales Follow-up            Following up on {{topic}}            sales (3 vars) [used 5x]
  tpl_170012347...  Meeting Request            Request for meeting                  - (0 vars)

Total: 3 template(s)
```

**Example with category filter:**
```bash
$ nylas email templates list --category sales

EMAIL TEMPLATES
Category: sales
Storage: /home/user/.config/nylas/templates.json

  tpl_170012346...  Sales Follow-up            Following up on {{topic}}            sales (3 vars) [used 5x]

Total: 1 template(s)
```

**Available flags:**
- `--category, -c` - Filter templates by category
- `--json` - Output as JSON
- `--quiet, -q` - Only output template IDs

---

### Show Template

```bash
nylas email templates show <template-id>
nylas email templates show <template-id> --json
```

**Example output:**
```bash
$ nylas email templates show tpl_1700123456789

────────────────────────────────────────────────────────────
Template: Welcome Email
────────────────────────────────────────────────────────────
ID:         tpl_1700123456789
Name:       Welcome Email
Category:   onboarding
Created:    Jan 15, 2026 10:30 AM
Updated:    Jan 20, 2026 2:45 PM
Used:       12 time(s)
Variables:  name, company

Subject:
  Welcome to {{company}}, {{name}}!

Body:
  Hi {{name}},

  Welcome to {{company}}! We're excited to have you on board.

  Best regards,
  The Team

────────────────────────────────────────────────────────────
Usage:
  nylas email templates use tpl_1700123456789 --to <email> --var name=<value> --var company=<value>
```

---

### Create Template

```bash
nylas email templates create --name "Template Name" --subject "Subject" --body "Body"
nylas email templates create --name "Welcome" --subject "Hello {{name}}!" --body "Hi {{name}}" --category onboarding
nylas email templates create --interactive
```

**Example output:**
```bash
$ nylas email templates create \
    --name "Sales Follow-up" \
    --subject "Following up on {{topic}}" \
    --body "Hi {{name}},\n\nWanted to follow up on our conversation about {{topic}}.\n\nLet me know if you have any questions.\n\nBest,\n{{sender}}" \
    --category sales

✓ Template created successfully!

  ID:        tpl_1700123456789
  Name:      Sales Follow-up
  Subject:   Following up on {{topic}}
  Category:  sales
  Variables: name, topic, sender

Use this template with:
  nylas email templates use tpl_1700123456789 --to recipient@example.com --var name=<value> --var topic=<value> --var sender=<value>
```

**Available flags:**
- `--name, -n` - Template name (required)
- `--subject, -s` - Email subject, supports `{{variables}}`
- `--body, -b` - Email body, supports `{{variables}}`
- `--category, -c` - Template category (e.g., sales, support, marketing)
- `--interactive, -i` - Interactive mode with prompts

**Interactive mode:**
```bash
$ nylas email templates create --interactive

Create a new email template
Use {{variable}} syntax for placeholders

Template name: Welcome Email
Subject (supports {{variables}}): Welcome {{name}}!
Body (end with a line containing only '.'):
Hi {{name}},

Welcome to our platform!

Best regards
.
Category (optional, press Enter to skip): onboarding

✓ Template created successfully!
```

---

### Update Template

```bash
nylas email templates update <template-id> --name "New Name"
nylas email templates update <template-id> --subject "Updated subject"
nylas email templates update <template-id> --body "Updated body"
nylas email templates update <template-id> --category marketing
```

**Example output:**
```bash
$ nylas email templates update tpl_1700123456789 --name "Sales Outreach" --category outbound

✓ Template updated successfully!

  ID:        tpl_1700123456789
  Name:      Sales Outreach
  Subject:   Following up on {{topic}}
  Category:  outbound
  Variables: name, topic, sender
  Updated:   Jan 25, 2026 3:30 PM
```

**Available flags:**
- `--name, -n` - New template name
- `--subject, -s` - New email subject
- `--body, -b` - New email body
- `--category, -c` - New template category

Variables are automatically re-extracted when subject or body is updated.

---

### Delete Template

```bash
nylas email templates delete <template-id>
nylas email templates delete <template-id> --force
```

**Example output (with confirmation):**
```bash
$ nylas email templates delete tpl_1700123456789

Delete template "Sales Follow-up"?
  ID:      tpl_1700123456789
  Subject: Following up on {{topic}}
  Used:    5 time(s)

Are you sure? [y/N]: y
✓ Template "Sales Follow-up" deleted successfully
```

**Example output (force delete):**
```bash
$ nylas email templates delete tpl_1700123456789 --force

✓ Template "Sales Follow-up" deleted successfully
```

**Available flags:**
- `--force, -f` - Skip confirmation prompt

---

### Use Template

Use a template to compose and send an email with variable substitution.

```bash
nylas email templates use <template-id> --to user@example.com --var name=John
nylas email templates use <template-id> --to user@example.com --var name=John --var company=Acme
nylas email templates use <template-id> --to user@example.com --var name=John --preview
```

**Example with preview:**
```bash
$ nylas email templates use tpl_1700123456789 \
    --to alice@example.com \
    --var name=Alice \
    --var topic="Q4 Planning" \
    --var sender=Bob \
    --preview

────────────────────────────────────────────────────────────
TEMPLATE PREVIEW: Sales Follow-up
────────────────────────────────────────────────────────────
To:      alice@example.com
Subject: Following up on Q4 Planning

Body:
────────────────────────────────────────
Hi Alice,

Wanted to follow up on our conversation about Q4 Planning.

Let me know if you have any questions.

Best,
Bob
────────────────────────────────────────

This is a preview. Remove --preview to send the email.
```

**Example sending email:**
```bash
$ nylas email templates use tpl_1700123456789 \
    --to alice@example.com \
    --var name=Alice \
    --var topic="Q4 Planning" \
    --var sender=Bob

Email preview:
  Template: Sales Follow-up
  To:       alice@example.com
  Subject:  Following up on Q4 Planning
  Body:     Hi Alice, Wanted to follow up on our conversa...

Send this email? [y/N]: y
Sending email...
✓ Email sent successfully! Message ID: msg_abc123xyz
```

**Example with missing variables:**
```bash
$ nylas email templates use tpl_1700123456789 --to alice@example.com --var name=Alice

✗ Error: missing variable(s): topic, sender
Provide values with: --var topic=<value> --var sender=<value>
```

**Available flags:**
- `--to, -t` - Recipient email addresses (required, can be repeated)
- `--cc` - CC email addresses (can be repeated)
- `--bcc` - BCC email addresses (can be repeated)
- `--var` - Variable values as `key=value` (can be repeated)
- `--preview, -p` - Preview the expanded template without sending
- `--yes, -y` - Skip confirmation prompt
- `--json` - Output result as JSON

**Multiple recipients:**
```bash
$ nylas email templates use tpl_1700123456789 \
    --to alice@example.com \
    --to bob@example.com \
    --cc manager@example.com \
    --var name=Team \
    --var topic="Project Update" \
    --var sender="The Team"
```

---

### Variable Syntax

Templates support `{{variable}}` placeholders in both subject and body:

```
Subject: Welcome to {{company}}, {{name}}!
Body: Hi {{name}}, we're glad to have you at {{company}}.
```

**Variable rules:**
- Variables are enclosed in double curly braces: `{{variable}}`
- Variable names can contain letters, numbers, and underscores
- Whitespace inside braces is trimmed: `{{ name }}` works the same as `{{name}}`
- Variables are case-sensitive: `{{Name}}` and `{{name}}` are different
- Duplicate variables only need to be provided once

**When creating or updating a template, variables are automatically extracted:**
```bash
$ nylas email templates create --name "Test" --subject "Hello {{first_name}}" --body "Dear {{first_name}} {{last_name}}"

✓ Template created successfully!
  Variables: first_name, last_name
```

---

### Storage Location

Templates are stored locally in a JSON file:

- **Default location:** `~/.config/nylas/templates.json`
- **With XDG_CONFIG_HOME:** `$XDG_CONFIG_HOME/nylas/templates.json`

**File format:**
```json
{
  "templates": [
    {
      "id": "tpl_1700123456789",
      "name": "Welcome Email",
      "subject": "Welcome {{name}}!",
      "html_body": "Hi {{name}}, welcome to {{company}}!",
      "variables": ["name", "company"],
      "category": "onboarding",
      "usage_count": 5,
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-20T14:45:00Z"
    }
  ]
}
```

Templates are local to your machine and are not synced with Nylas API.

---

### Common Use Cases

**Customer onboarding:**
```bash
nylas email templates create \
  --name "Customer Welcome" \
  --subject "Welcome to {{product}}, {{name}}!" \
  --body "Hi {{name}},\n\nThank you for choosing {{product}}. Your account is ready.\n\nGetting started: {{getting_started_url}}\n\nBest,\n{{sender}}" \
  --category onboarding
```

**Meeting follow-up:**
```bash
nylas email templates create \
  --name "Meeting Follow-up" \
  --subject "Great meeting today, {{name}}!" \
  --body "Hi {{name}},\n\nThank you for taking the time to meet with me today about {{topic}}.\n\nAs discussed, I'll {{next_steps}}.\n\nLet me know if you have any questions.\n\nBest,\n{{sender}}" \
  --category meetings
```

**Weekly update:**
```bash
nylas email templates create \
  --name "Weekly Update" \
  --subject "Weekly Update - {{week}}" \
  --body "Team,\n\nHere's this week's update:\n\n{{highlights}}\n\nNext week focus:\n{{next_week}}\n\nQuestions? Reply to this email.\n\n{{sender}}" \
  --category internal
```

---
