package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	cookie "github.com/Elagoht/collage-session"

	"passenger-go/internal/session"
)

// cookieSession is what VaultSession needs of the elagoht/session plugin's
// session: the handle it holds under cookie.UserKey.
type cookieSession interface {
	Get(key string) string
	Set(key, value string) error
}

// pluginSession returns the plugin's session of the request ctx belongs to, or
// nil without the plugin. A variable so tests can stand in for the plugin.
var pluginSession = func(ctx context.Context) cookieSession {
	if s := cookie.FromContext(ctx); s != nil {
		return s
	}
	return nil
}

// VaultSession joins the two halves of a signed-in session. The
// elagoht/session plugin, outside it, has read the cookie, which holds only a
// handle under cookie.UserKey; the manager maps the handle to the DEK (see
// internal/session for why the key cannot live in the cookie).
//
//   - A live handle puts the request's copy of the session, DEK included, in
//     the context (session.FromContext), and the copy is wiped once the
//     request is served. The guards decide on that session alone, never on
//     the plugin's UserKey: a handle the manager no longer has (expired,
//     signed out, rotated or ended by a recovery) grants nothing.
//   - A dead handle is never cleared from the cookie here. A request carrying
//     the old handle may be answered after a passphrase change rotated it, and
//     its clearing cookie, landing last, would delete the rotated one and sign
//     the owner out. Instead, when the response is about to be written and the
//     handler left the handle alone, a handle the manager no longer has marks
//     the response dead (see SessionCookie), which then carries no session
//     cookie at all: neither a clear nor the old handle again.
//   - Every request but /static/ and /session is the reader's activity: it
//     slides the manager's idle window and, by setting the handle again when
//     the response is written (only if the handle is still live and the
//     handler did not change it), makes the plugin rewrite its cookie with a
//     fresh idle limit, so both end at the same moment (to the second; the
//     plugin counts whole seconds). /session only peeks and /static/ is
//     skipped; SessionCookie keeps the plugin from sliding on those.
//
// The handle is set again when the response is written, not when the request
// arrives: a request admitted before a rotation and answered after it then
// finds its handle gone and sends nothing, rather than the revoked handle.
// VaultSession sits inside the plugin, so its writer runs before the plugin's
// commits the cookie.
func VaultSession(m *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cs := pluginSession(r.Context())
			if cs == nil || isStatic(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			handle := cs.Get(cookie.UserKey)
			if handle == "" {
				next.ServeHTTP(w, r)
				return
			}
			active := r.URL.Path != statusPath
			var s *session.Session
			var ok bool
			if active {
				s, ok = m.Touch(handle)
			} else {
				s, ok = m.Peek(handle)
			}
			vw := &vaultWriter{
				ResponseWriter: w, m: m, cs: cs, handle: handle,
				refresh: active && ok, dead: deadFlag(r.Context()),
			}
			// The plugin also commits after this returns, when nothing was
			// written.
			defer vw.settle()
			if !ok {
				next.ServeHTTP(vw, r)
				return
			}
			defer s.DEK.Wipe()
			next.ServeHTTP(vw, r.WithContext(session.WithSession(r.Context(), s)))
		})
	}
}

func isStatic(path string) bool { return strings.HasPrefix(path, "/static/") }

// vaultWriter settles, just before the plugin it sits inside commits its
// cookie, what that cookie may say.
type vaultWriter struct {
	http.ResponseWriter
	m       *session.Manager
	cs      cookieSession
	handle  string       // the handle the request arrived with
	refresh bool         // set the handle again: an active request on a live handle
	dead    *atomic.Bool // SessionCookie's flag; nil without SessionCookie
	done    bool
}

// settle runs once. A handler that changed the handle (sign-in, rotation,
// sign-out) keeps its change. Otherwise a handle the manager no longer has
// marks the response dead, and a live one is set again on an active request.
// A Set that fails (the response already committed) keeps the cookie as it
// was, which ends no later than the manager's session.
func (w *vaultWriter) settle() {
	if w.done {
		return
	}
	w.done = true
	if w.cs.Get(cookie.UserKey) != w.handle {
		return
	}
	if _, ok := w.m.Peek(w.handle); !ok {
		if w.dead != nil {
			w.dead.Store(true)
		}
		return
	}
	if w.refresh {
		_ = w.cs.Set(cookie.UserKey, w.handle)
	}
}

func (w *vaultWriter) WriteHeader(status int) {
	w.settle()
	w.ResponseWriter.WriteHeader(status)
}

func (w *vaultWriter) Write(b []byte) (int, error) {
	w.settle()
	return w.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer when it can flush.
func (w *vaultWriter) Flush() {
	w.settle()
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *vaultWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
