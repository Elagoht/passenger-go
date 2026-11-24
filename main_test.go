package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
	"passenger-go/internal/store"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

func TestNewAppRequiresCSRFKey(t *testing.T) {
	t.Setenv("CSRF_KEY", "")
	if _, err := newApp(collage.Env{CacheDir: t.TempDir()}); err == nil {
		t.Fatal("newApp() = nil error, want CSRF_KEY error")
	}
}

func TestUninitializedAppRedirectsToSetup(t *testing.T) {
	c, _ := newTestApp(t)
	res := c.Get("/").WantStatus(http.StatusSeeOther)
	if loc := res.Location(); loc != "/setup" {
		t.Fatalf("Location = %q, want /setup", loc)
	}
	if got := res.Header.Get("Content-Security-Policy"); got != web.ContentSecurityPolicy {
		t.Errorf("CSP = %q", got)
	}
}

// createVaultAndSession writes a vault row directly and opens a session on it,
// returning the session token.
func createVaultAndSession(t *testing.T, env *testEnv) *session.Session {
	t.Helper()
	dek, err := vault.NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	pass, err := vault.Wrap("pass", dek, env.Deps.KDF, vault.AADPass)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := vault.Wrap("rec", dek, env.Deps.KDF, vault.AADRecovery)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Deps.Store.CreateVault(context.Background(), store.VaultRow{KDF: env.Deps.KDF, Pass: pass, Recovery: rec}); err != nil {
		t.Fatal(err)
	}
	s, err := env.Deps.Sessions.Create(dek)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAppLayout(t *testing.T) {
	c, env := newTestApp(t)
	if env.Deps.KDF != testKDF {
		t.Fatalf("KDF = %+v, want the test override", env.Deps.KDF)
	}
	s := createVaultAndSession(t, env)

	// No cookie: to login, carrying the path.
	if loc := c.Get("/").WantStatus(http.StatusSeeOther).Location(); loc != "/login?next=%2F" {
		t.Fatalf("Location = %q, want /login?next=%%2F", loc)
	}

	env.Clock.Advance(5 * time.Minute) // the page view slides the session
	req := c.Request(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: s.Token})
	res := c.Do(req).WantStatus(http.StatusOK)

	wantExp := env.Clock.Now().Add(15 * time.Minute).UnixMilli()
	for _, want := range []string{
		`data-session-expires="` + strconv.FormatInt(wantExp, 10) + `"`,
		`id="session-banner"`,
		`href="/accounts/new"`, `href="/generate"`, `href="/import"`, `href="/export"`, `href="/settings/passphrase"`,
		`action="/logout"`,
		`name="_csrf"`,
	} {
		if !strings.Contains(res.Body, want) {
			t.Errorf("body lacks %q:\n%s", want, res.Body)
		}
	}
	if n := strings.Count(res.Body, "<title>"); n != 1 {
		t.Errorf("%d titles, want 1", n)
	}
	if n := strings.Count(res.Body, `name="_csrf"`); n != 1 {
		t.Errorf("%d csrf inputs, want 1", n)
	}
	if strings.Contains(res.Body, "style=") || strings.Contains(res.Body, "<style") {
		t.Error("inline style in layout")
	}
	for name, want := range map[string]string{
		"Content-Security-Policy": web.ContentSecurityPolicy,
		"Cache-Control":           "no-store",
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
	} {
		if got := res.Header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestStaticDoesNotTouchSession(t *testing.T) {
	c, env := newTestApp(t)
	s := createVaultAndSession(t, env)
	env.Clock.Advance(10 * time.Minute)
	req := c.Request(http.MethodGet, "/static/favicon.svg", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: s.Token})
	c.Do(req).WantStatus(http.StatusOK)
	env.Clock.Advance(6 * time.Minute)
	if _, ok := env.Deps.Sessions.Peek(s.Token); ok {
		t.Fatal("a static request extended the session")
	}
}
