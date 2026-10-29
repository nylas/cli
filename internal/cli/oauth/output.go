package oauth

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// The reports below are what --json and --format yaml print. They never
// carry a token: only facts about the session. Each returns an empty
// QuietField so --quiet prints nothing and the exit code is the answer.

// statusReport is `nylas oauth status` as structured output.
type statusReport struct {
	LoggedIn   bool            `json:"logged_in" yaml:"logged_in"`
	Expired    bool            `json:"expired" yaml:"expired"`
	Access     string          `json:"access,omitempty" yaml:"access,omitempty"`
	Server     string          `json:"server,omitempty" yaml:"server,omitempty"`
	Issuer     string          `json:"issuer" yaml:"issuer"`
	ClientID   string          `json:"client_id" yaml:"client_id"`
	Resource   string          `json:"resource,omitempty" yaml:"resource,omitempty"`
	Scopes     string          `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	ExpiresAt  *time.Time      `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	HasRefresh bool            `json:"has_refresh_token" yaml:"has_refresh_token"`
	Claims     *claimsReport   `json:"token_claims,omitempty" yaml:"token_claims,omitempty"`
	ClaimsErr  string          `json:"token_claims_error,omitempty" yaml:"token_claims_error,omitempty"`
	UserInfo   *userInfoReport `json:"userinfo,omitempty" yaml:"userinfo,omitempty"`
	// MCPCredentials are the MCP-only credentials, listed after the default
	// session; never set on a credential's own report.
	MCPCredentials []mcpCredentialReport `json:"mcp_credentials,omitempty" yaml:"mcp_credentials,omitempty"`
}

func (statusReport) QuietField() string { return "" }

// mcpCredentialReport is one MCP-only credential: its session, or why it
// could not be read.
type mcpCredentialReport struct {
	ID      string        `json:"id" yaml:"id"`
	Session *statusReport `json:"session,omitempty" yaml:"session,omitempty"`
	Error   string        `json:"error,omitempty" yaml:"error,omitempty"`
}

// claimsReport is what the access token says about itself, decoded locally
// and NOT verified.
type claimsReport struct {
	Verified  bool              `json:"verified" yaml:"verified"`
	Audience  []string          `json:"audience,omitempty" yaml:"audience,omitempty"`
	Org       string            `json:"org,omitempty" yaml:"org,omitempty"`
	Principal *principalReport  `json:"principal,omitempty" yaml:"principal,omitempty"`
	Scopes    string            `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	Grants    []claimGrantEntry `json:"grants,omitempty" yaml:"grants,omitempty"`
}

// principalReport is the IAM principal a token is narrowed by.
type principalReport struct {
	ID     string `json:"id" yaml:"id"`
	Region string `json:"region" yaml:"region"`
}

type claimGrantEntry struct {
	ID            string `json:"id" yaml:"id"`
	ApplicationID string `json:"application_id" yaml:"application_id"`
}

type userInfoReport struct {
	Subject       string `json:"subject" yaml:"subject"`
	Email         string `json:"email,omitempty" yaml:"email,omitempty"`
	EmailVerified bool   `json:"email_verified" yaml:"email_verified"`
	Org           string `json:"org,omitempty" yaml:"org,omitempty"`
}

// loginReport is `nylas oauth login` as structured output.
type loginReport struct {
	LoggedIn      bool       `json:"logged_in" yaml:"logged_in"`
	Issuer        string     `json:"issuer" yaml:"issuer"`
	ClientID      string     `json:"client_id" yaml:"client_id"`
	Resource      string     `json:"resource,omitempty" yaml:"resource,omitempty"`
	Scopes        string     `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	HasRefresh    bool       `json:"has_refresh_token" yaml:"has_refresh_token"`
	DroppedScopes []string   `json:"dropped_scopes,omitempty" yaml:"dropped_scopes,omitempty"`
	// MCPCredential is set for `--mcp-only`, which never signs the dashboard
	// commands in.
	MCPCredential *mcpCredentialInfo `json:"mcp_credential,omitempty" yaml:"mcp_credential,omitempty"`
	Dashboard     dashboardOutcome   `json:"dashboard" yaml:"dashboard"`
}

// mcpCredentialInfo is a stored MCP-only credential, without its token.
type mcpCredentialInfo struct {
	ID        string           `json:"id" yaml:"id"`
	Audience  string           `json:"audience" yaml:"audience"`
	Org       string           `json:"org,omitempty" yaml:"org,omitempty"`
	Principal *principalReport `json:"principal,omitempty" yaml:"principal,omitempty"`
}

func (loginReport) QuietField() string { return "" }

// dashboardOutcome is what the login did to the dashboard commands.
type dashboardOutcome struct {
	SignedIn bool `json:"signed_in" yaml:"signed_in"`
	// KeptDashboardLogin: a `nylas dashboard login` session was left in place.
	KeptDashboardLogin bool   `json:"kept_dashboard_login,omitempty" yaml:"kept_dashboard_login,omitempty"`
	OrgPublicID        string `json:"org_public_id,omitempty" yaml:"org_public_id,omitempty"`
	Error              string `json:"error,omitempty" yaml:"error,omitempty"`
}

