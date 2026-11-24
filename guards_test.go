package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collagetest"

	"passenger-go/internal/session"
)

// forged sends target with a session cookie the site never signed.
func forged(c *collagetest.Client, method, target string) *collagetest.Response {
	req := c.Request(method, target, nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "stale"})
	return c.Do(req)
}

// guards.SignedIn, through the pages and actions it guards (it was
// web.RequireSession, tested on its own).
func TestSignedInGuard(t *testing.T) {
	c, env := newTestApp(t)
	wantLocation(t, c.Get("/accounts/1").WantStatus(http.StatusSeeOther), "/setup")

	setUp(t, c, env, testPass)
	g := collagetest.New(t, env.App.Handler())
	wantLocation(t, g.Get("/accounts/1").WantStatus(http.StatusSeeOther), "/login?next=%2Faccounts%2F1")
	wantLocation(t, g.Get("/accounts/1?q=a%20b").WantStatus(http.StatusSeeOther), "/login?next=%2Faccounts%2F1%3Fq%3Da%2520b")

	// A cookie the site did not sign is no session.
	wantLocation(t, forged(g, http.MethodGet, "/accounts/1").WantStatus(http.StatusSeeOther), "/login?next=%2Faccounts%2F1")
	wantLocation(t, forged(g, http.MethodGet, "/").WantStatus(http.StatusSeeOther), "/login")

	// A live session passes (to the page's own 404 for an unknown account).
	c.Get("/accounts/1").WantStatus(http.StatusNotFound)

	// A signed cookie whose DEK has expired is no session either.
	env.Clock.Advance(16 * time.Minute)
	wantLocation(t, c.Get("/accounts/1").WantStatus(http.StatusSeeOther), "/login?next=%2Faccounts%2F1")
	wantLocation(t, c.Get("/").WantStatus(http.StatusSeeOther), "/login")
}

// next only for a GET or HEAD: after login the browser would GET it.
func TestSignedInGuardNextOnlyForReads(t *testing.T) {
	c, env := newTestApp(t)
	setUp(t, c, env, testPass)
	g := collagetest.New(t, env.App.Handler())
	for _, tc := range []struct {
		method, target string
		forge          bool
		want           string
	}{
		{http.MethodPost, "/logout", false, "/login"},
		{http.MethodPost, "/logout", true, "/login"},
		{http.MethodPost, "/accounts/new", false, "/login"},
		{http.MethodHead, "/generate", false, "/login?next=%2Fgenerate"},
		{http.MethodGet, "/generate", true, "/login?next=%2Fgenerate"},
	} {
		var res *collagetest.Response
		if tc.forge {
			res = forged(g, tc.method, tc.target)
		} else {
			res = g.Do(g.Request(tc.method, tc.target, nil))
		}
		if res.Status != http.StatusSeeOther || res.Location() != tc.want {
			t.Errorf("%s %s forged=%v: %d %q, want 303 %q", tc.method, tc.target, tc.forge, res.Status, res.Location(), tc.want)
		}
	}
}

// guards.Guest: the login and recovery pages, for readers without a session.
func TestGuestGuard(t *testing.T) {
	c, env := newTestApp(t)
	wantLocation(t, c.Get("/login").WantStatus(http.StatusSeeOther), "/setup")

	setUp(t, c, env, testPass)
	g := collagetest.New(t, env.App.Handler())
	g.Get("/login").WantStatus(http.StatusOK)
	g.Get("/recover").WantStatus(http.StatusOK)
	forged(g, http.MethodGet, "/login").WantStatus(http.StatusOK)

	// Signed in: home, or to a next on this site.
	wantLocation(t, c.Get("/login").WantStatus(http.StatusSeeOther), "/")
	wantLocation(t, c.Get("/recover").WantStatus(http.StatusSeeOther), "/")
	wantLocation(t, c.Get("/login?next=%2Fgenerate").WantStatus(http.StatusSeeOther), "/generate")
	wantLocation(t, c.Get("/login?next="+"%2F%2Fevil.com").WantStatus(http.StatusSeeOther), "/")
	// One that only decodes to an off-site path, which collage.SafeRedirect
	// alone lets through.
	wantLocation(t, c.Get("/login?next=%2F%252F%252Fevil.com").WantStatus(http.StatusSeeOther), "/")
}

// guards.Uninitialized and guards.PendingSetup: setup is there only until the
// vault is.
func TestSetupGuards(t *testing.T) {
	c, env := newTestApp(t)
	c.Get("/setup").WantStatus(http.StatusOK)
	wantLocation(t, c.Get("/setup/confirm").WantStatus(http.StatusSeeOther), "/setup")

	setUp(t, c, env, testPass)
	g := collagetest.New(t, env.App.Handler())
	wantLocation(t, g.Get("/setup").WantStatus(http.StatusSeeOther), "/login")
	wantLocation(t, g.Get("/setup/confirm").WantStatus(http.StatusSeeOther), "/login")
}
