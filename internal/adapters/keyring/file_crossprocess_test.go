package keyring

import (
	"fmt"
	"sync"
	"testing"
)

// TestEncryptedFileStore_WritesFromSeparateStoresAllSurvive stands in for two
// CLI processes: two stores on one directory share no mutex, so only the
// file lock stops a read-modify-write in one from discarding the other's —
// the lost write that, for an OAuth session, puts back a rotated refresh
// token and revokes the whole family on the next refresh.
func TestEncryptedFileStore_WritesFromSeparateStoresAllSurvive(t *testing.T) {
	dir := t.TempDir()
	setFileStorePassphrase(t)

	stores := make([]*EncryptedFileStore, 2)
	for i := range stores {
		store, err := NewEncryptedFileStore(dir)
		if err != nil {
			t.Fatalf("NewEncryptedFileStore: %v", err)
		}
		stores[i] = store
	}

	const perStore = 8
	var wg sync.WaitGroup
	for i, store := range stores {
		wg.Add(1)
		go func(i int, store *EncryptedFileStore) {
			defer wg.Done()
			for n := range perStore {
				if err := store.Set(fmt.Sprintf("key-%d-%d", i, n), "value"); err != nil {
					t.Errorf("Set: %v", err)
				}
			}
		}(i, store)
	}
	wg.Wait()

	reader, err := NewEncryptedFileStore(dir)
	if err != nil {
		t.Fatalf("NewEncryptedFileStore: %v", err)
	}
	for i := range stores {
		for n := range perStore {
			key := fmt.Sprintf("key-%d-%d", i, n)
			if _, err := reader.Get(key); err != nil {
				t.Errorf("%s was lost: %v", key, err)
			}
		}
	}
}
