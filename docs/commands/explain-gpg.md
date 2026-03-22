# Understanding GPG: A Complete Guide

A comprehensive guide to GPG (GNU Privacy Guard) cryptography, email signing, and verification.

---

## Table of Contents

1. [What is GPG?](#what-is-gpg)
2. [Asymmetric Cryptography](#asymmetric-cryptography)
3. [GPG Key Anatomy](#gpg-key-anatomy)
4. [Digital Signatures](#digital-signatures)
5. [Email Signing (PGP/MIME)](#email-signing-pgpmime)
6. [Signature Verification](#signature-verification)
7. [Trust Model](#trust-model)
8. [Key Servers](#key-servers)
9. [CLI Integration](#cli-integration)
10. [Security Considerations](#security-considerations)

---

## What is GPG?

**GPG (GNU Privacy Guard)** is a free implementation of the OpenPGP standard (RFC 4880). It provides:

- **Digital Signatures** - Prove authorship and integrity
- **Encryption** - Protect confidential data
- **Key Management** - Create, store, and share cryptographic keys

```
┌─────────────────────────────────────────────────────────────────┐
│                         GPG ECOSYSTEM                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│   │   OpenPGP    │    │     GPG      │    │  Key Servers │     │
│   │   Standard   │───▶│   Software   │◀──▶│   (Public)   │     │
│   │  (RFC 4880)  │    │              │    │              │     │
│   └──────────────┘    └──────────────┘    └──────────────┘     │
│                              │                                  │
│                              ▼                                  │
│                    ┌──────────────────┐                        │
│                    │   Your Keyring   │                        │
│                    │  ┌────────────┐  │                        │
│                    │  │Private Keys│  │                        │
│                    │  │Public Keys │  │                        │
│                    │  └────────────┘  │                        │
│                    └──────────────────┘                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### History

| Year | Event |
|------|-------|
| 1991 | Phil Zimmermann creates PGP (Pretty Good Privacy) |
| 1997 | OpenPGP standard published (RFC 2440) |
| 1999 | GPG 1.0 released as free software |
| 2007 | RFC 4880 updates OpenPGP standard |
| Today | GPG is the de facto standard for email encryption/signing |

---

## Asymmetric Cryptography

GPG uses **asymmetric (public-key) cryptography**, which uses two mathematically related keys:

```
┌─────────────────────────────────────────────────────────────────┐
│                    KEY PAIR RELATIONSHIP                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│        ┌─────────────────┐      ┌─────────────────┐            │
│        │   PRIVATE KEY   │      │   PUBLIC KEY    │            │
│        │   (Secret)      │      │   (Shareable)   │            │
│        │                 │      │                 │            │
│        │  🔐 Keep safe!  │      │  📢 Share it!   │            │
│        │  Never share    │      │  Upload to      │            │
│        │                 │      │  key servers    │            │
│        └────────┬────────┘      └────────┬────────┘            │
│                 │                        │                      │
│                 │    Mathematically      │                      │
│                 │◀──────Related────────▶│                      │
│                 │                        │                      │
│                 ▼                        ▼                      │
│        ┌─────────────────┐      ┌─────────────────┐            │
│        │     SIGNS       │      │    VERIFIES     │            │
│        │    DECRYPTS     │      │    ENCRYPTS     │            │
│        └─────────────────┘      └─────────────────┘            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### How It Works

**Signing (proves identity):**
```
┌──────────┐     Private Key      ┌───────────┐
│ Document │────────────────────▶│ Signature │
└──────────┘     (Only you have)  └───────────┘
```

**Verification (anyone can verify):**
```
┌──────────┐ + ┌───────────┐    Public Key    ┌─────────┐
│ Document │   │ Signature │─────────────────▶│ Valid?  │
└──────────┘   └───────────┘   (Anyone has)   │ Yes/No  │
                                              └─────────┘
```

### Common Algorithms

| Algorithm | Type | Key Size | Security Level |
|-----------|------|----------|----------------|
| RSA | Sign + Encrypt | 2048-4096 bits | Strong |
| DSA | Sign only | 2048-3072 bits | Strong |
| EdDSA (Ed25519) | Sign only | 256 bits | Very Strong |
| ECDH | Encrypt only | 256-521 bits | Very Strong |

---

## GPG Key Anatomy

A GPG key is more than just cryptographic data. It's a structured package containing:

```
┌─────────────────────────────────────────────────────────────────┐
│                      GPG KEY STRUCTURE                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                    PRIMARY KEY                             │ │
│  │  ┌─────────────────────────────────────────────────────┐  │ │
│  │  │ Key ID:      601FEE9B1D60185F (last 16 hex chars)   │  │ │
│  │  │ Fingerprint: DBADDF54A44EB10E9714F386601FEE9B1D60185F│  │ │
│  │  │ Algorithm:   RSA 4096                                │  │ │
│  │  │ Created:     2024-01-15                              │  │ │
│  │  │ Expires:     2026-01-15 (optional)                   │  │ │
│  │  │ Capabilities: [C]ertify, [S]ign                      │  │ │
│  │  └─────────────────────────────────────────────────────┘  │ │
│  └───────────────────────────────────────────────────────────┘ │
│                              │                                  │
│              ┌───────────────┼───────────────┐                 │
│              ▼               ▼               ▼                 │
│  ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐      │
│  │   USER IDs      │ │  SUBKEYS    │ │   SIGNATURES    │      │
│  │                 │ │             │ │                 │      │
│  │ John Doe        │ │ [E]ncrypt   │ │ Self-signature  │      │
│  │ <john@work.com> │ │ [S]ign      │ │ Other users'    │      │
│  │                 │ │ [A]uth      │ │ certifications  │      │
│  │ John Doe        │ │             │ │                 │      │
│  │ <john@home.com> │ │             │ │                 │      │
│  └─────────────────┘ └─────────────┘ └─────────────────┘      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Key Identifiers

```
┌─────────────────────────────────────────────────────────────────┐
│                    KEY IDENTIFIER TYPES                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  FINGERPRINT (40 hex characters) - Most Secure                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ DBAD DF54 A44E B10E 9714 F386 601F EE9B 1D60 185F       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  LONG KEY ID (16 hex characters)                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                          601FEE9B1D60185F               │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  SHORT KEY ID (8 hex characters) - NOT RECOMMENDED              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                                    1D60185F             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ⚠️  Short Key IDs can have collisions! Always use fingerprint  │
│     or long key ID for security-critical operations.            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### User IDs (UIDs)

A key can have multiple User IDs, typically in the format:

```
Name (Comment) <email@example.com>
```

**Examples:**
- `John Doe <john@example.com>`
- `John Doe (Work) <john@company.com>`
- `John Doe (Personal) <john@gmail.com>`

---

## Digital Signatures

### How Signing Works

```
┌─────────────────────────────────────────────────────────────────┐
│                    SIGNING PROCESS                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  STEP 1: Hash the Document                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                                                         │   │
│  │   "Hello, this is my email..."                          │   │
│  │              │                                          │   │
│  │              ▼                                          │   │
│  │      ┌─────────────┐                                    │   │
│  │      │   SHA-256   │  (Hash Function)                   │   │
│  │      └──────┬──────┘                                    │   │
│  │             ▼                                           │   │
│  │   a1b2c3d4e5f6... (256-bit hash)                        │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  STEP 2: Encrypt Hash with Private Key                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                                                         │   │
│  │   a1b2c3d4e5f6...  +  🔐 Private Key                    │   │
│  │              │                                          │   │
│  │              ▼                                          │   │
│  │      ┌─────────────┐                                    │   │
│  │      │   RSA/DSA   │  (Asymmetric Encryption)           │   │
│  │      └──────┬──────┘                                    │   │
│  │             ▼                                           │   │
│  │   -----BEGIN PGP SIGNATURE-----                         │   │
│  │   iQJJBAEBCAAzFiEE263fVKROsQ...                         │   │
│  │   -----END PGP SIGNATURE-----                           │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  The signature contains:                                        │
│  • Encrypted hash                                               │
│  • Key ID used for signing                                      │
│  • Timestamp                                                    │
│  • Hash algorithm identifier (SHA256, SHA512, etc.)             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Signature Properties

| Property | What It Proves |
|----------|----------------|
| **Authenticity** | The message came from the key owner |
| **Integrity** | The message hasn't been modified |
| **Non-repudiation** | The signer cannot deny signing |
| **Timestamp** | When the signature was made |

### Signature Types

```
┌─────────────────────────────────────────────────────────────────┐
│                    SIGNATURE TYPES                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. INLINE SIGNATURE (Clearsign)                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ -----BEGIN PGP SIGNED MESSAGE-----                      │   │
│  │ Hash: SHA256                                            │   │
│  │                                                         │   │
│  │ This is my message text.                                │   │
│  │ -----BEGIN PGP SIGNATURE-----                           │   │
│  │ iQJJBAEBCAAzFiEE...                                     │   │
│  │ -----END PGP SIGNATURE-----                             │   │
│  └─────────────────────────────────────────────────────────┘   │
│  Use: Simple text messages, readable without GPG               │
│                                                                 │
│  2. DETACHED SIGNATURE (Separate file)                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ document.pdf  ←── Original file (unchanged)             │   │
│  │ document.pdf.sig  ←── Signature file                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│  Use: Binary files, documents, email (PGP/MIME)                │
│                                                                 │
│  3. PGP/MIME (Email standard - RFC 3156)                        │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ multipart/signed                                        │   │
│  │ ├── Part 1: Original message (any content type)         │   │
│  │ └── Part 2: application/pgp-signature                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│  Use: Email signing (supported by most email clients)          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Email Signing (PGP/MIME)

### RFC 3156 Standard

PGP/MIME (RFC 3156) defines how to sign emails while preserving MIME structure:

```
┌─────────────────────────────────────────────────────────────────┐
│                  PGP/MIME EMAIL STRUCTURE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  From: sender@example.com                                       │
│  To: recipient@example.com                                      │
│  Subject: Signed Email                                          │
│  Content-Type: multipart/signed;                                │
│      protocol="application/pgp-signature";                      │
│      micalg=pgp-sha256;                                         │
│      boundary="=_signed_abc123"                                 │
│                                                                 │
│  --=_signed_abc123                                              │
│  Content-Type: text/plain; charset=utf-8                        │
│  Content-Transfer-Encoding: quoted-printable                    │
│                                                                 │
│  This is the email body that is signed.                         │
│  Any modifications will invalidate the signature.               │
│                                                                 │
│  --=_signed_abc123                                              │
│  Content-Type: application/pgp-signature;                       │
│      name="signature.asc"                                       │
│  Content-Disposition: attachment;                               │
│      filename="signature.asc"                                   │
│                                                                 │
│  -----BEGIN PGP SIGNATURE-----                                  │
│                                                                 │
│  iQJJBAEBCAAzFiEE263fVKROsQ6XFPOGYB/umx1gGF8FAmV1234AABQJEA     │
│  ... (base64 encoded signature data) ...                        │
│  =Ab12                                                          │
│  -----END PGP SIGNATURE-----                                    │
│                                                                 │
│  --=_signed_abc123--                                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Content-Type Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `protocol` | Signature format | `application/pgp-signature` |
| `micalg` | Hash algorithm | `pgp-sha256`, `pgp-sha512` |
| `boundary` | MIME part separator | Random string |

### What Gets Signed

```
┌─────────────────────────────────────────────────────────────────┐
│                   WHAT IS SIGNED                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ❌ NOT SIGNED (Outer headers):                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ From: sender@example.com                                │   │
│  │ To: recipient@example.com                               │   │
│  │ Subject: Important Message                              │   │
│  │ Date: Sat, 01 Feb 2026 12:00:00 +0000                   │   │
│  │ Content-Type: multipart/signed; ...                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ✅ SIGNED (First MIME part - exact bytes):                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Content-Type: text/plain; charset=utf-8\r\n             │   │
│  │ Content-Transfer-Encoding: quoted-printable\r\n         │   │
│  │ \r\n                                                    │   │
│  │ This is the email body.\r\n                             │   │
│  │ Every byte matters for verification.                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ⚠️  CRITICAL: Line endings must be CRLF (\r\n)                 │
│     Any change to whitespace invalidates the signature!        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Signing Flow in Nylas CLI

```
┌─────────────────────────────────────────────────────────────────┐
│              NYLAS CLI SIGNING FLOW                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  $ nylas email send --to user@example.com --sign                │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 1. DETERMINE SIGNING KEY                                │   │
│  │    Priority order:                                      │   │
│  │    a) --gpg-key flag (explicit)                         │   │
│  │    b) gpg.default_key from nylas config                 │   │
│  │    c) Key matching sender's email                       │   │
│  │    d) user.signingkey from git config                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 2. BUILD MIME CONTENT                                   │   │
│  │    • Create Content-Type header                         │   │
│  │    • Encode body (quoted-printable)                     │   │
│  │    • Normalize line endings to CRLF                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 3. CREATE SIGNATURE                                     │   │
│  │    $ gpg --detach-sign --armor                          │   │
│  │          --local-user <KEY_ID>                          │   │
│  │          --sender <FROM_EMAIL>                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 4. ASSEMBLE PGP/MIME MESSAGE                            │   │
│  │    • multipart/signed container                         │   │
│  │    • Part 1: Original content                           │   │
│  │    • Part 2: Detached signature                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 5. SEND VIA NYLAS API                                   │   │
│  │    POST /v3/grants/{id}/messages/send?type=mime         │   │
│  │    Body: Raw RFC 822 message                            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Signature Verification

### Verification Process

```
┌─────────────────────────────────────────────────────────────────┐
│                  VERIFICATION PROCESS                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  INPUT:                                                         │
│  ┌─────────────────┐    ┌─────────────────┐                    │
│  │ Signed Content  │    │   Signature     │                    │
│  │ (exact bytes)   │    │ (from email)    │                    │
│  └────────┬────────┘    └────────┬────────┘                    │
│           │                      │                              │
│           ▼                      ▼                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ STEP 1: Extract Key ID from Signature                   │   │
│  │                                                         │   │
│  │ The signature contains the Key ID of the signer         │   │
│  │ (NOT from the email's From header!)                     │   │
│  │                                                         │   │
│  │ Signature → Key ID: 601FEE9B1D60185F                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ STEP 2: Find Public Key                                 │   │
│  │                                                         │   │
│  │ Search order:                                           │   │
│  │ 1. Local keyring (~/.gnupg/pubring.kbx)                 │   │
│  │ 2. If not found → Fetch from key servers                │   │
│  │    • keys.openpgp.org                                   │   │
│  │    • keyserver.ubuntu.com                               │   │
│  │    • pgp.mit.edu                                        │   │
│  │    • keys.gnupg.net                                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ STEP 3: Decrypt Signature                               │   │
│  │                                                         │   │
│  │ Signature + Public Key → Original Hash                  │   │
│  │                                                         │   │
│  │ ┌───────────┐     ┌───────────┐     ┌───────────┐      │   │
│  │ │ Encrypted │  +  │  Public   │  =  │ Original  │      │   │
│  │ │   Hash    │     │    Key    │     │   Hash    │      │   │
│  │ └───────────┘     └───────────┘     └───────────┘      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ STEP 4: Hash the Content                                │   │
│  │                                                         │   │
│  │ Signed Content → SHA-256 → Computed Hash                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ STEP 5: Compare Hashes                                  │   │
│  │                                                         │   │
│  │ Original Hash  ══════════════════  Computed Hash        │   │
│  │     (from signature)                  (from content)    │   │
│  │                                                         │   │
│  │         │                                 │             │   │
│  │         └──────────── = ? ────────────────┘             │   │
│  │                       │                                 │   │
│  │                       ▼                                 │   │
│  │              ┌─────────────────┐                        │   │
│  │              │  MATCH = VALID  │                        │   │
│  │              │ NO MATCH = BAD  │                        │   │
│  │              └─────────────────┘                        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Verification Results

```
┌─────────────────────────────────────────────────────────────────┐
│                  VERIFICATION OUTCOMES                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ✅ GOOD SIGNATURE                                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ • Hashes match                                          │   │
│  │ • Content has not been modified                         │   │
│  │ • Signature was made with the claimed key               │   │
│  │                                                         │   │
│  │ Output:                                                 │   │
│  │ ────────────────────────────────────────────────────    │   │
│  │ ✓ Good signature                                        │   │
│  │ ────────────────────────────────────────────────────    │   │
│  │   Signer: John Doe <john@example.com>                   │   │
│  │   Key ID: 601FEE9B1D60185F                              │   │
│  │   Trust: ultimate                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ❌ BAD SIGNATURE                                               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ • Hashes do NOT match                                   │   │
│  │ • Content was modified after signing, OR                │   │
│  │ • Signature was tampered with                           │   │
│  │                                                         │   │
│  │ Output:                                                 │   │
│  │ ────────────────────────────────────────────────────    │   │
│  │ ✗ BAD signature                                         │   │
│  │ ────────────────────────────────────────────────────    │   │
│  │   Signer: John Doe <john@example.com>                   │   │
│  │   Key ID: 601FEE9B1D60185F                              │   │
│  │   ⚠️  Content may have been tampered with!              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ⚠️  NO PUBLIC KEY                                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ • Cannot find the signer's public key                   │   │
│  │ • Cannot verify the signature                           │   │
│  │                                                         │   │
│  │ Solution:                                               │   │
│  │ gpg --keyserver keys.openpgp.org --recv-keys <KEY_ID>   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Key ID vs From Address

**Important Security Note:**

```
┌─────────────────────────────────────────────────────────────────┐
│           KEY IDENTIFICATION - SECURITY WARNING                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  The signing key is identified by the KEY ID embedded in the    │
│  signature, NOT by the email's From header!                     │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Email Header:                                           │   │
│  │   From: alice@example.com  ←── Can be forged!           │   │
│  │                                                         │   │
│  │ Signature:                                              │   │
│  │   Key ID: 601FEE9B1D60185F  ←── Cryptographically bound │   │
│  │   UID: Bob <bob@example.com>  ←── Actual signer         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  This means:                                                    │
│  • An attacker can forge the From header                        │
│  • But they CANNOT forge a valid signature without the key      │
│  • Always verify the signer's UID matches who you expect        │
│                                                                 │
│  ⚠️  A "Good signature" only proves the key owner signed it.    │
│     It does NOT prove the From address is correct.              │
│     YOU must verify the signer's identity (UID) is trustworthy. │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Trust Model

### Web of Trust

GPG uses a decentralized trust model called the "Web of Trust":

```
┌─────────────────────────────────────────────────────────────────┐
│                      WEB OF TRUST                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│                         YOU                                     │
│                          │                                      │
│                     [ULTIMATE]                                  │
│                          │                                      │
│          ┌───────────────┼───────────────┐                     │
│          ▼               ▼               ▼                     │
│       ┌─────┐        ┌─────┐        ┌─────┐                    │
│       │Alice│        │ Bob │        │Carol│                    │
│       └──┬──┘        └──┬──┘        └──┬──┘                    │
│      [FULL]         [FULL]       [MARGINAL]                    │
│          │               │               │                      │
│     ┌────┴────┐     ┌────┴────┐     ┌────┴────┐                │
│     ▼         ▼     ▼         ▼     ▼         ▼                │
│  ┌─────┐  ┌─────┐ ┌─────┐  ┌─────┐ ┌─────┐  ┌─────┐           │
│  │Dave │  │Eve  │ │Frank│  │Grace│ │Henry│  │Ivy  │           │
│  └─────┘  └─────┘ └─────┘  └─────┘ └─────┘  └─────┘           │
│                                                                 │
│  Trust flows through the network based on your direct trust    │
│  assignments and the signatures (certifications) between keys. │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Trust Levels

| Level | Meaning | When Assigned |
|-------|---------|---------------|
| **Ultimate** | Your own keys | Automatically for your keys |
| **Full** | Completely trust this person's certifications | You verified their identity in person |
| **Marginal** | Somewhat trust their certifications | You have some confidence in them |
| **Undefined** | No trust decision made | Default for new keys |
| **Never** | Do not trust at all | Explicitly untrusted |

### Key Validity

```
┌─────────────────────────────────────────────────────────────────┐
│                    KEY VALIDITY CALCULATION                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  A key is considered VALID if:                                  │
│                                                                 │
│  1. You signed it yourself (ultimate trust), OR                 │
│                                                                 │
│  2. It has enough trusted signatures:                           │
│     • 1 fully trusted signature, OR                             │
│     • 3 marginally trusted signatures                           │
│                                                                 │
│  Example:                                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                                                         │   │
│  │  Unknown Key: Frank <frank@example.com>                 │   │
│  │                                                         │   │
│  │  Signed by:                                             │   │
│  │  • Alice (your FULL trust)     ←── 1 full = VALID      │   │
│  │                                                         │   │
│  │  OR                                                     │   │
│  │                                                         │   │
│  │  Signed by:                                             │   │
│  │  • Carol (MARGINAL)                                     │   │
│  │  • Dave (MARGINAL)                                      │   │
│  │  • Eve (MARGINAL)              ←── 3 marginal = VALID  │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Setting Trust

```bash
# Edit key trust level
gpg --edit-key <KEY_ID>
gpg> trust
# Select trust level (1-5)
gpg> save

# Sign someone's key (certify their identity)
gpg --sign-key <KEY_ID>
```

---

## Key Servers

### How Key Servers Work

```
┌─────────────────────────────────────────────────────────────────┐
│                    KEY SERVER NETWORK                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│                    ┌─────────────────┐                         │
│                    │  Your Computer  │                         │
│                    │   (GPG Client)  │                         │
│                    └────────┬────────┘                         │
│                             │                                   │
│              ┌──────────────┼──────────────┐                   │
│              │              │              │                    │
│              ▼              ▼              ▼                    │
│      ┌─────────────┐ ┌─────────────┐ ┌─────────────┐          │
│      │   keys.     │ │  keyserver. │ │   pgp.      │          │
│      │ openpgp.org │ │ ubuntu.com  │ │  mit.edu    │          │
│      └──────┬──────┘ └──────┬──────┘ └──────┬──────┘          │
│             │               │               │                   │
│             └───────────────┼───────────────┘                   │
│                             │                                   │
│                             ▼                                   │
│                    ┌─────────────────┐                         │
│                    │  Synchronization│                         │
│                    │    (SKS Pool)   │                         │
│                    └─────────────────┘                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Key Server Operations

```bash
# Upload your public key
gpg --keyserver keys.openpgp.org --send-keys <YOUR_KEY_ID>

# Download someone's public key
gpg --keyserver keys.openpgp.org --recv-keys <THEIR_KEY_ID>

# Search for a key by email
gpg --keyserver keys.openpgp.org --search-keys user@example.com

# Refresh all keys (update signatures, revocations)
gpg --keyserver keys.openpgp.org --refresh-keys
```

### Key Servers Used by Nylas CLI

| Server | Description | Privacy |
|--------|-------------|---------|
| `keys.openpgp.org` | Modern, privacy-focused | Email verification required |
| `keyserver.ubuntu.com` | Ubuntu's server | Open |
| `pgp.mit.edu` | MIT's classic server | Open |
| `keys.gnupg.net` | GnuPG project pool | Open |

### Privacy Considerations

```
┌─────────────────────────────────────────────────────────────────┐
│                    KEY SERVER PRIVACY                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Traditional Key Servers (MIT, Ubuntu, GnuPG):                  │
│  • Anyone can upload a key with any email                       │
│  • No email verification                                        │
│  • Cannot delete keys once uploaded                             │
│  • All signatures and UIDs are public                           │
│                                                                 │
│  Modern Key Servers (keys.openpgp.org):                         │
│  • Email verification required                                  │
│  • Can delete your key                                          │
│  • Only verified emails are searchable                          │
│  • Third-party signatures not distributed                       │
│                                                                 │
│  ⚠️  Once uploaded, consider your key ID permanently public.    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## CLI Integration

### Commands Reference

```bash
# ═══════════════════════════════════════════════════════════════
# SENDING SIGNED EMAILS
# ═══════════════════════════════════════════════════════════════

# Sign with default key (from git config or nylas config)
nylas email send --to user@example.com --subject "Hello" --body "..." --sign

# Sign with specific key
nylas email send --to user@example.com --subject "Hello" --body "..." \
    --sign --gpg-key 601FEE9B1D60185F

# List available signing keys
nylas email send --list-gpg-keys

# ═══════════════════════════════════════════════════════════════
# VERIFYING SIGNATURES
# ═══════════════════════════════════════════════════════════════

# Verify a signed email
nylas email read <message-id> --verify

# View raw MIME to inspect signature structure
nylas email read <message-id> --mime

# ═══════════════════════════════════════════════════════════════
# CONFIGURATION
# ═══════════════════════════════════════════════════════════════

# Set default signing key
nylas config set gpg.default_key 601FEE9B1D60185F

# Enable auto-sign for all emails
nylas config set gpg.auto_sign true

# View current GPG settings
nylas config get gpg.default_key
nylas config get gpg.auto_sign
```

### Key Selection Priority

```
┌─────────────────────────────────────────────────────────────────┐
│                  KEY SELECTION ORDER                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  When signing, the CLI selects a key in this order:             │
│                                                                 │
│  1. --gpg-key flag (highest priority)                           │
│     └── Explicit key ID from command line                       │
│                                                                 │
│  2. gpg.default_key from Nylas config                           │
│     └── nylas config set gpg.default_key <KEY_ID>               │
│                                                                 │
│  3. Key matching sender's email address                         │
│     └── Searches GPG keyring for matching UID                   │
│                                                                 │
│  4. user.signingkey from git config (lowest priority)           │
│     └── git config --global user.signingkey <KEY_ID>            │
│                                                                 │
│  5. Error if no key found                                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Security Considerations

### Best Practices

```
┌─────────────────────────────────────────────────────────────────┐
│                  SECURITY BEST PRACTICES                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  🔐 KEY PROTECTION                                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ • Use a strong passphrase (20+ characters)              │   │
│  │ • Never share your private key                          │   │
│  │ • Use gpg-agent to cache passphrase                     │   │
│  │ • Store backup in secure offline location               │   │
│  │ • Consider hardware key (YubiKey, etc.)                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  🔑 KEY MANAGEMENT                                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ • Set expiration dates (1-2 years recommended)          │   │
│  │ • Create revocation certificate immediately             │   │
│  │ • Use subkeys for daily operations                      │   │
│  │ • Rotate keys periodically                              │   │
│  │ • Revoke compromised keys immediately                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ✅ VERIFICATION                                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ • Verify key fingerprints out-of-band (in person)       │   │
│  │ • Check the signer's UID, not just "Good signature"     │   │
│  │ • Be aware that From headers can be forged              │   │
│  │ • Maintain healthy skepticism                           │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Common Attacks

| Attack | Description | Mitigation |
|--------|-------------|------------|
| **Key Spoofing** | Creating a key with someone else's name | Verify fingerprint in person |
| **From Header Forgery** | Sending email with fake From address | Check signer UID, not From header |
| **Signature Stripping** | Removing signature from email | Train users to expect signatures |
| **Downgrade Attack** | Convincing you to accept unsigned email | Require signatures for sensitive comms |

### What Signatures Do NOT Prove

```
┌─────────────────────────────────────────────────────────────────┐
│              LIMITATIONS OF SIGNATURES                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ❌ Does NOT prove the From header is accurate                  │
│  ❌ Does NOT prove the sender is who they claim to be           │
│     (only that they control the signing key)                    │
│  ❌ Does NOT encrypt the message (content is still readable)    │
│  ❌ Does NOT hide metadata (To, Subject, Date visible)          │
│  ❌ Does NOT prevent the recipient from forwarding              │
│                                                                 │
│  ✅ DOES prove the content hasn't been modified                 │
│  ✅ DOES prove the key owner signed it                          │
│  ✅ DOES provide timestamp of signing                           │
│  ✅ DOES enable non-repudiation (signer can't deny signing)     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quick Reference

### GPG Commands Cheat Sheet

```bash
# Key Management
gpg --gen-key                      # Generate new key pair
gpg --list-keys                    # List public keys
gpg --list-secret-keys             # List private keys
gpg --export -a KEY_ID > pub.asc   # Export public key
gpg --import key.asc               # Import a key
gpg --delete-key KEY_ID            # Delete public key
gpg --delete-secret-key KEY_ID     # Delete private key

# Signing
gpg --sign file.txt                # Create signed file
gpg --clearsign file.txt           # Create inline signed file
gpg --detach-sign file.txt         # Create detached signature
gpg --verify file.txt.sig file.txt # Verify detached signature

# Key Servers
gpg --send-keys KEY_ID             # Upload to key server
gpg --recv-keys KEY_ID             # Download from key server
gpg --search-keys email@example.com # Search key server

# Trust
gpg --edit-key KEY_ID              # Edit key (trust, sign, etc.)
gpg --sign-key KEY_ID              # Sign (certify) a key
```

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `GNUPGHOME` | GPG home directory (default: `~/.gnupg`) |
| `GPG_TTY` | Terminal for passphrase prompts |
| `GPG_AGENT_INFO` | GPG agent socket location |

---

## Further Reading

- [GPG Encrypted Email from the CLI](https://cli.nylas.com/guides/gpg-encrypted-email-cli) - Send and receive GPG/PGP encrypted email from your terminal
- [RFC 4880 - OpenPGP Message Format](https://tools.ietf.org/html/rfc4880)
- [RFC 3156 - MIME Security with OpenPGP](https://tools.ietf.org/html/rfc3156)
- [GnuPG Manual](https://www.gnupg.org/documentation/manuals/gnupg/)
- [Email Self-Defense (EFF)](https://emailselfdefense.fsf.org/)

---

**Last Updated:** 2026-02-01
