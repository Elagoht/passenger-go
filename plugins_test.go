package main

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collage"
	"github.com/Elagoht/collage/pkg/collagetest"

	"passenger-go/internal/session"
)

// The session cookie is the elagoht/session plugin's: encrypted, HttpOnly,
// SameSite=Strict, and kept by the browser for the shorter of SESSION_IDLE and
// what is left of SESSION_MAX.
func TestSessionCookie(t *testing.T) {
	for _, tc := range []struct {
		idle, max string
		maxAge    string
	}{
		{"", "", "Max-Age=900"},       // 15m idle, 12h in all
		{"20m", "10m", "Max-Age=600"}, // SESSION_MAX is sooner
		{"5m", "1h", "Max-Age=300"},
	} {
		t.Run(tc.maxAge, func(t *testing.T) {
			t.Setenv("SESSION_IDLE", tc.idle)
			t.Setenv("SESSION_MAX", tc.max)
			c, env := newTestApp(t)
			setUp(t, c, env, testPass)
			line := sessionSetCookie(c.Get("/").WantStatus(http.StatusOK))
			for _, want := range []string{session.CookieName + "=e.", "HttpOnly", "SameSite=Strict", "Path=/", tc.maxAge} {
				if !strings.Contains(line, want) {
					t.Errorf("Set-Cookie lacks %q: %s", want, line)
				}
			}
		})
	}
}

// The session cookie is Secure when the request came over TLS, and only then.
func TestSessionCookieSecureOverTLS(t *testing.T) {
	c, env := loggedIn(t)
	if line := sessionSetCookie(c.Get("/").WantStatus(http.StatusOK)); strings.Contains(line, "Secure") {
		t.Errorf("plain HTTP cookie is Secure: %s", line)
	}
	tls := collagetest.New(t, env.App.Handler())
	page := tls.Get("https://example.com/login").WantStatus(http.StatusOK)
	res := tls.Submit(page, "https://example.com/login", url.Values{"passphrase": {testPass}}).WantStatus(http.StatusSeeOther)
	if line := sessionSetCookie(res); !strings.Contains(line, "Secure") {
		t.Errorf("TLS cookie is not Secure: %s", line)
	}
}

