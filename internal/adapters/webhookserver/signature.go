package webhookserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// NormalizeSignature trims spaces and accepts optional sha256= prefixes.
func NormalizeSignature(signature string) string {
	signature = strings.ToLower(strings.TrimSpace(signature))
	return strings.TrimPrefix(signature, "sha256=")
}

// ComputeSignature computes a hex-encoded HMAC-SHA256 signature for a payload.
func ComputeSignature(payload []byte, webhookSecret string) string {
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature checks that the payload matches the provided signature.
func VerifySignature(payload []byte, signature, webhookSecret string) bool {
	if webhookSecret == "" {
		return false
	}
	expected := ComputeSignature(payload, webhookSecret)
	normalizedSignature := NormalizeSignature(signature)
	return hmac.Equal([]byte(normalizedSignature), []byte(expected))
}
