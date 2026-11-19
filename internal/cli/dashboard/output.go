package dashboard

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

// The reports below are what --json and --format yaml print for the
// dashboard commands, so an agent or a script can read the result instead of
// scraping the human text. None of them ever carries a secret: a new API key
// or client secret is reported by HOW it was delivered, never by value.

// statusReport is `nylas dashboard status` as structured output. A missing
// session is a report with logged_in false, not an error, matching the text
// form's exit code.
type statusReport struct {
	LoggedIn          bool             `json:"logged_in" yaml:"logged_in"`
	UserID            string           `json:"user_id,omitempty" yaml:"user_id,omitempty"`
	OrganizationID    string           `json:"organization_id,omitempty" yaml:"organization_id,omitempty"`
	OrganizationName  string           `json:"organization_name,omitempty" yaml:"organization_name,omitempty"`
	OrganizationCount int              `json:"organization_count,omitempty" yaml:"organization_count,omitempty"`
	HasOrgToken       bool             `json:"has_org_token" yaml:"has_org_token"`
	ActiveApp         *activeAppReport `json:"active_app,omitempty" yaml:"active_app,omitempty"`
	DPoPThumbprint    string           `json:"dpop_thumbprint,omitempty" yaml:"dpop_thumbprint,omitempty"`
}

// QuietField is empty: like `nylas oauth status`, --quiet prints nothing.
func (statusReport) QuietField() string { return "" }

// activeAppReport is the application the apikeys commands default to.
type activeAppReport struct {
	ApplicationID string `json:"application_id" yaml:"application_id"`
	Region        string `json:"region" yaml:"region"`
}

func (r activeAppReport) QuietField() string { return r.ApplicationID }

// appCreatedReport is `nylas dashboard apps create` as structured output.
type appCreatedReport struct {
	ApplicationID string          `json:"application_id" yaml:"application_id"`
	Region        string          `json:"region" yaml:"region"`
	Environment   string          `json:"environment,omitempty" yaml:"environment,omitempty"`
	ClientSecret  *deliveryReport `json:"client_secret,omitempty" yaml:"client_secret,omitempty"`
}

func (r appCreatedReport) QuietField() string { return r.ApplicationID }

// apiKeyCreatedReport is `nylas dashboard apps apikeys create` as structured
// output.
type apiKeyCreatedReport struct {
	ID            string         `json:"id" yaml:"id"`
	Name          string         `json:"name" yaml:"name"`
	ApplicationID string         `json:"application_id" yaml:"application_id"`
	Region        string         `json:"region" yaml:"region"`
	Delivery      deliveryReport `json:"delivery" yaml:"delivery"`
}

func (r apiKeyCreatedReport) QuietField() string { return r.ID }

// orgSwitchReport is `nylas dashboard orgs switch` as structured output.
type orgSwitchReport struct {
	OrganizationID   string `json:"organization_id" yaml:"organization_id"`
	OrganizationName string `json:"organization_name,omitempty" yaml:"organization_name,omitempty"`
	Switched         bool   `json:"switched" yaml:"switched"`
}

func (r orgSwitchReport) QuietField() string { return r.OrganizationID }

// deliveryReport says where a secret went. Method is the one actually used:
// a clipboard that is unavailable falls back to a file, and says so in
// Warning. Delivered is false only when the secret reached nowhere, which
// today is an activation that failed.
type deliveryReport struct {
	Method    string `json:"method" yaml:"method"`
	Delivered bool   `json:"delivered" yaml:"delivered"`
	File      string `json:"file,omitempty" yaml:"file,omitempty"`
	Warning   string `json:"warning,omitempty" yaml:"warning,omitempty"`

	// fallbackFrom and cause keep the text form's wording; they are
	// unexported so no encoder writes them.
	fallbackFrom string
	cause        error
}

// structuredOutput reports whether --json, --format json|yaml|quiet or
// --quiet was given. Structured runs never prompt: a prompt would write to
// the stream a reader is parsing.
func structuredOutput(cmd *cobra.Command) bool {
	return common.IsStructuredOutput(cmd)
}

// deliverAPIKey hands a new API key to method and reports what happened. It
// prints nothing, so it serves both the text and the structured forms.
func deliverAPIKey(apiKey, appID, region, method string) (deliveryReport, error) {
	switch method {
	case "activate":
		if err := activateAPIKey(apiKey, appID, region, ""); err != nil {
			return deliveryReport{
				Method:  "activate",
				Warning: "could not activate: " + err.Error(),
				cause:   err,
			}, nil
		}
		return deliveryReport{Method: "activate", Delivered: true}, nil
	case "clipboard":
		return deliverToClipboard(apiKey, "nylas-api-key.txt")
	case "file":
		return deliverToFile(apiKey, "nylas-api-key.txt")
	default:
		return deliveryReport{}, dashboardError(
			"invalid API key delivery method",
			"Use --delivery activate, --delivery clipboard, or --delivery file",
		)
	}
}

// deliverClientSecret hands an application's client secret to method.
func deliverClientSecret(secret, method string) (deliveryReport, error) {
	switch method {
	case "clipboard":
		return deliverToClipboard(secret, "nylas-client-secret.txt")
	case "file":
		return deliverToFile(secret, "nylas-client-secret.txt")
	default:
		return deliveryReport{}, dashboardError(
			"invalid secret delivery method",
			"Use --secret-delivery clipboard or --secret-delivery file",
		)
	}
}

func deliverToClipboard(secret, fallbackFile string) (deliveryReport, error) {
	err := common.CopyToClipboard(secret)
	if err == nil {
		return deliveryReport{Method: "clipboard", Delivered: true}, nil
	}
	report, fileErr := deliverToFile(secret, fallbackFile)
	if fileErr != nil {
		return deliveryReport{}, fileErr
	}
	report.Warning = "clipboard unavailable, saved to a file instead: " + err.Error()
	report.fallbackFrom = "clipboard"
	report.cause = err
	return report, nil
}

func deliverToFile(secret, filename string) (deliveryReport, error) {
	path, err := writeSecretTempFile(secret, filename)
	if err != nil {
		return deliveryReport{}, wrapDashboardError(fmt.Errorf("failed to write file: %w", err))
	}
	return deliveryReport{Method: "file", Delivered: true, File: path}, nil
}

// printDelivery is the text form of a delivery, worded as it always was.
func printDelivery(label string, report deliveryReport) {
	if report.fallbackFrom == "clipboard" {
		_, _ = common.Yellow.Printf("  Clipboard unavailable: %v\n", report.cause)
		_, _ = common.Dim.Println("  Falling back to file save")
	}
	switch report.Method {
	case "activate":
		if !report.Delivered {
			_, _ = common.Yellow.Printf("  Could not activate: %v\n", report.cause)
			return
		}
		_, _ = common.Green.Println("✓ API key activated — CLI is ready to use")
		_, _ = common.Dim.Println("  Try: nylas auth status")
	case "clipboard":
		_, _ = common.Green.Printf("✓ %s copied to clipboard\n", label)
	case "file":
		_, _ = common.Green.Printf("✓ %s saved to: %s\n", label, report.File)
		_, _ = common.Dim.Println("  Read it, then delete the file")
	}
}
