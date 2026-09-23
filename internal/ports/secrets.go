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

	// OAuth authorization server keys. KeyOAuthIssuer records which server
	// issued the stored tokens. The client id is not stored: the CLI is a
	// static public client (domain.DefaultOAuthClientID).
	KeyOAuthIssuer       = "oauth_issuer"
	KeyOAuthResource     = "oauth_resource"
	KeyOAuthAccessToken  = "oauth_access_token"
	KeyOAuthRefreshToken = "oauth_refresh_token"
	KeyOAuthIDToken      = "oauth_id_token"
	KeyOAuthExpiresAt    = "oauth_expires_at"
	KeyOAuthScope        = "oauth_scope"
)
