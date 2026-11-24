package session

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"passenger-go/internal/vault"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func setup() (*Manager, *clock) {
	c := &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return NewManager(15*time.Minute, 12*time.Hour, c.now), c
}

func testDEK(b byte) vault.DEK {
	var d vault.DEK
	for i := range d {
		d[i] = b
	}
	return d
}

func TestSlidingIdle(t *testing.T) {
	m, c := setup()
	s, err := m.Create(testDEK(1))
	if err != nil {
		t.Fatal(err)
	}
	c.advance(14 * time.Minute)
	if _, ok := m.Touch(s.Token); !ok {
		t.Fatal("touch at +14m should be ok")
	}
	c.advance(14 * time.Minute)
	if _, ok := m.Touch(s.Token); !ok {
		t.Fatal("touch at +28m should be ok (sliding)")
	}
	c.advance(16 * time.Minute)
	if _, ok := m.Touch(s.Token); ok {
		t.Fatal("idle exceeded should fail")
	}
	if _, ok := m.Peek(s.Token); ok {
		t.Fatal("session should be gone")
	}
}

func TestAbsoluteMax(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	for i := 0; i < 72; i++ { // touch every 10m up to +12h
		c.advance(10 * time.Minute)
		if _, ok := m.Touch(s.Token); !ok {
			t.Fatalf("unexpected expiry at step %d", i)
		}
	}
	c.advance(time.Minute) // +12h01m
	if _, ok := m.Touch(s.Token); ok {
		t.Fatal("absolute max exceeded should fail")
	}
}

func TestPeekDoesNotExtend(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	c.advance(10 * time.Minute)
	if _, ok := m.Peek(s.Token); !ok {
		t.Fatal("peek should be ok")
	}
	c.advance(6 * time.Minute)
	if _, ok := m.Touch(s.Token); ok {
		t.Fatal("touch at +16m should fail after peek-only")
	}
}

func TestExpiresAtIdleBound(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	if got := m.ExpiresAt(s); !got.Equal(c.t.Add(15 * time.Minute)) {
		t.Fatalf("idle bound: %v", got)
	}
}

func TestExpiresAtAbsoluteBound(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	created := c.t
	for i := 0; i < 71; i++ {
		c.advance(10 * time.Minute)
		m.Touch(s.Token)
	}
	c.advance(5 * time.Minute) // +11h55m
	cur, ok := m.Touch(s.Token)
	if !ok {
		t.Fatal("expected valid")
	}
	if got := m.ExpiresAt(cur); !got.Equal(created.Add(12 * time.Hour)) {
		t.Fatalf("absolute bound: %v", got)
	}
}

func TestRotate(t *testing.T) {
	m, _ := setup()
	a, _ := m.Create(testDEK(7))
	b, _ := m.Create(testDEK(9))
	n, err := m.Rotate(a.Token)
	if err != nil {
		t.Fatal(err)
	}
	if n.Token == a.Token {
		t.Fatal("token must change")
	}
	if _, ok := m.Peek(a.Token); ok {
		t.Fatal("old token valid")
	}
	if _, ok := m.Peek(b.Token); ok {
		t.Fatal("other session valid")
	}
	got, ok := m.Peek(n.Token)
	if !ok || got.DEK != testDEK(7) {
		t.Fatal("rotated session missing or DEK differs")
	}
	if _, err := m.Rotate("nope"); err == nil {
		t.Fatal("rotate of unknown token should error")
	}
}

func TestExpiredDEKZeroed(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(5))
	m.mu.Lock()
	internal := m.sessions[s.Token]
	m.mu.Unlock()
	c.advance(16 * time.Minute)
	if _, ok := m.Touch(s.Token); ok {
		t.Fatal("should be expired")
	}
	if internal.DEK != (vault.DEK{}) {
		t.Fatal("DEK not zeroed")
	}
}

func TestSweepAndDelete(t *testing.T) {
	m, c := setup()
	a, _ := m.Create(testDEK(1))
	c.advance(10 * time.Minute)
	b, _ := m.Create(testDEK(2))
	c.advance(6 * time.Minute)
	m.Sweep()
	m.mu.Lock()
	_, hasA := m.sessions[a.Token]
	_, hasB := m.sessions[b.Token]
	m.mu.Unlock()
	if hasA || !hasB {
		t.Fatalf("sweep wrong: a=%v b=%v", hasA, hasB)
	}
	m.Delete(b.Token)
	if _, ok := m.Peek(b.Token); ok {
		t.Fatal("deleted session valid")
	}
	m.Create(testDEK(3))
	m.DeleteAll()
	if len(m.sessions) != 0 {
		t.Fatal("DeleteAll left sessions")
	}
}

