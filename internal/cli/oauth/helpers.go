package oauth

import (
	"context"
	"errors"

	"github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	oauthadapter "github.com/nylas/cli/internal/adapters/oauth"
	"github.com/nylas/cli/internal/adapters/oauthas"
	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/dashboard"
	"github.com/nylas/cli/internal/domain"
)

// loginService is the slice of oauthlogin.Service the commands use, named so
// tests can substitute a fake via createLoginServiceFn.
type loginService interface {
	Login(ctx context.Context, scopes []string) (*oauthlogin.LoginResult, error)
	Status() (*oauthlogin.Session, error)
	AccessToken(ctx context.Context) (string, error)
	UserInfo(ctx context.Context) (*domain.OAuthUserInfo, error)
	Logout(ctx context.Context) error
}

var createLoginServiceFn = func() (loginService, error) { return createLoginService() }

// createLoginService wires the OAuth login service. The authorization server
// is hosted by dashboard-account, so it resolves to the same base URL as the
// `nylas dashboard` commands (NYLAS_DASHBOARD_ACCOUNT_URL overrides it).
func createLoginService() (*oauthlogin.Service, error) {
	secrets, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return nil, err
	}

	cfg, _ := config.NewDefaultFileStore().Load()
	callbackPort := 0
	if cfg != nil {
		callbackPort = cfg.CallbackPort
	}

	client := oauthas.NewClient(dashboard.AccountBaseURL())
	callbackServer := oauthadapter.NewCallbackServer(callbackPort)

	return oauthlogin.NewService(client, callbackServer, browser.NewDefaultBrowser(), secrets), nil
}

// wrapOAuthError turns the not-logged-in sentinel into a CLI error that
// names the command to run, and passes everything else through.
func wrapOAuthError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrOAuthNotLoggedIn) {
		return &common.CLIError{
			Err:     err,
			Message: "Not logged in\n  Hint: run `nylas oauth login`",
		}
	}
	var cliErr *common.CLIError
	if errors.As(err, &cliErr) {
		return cliErr
	}
	return &common.CLIError{Err: err, Message: err.Error()}
}
