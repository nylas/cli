package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConnectorClient implements just enough of ports.NylasClient to exercise
// resolveConnectorID; the embedded interface panics if anything else is called.
type fakeConnectorClient struct {
	ports.NylasClient

	connectors []domain.Connector
	err        error
	calls      int
}

func (f *fakeConnectorClient) ListConnectors(ctx context.Context) ([]domain.Connector, error) {
	f.calls++
	return f.connectors, f.err
}

func TestResolveConnectorID(t *testing.T) {
	t.Run("explicit provider is preserved", func(t *testing.T) {
		client := &fakeConnectorClient{connectors: []domain.Connector{{ID: "connector-1", Provider: "microsoft"}}}
		got, err := resolveConnectorID(context.Background(), client, "microsoft")
		require.NoError(t, err)
		assert.Equal(t, "microsoft", got)
		assert.Equal(t, 1, client.calls)
	})

	t.Run("legacy connector ID maps to provider", func(t *testing.T) {
		client := &fakeConnectorClient{connectors: []domain.Connector{{ID: "connector-1", Provider: "google"}}}
		got, err := resolveConnectorID(context.Background(), client, "connector-1")
		require.NoError(t, err)
		assert.Equal(t, "google", got)
		assert.Equal(t, 1, client.calls)
	})

	t.Run("explicit provider survives connector discovery failure", func(t *testing.T) {
		client := &fakeConnectorClient{err: errors.New("boom")}
		got, err := resolveConnectorID(context.Background(), client, "google")
		require.NoError(t, err)
		assert.Equal(t, "google", got)
	})

	t.Run("explicit deprecated provider is rejected", func(t *testing.T) {
		// The explicit path must not leak a deprecated provider that auto-detect
		// and listings hide; otherwise --connector inbox would hit the API.
		client := &fakeConnectorClient{connectors: []domain.Connector{{Provider: "google"}}}
		_, err := resolveConnectorID(context.Background(), client, "inbox")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer supported")
	})

	t.Run("sole connector is auto-detected by provider", func(t *testing.T) {
		client := &fakeConnectorClient{connectors: []domain.Connector{{Provider: "google"}}}
		got, err := resolveConnectorID(context.Background(), client, "")
		require.NoError(t, err)
		assert.Equal(t, "google", got)
	})

	t.Run("deprecated connectors are ignored when auto-detecting", func(t *testing.T) {
		// A single visible connector alongside a hidden "inbox" connector must
		// still auto-detect the visible one, not error as ambiguous.
		client := &fakeConnectorClient{connectors: []domain.Connector{
			{Provider: "inbox"},
			{Provider: "google"},
		}}
		got, err := resolveConnectorID(context.Background(), client, "")
		require.NoError(t, err)
		assert.Equal(t, "google", got)
	})

	t.Run("no connectors is a user error", func(t *testing.T) {
		client := &fakeConnectorClient{connectors: nil}
		_, err := resolveConnectorID(context.Background(), client, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no connectors")
	})

	t.Run("multiple connectors require an explicit choice", func(t *testing.T) {
		client := &fakeConnectorClient{connectors: []domain.Connector{
			{Provider: "google"},
			{Provider: "microsoft"},
		}}
		_, err := resolveConnectorID(context.Background(), client, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "google")
		assert.Contains(t, err.Error(), "microsoft")
	})

	t.Run("ListConnectors failure is surfaced", func(t *testing.T) {
		client := &fakeConnectorClient{err: errors.New("boom")}
		_, err := resolveConnectorID(context.Background(), client, "")
		require.Error(t, err)
	})

	t.Run("legacy ID during discovery failure surfaces the real error, not 'unknown provider'", func(t *testing.T) {
		// A UUID-like legacy ID can't be validated while discovery is down; the
		// user must see the discovery failure, not a misleading "unknown provider".
		client := &fakeConnectorClient{err: errors.New("boom")}
		_, err := resolveConnectorID(context.Background(), client, "conn-legacy-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
		assert.NotContains(t, err.Error(), "unknown connector provider")
	})
}

func TestCredentialCommands_Structure(t *testing.T) {
	// The connector is optional everywhere (auto-detected); the credential ID is
	// the required positional for show/update/delete.
	list := newCredentialListCmd()
	assert.Equal(t, "list [connector]", list.Use)

	show := newCredentialShowCmd()
	assert.Equal(t, "show <credential-id>", show.Use)
	assert.NotNil(t, show.Flags().Lookup("connector"), "show must expose --connector")

	update := newCredentialUpdateCmd()
	assert.Equal(t, "update <credential-id>", update.Use)
	assert.NotNil(t, update.Flags().Lookup("connector"))

	del := newCredentialDeleteCmd()
	assert.Equal(t, "delete <credential-id>", del.Use)
	assert.NotNil(t, del.Flags().Lookup("connector"))

	create := newCredentialCreateCmd()
	assert.NotNil(t, create.Flags().Lookup("connector"), "create must expose --connector")
	// --connector-id is kept as a deprecated alias for backward compatibility.
	legacy := create.Flags().Lookup("connector-id")
	require.NotNil(t, legacy)
	assert.NotEmpty(t, legacy.Deprecated, "--connector-id must be marked deprecated")

	// The v3 create request requires credential_data, which the command builds
	// from the client ID/secret, so both must be required flags.
	for _, name := range []string{"name", "type", "client-id", "client-secret"} {
		f := create.Flags().Lookup(name)
		require.NotNil(t, f, "create must expose --%s", name)
		assert.Contains(t, f.Annotations[cobra.BashCompOneRequiredFlag], "true", "--%s must be required", name)
	}
}

func TestCredentialCreateRejectsUnsupportedType(t *testing.T) {
	// The v3 API accepts connector/serviceaccount/adminconsent, but this
	// command only builds connector-shaped credential_data — any other --type
	// must fail fast with a clear message instead of an opaque provider 400.
	// "" covers --type "" — cobra's required-flag check is satisfied by an
	// explicitly set empty value, so the guard must reject it too.
	for _, unsupported := range []string{"oauth", "serviceaccount", "adminconsent", "bogus", ""} {
		t.Run(unsupported, func(t *testing.T) {
			cmd := newCredentialCreateCmd()
			require.NoError(t, cmd.Flags().Set("name", "test"))
			require.NoError(t, cmd.Flags().Set("type", unsupported))
			require.NoError(t, cmd.Flags().Set("client-id", "cid"))
			require.NoError(t, cmd.Flags().Set("client-secret", "secret"))

			err := cmd.RunE(cmd, nil)
			require.Error(t, err)
			// Validation must fire before any client/API work.
			assert.Contains(t, err.Error(), "invalid type")
		})
	}
}
