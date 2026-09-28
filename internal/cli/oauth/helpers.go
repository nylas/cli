package oauth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/filelock"
	"github.com/nylas/cli/internal/adapters/keyring"
	oauthadapter "github.com/nylas/cli/internal/adapters/oauth"
	"github.com/nylas/cli/internal/adapters/oauthas"
	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// loginService is the slice of oauthlogin.Service the commands use, named so
// tests can substitute a fake via createLoginServiceFn.
type loginService interface {
	Login(ctx context.Context, opts oauthlogin.LoginOptions) (*oauthlogin.LoginResult, error)
	Status() (*oauthlogin.Session, error)
	AccessToken(ctx context.Context) (string, error)
	UserInfo(ctx context.Context) (*domain.OAuthUserInfo, error)
	Logout(ctx context.Context) error
}

var createLoginServiceFn = func() (loginService, error) { return createLoginService() }

var (
	exchangeDashboardSessionFn = dashboard.ExchangeOAuthSession
	clearDashboardSessionFn    = dashboard.ClearOAuthSession
)

// The dashboard commands renew a session from `nylas oauth login` through the
// same login service, so refreshes share this machine's session lock.
func init() {
	dashboard.OAuthTokenSource = func() (dashboardapp.OAuthAccessTokens, error) {
		return createLoginService()
	}
}

// NewLoginService returns the OAuth login service wired to this machine's
// keyring, session lock and authorization server. `nylas mcp serve --auth
// oauth` uses it so its refreshes share the lock with every other process.
func NewLoginService() (*oauthlogin.Service, error) {
	return createLoginService()
}

// configuredRegion is the CLI's configured region, "" when none is set. A
// variable so tests can set it without a config file.
var configuredRegion = func() string {
	cfg, err := config.NewDefaultFileStore().Load()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.Region
}

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

	return oauthlogin.NewService(clientID, client, callbackServer, browser.NewDefaultBrowser(), secrets, sessionLock(secrets)), nil
}

// sessionLockFile serialises OAuth session writes across every CLI process
// on the machine — in practice, several `nylas mcp serve` processes started
// by different assistants that share one keyring session.
const sessionLockFile = "oauth-session.lock"

func sessionLock(secrets ports.SecretStore) *filelock.Lock {
	return filelock.New(sessionLockPath(secrets))
}

// sessionLockPath puts the lock with the secrets it protects, so every
// process sharing a session also shares its lock.
//
// The encrypted file store lives in the config directory, so its lock does
// too. The system keyring is one store per user whatever XDG_CONFIG_HOME says,
// and an editor or MCP host can start `nylas mcp serve` with a different
// XDG_CONFIG_HOME than the user's shell. A lock that followed it would let two
// processes refresh the same session at once, and a replayed refresh token
// revokes the whole family. So for the keyring the lock sits under the
// account's home directory from the user database, not from the environment.
func sessionLockPath(secrets ports.SecretStore) string {
	if _, ok := secrets.(*keyring.EncryptedFileStore); ok {
		return filepath.Join(config.DefaultConfigDir(), sessionLockFile)
	}
	if account, err := user.Current(); err == nil && account.HomeDir != "" {
		return filepath.Join(account.HomeDir, ".config", "nylas", sessionLockFile)
	}
	return filepath.Join(config.DefaultConfigDir(), sessionLockFile)
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
