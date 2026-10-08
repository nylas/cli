//go:build !integration

package domain

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unsignedJWT builds a token with the given payload and a junk signature. The
// decoder must not care about the signature, which is the point: it cannot
// verify one and must not pretend to.
func unsignedJWT(t *testing.T, payload any) string {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"at+jwt"}`))
	return header + "." + base64.RawURLEncoding.EncodeToString(body) + ".not-a-real-signature"
}

func TestDefaultOAuthClientID_IsValid(t *testing.T) {
	require.NoError(t, ValidateOAuthClientID(DefaultOAuthClientID))
}

func TestValidateOAuthClientID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"uuid", "b3a94d82-fc7d-4a22-803e-e603ae0f735c", false},
		{"dev id", "dev.client_1", false},
		{"empty", "", true},
		{"space", "a b", true},
		{"form metacharacter", "a&client_secret=x", true},
		{"too long", strings.Repeat("a", 129), true},
		{"newline", "abc\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOAuthClientID(tt.id)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrOAuthInvalidClientID)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateOAuthLoopbackRedirectURI(t *testing.T) {
	tests := []struct {
		uri     string
		wantErr bool
	}{
		{"http://127.0.0.1:53682/callback", false},
		{"http://localhost:9007/callback", false},
		{"http://127.0.0.1/callback", true},       // the port is what the client fills in
		{"https://127.0.0.1:1234/callback", true}, // not registered
		{"http://[::1]:1234/callback", true},      // not registered
		{"http://127.0.0.2:1234/callback", true},  // not registered
		{"http://127.0.0.1:1234/callback/", true}, // path must match exactly
		{"http://127.0.0.1:1234/callback?x=1", true},
		{"http://user@127.0.0.1:1234/callback", true},
		{"http://127.0.0.1:0/callback", true},
		{"http://127.0.0.1:70000/callback", true},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			err := ValidateOAuthLoopbackRedirectURI(tt.uri)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrOAuthRedirectURI)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDecodeOAuthAccessToken_ReadsTheContractClaims(t *testing.T) {
	token := unsignedJWT(t, map[string]any{
		"sub":       "user-1",
		"aud":       "https://mcp.us.nylas.com",
		"scope":     "email.read calendar.read offline_access",
		"org":       "org-1",
		"client_id": DefaultOAuthClientID,
		"exp":       1790000000,
		"grants": []map[string]string{
			{"id": "grant-1", "application_id": "app-1"},
			{"id": "grant-2", "application_id": "app-2"},
		},
	})

	claims, err := DecodeOAuthAccessToken(token)
	require.NoError(t, err)

	assert.Equal(t, "user-1", claims.Subject)
	assert.Equal(t, []string{"https://mcp.us.nylas.com"}, claims.Audience)
	assert.Equal(t, []string{"email.read", "calendar.read", "offline_access"}, claims.Scopes())
	assert.Equal(t, "org-1", claims.Org)
	assert.Equal(t, DefaultOAuthClientID, claims.ClientID)
	assert.Equal(t, time.Unix(1790000000, 0).UTC(), claims.ExpiresAt)
	assert.Equal(t, []OAuthTokenGrant{{ID: "grant-1", ApplicationID: "app-1"}, {ID: "grant-2", ApplicationID: "app-2"}}, claims.Grants)
	assert.True(t, claims.HasGrant("grant-2"))
	assert.False(t, claims.HasGrant("grant-3"))
	assert.False(t, claims.HasGrant(""))
}

func TestDecodeOAuthAccessToken_AcceptsAudienceList(t *testing.T) {
	claims, err := DecodeOAuthAccessToken(unsignedJWT(t, map[string]any{
		"aud": []string{"https://mcp.eu.nylas.com", "other"},
	}))
	require.NoError(t, err)
	assert.Equal(t, []string{"https://mcp.eu.nylas.com", "other"}, claims.Audience)
	assert.True(t, claims.ExpiresAt.IsZero())
}

func TestDecodeOAuthAccessToken_RejectsWithoutLeakingTheToken(t *testing.T) {
	secretish := "opaque-secret-token-value"
	for name, token := range map[string]string{
		"opaque":      secretish,
		"two parts":   "a." + secretish,
		"bad base64":  "a.!!!" + secretish + ".c",
		"not json":    "a." + base64.RawURLEncoding.EncodeToString([]byte(secretish)) + ".c",
		"bad aud":     unsignedJWT(t, map[string]any{"aud": 42}),
		"bad exp":     unsignedJWT(t, map[string]any{"exp": "soon"}),
		"float exp":   unsignedJWT(t, map[string]any{"exp": 1.5}),
		"empty token": "",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeOAuthAccessToken(token)
			require.ErrorIs(t, err, ErrOAuthTokenNotJWT)
			assert.NotContains(t, err.Error(), secretish)
		})
	}
}
