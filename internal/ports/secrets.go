// Package ports defines the interfaces for external dependencies.
package ports

// SecretStore defines the interface for storing secrets securely.
type SecretStore interface {
	// Set stores a secret value for the given key.
	Set(key, value string) error

	// Get retrieves a secret value for the given key.
	Get(key string) (string, error)

	// Delete removes a secret for the given key.
	Delete(key string) error

	// IsAvailable checks if the secret store is available.
	IsAvailable() bool

	// Name returns the name of the secret store backend.
	Name() string
}

// Secret key constants.
const (
	KeyClientID     = "client_id"
	KeyClientSecret = "client_secret"
	KeyAPIKey       = "api_key"
	KeyOrgID        = "org_id"

	// Dashboard auth keys
	KeyDashboardUserToken    = "dashboard_user_token"
	KeyDashboardOrgToken     = "dashboard_org_token"
	KeyDashboardUserPublicID = "dashboard_user_public_id"
	KeyDashboardOrgPublicID  = "dashboard_org_public_id"
	KeyDashboardDPoPKey      = "dashboard_dpop_key"
	KeyDashboardAppID        = "dashboard_app_id"
	KeyDashboardAppRegion    = "dashboard_app_region"
	// Set when the dashboard session was exchanged from an OAuth access
	// token. Such a session cannot be refreshed, only exchanged again before
	// KeyDashboardSessionExpiresAt (RFC 3339).
	KeyDashboardSessionOrigin    = "dashboard_session_origin"
	KeyDashboardSessionExpiresAt = "dashboard_session_expires_at"
	// KeyDashboardSessionServer records the servers a dashboard session was
	// issued for (the account URL and both gateways), so its tokens are never
	// sent to a server that did not issue them.
	KeyDashboardSessionServer = "dashboard_session_server"

	// OAuth authorization server keys. KeyOAuthIssuer records which server
	// issued the stored tokens. The client id is not stored: the CLI is a
	// static public client (domain.DefaultOAuthClientID).
	KeyOAuthIssuer = "oauth_issuer"
	// KeyOAuthServerURL is the authorization server URL the session was
	// obtained through; its tokens are only ever sent back there.
	KeyOAuthServerURL    = "oauth_server_url"
	KeyOAuthResource     = "oauth_resource"
	KeyOAuthAccessToken  = "oauth_access_token"
	KeyOAuthRefreshToken = "oauth_refresh_token"
	KeyOAuthIDToken      = "oauth_id_token"
	KeyOAuthExpiresAt    = "oauth_expires_at"
	KeyOAuthScope        = "oauth_scope"
)
