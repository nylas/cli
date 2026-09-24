//go:build !integration

package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serverChallengePattern is the regex dashboard-account enforces on
// code_challenge. A challenge that fails it is rejected before the
// authorization request is even considered, so this is the contract.
var serverChallengePattern = regexp.MustCompile(`^[A-Za-z0-9\-_]{43}$`)

func TestNewPKCE_ChallengeSatisfiesServerPattern(t *testing.T) {
	pkce, err := NewPKCE()
	require.NoError(t, err)

	assert.Regexp(t, serverChallengePattern, pkce.Challenge,
		"authorization server rejects any challenge outside this shape")
}

func TestNewPKCE_ChallengeIsRFC7636S256(t *testing.T) {
	pkce, err := NewPKCE()
	require.NoError(t, err)

	// RFC 7636 4.2: challenge = base64url(sha256(ASCII(verifier))), unpadded.
	sum := sha256.Sum256([]byte(pkce.Verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])

	assert.Equal(t, want, pkce.Challenge)
}

func TestNewPKCE_VerifierLengthWithinRFCRange(t *testing.T) {
	pkce, err := NewPKCE()
	require.NoError(t, err)

	// RFC 7636 4.1 allows 43-128 characters.
	assert.GreaterOrEqual(t, len(pkce.Verifier), 43)
	assert.LessOrEqual(t, len(pkce.Verifier), 128)
}

func TestNewPKCE_IsNotTheNylasHostedAuthEncoding(t *testing.T) {
	pkce, err := NewPKCE()
	require.NoError(t, err)

	// app/auth uses base64std(hex(sha256(v))) for Nylas hosted auth. That
	// form is 88 characters and would be refused here; guard the divergence
	// so the two flows cannot be collapsed by a well-meaning refactor.
	sum := sha256.Sum256([]byte(pkce.Verifier))
	hosted := base64.RawStdEncoding.EncodeToString([]byte(hexString(sum[:])))

	assert.NotEqual(t, hosted, pkce.Challenge)
}

func hexString(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&0x0f])
	}
	return string(out)
}

func TestNewPKCE_IsRandomPerCall(t *testing.T) {
	first, err := NewPKCE()
	require.NoError(t, err)
	second, err := NewPKCE()
	require.NoError(t, err)

	assert.NotEqual(t, first.Verifier, second.Verifier)
	assert.NotEqual(t, first.Challenge, second.Challenge)
}

func TestNewOAuthState_IsRandomAndURLSafe(t *testing.T) {
	first, err := NewOAuthState()
	require.NoError(t, err)
	second, err := NewOAuthState()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.Regexp(t, `^[A-Za-z0-9\-_]+$`, first)
}

func TestOAuthTokens_IsExpired(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"unknown expiry is treated as expired", time.Time{}, true},
		{"already past", now.Add(-time.Second), true},
		{"inside the leeway window", now.Add(10 * time.Second), true},
		{"comfortably valid", now.Add(time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := &OAuthTokens{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.want, tokens.IsExpired(now))
		})
	}
}

func TestOAuthServerMetadata_Validate(t *testing.T) {
	valid := func() *OAuthServerMetadata {
		return &OAuthServerMetadata{
			Issuer:                        "https://example.test",
			AuthorizationEndpoint:         "https://example.test/oauth/authorize",
			TokenEndpoint:                 "https://example.test/oauth/token",
			CodeChallengeMethodsSupported: []string{"S256"},
		}
	}

	t.Run("accepts a complete document", func(t *testing.T) {
		require.NoError(t, valid().Validate())
	})

	t.Run("rejects a document with no token endpoint", func(t *testing.T) {
		metadata := valid()
		metadata.TokenEndpoint = ""

		err := metadata.Validate()

		require.ErrorIs(t, err, ErrOAuthMetadata)
		assert.Contains(t, err.Error(), "token_endpoint")
	})

	t.Run("rejects a server without S256", func(t *testing.T) {
		metadata := valid()
		metadata.CodeChallengeMethodsSupported = []string{"plain"}

		err := metadata.Validate()

		require.ErrorIs(t, err, ErrOAuthMetadata)
		assert.Contains(t, err.Error(), "S256")
	})
}

func TestOAuthError_Error(t *testing.T) {
	withDescription := &OAuthError{Code: "invalid_grant", Description: "code expired", StatusCode: 400}
	assert.Contains(t, withDescription.Error(), "invalid_grant")
	assert.Contains(t, withDescription.Error(), "code expired")

	bare := &OAuthError{Code: "invalid_client", StatusCode: 401}
	assert.Contains(t, bare.Error(), "invalid_client")
}

func TestDefaultOAuthScopes_RequestsOfflineAccess(t *testing.T) {
	// offline_access is the only way the server issues a refresh token;
	// without it every login would need the browser again in an hour.
	assert.Contains(t, DefaultOAuthScopes(), OAuthScopeOfflineAccess)
}
