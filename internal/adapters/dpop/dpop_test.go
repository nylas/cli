package dpop

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSecretStore is a simple in-memory secret store for testing.
type mockSecretStore struct {
	data map[string]string
}

func newMockSecretStore() *mockSecretStore {
	return &mockSecretStore{data: make(map[string]string)}
}

func (m *mockSecretStore) Set(key, value string) error { m.data[key] = value; return nil }
func (m *mockSecretStore) Get(key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", nil
	}
	return v, nil
}
func (m *mockSecretStore) Delete(key string) error { delete(m.data, key); return nil }
func (m *mockSecretStore) IsAvailable() bool       { return true }
func (m *mockSecretStore) Name() string            { return "mock" }

func TestNew_GeneratesKey(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()

	svc, err := New(store)
	require.NoError(t, err)
	require.NotNil(t, svc)

	// Key should be persisted
	seedB64, err := store.Get("dashboard_dpop_key")
	require.NoError(t, err)
	assert.NotEmpty(t, seedB64)

	// Seed should be 32 bytes
	seed, err := base64.StdEncoding.DecodeString(seedB64)
	require.NoError(t, err)
	assert.Len(t, seed, ed25519.SeedSize)
}

func TestNew_LoadsExistingKey(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()

	// Create first instance
	svc1, err := New(store)
	require.NoError(t, err)
	thumb1 := svc1.Thumbprint()

	// Create second instance - should load same key
	svc2, err := New(store)
	require.NoError(t, err)
	thumb2 := svc2.Thumbprint()

	assert.Equal(t, thumb1, thumb2, "reloaded key should produce same thumbprint")
}

func TestGenerateProof_Structure(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()
	svc, err := New(store)
	require.NoError(t, err)

	proof, err := svc.GenerateProof("POST", "https://example.com/auth/cli/login", "")
	require.NoError(t, err)

	parts := strings.Split(proof, ".")
	require.Len(t, parts, 3, "JWT should have 3 parts")

	// Decode and verify header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)

	var header jwtHeader
	require.NoError(t, json.Unmarshal(headerJSON, &header))
	assert.Equal(t, "dpop+jwt", header.Typ)
	assert.Equal(t, "EdDSA", header.Alg)
	require.NotNil(t, header.JWK)
	assert.Equal(t, "OKP", header.JWK.Kty)
	assert.Equal(t, "Ed25519", header.JWK.Crv)
	assert.NotEmpty(t, header.JWK.X)

	// Decode and verify claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims jwtClaims
	require.NoError(t, json.Unmarshal(claimsJSON, &claims))
	assert.NotEmpty(t, claims.JTI, "jti must be present")
	assert.Equal(t, "POST", claims.HTM)
	assert.Equal(t, "https://example.com/auth/cli/login", claims.HTU)
	assert.NotZero(t, claims.IAT)
	assert.Empty(t, claims.ATH, "ath should be empty when no access token")
}

func TestGenerateProof_WithAccessToken(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()
	svc, err := New(store)
	require.NoError(t, err)

	accessToken := "test-access-token"
	proof, err := svc.GenerateProof("GET", "https://example.com/api", accessToken)
	require.NoError(t, err)

	parts := strings.Split(proof, ".")
	require.Len(t, parts, 3)

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims jwtClaims
	require.NoError(t, json.Unmarshal(claimsJSON, &claims))
	assert.NotEmpty(t, claims.ATH, "ath should be present when access token provided")

	// Verify ath is SHA-256 of the access token
	expectedHash := sha256.Sum256([]byte(accessToken))
	expectedATH := base64.RawURLEncoding.EncodeToString(expectedHash[:])
	assert.Equal(t, expectedATH, claims.ATH)
}

func TestGenerateProof_SignatureVerifies(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()
	svc, err := New(store)
	require.NoError(t, err)

	proof, err := svc.GenerateProof("POST", "https://example.com/test", "")
	require.NoError(t, err)

	parts := strings.Split(proof, ".")
	require.Len(t, parts, 3)

	// Extract public key from header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)

	var header jwtHeader
	require.NoError(t, json.Unmarshal(headerJSON, &header))

	pubKeyBytes, err := base64.RawURLEncoding.DecodeString(header.JWK.X)
	require.NoError(t, err)
	pubKey := ed25519.PublicKey(pubKeyBytes)

	// Verify signature
	signingInput := []byte(parts[0] + "." + parts[1])
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	assert.True(t, ed25519.Verify(pubKey, signingInput, signature), "signature should verify")
}

func TestGenerateProof_UniqueJTI(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()
	svc, err := New(store)
	require.NoError(t, err)

	proof1, err := svc.GenerateProof("POST", "https://example.com/test", "")
	require.NoError(t, err)

	proof2, err := svc.GenerateProof("POST", "https://example.com/test", "")
	require.NoError(t, err)

	// Extract JTIs
	jti1 := extractClaim(t, proof1, "jti")
	jti2 := extractClaim(t, proof2, "jti")

	assert.NotEqual(t, jti1, jti2, "each proof should have a unique jti")
}

func TestGenerateProof_MethodUppercased(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()
	svc, err := New(store)
	require.NoError(t, err)

	proof, err := svc.GenerateProof("post", "https://example.com/test", "")
	require.NoError(t, err)

	htm := extractClaim(t, proof, "htm")
	assert.Equal(t, "POST", htm)
}

func TestGenerateProof_URLNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		inputURL string
		expected string
	}{
		{
			name:     "strips fragment",
			inputURL: "https://example.com/path#fragment",
			expected: "https://example.com/path",
		},
		{
			name:     "strips query",
			inputURL: "https://example.com/path?key=value",
			expected: "https://example.com/path",
		},
		{
			name:     "preserves path",
			inputURL: "https://example.com/auth/cli/login",
			expected: "https://example.com/auth/cli/login",
		},
	}

	store := newMockSecretStore()
	svc, err := New(store)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proof, err := svc.GenerateProof("POST", tt.inputURL, "")
			require.NoError(t, err)

			htu := extractClaim(t, proof, "htu")
			assert.Equal(t, tt.expected, htu)
		})
	}
}

func TestThumbprint_Consistent(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()
	svc, err := New(store)
	require.NoError(t, err)

	thumb1 := svc.Thumbprint()
	thumb2 := svc.Thumbprint()

	assert.NotEmpty(t, thumb1)
	assert.Equal(t, thumb1, thumb2, "thumbprint should be deterministic")
}

func TestThumbprint_MatchesRFC7638(t *testing.T) {
	t.Parallel()
	store := newMockSecretStore()
	svc, err := New(store)
	require.NoError(t, err)

	// Manually compute the expected thumbprint
	xB64 := base64urlEncode(svc.publicKey)
	canonical := `{"crv":"Ed25519","kty":"OKP","x":"` + xB64 + `"}`
	hash := sha256.Sum256([]byte(canonical))
	expected := base64urlEncode(hash[:])

	assert.Equal(t, expected, svc.Thumbprint())
}

// extractClaim extracts a string claim value from a JWT proof.
func extractClaim(t *testing.T, proof, key string) string {
	t.Helper()
	parts := strings.Split(proof, ".")
	require.Len(t, parts, 3)

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(claimsJSON, &raw))

	val, ok := raw[key]
	require.True(t, ok, "claim %q not found", key)

	str, ok := val.(string)
	if ok {
		return str
	}
	return ""
}