// A handler that panics still answers with the security headers: secure and
// NoStore set them before the handler runs, or as the error page is written.
func TestSecurityHeadersSurviveAPanic(t *testing.T) {
	t.Setenv("CSRF_KEY", strings.Repeat("k", 32))
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "passenger.db"))
	app, _, err := buildApp(collage.Env{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	if err := app.Handle("/boom/", boom, collage.WithMiddleware()); err != nil {
		t.Fatal(err)
	}
	res := collagetest.New(t, app.Handler()).Get("/boom/").WantStatus(http.StatusInternalServerError)
	for name, want := range map[string]string{
		"Content-Security-Policy": contentSecurityPolicy,
		"Cache-Control":           "no-store",
		"X-Content-Type-Options":  "nosniff",
	} {
		if got := res.Header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

// /session and /static/ never refresh the cookie, even when the plugin finds it
// due for a rewrite (a quarter of SESSION_IDLE after the last one); a page
// does, and so does /session/extend.
func TestQuietPathsDoNotRefreshTheCookie(t *testing.T) {
	t.Setenv("SESSION_IDLE", "4s") // due for a rewrite after 1s
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	token := c.Get("/").WantStatus(http.StatusOK).CSRFToken()
	time.Sleep(1100 * time.Millisecond) // the plugin reads the real clock
	if line := sessionSetCookie(c.Get("/session").WantStatus(http.StatusOK)); line != "" {
		t.Errorf("/session refreshed the cookie: %s", line)
	}
	if line := sessionSetCookie(c.Get("/static/app.css").WantStatus(http.StatusOK)); line != "" {
		t.Errorf("/static/ refreshed the cookie: %s", line)
	}
	if line := sessionSetCookie(extend(c, token).WantStatus(http.StatusOK)); line == "" {
		t.Error("/session/extend did not refresh the cookie")
	}
	if line := sessionSetCookie(c.Get("/").WantStatus(http.StatusOK)); line == "" {
		t.Error("a page did not refresh the cookie")
	}
}

// When the session expires the manager's DEK is gone (wiped as it is removed,
// see internal/session's TestExpiredDEKZeroed), and the request that finds it
// expired clears the cookie.
func TestExpiredSessionDropsTheDEK(t *testing.T) {
	c, env := loggedIn(t)
	if n := env.Deps.Sessions.Len(); n != 1 {
		t.Fatalf("sessions = %d, want 1", n)
	}
	env.Clock.Advance(16 * time.Minute)
	res := c.Get("/generate").WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/login?next=%2Fgenerate")
	if n := env.Deps.Sessions.Len(); n != 0 {
		t.Fatalf("sessions after expiry = %d, want 0: the DEK is still held", n)
	}
	line := sessionSetCookie(res)
	if !strings.Contains(line, "Max-Age=0") {
		t.Fatalf("expired session's cookie not cleared: %q", line)
	}
	// The session is gone for /session too.
	c.Get("/session").WantStatus(http.StatusUnauthorized)
}

// elagoht/ratelimit limits every form that checks a secret, ten at once per
// client, before the backoff even looks at the guess.
func TestSecretFormsAreRequestLimited(t *testing.T) {
	c, env := loggedIn(t)
	guest := collagetest.New(t, env.App.Handler())
	for _, target := range []string{"/login", "/recover", "/setup", "/export", "/settings/passphrase", "/settings/recovery"} {
		client := c
		if target == "/login" || target == "/recover" || target == "/setup" {
			client = guest
		}
		res := client.Do(client.Request(http.MethodPost, target, nil))
		if got := res.Header.Get("RateLimit-Limit"); got != "10" {
			t.Errorf("POST %s: RateLimit-Limit = %q, want 10", target, got)
		}
	}
	// A refused new passphrase is not a guess, so the backoff never sees
	// these; the request limit still stops the eleventh.
	page := guest.Get("/recover").WantStatus(http.StatusOK)
	bad := url.Values{"recovery": {"AAAA"}, "passphrase": {"short"}, "confirm": {"short"}}
	for range 9 { // one was spent above
		guest.Submit(page, "/recover", bad).WantStatus(http.StatusUnprocessableEntity)
	}
	res := guest.Submit(page, "/recover", bad).WantStatus(http.StatusTooManyRequests)
	if res.Header.Get("Retry-After") == "" {
		t.Error("request limit's 429 has no Retry-After")
	}
	// Pages are not limited.
	for range 12 {
		guest.Get("/login").WantStatus(http.StatusOK)
	}
}

// elagoht/fail2ban bans a client that probes for files no site should serve.
func TestProbesAreBanned(t *testing.T) {
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	scanner := collagetest.New(t, env.App.Handler())
	for range 3 {
		scanner.Get("/.env")
	}
	res := scanner.Get("/login").WantStatus(http.StatusForbidden)
	if strings.TrimSpace(res.Body) != "Forbidden" {
		t.Fatalf("ban body = %q", res.Body)
	}
}

// Every secret guess is also struck in fail2ban's secret jail: a client that
// keeps guessing past the backoff, for an hour, is banned from the whole site.
func TestPersistentGuessingIsBanned(t *testing.T) {
	t.Setenv("SESSION_IDLE", "1h") // the clock moves 30 minutes between c's requests
	c, env := loggedIn(t)
	guest := collagetest.New(t, env.App.Handler())
	guess := func(client *collagetest.Client, target string, v url.Values) {
		t.Helper()
		env.Clock.Advance(10 * time.Minute) // past the backoff, which reads this clock
		page := client.Get(target).WantStatus(http.StatusOK)
		client.Submit(page, target, v).WantStatus(http.StatusUnprocessableEntity)
	}
	// Ten per form (each form has its own request limit of ten at once), all
	// from one address: thirty strikes.
	for i := range 10 {
		guess(c, "/export", url.Values{"passphrase": {"wrong passphrase!"}})
		guess(c, "/settings/recovery", url.Values{"current": {"wrong passphrase!"}})
		if i == 9 { // 29 strikes: not yet
			c.Get("/").WantStatus(http.StatusOK)
		}
		guess(guest, "/login", url.Values{"passphrase": {"wrong passphrase!"}})
	}
	c.Get("/").WantStatus(http.StatusForbidden)
}

// elagoht/health answers the probes, the database checked for readiness.
func TestHealth(t *testing.T) {
	c, _ := newTestApp(t)
	for _, target := range []string{"/healthz", "/readyz"} {
		res := c.Get(target).WantStatus(http.StatusOK)
		if strings.TrimSpace(res.Body) != "ok" {
			t.Errorf("%s = %q, want ok", target, res.Body)
		}
	}
}
