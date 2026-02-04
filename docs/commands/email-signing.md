# GPG Email Signing

Sign outgoing emails with your GPG/PGP key for cryptographic authentication.

---

## Overview

The Nylas CLI supports signing emails with GPG (GNU Privacy Guard) using the OpenPGP standard (RFC 3156). Signed emails allow recipients to verify:
- The email was sent by you (authentication)
- The email content hasn't been tampered with (integrity)

**Note:** This feature signs emails but does not encrypt them. Email content remains readable.

---

## Prerequisites

### 1. Install GPG

**Linux (Debian/Ubuntu):**
```bash
sudo apt install gnupg
```

**macOS:**
```bash
brew install gnupg
```

**Windows:**
Download from: https://gnupg.org/download/

### 2. Generate a GPG Key

If you don't have a GPG key:

```bash
gpg --gen-key
```

Follow the prompts to create your key. Use the same email address you send emails from.

### 3. Configure Git (Optional but Recommended)

Set your default signing key:

```bash
gpg --list-secret-keys --keyid-format=long
# Copy your key ID (e.g., 601FEE9B1D60185F)

git config --global user.signingkey 601FEE9B1D60185F
```

### 4. Configure Nylas CLI (Optional)

Set default GPG key and auto-sign preferences:

```bash
# Set default GPG signing key
nylas config set gpg.default_key 601FEE9B1D60185F

# Enable auto-sign for all outgoing emails
nylas config set gpg.auto_sign true

# View GPG configuration
nylas config get gpg.default_key
nylas config get gpg.auto_sign
```

**Key Selection Priority:**
1. `--gpg-key <key-id>` flag (highest priority)
2. `gpg.default_key` from Nylas config
3. From email address (auto-detected)
4. `user.signingkey` from git config (lowest priority)

---

## Usage

### List Available GPG Keys

See all GPG keys available for signing:

```bash
nylas email send --list-gpg-keys
```

**Output:**
```
Available GPG signing keys (1):

1. Key ID: 601FEE9B1D60185F
   Fingerprint: E4D944EDAD2E329591A8854B07DD577603B27996
   UID: John Doe <john@example.com>
   Expires: 2031-01-31

Default signing keys:
  From Nylas config: 601FEE9B1D60185F
  From git config: 601FEE9B1D60185F
```

### Sign with Default Key

Sign an email using your configured default key (Nylas config or git config):

```bash
nylas email send \
  --to recipient@example.com \
  --subject "Signed Email" \
  --body "This email is cryptographically signed." \
  --sign
```

### Sign with Specific Key

Sign with a specific GPG key ID:

```bash
nylas email send \
  --to recipient@example.com \
  --subject "Secure Message" \
  --body "Signed with my work key." \
  --sign \
  --gpg-key 601FEE9B1D60185F
```

### Sign HTML Emails

GPG signing works with both plain text and HTML emails:

```bash
nylas email send \
  --to recipient@example.com \
  --subject "HTML Signed Email" \
  --body "<html><body><h1>Signed HTML</h1></body></html>" \
  --sign
```

### Auto-Sign All Emails

Enable auto-sign to automatically sign all outgoing emails:

```bash
# Enable auto-sign
nylas config set gpg.auto_sign true

# Now all emails are signed automatically (no --sign flag needed)
nylas email send \
  --to recipient@example.com \
  --subject "Auto-signed Email" \
  --body "This email is automatically signed."

# Disable auto-sign
nylas config set gpg.auto_sign false
```

**Note:** Auto-sign uses the configured default key (Nylas config or git config). You can still override with `--gpg-key` flag.

---

## How It Works

1. **Build Email**: CLI constructs the email content
2. **Sign with GPG**: Content is signed using your private key
3. **Build PGP/MIME**: Email is wrapped in RFC 3156 PGP/MIME structure
4. **Send**: Signed email is sent via Nylas API as raw MIME

The result is a `multipart/signed` email with:
- Part 1: Original email content
- Part 2: Detached PGP signature (`application/pgp-signature`)

---

## Verification

### In Email Clients

Recipients can verify your signature in compatible email clients:

**Gmail**: Shows a signed icon and "This message is signed" banner

**Thunderbird/Outlook**: Shows signature verification with your key details

**Apple Mail**: Shows signed badge in message header

### Manual Verification

To manually verify a signed email:

1. Save the raw email to a file (`.eml`)
2. Extract the signature and content
3. Verify with GPG:

```bash
gpg --verify signature.asc content.txt
```

---

## Troubleshooting

### "GPG not found"

**Problem:** GPG is not installed or not in PATH

