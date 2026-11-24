package legacy

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

var fastKDF = vault.KDFParams{Alg: "argon2id", T: 1, M: 1024, P: 1}

const (
	testSecret = "0123456789abcdef0123456789ABCDEF"
	testMaster = "correct horse battery staple"
)

// Old schema, copied from main:backend/utilities/database/database.queries.go.
const (
	queryCreateUserTable = `
	CREATE TABLE IF NOT EXISTS user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		passphrase TEXT NOT NULL UNIQUE,
		recovery TEXT NOT NULL UNIQUE,
		validated BOOLEAN DEFAULT FALSE
	)
	`
	queryCreateAccountsTable = `
	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		identifier TEXT NOT NULL,
		url TEXT DEFAULT NULL,
		passphrase TEXT NOT NULL,
		notes TEXT DEFAULT NULL,
		strength TEXT DEFAULT NULL,
		UNIQUE(platform, identifier)
	)
	`
)

// oldEncrypt mirrors main:backend/utilities/encrypt/encrypt.go aesGCMEncrypt.
func oldEncrypt(t *testing.T, secret, data string) string {
	t.Helper()
	gcm := oldGCM(t, secret)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(data), nil))
}

// oldEncryptDeterministic mirrors aesGCMEncryptDeterministic.
func oldEncryptDeterministic(t *testing.T, secret, data string) string {
	t.Helper()
	gcm := oldGCM(t, secret)
	hash := sha256.Sum256(append([]byte(data), secret...))
	nonce := hash[:gcm.NonceSize()]
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(data), nil))
}

