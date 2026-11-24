package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/ratelimit"
	"passenger-go/internal/session"
	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

var fastKDF = vault.KDFParams{Alg: "argon2id", T: 1, M: 1024, P: 1}

func TestSafeNext(t *testing.T) {
	for raw, want := range map[string]string{
		"/accounts/1?x=1":  "/accounts/1?x=1",
		"/":                "/",
		"//evil.com":       "/",
		"/\\evil.com":      "/",
		"https://evil.com": "/",
		"/%2F%2Fevil.com":  "/",
		"/%5Cevil.com":     "/",
		"/a\\b":            "/",
		"":                 "/",
		"accounts":         "/",
		"/%zz":             "/",
		"/\t/evil.com":     "/",
		"/%09/evil.com":    "/",
		"/%0A/evil.com":    "/",
	} {
		if got := SafeNext(raw); got != want {
			t.Errorf("SafeNext(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	const csp = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
		"frame-ancestors 'none'; base-uri 'none'; form-action 'self'"
	// The inner handler sets its own Cache-Control, as collage does for guarded
	// pages and static files; ours must still win.
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-cache")
		w.Write([]byte("ok"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	for name, want := range map[string]string{
		"Content-Security-Policy": csp,
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "no-store",
		"X-Content-Type-Options":  "nosniff",
	} {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

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

// runGuard sends a request through the session middleware (which sets the
// session and the stale-cookie flag) and returns the guard's decision.
func runGuard(t *testing.T, d *Deps, g collage.GuardFunc, target, cookie string) *collage.GuardDecision {
	t.Helper()
	var dec *collage.GuardDecision
	var gerr error
	h := d.Sessions.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dec, gerr = g(r.Context(), r)
	}))
	r := httptest.NewRequest("GET", target, nil)
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: session.CookieName, Value: cookie})
	}
	h.ServeHTTP(httptest.NewRecorder(), r)
	if gerr != nil {
		t.Fatalf("guard error: %v", gerr)
	}
	return dec
}

func wantRedirect(t *testing.T, dec *collage.GuardDecision, loc string) {
	t.Helper()
	if dec == nil {
		t.Fatalf("decision = nil (allowed), want 303 %s", loc)
	}
	if dec.Status != http.StatusSeeOther || dec.Location != loc {
		t.Fatalf("decision = %d %q, want 303 %q", dec.Status, dec.Location, loc)
	}
}

func TestRequireSession(t *testing.T) {
	d := newDeps(t)
	wantRedirect(t, runGuard(t, d, d.RequireSession, "/accounts/1", ""), "/setup")

	dek := initVault(t, d)
	wantRedirect(t, runGuard(t, d, d.RequireSession, "/accounts/1", ""), "/login?next=%2Faccounts%2F1")
	wantRedirect(t, runGuard(t, d, d.RequireSession, "/accounts/1?q=a%20b", ""), "/login?next=%2Faccounts%2F1%3Fq%3Da%2520b")

	dec := runGuard(t, d, d.RequireSession, "/accounts/1", "stale")
	if dec == nil || !strings.Contains(dec.Location, "expired=1") || !strings.Contains(dec.Location, "next=%2Faccounts%2F1") {
		t.Fatalf("stale cookie decision = %+v, want expired=1 and next", dec)
	}

	s, err := d.Sessions.Create(dek)
	if err != nil {
		t.Fatal(err)
	}
	if dec := runGuard(t, d, d.RequireSession, "/accounts/1", s.Token); dec != nil {
		t.Fatalf("valid session decision = %+v, want allowed", dec)
	}
}

func TestRequireGuest(t *testing.T) {
	d := newDeps(t)
	wantRedirect(t, runGuard(t, d, d.RequireGuest, "/login", ""), "/setup")
	dek := initVault(t, d)
	if dec := runGuard(t, d, d.RequireGuest, "/login", ""); dec != nil {
		t.Fatalf("guest decision = %+v, want allowed", dec)
	}
	if dec := runGuard(t, d, d.RequireGuest, "/login", "stale"); dec != nil {
		t.Fatalf("stale cookie decision = %+v, want allowed", dec)
	}
	s, _ := d.Sessions.Create(dek)
	wantRedirect(t, runGuard(t, d, d.RequireGuest, "/login", s.Token), "/")
}

func TestRequireUninitialized(t *testing.T) {
	d := newDeps(t)
	if dec := runGuard(t, d, d.RequireUninitialized, "/setup", ""); dec != nil {
		t.Fatalf("uninitialized decision = %+v, want allowed", dec)
	}
	initVault(t, d)
	wantRedirect(t, runGuard(t, d, d.RequireUninitialized, "/setup", ""), "/login")
}

func TestSessionFromNil(t *testing.T) {
	if s := SessionFrom(nil); s != nil {
		t.Fatal("SessionFrom(nil) != nil")
	}
	if s := SessionFrom(&collage.RenderContext{}); s != nil {
		t.Fatal("SessionFrom(empty) != nil")
	}
}
