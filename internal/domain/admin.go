package domain

import (
	"errors"
	"strings"
)

// deprecatedConnectorProviderInbox is the provider the CLI and API surface no
// longer support.
const deprecatedConnectorProviderInbox = "inbox"

// IsDeprecatedConnectorProvider reports whether a connector provider is no
// longer supported and should be hidden or rejected.
func IsDeprecatedConnectorProvider(provider string) bool {
	return strings.EqualFold(strings.TrimSpace(provider), deprecatedConnectorProviderInbox)
}

// supportedConnectorProviders is the Nylas v3 provider allow-list (the {provider}
// path-segment enum). Used to reject a typo'd/unknown explicit connector value
// before it is sent as a bogus path segment.
var supportedConnectorProviders = map[string]bool{
	"google": true, "microsoft": true, "imap": true, "icloud": true,
	"yahoo": true, "ews": true, "virtual-calendar": true, "zoom": true, "nylas": true,
}

// IsSupportedConnectorProvider reports whether a value is a recognized Nylas v3
// connector provider name.
func IsSupportedConnectorProvider(provider string) bool {
	return supportedConnectorProviders[strings.ToLower(strings.TrimSpace(provider))]
}

// FilterVisibleConnectors removes deprecated connector providers from a list
// while leaving the backend API surface unchanged.
func FilterVisibleConnectors(connectors []Connector) []Connector {
	if len(connectors) == 0 {
		return connectors
	}

	filtered := make([]Connector, 0, len(connectors))
	for _, connector := range connectors {
		if IsDeprecatedConnectorProvider(connector.Provider) {
			continue
		}
		filtered = append(filtered, connector)
	}

	return filtered
}

// VisibleConnectorProviders returns the providers of all non-deprecated
// connectors, in order. Auto-detecting a sole connector is len()==1; a longer
// list is the "which one?" disambiguation set. Shared by the CLI and RPC
// credential surfaces so they resolve the same connector.
func VisibleConnectorProviders(connectors []Connector) []string {
	visible := FilterVisibleConnectors(connectors)
	providers := make([]string, len(visible))
	for i, c := range visible {
		providers[i] = c.Provider
	}
	return providers
}

// ConnectorProviderForIdentifier maps either a provider name or a legacy
// connector ID to the provider required by provider-scoped connector APIs.
func ConnectorProviderForIdentifier(connectors []Connector, identifier string) (string, bool) {
	for _, connector := range connectors {
		if strings.EqualFold(identifier, connector.Provider) || identifier == connector.ID {
			return connector.Provider, true
		}
	}
	return "", false
}

// ErrNoConnectors is returned by ResolveSoleConnectorProvider when there are no
// visible connectors to auto-detect.
var ErrNoConnectors = errors.New("no connectors found")

// MultipleConnectorsError is returned by ResolveSoleConnectorProvider when
// auto-detection is ambiguous because more than one visible connector exists.
// Providers lists the candidates so callers can name them.
type MultipleConnectorsError struct{ Providers []string }

func (e *MultipleConnectorsError) Error() string {
	return "multiple connectors found: " + strings.Join(e.Providers, ", ")
}

// ResolveSoleConnectorProvider returns the provider of the single visible
// connector, or ErrNoConnectors / *MultipleConnectorsError. It is the single
// source of truth for the auto-detect decision shared by the CLI and RPC
// credential commands; each caller maps the error to its own surface's format.
func ResolveSoleConnectorProvider(connectors []Connector) (string, error) {
	providers := VisibleConnectorProviders(connectors)
	switch len(providers) {
	case 1:
		return providers[0], nil
	case 0:
		return "", ErrNoConnectors
	default:
		return "", &MultipleConnectorsError{Providers: providers}
	}
}

// ErrDeprecatedConnector is returned by ResolveConnectorProvider when the caller
// explicitly names a provider that is no longer supported.
var ErrDeprecatedConnector = errors.New("connector provider is no longer supported")

// ErrUnknownConnector is returned by ResolveConnectorProvider when an explicit
// value matches no known connector and is not a recognized provider name.
var ErrUnknownConnector = errors.New("unknown connector provider")

// ResolveConnectorProvider is the single decision shared by the CLI and RPC
// credential surfaces: it maps an explicit provider/legacy connector ID to a
// provider, or auto-detects the sole visible connector when explicit is empty.
// Callers pass whatever ListConnectors returned (nil/empty is fine — an explicit
// value still resolves by passthrough) and map the returned error to their own
// surface's format. Keeping the policy here stops the two surfaces from silently
// diverging.
func ResolveConnectorProvider(connectors []Connector, explicit string) (string, error) {
	if explicit != "" {
		if IsDeprecatedConnectorProvider(explicit) {
			return "", ErrDeprecatedConnector
		}
		if provider, ok := ConnectorProviderForIdentifier(connectors, explicit); ok {
			// A legacy connector ID can map to a deprecated provider even though
			// the ID itself is not the literal "inbox" string, so re-check the
			// resolved provider to close that bypass.
			if IsDeprecatedConnectorProvider(provider) {
				return "", ErrDeprecatedConnector
			}
			return provider, nil
		}
		// Not a known connector (by name or ID). Accept it only if it is a
		// recognized provider name — this keeps `--connector google` working when
		// no connector exists yet or discovery is unavailable, while rejecting a
		// typo instead of sending it as a bogus /connectors/{value}/creds segment.
		// Return the normalized form so casing/whitespace can't leak into the path.
		if IsSupportedConnectorProvider(explicit) {
			return strings.ToLower(strings.TrimSpace(explicit)), nil
		}
		return "", ErrUnknownConnector
	}
	return ResolveSoleConnectorProvider(connectors)
}

