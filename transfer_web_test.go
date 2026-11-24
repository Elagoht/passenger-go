package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Elagoht/collage/pkg/collagetest"

	"passenger-go/internal/session"
)

// upload posts file as the import form's multipart "file" field.
func upload(t *testing.T, c *collagetest.Client, token, name, file string) *collagetest.Response {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("_csrf", token); err != nil {
		t.Fatal(err)
	}
	if file != "" {
		part, err := w.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte(file))
	}
	w.Close()
	req := c.Request(http.MethodPost, "/import", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.Do(req)
}

func TestImport(t *testing.T) {
	c, _ := loggedIn(t)
	createAccount(t, c, accountValues("GitHub", "alice", "one-passphrase"))
	page := c.Get("/import").WantStatus(http.StatusOK)

	file := "name,url,username,password,note\r\n" +
		"GitHub,https://github.com,alice,SECRET-DUP,\r\n" +
		"GitLab,https://gitlab.com,bob,pw2,\"a, b\"\r\n" +
		"Codeberg,https://codeberg.org,carol,pw3,\r\n"
	res := upload(t, c, page.CSRFToken(), "pw.csv", file).WantStatus(http.StatusOK)
	wantBody(t, res, "2 accounts imported")
	wantBody(t, res, "1 failed")
	wantBody(t, res, "duplicate")
	wantBody(t, res, "alice")
	wantNoBody(t, res, "SECRET-DUP")
	wantBody(t, c.Get("/").WantStatus(http.StatusOK), "Codeberg")
}

func TestImportReasonsAndErrors(t *testing.T) {
	c, _ := loggedIn(t)
	page := c.Get("/import").WantStatus(http.StatusOK)
	file := "platform,identifier,passphrase,url,notes\n" +
		",a,SECRET1,,\n" +
		"P,,SECRET2,,\n" +
		"P,b,,,\n"
	res := upload(t, c, page.CSRFToken(), "x.csv", file).WantStatus(http.StatusOK)
	for _, want := range []string{"0 accounts imported", "3 failed", "missing platform", "missing identifier", "missing passphrase"} {
		wantBody(t, res, want)
	}
	wantNoBody(t, res, "SECRET")

	wantBody(t, upload(t, c, page.CSRFToken(), "x.csv", "a,b\n1,2\n").WantStatus(http.StatusUnprocessableEntity), "not a supported")
	wantBody(t, upload(t, c, page.CSRFToken(), "", "").WantStatus(http.StatusUnprocessableEntity), "Choose a CSV file")
}

func TestImportRequiresSession(t *testing.T) {
	c, _ := newTestApp(t)
	wantLocation(t, c.Get("/import").WantStatus(http.StatusSeeOther), "/setup")
}

func TestExport(t *testing.T) {
	c, _ := loggedIn(t)
	createAccount(t, c, accountValues("GitHub", "alice", "one-passphrase"))
	page := c.Get("/export").WantStatus(http.StatusOK)

	res := c.Submit(page, "/export", url.Values{"passphrase": {"wrong passphrase!"}}).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Wrong passphrase.")
	wantNoBody(t, res, "one-passphrase")
	wantNoBody(t, res, "wrong passphrase!")

	res = c.Submit(page, "/export", url.Values{"passphrase": {testPass}}).WantStatus(http.StatusOK)
	if ct := res.Header.Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cd := res.Header.Get("Content-Disposition"); cd != `attachment; filename="passenger-export-2026-01-01.csv"` {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q", cc)
	}
	if !strings.HasPrefix(res.Body, "platform,identifier,passphrase,url,notes\n") {
		t.Fatalf("body = %q", res.Body)
	}
	wantBody(t, res, "GitHub,alice,one-passphrase,github.com,some notes")
}

func TestExportIsRateLimited(t *testing.T) {
	c, _ := loggedIn(t)
	page := c.Get("/export").WantStatus(http.StatusOK)
	for range 5 {
		c.Submit(page, "/export", url.Values{"passphrase": {"nope nope nope"}}).WantStatus(http.StatusUnprocessableEntity)
	}
	c.Submit(page, "/export", url.Values{"passphrase": {testPass}}).WantStatus(http.StatusTooManyRequests)
}

