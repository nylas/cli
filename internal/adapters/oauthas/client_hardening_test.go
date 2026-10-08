//go:build !integration

package oauthas

import (
	"context"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ExpiryFallsBackToTheTokensExpClaim(t *testing.T) {
	// expires_in is only RECOMMENDED. Without it a zero expiry reads as
	// expired, and every command would spend the refresh token.
	exp := time.Now().Add(15 * time.Minute).Truncate(time.Second).UTC()
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"exp":` + strconv.FormatInt(exp.Unix(), 10) + `}`))
	server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"h.` + payload + `.s","token_type":"Bearer"}`))
	})

	tokens, err := NewClient(server.URL).ExchangeCode(context.Background(), domain.OAuthCodeExchange{ClientID: "c"})

	require.NoError(t, err)
	assert.True(t, exp.Equal(tokens.ExpiresAt), "got %s, want %s", tokens.ExpiresAt, exp)
}

func TestClient_ServerErrorTextCannotRewriteTheTerminal(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant\u001b[2J","error_description":"bad\u001b]0;pwned\u0007\r\nforged"}`))
	})

	_, err := NewClient(server.URL).ExchangeCode(context.Background(), domain.OAuthCodeExchange{ClientID: "c"})

	var oauthErr *domain.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	for _, text := range []string{oauthErr.Code, oauthErr.Description, err.Error()} {
		assert.False(t, strings.ContainsAny(text, "\x1b\x07\r\n"), "%q", text)
	}
	assert.Contains(t, oauthErr.Description, "forged")
}
