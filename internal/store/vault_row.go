package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"passenger-go/internal/vault"
)

// VaultRow is the single vault record: KDF parameters and the wrapped DEK.
type VaultRow struct {
	KDF       vault.KDFParams
	Pass      vault.Wrapped
	Recovery  vault.Wrapped
	CreatedAt time.Time
}

// Initialized reports whether the vault row exists.
func (s *Store) Initialized(ctx context.Context) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vault`).Scan(&n); err != nil {
		return false, fmt.Errorf("store: count vault: %w", err)
	}
	return n > 0, nil
}

// Vault returns the vault row, or ErrNotInitialized.
func (s *Store) Vault(ctx context.Context) (VaultRow, error) {
	var (
		row     VaultRow
		kdf     string
		created int64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT kdf_params, salt_pass, wrapped_dek_pass, salt_rec, wrapped_dek_rec, created_at FROM vault WHERE id = 1`).
		Scan(&kdf, &row.Pass.Salt, &row.Pass.Box, &row.Recovery.Salt, &row.Recovery.Box, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return VaultRow{}, ErrNotInitialized
	}
	if err != nil {
		return VaultRow{}, fmt.Errorf("store: read vault: %w", err)
	}
	if err := json.Unmarshal([]byte(kdf), &row.KDF); err != nil {
		return VaultRow{}, fmt.Errorf("store: decode kdf params: %w", err)
	}
	row.CreatedAt = time.Unix(created, 0)
	return row, nil
}

// CreateVault stores the vault row, or returns ErrAlreadyInitialized.
func (s *Store) CreateVault(ctx context.Context, v VaultRow) error {
	kdf, err := json.Marshal(v.KDF)
	if err != nil {
		return fmt.Errorf("store: encode kdf params: %w", err)
	}
	created := v.CreatedAt
	if created.IsZero() {
		created = s.now()
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO vault (id, kdf_params, salt_pass, wrapped_dek_pass, salt_rec, wrapped_dek_rec, created_at)
		 VALUES (1, ?, ?, ?, ?, ?, ?)`,
		string(kdf), v.Pass.Salt, v.Pass.Box, v.Recovery.Salt, v.Recovery.Box, created.Unix())
	if isPrimaryKey(err) || isUnique(err) {
		return ErrAlreadyInitialized
	}
	if err != nil {
		return fmt.Errorf("store: create vault: %w", err)
	}
	return nil
}

// UpdatePassWrap replaces the passphrase-wrapped DEK.
func (s *Store) UpdatePassWrap(ctx context.Context, w vault.Wrapped) error {
	return s.updateWrap(ctx, `UPDATE vault SET salt_pass = ?, wrapped_dek_pass = ? WHERE id = 1`, w)
}

// UpdateRecoveryWrap replaces the recovery-key-wrapped DEK.
func (s *Store) UpdateRecoveryWrap(ctx context.Context, w vault.Wrapped) error {
	return s.updateWrap(ctx, `UPDATE vault SET salt_rec = ?, wrapped_dek_rec = ? WHERE id = 1`, w)
}

func (s *Store) updateWrap(ctx context.Context, query string, w vault.Wrapped) error {
	res, err := s.db.ExecContext(ctx, query, w.Salt, w.Box)
	if err != nil {
		return fmt.Errorf("store: update wrap: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("store: update wrap: %w", err)
	} else if n == 0 {
		return ErrNotInitialized
	}
	return nil
}
