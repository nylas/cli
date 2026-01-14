# Email Troubleshooting

Comprehensive guide for resolving email-related issues.

---

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Common Issues](#common-issues)
- [No Emails Showing](#no-emails-showing)
- [Send Failures](#send-failures)
- [Search Issues](#search-issues)
- [Scheduled Email Issues](#scheduled-email-issues)
- [Attachment Problems](#attachment-problems)
- [Performance Issues](#performance-issues)

---

## Quick Diagnostics

```bash
# Test basic email listing
nylas email list --limit 1

# Check grant permissions
nylas admin grants show <grant-id>

# Verify API connectivity
nylas doctor
```

---

## Common Issues

### Issue: No emails showing / Empty list

**Symptoms:**
- `nylas email list` returns empty
- "Found 0 emails" message
- All filters return no results

**Solutions:**

1. **Verify Grant ID is correct:**
```bash
# Check configured grant
nylas auth status

# List available grants
nylas admin grants list

# Try specific grant
nylas email list grant_xxx...

# Reconfigure with correct grant
nylas auth config
```

2. **Check email provider connection:**
```bash
# Verify grant status
nylas admin grants show <grant-id>

# Status should be "active"
# If "invalid" or "revoked", need to re-authenticate

# Re-authenticate
nylas auth login
```

3. **Verify email scope/permissions:**
```bash
# Check grant scopes
nylas admin grants show <grant-id>

# Should include: "email.read" or "email.read_only"
# If missing, re-authorize with correct scopes
```

4. **Try different filters:**
```bash
# Remove filters
nylas email list --limit 10

# Try different time ranges
nylas email list --limit 100

# Check specific folder
nylas email list --in INBOX
```

---

### Issue: Email send fails

**Symptoms:**
- Error sending email
- "Failed to send" message
- Email stuck in drafts

**Solutions:**

1. **Verify send permissions:**
```bash
# Check grant scopes
nylas admin grants show <grant-id>

# Should include: "email.send" or "email.modify"
```

2. **Check recipient format:**
```bash
# ✅ Correct format
nylas email send --to "user@example.com" --subject "Test" --body "Hello"

# ✅ Multiple recipients
nylas email send --to "user1@example.com,user2@example.com" --subject "Test"

# ❌ Incorrect - missing quotes
nylas email send --to user@example.com

# ❌ Incorrect - invalid email
nylas email send --to "not-an-email"
```

3. **Check required fields:**
```bash
# All required: --to, --subject, --body
nylas email send \
  --to "recipient@example.com" \
  --subject "Subject line" \
  --body "Message body"

# Optional: --cc, --bcc
nylas email send \
  --to "to@example.com" \
  --cc "cc@example.com" \
  --bcc "bcc@example.com" \
  --subject "Subject" \
  --body "Body"
```

4. **Provider-specific limits:**
```bash
# Some providers have sending limits
# Gmail: 500/day (free), 2000/day (workspace)
# Outlook: 300/day (free), 10000/day (365)

# Check provider status in Nylas Dashboard
# Verify you haven't exceeded daily limit
```

5. **Send with minimal options:**
```bash
# Test with simplest command
nylas email send \
  --to "yourself@example.com" \
  --subject "Test" \
  --body "Test message"

# If this works, add options incrementally
```

---

### Issue: Scheduled email not sending

**Symptoms:**
- Scheduled email doesn't send at specified time
- Email stuck in "scheduled" state
- No error message

**Solutions:**

1. **Verify schedule format:**
```bash
# ✅ Correct formats
nylas email send --to "user@example.com" --subject "Test" --schedule "2h"
nylas email send --to "user@example.com" --subject "Test" --schedule "tomorrow 9am"
nylas email send --to "user@example.com" --subject "Test" --schedule "2024-12-25 10:00"

# ❌ Incorrect - past time
nylas email send --to "user@example.com" --subject "Test" --schedule "yesterday"

# ❌ Incorrect - invalid format
nylas email send --to "user@example.com" --subject "Test" --schedule "next week"
```

2. **Check scheduled emails:**
```bash
# List drafts/scheduled messages
nylas drafts list

# Check if email is in drafts
# Scheduled emails appear as drafts until sent
```

3. **Provider support:**
```bash
# Not all providers support scheduled sending
# Gmail: ✅ Supported
# Outlook: ✅ Supported
# Some providers: ❌ Not supported

# Check Nylas Dashboard for provider capabilities
```

4. **Timezone considerations:**
```bash
# Schedule time is in YOUR timezone by default
# Verify timezone is correct

# Be explicit with timezone
nylas email send \
  --to "user@example.com" \
  --subject "Meeting" \
  --schedule "2024-12-25 10:00 America/New_York"
```

---

### Issue: Search not finding emails

**Symptoms:**
- Search returns no results
- Known emails not appearing
- Filters not working

**Solutions:**

1. **Verify search syntax:**
```bash
# ✅ Correct search
nylas email list --from "sender@example.com"
nylas email list --subject "Meeting"
nylas email list --unread
nylas email list --starred

# Search is case-insensitive
nylas email list --from "SENDER@EXAMPLE.COM"  # Works
```

2. **Check filter combinations:**
```bash
# Multiple filters work together (AND logic)
nylas email list --from "sender@example.com" --unread

# Use --limit to see more results
nylas email list --from "sender@example.com" --limit 100
```

3. **Metadata search:**
```bash
# Metadata uses exact key:value format
nylas email list --metadata "order_id:12345"

# Only works with metadata keys: key1-key5
# Custom keys must be set when sending
```

4. **Provider search limitations:**
```bash
# Some providers have search delays
# New emails may take 1-2 minutes to be searchable

# Try without filters first
nylas email list --limit 10
```

---

## No Emails Showing

### Checklist:

- [ ] Grant ID is correct (`nylas auth status`)
- [ ] Grant is active (`nylas admin grants show <grant-id>`)
- [ ] Email scope is enabled (check grant scopes)
- [ ] Email provider is connected (check Nylas Dashboard)
- [ ] Trying without filters (`nylas email list --limit 10`)
- [ ] Increase limit (`--limit 100`)

### Provider-specific issues:

**Gmail:**
- Check if IMAP is enabled in Gmail settings
- Verify "Less secure app access" if using OAuth
- Check if 2FA is blocking access

**Outlook:**
- Verify account is active
- Check if legacy auth is disabled
- Ensure modern auth is enabled

**Exchange:**
- Verify Exchange server connectivity
- Check firewall/network access
- Confirm EWS is enabled

---

## Send Failures

### Common send errors:

| Error | Cause | Solution |
|-------|-------|----------|
| **401 Unauthorized** | Invalid credentials | Reconfigure: `nylas auth config` |
| **403 Forbidden** | Missing send scope | Re-authorize with email.send scope |
| **422 Invalid** | Invalid email format | Check recipient email addresses |
| **429 Rate Limited** | Too many requests | Wait and retry |
| **500 Server Error** | Provider issue | Check provider status, retry |

### Testing send capability:

```bash
# Test with simple email to yourself
nylas email send \
  --to "your-email@example.com" \
  --subject "Test from Nylas CLI" \
  --body "This is a test message"

# If successful, try with more options
# If fails, check error message carefully
```

---

## Search Issues

### Search tips:

```bash
# Use specific filters
nylas email list --from "exact-email@example.com"

# Increase limit
nylas email list --limit 100

# Remove filters to see all
nylas email list --limit 50

# Check specific fields
nylas email list --subject "keyword"
nylas email list --unread
nylas email list --starred
```

### Search limitations:

- Search is case-insensitive
- Partial matches may not work for all providers
- Some providers have indexing delays
- Metadata search requires exact key:value match

---

## Scheduled Email Issues

### Troubleshooting scheduled sends:

1. **Verify schedule time is in future:**
```bash
# Check current time
date

# Schedule must be future time
nylas email send --to "user@example.com" --schedule "1h"
```

2. **Check draft status:**
```bash
# Scheduled emails appear as drafts
nylas drafts list

# To cancel scheduled email, delete draft
nylas drafts delete <draft-id>
```

3. **Provider support:**
   - Not all providers support native scheduled sending
   - Nylas may use draft mechanism for some providers
   - Check Nylas Dashboard for provider capabilities

---

## Attachment Problems

### Sending attachments:

```bash
# Attach single file
nylas email send \
  --to "user@example.com" \
  --subject "Document" \
  --body "Please find attached" \
  --attach "/path/to/file.pdf"

# Attach multiple files
nylas email send \
  --to "user@example.com" \
  --subject "Documents" \
  --body "Files attached" \
  --attach "/path/to/file1.pdf" \
  --attach "/path/to/file2.docx"
```

### Attachment issues:

1. **File not found:**
```bash
# Use absolute path
nylas email send --attach "/Users/username/Documents/file.pdf"

# Or relative to current directory
nylas email send --attach "./file.pdf"

# Verify file exists
ls -lh /path/to/file.pdf
```

2. **File too large:**
```bash
# Most providers limit: 25MB (Gmail), 20MB (Outlook)
# Check file size
ls -lh /path/to/file.pdf

# Compress large files
# Or use file sharing service and send link
```

3. **Unsupported file type:**
```bash
# Some providers block certain file types (.exe, .scr, etc.)
# Compress to .zip if blocked
zip archive.zip blocked-file.exe
nylas email send --attach "archive.zip"
```

---

## Performance Issues

### Slow email listing:

```bash
# Reduce limit
nylas email list --limit 10

# Use specific filters to reduce result set
nylas email list --from "sender@example.com" --limit 10

# Check API status
nylas doctor
```

### Timeout errors:

```bash
# Increase timeout (if supported)
# Or break into smaller requests

# Instead of --limit 1000
# Use multiple requests with --limit 100
```

---

## Still Having Issues?

1. **Check FAQ:** [faq.md](faq.md)
2. **Run diagnostics:** `nylas doctor`
3. **Verify credentials:** `nylas auth status`
4. **Check provider status:** Nylas Dashboard
5. **Report issue:** https://github.com/nylas/cli/issues
