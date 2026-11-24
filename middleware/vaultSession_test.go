package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cookie "github.com/Elagoht/collage-session"

	"passenger-go/internal/session"
	"passenger-go/internal/vault"
)

// fakeCookie stands in for the session plugin's session.
type fakeCookie struct {
	data    map[string]string
	sets    int
	cleared bool
}

func (f *fakeCookie) Get(key string) string { return f.data[key] }
func (f *fakeCookie) Set(key, value string) error {
	f.data[key] = value
	f.sets++
	return nil
}
func (f *fakeCookie) Clear() {
	f.data = map[string]string{}
	f.cleared = true
}

type fakeCookieKey struct{}

// usePlugin makes VaultSession read the fake from the request's context for
// the test's duration.
func usePlugin(t *testing.T) {
	t.Helper()
	prev := pluginSession
	pluginSession = func(ctx context.Context) cookieSession {
		if f, ok := ctx.Value(fakeCookieKey{}).(*fakeCookie); ok {
			return f
		}
		return nil
	}
	t.Cleanup(func() { pluginSession = prev })
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func managerAt() (*session.Manager, *clock) {
	c := &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return session.NewManager(15*time.Minute, 12*time.Hour, c.now), c
}

func testDEK(b byte) vault.DEK {
	var d vault.DEK
	for i := range d {
		d[i] = b
	}
	return d
}

// serve sends a GET of path carrying handle (none when "") through
// VaultSession and returns the plugin's fake and what the handler saw.
func serve(m *session.Manager, path, handle string) (*fakeCookie, *session.Session) {
	f := &fakeCookie{data: map[string]string{}}
	if handle != "" {
		f.data[cookie.UserKey] = handle
	}
	var seen *session.Session
	h := VaultSession(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = session.FromContext(r.Context())
	}))
	r := httptest.NewRequest("GET", path, nil)
	h.ServeHTTP(httptest.NewRecorder(), r.WithContext(context.WithValue(r.Context(), fakeCookieKey{}, f)))
	return f, seen
}

func TestVaultSession(t *testing.T) {
	usePlugin(t)
	m, c := managerAt()
	s, _ := m.Create(testDEK(1))

	if _, seen := serve(m, "/", ""); seen != nil {
		t.Fatal("no handle: want no session")
	}
	f, seen := serve(m, "/", "stale")
	if seen != nil || !f.cleared {
		t.Fatalf("unknown handle: session %v, cleared %v; want none, cleared", seen, f.cleared)
	}
	f, seen = serve(m, "/", s.Handle)
	if seen == nil || f.cleared {
		t.Fatal("live handle: want a session, not cleared")
	}
	if f.sets != 1 || f.data[cookie.UserKey] != s.Handle {
		t.Fatalf("live handle: %d sets of %q, want the handle set again once", f.sets, f.data[cookie.UserKey])
	}

	// /session peeks and leaves the cookie alone; /session/extend touches.
	c.advance(10 * time.Minute)
	if f, _ := serve(m, "/session", s.Handle); f.sets != 0 {
		t.Fatalf("/session set the cookie %d times", f.sets)
	}
	c.advance(6 * time.Minute)
	if _, ok := m.Peek(s.Handle); ok {
		t.Fatal("/session must not extend")
	}
	s2, _ := m.Create(testDEK(2))
	c.advance(10 * time.Minute)
	if f, _ := serve(m, "/session/extend", s2.Handle); f.sets != 1 {
		t.Fatalf("/session/extend set the cookie %d times, want 1", f.sets)
	}
	c.advance(10 * time.Minute)
	if _, ok := m.Peek(s2.Handle); !ok {
		t.Fatal("/session/extend must extend")
	}
}

// An expired session's handle clears the cookie, and the manager's copy of
// the DEK is gone.
func TestVaultSessionExpired(t *testing.T) {
	usePlugin(t)
	m, c := managerAt()
	s, _ := m.Create(testDEK(5))
	c.advance(16 * time.Minute)
	f, seen := serve(m, "/accounts/new", s.Handle)
	if seen != nil || !f.cleared {
		t.Fatalf("expired: session %v, cleared %v", seen, f.cleared)
	}
	if m.Len() != 0 {
		t.Fatalf("expired session still held: Len = %d", m.Len())
	}
}

func TestVaultSessionSkipsStatic(t *testing.T) {
	usePlugin(t)
	m, c := managerAt()
	s, _ := m.Create(testDEK(1))
	c.advance(10 * time.Minute)
	f, seen := serve(m, "/static/app.css", s.Handle)
	if seen != nil || f.sets != 0 {
		t.Fatal("/static/ request must not resolve the session or set the cookie")
	}
	c.advance(6 * time.Minute)
	if _, ok := m.Peek(s.Handle); ok {
		t.Fatal("/static/ request must not extend the session")
	}
}

// The request's copy of the session's key is wiped once the request is
// served; the stored session keeps its own.
func TestVaultSessionWipesTheRequestsDEK(t *testing.T) {
	usePlugin(t)
	for _, path := range []string{"/", "/session"} {
		m, _ := managerAt()
		s, _ := m.Create(testDEK(7))
		var during vault.DEK
		f := &fakeCookie{data: map[string]string{cookie.UserKey: s.Handle}}
		var seen *session.Session
		h := VaultSession(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen, _ = session.FromContext(r.Context())
			during = seen.DEK
		}))
		r := httptest.NewRequest("GET", path, nil)
		h.ServeHTTP(httptest.NewRecorder(), r.WithContext(context.WithValue(r.Context(), fakeCookieKey{}, f)))
		if during != testDEK(7) {
			t.Fatalf("%s: handler saw DEK %v", path, during[0])
		}
		if seen.DEK != (vault.DEK{}) {
			t.Fatalf("%s: request's DEK copy not wiped", path)
		}
		if stored, ok := m.Peek(s.Handle); !ok || stored.DEK != testDEK(7) {
			t.Fatalf("%s: stored session lost its DEK", path)
		}
	}
}

// Without the plugin there is no session to read, and nothing happens.
func TestVaultSessionWithoutThePlugin(t *testing.T) {
	m, _ := managerAt()
	var seen bool
	h := VaultSession(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, seen = session.FromContext(r.Context())
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if seen {
		t.Fatal("a session without the plugin")
	}
}