func TestChangePassphrase(t *testing.T) {
	owner, env := loggedIn(t)
	createAccount(t, owner, accountValues("GitHub", "alice", "one-passphrase"))
	second := collagetest.New(t, env.App.Handler())
	wantLocation(t, login(second, "/login", testPass).WantStatus(http.StatusSeeOther), "/")
	second.Get("/").WantStatus(http.StatusOK)

	page := owner.Get("/settings/passphrase").WantStatus(http.StatusOK)
	// The owner's cookie before the change, to replay once it has.
	before := sessionCookieValue(t, page)
	values := func(cur, pass, confirm string) url.Values {
		return url.Values{"current": {cur}, "passphrase": {pass}, "confirm": {confirm}}
	}
	res := owner.Submit(page, "/settings/passphrase", values("wrong current!!", otherPass, otherPass)).
		WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Wrong passphrase.")
	// Neither the current passphrase nor the new one is typed back.
	wantNoBody(t, res, "wrong current!!")
	wantNoBody(t, res, otherPass)
	wantBody(t, owner.Submit(page, "/settings/passphrase", values(testPass, "short", "short")).
		WantStatus(http.StatusUnprocessableEntity), "at least 12")
	wantBody(t, owner.Submit(page, "/settings/passphrase", values(testPass, otherPass, testPass)).
		WantStatus(http.StatusUnprocessableEntity), "do not match")

	res = owner.Submit(page, "/settings/passphrase", values(testPass, otherPass, otherPass)).WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/settings/passphrase")
	// The flash message is shown once: after the redirect, not on a reload.
	wantBody(t, owner.Follow(res).WantStatus(http.StatusOK), "Passphrase changed.")
	wantNoBody(t, owner.Get("/settings/passphrase").WantStatus(http.StatusOK), "Passphrase changed.")

	// The cookie from before the change holds the old handle, which the
	// rotation revoked: replayed, it opens nothing.
	replay := collagetest.New(t, env.App.Handler())
	req := replay.Request(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: before})
	wantLocation(t, replay.Do(req).WantStatus(http.StatusSeeOther), "/login")

	// The rotated session still works; the other one ended.
	wantBody(t, owner.Get("/").WantStatus(http.StatusOK), "GitHub")
	if loc := second.Get("/").WantStatus(http.StatusSeeOther).Location(); !strings.HasPrefix(loc, "/login") {
		t.Fatalf("second Location = %q, want /login…", loc)
	}

	fresh := collagetest.New(t, env.App.Handler())
	login(fresh, "/login", testPass).WantStatus(http.StatusUnprocessableEntity)
	wantLocation(t, login(fresh, "/login", otherPass).WantStatus(http.StatusSeeOther), "/")
	// The data is still readable under the new passphrase.
	wantBody(t, fresh.Get("/").WantStatus(http.StatusOK), "GitHub")
}

func TestRegenerateRecoveryKey(t *testing.T) {
	owner, env := newTestApp(t)
	oldKey := setUp(t, owner, env, testPass)
	page := owner.Get("/settings/recovery").WantStatus(http.StatusOK)

	wantBody(t, owner.Submit(page, "/settings/recovery", url.Values{"current": {"wrong current!!"}}).
		WantStatus(http.StatusUnprocessableEntity), "Wrong passphrase.")

	res := owner.Submit(page, "/settings/recovery", url.Values{"current": {testPass}}).WantStatus(http.StatusOK)
	newKey := recoveryKeyRE.FindString(res.Body)
	if newKey == "" || newKey == oldKey {
		t.Fatalf("new key = %q (old %q):\n%s", newKey, oldKey, res.Body)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q", cc)
	}
	// The key is shown once.
	wantNoBody(t, owner.Get("/settings/recovery").WantStatus(http.StatusOK), newKey)

	c := collagetest.New(t, env.App.Handler())
	rp := c.Get("/recover").WantStatus(http.StatusOK)
	recoverForm := func(key string) url.Values { return recoverValues(key, otherPass, otherPass) }
	c.Submit(rp, "/recover", recoverForm(oldKey)).WantStatus(http.StatusUnprocessableEntity)
	c.Submit(rp, "/recover", recoverForm(newKey)).WantStatus(http.StatusOK)
}