// logoutReport is `nylas oauth logout` as structured output.
type logoutReport struct {
	LoggedOut      bool   `json:"logged_out" yaml:"logged_out"`
	DashboardError string `json:"dashboard_error,omitempty" yaml:"dashboard_error,omitempty"`
}

func (logoutReport) QuietField() string { return "" }

func newStatusReport(session *oauthlogin.Session, now time.Time) statusReport {
	report := statusReport{
		LoggedIn:   true,
		Expired:    session.Tokens.IsExpired(now),
		Access:     accessMode(session),
		Server:     session.ServerURL,
		Issuer:     session.Issuer,
		ClientID:   session.ClientID,
		Resource:   session.Resource,
		Scopes:     session.Tokens.Scope,
		ExpiresAt:  timePtr(session.Tokens.ExpiresAt),
		HasRefresh: session.Tokens.RefreshToken != "",
	}
	claims, err := domain.DecodeOAuthAccessToken(session.Tokens.AccessToken)
	if err != nil {
		report.ClaimsErr = err.Error()
		return report
	}
	report.Claims = &claimsReport{
		Audience:  claims.Audience,
		Org:       claims.Org,
		Scopes:    claims.Scope,
		ExpiresAt: timePtr(claims.ExpiresAt),
	}
	if claims.Principal != nil {
		report.Claims.Principal = &principalReport{ID: claims.Principal.ID, Region: claims.Principal.Region}
	}
	for _, grant := range claims.Grants {
		report.Claims.Grants = append(report.Claims.Grants,
			claimGrantEntry{ID: grant.ID, ApplicationID: grant.ApplicationID})
	}
	return report
}

// writeStatusReport is the structured form of `nylas oauth status`. Like the
// text form, a missing default session is an error only when no MCP-only
// credential is stored either, and --verify asks about the session shown
// first: the one --credential names, or else the default one.
func writeStatusReport(cmd *cobra.Command, svc loginService, credential string, remote bool) error {
	if credential != "" {
		bound, err := svc.ForMCPCredential(credential)
		if err != nil {
			return wrapOAuthError(err)
		}
		report, err := sessionReport(bound, remote)
		if err != nil {
			return wrapOAuthError(err)
		}
		return common.GetOutputWriter(cmd).Write(report)
	}

	records, err := svc.ListMCPCredentials()
	if err != nil {
		return wrapOAuthError(err)
	}
	report, err := sessionReport(svc, remote)
	if err != nil {
		if !errors.Is(err, domain.ErrOAuthNotLoggedIn) || len(records) == 0 {
			return wrapOAuthError(err)
		}
		report = statusReport{}
	}
	for _, record := range records {
		entry := mcpCredentialReport{ID: record.ID}
		bound, err := svc.ForMCPCredential(record.ID)
		if err == nil {
			var session statusReport
			if session, err = sessionReport(bound, false); err == nil {
				entry.Session = &session
			}
		}
		if err != nil {
			entry.Error = err.Error()
		}
		report.MCPCredentials = append(report.MCPCredentials, entry)
	}
	return common.GetOutputWriter(cmd).Write(report)
}

// sessionReport reports one stored session, and with remote proves its token
// against the live server rather than trusting what is on disk.
func sessionReport(svc loginService, remote bool) (statusReport, error) {
	session, err := svc.Status()
	if err != nil {
		return statusReport{}, err
	}
	report := newStatusReport(session, time.Now())
	if !remote {
		return report, nil
	}
	ctx, cancel := common.CreateContext()
	defer cancel()
	info, err := svc.UserInfo(ctx)
	if err != nil {
		return statusReport{}, err
	}
	report.UserInfo = &userInfoReport{
		Subject:       info.Subject,
		Email:         info.Email,
		EmailVerified: info.EmailVerified,
		Org:           info.Org,
	}
	return report, nil
}

func newLoginReport(result *oauthlogin.LoginResult) loginReport {
	report := loginReport{
		LoggedIn:      true,
		Issuer:        result.Issuer,
		ClientID:      result.ClientID,
		Resource:      result.Resource,
		Scopes:        result.Scope,
		ExpiresAt:     timePtr(result.ExpiresAt),
		HasRefresh:    result.HasRefresh,
		DroppedScopes: result.DroppedScopes,
	}
	if record := result.Credential; record != nil {
		report.MCPCredential = &mcpCredentialInfo{ID: record.ID, Audience: record.Audience, Org: record.Org}
		if record.Principal != nil {
			report.MCPCredential.Principal = &principalReport{ID: record.Principal.ID, Region: record.Principal.Region}
		}
	}
	return report
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// structuredOutput reports whether --json, --format json|yaml|quiet or
// --quiet was given, in which case the commands write a report instead of
// the human-readable text.
func structuredOutput(cmd *cobra.Command) bool {
	return common.IsStructuredOutput(cmd)
}

// quietOutput reports whether informational text must be suppressed.
func quietOutput(cmd *cobra.Command) bool {
	quiet, _ := cmd.Flags().GetBool("quiet")
	format, _ := cmd.Flags().GetString("format")
	return quiet || format == "quiet" || common.IsQuiet()
}
