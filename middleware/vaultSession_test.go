package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	cookie "github.com/Elagoht/collage-session"

	"passenger-go/internal/session"
	"passenger-go/internal/vault"
)

// fakeCookie stands in for the session plugin's session.
type fakeCookie struct {
	data map[string]string
	sets int
}

func (f *fakeCookie) Get(key string) string { return f.data[key] }

func (f *fakeCookie) Set(key, value string) error {
	f.data[key] = value
	f.sets++
	return nil
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

// served is what one request through VaultSession left behind.
type served struct {
	cookie *fakeCookie
	seen   *session.Session // the session the handler saw
	dead   bool             // the flag SessionCookie reads
}

// serveWith sends a GET of path carrying handle (none when "") through
// VaultSession to handler, with SessionCookie's dead flag in the context.
func serveWith(m *session.Manager, path, handle string, handler func(http.ResponseWriter, *http.Request)) served {
	f := &fakeCookie{data: map[string]string{}}
	if handle != "" {
		f.data[cookie.UserKey] = handle
	}
	var out served
	var dead atomic.Bool
	h := VaultSession(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out.seen, _ = session.FromContext(r.Context())
		if handler != nil {
			handler(w, r)
		}
	}))
	r := httptest.NewRequest("GET", path, nil)
	ctx := context.WithValue(r.Context(), fakeCookieKey{}, f)
	ctx = context.WithValue(ctx, deadKey{}, &dead)
	h.ServeHTTP(httptest.NewRecorder(), r.WithContext(ctx))
	out.cookie, out.dead = f, dead.Load()
	return out
}

// serve is serveWith a handler that does nothing but look.
func serve(m *session.Manager, path, handle string) served {
	return serveWith(m, path, handle, nil)
}

func TestVaultSession(t *testing.T) {
	usePlugin(t)
	m, c := managerAt()
	s, _ := m.Create(testDEK(1))

	if res := serve(m, "/", ""); res.seen != nil || res.dead {
		t.Fatal("no handle: want no session, not dead")
	}

	// An unknown handle grants nothing, and is neither cleared nor set
	// again: the response is marked dead instead.
	res := serve(m, "/", "stale")
	if res.seen != nil || res.cookie.sets != 0 || res.cookie.data[cookie.UserKey] != "stale" || !res.dead {
		t.Fatalf("unknown handle: session %v, %d sets, handle %q, dead %v; want none, 0, kept, dead",
			res.seen, res.cookie.sets, res.cookie.data[cookie.UserKey], res.dead)
	}

	res = serve(m, "/", s.Handle)
	if res.seen == nil || res.dead {
		t.Fatal("live handle: want a session, not dead")
	}
	if res.cookie.sets != 1 || res.cookie.data[cookie.UserKey] != s.Handle {
		t.Fatalf("live handle: %d sets of %q, want the handle set again once", res.cookie.sets, res.cookie.data[cookie.UserKey])
	}

	// /session peeks and leaves the cookie alone; /session/extend touches.
	c.advance(10 * time.Minute)
	if res := serve(m, "/session", s.Handle); res.cookie.sets != 0 || res.dead {
		t.Fatalf("/session set the cookie %d times (dead %v)", res.cookie.sets, res.dead)
	}
	c.advance(6 * time.Minute)
	if _, ok := m.Peek(s.Handle); ok {
		t.Fatal("/session must not extend")
	}
	s2, _ := m.Create(testDEK(2))
	c.advance(10 * time.Minute)
	if res := serve(m, "/session/extend", s2.Handle); res.cookie.sets != 1 {
		t.Fatalf("/session/extend set the cookie %d times, want 1", res.cookie.sets)
	}
	c.advance(10 * time.Minute)
	if _, ok := m.Peek(s2.Handle); !ok {
		t.Fatal("/session/extend must extend")
	}
}

// An expired session's handle grants nothing and marks the response dead,
// and the manager's copy of the DEK is gone.
func TestVaultSessionExpired(t *testing.T) {
	usePlugin(t)
	m, c := managerAt()
	s, _ := m.Create(testDEK(5))
	c.advance(16 * time.Minute)
	res := serve(m, "/accounts/new", s.Handle)
	if res.seen != nil || res.cookie.sets != 0 || !res.dead {
		t.Fatalf("expired: session %v, %d sets, dead %v", res.seen, res.cookie.sets, res.dead)
	}
	if m.Len() != 0 {
		t.Fatalf("expired session still held: Len = %d", m.Len())
	}
}

// A request admitted before a rotation and answered after it finds its
// handle gone when the response is written: it does not set the revoked
// handle again, and marks the response dead.
func TestVaultSessionRotatedDuringTheRequest(t *testing.T) {
	usePlugin(t)
	m, _ := managerAt()
	s, _ := m.Create(testDEK(3))
	var rotated *session.Session
	res := serveWith(m, "/", s.Handle, func(w http.ResponseWriter, r *http.Request) {
		rotated, _ = m.Rotate(s.Handle) // another request's passphrase change
		w.WriteHeader(http.StatusOK)
	})
	if res.seen == nil {
		t.Fatal("the request was admitted with a live session")
	}
	if res.cookie.sets != 0 || !res.dead {
		t.Fatalf("%d sets, dead %v; want the old handle not set again, dead", res.cookie.sets, res.dead)
	}
	if _, ok := m.Peek(rotated.Handle); !ok {
		t.Fatal("the rotated session is gone")
	}
}

// A handler that changes the handle (sign-in, the rotation itself, sign-out)
// keeps its change: the old handle is not set over it, and the response is
// not dead.
func TestVaultSessionKeepsTheHandlersChange(t *testing.T) {
	usePlugin(t)
	m, _ := managerAt()
	for name, change := range map[string]func(*fakeCookie, string){
		"rotate": func(f *fakeCookie, handle string) {
			n, _ := m.Rotate(handle)
			_ = f.Set(cookie.UserKey, n.Handle)
		},
		"sign out": func(f *fakeCookie, handle string) {
			m.Delete(handle)
			_ = f.Set(cookie.UserKey, "")
		},
	} {
		s, _ := m.Create(testDEK(4))
		var want string
		res := serveWith(m, "/", s.Handle, func(w http.ResponseWriter, r *http.Request) {
			f := r.Context().Value(fakeCookieKey{}).(*fakeCookie)
			change(f, s.Handle)
			want = f.data[cookie.UserKey]
		})
		if res.dead || res.cookie.sets != 1 || res.cookie.data[cookie.UserKey] != want {
			t.Errorf("%s: dead %v, %d sets, handle %q; want not dead, the handler's one set kept",
				name, res.dead, res.cookie.sets, res.cookie.data[cookie.UserKey])
		}
	}

	// A sign-in over a dead handle is let through too.
	res := serveWith(m, "/login", "stale", func(w http.ResponseWriter, r *http.Request) {
		n, _ := m.Create(testDEK(6))
		_ = r.Context().Value(fakeCookieKey{}).(*fakeCookie).Set(cookie.UserKey, n.Handle)
	})
	if res.dead {
		t.Error("sign-in over a dead handle marked the response dead")
	}
}

func TestVaultSessionSkipsStatic(t *testing.T) {
	usePlugin(t)
	m, c := managerAt()
	s, _ := m.Create(testDEK(1))
	c.advance(10 * time.Minute)
	if res := serve(m, "/static/app.css", s.Handle); res.seen != nil || res.cookie.sets != 0 || res.dead {
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
