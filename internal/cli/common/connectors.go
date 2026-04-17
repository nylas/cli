package common

import (
	"strings"

	"github.com/nylas/cli/internal/domain"
)

const deprecatedConnectorProviderInbox = "inbox"

// IsDeprecatedConnectorProvider reports whether the CLI should hide or reject a
// connector provider that is no longer supported.
func IsDeprecatedConnectorProvider(provider string) bool {
	return strings.EqualFold(strings.TrimSpace(provider), deprecatedConnectorProviderInbox)
}

// FilterVisibleConnectors removes deprecated connector providers from CLI-facing
// listings while leaving the backend API surface unchanged.
func FilterVisibleConnectors(connectors []domain.Connector) []domain.Connector {
	if len(connectors) == 0 {
		return connectors
	}

	filtered := make([]domain.Connector, 0, len(connectors))
	for _, connector := range connectors {
		if IsDeprecatedConnectorProvider(connector.Provider) {
			continue
		}
		filtered = append(filtered, connector)
	}

	return filtered
}

// ValidateSupportedConnectorProvider rejects connector providers that are no
// longer supported by the CLI.
func ValidateSupportedConnectorProvider(provider string) error {
	if !IsDeprecatedConnectorProvider(provider) {
		return nil
	}

	return NewUserError(
		"invalid provider: inbox",
		"The inbox connector is no longer supported",
	)
}
