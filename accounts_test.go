package main

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Elagoht/collage/pkg/collage"
	"github.com/Elagoht/collage/pkg/collagetest"
)

// loggedIn is a browser with a finished setup and a live session.
func loggedIn(t *testing.T) (*collagetest.Client, *testEnv) {
	t.Helper()
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	return c, env
}

// post sends a form with the token in the header, as a script would, for the
// pages whose forms Submit cannot tell apart or that have none.
func post(c *collagetest.Client, target, token string, v url.Values) *collagetest.Response {
	req := c.Request(http.MethodPost, target, strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", token)
	return c.Do(req)
}

func accountValues(platform, identifier, passphrase string) url.Values {
	return url.Values{
		"platform": {platform}, "identifier": {identifier}, "passphrase": {passphrase},
		"url": {"github.com"}, "notes": {"some notes"},
	}
}

// createAccount posts the new-account form and returns the new account's path.
func createAccount(t *testing.T, c *collagetest.Client, v url.Values) string {
	t.Helper()
	page := c.Get("/accounts/new").WantStatus(http.StatusOK)
	res := c.Submit(page, "/accounts/new", v).WantStatus(http.StatusSeeOther)
	loc := res.Location()
	if !strings.HasPrefix(loc, "/accounts/") {
		t.Fatalf("Location = %q, want /accounts/{id}", loc)
	}
	return loc
}

func TestCreateAccount(t *testing.T) {
	c, _ := loggedIn(t)
	loc := createAccount(t, c, accountValues("GitHub", "alice", "Str0ng!Passphrase#2026"))

	detail := c.Get(loc).WantStatus(http.StatusOK)
	wantBody(t, detail, `value="GitHub"`)
	wantBody(t, detail, `type="password"`)

	list := c.Get("/").WantStatus(http.StatusOK)
	wantBody(t, list, "GitHub")
	wantBody(t, list, "alice")
	wantBody(t, list, "strength-strong")
	wantNoBody(t, list, "some notes")
	wantBody(t, list, `href="https://github.com"`)
	wantBody(t, list, `class="initials" aria-hidden="true">G<`)
	wantNoBody(t, list, "Str0ng!Passphrase")
}

func TestCreateAccountKeepsValuesWhenInvalid(t *testing.T) {
	c, _ := loggedIn(t)
	page := c.Get("/accounts/new").WantStatus(http.StatusOK)
	res := c.Submit(page, "/accounts/new", accountValues("", "alice", "secret-x")).
		WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Platform is required.")
	wantBody(t, res, `value="alice"`)
	wantBody(t, res, `value="github.com"`)
	wantBody(t, res, "some notes")
	// The account's passphrase is kept like the rest (only the master
	// passphrase and the recovery key are never echoed).
	wantBody(t, res, `value="secret-x"`)
}

func TestDuplicateAccountIsRefused(t *testing.T) {
	c, _ := loggedIn(t)
	createAccount(t, c, accountValues("GitHub", "alice", "one-passphrase"))

	page := c.Get("/accounts/new").WantStatus(http.StatusOK)
	res := c.Submit(page, "/accounts/new", accountValues("GitHub", "alice", "two-passphrase")).
		WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "An account with this platform and identifier already exists")
}

func TestEditAccount(t *testing.T) {
	c, _ := loggedIn(t)
	loc := createAccount(t, c, accountValues("GitHub", "alice", "one-passphrase"))
	other := createAccount(t, c, accountValues("GitLab", "alice", "two-passphrase"))

	page := c.Get(loc).WantStatus(http.StatusOK)
	res := c.Submit(page, loc, accountValues("GitHub", "alice2", "changed-passphrase")).
		WantStatus(http.StatusSeeOther)
	wantLocation(t, res, loc)
	wantBody(t, c.Get(loc).WantStatus(http.StatusOK), `value="alice2"`)

	// Colliding with another account is refused, values kept.
	page = c.Get(other).WantStatus(http.StatusOK)
	res = c.Submit(page, other, accountValues("GitHub", "alice2", "x-passphrase")).
		WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "An account with this platform and identifier already exists")
	wantBody(t, res, `value="x-passphrase"`)

	// Edit failure on a missing field keeps the rest.
	res = c.Submit(page, other, accountValues("GitLab", "", "x")).
		WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Identifier is required.")
	wantBody(t, res, `value="GitLab"`)
}

