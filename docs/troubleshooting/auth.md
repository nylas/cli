# Authentication Troubleshooting

Comprehensive guide for resolving authentication and credential issues.

---

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Common Issues](#common-issues)
- [Initial Setup](#initial-setup)
- [Login Issues](#login-issues)
- [Credential Storage](#credential-storage)
- [Multi-Account Management](#multi-account-management)
- [API Key Issues](#api-key-issues)
- [OAuth Issues](#oauth-issues)
- [Advanced Troubleshooting](#advanced-troubleshooting)

---

## Quick Diagnostics

```bash
# Check authentication status
nylas auth status

# Verify configuration
cat ~/.config/nylas/config.yaml

# Run full diagnostics
nylas doctor
```

---

## Common Issues

### Issue: "Invalid API key" or "Authentication failed"

**Symptoms:**
- Error: `401 Unauthorized`
- Error: `Invalid API key`
- Commands fail with authentication errors

**Solutions:**

1. **Reconfigure API credentials:**
```bash
nylas auth config
# Enter your Nylas API key when prompted
# Enter your Grant ID when prompted
```

2. **Verify credentials are correct:**
```bash
# Check your API key on https://dashboardv3.nylas.com
# Copy the correct API key
# Copy the correct Grant ID

# Reconfigure
nylas auth config
```

3. **Check credential storage:**
```bash
# Verify keyring access (macOS)
security find-generic-password -s nylas-api-key

# Verify keyring access (Linux - GNOME Keyring)
secret-tool lookup service nylas attribute api-key

# If keyring fails, credentials fall back to config file
cat ~/.config/nylas/config.yaml
```

---

### Issue: "No grant ID configured"

**Symptoms:**
- Error: `grant ID is required`
- Email/calendar commands fail
- API returns 404 errors

**Solutions:**

1. **Get your Grant ID from Nylas Dashboard:**
   - Login to https://dashboardv3.nylas.com
   - Go to "Grants" section
   - Copy your Grant ID (format: `grant_xxxx`)

2. **Configure Grant ID:**
```bash
nylas auth config
# Paste your Grant ID when prompted
```

3. **Specify Grant ID per command:**
```bash
# Use --grant-id flag
nylas email list --grant-id "grant_xxxx"

# Or pass as argument
nylas email list grant_xxxx
```

4. **Manage multiple grants:**
```bash
# List all grants
nylas admin grants list

# Switch between grants
nylas auth config  # Update default grant
```

---

### Issue: OAuth login fails / Browser not opening

**Symptoms:**
- `nylas auth login` doesn't open browser
- Browser opens but shows error
- OAuth redirect fails

**Solutions:**

1. **Manual browser opening:**
```bash
# If browser doesn't auto-open, copy the URL manually
nylas auth login
# Copy the URL shown in terminal
# Open it manually in your browser
```

2. **Check OAuth configuration in Nylas Dashboard:**
   - Verify redirect URIs are configured
   - Check that your application is approved
   - Ensure OAuth is enabled for your app

3. **Firewall/Network issues:**
```bash
# Test connectivity
curl https://api.nylas.com/v3/grants

# Check if localhost callback works
# Ensure port 8080 is not blocked
```

4. **Use different OAuth flow:**
   - Try using API key + Grant ID instead of OAuth
   - See: https://developer.nylas.com/docs/api/v3/#tag/Auth

---

### Issue: "Permission denied" when accessing credentials

**Symptoms:**
- Error: `failed to access keyring`
- Error: `permission denied`
- Credentials not saved

**Solutions:**

1. **macOS - Keychain access:**
```bash
# Grant terminal access to Keychain
# System Preferences → Security & Privacy → Privacy → Full Disk Access
# Add your terminal app (Terminal.app, iTerm2, etc.)

# Verify keychain is unlocked
security unlock-keychain

# Manually check keychain
open /Applications/Utilities/Keychain\ Access.app
```

2. **Linux - GNOME Keyring:**
```bash
# Install GNOME Keyring if missing
sudo apt-get install gnome-keyring  # Debian/Ubuntu
sudo dnf install gnome-keyring      # Fedora

# Start keyring daemon
gnome-keyring-daemon --start

# Check keyring status
secret-tool search service nylas
```

3. **Windows - Credential Manager:**
```powershell
# Open Credential Manager
control /name Microsoft.CredentialManager

# Look for "nylas" credentials
# Grant access if prompted
```

4. **Fallback to file storage:**
```bash
# If keyring fails, credentials fall back to:
# ~/.config/nylas/config.yaml
# This is less secure but works without keyring

# Verify file permissions
chmod 600 ~/.config/nylas/config.yaml
```

---

## Initial Setup

### First-time configuration:

```bash
# Step 1: Get your Nylas credentials
# Go to: https://dashboardv3.nylas.com
# Create an app or use existing app
# Get: API Key, Grant ID

# Step 2: Configure CLI
nylas auth config

# Step 3: Verify setup
nylas auth status

# Step 4: Test authentication
nylas email list --limit 1
```

### What you need:

| Credential | Where to find | Format |
|------------|---------------|--------|
| **API Key** | Dashboard → Apps → Your App → API Keys | `nyk_xxx...` |
| **Grant ID** | Dashboard → Grants → Your Grant | `grant_xxx...` |

---

## Login Issues

### OAuth vs API Key authentication:

**API Key + Grant ID (Recommended for CLI):**
```bash
nylas auth config
# Enter API Key: nyk_xxx...
# Enter Grant ID: grant_xxx...
```

**OAuth Login:**
```bash
nylas auth login
# Opens browser for OAuth flow
# Redirects back to CLI after authentication
```

### When OAuth login fails:

1. **Check redirect URI configuration**
2. **Verify application is approved**
3. **Try API Key method instead**
4. **Check browser console for errors**

---

## Credential Storage

### Storage hierarchy:

1. **Primary: System keyring** (most secure)
   - macOS: Keychain
   - Linux: Secret Service (GNOME Keyring, KWallet)
   - Windows: Windows Credential Manager

2. **Fallback: Config file** (if keyring unavailable)
   - Location: `~/.config/nylas/config.yaml`
   - Permissions: `600` (read/write for user only)

### Viewing stored credentials:

```bash
# View config file
cat ~/.config/nylas/config.yaml

# macOS: View keychain
security find-generic-password -s nylas-api-key -w

# Linux: View GNOME Keyring
secret-tool lookup service nylas attribute api-key
```

### Clearing credentials:

```bash
# Remove config file
rm ~/.config/nylas/config.yaml

# macOS: Remove from keychain
security delete-generic-password -s nylas-api-key

# Linux: Remove from GNOME Keyring
secret-tool clear service nylas attribute api-key

# Reconfigure
nylas auth config
```

---

## Multi-Account Management

### Managing multiple accounts:

```bash
# List all grants
nylas admin grants list

# Use different grant per command
nylas email list grant_account1
nylas email list grant_account2

# Switch default grant
nylas auth config  # Update default grant ID
```

### Configuration for multiple accounts:

```yaml
# ~/.config/nylas/config.yaml
default_grant: grant_account1

# Specify grant per command
nylas email list grant_account2
```

---

## API Key Issues

### Invalid API key errors:

1. **Verify API key is correct:**
   - Login to https://dashboardv3.nylas.com
   - Go to Apps → Your App → API Keys
   - Copy the **correct** API key
   - Format should be: `nyk_xxx...`

2. **Check API key permissions:**
   - Some API keys may have limited scopes
   - Verify key has necessary permissions
   - Try creating a new API key if needed

3. **API key expiration:**
   - Check if key is still active in dashboard
   - Rotate keys if compromised
   - Update CLI configuration with new key

---

## OAuth Issues

### OAuth flow troubleshooting:

1. **Redirect URI mismatch:**
```bash
# Ensure redirect URI in Nylas Dashboard matches
# Default: http://localhost:8080/callback

# Check your app settings:
# Dashboard → Apps → Your App → OAuth → Redirect URIs
```

2. **OAuth scope issues:**
   - Verify required scopes are enabled
   - Common scopes: `email`, `calendar`, `contacts`

3. **OAuth consent screen:**
   - Ensure app is published/approved
   - Check consent screen configuration

---

## Advanced Troubleshooting

### Debug mode:

```bash
# Enable verbose output
nylas --debug email list

# Check logs
tail -f ~/.config/nylas/nylas.log
```

### API connectivity test:

```bash
# Test API directly
curl -H "Authorization: Bearer YOUR_API_KEY" \
     https://api.nylas.com/v3/grants/YOUR_GRANT_ID

# Should return grant details if auth is working
```

### Common HTTP status codes:

| Code | Meaning | Solution |
|------|---------|----------|
| **401** | Unauthorized | Invalid API key → Reconfigure |
| **403** | Forbidden | Insufficient permissions → Check scopes |
| **404** | Not found | Invalid Grant ID → Verify Grant ID |
| **429** | Rate limited | Wait before retrying |

### Environment variables:

```bash
# Override config with environment variables
export NYLAS_API_KEY="nyk_xxx..."
export NYLAS_GRANT_ID="grant_xxx..."

# Test with env vars
nylas email list
```

### Config file structure:

```yaml
# ~/.config/nylas/config.yaml
api_key: nyk_xxx...
grant_id: grant_xxx...
api_url: https://api.nylas.com
```

---

## Still Having Issues?

1. **Run diagnostics:**
```bash
nylas doctor
```

2. **Check system keyring:**
   - Verify keyring is running
   - Grant necessary permissions
   - Try fallback to config file

3. **Verify credentials:**
   - Login to Nylas Dashboard
   - Verify API key is active
   - Verify Grant ID exists

4. **Get help:**
   - Check [FAQ](faq.md)
   - Report issue: https://github.com/nylas/cli/issues
   - Nylas support: https://support.nylas.com
