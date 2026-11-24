package middleware

import (
	"context"
	"net/http"
	"strings"

	cookie "github.com/Elagoht/collage-session"

	"passenger-go/internal/session"
)

// cookieSession is what VaultSession needs of the elagoht/session plugin's
// session: the handle it holds under cookie.UserKey.
type cookieSession interface {
	Get(key string) string
	Set(key, value string) error
	Clear()
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
//     request is served.
//   - A handle the manager no longer has — expired, signed out, rotated or
//     ended by a recovery — clears the plugin's session too, before any guard
//     reads it: the cookie itself cannot be revoked, the handle can.
//   - Every request but /static/ and /session is the reader's activity: it
//     slides the manager's idle window and, by setting the handle again, makes
//     the plugin rewrite its cookie with a fresh idle limit, so both end at the
//     same moment (to the second; the plugin counts whole seconds). /session
//     only peeks and /static/ is skipped; QuietSession keeps the plugin from
//     sliding on those.
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
			if !ok {
				cs.Clear()
				next.ServeHTTP(w, r)
				return
			}
			defer s.DEK.Wipe()
			if active {
				if err := cs.Set(cookie.UserKey, handle); err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(session.WithSession(r.Context(), s)))
		})
	}
}

func isStatic(path string) bool { return strings.HasPrefix(path, "/static/") }
