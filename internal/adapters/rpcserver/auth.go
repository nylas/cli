package rpcserver

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	// KeyRPCSessionToken is the SecretStore key for the brokered token.
	KeyRPCSessionToken = "rpc_session_token"
	// EnvWSToken overrides the stored token for headless or file-backed setups.
	EnvWSToken = "NYLAS_WS_TOKEN"
)

// GenerateToken returns a cryptographically random URL-safe token.
func GenerateToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("generate rpc session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

// ResolveToken returns the session token from env, storage, or a newly persisted token.
func ResolveToken(store ports.SecretStore, getenv func(string) string) (string, error) {
	if token := getenv(EnvWSToken); token != "" {
		return token, nil
	}

	token, err := store.Get(KeyRPCSessionToken)
	if err != nil && !errors.Is(err, domain.ErrSecretNotFound) {
		return "", fmt.Errorf("get rpc session token: %w", err)
	}
	if token != "" {
		return token, nil
	}

	token, err = GenerateToken()
	if err != nil {
		return "", err
	}
	if err := store.Set(KeyRPCSessionToken, token); err != nil {
		return "", fmt.Errorf("set rpc session token: %w", err)
	}
	return token, nil
}

// ValidateToken does a constant-time comparison. Empty tokens are rejected.
// Both tokens are hashed to a fixed-length digest first so the comparison does
// not leak the token length via timing (ConstantTimeCompare returns early when
// the inputs differ in length).
func ValidateToken(expected, provided string) bool {
	if expected == "" || provided == "" {
		return false
	}
	he := sha256.Sum256([]byte(expected))
	hp := sha256.Sum256([]byte(provided))
	return subtle.ConstantTimeCompare(he[:], hp[:]) == 1
}

// ValidateOrigin returns true if origin is allowed. Empty origin is allowed for non-browser clients; the token is the gate.
func ValidateOrigin(origin string, allowed []string) bool {
	if origin == "" {
		return true
	}
	for _, candidate := range allowed {
		if origin == candidate {
			return true
		}
	}
	return false
}

// IsLoopback reports whether a bind address's host is loopback.
func IsLoopback(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false, fmt.Errorf("parse bind address: %w", err)
	}
	if host == "localhost" {
		return true, nil
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback(), nil
}
