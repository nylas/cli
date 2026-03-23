package domain

// DashboardUser represents an authenticated dashboard user.
type DashboardUser struct {
	PublicID     string `json:"publicId"`
	EmailAddress string `json:"emailAddress,omitempty"`
	FirstName    string `json:"firstName,omitempty"`
	LastName     string `json:"lastName,omitempty"`
}

// DashboardOrganization represents a Nylas organization.
type DashboardOrganization struct {
	PublicID string `json:"publicId"`
	Name     string `json:"name,omitempty"`
	Region   string `json:"region,omitempty"`
	Role     string `json:"role,omitempty"`
}

// DashboardRegisterResponse is the response from a successful registration.
type DashboardRegisterResponse struct {
	VerificationChannel string `json:"verificationChannel"`
	ExpiresAt           string `json:"expiresAt"`
}

// DashboardAuthResponse is the response from a successful login or verification.
type DashboardAuthResponse struct {
	UserToken     string                  `json:"userToken"`
	OrgToken      string                  `json:"orgToken"`
	User          DashboardUser           `json:"user"`
	Organizations []DashboardOrganization `json:"organizations"`
}

// DashboardMFARequired is returned when MFA is needed after login.
type DashboardMFARequired struct {
	User          DashboardUser           `json:"user"`
	Organizations []DashboardOrganization `json:"organizations"`
	TOTPFactor    *DashboardTOTPFactor    `json:"totpFactor"`
}

// DashboardTOTPFactor contains the TOTP factor details for MFA.
type DashboardTOTPFactor struct {
	FactorSID string `json:"factorSid"`
	Binding   string `json:"binding,omitempty"`
}

// DashboardRefreshResponse is the response from a session refresh.
type DashboardRefreshResponse struct {
	UserToken string `json:"userToken"`
	OrgToken  string `json:"orgToken,omitempty"`
}

// DashboardSSOStartResponse is the response from starting an SSO device flow.
type DashboardSSOStartResponse struct {
	FlowID                  string `json:"flowId"`
	VerificationURI         string `json:"verificationUri"`
	VerificationURIComplete string `json:"verificationUriComplete,omitempty"`
	UserCode                string `json:"userCode"`
	ExpiresIn               int    `json:"expiresIn"`
	Interval                int    `json:"interval"`
}

// DashboardSSOPollResponse represents the poll result for an SSO device flow.
type DashboardSSOPollResponse struct {
	Status     string `json:"status"`
	RetryAfter int    `json:"retryAfter,omitempty"`

	// Populated when Status == "complete"
	Auth *DashboardAuthResponse `json:"-"`

	// Populated when Status == "mfa_required"
	MFA *DashboardMFARequired `json:"-"`
}

// SSO poll status constants.
const (
	SSOStatusPending      = "authorization_pending"
	SSOStatusAccessDenied = "access_denied"
	SSOStatusExpired      = "expired_token"
	SSOStatusComplete     = "complete"
	SSOStatusMFARequired  = "mfa_required"
)

// SSO login type constants matching the server schema.
const (
	SSOLoginTypeGoogle    = "google_SSO"
	SSOLoginTypeMicrosoft = "microsoft_SSO"
	SSOLoginTypeGitHub    = "github_SSO"
)

// GatewayApplication is an application as returned by the dashboard API gateway.
type GatewayApplication struct {
	ApplicationID  string                   `json:"applicationId"`
	OrganizationID string                   `json:"organizationId"`
	Region         string                   `json:"region"`
	Environment    string                   `json:"environment"`
	Branding       *GatewayApplicationBrand `json:"branding,omitempty"`
}

// GatewayCreatedApplication includes the client secret shown once on creation.
type GatewayCreatedApplication struct {
	ApplicationID  string                   `json:"applicationId"`
	ClientSecret   string                   `json:"clientSecret"`
	OrganizationID string                   `json:"organizationId"`
	Region         string                   `json:"region"`
	Environment    string                   `json:"environment"`
	Branding       *GatewayApplicationBrand `json:"branding,omitempty"`
}

// GatewayApplicationBrand holds application branding info.
type GatewayApplicationBrand struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// DashboardConfig holds dashboard authentication settings.
type DashboardConfig struct {
	AccountBaseURL string `yaml:"account_base_url,omitempty"`
}

// DefaultDashboardAccountBaseURL is the global dashboard-account endpoint.
const DefaultDashboardAccountBaseURL = "https://dashboard-account.eu.nylas.com"

// Dashboard API gateway URLs by region.
const (
	GatewayBaseURLUS = "https://dashboard-api-gateway.us.nylas.com/graphql"
	GatewayBaseURLEU = "https://dashboard-api-gateway.eu.nylas.com/graphql"
)