func TestCopyIsolated(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(4))
	cp, _ := m.Touch(s.Token)
	c.advance(20 * time.Minute)
	m.Sweep()
	if cp.DEK != testDEK(4) {
		t.Fatal("returned copy was wiped by sweep")
	}
}

func TestMiddleware(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	var gotSess bool
	var gotHad bool
	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, gotSess = FromContext(r.Context())
		gotHad = HadCookie(r.Context())
	}))
	do := func(path, token string) {
		r := httptest.NewRequest("GET", path, nil)
		if token != "" {
			r.AddCookie(&http.Cookie{Name: CookieName, Value: token})
		}
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
	do("/", "")
	if gotSess || gotHad {
		t.Fatal("no cookie: want false,false")
	}
	do("/", "stale")
	if gotSess || !gotHad {
		t.Fatal("stale cookie: want false,true")
	}
	do("/", s.Token)
	if !gotSess || gotHad {
		t.Fatal("valid cookie: want true,false")
	}
	// /session peeks, /session/extend touches
	c.advance(10 * time.Minute)
	do("/session", s.Token)
	c.advance(6 * time.Minute)
	if _, ok := m.Peek(s.Token); ok {
		t.Fatal("/session must not extend")
	}
	s2, _ := m.Create(testDEK(2))
	c.advance(10 * time.Minute)
	do("/session/extend", s2.Token)
	c.advance(10 * time.Minute)
	if _, ok := m.Peek(s2.Token); !ok {
		t.Fatal("/session/extend must extend")
	}
}

func TestCookies(t *testing.T) {
	m, _ := setup()
	s, _ := m.Create(testDEK(1))
	for _, tlsOn := range []bool{false, true} {
		r := httptest.NewRequest("GET", "/", nil)
		if tlsOn {
			r.TLS = &tls.ConnectionState{}
		}
		h := Cookie(r, s.Token).String()
		for _, want := range []string{CookieName + "=" + s.Token, "HttpOnly", "SameSite=Strict", "Path=/"} {
			if !strings.Contains(h, want) {
				t.Fatalf("missing %q in %q", want, h)
			}
		}
		if strings.Contains(h, "Secure") != tlsOn {
			t.Fatalf("Secure mismatch (tls=%v): %q", tlsOn, h)
		}
		h = ExpiredCookie(r).String()
		if !strings.Contains(h, "Max-Age=0") && !strings.Contains(h, "Max-Age=-1") {
			t.Fatalf("clear cookie not expiring: %q", h)
		}
	}
}

func TestMiddlewareSkipsStatic(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	var gotSess bool
	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, gotSess = FromContext(r.Context())
	}))
	c.advance(10 * time.Minute)
	r := httptest.NewRequest("GET", "/static/app.css", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: s.Token})
	h.ServeHTTP(httptest.NewRecorder(), r)
	if gotSess {
		t.Fatal("/static/ request must not resolve the session")
	}
	c.advance(6 * time.Minute)
	if _, ok := m.Peek(s.Token); ok {
		t.Fatal("/static/ request must not extend the session")
	}
}

// The request's copy of the session's key is wiped once the request is
// served; the stored session keeps its own.
func TestMiddlewareWipesTheRequestsDEK(t *testing.T) {
	for _, path := range []string{"/", "/session"} {
		m, _ := setup()
		s, _ := m.Create(testDEK(7))
		var seen *Session
		h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen, _ = FromContext(r.Context())
			if seen.DEK != testDEK(7) {
				t.Errorf("%s: handler sees DEK %v", path, seen.DEK)
			}
		}))
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: CookieName, Value: s.Token})
		h.ServeHTTP(httptest.NewRecorder(), r)
		if seen == nil {
			t.Fatalf("%s: no session in the request", path)
		}
		if seen.DEK != (vault.DEK{}) {
			t.Fatalf("%s: request's DEK copy not wiped", path)
		}
		if stored, ok := m.Peek(s.Token); !ok || stored.DEK != testDEK(7) {
			t.Fatalf("%s: stored session lost its DEK", path)
		}
	}
}
