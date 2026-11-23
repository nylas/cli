package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// The structured reports exist so an agent can drive the dashboard commands.
// An agent's stdout lands in its transcript, so the property these tests
// guard above all is that a secret never reaches stdout in any format.

const testSecret = "nyk_v0_THIS-MUST-NEVER-BE-PRINTED"

func structuredCmd(t *testing.T, flag string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	// Secret files land in a directory the test removes, never the real temp dir.
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &cobra.Command{Use: "test"}
	addTestOutputFlags(cmd)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if flag != "" {
		require.NoError(t, cmd.Flags().Set(flag, "true"))
	}
	return cmd, &out
}

func TestWriteAPIKeyCreated_ReportsWhereTheKeyWentAndNeverTheKey(t *testing.T) {
	cmd, out := structuredCmd(t, "json")
	key := &domain.GatewayCreatedAPIKey{ID: "key_1", Name: "agent key", APIKey: testSecret}

	require.NoError(t, writeAPIKeyCreated(cmd, key, "app_1", "us", "file"))

	assert.NotContains(t, out.String(), testSecret)
	var report map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	assert.Equal(t, "key_1", report["id"])
	assert.Equal(t, "agent key", report["name"])
	assert.Equal(t, "app_1", report["application_id"])
	assert.Equal(t, "us", report["region"])

	delivery := report["delivery"].(map[string]any)
	assert.Equal(t, "file", delivery["method"])
	assert.Equal(t, true, delivery["delivered"])
	path, _ := delivery["file"].(string)

	// The key is where the report says, and readable only by its owner.
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, testSecret+"\n", string(contents))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestWriteAPIKeyCreated_QuietPrintsOnlyTheKeyID(t *testing.T) {
	cmd, out := structuredCmd(t, "quiet")
	key := &domain.GatewayCreatedAPIKey{ID: "key_1", Name: "agent key", APIKey: testSecret}

	require.NoError(t, writeAPIKeyCreated(cmd, key, "app_1", "us", "file"))

	assert.Equal(t, "key_1\n", out.String())
}

func TestWriteAPIKeyCreated_YAMLNeverCarriesTheKey(t *testing.T) {
	cmd, out := structuredCmd(t, "")
	require.NoError(t, cmd.Flags().Set("format", "yaml"))
	key := &domain.GatewayCreatedAPIKey{ID: "key_1", Name: "agent key", APIKey: testSecret}

	require.NoError(t, writeAPIKeyCreated(cmd, key, "app_1", "us", "file"))

	assert.NotContains(t, out.String(), testSecret)
	assert.Contains(t, out.String(), "id: key_1")
	assert.Contains(t, out.String(), "method: file")
}

func TestWriteAppCreated_ReportsTheClientSecretDeliveryAndNeverTheSecret(t *testing.T) {
	cmd, out := structuredCmd(t, "json")
	app := &domain.GatewayCreatedApplication{
		ApplicationID: "app_1",
		Region:        "eu",
		Environment:   "sandbox",
		ClientSecret:  testSecret,
	}

	require.NoError(t, writeAppCreated(cmd, app, "file"))

	assert.NotContains(t, out.String(), testSecret)
	var report appCreatedReport
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	assert.Equal(t, "app_1", report.ApplicationID)
	assert.Equal(t, "eu", report.Region)
	assert.Equal(t, "sandbox", report.Environment)
	require.NotNil(t, report.ClientSecret)
	assert.Equal(t, "file", report.ClientSecret.Method)
	assert.True(t, report.ClientSecret.Delivered)
}

func TestWriteAppCreated_WithoutAClientSecretSaysNothingAboutOne(t *testing.T) {
	cmd, out := structuredCmd(t, "json")
	app := &domain.GatewayCreatedApplication{ApplicationID: "app_1", Region: "us"}

	require.NoError(t, writeAppCreated(cmd, app, "file"))

	assert.JSONEq(t, `{"application_id":"app_1","region":"us"}`, out.String())
}

