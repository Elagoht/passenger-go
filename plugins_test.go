package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collage"
	"github.com/Elagoht/collage/pkg/collagetest"

	"passenger-go/internal/session"
	"passenger-go/internal/web"
)

// The session cookie is the elagoht/session plugin's, encrypted, HttpOnly,
// SameSite=Strict, and kept for the browser session only, as the security
// model has it: middleware.SessionCookie takes off the Max-Age the plugin
// gives it.
func TestSessionCookie(t *testing.T) {
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	for _, res := range []*collagetest.Response{
		c.Get("/").WantStatus(http.StatusOK),
		login(collagetest.New(t, env.App.Handler()), "/login", testPass).WantStatus(http.StatusSeeOther),
	} {
		line := sessionSetCookie(res)
		for _, want := range []string{session.CookieName + "=e.", "HttpOnly", "SameSite=Strict", "Path=/"} {
			if !strings.Contains(line, want) {
				t.Errorf("%s %s: Set-Cookie lacks %q: %s", res.Method, res.URL, want, line)
			}
		}
		for _, unwanted := range []string{"Max-Age", "Expires"} {
			if strings.Contains(line, unwanted) {
				t.Errorf("%s %s: Set-Cookie has %s: %s", res.Method, res.URL, unwanted, line)
			}
		}
	}
}

// The cookie's own limits are SESSION_IDLE and SESSION_MAX, signed into it by
// the plugin, which reads the real clock: past either, the cookie opens
// nothing, whatever the DEK manager (on the test's clock) still holds.
func TestSessionCookieLimits(t *testing.T) {
	t.Run("idle", func(t *testing.T) {
		t.Setenv("SESSION_IDLE", "1s")
		c, env := newTestApp(t)
		setUp(t, c, env, testPass)
		c.Get("/").WantStatus(http.StatusOK)
		time.Sleep(2100 * time.Millisecond)
		wantLocation(t, c.Get("/generate").WantStatus(http.StatusSeeOther), "/login?next=%2Fgenerate")
	})
	t.Run("max", func(t *testing.T) {
		// Generous margins: the plugin counts whole seconds, and a slow
		// machine (CI under -race) spends a while in setUp.
		t.Setenv("SESSION_IDLE", "1h")
		t.Setenv("SESSION_MAX", "4s")
		c, env := newTestApp(t)
		setUp(t, c, env, testPass)
		time.Sleep(time.Second)
		c.Get("/").WantStatus(http.StatusOK) // activity does not push the end back
		time.Sleep(4 * time.Second)
		wantLocation(t, c.Get("/generate").WantStatus(http.StatusSeeOther), "/login?next=%2Fgenerate")
	})
}

// A vault page is never compressed (BREACH: it reflects ?q= beside secrets);
// a static file is.
func TestOnlyStaticFilesAreCompressed(t *testing.T) {
	c, _ := loggedIn(t)
	createAccount(t, c, accountValues("GitHub", "alice", "one-passphrase"))
	for _, target := range []string{"/?q=git", "/accounts/new", "/static/app.css"} {
		req := c.Request(http.MethodGet, target, nil)
		req.Header.Set("Accept-Encoding", "gzip, br")
		res := c.Do(req).WantStatus(http.StatusOK)
		got := res.Header.Get("Content-Encoding")
		if static := strings.HasPrefix(target, "/static/"); static != (got != "") {
			t.Errorf("%s: Content-Encoding = %q", target, got)
		}
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
// expired sends no session cookie: neither the dead handle again nor a clear
// (see middleware.VaultSession). The dead handle grants nothing.
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
	if line := sessionSetCookie(res); line != "" {
		t.Fatalf("expired session's response set the session cookie: %q", line)
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

// A reverse proxy on the same machine that sends no X-Forwarded-For puts every
// client on 127.0.0.1: probes from there are never counted, or one scanner
// would ban the whole site, its owner included.
func TestLoopbackIsNeverBanned(t *testing.T) {
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	local := collagetest.New(t, env.App.Handler())
	get := func(target string) *collagetest.Response {
		req := local.Request(http.MethodGet, target, nil)
		req.RemoteAddr = "127.0.0.1:41234"
		return local.Do(req)
	}
	for range 10 {
		get("/.env")
	}
	get("/login").WantStatus(http.StatusOK)
}

// Every secret guess is struck in fail2ban's secret jail, through
// web.Throttle: MaxRetry strikes ban the client from the whole site. That a
// client can reach MaxRetry within FindTime past the backoff is
// internal/web's TestSecretJailIsReachable, on the limiter's clock; here every
// strike lands at once on fail2ban's own clock, the real one.
func TestPersistentGuessingIsBanned(t *testing.T) {
	c, env := loggedIn(t)
	// The strikes of earlier guesses, as fail2ban counts them, from the
	// address every test request comes from.
	earlier := httptest.NewRequest(http.MethodPost, "/login", nil)
	for range web.SecretJailSettings.MaxRetry - 2 {
		env.Deps.Bans.Report(earlier, web.SecretJail)
	}
	guest := collagetest.New(t, env.App.Handler())
	guess := func() {
		t.Helper()
		env.Clock.Advance(10 * time.Minute) // past the backoff, which reads this clock
		page := guest.Get("/login").WantStatus(http.StatusOK)
		guest.Submit(page, "/login", url.Values{"passphrase": {"wrong passphrase!"}}).
			WantStatus(http.StatusUnprocessableEntity)
	}
	guess() // MaxRetry-1: not yet
	c.Get("/").WantStatus(http.StatusOK)
	guess()
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
