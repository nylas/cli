package domain

import "slices"

// Provider represents an email provider type.
type Provider string

const (
	ProviderGoogle    Provider = "google"
	ProviderMicrosoft Provider = "microsoft"
	ProviderIMAP      Provider = "imap"
	ProviderVirtual   Provider = "virtual"
	ProviderInbox     Provider = "inbox" // Nylas Native Auth
	ProviderNylas     Provider = "nylas"
)

// SupportedAirProviders lists providers supported by the Air web UI.
var SupportedAirProviders = []Provider{ProviderGoogle, ProviderMicrosoft, ProviderNylas}

// DisplayName returns the user-friendly name for the provider.
func (p Provider) DisplayName() string {
	switch p {
	case ProviderGoogle:
		return "Google"
	case ProviderMicrosoft:
		return "Microsoft"
	case ProviderIMAP:
		return "IMAP"
	case ProviderVirtual:
		return "Virtual"
	case ProviderInbox:
		return "Inbox"
	case ProviderNylas:
		return "Nylas"
	default:
		return string(p)
	}
}

// IsValid checks if the provider is a known type.
func (p Provider) IsValid() bool {
	switch p {
	case ProviderGoogle, ProviderMicrosoft, ProviderIMAP, ProviderVirtual, ProviderInbox, ProviderNylas:
		return true
	default:
		return false
	}
}

// IsSupportedByAir checks if the provider is supported by the Air web UI.
func (p Provider) IsSupportedByAir() bool {
	return slices.Contains(SupportedAirProviders, p)
}

// ParseProvider converts a string to a Provider.
func ParseProvider(s string) (Provider, error) {
	p := Provider(s)
	if !p.IsValid() {
		return "", ErrInvalidProvider
	}
	return p, nil
}
