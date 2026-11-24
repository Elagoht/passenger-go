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

func TestListenPort(t *testing.T) {
	tests := []struct {
		name string
		env  int    // collage's Env.Port: -port, PORT, or 6060
		port string // the PORT variable
		want int
	}{
		{"PORT unset uses 8080", 6060, "", 8080},
		{"PORT set is kept", 6060, "6060", 6060},
		{"PORT set to another port", 9000, "9000", 9000},
		{"-port flag without PORT is kept", 9000, "", 9000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(k string) string {
				if k == "PORT" {
					return tt.port
				}
				return ""
			}
			if got := listenPort(collage.Env{Port: tt.env}, getenv); got != tt.want {
				t.Fatalf("listenPort = %d, want %d", got, tt.want)
			}
		})
	}
}

// Static files may be cached (they carry no secrets) but keep the other
// security headers.
func TestStaticHeaders(t *testing.T) {
	c, _ := newTestApp(t)
	res := c.Get("/static/app.css").WantStatus(http.StatusOK)
	if cc := res.Header.Get("Cache-Control"); strings.Contains(cc, "no-store") {
		t.Errorf("Cache-Control = %q, want static files cacheable", cc)
	}
	for name, want := range map[string]string{
		"Content-Security-Policy": web.ContentSecurityPolicy,
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
	} {
		if got := res.Header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestShutdownEndsSessions(t *testing.T) {
	_, env := newTestApp(t)
	s := createVaultAndSession(t, env)
	if err := env.App.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := env.Deps.Sessions.Peek(s.Token); ok {
		t.Fatal("a session outlived the shutdown")
	}
}
