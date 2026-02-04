# GPG Email Encryption

Encrypt outgoing emails with GPG/PGP so only the intended recipient can read them.

---

## Overview

The Nylas CLI supports encrypting emails with GPG (GNU Privacy Guard) using the OpenPGP standard (RFC 3156). Encrypted emails provide:

- **Confidentiality**: Only the recipient can decrypt and read the message
- **End-to-end security**: Content is encrypted before leaving your device

**Encryption vs Signing:**

| Feature | Signing | Encryption |
|---------|---------|------------|
| Purpose | Verify sender identity | Hide content from others |
| Key used | Your private key | Recipient's public key |
| Who can read | Anyone | Only recipient |
| Verifiable | Yes (authenticity) | No (confidentiality) |

**Best Practice:** Use both signing AND encryption for maximum security.

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

### 2. Have Recipient's Public Key

To encrypt for someone, you need their public key. The CLI can:
- Use keys already in your keyring
- **Auto-fetch** keys from public key servers

**Supported key servers:**
- keys.openpgp.org (primary)
- keyserver.ubuntu.com
- pgp.mit.edu
- keys.gnupg.net

---

## Sending Encrypted Email

### Basic Encryption

Encrypt an email to a recipient (auto-fetches their public key if needed):

```bash
nylas email send \
  --to recipient@example.com \
  --subject "Confidential Information" \
  --body "This message is encrypted and only you can read it." \
  --encrypt
```

**What happens:**
1. CLI looks up recipient's public key in your local keyring
2. If not found, searches key servers automatically
3. Encrypts the message with their public key
4. Sends as PGP/MIME encrypted email

### Encrypt with Specific Key

If you have multiple keys for a recipient or want to specify exactly which key:

```bash
nylas email send \
  --to recipient@example.com \
  --subject "Secret Project" \
  --body "Here are the project details..." \
  --encrypt \
  --recipient-key ABCD1234EFGH5678
```

### Encrypt to Multiple Recipients

When sending to multiple recipients (To, Cc, Bcc), the message is encrypted to all of them:

```bash
nylas email send \
  --to alice@example.com \
  --cc bob@example.com \
  --subject "Team Secret" \
  --body "Confidential team information." \
  --encrypt
```

Each recipient can decrypt the message with their own private key.

---

## Sign AND Encrypt (Recommended)

For maximum security, sign AND encrypt your emails:

```bash
nylas email send \
  --to recipient@example.com \
  --subject "Highly Confidential" \
  --body "This message is signed and encrypted." \
  --sign \
  --encrypt
```

**Benefits of sign+encrypt:**
- **Confidentiality**: Only recipient can read it
- **Authentication**: Recipient can verify it's from you
- **Integrity**: Content hasn't been tampered with

**Order of operations:**
1. Sign the content with your private key
2. Encrypt the signed content with recipient's public key
3. Recipient decrypts first, then verifies signature

---

## Reading Encrypted Email

### Decrypt a Message

To decrypt an encrypted email you received:

```bash
nylas email read <message-id> --decrypt
```

**Example output:**
```
────────────────────────────────────────────────────────────
EMAIL HEADERS
Message ID: 19c26a739a6a34d0
────────────────────────────────────────────────────────────
From: sender@example.com
To: you@example.com
Subject: Confidential Information

────────────────────────────────────────────────────────────
✓ Message decrypted successfully
────────────────────────────────────────────────────────────
  Decrypted with: B6F0A2849DC14AA2

────────────────────────────────────────────────────────────
Decrypted Content:
────────────────────────────────────────────────────────────

This is the secret message content.
```

### Decrypt AND Verify Signature

If the message was signed and encrypted, use both flags:

```bash
nylas email read <message-id> --decrypt --verify
```

**Example output:**
```
────────────────────────────────────────────────────────────
✓ Message decrypted successfully
────────────────────────────────────────────────────────────
  Decrypted with: B6F0A2849DC14AA2

  ✓ Signature verified
    Signer: Alice <alice@example.com>
    Key ID: FF8780A3D2CDCCF4A5B38D3055023BA972AB76E8

────────────────────────────────────────────────────────────
Decrypted Content:
────────────────────────────────────────────────────────────

This is the signed and encrypted message.
```

### Verify Only (Signed but Not Encrypted)

For messages that are signed but not encrypted:

```bash
nylas email read <message-id> --verify
```

---

## Key Management

### List Public Keys

See all public keys in your keyring:

```bash
gpg --list-keys
```

### Import a Public Key

Import a key from a file:

```bash
gpg --import recipient-key.asc
```

### Fetch Key from Server

Manually fetch a key from key servers:

```bash
gpg --keyserver keys.openpgp.org --recv-keys KEYID
```

Or search by email:

```bash
gpg --keyserver keys.openpgp.org --search-keys user@example.com
```

### Auto-Fetch Behavior

When you use `--encrypt`, the CLI automatically:

1. Searches your local keyring for the recipient's email
2. If not found, tries Web Key Directory (WKD)
3. Falls back to key servers in order:
   - keys.openpgp.org
   - keyserver.ubuntu.com
   - pgp.mit.edu
   - keys.gnupg.net
4. Imports the key to your keyring for future use

### Export Your Public Key

Share your public key so others can encrypt messages to you:

```bash
# Export to file
gpg --armor --export your@email.com > my-public-key.asc

# Upload to key server
gpg --keyserver keys.openpgp.org --send-keys YOUR_KEY_ID
```

---

## Examples

### Send Encrypted Project Update