func TestUnknownAccountIs404(t *testing.T) {
	c, _ := loggedIn(t)
	c.Get("/accounts/nope").WantStatus(http.StatusNotFound)
	c.Get("/accounts/nope/delete").WantStatus(http.StatusNotFound)

	page := c.Get("/accounts/new").WantStatus(http.StatusOK)
	post(c, "/accounts/nope", page.CSRFToken(), accountValues("a", "b", "c")).WantStatus(http.StatusNotFound)
	post(c, "/accounts/nope", page.CSRFToken(), accountValues("", "", "")).WantStatus(http.StatusNotFound)
	post(c, "/accounts/nope/delete", page.CSRFToken(), nil).WantStatus(http.StatusNotFound)
	req := c.Request(http.MethodPost, "/accounts/nope/passphrase", nil)
	req.Header.Set("X-CSRF-Token", page.CSRFToken())
	c.Do(req).WantStatus(http.StatusNotFound)
}

func TestDeleteAccount(t *testing.T) {
	c, _ := loggedIn(t)
	loc := createAccount(t, c, accountValues("GitHub", "alice", "one-passphrase"))

	confirm := c.Get(loc + "/delete").WantStatus(http.StatusOK)
	wantBody(t, confirm, "GitHub")
	res := c.Submit(confirm, loc+"/delete", nil).WantStatus(http.StatusSeeOther)
	wantLocation(t, res, "/")

	wantNoBody(t, c.Get("/").WantStatus(http.StatusOK), "alice")
	c.Get(loc).WantStatus(http.StatusNotFound)
}

func TestSearch(t *testing.T) {
	c, _ := loggedIn(t)
	createAccount(t, c, accountValues("GitHub", "alice", "one-passphrase"))
	v := accountValues("Gmail", "bob", "two-passphrase")
	v.Set("url", "mail.example.com")
	v.Set("notes", "")
	createAccount(t, c, v)

	res := c.Get("/?q=git").WantStatus(http.StatusOK)
	wantBody(t, res, "GitHub")
	wantNoBody(t, res, "Gmail")

	// Case-insensitive, and matches notes.
	wantBody(t, c.Get("/?q=SOME+NOTES").WantStatus(http.StatusOK), "GitHub")

	res = c.Get("/?q=" + url.QueryEscape("<script>alert(1)</script>")).WantStatus(http.StatusOK)
	wantBody(t, res, "&lt;script&gt;")
	wantNoBody(t, res, "<script>alert(1)")
}

func TestAccountFieldsAreEscaped(t *testing.T) {
	c, _ := loggedIn(t)
	loc := createAccount(t, c, accountValues("<img src=x onerror=alert(1)>", "<b>id</b>", "pw-passphrase"))
	for _, page := range []*collagetest.Response{c.Get("/"), c.Get(loc)} {
		page.WantStatus(http.StatusOK)
		wantBody(t, page, "&lt;img")
		wantNoBody(t, page, "<img src=x")
		wantNoBody(t, page, "<b>id</b>")
	}
}

func TestExternalLinkOnlyForHTTP(t *testing.T) {
	c, _ := loggedIn(t)
	v := accountValues("Evil", "e", "pw-passphrase")
	v.Set("url", "javascript:alert(1)")
	createAccount(t, c, v)
	v = accountValues("Evil2", "e", "pw-passphrase")
	v.Set("url", "javascript://x/%0Aalert(1)")
	createAccount(t, c, v)
	v = accountValues("Plain", "p", "pw-passphrase")
	v.Set("url", "http://example.com/a")
	createAccount(t, c, v)

	list := c.Get("/").WantStatus(http.StatusOK)
	wantNoBody(t, list, `href="javascript`)
	wantBody(t, list, `href="http://example.com/a"`)
	if n := strings.Count(list.Body, `>Open site<`); n != 1 {
		t.Fatalf("%d external links, want 1", n)
	}
}

