package cache

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// bufferPool provides reusable buffers for JSON marshaling to reduce allocations.
var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// CachedEmail represents an email stored in the cache.
type CachedEmail struct {
	ID             string    `json:"id"`
	ThreadID       string    `json:"thread_id,omitempty"`
	FolderID       string    `json:"folder_id,omitempty"`
	Subject        string    `json:"subject"`
	Snippet        string    `json:"snippet"`
	FromName       string    `json:"from_name"`
	FromEmail      string    `json:"from_email"`
	To             []string  `json:"to,omitempty"`
	CC             []string  `json:"cc,omitempty"`
	BCC            []string  `json:"bcc,omitempty"`
	Date           time.Time `json:"date"`
	Unread         bool      `json:"unread"`
	Starred        bool      `json:"starred"`
	HasAttachments bool      `json:"has_attachments"`
	BodyHTML       string    `json:"body_html,omitempty"`
	BodyText       string    `json:"body_text,omitempty"`
	CachedAt       time.Time `json:"cached_at"`
}

// EmailStore provides email caching operations.
type EmailStore struct {
	db *sql.DB
}

// NewEmailStore creates an email store for a database.
func NewEmailStore(db *sql.DB) *EmailStore {
	return &EmailStore{db: db}
}

// Put stores an email in the cache.
func (s *EmailStore) Put(email *CachedEmail) error {
	toJSON, _ := json.Marshal(email.To)
	ccJSON, _ := json.Marshal(email.CC)
	bccJSON, _ := json.Marshal(email.BCC)

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO emails (
			id, thread_id, folder_id, subject, snippet,
			from_name, from_email, to_json, cc_json, bcc_json,
			date, unread, starred, has_attachments,
			body_html, body_text, cached_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		email.ID, email.ThreadID, email.FolderID, email.Subject, email.Snippet,
		email.FromName, email.FromEmail, string(toJSON), string(ccJSON), string(bccJSON),
		email.Date.Unix(), boolToInt(email.Unread), boolToInt(email.Starred), boolToInt(email.HasAttachments),
		email.BodyHTML, email.BodyText, time.Now().Unix(),
	)
	return err
}

// PutBatch stores multiple emails in a transaction.
func (s *EmailStore) PutBatch(emails []*CachedEmail) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO emails (
			id, thread_id, folder_id, subject, snippet,
			from_name, from_email, to_json, cc_json, bcc_json,
			date, unread, starred, has_attachments,
			body_html, body_text, cached_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	// Get reusable buffer from pool to avoid allocations per email
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)

	now := time.Now().Unix()
	for _, email := range emails {
		// Marshal To field
		buf.Reset()
		_ = json.NewEncoder(buf).Encode(email.To)
		toJSON := strings.TrimSuffix(buf.String(), "\n")

		// Marshal CC field
		buf.Reset()
		_ = json.NewEncoder(buf).Encode(email.CC)
		ccJSON := strings.TrimSuffix(buf.String(), "\n")

		// Marshal BCC field
		buf.Reset()
		_ = json.NewEncoder(buf).Encode(email.BCC)
		bccJSON := strings.TrimSuffix(buf.String(), "\n")

		_, err = stmt.Exec(
			email.ID, email.ThreadID, email.FolderID, email.Subject, email.Snippet,
			email.FromName, email.FromEmail, toJSON, ccJSON, bccJSON,
			email.Date.Unix(), boolToInt(email.Unread), boolToInt(email.Starred), boolToInt(email.HasAttachments),
			email.BodyHTML, email.BodyText, now,
		)
		if err != nil {
			return fmt.Errorf("insert email %s: %w", email.ID, err)
		}
	}

	return tx.Commit()
}

// Get retrieves an email by ID.
func (s *EmailStore) Get(id string) (*CachedEmail, error) {
	row := s.db.QueryRow(`
		SELECT id, thread_id, folder_id, subject, snippet,
			from_name, from_email, to_json, cc_json, bcc_json,
			date, unread, starred, has_attachments,
			body_html, body_text, cached_at
		FROM emails WHERE id = ?
	`, id)

	return scanEmail(row)
}