// Application represents a Nylas application
type Application struct {
	ID               string            `json:"id,omitempty"`
	ApplicationID    string            `json:"application_id,omitempty"`
	OrganizationID   string            `json:"organization_id,omitempty"`
	Region           string            `json:"region,omitempty"`
	Environment      string            `json:"environment,omitempty"`
	BrandingSettings *BrandingSettings `json:"branding,omitempty"`
	CallbackURIs     []CallbackURI     `json:"callback_uris,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        *UnixTime         `json:"created_at,omitempty"`
	UpdatedAt        *UnixTime         `json:"updated_at,omitempty"`
}

// CallbackURI represents a callback URI configuration
type CallbackURI struct {
	ID       string `json:"id,omitempty"`
	Platform string `json:"platform,omitempty"`
	URL      string `json:"url,omitempty"`
}

// CreateCallbackURIRequest represents a request to create a callback URI
type CreateCallbackURIRequest struct {
	URL      string `json:"url"`
	Platform string `json:"platform"`
}

// UpdateCallbackURIRequest represents a request to update a callback URI
type UpdateCallbackURIRequest struct {
	URL      *string `json:"url,omitempty"`
	Platform *string `json:"platform,omitempty"`
}

// BrandingSettings represents application branding configuration
type BrandingSettings struct {
	Name              string `json:"name,omitempty"`
	IconURL           string `json:"icon_url,omitempty"`
	WebsiteURL        string `json:"website_url,omitempty"`
	Description       string `json:"description,omitempty"`
	PrivacyPolicyURL  string `json:"privacy_policy_url,omitempty"`
	TermsOfServiceURL string `json:"terms_of_service_url,omitempty"`
}

// CreateApplicationRequest represents a request to create an application
type CreateApplicationRequest struct {
	Name             string            `json:"name"`
	Region           string            `json:"region,omitempty"` // "us", "eu", etc.
	BrandingSettings *BrandingSettings `json:"branding,omitempty"`
	CallbackURIs     []string          `json:"callback_uris,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// UpdateApplicationRequest represents a request to update an application
type UpdateApplicationRequest struct {
	Name             *string           `json:"name,omitempty"`
	BrandingSettings *BrandingSettings `json:"branding,omitempty"`
	CallbackURIs     []string          `json:"callback_uris,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// Connector represents a provider connector (Google, Microsoft, iCloud, Yahoo, IMAP)
type Connector struct {
	ID                 string             `json:"id,omitempty"`
	Name               string             `json:"name"`
	Provider           string             `json:"provider"` // "google", "microsoft", "icloud", "yahoo", "imap"
	DefaultWorkspaceID string             `json:"default_workspace_id,omitempty"`
	Settings           *ConnectorSettings `json:"settings,omitempty"`
	Scopes             []string           `json:"scopes,omitempty"`
	CreatedAt          *UnixTime          `json:"created_at,omitempty"`
	UpdatedAt          *UnixTime          `json:"updated_at,omitempty"`
}

// ConnectorSettings holds provider-specific configuration
type ConnectorSettings struct {
	// OAuth providers (Google, Microsoft, etc.)
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Tenant       string `json:"tenant,omitempty"` // For Microsoft

	// IMAP-specific settings
	IMAPHost     string `json:"imap_host,omitempty"`
	IMAPPort     int    `json:"imap_port,omitempty"`
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	IMAPSecurity string `json:"imap_security,omitempty"` // "ssl", "starttls", "none"
	SMTPSecurity string `json:"smtp_security,omitempty"` // "ssl", "starttls", "none"
}

// CreateConnectorRequest represents a request to create a connector
type CreateConnectorRequest struct {
	Name     string             `json:"name"`
	Provider string             `json:"provider"` // "google", "microsoft", "icloud", "yahoo", "imap"
	Settings *ConnectorSettings `json:"settings,omitempty"`
	Scopes   []string           `json:"scopes,omitempty"`
}

// UpdateConnectorRequest represents a request to update a connector
type UpdateConnectorRequest struct {
	Name     *string            `json:"name,omitempty"`
	Settings *ConnectorSettings `json:"settings,omitempty"`
	Scopes   []string           `json:"scopes,omitempty"`
}

// ConnectorCredential is the Nylas v3 credential response object. Per the spec
// (CredentialObject) the API returns only these fields; credential_type and
// credential_data are request-only inputs and are never echoed back.
type ConnectorCredential struct {
	ID        string    `json:"id,omitempty"`
	Name      string    `json:"name"`
	CreatedAt *UnixTime `json:"created_at,omitempty"`
	UpdatedAt *UnixTime `json:"updated_at,omitempty"`
}

// CreateCredentialRequest represents a request to create a credential
type CreateCredentialRequest struct {
	Name           string         `json:"name"`
	CredentialType string         `json:"credential_type"` // "oauth", "service_account", "connector"
	CredentialData map[string]any `json:"credential_data,omitempty"`
}

// UpdateCredentialRequest represents a request to update a credential
type UpdateCredentialRequest struct {
	Name           *string        `json:"name,omitempty"`
	CredentialData map[string]any `json:"credential_data,omitempty"`
}

// GrantsQueryParams represents query parameters for listing grants
type GrantsQueryParams struct {
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
	ConnectorID string `json:"connector_id,omitempty"`
	Status      string `json:"status,omitempty"` // "valid", "invalid"
}

// GrantStats represents grant statistics
type GrantStats struct {
	Total      int            `json:"total"`
	ByProvider map[string]int `json:"by_provider"`
	ByStatus   map[string]int `json:"by_status"`
	Valid      int            `json:"valid"`
	Invalid    int            `json:"invalid"`
	Revoked    int            `json:"revoked"`
}
