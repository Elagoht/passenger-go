package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"regexp"
	"strings"
	"testing"
	"time"

	"passenger-go/internal/session"
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
		time.Second:             "Try again in 1 second.",
		1500 * time.Millisecond: "Try again in 2 seconds",
		time.Millisecond:        "Try again in 1 second.",
		4 * time.Second:         "Try again in 4 seconds",
	} {
		if got := RetryMessage(wait); !strings.Contains(got, want) {
			t.Errorf("RetryMessage(%v) = %q, want %q", wait, got, want)
		}
	}
}

func TestRequireSessionNextOnlyForReads(t *testing.T) {
	d := newDeps(t)
	initVault(t, d)
	h := d.Sessions.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dec, err := d.RequireSession(r.Context(), r)
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Location", dec.Location)
	}))
	for _, tc := range []struct{ method, cookie, want string }{
		{"POST", "", "/login"},
		{"POST", "stale", "/login"},
		{"HEAD", "", "/login?next=%2Fx"},
		{"GET", "stale", "/login?next=%2Fx"},
	} {
		r := httptest.NewRequest(tc.method, "/x", nil)
		if tc.cookie != "" {
			r.AddCookie(&http.Cookie{Name: session.CookieName, Value: tc.cookie})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if got := rec.Header().Get("Location"); got != tc.want {
			t.Errorf("%s cookie=%q: %q, want %q", tc.method, tc.cookie, got, tc.want)
		}
	}
}

func TestKDFSlotsHonourContext(t *testing.T) {
	for i := 0; i < cap(kdfSlots); i++ {
		if err := acquireKDF(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for i := 0; i < cap(kdfSlots); i++ {
			releaseKDF()
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Unwrap(ctx, "x", vault.Wrapped{}, fastKDF, vault.AADPass); !errors.Is(err, context.Canceled) {
		t.Fatalf("Unwrap with full slots and cancelled ctx = %v, want context.Canceled", err)
	}
	if _, err := Wrap(ctx, "x", vault.DEK{}, fastKDF, vault.AADPass); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wrap with full slots and cancelled ctx = %v, want context.Canceled", err)
	}
}

func TestThrottleHonoursContext(t *testing.T) {
	d := newDeps(t)
	for i := 0; i <= 100; i++ { // past the global ceiling: a 2s delay
		d.Limiter.Begin(netip.AddrFrom4([4]byte{10, 0, byte(i >> 8), byte(i)}))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("POST", "/login", nil).WithContext(ctx)
	start := time.Now()
	if _, _, err := d.Throttle(r); !errors.Is(err, context.Canceled) {
		t.Fatalf("Throttle = %v, want context.Canceled", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("Throttle slept through a cancelled context")
	}
}
