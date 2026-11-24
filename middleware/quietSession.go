package middleware

import (
	"net/http"
	"strings"

	"passenger-go/internal/session"
)

// QuietSession keeps the requests that are not the reader's doing from
// extending the session cookie: a file under /static/, and /session, which the
// expiry banner polls. The elagoht/session plugin rewrites its cookie, pushing
// its idle limit back, on any request that comes a quarter of IdleTimeout after
// the last rewrite, whatever the path; the DEK manager is not extended by
// these requests (see VaultSession), so the cookie must not be either, or the
// two would disagree about when the session ends.
//
// It drops the plugin's refreshing Set-Cookie from those responses and lets one
// that clears the cookie through. It must run outside the plugin, which writes
// its cookie to the writer it was handed: routes.go adds it before registering
// the plugin.
func QuietSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !quietPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		qw := &quietWriter{ResponseWriter: w}
		// The plugin also commits after the handler returns, when it wrote
		// nothing; net/http sends the header map after this returns.
		defer qw.strip()
		next.ServeHTTP(qw, r)
	})
}

// quietPath reports whether a request to path is not the reader's activity.
func quietPath(path string) bool {
	return strings.HasPrefix(path, "/static/") || path == statusPath
}

// statusPath is the expiry banner's GET /session: reading the session's end
// must not move it.
const statusPath = "/session"

type quietWriter struct {
	http.ResponseWriter
	done bool
}

// strip removes the Set-Cookie lines that refresh the session cookie.
func (w *quietWriter) strip() {
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
		if err == nil && c.Name == session.CookieName && c.Value != "" && c.MaxAge >= 0 {
			continue
		}
		h.Add("Set-Cookie", line)
	}
}

func (w *quietWriter) WriteHeader(status int) {
	w.strip()
	w.ResponseWriter.WriteHeader(status)
}

func (w *quietWriter) Write(b []byte) (int, error) {
	w.strip()
	return w.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer when it can flush.
func (w *quietWriter) Flush() {
	w.strip()
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *quietWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
