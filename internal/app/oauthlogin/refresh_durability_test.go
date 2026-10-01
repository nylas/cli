//go:build !integration

package oauthlogin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingWrites makes the fixture's store refuse writes to one key while it
// still behaves as a store for every other key. Setting key to "" heals it.
type failingWrites struct{ key string }

func failWritesTo(f *fixture, key string) *failingWrites {
	fw := &failingWrites{key: key}
	backing := keyring.NewMockSecretStore()
	for k, v := range f.secrets.GetAll() {
		_ = backing.Set(k, v)
	}
	f.secrets.GetFunc = backing.Get
	f.secrets.DeleteFunc = backing.Delete
	f.secrets.SetFunc = func(k, v string) error {
		if fw.key != "" && k == fw.key {
			return errors.New("keychain refused the write")
		}
		return backing.Set(k, v)
	}
	return fw
}

func loggedInAndExpired(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	f.advance(2 * time.Hour)
	return f
}

func TestRefresh_FinishesAndStoresTheRotatedTokenWhenTheCallerGivesUp(t *testing.T) {
	// The server rotates as soon as it answers. A caller that gives up
	// mid-request (Ctrl-C, an MCP request timing out) must not abandon that
	// answer: the stored refresh token is already spent.
	f := loggedInAndExpired(t)
	ctx, cancel := context.WithCancel(context.Background())
	f.client.RefreshFunc = func(reqCtx context.Context, _, _, _ string) (*domain.OAuthTokens, error) {
		cancel()
		if err := reqCtx.Err(); err != nil {
			return nil, err // what an HTTP client does with a cancelled request
		}
		return &domain.OAuthTokens{
			AccessToken:  "at-rotated",
			RefreshToken: "rt-rotated",
			ExpiresAt:    f.clock.Add(time.Hour),
		}, nil
	}

	token, err := f.service.AccessToken(ctx)

	require.NoError(t, err)
	assert.Equal(t, "at-rotated", token)
	assert.Equal(t, "rt-rotated", f.secrets.GetAll()[ports.KeyOAuthRefreshToken])
}

func TestRefresh_DropsTheSpentRefreshTokenWhenTheNewOneCannotBeStored(t *testing.T) {
	f := loggedInAndExpired(t)
	failWritesTo(f, ports.KeyOAuthRefreshToken)

	_, err := f.service.AccessToken(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nylas oauth login")
	_, getErr := f.secrets.Get(ports.KeyOAuthRefreshToken)
	assert.ErrorIs(t, getErr, domain.ErrSecretNotFound, "the spent token would be replayed and burn the family")
}

func TestRefresh_KeepsTheNewRefreshTokenWhenALaterWriteFails(t *testing.T) {
	f := loggedInAndExpired(t)
	failWritesTo(f, ports.KeyOAuthExpiresAt)

	_, err := f.service.AccessToken(context.Background())
	require.Error(t, err)

	stored, getErr := f.secrets.Get(ports.KeyOAuthRefreshToken)
	require.NoError(t, getErr)
	assert.Equal(t, "mock-refresh-token-2", stored)
}

func TestRefresh_APartialWriteRecoversOnTheNextCommand(t *testing.T) {
	// The expiry is written last, so a write that stops before it leaves a
	// session that reads as expired and refreshes with the token just stored.
	f := loggedInAndExpired(t)
	fw := failWritesTo(f, ports.KeyOAuthExpiresAt)
	_, err := f.service.AccessToken(context.Background())
	require.Error(t, err)

	fw.key = ""
	f.client.RefreshCalls = nil

	token, err := f.service.AccessToken(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "mock-access-token-2", token)
	assert.Equal(t, []string{"mock-refresh-token-2"}, f.client.RefreshCalls)
}

func TestSaveTokens_WritesTheExpiryLast(t *testing.T) {
	f := newFixture(t)
	var order []string
	backing := keyring.NewMockSecretStore()
	f.secrets.GetFunc = backing.Get
	f.secrets.DeleteFunc = backing.Delete
	f.secrets.SetFunc = func(k, v string) error {
		order = append(order, k)
		return backing.Set(k, v)
	}

	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, order)
	assert.Equal(t, ports.KeyOAuthExpiresAt, order[len(order)-1])
	assert.Less(t, indexOf(order, ports.KeyOAuthRefreshToken), indexOf(order, ports.KeyOAuthAccessToken))
	assert.Less(t, indexOf(order, ports.KeyOAuthServerURL), indexOf(order, ports.KeyOAuthRefreshToken))
}

func indexOf(list []string, want string) int {
	for i, v := range list {
		if v == want {
			return i
		}
	}
	return -1
}
