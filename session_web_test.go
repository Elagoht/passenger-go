package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collagetest"
)

func expiresAt(t *testing.T, res *collagetest.Response) int64 {
	t.Helper()
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q", ct)
	}
	var got struct {
		ExpiresAt int64 `json:"expiresAt"`
	}
	if err := json.Unmarshal([]byte(res.Body), &got); err != nil {
		t.Fatalf("body %q: %v", res.Body, err)
	}
	return got.ExpiresAt
}

func extend(c *collagetest.Client, token string) *collagetest.Response {
	req := c.Request(http.MethodPost, "/session/extend", nil)
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}
	return c.Do(req)
}

func TestSessionStatusDoesNotExtend(t *testing.T) {
	c, env := loggedIn(t)
	start := env.Clock.Now()
	want := start.Add(15 * time.Minute).UnixMilli()

	if got := expiresAt(t, c.Get("/session").WantStatus(http.StatusOK)); got != want {
		t.Fatalf("expiresAt = %d, want %d", got, want)
	}
	env.Clock.Advance(10 * time.Minute)
	if got := expiresAt(t, c.Get("/session").WantStatus(http.StatusOK)); got != want {
		t.Fatalf("expiresAt after polling = %d, want unchanged %d", got, want)
	}
}

func TestSessionExtend(t *testing.T) {
	c, env := loggedIn(t)
	page := c.Get("/").WantStatus(http.StatusOK)
	token := page.CSRFToken()
	env.Clock.Advance(10 * time.Minute)
	want := env.Clock.Now().Add(15 * time.Minute).UnixMilli()

	if got := expiresAt(t, extend(c, token).WantStatus(http.StatusOK)); got != want {
		t.Fatalf("expiresAt = %d, want %d", got, want)
	}
	if got := expiresAt(t, c.Get("/session").WantStatus(http.StatusOK)); got != want {
		t.Fatalf("/session after extend = %d, want %d", got, want)
	}
	extend(c, "").WantStatus(http.StatusForbidden)
}

func TestSessionEndpointsAnswer401WithoutSession(t *testing.T) {
	c, env := loggedIn(t)
	token := c.Get("/").CSRFToken()
	env.Clock.Advance(16 * time.Minute)

	res := c.Get("/session").WantStatus(http.StatusUnauthorized)
	if res.Body != `{"expiresAt":0}` {
		t.Fatalf("body = %q", res.Body)
	}
	extend(c, token).WantStatus(http.StatusUnauthorized)

	guest := collagetest.New(t, env.App.Handler())
	guest.Get("/session").WantStatus(http.StatusUnauthorized)
}

func TestLayoutCarriesSessionExpiry(t *testing.T) {
	c, env := loggedIn(t)
	want := env.Clock.Now().Add(15 * time.Minute).UnixMilli()
	wantBody(t, c.Get("/").WantStatus(http.StatusOK),
		`<body class="app" data-session-expires="`+strconv.FormatInt(want, 10)+`"`)
	wantBody(t, c.Get("/accounts/new").WantStatus(http.StatusOK), "data-track")
}
