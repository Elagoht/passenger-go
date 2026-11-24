package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"passenger-go/internal/session"
)

// SessionCookie adjusts the cookie the elagoht/session plugin writes, in two
// ways the plugin has no option for. It must run outside the plugin, which
// writes its cookie to the writer it was handed: routes.go adds it before
// registering the plugin.
//
//   - The cookie lasts as long as the browser session, as the security model
//     wants: the plugin gives it a Max-Age (the idle limit), so closing the
//     browser would not end the session. Max-Age and Expires are taken off.
//     The limits still hold: the plugin signs the session's start and last
//     request into the cookie, and the DEK manager keeps the same limits.
//   - A file under /static/, and /session, which the expiry banner polls, are
//     not the reader's doing, and must not extend the session. The plugin
//     rewrites its cookie, pushing its idle limit back, on any request that
//     comes a quarter of IdleTimeout after the last rewrite, whatever the path;
//     the DEK manager is not extended by these requests (see VaultSession), so
//     the cookie is not refreshed on them either, or the two would disagree
//     about when the session ends.
//
// A Set-Cookie that clears the session is let through as it is, except on a
// response VaultSession marked dead: one to a request whose handle the manager
// no longer has, and that the handler left alone. That response carries no
// session cookie at all. Its request may hold a handle a passphrase change has
// just rotated, and anything it sets, the old handle or a clear, could land
// after the rotated cookie and sign the owner out.
func SessionCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cw := &cookieWriter{ResponseWriter: w, quiet: quietPath(r.URL.Path)}
		// The plugin also commits after the handler returns, when it wrote
		// nothing; net/http sends the header map after this returns.
		defer cw.fix()
		next.ServeHTTP(cw, r.WithContext(context.WithValue(r.Context(), deadKey{}, &cw.dead)))
	})
}

// deadKey is the context key of the flag VaultSession sets on a response
// whose handle is dead.
type deadKey struct{}

// deadFlag is the flag SessionCookie put in ctx, or nil without it.
func deadFlag(ctx context.Context) *atomic.Bool {
	f, _ := ctx.Value(deadKey{}).(*atomic.Bool)
	return f
}

// quietPath reports whether a request to path is not the reader's activity.
func quietPath(path string) bool {
	return strings.HasPrefix(path, "/static/") || path == statusPath
}

// statusPath is the expiry banner's GET /session: reading the session's end
// must not move it.
const statusPath = "/session"

type cookieWriter struct {
	http.ResponseWriter
	quiet bool
	dead  atomic.Bool // set by VaultSession: drop every session cookie
	done  bool
}

// fix rewrites, or on a quiet path drops, the Set-Cookie lines that set the
// session cookie; on a dead response it drops every one, clears included.
func (w *cookieWriter) fix() {
	if w.done {
		return
	}
	w.done = true
	h := w.Header()
	lines := h.Values("Set-Cookie")
	if len(lines) == 0 {
		return
	}
	h.Del("Set-Cookie")
	for _, line := range lines {
		c, err := http.ParseSetCookie(line)
		if err == nil && c.Name == session.CookieName && w.dead.Load() {
			continue
		}
		if err == nil && c.Name == session.CookieName && c.Value != "" && c.MaxAge >= 0 {
			if w.quiet {
				continue
			}
			c.MaxAge, c.Expires = 0, time.Time{}
			line = c.String()
		}
		h.Add("Set-Cookie", line)
	}
}

func (w *cookieWriter) WriteHeader(status int) {
	w.fix()
	w.ResponseWriter.WriteHeader(status)
}

func (w *cookieWriter) Write(b []byte) (int, error) {
	w.fix()
	return w.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer when it can flush.
func (w *cookieWriter) Flush() {
	w.fix()
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *cookieWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
