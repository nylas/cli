// Package dpop implements DPoP (Demonstrating Proof-of-Possession) proof
// generation using Ed25519 keys for CLI authentication.
package dpop

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// Service implements the ports.DPoP interface using Ed25519 keys.
type Service struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	thumbprint string
}

// New creates a DPoP service, loading an existing key from the secret store
// or generating a new Ed25519 keypair if none exists.
func New(secrets ports.SecretStore) (*Service, error) {
	s := &Service{}

	// Try to load existing key
	seedB64, err := secrets.Get(ports.KeyDashboardDPoPKey)
	if err == nil && seedB64 != "" {
		seed, decErr := base64.StdEncoding.DecodeString(seedB64)
		if decErr == nil && len(seed) == ed25519.SeedSize {
			s.privateKey = ed25519.NewKeyFromSeed(seed)
			s.publicKey = s.privateKey.Public().(ed25519.PublicKey)
			s.thumbprint = s.computeThumbprint()
			return s, nil
		}
	}

	// Generate new keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrDashboardDPoP, err)
	}
	s.privateKey = priv
	s.publicKey = pub

	// Persist the seed (first 32 bytes of the 64-byte private key)
	seed := priv.Seed()
	if err := secrets.Set(ports.KeyDashboardDPoPKey, base64.StdEncoding.EncodeToString(seed)); err != nil {
		return nil, fmt.Errorf("failed to store DPoP key: %w", err)
	}

	s.thumbprint = s.computeThumbprint()
	return s, nil
}

// GenerateProof creates a DPoP proof JWT for the given HTTP method and URL.
// If accessToken is non-empty, the proof includes an ath claim.
func (s *Service) GenerateProof(method, rawURL string, accessToken string) (string, error) {
	// Normalize the URL: strip fragment and query
	htu, err := normalizeHTU(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: invalid URL: %w", domain.ErrDashboardDPoP, err)
	}

	// Build header
	header := jwtHeader{
		Typ: "dpop+jwt",
		Alg: "EdDSA",
		JWK: &jwkOKP{
			Kty: "OKP",
			Crv: "Ed25519",
			X:   base64urlEncode(s.publicKey),
		},
	}

	// Build claims
	claims := jwtClaims{
		JTI: uuid.NewString(),
		HTM: strings.ToUpper(method),
		HTU: htu,
		IAT: time.Now().Unix(),
	}

	// Add access token hash if provided
	if accessToken != "" {
		hash := sha256.Sum256([]byte(accessToken))
		claims.ATH = base64urlEncode(hash[:])
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrDashboardDPoP, err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrDashboardDPoP, err)
	}

	// Create signing input
	headerB64 := base64urlEncode(headerJSON)
	claimsB64 := base64urlEncode(claimsJSON)
	signingInput := headerB64 + "." + claimsB64

	// Sign with Ed25519
	signature := ed25519.Sign(s.privateKey, []byte(signingInput))

	return signingInput + "." + base64urlEncode(signature), nil
}

// Thumbprint returns the JWK thumbprint (RFC 7638) of the DPoP public key.
func (s *Service) Thumbprint() string {
	return s.thumbprint
}

// computeThumbprint computes the RFC 7638 JWK thumbprint.
// For OKP keys, the canonical JSON uses lexicographically sorted members:
// {"crv":"Ed25519","kty":"OKP","x":"<base64url>"}
func (s *Service) computeThumbprint() string {
	canonical := fmt.Sprintf(
		`{"crv":"Ed25519","kty":"OKP","x":"%s"}`,
		base64urlEncode(s.publicKey),
	)
	hash := sha256.Sum256([]byte(canonical))
	return base64urlEncode(hash[:])
}

// normalizeHTU strips the fragment and query from a URL per DPoP spec.
func normalizeHTU(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.Fragment = ""
	u.RawQuery = ""
	return u.String(), nil
}

// base64urlEncode encodes bytes as base64url without padding.
func base64urlEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// JWT types for serialization.

type jwtHeader struct {
	Typ string  `json:"typ"`
	Alg string  `json:"alg"`
	JWK *jwkOKP `json:"jwk"`
}

type jwkOKP struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
}

type jwtClaims struct {
	JTI string `json:"jti"`
	HTM string `json:"htm"`
	HTU string `json:"htu"`
	IAT int64  `json:"iat"`
	ATH string `json:"ath,omitempty"`
}