func oldGCM(t *testing.T, secret string) cipher.AEAD {
	t.Helper()
	block, err := aes.NewCipher([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	return gcm
}

// oldAccount is one fixture row; nil URL/Notes are stored as NULL.
type oldAccount struct {
	Platform, Identifier, Passphrase string
	URL, Notes                       *string
}

func strPtr(s string) *string { return &s }

type fixture struct {
	dir, from, env, to string
	ids                []int64
}

func newFixture(t *testing.T, validated bool, accounts []oldAccount) fixture {
	t.Helper()
	dir := t.TempDir()
	f := fixture{
		dir:  dir,
		from: filepath.Join(dir, "old", "passenger.db"),
		env:  filepath.Join(dir, "old", ".env"),
		to:   filepath.Join(dir, "new", "passenger.db"),
	}
	if err := os.MkdirAll(filepath.Dir(f.from), 0o700); err != nil {
		t.Fatal(err)
	}
	env := "# old config\nJWT_SECRET=jwt\nexport AES_GCM_SECRET=\"" + testSecret + "\"\r\nPORT=1234\nSALT=salt\n"
	if err := os.WriteFile(f.env, []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+f.from)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{queryCreateUserTable, queryCreateAccountsTable} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO user (passphrase, recovery, validated) VALUES (?, ?, ?)`,
		oldEncrypt(t, testSecret, testMaster), "old-recovery-key", validated); err != nil {
		t.Fatal(err)
	}
	for _, a := range accounts {
		var url, notes sql.NullString
		if a.URL != nil {
			url = sql.NullString{String: oldEncryptDeterministic(t, testSecret, *a.URL), Valid: true}
		}
		if a.Notes != nil {
			notes = sql.NullString{String: oldEncryptDeterministic(t, testSecret, *a.Notes), Valid: true}
		}
		res, err := db.Exec(`INSERT INTO accounts (platform, identifier, url, passphrase, notes, strength) VALUES (?, ?, ?, ?, ?, ?)`,
			oldEncryptDeterministic(t, testSecret, a.Platform),
			oldEncryptDeterministic(t, testSecret, a.Identifier),
			url,
			oldEncrypt(t, testSecret, a.Passphrase),
			notes,
			oldEncryptDeterministic(t, testSecret, "3"))
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		f.ids = append(f.ids, id)
	}
	return f
}

func (f fixture) options() Options {
	return Options{From: f.from, EnvFile: f.env, To: f.to, KDF: fastKDF, Out: &bytes.Buffer{}}
}

func threeAccounts() []oldAccount {
	return []oldAccount{
		{Platform: "GitHub", Identifier: "me@example.com", Passphrase: "gh-pass", URL: strPtr("https://github.com"), Notes: strPtr("work")},
		{Platform: "Bank", Identifier: "12345", Passphrase: "b@nk!", URL: nil, Notes: nil},
		{Platform: "Şirket İçi", Identifier: "çağrı.öğün", Passphrase: "ğüşıöç ĞÜŞİÖÇ", URL: strPtr(""), Notes: strPtr("Türkçe not: ılık su")},
	}
}

func wantRecord(a oldAccount) vault.Record {
	r := vault.Record{Platform: a.Platform, Identifier: a.Identifier, Passphrase: a.Passphrase}
	if a.URL != nil {
		r.URL = *a.URL
	}
	if a.Notes != nil {
		r.Notes = *a.Notes
	}
	return r
}

func fileHash(t *testing.T, path string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}

func assertAbsent(t *testing.T, path string) {
	t.Helper()
	for _, p := range []string{path, path + "-wal", path + "-shm", path + "-journal"} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s should not exist (err=%v)", p, err)
		}
	}
}

func assertNoSideFiles(t *testing.T, path string) {
	t.Helper()
	for _, p := range []string{path + "-wal", path + "-shm", path + "-journal"} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s should not exist (err=%v)", p, err)
		}
	}
}

func TestMigrateSuccess(t *testing.T) {
	accounts := threeAccounts()
	f := newFixture(t, true, accounts)
	before := fileHash(t, f.from)
	opts := f.options()
	out := &bytes.Buffer{}
	opts.Out = out

	res, err := Migrate(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Accounts != 3 {
		t.Fatalf("accounts = %d, want 3", res.Accounts)
	}
	if fileHash(t, f.from) != before {
		t.Fatal("old database changed")
	}
	assertNoSideFiles(t, f.from)

	s, err := store.Open(f.to)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	row, err := s.Vault(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if row.KDF != fastKDF {
		t.Fatalf("kdf = %+v", row.KDF)
	}
	dek, err := vault.Unwrap(testMaster, row.Pass, row.KDF, vault.AADPass)
	if err != nil {
		t.Fatalf("unwrap with master passphrase: %v", err)
	}
	recDEK, err := vault.Unwrap(vault.NormalizeRecoveryKey(res.RecoveryKey), row.Recovery, row.KDF, vault.AADRecovery)
	if err != nil {
		t.Fatalf("unwrap with new recovery key: %v", err)
	}
	if recDEK != dek {
		t.Fatal("recovery key unwraps a different DEK")
	}
	if _, err := vault.Unwrap("old-recovery-key", row.Recovery, row.KDF, vault.AADRecovery); err == nil {
		t.Fatal("old recovery key must not work")
	}
	got, err := s.ListAccounts(ctx, dek)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(accounts) {
		t.Fatalf("got %d accounts", len(got))
	}
	want := map[string]vault.Record{}
	for _, a := range accounts {
		want[a.Platform+"\x00"+a.Identifier] = wantRecord(a)
	}
	for _, a := range got {
		w, ok := want[a.Platform+"\x00"+a.Identifier]
		if !ok || w != a.Record {
			t.Fatalf("record mismatch for %q: got %+v want %+v", a.Platform, a.Record, w)
		}
		delete(want, a.Platform+"\x00"+a.Identifier)
	}
	if strings.Contains(out.String(), testMaster) || strings.Contains(out.String(), "gh-pass") {
		t.Fatal("output leaks master passphrase")
	}
}

func TestMigrateEmptyVault(t *testing.T) {
	f := newFixture(t, true, nil)
	res, err := Migrate(context.Background(), f.options())
	if err != nil {
		t.Fatal(err)
	}
	if res.Accounts != 0 || res.RecoveryKey == "" {
		t.Fatalf("result = %+v", res.Accounts)
	}
}

func TestMigrateTargetExists(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	if err := os.MkdirAll(filepath.Dir(f.to), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.to, []byte("precious"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(context.Background(), f.options()); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
	b, err := os.ReadFile(f.to)
	if err != nil || string(b) != "precious" {
		t.Fatalf("target touched: %q %v", b, err)
	}
	assertNoSideFiles(t, f.to)
}

func TestMigrateTargetSideFileExists(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	if err := os.MkdirAll(filepath.Dir(f.to), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.to+"-wal", []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(context.Background(), f.options()); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Lstat(f.to); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("target created")
	}
}

func TestMigrateNotValidated(t *testing.T) {
	f := newFixture(t, false, threeAccounts())
	if _, err := Migrate(context.Background(), f.options()); err == nil || !strings.Contains(err.Error(), "validated") {
		t.Fatalf("err = %v", err)
	}
	assertAbsent(t, f.to)
}

func TestMigrateCorruptedRow(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	db, err := sql.Open("sqlite", "file:"+f.from)
	if err != nil {
		t.Fatal(err)
	}
	bad := f.ids[1]
	if _, err := db.Exec(`UPDATE accounts SET passphrase = ? WHERE id = ?`, base64.StdEncoding.EncodeToString(make([]byte, 40)), bad); err != nil {
		t.Fatal(err)
	}
	db.Close()
	_, err = Migrate(context.Background(), f.options())
	if err == nil || !strings.Contains(err.Error(), "accounts.id 2") {
		t.Fatalf("err = %v", err)
	}
	assertAbsent(t, f.to)
}

func TestMigrateWrongSecret(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	if err := os.WriteFile(f.env, []byte("AES_GCM_SECRET='ffffffffffffffffffffffffffffffff'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(context.Background(), f.options()); err == nil || !strings.Contains(err.Error(), "AES_GCM_SECRET") {
		t.Fatalf("err = %v", err)
	}
	assertAbsent(t, f.to)
}

func TestMigrateSecretWrongLength(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	if err := os.WriteFile(f.env, []byte("AES_GCM_SECRET=short\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(context.Background(), f.options()); err == nil || !strings.Contains(err.Error(), "32") {
		t.Fatalf("err = %v", err)
	}
	assertAbsent(t, f.to)
}

func TestMigrateLookupCollision(t *testing.T) {
	f := newFixture(t, true, []oldAccount{
		{Platform: "GitHub", Identifier: "me", Passphrase: "a"},
		{Platform: "x", Identifier: "y", Passphrase: "b"},
		{Platform: "github ", Identifier: "me", Passphrase: "c"},
	})
	_, err := Migrate(context.Background(), f.options())
	if err == nil || !strings.Contains(err.Error(), "accounts.id 1") || !strings.Contains(err.Error(), "accounts.id 3") {
		t.Fatalf("err = %v", err)
	}
	assertAbsent(t, f.to)
}

func TestMigrateRefusesPendingOldJournal(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	if err := os.WriteFile(f.from+"-wal", []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(context.Background(), f.options()); err == nil || !strings.Contains(err.Error(), "-wal") {
		t.Fatalf("err = %v", err)
	}
	assertAbsent(t, f.to)
}

func TestOldDBIsReadOnly(t *testing.T) {
	f := newFixture(t, true, nil)
	db, err := openOld(f.from)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE x (a)`); err == nil {
		t.Fatal("old database must be opened read-only")
	}
}

func TestReadSecret(t *testing.T) {
	cases := map[string]string{
		"AES_GCM_SECRET=" + testSecret + "\n":                    testSecret,
		"# c\nAES_GCM_SECRET = '" + testSecret + "'\n":           testSecret,
		"export AES_GCM_SECRET=\"" + testSecret + "\"\r\n":       testSecret,
		"AES_GCM_SECRET=" + "0123456789#bcdef0123456789ABCDEF\n": "0123456789#bcdef0123456789ABCDEF",
	}
	for content, want := range cases {
		p := filepath.Join(t.TempDir(), ".env")
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := readSecret(p)
		if err != nil || string(got) != want {
			t.Fatalf("%q: got %q, %v", content, got, err)
		}
	}
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte("OTHER=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecret(p); err == nil {
		t.Fatal("missing key must error")
	}
}

func TestRunCLIMissingFrom(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := RunCLI([]string{"--env", "x"}, &stdout, &stderr); code != 2 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage") || !strings.Contains(stderr.String(), "--from") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunCLIOutput(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--from", f.from, "--env", f.env, "--to", f.to}, &stdout, &stderr, fastKDF)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	n := len(lines)
	if n < 5 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	want := []string{
		"Migrated 3 accounts.",
		"Your master passphrase is unchanged.",
		"",
		"Your old recovery key no longer works. Store the new one safely.",
		"Now delete the old database and the old .env (it holds AES_GCM_SECRET): " + f.from + ", " + f.env,
	}
	got := lines[n-5:]
	for i, w := range want {
		if i == 2 {
			if len(strings.ReplaceAll(got[i], "-", "")) != 52 {
				t.Fatalf("recovery key line = %q", got[i])
			}
			continue
		}
		if got[i] != w {
			t.Fatalf("line %d = %q, want %q", i, got[i], w)
		}
	}
	if strings.Contains(stdout.String()+stderr.String(), testMaster) || strings.Contains(stdout.String(), "gh-pass") {
		t.Fatal("output leaks secrets")
	}
	if strings.Count(stdout.String(), lines[n-3]) != 1 {
		t.Fatal("recovery key must be printed exactly once")
	}
}

