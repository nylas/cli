package mcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/mcp"
	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/common"
	oauthcli "github.com/nylas/cli/internal/cli/oauth"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

// Values of `nylas mcp serve --auth`.
const (
	authAPIKey = "api-key"
	authOAuth  = "oauth"
)

func newServeCmd() *cobra.Command {
	var authMode string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start the MCP server to enable AI assistants to interact with Nylas.

This command acts as a proxy to the official Nylas MCP server, providing
access to all Nylas email, calendar, and contacts tools through the
Model Context Protocol.

The proxy dynamically discovers available tools from the upstream Nylas
MCP server and adds local enhancements:
  - Automatic grant_id injection (no need to specify which account)
  - Local grant lookup (get_grant without email)
  - Timezone-aware timestamp display
  - Secure credential handling via system keyring

Authentication (--auth):
  api-key  (default) the API key from 'nylas auth login', sent to the MCP
           server of the configured region.
  oauth    the OAuth session from 'nylas oauth login --for mcp'. The token
           is refreshed as it nears expiry (one refresher per machine, shared
           with every other 'nylas mcp serve'), and requests go to the MCP
           server named in the token's audience. The default grant is only
           offered as a hint when the token lists it; local grant lookup is
           off, since local grants belong to the API key's application.

The server communicates via STDIO (standard input/output).

For more information: https://developer.nylas.com/docs/dev-guide/mcp/`,
		Example: `  # Proxy with the API key (default)
  nylas mcp serve

  # Proxy with an OAuth session
  nylas oauth login --for mcp
  nylas mcp serve --auth oauth`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proxy, err := buildProxy(authMode)
			if err != nil {
				return err
			}
			return runProxy(proxy)
		},
	}

	cmd.Flags().StringVar(&authMode, "auth", authAPIKey, "credential to authenticate with: api-key or oauth")

	return cmd
}

// buildProxy constructs the proxy for an --auth value. Anything but the two
// known modes is refused rather than defaulted.
func buildProxy(authMode string) (*mcp.Proxy, error) {
	switch authMode {
	case authAPIKey:
		return buildAPIKeyProxy()
	case authOAuth:
		return buildOAuthProxy()
	default:
		return nil, common.NewUserError(
			fmt.Sprintf("unknown --auth value %q", authMode),
			"supported: api-key, oauth",
		)
	}
}

func buildAPIKeyProxy() (*mcp.Proxy, error) {
	// Get API key from credentials
	apiKey, err := common.GetAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w\n\nPlease run 'nylas auth login' first", err)
	}

	// Get region from config (defaults to "us")
	region := "us"
	configStore := config.NewDefaultFileStore()
	if cfg, err := configStore.Load(); err == nil && cfg.Region != "" {
		region = cfg.Region
	}

	// Create MCP proxy with region
	proxy := mcp.NewProxy(apiKey, region)
	setDefaultGrant(proxy)

	// Set up grant store for local grant lookups (allows get_grant without email).
	if grantStore, err := common.NewDefaultGrantStore(); err == nil {
		proxy.SetGrantStore(grantStore)
	}
	return proxy, nil
}

// newOAuthCredentialsFn is swapped in tests.
var newOAuthCredentialsFn = func() (ports.MCPCredentialSource, error) {
	svc, err := oauthcli.NewLoginService()
	if err != nil {
		return nil, err
	}
	return oauthlogin.NewMCPCredentials(svc), nil
}

// oauthPreflightTimeout bounds the start-up check, which may refresh.
const oauthPreflightTimeout = 30 * time.Second

func buildOAuthProxy() (*mcp.Proxy, error) {
	creds, err := newOAuthCredentialsFn()
	if err != nil {
		return nil, err
	}

	// Fail at start-up, where the message reaches a terminal or the
	// assistant's server log, rather than on the first tool call.
	ctx, cancel := context.WithTimeout(context.Background(), oauthPreflightTimeout)
	defer cancel()
	if _, err := creds.Credential(ctx); err != nil {
		return nil, fmt.Errorf("no usable OAuth session for the Nylas MCP server: %w\n\nRun '%s' first", err, mcp.OAuthLoginCommand)
	}

	proxy := mcp.NewOAuthProxy(creds)
	// A hint only: the proxy offers it per request, and only when the
	// current token lists that grant.
	setDefaultGrant(proxy)
	return proxy, nil
}

func setDefaultGrant(proxy *mcp.Proxy) {
	// Get default grant ID (optional - helps Claude know which account to use)
	if grantID, _ := common.GetGrantID(nil); grantID != "" {
		proxy.SetDefaultGrant(grantID)
	}
}

func runProxy(proxy *mcp.Proxy) error {
	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Run the proxy (blocks until context is cancelled or error)
	return proxy.Run(ctx)
}
