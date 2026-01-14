# API Troubleshooting

Comprehensive guide for resolving API connectivity and error issues.

---

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [HTTP Status Codes](#http-status-codes)
- [Rate Limiting](#rate-limiting)
- [Connection Issues](#connection-issues)
- [Permission Errors](#permission-errors)
- [API Versioning](#api-versioning)

---

## Quick Diagnostics

```bash
# Run health check
nylas doctor

# Test API connectivity
curl -H "Authorization: Bearer YOUR_API_KEY" \
     https://api.nylas.com/v3/grants

# Check authentication
nylas auth status
```

---

## HTTP Status Codes

### 401 Unauthorized

**Symptoms:**
```
Error: 401 Unauthorized
Authentication failed
Invalid API key
```

**Causes:**
- Invalid or expired API key
- Missing API key
- Incorrect API key format

**Solutions:**
```bash
# Reconfigure with correct API key
nylas auth config

# Verify API key in Nylas Dashboard
# https://dashboardv3.nylas.com → Apps → Your App → API Keys

# Check configured key
nylas auth status

# Test with environment variable
export NYLAS_API_KEY="nyk_xxx..."
nylas email list
```

---

### 403 Forbidden

**Symptoms:**
```
Error: 403 Forbidden
Permission denied
Insufficient scope
```

**Causes:**
- Missing required scope
- Grant doesn't have necessary permissions
- Account limitations

**Solutions:**
```bash
# Check grant scopes
nylas admin grants show <grant-id>

# Required scopes by feature:
# - email: email.read, email.send
# - calendar: calendar.read, calendar.modify
# - contacts: contacts.read, contacts.modify

# Re-authorize with correct scopes
nylas auth login

# Or create new grant with required scopes
# In Nylas Dashboard: Grants → Create Grant
```

---

### 404 Not Found

**Symptoms:**
```
Error: 404 Not Found
Resource not found
Invalid grant ID
```

**Causes:**
- Invalid Grant ID
- Resource ID doesn't exist
- Wrong API endpoint

**Solutions:**
```bash
# Verify Grant ID
nylas admin grants list

# Check configured grant
nylas auth status

# Use correct grant
nylas auth config  # Update grant ID

# Verify resource exists
nylas email list  # Check if email ID is valid
```

---

### 422 Unprocessable Entity

**Symptoms:**
```
Error: 422 Unprocessable Entity
Invalid request
Validation failed
```

**Causes:**
- Invalid parameter format
- Missing required field
- Data validation failure

**Solutions:**
```bash
# Check required fields
# Email send requires: --to, --subject, --body
nylas email send \
  --to "recipient@example.com" \
  --subject "Subject" \
  --body "Body"

# Verify email format
# ✅ Correct: user@example.com
# ❌ Wrong: user@example

# Check date formats
# ✅ Correct: 2024-12-25 10:00
# ❌ Wrong: 25/12/2024
```

---

### 429 Too Many Requests

**Symptoms:**
```
Error: 429 Too Many Requests
Rate limit exceeded
Try again later
```

**Causes:**
- Exceeded API rate limits
- Too many requests in short time
- Burst limit exceeded

**Solutions:**
```bash
# Wait before retrying
sleep 60 && nylas email list

# Reduce request frequency
# Add delays between commands

# Use batch operations where possible
# Instead of multiple single requests

# Check rate limits
# See: Rate Limiting section below
```

---

### 500 Internal Server Error

**Symptoms:**
```
Error: 500 Internal Server Error
Server error
Service unavailable
```

**Causes:**
- Nylas API issue
- Provider (Gmail/Outlook) issue
- Temporary service disruption

**Solutions:**
```bash
# Retry after delay
sleep 30 && nylas email list

# Check Nylas status
# Visit: https://status.nylas.com

# Try different endpoint
# If emails fail, try calendar

# Report if persistent
# https://support.nylas.com
```

---

### 503 Service Unavailable

**Symptoms:**
```
Error: 503 Service Unavailable
Service temporarily unavailable
Maintenance in progress
```

**Causes:**
- Scheduled maintenance
- Service degradation
- Provider issues

**Solutions:**
```bash
# Check status page
# https://status.nylas.com

# Wait and retry
sleep 300 && nylas email list  # Wait 5 minutes

# Check provider status
# Gmail: https://www.google.com/appsstatus
# Outlook: https://portal.office.com/servicestatus
```

---

## Rate Limiting

### Understanding rate limits:

**Nylas API v3 limits:**
- **Free tier:** 100 requests/minute
- **Paid tier:** Higher limits (varies by plan)
- **Burst limit:** Short-term burst allowed

### Rate limit headers:

When you hit rate limit, API returns:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1640000000
Retry-After: 60
```

### Handling rate limits:

```bash
# Add delays between commands
nylas email list
sleep 1
nylas email list

# Reduce request frequency
# Instead of polling every second, poll every minute

# Use webhooks for real-time updates
# Instead of frequent polling
nylas webhook create --url "https://myapp.com/webhook"

# Batch operations
# Get multiple emails in one request with --limit
nylas email list --limit 50
```

### Rate limit best practices:

1. **Cache results** - Don't fetch same data repeatedly
2. **Use webhooks** - Real-time updates instead of polling
3. **Increase limits** - Upgrade plan if needed
4. **Exponential backoff** - Wait longer after each retry
5. **Monitor usage** - Track requests per minute

### Exponential backoff example:

```bash
# Simple retry with increasing delays
retry_count=0
max_retries=5

while [ $retry_count -lt $max_retries ]; do
  if nylas email list; then
    break
  fi

  retry_count=$((retry_count + 1))
  wait_time=$((2 ** retry_count))
  echo "Retry $retry_count/$max_retries after ${wait_time}s..."
  sleep $wait_time
done
```

---

## Connection Issues

### Network connectivity problems:

**Symptoms:**
- Connection timeout
- Connection refused
- DNS resolution failed

**Solutions:**

1. **Test basic connectivity:**
```bash
# Test DNS resolution
nslookup api.nylas.com

# Test HTTPS connection
curl -I https://api.nylas.com

# Test with verbose output
curl -v https://api.nylas.com/v3/grants
```

2. **Check firewall/proxy:**
```bash
# If behind corporate firewall/proxy
# Configure proxy
export HTTP_PROXY="http://proxy.company.com:8080"
export HTTPS_PROXY="http://proxy.company.com:8080"

# Test with proxy
nylas email list
```

3. **Check network:**
```bash
# Verify internet connection
ping 8.8.8.8

# Test DNS
ping api.nylas.com

# Check routing
traceroute api.nylas.com
```

4. **Timeout issues:**
```bash
# Increase timeout if slow network
# (CLI may not support this directly)

# Test with curl
curl --max-time 60 https://api.nylas.com/v3/grants
```

---

### SSL/TLS issues:

**Symptoms:**
- SSL certificate error
- Certificate verification failed
- SSL handshake failed

**Solutions:**
```bash
# Verify SSL certificate
openssl s_client -connect api.nylas.com:443

# Check system time (affects cert validation)
date

# Update CA certificates
# macOS: Handled by Keychain
# Linux: sudo update-ca-certificates
# Windows: Update via Windows Update

# Test without cert verification (not recommended)
curl -k https://api.nylas.com
```

---

## Permission Errors

### Missing scopes:

**Check required scopes by feature:**

| Feature | Required Scopes |
|---------|----------------|
| **List emails** | `email.read` or `email.read_only` |
| **Send emails** | `email.send` |
| **Read calendar** | `calendar.read` or `calendar.read_only` |
| **Modify calendar** | `calendar.modify` |
| **Read contacts** | `contacts.read` or `contacts.read_only` |
| **Modify contacts** | `contacts.modify` |
| **Webhooks** | Varies by event type |

### Checking grant scopes:

```bash
# View grant details including scopes
nylas admin grants show <grant-id>

# Output shows:
# Scopes: email.read, calendar.read
```

### Requesting additional scopes:

```bash
# Re-authorize with additional scopes
nylas auth login

# In Nylas Dashboard:
# 1. Go to Grants
# 2. Delete old grant
# 3. Create new grant with required scopes
```

---

## API Versioning

### Nylas CLI uses v3 API only:

**Important:**
- This CLI **only supports Nylas v3 API**
- v1 and v2 are **not supported**
- Ensure your Nylas account uses v3

### Checking API version:

```bash
# CLI always uses v3
# Check API endpoint
curl -I https://api.nylas.com/v3/grants

# Verify in code
# All endpoints use: https://api.nylas.com/v3/...
```

### Migrating from v1/v2:

If using old Nylas v1 or v2:
1. **Upgrade to v3** in Nylas Dashboard
2. **Create new grant** for v3
3. **Update credentials** in CLI
4. **Test authentication**

See: https://developer.nylas.com/docs/v3/

---

## Advanced Debugging

### Enable debug mode:

```bash
# Run with debug flag (if supported)
nylas --debug email list

# Check logs
tail -f ~/.config/nylas/nylas.log
```

### Manual API testing:

```bash
# Test API directly
curl -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     https://api.nylas.com/v3/grants/YOUR_GRANT_ID/messages?limit=1

# Expected response: JSON with messages
# Error response: JSON with error details
```

### Common API request format:

```bash
# GET request
curl -H "Authorization: Bearer $NYLAS_API_KEY" \
     https://api.nylas.com/v3/grants/$GRANT_ID/messages

# POST request (send email)
curl -X POST \
     -H "Authorization: Bearer $NYLAS_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "subject": "Test",
       "to": [{"email": "recipient@example.com"}],
       "body": "Hello"
     }' \
     https://api.nylas.com/v3/grants/$GRANT_ID/messages/send
```

---

## Monitoring and Logging

### Check CLI configuration:

```bash
# View config
cat ~/.config/nylas/config.yaml

# Check logs (if available)
ls -la ~/.config/nylas/

# View recent errors
tail -20 ~/.config/nylas/nylas.log
```

### API health check:

```bash
# Quick health check
nylas doctor

# Test each component
nylas auth status          # Auth
nylas email list --limit 1 # Email API
nylas calendar list        # Calendar API
```

---

## Still Having Issues?

1. **Check API status:** https://status.nylas.com
2. **Review docs:** https://developer.nylas.com/docs/api/v3/
3. **Run diagnostics:** `nylas doctor`
4. **Check FAQ:** [faq.md](faq.md)
5. **Report issue:** https://github.com/nylas/cli/issues
6. **Nylas support:** https://support.nylas.com
