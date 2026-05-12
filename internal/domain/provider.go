package domain

import "slices"

// Provider represents an email provider type.
type Provider string

const (
	ProviderGoogle    Provider = "google"
	ProviderMicrosoft Provider = "microsoft"
	ProviderEWS       Provider = "ews"
	ProviderIMAP      Provider = "imap"
	ProviderICloud    Provider = "icloud"
	ProviderYahoo     Provider = "yahoo"
	ProviderVirtual   Provider = "virtual"
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
	case ProviderEWS:
		return "Exchange (EWS)"
	case ProviderIMAP:
		return "IMAP"
	case ProviderICloud:
		return "iCloud"
	case ProviderYahoo:
		return "Yahoo"
	case ProviderVirtual:
		return "Virtual"
	case ProviderNylas:
		return "Nylas"
	default:
		return string(p)
	}
}

// IsValid checks if the provider is a known type.
func (p Provider) IsValid() bool {
	switch p {
	case ProviderGoogle, ProviderMicrosoft, ProviderEWS, ProviderIMAP, ProviderICloud, ProviderYahoo, ProviderVirtual, ProviderNylas:
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
