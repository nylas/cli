package oauth

import (
	"context"
	"errors"
	"fmt"
	"os"

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

	clientID, err := resolveClientID()
	if err != nil {
		return nil, err
	}

	client := oauthas.NewClient(dashboard.AccountBaseURL())
	// 127.0.0.1 rather than localhost: it is one of the two spellings the
	// server registers for the static client, and it is the address the
	// server actually binds, so the browser cannot land on the other family.
	callbackServer := oauthadapter.NewLoopbackIPCallbackServer(callbackPort)

	return oauthlogin.NewService(clientID, client, callbackServer, browser.NewDefaultBrowser(), secrets), nil
}

// clientIDEnv overrides the static public client id, for a local or dev
// authorization server that registers the CLI under another id.
const clientIDEnv = "NYLAS_OAUTH_CLIENT_ID"

// resolveClientID returns the client id override when one is set, and the
// static id otherwise. An override that fails validation is an error rather
// than a silent fall back to the default, so a typo is noticed.
func resolveClientID() (string, error) {
	override := os.Getenv(clientIDEnv)
	if override == "" {
		return domain.DefaultOAuthClientID, nil
	}
	if err := domain.ValidateOAuthClientID(override); err != nil {
		return "", fmt.Errorf("%s: %w", clientIDEnv, err)
	}
	return override, nil
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