```bash
nylas email send \
  --to manager@company.com \
  --subject "Q4 Financial Data" \
  --body "Attached are the confidential Q4 numbers..." \
  --encrypt
```

### Send Signed and Encrypted Contract

```bash
nylas email send \
  --to legal@partner.com \
  --subject "Contract Draft v2" \
  --body "Please review the attached contract draft." \
  --sign \
  --encrypt
```

### Decrypt Email from Inbox

```bash
# First, find the encrypted message
nylas email list --limit 10

# Then decrypt it
nylas email read 19c26a739a6a34d0 --decrypt
```

### Decrypt and Verify Important Email

```bash
nylas email read 19c26a739a6a34d0 --decrypt --verify
```

### Send to Multiple Recipients

```bash
nylas email send \
  --to alice@example.com \
  --cc bob@example.com \
  --bcc charlie@example.com \
  --subject "Board Meeting Notes" \
  --body "Confidential notes from today's meeting." \
  --sign \
  --encrypt
```

---

## Troubleshooting

### "No public key found for recipient"

**Problem:** Recipient's public key not in your keyring and not on key servers.

**Solution:**
```bash
# Ask recipient to share their public key, then import it
gpg --import their-key.asc

# Or ask them to upload to a key server
# They run: gpg --keyserver keys.openpgp.org --send-keys THEIR_KEY_ID
```

### "Message is not PGP/MIME encrypted"

**Problem:** Trying to decrypt a message that isn't encrypted.

**Solution:** Check if the message is actually encrypted:
```bash
# View raw MIME to check Content-Type
nylas email read <message-id> --mime
```

Encrypted messages have:
```
Content-Type: multipart/encrypted; protocol="application/pgp-encrypted"
```

### "No secret key available to decrypt"

**Problem:** You don't have the private key needed to decrypt.

**Solution:** The message was encrypted for a different recipient. You can only decrypt messages encrypted to your public key.

### "GPG decryption failed: Timeout"

**Problem:** GPG passphrase prompt timed out.

**Solution:**
```bash
# Start gpg-agent
gpg-agent --daemon

# Or configure pinentry
echo "pinentry-program /usr/bin/pinentry-tty" >> ~/.gnupg/gpg-agent.conf
gpgconf --kill gpg-agent
```

### "Key is expired"

**Problem:** Recipient's public key has expired.

**Solution:**
```bash
# Check key expiration
gpg --list-keys recipient@example.com

# Ask recipient to extend their key or create a new one
# Then re-fetch: gpg --keyserver keys.openpgp.org --recv-keys THEIR_KEY_ID
```

---

## Security Considerations

### Key Trust

When auto-fetching keys, the CLI uses `--trust-model always` for the encryption operation. This means:
- Keys are used even if not explicitly trusted
- You should verify key fingerprints for sensitive communications

**Verify a key fingerprint:**
```bash
gpg --fingerprint recipient@example.com
```

Compare with the recipient through a secure channel (phone, in person).

### Multiple Recipients

When encrypting to multiple recipients:
- Each recipient can decrypt independently
- Each recipient sees the full list of To/Cc recipients
- Bcc recipients are hidden but can still decrypt

### Forward Secrecy

PGP encryption does NOT provide forward secrecy:
- If a private key is compromised later, past messages can be decrypted
- For highly sensitive data, consider additional protections

---

## Technical Details

### MIME Structure (Encrypted)

Encrypted emails use RFC 3156 PGP/MIME format:

```
Content-Type: multipart/encrypted;
    protocol="application/pgp-encrypted";
    boundary="encrypted_boundary"

--encrypted_boundary
Content-Type: application/pgp-encrypted

Version: 1

--encrypted_boundary
Content-Type: application/octet-stream

-----BEGIN PGP MESSAGE-----
[Encrypted data]
-----END PGP MESSAGE-----
--encrypted_boundary--
```

### MIME Structure (Signed + Encrypted)

When using both `--sign` and `--encrypt`:

1. Content is signed first (creates signature)
2. Signed content is encrypted
3. Result is PGP/MIME encrypted message containing signed content

### Encryption Algorithm

- **Symmetric**: AES-256 (for message encryption)
- **Asymmetric**: RSA/ECDH (for key exchange)
- **Determined by**: Recipient's public key preferences

---

## Command Reference

### Send Flags

| Flag | Description |
|------|-------------|
| `--encrypt` | Encrypt email with recipient's public key |
| `--recipient-key <id>` | Specify recipient's GPG key ID |
| `--sign` | Also sign the email (recommended with encrypt) |
| `--gpg-key <id>` | Specify signing key (for --sign) |

### Read Flags

| Flag | Description |
|------|-------------|
| `--decrypt` | Decrypt PGP/MIME encrypted message |
| `--verify` | Verify GPG signature (use with --decrypt for sign+encrypt) |
| `--mime` | Show raw MIME (to inspect encryption) |

---

## Related Documentation

- [GPG Email Signing](email-signing.md) - Sign emails without encryption
- [GPG Explained](explain-gpg.md) - Understanding GPG concepts
- `nylas email send --help` - Full command options
- `nylas email read --help` - Full read options

---

## Quick Reference

```bash
# Encrypt only
nylas email send --to user@example.com --subject "Secret" --body "..." --encrypt

# Sign + Encrypt (recommended)
nylas email send --to user@example.com --subject "Secret" --body "..." --sign --encrypt

# Decrypt received email
nylas email read <message-id> --decrypt

# Decrypt and verify signature
nylas email read <message-id> --decrypt --verify

# Verify signature only (not encrypted)
nylas email read <message-id> --verify
```

---

**Last Updated:** 2026-02-04
