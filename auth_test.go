package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collage"
	"github.com/Elagoht/collage/pkg/collagetest"
)

const (
	testPass  = "correct horse battery"
	otherPass = "another long passphrase"
)

var recoveryKeyRE = regexp.MustCompile(`[A-Z2-7]{4}(?:-[A-Z2-7]{4}){12}`)

func wantLocation(t *testing.T, res *collagetest.Response, want string) {
	t.Helper()
	if loc := res.Location(); loc != want {
		t.Fatalf("%s %s: Location = %q, want %q", res.Method, res.URL, loc, want)
	}
}

func wantBody(t *testing.T, res *collagetest.Response, want string) {
	t.Helper()
	if !strings.Contains(res.Body, want) {
		t.Fatalf("%s %s: body lacks %q:\n%s", res.Method, res.URL, want, res.Body)
	}
}

func wantNoBody(t *testing.T, res *collagetest.Response, unwanted string) {
	t.Helper()
	if strings.Contains(res.Body, unwanted) {
		t.Fatalf("%s %s: body contains %q", res.Method, res.URL, unwanted)
	}
}

func setupValues(code, pass, confirm string) url.Values {
	return url.Values{"code": {code}, "passphrase": {pass}, "confirm": {confirm}}
}

// setUp creates the vault through the UI, leaving c logged in, and returns the
// recovery key the confirm page showed.
func setUp(t *testing.T, c *collagetest.Client, env *testEnv, pass string) string {
	t.Helper()
	page := c.Get("/setup").WantStatus(http.StatusOK)
	res := c.Submit(page, "/setup", setupValues(env.Deps.SetupCode.String(), pass, pass)).
		WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/setup/confirm")
	confirm := c.Follow(res).WantStatus(http.StatusOK)
	key := recoveryKeyRE.FindString(confirm.Body)
	if key == "" {
		t.Fatalf("confirm page shows no recovery key:\n%s", confirm.Body)
	}
	done := c.Submit(confirm, "/setup/confirm", url.Values{"recovery": {key}}).WantStatus(http.StatusSeeOther)
	wantLocation(t, done, "/")
	return key
}

// login submits the login form reached at target (which may carry next).
func login(c *collagetest.Client, target, pass string) *collagetest.Response {
	page := c.Get(target).WantStatus(http.StatusOK)
	return c.Submit(page, "/login", url.Values{"passphrase": {pass}})
}

func TestFreshAppSendsEverythingToSetup(t *testing.T) {
	c, _ := newTestApp(t)
	for _, target := range []string{"/", "/login", "/recover"} {
		wantLocation(t, c.Get(target).WantStatus(http.StatusSeeOther), "/setup")
	}
	wantLocation(t, c.Get("/setup/confirm").WantStatus(http.StatusSeeOther), "/setup")
}

func TestSetupFlow(t *testing.T) {
	c, env := newTestApp(t)
	code := env.Deps.SetupCode.String()
	page := c.Get("/setup").WantStatus(http.StatusOK)

	wrongCode := "00000000"
	if wrongCode == code {
		wrongCode = "11111111"
	}
	res := c.Submit(page, "/setup", setupValues(wrongCode, testPass, testPass)).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Wrong setup code")
	wantNoBody(t, res, testPass)

	res = c.Submit(page, "/setup", setupValues(code, "short", "short")).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "at least 12 characters")

	res = c.Submit(page, "/setup", setupValues(code, testPass, otherPass)).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "do not match")
	wantNoBody(t, res, testPass)
	wantNoBody(t, res, otherPass)

	res = c.Submit(page, "/setup", setupValues(code, testPass, testPass)).WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/setup/confirm")
	confirm := c.Follow(res).WantStatus(http.StatusOK)
	key := recoveryKeyRE.FindString(confirm.Body)
	if key == "" {
		t.Fatalf("confirm page shows no recovery key:\n%s", confirm.Body)
	}
	if ok, _ := env.Deps.Store.Initialized(context.Background()); ok {
		t.Fatal("vault written before the recovery key was confirmed")
	}

	res = c.Submit(confirm, "/setup/confirm", url.Values{"recovery": {"AAAA-BBBB"}}).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "does not match")
	wantBody(t, res, key) // shown again

	typed := strings.ToLower(strings.ReplaceAll(key, "-", " "))
	res = c.Submit(confirm, "/setup/confirm", url.Values{"recovery": {typed}}).WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/")
	cookie := strings.Join(res.Header.Values("Set-Cookie"), "\n")
	for _, want := range []string{"passenger_session=", "HttpOnly", "SameSite=Strict"} {
		if !strings.Contains(cookie, want) {
			t.Errorf("Set-Cookie lacks %q:\n%s", want, cookie)
		}
	}

	home := c.Get("/").WantStatus(http.StatusOK)
	wantBody(t, home, "No accounts yet.")
	wantLocation(t, c.Get("/setup").WantStatus(http.StatusSeeOther), "/login")
	wantLocation(t, c.Get("/setup/confirm").WantStatus(http.StatusSeeOther), "/login")
}

