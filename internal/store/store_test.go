package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"passenger-go/internal/vault"
)

var ctx = context.Background()

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data", "p.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

func testDEK(b byte) vault.DEK {
	var d vault.DEK
	for i := range d {
		d[i] = b
	}
	return d
}

func rec(p, i string) vault.Record {
	return vault.Record{Platform: p, Identifier: i, Passphrase: "pw", URL: "u", Notes: "n"}
}

func TestOpenModesAndReopen(t *testing.T) {
	s, path := newStore(t)
	for p, want := range map[string]os.FileMode{path: 0o600, filepath.Dir(path): 0o700} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != want {
			t.Errorf("%s mode = %o, want %o", p, fi.Mode().Perm(), want)
		}
	}
	var v int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil || v != 1 {
		t.Fatalf("user_version = %d, %v", v, err)
	}
	if err := s.CreateVault(ctx, VaultRow{Pass: vault.Wrapped{Salt: []byte{1}, Box: []byte{2}}, Recovery: vault.Wrapped{Salt: []byte{3}, Box: []byte{4}}, CreatedAt: time.Unix(5, 0)}); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if ok, _ := s2.Initialized(ctx); !ok {
		t.Fatal("data lost on reopen")
	}
}

func TestOpenRefusesNewerVersion(t *testing.T) {
	s, path := newStore(t)
	if _, err := s.db.Exec(`PRAGMA user_version = 2`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if _, err := Open(path); err == nil {
		t.Fatal("expected error for newer schema")
	}
}

func TestVaultLifecycle(t *testing.T) {
	s, _ := newStore(t)
	if ok, err := s.Initialized(ctx); err != nil || ok {
		t.Fatalf("Initialized = %v, %v", ok, err)
	}
	if _, err := s.Vault(ctx); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("Vault err = %v", err)
	}
	row := VaultRow{
		KDF:       vault.KDFParams{Alg: "argon2id", T: 3, M: 65536, P: 4},
		Pass:      vault.Wrapped{Salt: []byte("sp"), Box: []byte("bp")},
		Recovery:  vault.Wrapped{Salt: []byte("sr"), Box: []byte("br")},
		CreatedAt: time.Unix(1000, 0),
	}
	if err := s.CreateVault(ctx, row); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.Initialized(ctx); !ok {
		t.Fatal("want initialized")
	}
	if err := s.CreateVault(ctx, row); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("second create err = %v", err)
	}
	got, err := s.Vault(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.KDF != row.KDF || !got.CreatedAt.Equal(row.CreatedAt) ||
		!bytes.Equal(got.Pass.Box, []byte("bp")) || !bytes.Equal(got.Recovery.Salt, []byte("sr")) {
		t.Fatalf("got %+v", got)
	}
	if err := s.UpdatePassWrap(ctx, vault.Wrapped{Salt: []byte("a"), Box: []byte("b")}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateRecoveryWrap(ctx, vault.Wrapped{Salt: []byte("c"), Box: []byte("d")}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Vault(ctx)
	if string(got.Pass.Salt) != "a" || string(got.Pass.Box) != "b" ||
		string(got.Recovery.Salt) != "c" || string(got.Recovery.Box) != "d" {
		t.Fatalf("wraps not updated: %+v", got)
	}
}

func TestUpdateWrapNotInitialized(t *testing.T) {
	s, _ := newStore(t)
	if err := s.UpdatePassWrap(ctx, vault.Wrapped{}); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("err = %v", err)
	}
}

func TestAccountRoundTrip(t *testing.T) {
	s, _ := newStore(t)
	dek := testDEK(1)
	t0 := time.Unix(1000, 0)
	s.SetClock(func() time.Time { return t0 })

	b, err := s.CreateAccount(ctx, dek, rec("beta", "x"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAccount(ctx, dek, rec("Alpha", "z")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAccount(ctx, dek, rec("alpha", "y")); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListAccounts(ctx, dek)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, a := range list {
		order = append(order, a.Platform+"/"+a.Identifier)
	}
	want := []string{"alpha/y", "Alpha/z", "beta/x"}
	if len(order) != 3 || order[0] != want[0] || order[1] != want[1] || order[2] != want[2] {
		t.Fatalf("order = %v", order)
	}

	got, err := s.GetAccount(ctx, dek, b.ID)
	if err != nil || got.Record != b.Record || !got.CreatedAt.Equal(t0) {
		t.Fatalf("get = %+v, %v", got, err)
	}

	t1 := t0.Add(time.Hour)
	s.SetClock(func() time.Time { return t1 })
	upd := rec("beta", "x2")
	upd.Passphrase = "new"
	if err := s.UpdateAccount(ctx, dek, b.ID, upd); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetAccount(ctx, dek, b.ID)
	if got.Record != upd || !got.CreatedAt.Equal(t0) || !got.UpdatedAt.Equal(t1) {
		t.Fatalf("after update = %+v", got)
	}
	// updating a row to its own identity is not a duplicate
	if err := s.UpdateAccount(ctx, dek, b.ID, upd); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteAccount(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAccount(ctx, dek, b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete = %v", err)
	}
	if err := s.DeleteAccount(ctx, b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete twice = %v", err)
	}
	if err := s.UpdateAccount(ctx, dek, b.ID, upd); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing = %v", err)
	}
}

func TestDuplicates(t *testing.T) {
	s, _ := newStore(t)
	dek := testDEK(2)
	if _, err := s.CreateAccount(ctx, dek, rec("GitHub", "me")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAccount(ctx, dek, rec(" github", "me")); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("create dup = %v", err)
	}
	other, err := s.CreateAccount(ctx, dek, rec("GitLab", "me"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateAccount(ctx, dek, other.ID, rec(" github", "me")); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("update dup = %v", err)
	}
	got, _ := s.GetAccount(ctx, dek, other.ID)
	if got.Platform != "GitLab" {
		t.Fatalf("row changed despite error: %+v", got)
	}
}

func TestWrongDEK(t *testing.T) {
	s, _ := newStore(t)
	a, err := s.CreateAccount(ctx, testDEK(1), rec("p", "i"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.GetAccount(ctx, testDEK(9), a.ID); err == nil {
		t.Fatalf("expected error, got %+v", got)
	}
	if _, err := s.ListAccounts(ctx, testDEK(9)); err == nil {
		t.Fatal("list must fail, not skip rows")
	}
}

func TestInsertAccountsTxAtomic(t *testing.T) {
	s, _ := newStore(t)
	dek := testDEK(3)
	err := s.InsertAccountsTx(ctx, dek, []vault.Record{rec("a", "1"), rec("b", "2"), rec("A", "1")})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("err = %v", err)
	}
	if l, _ := s.ListAccounts(ctx, dek); len(l) != 0 {
		t.Fatalf("rows written: %d", len(l))
	}
	if err := s.InsertAccountsTx(ctx, dek, []vault.Record{rec("a", "1"), rec("b", "2")}); err != nil {
		t.Fatal(err)
	}
	if l, _ := s.ListAccounts(ctx, dek); len(l) != 2 {
		t.Fatalf("rows: %d", len(l))
	}
}

func TestNoPlaintextOnDisk(t *testing.T) {
	s, path := newStore(t)
	const marker = "hunter2-unique-marker"
	r := rec("plat-marker-x", "ident-marker-y")
	r.Passphrase = marker
	if _, err := s.CreateAccount(ctx, testDEK(4), r); err != nil {
		t.Fatal(err)
	}
	s.Close()
	for _, suffix := range []string{"", "-wal"} {
		data, err := os.ReadFile(path + suffix)
		if suffix == "" && err != nil {
			t.Fatalf("read the database file: %v", err)
		}
		if err != nil { // the WAL is gone after a clean close
			continue
		}
		for _, m := range []string{marker, "plat-marker-x", "ident-marker-y"} {
			if bytes.Contains(data, []byte(m)) {
				t.Errorf("plaintext %q found in %s", m, path+suffix)
			}
		}
	}
}

func TestUpdateWraps(t *testing.T) {
	s, _ := newStore(t)
	pass := vault.Wrapped{Salt: []byte("a"), Box: []byte("b")}
	recov := vault.Wrapped{Salt: []byte("c"), Box: []byte("d")}
	if err := s.UpdateWraps(ctx, pass, recov); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("uninitialized err = %v", err)
	}
	row := VaultRow{
		KDF:      vault.KDFParams{Alg: "argon2id", T: 3, M: 65536, P: 4},
		Pass:     vault.Wrapped{Salt: []byte("sp"), Box: []byte("bp")},
		Recovery: vault.Wrapped{Salt: []byte("sr"), Box: []byte("br")},
	}
	if err := s.CreateVault(ctx, row); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateWraps(ctx, pass, recov); err != nil {
		t.Fatal(err)
	}
	got, err := s.Vault(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Pass.Salt) != "a" || string(got.Pass.Box) != "b" ||
		string(got.Recovery.Salt) != "c" || string(got.Recovery.Box) != "d" {
		t.Fatalf("wraps not updated: %+v", got)
	}
	if got.KDF != row.KDF {
		t.Fatalf("kdf changed: %+v", got.KDF)
	}
}