func TestRunCLIFailure(t *testing.T) {
	f := newFixture(t, false, nil)
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"--from", f.from, "--env", f.env, "--to", f.to}, &stdout, &stderr, fastKDF); code != 1 {
		t.Fatalf("code = %d", code)
	}
	if stderr.Len() == 0 {
		t.Fatal("want error on stderr")
	}
}

// assertNothingLeft checks that a failed run left no target, no temp file and
// (when the run created it) no target directory.
func assertNothingLeft(t *testing.T, f fixture, dirCreated bool) {
	t.Helper()
	assertAbsent(t, f.to)
	dir := filepath.Dir(f.to)
	entries, err := os.ReadDir(dir)
	if dirCreated {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s should have been removed (entries %v, err %v)", dir, entries, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), filepath.Base(f.to)) {
			t.Fatalf("left behind: %s", e.Name())
		}
	}
}

func TestMigrateFailureAfterCreateVaultLeavesNothing(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	afterCreateVault = func() error { return errors.New("injected") }
	t.Cleanup(func() { afterCreateVault = nil })

	if _, err := Migrate(context.Background(), f.options()); err == nil || !strings.Contains(err.Error(), "injected") {
		t.Fatalf("err = %v", err)
	}
	assertNothingLeft(t, f, true)
}