func TestSetupRequiresCSRFToken(t *testing.T) {
	c, env := newTestApp(t)
	body := setupValues(env.Deps.SetupCode.String(), testPass, testPass).Encode()
	req := c.Request(http.MethodPost, "/setup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Do(req).WantStatus(http.StatusForbidden)
	if ok, _ := env.Deps.Store.Initialized(context.Background()); ok {
		t.Fatal("vault initialized without a token")
	}
}

func TestSetupCodeIsRateLimited(t *testing.T) {
	c, env := newTestApp(t)
	code := env.Deps.SetupCode.String()
	wrong := "00000000"
	if wrong == code {
		wrong = "11111111"
	}
	page := c.Get("/setup").WantStatus(http.StatusOK)
	for i := 0; i < 5; i++ {
		c.Submit(page, "/setup", setupValues(wrong, testPass, testPass)).WantStatus(http.StatusUnprocessableEntity)
	}
	res := c.Submit(page, "/setup", setupValues(code, testPass, testPass)).WantStatus(http.StatusTooManyRequests)
	wantBody(t, res, "Try again in 1 seconds")

	env.Clock.Advance(time.Second)
	c.Submit(page, "/setup", setupValues(code, testPass, testPass)).WantStatus(http.StatusSeeOther)
}

func TestSetupCodeIsLoggedOnlyWhenUninitialized(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	c, env := newTestApp(t)
	if want := "setup code: " + env.Deps.SetupCode.String(); !strings.Contains(buf.String(), want) {
		t.Fatalf("log = %q, want %q", buf.String(), want)
	}
	setUp(t, c, env, testPass)

	buf.Reset()
	app, _, err := buildApp(collage.Env{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if strings.Contains(buf.String(), "setup code") {
		t.Fatalf("initialized vault logged a setup code: %q", buf.String())
	}
}

func TestLogin(t *testing.T) {
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	guest := collagetest.New(t, env.App.Handler())

	res := login(guest, "/login", "wrong passphrase!").WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Wrong passphrase")
	wantNoBody(t, res, "wrong passphrase!")

	res = login(guest, "/login?next=%2Faccounts%2Fnew", testPass).WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/accounts/new")

	// Logged in: the login page sends the reader home.
	wantLocation(t, guest.Get("/login").WantStatus(http.StatusSeeOther), "/")

	evil := collagetest.New(t, env.App.Handler())
	res = login(evil, "/login?next="+url.QueryEscape("//evil.com"), testPass).WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/")
}

func TestLoginIsRateLimited(t *testing.T) {
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	guest := collagetest.New(t, env.App.Handler())
	for i := 0; i < 5; i++ {
		login(guest, "/login", "wrong passphrase!").WantStatus(http.StatusUnprocessableEntity)
	}
	res := login(guest, "/login", testPass).WantStatus(http.StatusTooManyRequests)
	wantBody(t, res, "Try again in 1 seconds")

	env.Clock.Advance(time.Second)
	wantLocation(t, login(guest, "/login", testPass).WantStatus(http.StatusSeeOther), "/")
}

func TestSessionExpiry(t *testing.T) {
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	env.Clock.Advance(16 * time.Minute)

	res := c.Get("/").WantStatus(http.StatusSeeOther)
	loc := res.Location()
	if !strings.Contains(loc, "expired=1") || !strings.Contains(loc, "next=%2F") {
		t.Fatalf("Location = %q, want expired=1 and next=%%2F", loc)
	}
	page := c.Follow(res).WantStatus(http.StatusOK)
	wantBody(t, page, "Your session expired")

	// Signing in again returns to the path in next.
	res = c.Submit(page, "/login", url.Values{"passphrase": {testPass}}).WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/")
}

func TestLogout(t *testing.T) {
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	home := c.Get("/").WantStatus(http.StatusOK)

	res := c.Submit(home, "/logout", nil).WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/login")
	wantLocation(t, c.Get("/").WantStatus(http.StatusSeeOther), "/login?next=%2F")
}

func recoverValues(key, pass, confirm string) url.Values {
	return url.Values{"recovery": {key}, "passphrase": {pass}, "confirm": {confirm}}
}

func TestRecover(t *testing.T) {
	owner, env := newTestApp(t)
	oldKey := setUp(t, owner, env, testPass)
	owner.Get("/").WantStatus(http.StatusOK)

	c := collagetest.New(t, env.App.Handler())
	page := c.Get("/recover").WantStatus(http.StatusOK)

	res := c.Submit(page, "/recover", recoverValues(oldKey, otherPass, testPass)).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "do not match")
	wantNoBody(t, res, oldKey)

	res = c.Submit(page, "/recover", recoverValues("AAAA-BBBB", otherPass, otherPass)).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Wrong recovery key")

	typed := strings.ToLower(strings.ReplaceAll(oldKey, "-", " "))
	res = c.Submit(page, "/recover", recoverValues(typed, otherPass, otherPass)).WantStatus(http.StatusOK)
	newKey := recoveryKeyRE.FindString(res.Body)
	if newKey == "" || newKey == oldKey {
		t.Fatalf("new key = %q (old %q):\n%s", newKey, oldKey, res.Body)
	}
	wantBody(t, res, `href="/login"`)

	// The owner's session ended.
	if loc := owner.Get("/").WantStatus(http.StatusSeeOther).Location(); !strings.HasPrefix(loc, "/login") {
		t.Fatalf("owner Location = %q, want /login…", loc)
	}

	page = c.Get("/recover").WantStatus(http.StatusOK)
	c.Submit(page, "/recover", recoverValues(oldKey, testPass, testPass)).WantStatus(http.StatusUnprocessableEntity)

	login(c, "/login", testPass).WantStatus(http.StatusUnprocessableEntity)
	wantLocation(t, login(c, "/login", otherPass).WantStatus(http.StatusSeeOther), "/")
}
