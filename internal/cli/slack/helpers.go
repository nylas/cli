// helpers.go provides shared utilities for Slack CLI commands including
// token management, client creation, and channel resolution.

package slack

import (
	"context"
	"os"
	"strings"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	slackadapter "github.com/nylas/cli/internal/adapters/slack"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const slackTokenKey = "slack_user_token"

// storeSlackToken stores the Slack token in the keyring.
func storeSlackToken(token string) error {
	store, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return err
	}
	return store.Set(slackTokenKey, token)
}

// getSlackToken retrieves the Slack token from environment or keyring.
func getSlackToken() (string, error) {
	if token := os.Getenv("SLACK_USER_TOKEN"); token != "" {
		return token, nil
	}

	store, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return "", err
	}

	token, err := store.Get(slackTokenKey)
	if err != nil {
		return "", domain.ErrSlackNotConfigured
	}

	return token, nil
}

// removeSlackToken removes the Slack token from the keyring.
func removeSlackToken() error {
	store, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return err
	}
	return store.Delete(slackTokenKey)
}

// getSlackClientFromKeyring creates a client using stored credentials.
func getSlackClientFromKeyring() (ports.SlackClient, error) {
	token, err := getSlackToken()
	if err != nil {
		return nil, err
	}
	return getSlackClient(token)
}

// getSlackClient creates a new Slack client with the given token.
func getSlackClient(token string) (ports.SlackClient, error) {
	config := slackadapter.DefaultConfig()
	config.UserToken = token
	config.Debug = os.Getenv("SLACK_DEBUG") == "true"
	return slackadapter.NewClient(config)
}

// withSlackClient creates a Slack client and context, then runs fn.
func withSlackClient(fn func(ctx context.Context, client ports.SlackClient) error) error {
	client, err := getSlackClientOrError()
	if err != nil {
		return err
	}
	ctx, cancel := common.CreateContext()
	defer cancel()
	return fn(ctx, client)
}

// getSlackClientOrError wraps getSlackClientFromKeyring with a user-friendly error.
func getSlackClientOrError() (ports.SlackClient, error) {
	client, err := getSlackClientFromKeyring()
	if err != nil {
		return nil, common.NewUserError(
			"not authenticated with Slack",
			"Run: nylas slack auth set --token YOUR_TOKEN",
		)
	}
	return client, nil
}

// createContext creates a context with default timeout.
// Uses common.CreateContext for consistency across CLI packages.

// resolveChannelName resolves a channel name to its ID.
// If name already looks like a channel ID (starts with C, G, or D), returns it directly.
// NOTE: For workspaces with many channels, use channel ID directly to avoid rate limits.
func resolveChannelName(ctx context.Context, client ports.SlackClient, name string) (string, error) {
	// If it's already a channel ID, return it directly
	if isChannelID(name) {
		return name, nil
	}

	// Normalize name
	name = strings.TrimPrefix(name, "#")
	name = strings.ToLower(name)

	// Try to resolve using the adapter (searches first page only for speed)
	slackClient, ok := client.(*slackadapter.Client)
	if ok {
		ch, err := slackClient.ResolveChannelByName(ctx, name)
		if err != nil {
			return "", err
		}
		return ch.ID, nil
	}

	// Fallback for mock: search first page only
	resp, err := client.ListChannels(ctx, &domain.SlackChannelQueryParams{
		Types:           []string{"public_channel", "private_channel"},
		ExcludeArchived: true,
		Limit:           200,
	})
	if err != nil {
		return "", err
	}

	for _, ch := range resp.Channels {
		if strings.ToLower(ch.Name) == name {
			return ch.ID, nil
		}
	}

	return "", domain.ErrSlackChannelNotFound
}

// isChannelID checks if a string looks like a Slack channel ID.
// Channel IDs start with C (public), G (private/group), or D (DM).
func isChannelID(s string) bool {
	if len(s) < 9 {
		return false
	}
	first := s[0]
	return first == 'C' || first == 'G' || first == 'D'
}