func TestDeliverAPIKey_RejectsAnUnknownMethodWithoutTouchingTheKey(t *testing.T) {
	_, err := deliverAPIKey(testSecret, "app_1", "us", "print")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key delivery method")
	assert.NotContains(t, err.Error(), testSecret)
}

func TestDeliverClientSecret_RefusesActivate(t *testing.T) {
	// A client secret is not an API key: it cannot configure the CLI.
	_, err := deliverClientSecret(testSecret, "activate")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid secret delivery method")
}

func TestStructuredRunsNeverPrompt(t *testing.T) {
	// A prompt in a structured run would write into the stream being parsed,
	// and in an agent's shell it would hang until the call times out. So each
	// command that would prompt refuses instead, before calling any service.
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
		want string
	}{
		{
			name: "apps use without an application",
			cmd:  newAppsUseCmd,
			args: []string{"--json"},
			want: "application selection requires an interactive terminal",
		},
		{
			name: "apps create without a secret delivery",
			cmd:  newAppsCreateCmd,
			args: []string{"--json", "--name", "agent app", "--region", "us"},
			want: "client secret delivery requires an explicit choice",
		},
		{
			name: "apikeys create without a delivery",
			cmd:  newAPIKeysCreateCmd,
			args: []string{"--json", "--app", "app_1", "--region", "us"},
			want: "API key delivery requires an explicit choice",
		},
	}
	// Stand in a terminal: without --json each of these would prompt, so
	// --json has to be what refuses.
	orig := isInteractive
	isInteractive = func() bool { return true }
	t.Cleanup(func() { isInteractive = orig })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd()
			addTestOutputFlags(cmd)

			out, err := executeTestCommand(cmd, tt.args...)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
			assert.NotContains(t, out, "{")
		})
	}
}

func TestNewStatusReport_CarriesWhatTheTextFormShows(t *testing.T) {
	secrets := newMemSecretStore()
	require.NoError(t, secrets.Set(ports.KeyDashboardAppID, "app_1"))
	require.NoError(t, secrets.Set(ports.KeyDashboardAppRegion, "eu"))
	session := &domain.DashboardSessionResponse{
		User:       domain.DashboardUser{PublicID: "user_from_session"},
		CurrentOrg: "org_2",
		Relations: []domain.DashboardSessionRelation{
			{OrgPublicID: "org_1", OrgName: "First"},
			{OrgPublicID: "org_2", OrgName: "Second"},
		},
	}
	status := dashboardapp.Status{LoggedIn: true, OrgID: "org_1", HasOrgToken: true}

	report := newStatusReport(status, session, secrets, "thumb")

	assert.Equal(t, statusReport{
		LoggedIn:          true,
		UserID:            "user_from_session",
		OrganizationID:    "org_2",
		OrganizationName:  "Second",
		OrganizationCount: 2,
		HasOrgToken:       true,
		ActiveApp:         &activeAppReport{ApplicationID: "app_1", Region: "eu"},
		DPoPThumbprint:    "thumb",
	}, report)
}

func TestStatusReport_SignedOutIsAnExplicitFalse(t *testing.T) {
	// An agent reads logged_in to decide whether to sign in, so a signed-out
	// report must say false rather than leave the field out.
	b, err := json.Marshal(statusReport{})

	require.NoError(t, err)
	assert.JSONEq(t, `{"logged_in":false,"has_org_token":false}`, string(b))
}

func TestCaptureReloginOrg_ReportsTheOrganizationTheSignInChose(t *testing.T) {
	calls := 0
	chosen, relogin := captureReloginOrg(reloginTo("org_9", &calls))

	err := switchOAuthSessionOrg(context.Background(), &bytes.Buffer{}, "", relogin)

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, "org_9", *chosen)
}

func TestCaptureReloginOrg_KeepsAMissingReloginMissing(t *testing.T) {
	chosen, relogin := captureReloginOrg(nil)

	assert.Nil(t, relogin)
	assert.Empty(t, *chosen)
}