**Solution:**
```bash
# Linux
sudo apt install gnupg

# macOS
brew install gnupg

# Verify installation
gpg --version
```

### "No default GPG key configured"

**Problem:** No default signing key set in git config

**Solution:**
```bash
# List your keys
gpg --list-secret-keys --keyid-format=long

# Set default key
git config --global user.signingkey YOUR_KEY_ID
```

### "GPG signing failed: Inappropriate ioctl for device"

**Problem:** GPG can't prompt for passphrase in non-interactive environment

**Solution:**
```bash
# Configure gpg-agent to cache passphrase
echo "use-agent" >> ~/.gnupg/gpg.conf

# Start gpg-agent
gpg-agent --daemon

# Alternative: Remove passphrase from key (less secure)
gpg --edit-key YOUR_KEY_ID
# Then: passwd -> (enter old passphrase) -> (leave new passphrase empty)
```

### "GPG key not found or not usable for signing"

**Problem:** Specified key ID doesn't exist or can't sign

**Solution:**
```bash
# List all secret keys
nylas email send --list-gpg-keys

# Use a valid key ID
nylas email send --to user@example.com --subject "Test" --sign --gpg-key VALID_KEY_ID
```

### Recipients can't verify signature

**Problem:** Recipients don't have your public key

**Solution:** Export and share your public key:

```bash
# Export public key
gpg --armor --export YOUR_KEY_ID > my-public-key.asc

# Upload to key server
gpg --send-keys YOUR_KEY_ID --keyserver keys.openpgp.org
```

---

## Best Practices

### Security

- **Protect your private key**: Never share your private key
- **Use strong passphrase**: Protect your key with a strong passphrase
- **Set expiration dates**: Keys should expire and be renewed periodically
- **Backup your key**: Keep secure backups of your private key

### Key Management

- **One key per identity**: Use different keys for personal/work email
- **Publish public key**: Upload to key servers for easy verification
- **Revoke compromised keys**: If your key is compromised, revoke it immediately

### Usage

- **Sign important emails**: Use signing for contracts, financial communications
- **Don't sign spam**: Recipients may flag unsigned emails from you as suspicious
- **Test first**: Send a signed test email to yourself to verify setup

---

## Limitations

### Current Limitations

- **Signing only**: This document covers signing. For encryption, see [GPG Email Encryption](encryption.md)
- **No S/MIME**: Only PGP/MIME format is supported
- **Manual verification**: Some email clients don't auto-verify signatures

---

## Technical Details

### MIME Structure

Signed emails use RFC 3156 PGP/MIME format:

```
Content-Type: multipart/signed; protocol="application/pgp-signature";
    micalg=pgp-sha256; boundary="boundary"

--boundary
Content-Type: text/plain; charset=utf-8

[Email body]

--boundary
Content-Type: application/pgp-signature; name="signature.asc"

-----BEGIN PGP SIGNATURE-----
[Signature data]
-----END PGP SIGNATURE-----
--boundary--
```

### Signature Algorithm

- **Default**: SHA256 with RSA
- **Configurable**: Determined by your GPG key type
- **micalg**: Set automatically based on hash algorithm

### Key Selection Priority

The CLI determines which GPG key to use in this order:

1. `--gpg-key <key-id>` flag (highest priority - explicit override)
2. `gpg.default_key` from Nylas config (`nylas config set gpg.default_key`)
3. From email address (auto-detected from grant)
4. `user.signingkey` from git config (lowest priority - fallback)
5. Error if no key available

---

## Examples

### Basic Signed Email

```bash
nylas email send \
  --to colleague@company.com \
  --subject "Q4 Report" \
  --body "Please review the attached Q4 financial report." \
  --sign
```

### Signed with Attachment

```bash
# (Note: Attachment support requires additional flags - see email send --help)
nylas email send \
  --to finance@company.com \
  --subject "Invoice #12345" \
  --body "Attached is the signed invoice." \
  --sign \
  --gpg-key 601FEE9B1D60185F
```

### Scheduled Signed Email

```bash
nylas email send \
  --to team@company.com \
  --subject "Weekly Update" \
  --body "Here's this week's status update." \
  --sign \
  --schedule "tomorrow 9am"
```

---

## Related Documentation

- [GPG Email Encryption](encryption.md) - Encrypt emails for confidentiality
- [GPG Explained](explain-gpg.md) - Understanding GPG concepts
- `nylas email send --help` - See all email sending options
- `gpg --list-keys` - List all GPG keys
- `gpg --gen-key` - Generate new GPG key

---

**Last Updated:** 2026-02-04
