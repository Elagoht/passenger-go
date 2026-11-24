package legacy

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

// Options configures one migration run.
type Options struct {
	From, EnvFile, To string
	// KDF is the Argon2id cost for the new wraps; zero means vault.DefaultKDF.
	KDF vault.KDFParams
	// Out receives progress lines (never secrets); nil discards them.
	Out io.Writer
	// Err receives the paths a failed run could not clean up; nil discards them.
	Err io.Writer
}

// Result summarizes a successful migration.
type Result struct {
	Accounts    int
	RecoveryKey string
}

// afterCreateVault, when set by a test, runs between writing the vault row
// and writing the accounts, to inject a failure or a cancellation there.
var afterCreateVault func() error

// oldRecord is one decrypted old account with its old id.
type oldRecord struct {
	id int64
	vault.Record
}

// Migrate copies the old database into a new vault at o.To. Nothing is
// written to o.From. o.To appears only once the new vault is complete and
// verified; on any error, including ctx ending, nothing is left behind.
func Migrate(ctx context.Context, o Options) (Result, error) {
	if o.From == "" || o.EnvFile == "" || o.To == "" {
		return Result{}, errors.New("from, env and to are required")
	}
	if o.KDF == (vault.KDFParams{}) {
		o.KDF = vault.DefaultKDF
	}
	out := o.Out
	if out == nil {
		out = io.Discard
	}
	errOut := o.Err
	if errOut == nil {
		errOut = io.Discard
	}

	secret, err := readSecret(o.EnvFile)
	if err != nil {
		return Result{}, err
	}
	defer clear(secret)
	if err := checkTargetAbsent(o.To); err != nil {
		return Result{}, err
	}
	if err := checkOldSideFiles(o.From); err != nil {
		return Result{}, err
	}

	master, records, err := readOld(ctx, o.From, secret)
	if err != nil {
		return Result{}, err
	}
	fmt.Fprintf(out, "Read %d accounts from the old database.\n", len(records))

	dek, err := vault.NewDEK()
	if err != nil {
		return Result{}, err
	}
	defer dek.Wipe()
	if err := checkCollisions(dek, records); err != nil {
		return Result{}, err
	}
	recoveryKey, err := vault.NewRecoveryKey()
	if err != nil {
		return Result{}, err
	}
	passWrap, err := vault.Wrap(master, dek, o.KDF, vault.AADPass)
	if err != nil {
		return Result{}, err
	}
	recWrap, err := vault.Wrap(vault.NormalizeRecoveryKey(recoveryKey), dek, o.KDF, vault.AADRecovery)
	if err != nil {
		return Result{}, err
	}

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	// Everything is written to a temp file next to the target and renamed into
	// place only once verified, so an interrupted run (even one killed with no
	// chance to clean up) never leaves a partial vault at o.To.
	tmp, err := createTemp(o.To)
	if err != nil {
		return Result{}, err
	}
	ok := false
	defer func() {
		if !ok {
			tmp.remove(errOut)
		}
	}()

	if err := write(ctx, tmp.path, store.VaultRow{KDF: o.KDF, Pass: passWrap, Recovery: recWrap}, dek, records); err != nil {
		return Result{}, err
	}
	if err := verify(ctx, tmp.path, master, recoveryKey, dek, records); err != nil {
		return Result{}, fmt.Errorf("verification failed: %w", err)
	}
	if err := tmp.commit(ctx, o.To); err != nil {
		return Result{}, err
	}
	ok = true
	fmt.Fprintln(out, "Verified the new vault.")
	return Result{Accounts: len(records), RecoveryKey: recoveryKey}, nil
}

func sidePaths(path string) []string {
	return []string{path, path + "-wal", path + "-shm", path + "-journal"}
}

func checkTargetAbsent(path string) error {
	for _, p := range sidePaths(path) {
		if _, err := os.Lstat(p); err == nil {
			return fmt.Errorf("%s already exists; refusing to overwrite", p)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check %s: %w", p, err)
		}
	}
	return nil
}

// tempTarget is the file a run writes the new vault to before renaming it to
// the target.
type tempTarget struct {
	path string
	// dir is the target's directory and dirCreated whether this run made it.
	dir        string
	dirCreated bool
}

