package grantcache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_MissingFileReturnsEmpty(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "grants.json"))

	grants, err := store.ListGrants()
	require.NoError(t, err)
	assert.Empty(t, grants)

	defaultGrant, err := store.GetDefaultGrant()
	assert.Empty(t, defaultGrant)
	assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
}

func TestStore_CorruptFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grants.json")
	require.NoError(t, os.WriteFile(path, []byte("{not-json"), 0o600))

	store := New(path)
	grants, err := store.ListGrants()
	require.NoError(t, err)
	assert.Empty(t, grants)

	defaultGrant, err := store.GetDefaultGrant()
	assert.Empty(t, defaultGrant)
	assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
}

func TestStore_SaveGetDeleteAndClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nylas", "grants.json")
	store := New(path)

	grant1 := domain.GrantInfo{ID: "grant-1", Email: "one@example.com", Provider: domain.ProviderGoogle}
	grant2 := domain.GrantInfo{ID: "grant-2", Email: "two@example.com", Provider: domain.ProviderMicrosoft}

	require.NoError(t, store.SaveGrant(grant1))
	require.NoError(t, store.SaveGrant(grant2))
	require.NoError(t, store.SetDefaultGrant(grant1.ID))

	byID, err := store.GetGrant(grant1.ID)
	require.NoError(t, err)
	assert.Equal(t, grant1, *byID)

	byEmail, err := store.GetGrantByEmail(grant2.Email)
	require.NoError(t, err)
	assert.Equal(t, grant2, *byEmail)

	grants, err := store.ListGrants()
	require.NoError(t, err)
	assert.Equal(t, []domain.GrantInfo{grant1, grant2}, grants)

	updated := domain.GrantInfo{ID: "grant-2", Email: "new-two@example.com", Provider: domain.ProviderGoogle}
	require.NoError(t, store.SaveGrant(updated))
	grants, err = store.ListGrants()
	require.NoError(t, err)
	assert.Equal(t, []domain.GrantInfo{grant1, updated}, grants)

	require.NoError(t, store.DeleteGrant(grant1.ID))
	_, err = store.GetDefaultGrant()
	assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
	_, err = store.GetGrant(grant1.ID)
	assert.ErrorIs(t, err, domain.ErrGrantNotFound)

	require.NoError(t, store.ClearGrants())
	grants, err = store.ListGrants()
	require.NoError(t, err)
	assert.Empty(t, grants)
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestStore_FilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nylas", "grants.json")
	store := New(path)

	require.NoError(t, store.SaveGrant(domain.GrantInfo{
		ID:    "grant-1",
		Email: "one@example.com",
	}))

	dirInfo, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
}

func TestStore_ReplaceGrantsPreservesValidDefault(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "grants.json"))
	require.NoError(t, store.SaveGrant(domain.GrantInfo{
		ID:       "grant-1",
		Email:    "old@example.com",
		Provider: domain.ProviderGoogle,
	}))
	require.NoError(t, store.SetDefaultGrant("grant-1"))

	require.NoError(t, store.ReplaceGrants([]domain.GrantInfo{
		{ID: "grant-1", Email: "new@example.com", Provider: domain.ProviderMicrosoft},
		{ID: "grant-2", Email: "two@example.com", Provider: domain.ProviderGoogle},
	}))

	defaultGrant, err := store.GetDefaultGrant()
	require.NoError(t, err)
	assert.Equal(t, "grant-1", defaultGrant)

	grant, err := store.GetGrant("grant-1")
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", grant.Email)
	assert.Equal(t, domain.ProviderMicrosoft, grant.Provider)
}

func TestStore_ReplaceGrantsClearsMissingDefault(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "grants.json"))
	require.NoError(t, store.SaveGrant(domain.GrantInfo{
		ID:       "grant-1",
		Email:    "old@example.com",
		Provider: domain.ProviderGoogle,
	}))
	require.NoError(t, store.SetDefaultGrant("grant-1"))

	require.NoError(t, store.ReplaceGrants([]domain.GrantInfo{
		{ID: "grant-2", Email: "two@example.com", Provider: domain.ProviderGoogle},
	}))

	defaultGrant, err := store.GetDefaultGrant()
	assert.Empty(t, defaultGrant)
	assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
}

func TestStore_RejectsInvalidGrantInfo(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "grants.json"))

	assert.ErrorIs(t, store.SaveGrant(domain.GrantInfo{ID: "grant-1"}), domain.ErrInvalidInput)
	assert.ErrorIs(t, store.ReplaceGrants([]domain.GrantInfo{{ID: "grant-1"}}), domain.ErrInvalidInput)
}

func TestStore_ConcurrentWritersPreserveUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grants.json")
	stores := []*Store{New(path), New(path)}

	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 40; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- stores[i%len(stores)].SaveGrant(domain.GrantInfo{
				ID:    fmt.Sprintf("grant-%02d", i),
				Email: fmt.Sprintf("user-%02d@example.com", i),
			})
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	grants, err := New(path).ListGrants()
	require.NoError(t, err)
	assert.Len(t, grants, 40)
	seen := make(map[string]bool, len(grants))
	for _, grant := range grants {
		seen[grant.ID] = true
	}
	for i := 0; i < 40; i++ {
		assert.True(t, seen[fmt.Sprintf("grant-%02d", i)])
	}
}
