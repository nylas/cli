package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResetCmd(t *testing.T) {
	t.Run("command name and flags", func(t *testing.T) {
		cmd := newResetCmd()

		assert.Equal(t, "reset", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)

		flag := cmd.Flags().Lookup("force")
		require.NotNil(t, flag, "expected --force flag")
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestClearDashboardCredentials(t *testing.T) {
	t.Run("clears all dashboard keys", func(t *testing.T) {
		store := &memStore{data: map[string]string{
			"dashboard_user_token":     "tok",
			"dashboard_org_token":      "org-tok",
			"dashboard_user_public_id": "uid",
			"dashboard_org_public_id":  "oid",
			"dashboard_dpop_key":       "dpop",
			"dashboard_app_id":         "app",
			"dashboard_app_region":     "us",
			"api_key":                  "keep-me",
		}}

		clearDashboardCredentials(store)

		// Dashboard keys should be gone
		assert.Empty(t, store.data["dashboard_user_token"])
		assert.Empty(t, store.data["dashboard_org_token"])
		assert.Empty(t, store.data["dashboard_user_public_id"])
		assert.Empty(t, store.data["dashboard_org_public_id"])
		assert.Empty(t, store.data["dashboard_dpop_key"])
		assert.Empty(t, store.data["dashboard_app_id"])
		assert.Empty(t, store.data["dashboard_app_region"])

		// Non-dashboard keys should be untouched
		assert.Equal(t, "keep-me", store.data["api_key"])
	})
}

// memStore is a minimal in-memory SecretStore for testing.
type memStore struct {
	data map[string]string
}

func (m *memStore) Set(key, value string) error { m.data[key] = value; return nil }
func (m *memStore) Get(key string) (string, error) {
	return m.data[key], nil
}
func (m *memStore) Delete(key string) error { delete(m.data, key); return nil }
func (m *memStore) IsAvailable() bool       { return true }
func (m *memStore) Name() string             { return "mem" }
