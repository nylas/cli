package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveConnectorProvider pins the resolution policy shared by the CLI and
// RPC credential surfaces: explicit values win (legacy IDs map to providers,
// deprecated ones are rejected, unknown ones pass through), and an empty value
// auto-detects the sole visible connector.
func TestResolveConnectorProvider(t *testing.T) {
	connectors := []Connector{
		{ID: "conn-1", Provider: "google"},
		{Provider: "inbox"}, // deprecated, hidden from auto-detect
	}

	t.Run("explicit provider is returned", func(t *testing.T) {
		got, err := ResolveConnectorProvider(connectors, "google")
		require.NoError(t, err)
		assert.Equal(t, "google", got)
	})

	t.Run("legacy connector ID maps to provider", func(t *testing.T) {
		got, err := ResolveConnectorProvider(connectors, "conn-1")
		require.NoError(t, err)
		assert.Equal(t, "google", got)
	})

	t.Run("explicit deprecated provider is rejected", func(t *testing.T) {
		_, err := ResolveConnectorProvider(connectors, "inbox")
		require.ErrorIs(t, err, ErrDeprecatedConnector)
	})

	t.Run("legacy ID mapping to a deprecated provider is rejected", func(t *testing.T) {
		// The bypass: the identifier is an ID (not the literal "inbox"), so it
		// slips the by-name check but resolves to the deprecated inbox provider.
		withID := []Connector{{ID: "inbox-conn-9", Provider: "inbox"}}
		_, err := ResolveConnectorProvider(withID, "inbox-conn-9")
		require.ErrorIs(t, err, ErrDeprecatedConnector)
	})

	t.Run("supported provider name passes through even with no connectors", func(t *testing.T) {
		got, err := ResolveConnectorProvider(nil, "microsoft")
		require.NoError(t, err)
		assert.Equal(t, "microsoft", got)
	})

	t.Run("unknown/typo'd explicit value is rejected, not sent as a path segment", func(t *testing.T) {
		_, err := ResolveConnectorProvider(connectors, "googel")
		require.ErrorIs(t, err, ErrUnknownConnector)
	})

	t.Run("supported provider is normalized (casing/whitespace) so it can't leak into the path", func(t *testing.T) {
		for _, in := range []string{"Google", " google ", "GOOGLE"} {
			got, err := ResolveConnectorProvider(nil, in)
			require.NoError(t, err, in)
			assert.Equal(t, "google", got, in)
		}
	})

	t.Run("empty explicit auto-detects the sole visible connector", func(t *testing.T) {
		got, err := ResolveConnectorProvider(connectors, "")
		require.NoError(t, err)
		assert.Equal(t, "google", got)
	})

	t.Run("empty explicit with no connectors returns ErrNoConnectors", func(t *testing.T) {
		_, err := ResolveConnectorProvider(nil, "")
		require.ErrorIs(t, err, ErrNoConnectors)
	})

	t.Run("empty explicit with multiple visible connectors is ambiguous", func(t *testing.T) {
		multi := []Connector{{Provider: "google"}, {Provider: "microsoft"}}
		_, err := ResolveConnectorProvider(multi, "")
		var mErr *MultipleConnectorsError
		require.True(t, errors.As(err, &mErr))
		assert.ElementsMatch(t, []string{"google", "microsoft"}, mErr.Providers)
	})
}
