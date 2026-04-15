package cache

import (
	"database/sql"
	"testing"
)

func TestMigrateEncryptionPreservesAttachmentsAndOfflineQueue(t *testing.T) {
	originalGetKey := getOrCreateKeyFunc
	originalDeleteKey := deleteKeyFunc
	defer func() {
		getOrCreateKeyFunc = originalGetKey
		deleteKeyFunc = originalDeleteKey
	}()

	key, err := generateKey()
	if err != nil {
		t.Fatalf("generate encryption key: %v", err)
	}
	getOrCreateKeyFunc = func(string) ([]byte, error) { return key, nil }
	deleteKeyFunc = func(string) error { return nil }

	tmpDir := t.TempDir()
	cfg := Config{BasePath: tmpDir}
	email := "encrypted@example.com"

	plainMgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("new plain manager: %v", err)
	}
	defer func() { _ = plainMgr.Close() }()

	db, err := plainMgr.GetDB(email)
	if err != nil {
		t.Fatalf("get plain db: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO attachments (id, email_id, filename, content_type, size, hash, local_path, cached_at, accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "att-1", "email-1", "invoice.pdf", "application/pdf", 42, "hash-1", "/tmp/att-1", 1, 1); err != nil {
		t.Fatalf("insert attachment: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO offline_queue (type, resource_id, payload, created_at)
		VALUES (?, ?, ?, ?)
	`, string(ActionDelete), "email-1", `{"email_id":"email-1"}`, 1); err != nil {
		t.Fatalf("insert offline queue action: %v", err)
	}
	if err := plainMgr.CloseDB(email); err != nil {
		t.Fatalf("close plain db before migration: %v", err)
	}

	encMgr, err := NewEncryptedManager(cfg, EncryptionConfig{Enabled: true})
	if err != nil {
		t.Fatalf("new encrypted manager: %v", err)
	}
	defer func() { _ = encMgr.Close() }()

	if err := encMgr.MigrateToEncrypted(email); err != nil {
		t.Fatalf("migrate to encrypted: %v", err)
	}

	isEncrypted, err := IsEncrypted(encMgr.DBPath(email))
	if err != nil {
		t.Fatalf("detect encrypted db: %v", err)
	}
	if !isEncrypted {
		t.Fatal("expected database to be encrypted after migration")
	}

	rawDB, err := sql.Open(driverName, encMgr.DBPath(email))
	if err != nil {
		t.Fatalf("open encrypted db without key: %v", err)
	}
	defer func() { _ = rawDB.Close() }()

	var rawCount int
	if err := rawDB.QueryRow("SELECT COUNT(*) FROM attachments").Scan(&rawCount); err == nil {
		t.Fatal("expected unencrypted reader to fail on encrypted database")
	}

	encryptedDB, err := encMgr.GetDB(email)
	if err != nil {
		t.Fatalf("open encrypted db with key: %v", err)
	}

	var attachmentCount, queueCount int
	if err := encryptedDB.QueryRow("SELECT COUNT(*) FROM attachments").Scan(&attachmentCount); err != nil {
		t.Fatalf("count attachments after encrypt migration: %v", err)
	}
	if err := encryptedDB.QueryRow("SELECT COUNT(*) FROM offline_queue").Scan(&queueCount); err != nil {
		t.Fatalf("count offline queue after encrypt migration: %v", err)
	}
	if attachmentCount != 1 || queueCount != 1 {
		t.Fatalf("expected migrated attachment/offline queue rows to be preserved, got attachments=%d queue=%d", attachmentCount, queueCount)
	}

	if err := encMgr.CloseDB(email); err != nil {
		t.Fatalf("close encrypted db: %v", err)
	}
	if err := encMgr.MigrateToUnencrypted(email); err != nil {
		t.Fatalf("migrate to unencrypted: %v", err)
	}

	plainDB, err := sql.Open(driverName, encMgr.DBPath(email))
	if err != nil {
		t.Fatalf("open unencrypted db: %v", err)
	}
	defer func() { _ = plainDB.Close() }()

	if err := plainDB.QueryRow("SELECT COUNT(*) FROM attachments").Scan(&attachmentCount); err != nil {
		t.Fatalf("count attachments after decrypt migration: %v", err)
	}
	if err := plainDB.QueryRow("SELECT COUNT(*) FROM offline_queue").Scan(&queueCount); err != nil {
		t.Fatalf("count offline queue after decrypt migration: %v", err)
	}
	if attachmentCount != 1 || queueCount != 1 {
		t.Fatalf("expected round-trip migration to preserve attachment/offline queue rows, got attachments=%d queue=%d", attachmentCount, queueCount)
	}
}
