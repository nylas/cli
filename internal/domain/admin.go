package domain

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
	ID        string             `json:"id,omitempty"`
	Name      string             `json:"name"`
	Provider  string             `json:"provider"` // "google", "microsoft", "icloud", "yahoo", "imap"
	Settings  *ConnectorSettings `json:"settings,omitempty"`
	Scopes    []string           `json:"scopes,omitempty"`
	CreatedAt *UnixTime          `json:"created_at,omitempty"`
	UpdatedAt *UnixTime          `json:"updated_at,omitempty"`
}

// ConnectorSettings holds provider-specific configuration
type ConnectorSettings struct {
	// OAuth providers (Google, Microsoft, etc.)
	ClientID string `json:"client_id,omitempty"`
	Tenant   string `json:"tenant,omitempty"` // For Microsoft

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

// ConnectorCredential represents authentication credentials for a connector
type ConnectorCredential struct {
	ID             string         `json:"id,omitempty"`
	Name           string         `json:"name"`
	ConnectorID    string         `json:"connector_id,omitempty"`
	CredentialType string         `json:"credential_type"` // "oauth", "service_account", "connector"
	CredentialData map[string]any `json:"credential_data,omitempty"`
	CreatedAt      *UnixTime      `json:"created_at,omitempty"`
	UpdatedAt      *UnixTime      `json:"updated_at,omitempty"`
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
