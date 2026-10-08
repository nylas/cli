package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/testutil"
)

func TestLoginCmd_KeptDashboardLoginSessionNamesTheWayOutOfAnExpiredOne(t *testing.T) {
	// A kept `nylas dashboard login` session is not checked for expiry, so an
	// expired one blocks the exchange on every login until it is logged out.
	// The message is the only place the user learns how to get past it.
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{Issuer: "i", ClientID: "c", HasRefresh: true}}
	dash := withService(t, svc)
	dash.exchangeErr = dashboardapp.ErrDashboardLoginSessionKept

	stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd())
	require.NoError(t, err)

	assert.Contains(t, stdout, "if that session has expired")
	assert.Contains(t, stdout, "nylas dashboard logout")
}
