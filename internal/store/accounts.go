package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"passenger-go/internal/vault"
)

// Account is a decrypted account with its storage metadata.
type Account struct {
	ID string
	vault.Record
	CreatedAt, UpdatedAt time.Time
}

func (s *Store) insert(ctx context.Context, tx *sql.Tx, dek vault.DEK, r vault.Record) (Account, error) {
	id, err := vault.NewAccountID()
	if err != nil {
		return Account{}, err
	}
	blob, err := vault.Seal(dek, id, r)
	if err != nil {
		return Account{}, err
	}
	now := s.now()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO accounts (id, blob, lookup, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, blob, vault.Lookup(dek, r.Platform, r.Identifier), now.Unix(), now.Unix())
	if isUnique(err) {
		return Account{}, ErrDuplicate
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: insert account: %w", err)
	}
	t := time.Unix(now.Unix(), 0)
	return Account{ID: id, Record: r, CreatedAt: t, UpdatedAt: t}, nil
}

func (s *Store) decode(dek vault.DEK, id string, blob []byte, created, updated int64) (Account, error) {
	r, err := vault.Open(dek, id, blob)
	if err != nil {
		return Account{}, fmt.Errorf("store: decrypt account %s: %w", id, err)
	}
	return Account{ID: id, Record: r, CreatedAt: time.Unix(created, 0), UpdatedAt: time.Unix(updated, 0)}, nil
}

// CreateAccount encrypts and stores r. ErrDuplicate if (platform, identifier) exists.
func (s *Store) CreateAccount(ctx context.Context, dek vault.DEK, r vault.Record) (Account, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Account{}, fmt.Errorf("store: begin: %w", err)
	}
	defer tx.Rollback()
	a, err := s.insert(ctx, tx, dek, r)
	if err != nil {
		return Account{}, err
	}
	if err := tx.Commit(); err != nil {
		return Account{}, fmt.Errorf("store: commit: %w", err)
	}
	return a, nil
}

// InsertAccountsTx stores all records or none.
func (s *Store) InsertAccountsTx(ctx context.Context, dek vault.DEK, rs []vault.Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	defer tx.Rollback()
	for _, r := range rs {
		if _, err := s.insert(ctx, tx, dek, r); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetAccount returns and decrypts one account, or ErrNotFound.
func (s *Store) GetAccount(ctx context.Context, dek vault.DEK, id string) (Account, error) {
	var (
		blob             []byte
		created, updated int64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT blob, created_at, updated_at FROM accounts WHERE id = ?`, id).Scan(&blob, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("store: get account: %w", err)
	}
	return s.decode(dek, id, blob, created, updated)
}

// ListAccounts decrypts every account, sorted by lowercase platform then identifier.
// A row that fails to decrypt aborts with an error.
func (s *Store) ListAccounts(ctx context.Context, dek vault.DEK) ([]Account, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, blob, created_at, updated_at FROM accounts`)
	if err != nil {
		return nil, fmt.Errorf("store: list accounts: %w", err)
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		var (
			id               string
			blob             []byte
			created, updated int64
		)
		if err := rows.Scan(&id, &blob, &created, &updated); err != nil {
			return nil, fmt.Errorf("store: scan account: %w", err)
		}
		a, err := s.decode(dek, id, blob, created, updated)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list accounts: %w", err)
	}
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := strings.ToLower(out[i].Platform), strings.ToLower(out[j].Platform)
		if pi != pj {
			return pi < pj
		}
		return out[i].Identifier < out[j].Identifier
	})
	return out, nil
}

// UpdateAccount replaces the record of account id, keeping its created_at.
func (s *Store) UpdateAccount(ctx context.Context, dek vault.DEK, id string, r vault.Record) error {
	blob, err := vault.Seal(dek, id, r)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE accounts SET blob = ?, lookup = ?, updated_at = ? WHERE id = ?`,
		blob, vault.Lookup(dek, r.Platform, r.Identifier), s.now().Unix(), id)
	if isUnique(err) {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("store: update account: %w", err)
	}
	return affectedOrNotFound(res)
}

// DeleteAccount removes account id, or returns ErrNotFound.
func (s *Store) DeleteAccount(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete account: %w", err)
	}
	return affectedOrNotFound(res)
}

func affectedOrNotFound(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
