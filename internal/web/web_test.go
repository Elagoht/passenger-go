package web

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/ratelimit"
	"passenger-go/internal/session"
	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

var fastKDF = vault.KDFParams{Alg: "argon2id", T: 1, M: 1024, P: 1}

func newDeps(t *testing.T) *Deps {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	now := func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	return &Deps{
		Store:    st,
		Sessions: session.NewManager(15*time.Minute, 12*time.Hour, now),
		Limiter:  ratelimit.New(now),
		KDF:      fastKDF,
		Now:      now,
	}
}

func initVault(t *testing.T, d *Deps) vault.DEK {
	t.Helper()
	dek, err := vault.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	pass, err := vault.Wrap("pass", dek, d.KDF, vault.AADPass)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := vault.Wrap("rec", dek, d.KDF, vault.AADRecovery)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Store.CreateVault(context.Background(), store.VaultRow{KDF: d.KDF, Pass: pass, Recovery: rec}); err != nil {
		t.Fatal(err)
	}
	return dek
}

func TestSessionFromNil(t *testing.T) {
	if s := SessionFrom(nil); s != nil {
		t.Fatal("SessionFrom(nil) != nil")
	}
	if s := SessionFrom(&collage.RenderContext{}); s != nil {
		t.Fatal("SessionFrom(empty) != nil")
	}
}
