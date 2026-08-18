package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionsCleanupCommandScopesEmailSelectorToSender(t *testing.T) {
	var deleted []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "grant-test", "provider": "google"}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/folders":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "trash-id", "name": "Trash"}}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/messages":
			assert.Equal(t, "trash-id", r.URL.Query().Get("in"))
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{
					"id": "marketing", "date": time.Now().Unix(),
					"from": []map[string]string{{"email": "marketing@example.com"}},
					"headers": []map[string]string{
						{"name": "List-ID", "value": "shared.example.com"},
						{"name": "List-Unsubscribe", "value": "<https://example.com/marketing>"},
					},
				},
				{
					"id": "alerts", "date": time.Now().Add(-time.Hour).Unix(),
					"from": []map[string]string{{"email": "alerts@example.com"}},
					"headers": []map[string]string{
						{"name": "List-ID", "value": "shared.example.com"},
						{"name": "List-Unsubscribe", "value": "<https://example.com/alerts>"},
					},
				},
			}})
		case r.Method == http.MethodDelete:
			deleted = append(deleted, r.URL.Path)
			assert.Equal(t, "true", r.URL.Query().Get("hard_delete"))
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	stdout, stderr, err := executeSubscriptionsTestCommand(t, server.URL,
		"subscriptions", "cleanup", "marketing@example.com", "--permanent", "--yes", "--json",
	)

	require.NoError(t, err, stderr)
	assert.Equal(t, []string{"/v3/grants/grant-test/messages/marketing"}, deleted)
	assert.Contains(t, stdout, "Permanently deleted 1/1")
}

func TestSubscriptionsCleanupPermanentFailsClosedWhenTrashMissing(t *testing.T) {
	var messageGets, deletes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "grant-test", "provider": "microsoft"}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/folders":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/messages":
			messageGets++
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		case r.Method == http.MethodDelete:
			deletes++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	_, _, err := executeSubscriptionsTestCommand(t, server.URL,
		"subscriptions", "cleanup", "news@example.com", "--permanent", "--yes", "--json",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Could not resolve the required folder")
	assert.Zero(t, messageGets)
	assert.Zero(t, deletes)
}

func TestSubscriptionsCleanupDryRunDoesNotDelete(t *testing.T) {
	var deletes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "grant-test", "provider": "google"}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/folders":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "inbox-id", "name": "Inbox"}}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/messages":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{
				"id": "m1", "date": time.Now().Unix(),
				"from":    []map[string]string{{"email": "news@example.com"}},
				"headers": []map[string]string{{"name": "List-Unsubscribe", "value": "<https://example.com/unsubscribe>"}},
			}}})
		case r.Method == http.MethodDelete:
			deletes++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	stdout, stderr, err := executeSubscriptionsTestCommand(t, server.URL,
		"subscriptions", "cleanup", "news@example.com", "--dry-run", "--json",
	)

	require.NoError(t, err, stderr)
	assert.Zero(t, deletes)
	assert.Contains(t, stdout, "Would move 1 to Trash")
}

func TestSubscriptionsCleanupRejectedConfirmationDoesNotDelete(t *testing.T) {
	var deletes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "grant-test", "provider": "google"}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/folders":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "trash-id", "name": "Trash"}}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/messages":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{
				"id": "m1", "date": time.Now().Unix(),
				"from":    []map[string]string{{"email": "news@example.com"}},
				"headers": []map[string]string{{"name": "List-Unsubscribe", "value": "<https://example.com/unsubscribe>"}},
			}}})
		case r.Method == http.MethodDelete:
			deletes++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	input, writer, err := os.Pipe()
	require.NoError(t, err)
	_, err = writer.WriteString("n\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	oldStdin := os.Stdin
	os.Stdin = input
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = input.Close()
	})

	stdout, stderr, err := executeSubscriptionsTestCommand(t, server.URL,
		"subscriptions", "cleanup", "news@example.com", "--permanent",
	)

	require.NoError(t, err, stderr)
	assert.Zero(t, deletes)
	assert.Contains(t, stdout, "Cancelled.")
}

func TestFetchEmailSubscriptionsRequiredFolderFailsClosedOnLookupError(t *testing.T) {
	client := nylas.NewMockClient()
	client.GetGrantFunc = func(context.Context, string) (*domain.Grant, error) {
		return &domain.Grant{Provider: domain.ProviderGoogle}, nil
	}
	client.GetFoldersFunc = func(context.Context, string) ([]domain.Folder, error) {
		return nil, errors.New("folder lookup failed")
	}

	_, err := fetchEmailSubscriptions(context.Background(), newSubscriptionsListCmd(), client, "grant-test", subscriptionListOptions{
		limit:          10,
		since:          24 * time.Hour,
		folder:         "TRASH",
		folderRequired: true,
	}, time.Now())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Could not resolve the required folder")
	assert.False(t, client.GetMessagesWithParamsCalled)
}

func executeSubscriptionsTestCommand(t *testing.T, baseURL string, args ...string) (string, string, error) {
	t.Helper()
	common.ResetCachedClient()
	t.Cleanup(common.ResetCachedClient)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_API_KEY", "test-api-key")
	t.Setenv("NYLAS_GRANT_ID", "grant-test")
	t.Setenv("NYLAS_API_BASE_URL", baseURL)

	root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
	common.AddOutputFlags(root)
	root.AddCommand(newSubscriptionsCmd())
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}
