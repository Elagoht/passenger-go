package web

import (
	"net/http"
	"strings"
)

// ContentSecurityPolicy is sent on every response: no inline scripts or
// styles, nothing from other origins.
const ContentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self'; " +
	"img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"

// SecurityHeaders sets the CSP, Referrer-Policy and X-Content-Type-Options on
// every response, and Cache-Control: no-store on every response but the
// static files under /static/ (they hold no secrets and may be cached). They
// are applied when the response's header is written, so they override what an
// inner handler (a guarded page's "private, no-cache") set.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hw := &headerWriter{ResponseWriter: w, static: strings.HasPrefix(r.URL.Path, "/static/")}
		// Deferred, so that a handler that wrote nothing (net/http sends the
		// header map now) or panicked (whatever recovers it writes the error
		// page on this header map) still gets them.
		defer hw.apply()
		next.ServeHTTP(hw, r)
	})
}

type headerWriter struct {
	http.ResponseWriter
	static bool
	done   bool
}

func (w *headerWriter) apply() {
	if w.done {
		return
	}
	w.done = true
	h := w.Header()
	h.Set("Content-Security-Policy", ContentSecurityPolicy)
	h.Set("Referrer-Policy", "no-referrer")
	if !w.static {
		h.Set("Cache-Control", "no-store")
	}
	h.Set("X-Content-Type-Options", "nosniff")
}

func (w *headerWriter) WriteHeader(status int) {
	w.apply()
	w.ResponseWriter.WriteHeader(status)
}

func (w *headerWriter) Write(b []byte) (int, error) {
	w.apply()
	return w.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer when it can flush.
func (w *headerWriter) Flush() {
	w.apply()
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *headerWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
