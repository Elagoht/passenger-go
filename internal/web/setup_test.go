package web

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

func TestSetupCode(t *testing.T) {
	c, err := NewSetupCode()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9]{8}$`).MatchString(c.String()) {
		t.Fatalf("code = %q, want 8 digits", c.String())
	}
	if !c.Check(c.String()) {
		t.Fatal("Check(own code) = false")
	}
	if !c.Check(" " + c.String() + " ") {
		t.Fatal("Check should ignore surrounding space")
	}
	for _, bad := range []string{"", "1234567", c.String() + "0", "abcdefgh"} {
		if c.Check(bad) {
			t.Errorf("Check(%q) = true", bad)
		}
	}
}

type testClock struct{ t time.Time }

func (c *testClock) now() time.Time { return c.t }

func TestPendingSetup(t *testing.T) {
	clk := &testClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	p := NewPendingSetup(clk.now)
	var dek vault.DEK
	dek[0] = 7
	row := store.VaultRow{KDF: fastKDF}

	tok, err := p.Put(dek, row, "KEY-ONE")
	if err != nil {
		t.Fatal(err)
	}
	gotRow, rec, gotDEK, ok := p.Get(tok)
	if !ok || rec != "KEY-ONE" || gotDEK != dek || gotRow.KDF != fastKDF {
		t.Fatalf("Get = %+v %q %v %v", gotRow, rec, gotDEK[0], ok)
	}
	if _, _, _, ok := p.Get("wrong"); ok {
		t.Fatal("Get(wrong token) ok")
	}
	if _, _, _, ok := p.Get(""); ok {
		t.Fatal("Get(empty) ok")
	}

	// A new Put replaces the old one.
	tok2, err := p.Put(dek, row, "KEY-TWO")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := p.Get(tok); ok {
		t.Fatal("old token still valid after Put")
	}
	if _, rec, _, ok := p.Get(tok2); !ok || rec != "KEY-TWO" {
		t.Fatal("new token not valid")
	}

	// Expires after 15 minutes.
	clk.t = clk.t.Add(15*time.Minute - time.Second)
	if _, _, _, ok := p.Get(tok2); !ok {
		t.Fatal("expired too early")
	}
	clk.t = clk.t.Add(2 * time.Second)
	if _, _, _, ok := p.Get(tok2); ok {
		t.Fatal("still valid after 15 minutes")
	}

	tok3, _ := p.Put(dek, row, "KEY-THREE")
	p.Clear()
	if _, _, _, ok := p.Get(tok3); ok {
		t.Fatal("valid after Clear")
	}
}

func TestPendingCookies(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	c := PendingCookie(r, "tok").String()
	for _, want := range []string{PendingCookieName + "=tok", "HttpOnly", "SameSite=Strict", "Path=/"} {
		if !strings.Contains(c, want) {
			t.Errorf("cookie %q lacks %q", c, want)
		}
	}
	if strings.Contains(c, "Secure") {
		t.Errorf("plain http cookie is Secure: %q", c)
	}
	if c := ClearPendingCookie(r).String(); !strings.Contains(c, "Max-Age=0") {
		t.Errorf("clear cookie %q does not expire", c)
	}
}

func TestRequirePendingSetup(t *testing.T) {
	d := newDeps(t)
	code, _ := NewSetupCode()
	d.SetupCode, d.Pending = code, NewPendingSetup(d.Now)

	wantRedirect(t, runGuard(t, d, d.RequirePendingSetup, "/setup/confirm", ""), "/setup")

	tok, err := d.Pending.Put(vault.DEK{}, store.VaultRow{}, "K")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/setup/confirm", nil)
	r.AddCookie(&http.Cookie{Name: PendingCookieName, Value: tok})
	if dec, err := d.RequirePendingSetup(r.Context(), r); err != nil || dec != nil {
		t.Fatalf("valid pending: %+v, %v", dec, err)
	}

	initVault(t, d)
	if dec, _ := d.RequirePendingSetup(r.Context(), r); dec == nil || dec.Location != "/login" {
		t.Fatalf("initialized: %+v, want /login", dec)
	}
}

func TestRetryMessageRoundsUp(t *testing.T) {
	for wait, want := range map[time.Duration]string{
		time.Second:             "Try again in 1 seconds",
		1500 * time.Millisecond: "Try again in 2 seconds",
		time.Millisecond:        "Try again in 1 seconds",
		4 * time.Second:         "Try again in 4 seconds",
	} {
		if got := RetryMessage(wait); !strings.Contains(got, want) {
			t.Errorf("RetryMessage(%v) = %q, want %q", wait, got, want)
		}
	}
}