// List retrieves emails with pagination and filtering.
func (s *EmailStore) List(opts ListOptions) ([]*CachedEmail, error) {
	// Use strings.Builder to avoid repeated string concatenations
	var query strings.Builder
	query.Grow(512) // Pre-allocate for typical query size
	query.WriteString(`
		SELECT id, thread_id, folder_id, subject, snippet,
			from_name, from_email, to_json, cc_json, bcc_json,
			date, unread, starred, has_attachments,
			body_html, body_text, cached_at
		FROM emails
		WHERE 1=1`)

	var args []any

	if opts.FolderID != "" {
		query.WriteString(" AND folder_id = ?")
		args = append(args, opts.FolderID)
	}
	if opts.ThreadID != "" {
		query.WriteString(" AND thread_id = ?")
		args = append(args, opts.ThreadID)
	}
	if opts.UnreadOnly {
		query.WriteString(" AND unread = 1")
	}
	if opts.StarredOnly {
		query.WriteString(" AND starred = 1")
	}
	if !opts.Since.IsZero() {
		query.WriteString(" AND date >= ?")
		args = append(args, opts.Since.Unix())
	}
	if !opts.Before.IsZero() {
		query.WriteString(" AND date < ?")
		args = append(args, opts.Before.Unix())
	}

	query.WriteString(" ORDER BY date DESC")

	if opts.Limit > 0 {
		fmt.Fprintf(&query, " LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		fmt.Fprintf(&query, " OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query emails: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Pre-allocate slice if limit is known
	var emails []*CachedEmail
	if opts.Limit > 0 {
		emails = make([]*CachedEmail, 0, opts.Limit)
	}

	for rows.Next() {
		email, err := scanEmailGeneric(rows)
		if err != nil {
			return nil, fmt.Errorf("scan email: %w", err)
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}

// ListOptions configures email listing.
type ListOptions struct {
	FolderID    string
	ThreadID    string
	UnreadOnly  bool
	StarredOnly bool
	Since       time.Time
	Before      time.Time
	Limit       int
	Offset      int
}

// Search performs full-text search on emails.
func (s *EmailStore) Search(query string, limit int) ([]*CachedEmail, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT e.id, e.thread_id, e.folder_id, e.subject, e.snippet,
			e.from_name, e.from_email, e.to_json, e.cc_json, e.bcc_json,
			e.date, e.unread, e.starred, e.has_attachments,
			e.body_html, e.body_text, e.cached_at
		FROM emails e
		WHERE e.rowid IN (
			SELECT rowid FROM emails_fts WHERE emails_fts MATCH ?
		)
		ORDER BY e.date DESC
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search emails: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Pre-allocate slice with expected capacity
	emails := make([]*CachedEmail, 0, limit)
	for rows.Next() {
		email, err := scanEmailGeneric(rows)
		if err != nil {
			return nil, fmt.Errorf("scan email: %w", err)
		}
		emails = append(emails, email)
	}

	return emails, rows.Err()
}

// Delete removes an email from the cache.
func (s *EmailStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM emails WHERE id = ?", id)
	return err
}

// UpdateFlags updates read/starred status.
func (s *EmailStore) UpdateFlags(id string, unread, starred *bool) error {
	if unread == nil && starred == nil {
		return nil
	}

	return s.UpdateMessage(id, unread, starred, nil)
}

// UpdateMessage updates cached message flags and folder placement.
func (s *EmailStore) UpdateMessage(id string, unread, starred *bool, folders []string) error {
	if unread == nil && starred == nil && folders == nil {
		return nil
	}

	// Use strings.Builder to avoid string concatenation in loop
	var query strings.Builder
	query.WriteString("UPDATE emails SET")
	var args []any
	needComma := false

	if unread != nil {
		query.WriteString(" unread = ?")
		args = append(args, boolToInt(*unread))
		needComma = true
	}
	if starred != nil {
		if needComma {
			query.WriteString(",")
		}
		query.WriteString(" starred = ?")
		args = append(args, boolToInt(*starred))
		needComma = true
	}
	if folders != nil {
		if needComma {
			query.WriteString(",")
		}
		folderID := ""
		if len(folders) > 0 {
			folderID = folders[0]
		}
		query.WriteString(" folder_id = ?")
		args = append(args, folderID)
	}

	query.WriteString(" WHERE id = ?")
	args = append(args, id)

	_, err := s.db.Exec(query.String(), args...)
	return err
}

// Count returns the number of cached emails.
func (s *EmailStore) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM emails").Scan(&count)
	return count, err
}

// CountUnread returns the number of unread emails.
func (s *EmailStore) CountUnread() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM emails WHERE unread = 1").Scan(&count)
	return count, err
}

// scanner is a common interface for sql.Row and sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanEmailGeneric scans a row into a CachedEmail using the scanner interface.
// This consolidates scanEmail and scanEmailRow to avoid code duplication.
func scanEmailGeneric(s scanner) (*CachedEmail, error) {
	var email CachedEmail
	var toJSON, ccJSON, bccJSON sql.NullString
	var dateUnix, cachedAtUnix int64
	var unread, starred, hasAttach int

	err := s.Scan(
		&email.ID, &email.ThreadID, &email.FolderID, &email.Subject, &email.Snippet,
		&email.FromName, &email.FromEmail, &toJSON, &ccJSON, &bccJSON,
		&dateUnix, &unread, &starred, &hasAttach,
		&email.BodyHTML, &email.BodyText, &cachedAtUnix,
	)
	if err != nil {
		return nil, err
	}

	email.Date = time.Unix(dateUnix, 0)
	email.CachedAt = time.Unix(cachedAtUnix, 0)
	email.Unread = unread == 1
	email.Starred = starred == 1
	email.HasAttachments = hasAttach == 1

	if toJSON.Valid {
		_ = json.Unmarshal([]byte(toJSON.String), &email.To)
	}
	if ccJSON.Valid {
		_ = json.Unmarshal([]byte(ccJSON.String), &email.CC)
	}
	if bccJSON.Valid {
		_ = json.Unmarshal([]byte(bccJSON.String), &email.BCC)
	}

	return &email, nil
}

// scanEmail scans a single sql.Row into a CachedEmail.
func scanEmail(row *sql.Row) (*CachedEmail, error) {
	return scanEmailGeneric(row)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
