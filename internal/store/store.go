// Package store persists the vault row and encrypted accounts in SQLite.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"modernc.org/sqlite"
)

const schemaVersion = 1

const schema = `
CREATE TABLE vault (
  id               INTEGER PRIMARY KEY CHECK (id = 1),
  kdf_params       TEXT NOT NULL,
  salt_pass        BLOB NOT NULL,
  wrapped_dek_pass BLOB NOT NULL,
  salt_rec         BLOB NOT NULL,
  wrapped_dek_rec  BLOB NOT NULL,
  created_at       INTEGER NOT NULL
);
CREATE TABLE accounts (
  id         TEXT PRIMARY KEY,
  blob       BLOB NOT NULL,
  lookup     BLOB NOT NULL UNIQUE,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
`

var (
	ErrNotInitialized     = errors.New("store: vault not initialized")
	ErrAlreadyInitialized = errors.New("store: vault already initialized")
	ErrNotFound           = errors.New("store: not found")
	ErrDuplicate          = errors.New("store: duplicate account")
)

// SQLite extended result codes for constraint violations.
const (
	sqliteConstraintPrimaryKey = 1555
	sqliteConstraintUnique     = 2067
)

// Store is the SQLite-backed persistence layer.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

// Open opens (creating if needed) the database at path and applies the schema.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("store: create dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("store: create file: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	dsn := (&url.URL{Scheme: "file", Opaque: escapePath(path), RawQuery: q.Encode()}).String()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	s := &Store{db: db, now: time.Now}
	if err := s.init(context.Background(), path); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// escapePath percent-encodes characters that are special in a sqlite file URI.
func escapePath(p string) string {
	u := url.URL{Path: p}
	return u.EscapedPath()
}

func (s *Store) init(ctx context.Context, path string) error {
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("store: chmod: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	defer tx.Rollback()

	var v int
	if err := tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil {
		return fmt.Errorf("store: read user_version: %w", err)
	}
	switch {
	case v > schemaVersion:
		return fmt.Errorf("store: database schema version %d is newer than supported %d", v, schemaVersion)
	case v == 0:
		if _, err := tx.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("store: apply schema: %w", err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
			return fmt.Errorf("store: set user_version: %w", err)
		}
	}
	return tx.Commit()
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// SetClock overrides the time source (for tests).
func (s *Store) SetClock(now func() time.Time) { s.now = now }

func isUnique(err error) bool {
	var se *sqlite.Error
	return errors.As(err, &se) && se.Code() == sqliteConstraintUnique
}

func isPrimaryKey(err error) bool {
	var se *sqlite.Error
	return errors.As(err, &se) && se.Code() == sqliteConstraintPrimaryKey
}
