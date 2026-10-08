package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy_FailedNotificationIsNeverAnswered(t *testing.T) {
	// Every upstream call is refused, so each message fails to forward. Only
	// the two requests may get a reply; the notification between them must not.
	refuse := respondStatus(http.StatusUnauthorized, "")
	_, server := newOAuthTestProxy(t, refuse, refuse, refuse)
	proxy := NewOAuthProxy(&fakeCredentials{credentials: []*domain.MCPCredential{
		oauthCred(server, "a"), oauthCred(server, "b"), oauthCred(server, "c"),
	}})

	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":null,"method":"ping"}`,
	}, "\n") + "\n"
	var out bytes.Buffer

	require.NoError(t, proxy.serve(t.Context(), strings.NewReader(in), &out))

	var ids []any
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		var resp map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
		require.Contains(t, resp, "error")
		ids = append(ids, resp["id"])
	}
	assert.Equal(t, []any{float64(1), nil}, ids)
}

func TestIsNotification(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{"no id member", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, true},
		{"numeric id", `{"jsonrpc":"2.0","id":1,"method":"ping"}`, false},
		{"string id", `{"jsonrpc":"2.0","id":"a","method":"ping"}`, false},
		{"explicit null id is a request", `{"jsonrpc":"2.0","id":null,"method":"ping"}`, false},
		{"batch of notifications", `[{"jsonrpc":"2.0","method":"x"},{"jsonrpc":"2.0","method":"y"}]`, true},
		{"batch with a request", `[{"jsonrpc":"2.0","method":"x"},{"jsonrpc":"2.0","id":1,"method":"y"}]`, false},
		{"empty batch", `[]`, false},
		{"not an object", `"x"`, false},
		{"invalid JSON", `{`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isNotification([]byte(tt.message)))
		})
	}
}
