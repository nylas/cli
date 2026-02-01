package gpg

import "time"

// KeyInfo represents a GPG signing key.
type KeyInfo struct {
	KeyID       string     // Short key ID (e.g., "601FEE9B1D60185F")
	Fingerprint string     // Full fingerprint
	UIDs        []string   // User IDs (typically email addresses)
	Trust       string     // Trust level (ultimate, full, marginal, unknown)
	Expires     *time.Time // Expiration date (nil if no expiration)
	Type        string     // Key type (RSA, DSA, ECDSA, EdDSA)
	Length      int        // Key length in bits
	Created     time.Time  // Creation date
}

// SignResult contains the result of a signing operation.
type SignResult struct {
	Signature []byte    // Detached signature (ASCII armored)
	KeyID     string    // Key ID used for signing
	SignedAt  time.Time // Signature timestamp
	HashAlgo  string    // Hash algorithm used (e.g., "SHA256")
}

// VerifyResult contains the result of a signature verification.
type VerifyResult struct {
	Valid       bool      // Whether the signature is valid
	SignerKeyID string    // Key ID that created the signature
	SignerUID   string    // Primary UID of the signer (e.g., "Name <email@example.com>")
	SignedAt    time.Time // When the signature was created
	TrustLevel  string    // Trust level (ultimate, full, marginal, unknown, undefined)
	Fingerprint string    // Full fingerprint of signing key
}