func TestPassphraseJSON(t *testing.T) {
	c, env := loggedIn(t)
	loc := createAccount(t, c, accountValues("GitHub", "alice", "s3cret&<pass>"))
	page := c.Get("/").WantStatus(http.StatusOK)

	req := c.Request(http.MethodPost, loc+"/passphrase", nil)
	req.Header.Set("X-CSRF-Token", page.CSRFToken())
	res := c.Do(req).WantStatus(http.StatusOK)
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q", ct)
	}
	var got struct {
		Passphrase string `json:"passphrase"`
	}
	if err := json.Unmarshal([]byte(res.Body), &got); err != nil {
		t.Fatalf("body %q: %v", res.Body, err)
	}
	if got.Passphrase != "s3cret&<pass>" {
		t.Fatalf("passphrase = %q, want %q", got.Passphrase, "s3cret&<pass>")
	}

	// No token.
	c.Do(c.Request(http.MethodPost, loc+"/passphrase", nil)).WantStatus(http.StatusForbidden)

	// No session.
	guest := collagetest.New(t, env.App.Handler())
	req = guest.Request(http.MethodPost, loc+"/passphrase", nil)
	req.Header.Set("X-CSRF-Token", page.CSRFToken())
	guest.Do(req).WantStatus(http.StatusSeeOther)
}

func TestAccountPagesNeedASession(t *testing.T) {
	_, env := loggedIn(t)
	guest := collagetest.New(t, env.App.Handler())
	for _, target := range []string{"/", "/accounts/new", "/accounts/x", "/accounts/x/delete", "/generate"} {
		want := "/login?next=" + url.QueryEscape(target)
		if target == "/" {
			want = "/login"
		}
		wantLocation(t, guest.Get(target).WantStatus(http.StatusSeeOther), want)
	}
}

var generatedRE = regexp.MustCompile(`id="generated" readonly value="([^"]*)"`)

func generated(t *testing.T, res *collagetest.Response) string {
	t.Helper()
	m := generatedRE.FindStringSubmatch(res.Body)
	if m == nil {
		t.Fatalf("no result on the page:\n%s", res.Body)
	}
	return html.UnescapeString(m[1])
}

func TestGenerate(t *testing.T) {
	c, _ := loggedIn(t)
	page := c.Get("/generate").WantStatus(http.StatusOK)
	wantNoBody(t, page, `id="generated"`)

	res := post(c, "/generate", page.CSRFToken(), url.Values{"length": {"20"}}).WantStatus(http.StatusOK)
	if got := generated(t, res); utf8.RuneCountInString(got) != 20 {
		t.Fatalf("generated %q, want 20 runes", got)
	}

	res = post(c, "/generate", page.CSRFToken(), url.Values{"source": {"password"}}).WantStatus(http.StatusOK)
	if got := generated(t, res); utf8.RuneCountInString(got) != 8 {
		t.Fatalf("alternate %q, want 8 runes", got)
	}

	res = post(c, "/generate", page.CSRFToken(), url.Values{"length": {"many"}}).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Length must be a number.")
}

// The generator's forms post with fetch and get only the result fragment
// back, so the page (and what is typed in it) is not sent again. A plain
// post still answers with the whole page and keeps what was typed.
func TestGenerateLive(t *testing.T) {
	c, _ := loggedIn(t)
	page := c.Get("/generate").WantStatus(http.StatusOK)
	wantBody(t, page, `data-live="#generator-result"`)

	live := func(v url.Values) *collagetest.Response {
		req := c.Request(http.MethodPost, "/generate", strings.NewReader(v.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-CSRF-Token", page.CSRFToken())
		req.Header.Set(collage.FetchHeader, "1")
		return c.Do(req)
	}

	res := live(url.Values{"source": {"password"}}).WantStatus(http.StatusOK)
	if !strings.HasPrefix(strings.TrimSpace(res.Body), `<div id="generator-result"`) {
		t.Fatalf("live answer is not the result fragment alone:\n%s", res.Body)
	}
	wantNoBody(t, res, "<html")
	wantNoBody(t, res, "Look-alike version")
	if got := generated(t, res); utf8.RuneCountInString(got) != 8 {
		t.Fatalf("alternate %q, want 8 runes", got)
	}

	res = live(url.Values{"length": {"many"}}).WantStatus(http.StatusUnprocessableEntity)
	wantBody(t, res, "Length must be a number.")
	wantNoBody(t, res, "<html")

	// Without fetch: the whole page, with the typed word kept in its field.
	res = post(c, "/generate", page.CSRFToken(), url.Values{"source": {"hello"}}).WantStatus(http.StatusOK)
	wantBody(t, res, `name="source" value="hello"`)
}