func TestMigrateFailureKeepsExistingDirectory(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	if err := os.MkdirAll(filepath.Dir(f.to), 0o700); err != nil {
		t.Fatal(err)
	}
	afterCreateVault = func() error { return errors.New("injected") }
	t.Cleanup(func() { afterCreateVault = nil })

	if _, err := Migrate(context.Background(), f.options()); err == nil {
		t.Fatal("want error")
	}
	assertNothingLeft(t, f, false)
}

// A signal cancels the context; the run stops and cleans up.
func TestMigrateCanceledLeavesNothing(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	afterCreateVault = func() error { cancel(); return nil }
	t.Cleanup(func() { afterCreateVault = nil })

	if _, err := Migrate(ctx, f.options()); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	assertNothingLeft(t, f, true)
}

func TestMigrateSuccessLeavesNoTempFiles(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	if _, err := Migrate(context.Background(), f.options()); err != nil {
		t.Fatal(err)
	}
	assertNoSideFiles(t, f.to)
	entries, err := os.ReadDir(filepath.Dir(f.to))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(f.to) {
		t.Fatalf("directory holds %v, want only %s", entries, filepath.Base(f.to))
	}
	info, err := os.Stat(f.to)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
}

// A process killed mid-write (SIGKILL, power loss) runs no cleanup, so the
// target must not exist until the vault is complete and verified.
func TestMigrateTargetAppearsOnlyWhenComplete(t *testing.T) {
	f := newFixture(t, true, threeAccounts())
	var seen error
	afterCreateVault = func() error {
		_, seen = os.Lstat(f.to)
		return nil
	}
	t.Cleanup(func() { afterCreateVault = nil })

	if _, err := Migrate(context.Background(), f.options()); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(seen, os.ErrNotExist) {
		t.Fatalf("%s existed while the vault was being written (err=%v)", f.to, seen)
	}
}

func TestMigrateReportsWhatItCannotRemove(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs directory permissions that bind the current user")
	}
	f := newFixture(t, true, threeAccounts())
	dir := filepath.Dir(f.to)
	afterCreateVault = func() error {
		// Entries of a read-only directory cannot be removed.
		if err := os.Chmod(dir, 0o500); err != nil {
			return err
		}
		return errors.New("injected")
	}
	t.Cleanup(func() {
		afterCreateVault = nil
		os.Chmod(dir, 0o700)
	})
	opts := f.options()
	errOut := &bytes.Buffer{}
	opts.Err = errOut

	if _, err := Migrate(context.Background(), opts); err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(errOut.String(), "could not remove "+f.to+".tmp-") {
		t.Fatalf("stderr = %q", errOut.String())
	}
	if _, err := os.Lstat(f.to); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s must not exist (err=%v)", f.to, err)
	}
}