// createTemp creates an empty, exclusive, 0600 temp file named after target
// in target's directory, creating that directory (0700) when missing.
func createTemp(target string) (*tempTarget, error) {
	t := &tempTarget{dir: filepath.Dir(target)}
	if _, err := os.Lstat(t.dir); errors.Is(err, os.ErrNotExist) {
		t.dirCreated = true
	} else if err != nil {
		return nil, fmt.Errorf("check target directory: %w", err)
	}
	if err := os.MkdirAll(t.dir, 0o700); err != nil {
		return nil, fmt.Errorf("create target directory: %w", err)
	}
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return nil, err
	}
	t.path = target + ".tmp-" + hex.EncodeToString(suffix)
	f, err := os.OpenFile(t.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.removeDir(io.Discard)
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		t.remove(io.Discard)
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	return t, nil
}

// commit makes the closed, verified temp file durable and renames it to
// target, refusing when target appeared meanwhile.
func (t *tempTarget) commit(ctx context.Context, target string) error {
	// The store was closed, which checkpoints the WAL into the main file; a
	// side file left over would hold data the rename would leave behind.
	for _, p := range sidePaths(t.path)[1:] {
		if _, err := os.Lstat(p); err == nil {
			return fmt.Errorf("%s was left after closing the new vault", p)
		}
	}
	f, err := os.OpenFile(t.path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("sync new vault: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// os.Rename replaces an existing file; look once more right before it.
	if err := checkTargetAbsent(target); err != nil {
		return err
	}
	if err := os.Rename(t.path, target); err != nil {
		return fmt.Errorf("move new vault into place: %w", err)
	}
	// Best effort: persist the rename. Directories cannot be synced everywhere.
	if d, err := os.Open(t.dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}

// remove deletes the temp file and its SQLite side files, then the directory
// if this run created it, reporting on w whatever could not be removed.
func (t *tempTarget) remove(w io.Writer) {
	for _, p := range sidePaths(t.path) {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(w, "migrate: could not remove %s: %v\n", p, err)
		}
	}
	t.removeDir(w)
}

func (t *tempTarget) removeDir(w io.Writer) {
	if !t.dirCreated {
		return
	}
	if err := os.Remove(t.dir); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(w, "migrate: could not remove %s: %v\n", t.dir, err)
	}
}

// readOld returns the decrypted master passphrase and every decrypted account.
func readOld(ctx context.Context, path string, secret []byte) (string, []oldRecord, error) {
	db, err := openOld(path)
	if err != nil {
		return "", nil, err
	}
	defer db.Close()

	var users int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user`).Scan(&users); err != nil {
		return "", nil, fmt.Errorf("read old user: %w", err)
	}
	if users != 1 {
		return "", nil, fmt.Errorf("old database has %d user rows, want exactly 1", users)
	}
	var (
		encMaster string
		validated int64
	)
	if err := db.QueryRowContext(ctx, `SELECT passphrase, COALESCE(validated, 0) FROM user`).
		Scan(&encMaster, &validated); err != nil {
		return "", nil, fmt.Errorf("read old user: %w", err)
	}
	if validated == 0 {
		return "", nil, errors.New("old user is not validated; finish registration in the old app first")
	}
	master, err := oldDecrypt(secret, encMaster)
	if err != nil {
		return "", nil, fmt.Errorf("cannot decrypt the old master passphrase (wrong %s?): %w", secretKey, err)
	}
	if master == "" {
		return "", nil, errors.New("old master passphrase is empty")
	}

	rows, err := db.QueryContext(ctx,
		`SELECT id, platform, identifier, passphrase, url, notes FROM accounts ORDER BY id`)
	if err != nil {
		return "", nil, fmt.Errorf("read old accounts: %w", err)
	}
	defer rows.Close()
	records := []oldRecord{}
	for rows.Next() {
		var (
			id                    int64
			platform, ident, pass string
			urlCol, notesCol      sql.NullString
		)
		if err := rows.Scan(&id, &platform, &ident, &pass, &urlCol, &notesCol); err != nil {
			return "", nil, fmt.Errorf("read old accounts: %w", err)
		}
		r, err := decryptRow(secret, platform, ident, pass, urlCol, notesCol)
		if err != nil {
			return "", nil, fmt.Errorf("old accounts.id %d cannot be decrypted (%w); nothing was migrated", id, err)
		}
		records = append(records, oldRecord{id: id, Record: r})
	}
	if err := rows.Err(); err != nil {
		return "", nil, fmt.Errorf("read old accounts: %w", err)
	}
	return master, records, nil
}

func decryptRow(secret []byte, platform, ident, pass string, urlCol, notesCol sql.NullString) (vault.Record, error) {
	var (
		r   vault.Record
		err error
	)
	if r.Platform, err = oldDecrypt(secret, platform); err != nil {
		return vault.Record{}, fmt.Errorf("platform: %w", err)
	}
	if r.Identifier, err = oldDecrypt(secret, ident); err != nil {
		return vault.Record{}, fmt.Errorf("identifier: %w", err)
	}
	if r.Passphrase, err = oldDecrypt(secret, pass); err != nil {
		return vault.Record{}, fmt.Errorf("passphrase: %w", err)
	}
	if r.URL, err = decryptOptional(secret, urlCol); err != nil {
		return vault.Record{}, fmt.Errorf("url: %w", err)
	}
	if r.Notes, err = decryptOptional(secret, notesCol); err != nil {
		return vault.Record{}, fmt.Errorf("notes: %w", err)
	}
	return r, nil
}

// decryptOptional maps NULL and an empty column to "".
func decryptOptional(secret []byte, col sql.NullString) (string, error) {
	if !col.Valid || col.String == "" {
		return "", nil
	}
	return oldDecrypt(secret, col.String)
}

// checkCollisions aborts when two old accounts map to the same new lookup
// (e.g. platforms that differ only in case or surrounding spaces).
func checkCollisions(dek vault.DEK, records []oldRecord) error {
	seen := make(map[string]int64, len(records))
	for _, r := range records {
		k := string(vault.Lookup(dek, r.Platform, r.Identifier))
		if prev, dup := seen[k]; dup {
			return fmt.Errorf("old accounts.id %d and accounts.id %d have the same platform and identifier in the new vault (platform is compared case-insensitively); rename one in the old app and retry", prev, r.id)
		}
		seen[k] = r.id
	}
	return nil
}

func write(ctx context.Context, path string, row store.VaultRow, dek vault.DEK, records []oldRecord) error {
	s, err := store.Open(path)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := s.CreateVault(ctx, row); err != nil {
		return err
	}
	if afterCreateVault != nil {
		if err := afterCreateVault(); err != nil {
			return err
		}
	}
	rs := make([]vault.Record, len(records))
	for i, r := range records {
		rs[i] = r.Record
	}
	if err := s.InsertAccountsTx(ctx, dek, rs); err != nil {
		return fmt.Errorf("write accounts: %w", err)
	}
	return s.Close()
}

// verify reopens the target and checks that the master passphrase and the new
// recovery key both open it and that every old record is present unchanged.
func verify(ctx context.Context, path, master, recoveryKey string, dek vault.DEK, records []oldRecord) error {
	s, err := store.Open(path)
	if err != nil {
		return err
	}
	defer s.Close()
	row, err := s.Vault(ctx)
	if err != nil {
		return err
	}
	got, err := vault.Unwrap(master, row.Pass, row.KDF, vault.AADPass)
	if err != nil {
		return fmt.Errorf("master passphrase does not open the new vault: %w", err)
	}
	defer got.Wipe()
	if got != dek {
		return errors.New("master passphrase opens a different key")
	}
	rec, err := vault.Unwrap(vault.NormalizeRecoveryKey(recoveryKey), row.Recovery, row.KDF, vault.AADRecovery)
	if err != nil {
		return fmt.Errorf("new recovery key does not open the new vault: %w", err)
	}
	defer rec.Wipe()
	if rec != dek {
		return errors.New("new recovery key opens a different key")
	}
	accounts, err := s.ListAccounts(ctx, got)
	if err != nil {
		return err
	}
	if len(accounts) != len(records) {
		return fmt.Errorf("new vault has %d accounts, old database has %d", len(accounts), len(records))
	}
	want := make(map[string]oldRecord, len(records))
	for _, r := range records {
		want[r.Platform+"\x00"+r.Identifier] = r
	}
	for _, a := range accounts {
		k := a.Platform + "\x00" + a.Identifier
		w, ok := want[k]
		if !ok {
			return errors.New("new vault has an account that is not in the old database")
		}
		if w.Record != a.Record {
			return fmt.Errorf("old accounts.id %d differs in the new vault", w.id)
		}
		delete(want, k)
	}
	if len(want) != 0 {
		return fmt.Errorf("%d old accounts are missing from the new vault", len(want))
	}
	return nil
}
