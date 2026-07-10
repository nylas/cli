package common

import (
	"github.com/nylas/cli/internal/domain"
)

// IsDeprecatedConnectorProvider reports whether the CLI should hide or reject a
// connector provider that is no longer supported. The provider-deprecation
// knowledge lives in the domain layer so adapters can share it.
func IsDeprecatedConnectorProvider(provider string) bool {
	return domain.IsDeprecatedConnectorProvider(provider)
}

// FilterVisibleConnectors removes deprecated connector providers from CLI-facing
// listings while leaving the backend API surface unchanged.
func FilterVisibleConnectors(connectors []domain.Connector) []domain.Connector {
	return domain.FilterVisibleConnectors(connectors)
}

// ValidateSupportedConnectorProvider rejects connector providers that are no
// longer supported by the CLI.
func ValidateSupportedConnectorProvider(provider string) error {
	if !domain.IsDeprecatedConnectorProvider(provider) {
		return nil
	}

	return NewUserError(
		"invalid provider: inbox",
		"The inbox connector is no longer supported",
	)
}
