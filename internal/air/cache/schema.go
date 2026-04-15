package cache

import (
	"database/sql"
	"fmt"
)

// schemaVersion is used for migrations.
const schemaVersion = 1

// initSchema creates the database schema if it doesn't exist.
func initSchema(db *sql.DB) error {
	// Check current schema version
	var version int
	err := db.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}

	if version >= schemaVersion {
		return nil // Already up to date
	}

	// Create tables in a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Create emails table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS emails (
			id TEXT PRIMARY KEY,
			thread_id TEXT,
			folder_id TEXT,
			subject TEXT,
			snippet TEXT,
			from_name TEXT,
			from_email TEXT,
			to_json TEXT,
			cc_json TEXT,
			bcc_json TEXT,
			date INTEGER,
			unread INTEGER DEFAULT 1,
			starred INTEGER DEFAULT 0,
			has_attachments INTEGER DEFAULT 0,
			body_html TEXT,
			body_text TEXT,
			headers_json TEXT,
			cached_at INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("create emails table: %w", err)
	}

	// Create FTS5 index for emails
	_, err = tx.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS emails_fts USING fts5(
			subject,
			snippet,
			body_text,
			from_name,
			from_email,
			content='emails',
			content_rowid='rowid'
		)
	`)
	if err != nil {
		return fmt.Errorf("create emails_fts: %w", err)
	}

	// Create triggers to keep FTS in sync
	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS emails_ai AFTER INSERT ON emails BEGIN
			INSERT INTO emails_fts(rowid, subject, snippet, body_text, from_name, from_email)
			VALUES (new.rowid, new.subject, new.snippet, new.body_text, new.from_name, new.from_email);
		END
	`)
	if err != nil {
		return fmt.Errorf("create emails_ai trigger: %w", err)
	}

	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS emails_ad AFTER DELETE ON emails BEGIN
			INSERT INTO emails_fts(emails_fts, rowid, subject, snippet, body_text, from_name, from_email)
			VALUES ('delete', old.rowid, old.subject, old.snippet, old.body_text, old.from_name, old.from_email);
		END
	`)
	if err != nil {
		return fmt.Errorf("create emails_ad trigger: %w", err)
	}

	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS emails_au AFTER UPDATE ON emails BEGIN
			INSERT INTO emails_fts(emails_fts, rowid, subject, snippet, body_text, from_name, from_email)
			VALUES ('delete', old.rowid, old.subject, old.snippet, old.body_text, old.from_name, old.from_email);
			INSERT INTO emails_fts(rowid, subject, snippet, body_text, from_name, from_email)
			VALUES (new.rowid, new.subject, new.snippet, new.body_text, new.from_name, new.from_email);
		END
	`)
	if err != nil {
		return fmt.Errorf("create emails_au trigger: %w", err)
	}

	// Create events table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			calendar_id TEXT,
			title TEXT,
			description TEXT,
			location TEXT,
			start_time INTEGER,
			end_time INTEGER,
			all_day INTEGER DEFAULT 0,
			recurring INTEGER DEFAULT 0,
			rrule TEXT,
			participants_json TEXT,
			status TEXT,
			busy INTEGER DEFAULT 1,
			cached_at INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	// Create FTS5 index for events
	_, err = tx.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
			title,
			description,
			location,
			content='events',
			content_rowid='rowid'
		)
	`)
	if err != nil {
		return fmt.Errorf("create events_fts: %w", err)
	}

	// Create triggers for events FTS
	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS events_ai AFTER INSERT ON events BEGIN
			INSERT INTO events_fts(rowid, title, description, location)
			VALUES (new.rowid, new.title, new.description, new.location);
		END
	`)
	if err != nil {
		return fmt.Errorf("create events_ai trigger: %w", err)
	}

	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS events_ad AFTER DELETE ON events BEGIN
			INSERT INTO events_fts(events_fts, rowid, title, description, location)
			VALUES ('delete', old.rowid, old.title, old.description, old.location);
		END
	`)
	if err != nil {
		return fmt.Errorf("create events_ad trigger: %w", err)
	}

	// Create contacts table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS contacts (
			id TEXT PRIMARY KEY,
			given_name TEXT,
			surname TEXT,
			display_name TEXT,
			email TEXT,
			phone TEXT,
			company TEXT,
			job_title TEXT,
			notes TEXT,
			photo_url TEXT,
			groups_json TEXT,
			cached_at INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("create contacts table: %w", err)
	}

	// Create FTS5 index for contacts
	_, err = tx.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS contacts_fts USING fts5(
			given_name,
			surname,
			display_name,
			email,
			company,
			content='contacts',
			content_rowid='rowid'
		)
	`)
	if err != nil {
		return fmt.Errorf("create contacts_fts: %w", err)
	}

	// Create triggers for contacts FTS
	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS contacts_ai AFTER INSERT ON contacts BEGIN
			INSERT INTO contacts_fts(rowid, given_name, surname, display_name, email, company)
			VALUES (new.rowid, new.given_name, new.surname, new.display_name, new.email, new.company);
		END
	`)
	if err != nil {
		return fmt.Errorf("create contacts_ai trigger: %w", err)
	}

	_, err = tx.Exec(`
		CREATE TRIGGER IF NOT EXISTS contacts_ad AFTER DELETE ON contacts BEGIN
			INSERT INTO contacts_fts(contacts_fts, rowid, given_name, surname, display_name, email, company)
			VALUES ('delete', old.rowid, old.given_name, old.surname, old.display_name, old.email, old.company);
		END
	`)
	if err != nil {
		return fmt.Errorf("create contacts_ad trigger: %w", err)
	}

	// Create folders table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS folders (
			id TEXT PRIMARY KEY,
			name TEXT,
			type TEXT,
			unread_count INTEGER DEFAULT 0,
			total_count INTEGER DEFAULT 0,
			cached_at INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("create folders table: %w", err)
	}

	// Create calendars table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS calendars (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			is_primary INTEGER DEFAULT 0,
			read_only INTEGER DEFAULT 0,
			hex_color TEXT,
			cached_at INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("create calendars table: %w", err)
	}

	// Create sync_state table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS sync_state (
			resource TEXT PRIMARY KEY,
			last_sync INTEGER,
			cursor TEXT,
			metadata_json TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("create sync_state table: %w", err)
	}

	// Create attachments metadata table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS attachments (
			id TEXT PRIMARY KEY,
			email_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			content_type TEXT,
			size INTEGER NOT NULL,
			hash TEXT NOT NULL,
			local_path TEXT NOT NULL,
			cached_at INTEGER NOT NULL,
			accessed_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create attachments table: %w", err)
	}

	// Create offline action queue table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS offline_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			payload TEXT,
			created_at INTEGER NOT NULL,
			attempts INTEGER DEFAULT 0,
			last_error TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("create offline_queue table: %w", err)
	}

	// Create indexes for common queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_emails_folder ON emails(folder_id)",
		"CREATE INDEX IF NOT EXISTS idx_emails_thread ON emails(thread_id)",
		"CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date DESC)",
		"CREATE INDEX IF NOT EXISTS idx_emails_unread ON emails(unread) WHERE unread = 1",
		"CREATE INDEX IF NOT EXISTS idx_emails_starred ON emails(starred) WHERE starred = 1",
		"CREATE INDEX IF NOT EXISTS idx_events_calendar ON events(calendar_id)",
		"CREATE INDEX IF NOT EXISTS idx_events_time ON events(start_time, end_time)",
		"CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email)",
		"CREATE INDEX IF NOT EXISTS idx_attachments_email ON attachments(email_id)",
		"CREATE INDEX IF NOT EXISTS idx_attachments_hash ON attachments(hash)",
		"CREATE INDEX IF NOT EXISTS idx_attachments_accessed ON attachments(accessed_at)",
		"CREATE INDEX IF NOT EXISTS idx_offline_queue_created ON offline_queue(created_at)",
	}
	for _, idx := range indexes {
		if _, err = tx.Exec(idx); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	// Update schema version
	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion))
	if err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}

	return tx.Commit()
}
